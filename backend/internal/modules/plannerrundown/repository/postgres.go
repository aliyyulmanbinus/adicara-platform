// Package repository is the PostgreSQL side of planner menu 06. The pool runs pgx in QueryExecModeSimpleProtocol, so
// results arrive as text: NULL columns scan into pointers, TIME columns are read with to_char(…, 'HH24:MI'), arrays go in
// as []string with an explicit cast (text[]::uuid[]), and pgx rejects an argument a statement does not use, so every query
// is given exactly the arguments it mentions.
//
// The span of a session is computed from minutes (EXTRACT(EPOCH …)/60), never with time arithmetic: '00:30'::time +
// interval '24 hours' is still 00:30 (06 BE 2.1). Writers that add rows or change the order hold the project lock
// (SELECT … FOR UPDATE on planner_projects, LockProject).
package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/db"
	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerrundown/domain"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

var _ domain.Repository = (*PostgresRepository)(nil)

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) q(ctx context.Context) db.Queryable {
	return db.Querier(ctx, r.pool)
}

// mapErr turns the PostgreSQL errors the usecase cares about into domain sentinels. The foreign key into
// planner_event_sessions is deferred, so a wrong session normally shows up at COMMIT (the shared TxManager maps that
// one); the composite keys to vendors and agendas are immediate and show up at the statement.
func mapErr(err error) error {
	var pg *pgconn.PgError
	if !errors.As(err, &pg) {
		return err
	}
	switch pg.Code {
	case "23503":
		switch {
		case strings.HasSuffix(pg.ConstraintName, "_session_id_project_id_fkey"):
			return domain.ErrInvalidScope
		case strings.HasSuffix(pg.ConstraintName, "_vendor_id_project_id_fkey"):
			return domain.ErrInvalidVendor
		case strings.HasSuffix(pg.ConstraintName, "_rundown_id_project_id_fkey"):
			return domain.ErrInvalidAgenda
		}

		return domain.ErrConflict
	case "23505", "40001", "40P01":
		return domain.ErrConflict
	}

	return err
}

func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}

	return mapErr(err)
}

func affected(tag pgconn.CommandTag, err error) error {
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func idStrings(ids []uuid.UUID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}

	return out
}

// ---------- project, sessions ----------

func (r *PostgresRepository) LockProject(ctx context.Context, projectID uuid.UUID) error {
	var id uuid.UUID
	err := r.q(ctx).QueryRow(ctx, `SELECT id FROM planner_projects WHERE id=$1 FOR UPDATE`, projectID).Scan(&id)

	return notFound(err)
}

func (r *PostgresRepository) Sessions(ctx context.Context, projectID uuid.UUID) ([]domain.SessionRef, error) {
	rows, err := r.q(ctx).Query(ctx, `
		SELECT id, scope_key, name, icon, is_system, sort_order, session_date, location_name
		FROM planner_event_sessions WHERE project_id=$1 ORDER BY sort_order, created_at, id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.SessionRef{}
	for rows.Next() {
		var s domain.SessionRef
		if err := rows.Scan(&s.ID, &s.ScopeKey, &s.Name, &s.Icon, &s.IsSystem, &s.SortOrder, &s.Date, &s.Location); err != nil {
			return nil, err
		}
		out = append(out, s)
	}

	return out, rows.Err()
}

// ---------- agendas ----------

// agendaSelect is the query of 06 BE 2.2: the agenda with its session, its vendor (for the WhatsApp button of the PIC)
// and the number of songs that point at it. The clock columns are the strings of the API.
const agendaSelect = `
	SELECT r.id, r.session_id, s.name, s.icon, s.scope_key,
	       to_char(r.start_time, 'HH24:MI'), to_char(r.end_time, 'HH24:MI'), r.ends_next_day, r.duration_minutes,
	       r.activity_name, r.pic_name, r.description, r.highlights, r.status, r.started_at, r.completed_at, r.sort_order, r.template_key,
	       v.id, v.vendor_name, v.category_label, v.pic_name, v.phone_number, v.formatted_wa_number,
	       (SELECT count(*) FROM planner_playlists p WHERE p.rundown_id = r.id AND p.project_id = r.project_id)
	FROM planner_rundowns r
	JOIN planner_event_sessions s ON s.id = r.session_id
	LEFT JOIN planner_vendors v ON v.id = r.vendor_id AND v.project_id = r.project_id`

func scanAgenda(row pgx.Row) (domain.Rundown, error) {
	var a domain.Rundown
	var key, vName, vCategory, vPIC, vPhone, vWA *string
	var vID *uuid.UUID
	err := row.Scan(&a.ID, &a.SessionID, &a.SessionName, &a.SessionIcon, &a.ScopeKey,
		&a.StartTime, &a.EndTime, &a.EndsNextDay, &a.DurationMinutes,
		&a.ActivityName, &a.PICName, &a.Description, &a.Highlights, &a.Status, &a.StartedAt, &a.CompletedAt, &a.SortOrder, &key,
		&vID, &vName, &vCategory, &vPIC, &vPhone, &vWA, &a.LinkedSongs)
	if err != nil {
		return a, err
	}
	a.StartedAt, a.CompletedAt = utc(a.StartedAt), utc(a.CompletedAt)
	if key != nil {
		a.TemplateKey = *key
	}
	if vID != nil {
		a.Vendor = &domain.RundownVendor{ID: *vID, VendorName: deref(vName), CategoryLabel: deref(vCategory),
			PICName: deref(vPIC), PhoneNumber: deref(vPhone), FormattedWANumber: deref(vWA)}
	}

	return a, nil
}

// utc puts a time stamp in UTC: pgx hands timestamptz columns back in the zone of the process, and the API speaks UTC.
func utc(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()

	return &u
}

func deref(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}

func (r *PostgresRepository) ListAgendas(ctx context.Context, projectID uuid.UUID, sessionID *uuid.UUID) ([]domain.Rundown, error) {
	rows, err := r.q(ctx).Query(ctx, agendaSelect+`
		WHERE r.project_id = $1 AND ($2::uuid IS NULL OR r.session_id = $2)
		ORDER BY s.sort_order, r.sort_order, r.start_time, r.created_at, r.id`, projectID, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.Rundown{}
	for rows.Next() {
		a, err := scanAgenda(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}

	return out, rows.Err()
}

func (r *PostgresRepository) AgendaByID(ctx context.Context, projectID, id uuid.UUID) (domain.Rundown, error) {
	a, err := scanAgenda(r.q(ctx).QueryRow(ctx, agendaSelect+` WHERE r.id = $1 AND r.project_id = $2`, id, projectID))

	return a, notFound(err)
}

// spanSQL is the end of an agenda in minutes since the midnight of its day: the end (the start for an open agenda), a
// day later when it ends the next day.
const spanSQL = `EXTRACT(EPOCH FROM COALESCE(r.end_time, r.start_time))/60 + CASE WHEN r.ends_next_day THEN 1440 ELSE 0 END`

func (r *PostgresRepository) SessionSummaries(ctx context.Context, projectID uuid.UUID) ([]domain.RundownSessionSummary, error) {
	rows, err := r.q(ctx).Query(ctx, `
		SELECT s.id, s.scope_key, s.name, s.icon, to_char(s.session_date, 'YYYY-MM-DD'),
		       count(r.id),
		       count(r.id) FILTER (WHERE r.status = 'completed'),
		       count(r.id) FILTER (WHERE r.status = 'in_progress'),
		       COALESCE((max(`+spanSQL+`) - min(EXTRACT(EPOCH FROM r.start_time)/60))::int, 0),
		       to_char(min(r.start_time), 'HH24:MI'),
		       (array_agg(to_char(COALESCE(r.end_time, r.start_time), 'HH24:MI') ORDER BY `+spanSQL+` DESC))[1]
		FROM planner_event_sessions s LEFT JOIN planner_rundowns r ON r.session_id = s.id
		WHERE s.project_id = $1 AND NOT s.is_system
		GROUP BY s.id, s.scope_key, s.session_date, s.sort_order
		ORDER BY s.sort_order, s.created_at, s.id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.RundownSessionSummary{}
	for rows.Next() {
		var s domain.RundownSessionSummary
		if err := rows.Scan(&s.ID, &s.ScopeKey, &s.Name, &s.Icon, &s.Date, &s.AgendaCount, &s.Completed, &s.InProgress,
			&s.SpanMinutes, &s.FirstStart, &s.LastEnd); err != nil {
			return nil, err
		}
		out = append(out, s)
	}

	return out, rows.Err()
}

func (r *PostgresRepository) DistinctPICs(ctx context.Context, projectID uuid.UUID) (int, error) {
	var n int
	err := r.q(ctx).QueryRow(ctx, `
		SELECT count(DISTINCT CASE WHEN vendor_id IS NOT NULL THEN 'v:' || vendor_id::text ELSE 'p:' || lower(btrim(pic_name)) END)
		FROM planner_rundowns WHERE project_id = $1 AND (vendor_id IS NOT NULL OR btrim(pic_name) <> '')`, projectID).Scan(&n)

	return n, err
}

func (r *PostgresRepository) LiveItems(ctx context.Context, projectID uuid.UUID) ([]domain.RundownLiveItem, error) {
	rows, err := r.q(ctx).Query(ctx, `
		SELECT r.id, r.status, r.started_at, r.completed_at
		FROM planner_rundowns r JOIN planner_event_sessions s ON s.id = r.session_id
		WHERE r.project_id = $1 ORDER BY s.sort_order, r.sort_order, r.start_time, r.created_at, r.id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.RundownLiveItem{}
	for rows.Next() {
		var it domain.RundownLiveItem
		if err := rows.Scan(&it.ID, &it.Status, &it.StartedAt, &it.CompletedAt); err != nil {
			return nil, err
		}
		it.StartedAt, it.CompletedAt = utc(it.StartedAt), utc(it.CompletedAt)
		out = append(out, it)
	}

	return out, rows.Err()
}

func (r *PostgresRepository) CountAgendas(ctx context.Context, projectID uuid.UUID) (int, error) {
	var n int
	err := r.q(ctx).QueryRow(ctx, `SELECT count(*) FROM planner_rundowns WHERE project_id=$1`, projectID).Scan(&n)

	return n, err
}

func (r *PostgresRepository) InsertAgenda(ctx context.Context, projectID uuid.UUID, a *domain.AgendaWrite) error {
	_, err := r.q(ctx).Exec(ctx, `
		INSERT INTO planner_rundowns (id, project_id, session_id, start_time, end_time, ends_next_day, activity_name, pic_name,
		       vendor_id, description, highlights, sort_order)
		VALUES ($1, $2, $3, $4::time, $5::time, $6, $7, $8, $9, $10, $11,
		        (SELECT COALESCE(max(sort_order), 0) + 1 FROM planner_rundowns WHERE project_id = $2 AND session_id = $3))`,
		a.ID, projectID, a.SessionID, a.Times.Start, a.Times.End, a.Times.EndsNextDay, a.ActivityName, a.PICName,
		a.VendorID, a.Description, a.Highlights)

	return mapErr(err)
}

func (r *PostgresRepository) UpdateAgenda(ctx context.Context, projectID uuid.UUID, a *domain.AgendaWrite) error {
	// Every right-hand side reads the row as it was, so "session_id <> $3" compares with the old session.
	tag, err := r.q(ctx).Exec(ctx, `
		UPDATE planner_rundowns SET
		       sort_order = CASE WHEN session_id <> $3
		                         THEN (SELECT COALESCE(max(sort_order), 0) + 1 FROM planner_rundowns WHERE project_id = $2 AND session_id = $3)
		                         ELSE sort_order END,
		       session_id = $3, start_time = $4::time, end_time = $5::time, ends_next_day = $6, activity_name = $7, pic_name = $8,
		       vendor_id = $9, description = $10, highlights = $11, updated_at = now()
		WHERE id = $1 AND project_id = $2`,
		a.ID, projectID, a.SessionID, a.Times.Start, a.Times.End, a.Times.EndsNextDay, a.ActivityName, a.PICName,
		a.VendorID, a.Description, a.Highlights)

	return affected(tag, err)
}

func (r *PostgresRepository) SetAgendaStatus(ctx context.Context, projectID, id uuid.UUID, status domain.RundownStatus) error {
	tag, err := r.q(ctx).Exec(ctx, `
		UPDATE planner_rundowns SET
		  status       = $3,
		  started_at   = CASE $3 WHEN 'pending'   THEN NULL ELSE COALESCE(started_at, now()) END,
		  completed_at = CASE $3 WHEN 'completed' THEN COALESCE(completed_at, now()) ELSE NULL END,
		  updated_at   = now()
		WHERE id = $1 AND project_id = $2`, id, projectID, string(status))

	return affected(tag, err)
}

func (r *PostgresRepository) DeleteAgenda(ctx context.Context, projectID, id uuid.UUID) error {
	tag, err := r.q(ctx).Exec(ctx, `DELETE FROM planner_rundowns WHERE id=$1 AND project_id=$2`, id, projectID)

	return affected(tag, err)
}

func (r *PostgresRepository) AgendaIDs(ctx context.Context, projectID, sessionID uuid.UUID) ([]uuid.UUID, error) {
	return r.ids(ctx, `SELECT id FROM planner_rundowns WHERE project_id=$1 AND session_id=$2`, projectID, sessionID)
}

func (r *PostgresRepository) ids(ctx context.Context, query string, args ...any) ([]uuid.UUID, error) {
	rows, err := r.q(ctx).Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}

	return out, rows.Err()
}

func (r *PostgresRepository) ReorderAgendas(ctx context.Context, projectID, sessionID uuid.UUID, ids []uuid.UUID) error {
	_, err := r.q(ctx).Exec(ctx, `
		UPDATE planner_rundowns r SET sort_order = u.ord::int, updated_at = now()
		FROM unnest($1::text[]::uuid[]) WITH ORDINALITY AS u(id, ord)
		WHERE r.id = u.id AND r.project_id = $2 AND r.session_id = $3`, idStrings(ids), projectID, sessionID)

	return mapErr(err)
}

// ---------- agenda template ----------

func (r *PostgresRepository) TemplateAgendaStates(ctx context.Context, projectID uuid.UUID, keys []string) (map[string]domain.TemplateAgendaState, error) {
	rows, err := r.q(ctx).Query(ctx, `
		SELECT template_key, id, session_id, status FROM planner_rundowns
		WHERE project_id = $1 AND template_key = ANY ($2::text[])`, projectID, keys)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]domain.TemplateAgendaState{}
	for rows.Next() {
		var key string
		var st domain.TemplateAgendaState
		if err := rows.Scan(&key, &st.ID, &st.SessionID, &st.Status); err != nil {
			return nil, err
		}
		out[key] = st
	}

	return out, rows.Err()
}

func (r *PostgresRepository) InsertTemplateAgenda(ctx context.Context, projectID uuid.UUID, a *domain.AgendaWrite) (bool, error) {
	tag, err := r.q(ctx).Exec(ctx, `
		INSERT INTO planner_rundowns (id, project_id, session_id, start_time, end_time, ends_next_day, activity_name, pic_name,
		       description, highlights, template_key, sort_order)
		VALUES ($1, $2, $3, $4::time, $5::time, $6, $7, $8, $9, $10, $11,
		        (SELECT COALESCE(max(sort_order), 0) + 1 FROM planner_rundowns WHERE project_id = $2 AND session_id = $3))
		ON CONFLICT (project_id, template_key) WHERE template_key IS NOT NULL DO NOTHING`,
		a.ID, projectID, a.SessionID, a.Times.Start, a.Times.End, a.Times.EndsNextDay, a.ActivityName, a.PICName,
		a.Description, a.Highlights, a.TemplateKey)
	if err != nil {
		return false, mapErr(err)
	}

	return tag.RowsAffected() == 1, nil
}

func (r *PostgresRepository) ResetTemplateAgenda(ctx context.Context, projectID, id uuid.UUID, a *domain.AgendaWrite) error {
	tag, err := r.q(ctx).Exec(ctx, `
		UPDATE planner_rundowns SET start_time = $3::time, end_time = $4::time, ends_next_day = $5, activity_name = $6, pic_name = $7,
		       description = $8, highlights = $9, sort_order = $10, updated_at = now()
		WHERE id = $1 AND project_id = $2`,
		id, projectID, a.Times.Start, a.Times.End, a.Times.EndsNextDay, a.ActivityName, a.PICName, a.Description, a.Highlights, a.SortOrder)

	return affected(tag, err)
}

// ---------- playlist ----------

// playlistSelect joins the agenda a song is tied to, with its session, so the list needs no second request.
const playlistSelect = `
	SELECT p.id, p.rundown_id, p.song_title, p.artist, p.moment_tag, p.audio_format, p.spotify_url, p.youtube_url,
	       p.audio_cue_notes, p.status, p.sort_order, p.template_key,
	       r.activity_name, s.name, to_char(r.start_time, 'HH24:MI'), to_char(r.end_time, 'HH24:MI')
	FROM planner_playlists p
	LEFT JOIN planner_rundowns r ON r.id = p.rundown_id AND r.project_id = p.project_id
	LEFT JOIN planner_event_sessions s ON s.id = r.session_id`

func scanPlaylist(row pgx.Row) (domain.Playlist, error) {
	var p domain.Playlist
	var spotify, youtube, cue, key, aName, aSession, aStart, aEnd *string
	err := row.Scan(&p.ID, &p.RundownID, &p.SongTitle, &p.Artist, &p.MomentTag, &p.AudioFormat, &spotify, &youtube,
		&cue, &p.Status, &p.SortOrder, &key, &aName, &aSession, &aStart, &aEnd)
	if err != nil {
		return p, err
	}
	p.SpotifyURL, p.YouTubeURL, p.AudioCueNotes, p.TemplateKey = deref(spotify), deref(youtube), deref(cue), deref(key)
	if p.RundownID != nil && aName != nil {
		p.Agenda = &domain.PlaylistAgenda{ID: *p.RundownID, ActivityName: *aName, SessionName: deref(aSession), StartTime: deref(aStart), EndTime: aEnd}
	}

	return p, nil
}

func (r *PostgresRepository) ListPlaylists(ctx context.Context, projectID uuid.UUID) ([]domain.Playlist, error) {
	rows, err := r.q(ctx).Query(ctx, playlistSelect+` WHERE p.project_id = $1 ORDER BY p.sort_order, p.created_at, p.id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.Playlist{}
	for rows.Next() {
		p, err := scanPlaylist(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}

	return out, rows.Err()
}

func (r *PostgresRepository) PlaylistByID(ctx context.Context, projectID, id uuid.UUID) (domain.Playlist, error) {
	p, err := scanPlaylist(r.q(ctx).QueryRow(ctx, playlistSelect+` WHERE p.id = $1 AND p.project_id = $2`, id, projectID))

	return p, notFound(err)
}

func (r *PostgresRepository) CountPlaylists(ctx context.Context, projectID uuid.UUID) (int, error) {
	var n int
	err := r.q(ctx).QueryRow(ctx, `SELECT count(*) FROM planner_playlists WHERE project_id=$1`, projectID).Scan(&n)

	return n, err
}

func (r *PostgresRepository) InsertPlaylist(ctx context.Context, projectID uuid.UUID, s *domain.SongWrite) error {
	_, err := r.q(ctx).Exec(ctx, `
		INSERT INTO planner_playlists (id, project_id, rundown_id, song_title, artist, moment_tag, audio_format, spotify_url, youtube_url,
		       audio_cue_notes, status, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,
		        (SELECT COALESCE(max(sort_order), 0) + 1 FROM planner_playlists WHERE project_id = $2))`,
		s.ID, projectID, s.RundownID, s.SongTitle, s.Artist, s.MomentTag, string(s.AudioFormat), s.SpotifyURL, s.YouTubeURL,
		s.CueNotes, string(s.Status))

	return mapErr(err)
}

func (r *PostgresRepository) UpdatePlaylist(ctx context.Context, projectID uuid.UUID, s *domain.SongWrite) error {
	tag, err := r.q(ctx).Exec(ctx, `
		UPDATE planner_playlists SET rundown_id = $3, song_title = $4, artist = $5, moment_tag = $6, audio_format = $7,
		       spotify_url = $8, youtube_url = $9, audio_cue_notes = $10, status = $11, updated_at = now()
		WHERE id = $1 AND project_id = $2`,
		s.ID, projectID, s.RundownID, s.SongTitle, s.Artist, s.MomentTag, string(s.AudioFormat), s.SpotifyURL, s.YouTubeURL,
		s.CueNotes, string(s.Status))

	return affected(tag, err)
}

func (r *PostgresRepository) DeletePlaylist(ctx context.Context, projectID, id uuid.UUID) error {
	tag, err := r.q(ctx).Exec(ctx, `DELETE FROM planner_playlists WHERE id=$1 AND project_id=$2`, id, projectID)

	return affected(tag, err)
}

func (r *PostgresRepository) PlaylistIDs(ctx context.Context, projectID uuid.UUID) ([]uuid.UUID, error) {
	return r.ids(ctx, `SELECT id FROM planner_playlists WHERE project_id=$1`, projectID)
}

func (r *PostgresRepository) ReorderPlaylists(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) error {
	_, err := r.q(ctx).Exec(ctx, `
		UPDATE planner_playlists p SET sort_order = u.ord::int, updated_at = now()
		FROM unnest($1::text[]::uuid[]) WITH ORDINALITY AS u(id, ord)
		WHERE p.id = u.id AND p.project_id = $2`, idStrings(ids), projectID)

	return mapErr(err)
}

func (r *PostgresRepository) InsertTemplateSong(ctx context.Context, projectID uuid.UUID, s *domain.SongWrite) (bool, error) {
	tag, err := r.q(ctx).Exec(ctx, `
		INSERT INTO planner_playlists (id, project_id, song_title, artist, moment_tag, audio_format, status, template_key, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8,
		        (SELECT COALESCE(max(sort_order), 0) + 1 FROM planner_playlists WHERE project_id = $2))
		ON CONFLICT (project_id, template_key) WHERE template_key IS NOT NULL DO NOTHING`,
		s.ID, projectID, s.SongTitle, s.Artist, s.MomentTag, string(s.AudioFormat), string(s.Status), s.TemplateKey)
	if err != nil {
		return false, mapErr(err)
	}

	return tag.RowsAffected() == 1, nil
}

// ---------- dashboard widget ----------

// Dashboard runs the three queries of 01 BE 5.2: the counts, the agenda in progress (the one that started first) and the
// next session that has a date (today or later).
func (r *PostgresRepository) Dashboard(ctx context.Context, projectID uuid.UUID, today time.Time) (pd.RundownSummary, error) {
	var out pd.RundownSummary
	if err := r.q(ctx).QueryRow(ctx, `
		SELECT (SELECT count(*) FROM planner_event_sessions WHERE project_id = $1 AND NOT is_system),
		       count(*),
		       count(*) FILTER (WHERE status = 'completed'),
		       count(*) FILTER (WHERE status = 'in_progress'),
		       (SELECT count(*) FROM planner_playlists WHERE project_id = $1)
		FROM planner_rundowns WHERE project_id = $1`, projectID).
		Scan(&out.Sessions, &out.Agendas, &out.Completed, &out.InProgress, &out.Songs); err != nil {
		return out, err
	}

	var cur pd.CurrentAgenda
	err := r.q(ctx).QueryRow(ctx, `
		SELECT r.activity_name, s.name
		FROM planner_rundowns r JOIN planner_event_sessions s ON s.id = r.session_id
		WHERE r.project_id = $1 AND r.status = 'in_progress'
		ORDER BY r.started_at NULLS LAST, r.sort_order LIMIT 1`, projectID).Scan(&cur.Activity, &cur.Session)
	switch {
	case err == nil:
		out.CurrentAgenda = &cur
	case !errors.Is(err, pgx.ErrNoRows):
		return out, err
	}

	var next pd.NextSession
	err = r.q(ctx).QueryRow(ctx, `
		SELECT name, to_char(session_date, 'YYYY-MM-DD'), to_char(start_time, 'HH24:MI')
		FROM planner_event_sessions
		WHERE project_id = $1 AND NOT is_system AND session_date >= $2::date
		ORDER BY session_date, start_time NULLS LAST, sort_order LIMIT 1`, projectID, today.Format(pd.DateLayout)).
		Scan(&next.Name, &next.Date, &next.StartTime)
	switch {
	case err == nil:
		out.NextSession = &next
	case !errors.Is(err, pgx.ErrNoRows):
		return out, err
	}

	return out, nil
}
