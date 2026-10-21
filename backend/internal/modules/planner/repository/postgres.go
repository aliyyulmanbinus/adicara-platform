// Package repository is the PostgreSQL side of the planner module. The pool
// runs pgx in QueryExecModeSimpleProtocol, so results arrive as text: NULL
// columns scan into pointers, DATE into time.Time (00:00 UTC), TIME is read
// through to_char, and arrays/JSON/dates go in as strings with explicit casts.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/db"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) q(ctx context.Context) db.Queryable {
	return db.Querier(ctx, r.pool)
}

// mapErr turns the PostgreSQL errors the usecase cares about into domain
// sentinels. Anything else is returned unchanged (and becomes a 500).
func mapErr(err error) error {
	var pg *pgconn.PgError
	if !errors.As(err, &pg) {
		return err
	}
	switch pg.Code {
	case "23505":
		switch {
		case pg.ConstraintName == "planner_event_sessions_name_idx":
			return domain.ErrSessionNameTaken
		case strings.Contains(pg.ConstraintName, "scope_key"):
			return domain.ErrScopeKeyTaken
		}
	case "23503":
		if strings.HasSuffix(pg.ConstraintName, "_sub_event_scope_fkey") {
			return domain.ErrInvalidScope
		}

		return domain.ErrConflict
	case "23514":
		// planner_check_timezone raises check_violation for a zone name
		// PostgreSQL does not know.
		if strings.Contains(pg.Message, "unknown time zone") {
			return domain.ErrInvalidTimezone
		}
	case "40001", "40P01":
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

// TxManager implements domain.TxManager on the pool.
type TxManager struct {
	pool *pgxpool.Pool
}

func NewTxManager(pool *pgxpool.Pool) *TxManager { return &TxManager{pool: pool} }

func (m *TxManager) InTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := db.TxFrom(ctx); ok {
		return fn(ctx) // join the outer transaction
	}

	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()

	if err := fn(db.WithTx(ctx, tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		// Deferred foreign keys (sessions) are checked here, so a concurrent
		// change shows up as 23503 on COMMIT rather than on the statement.
		var pg *pgconn.PgError
		if errors.As(err, &pg) && (pg.Code == "23503" || pg.Code == "23505" || pg.Code == "40001" || pg.Code == "40P01") {
			return domain.ErrConflict
		}

		return err
	}

	return nil
}

// ---------- helpers ----------

func dateStr(t time.Time) string { return t.Format(domain.DateLayout) }

func dateStrPtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := dateStr(*t)

	return &s
}

func jsonString(m map[string]any) (string, error) {
	if m == nil {
		m = map[string]any{}
	}
	b, err := json.Marshal(m)

	return string(b), err
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}

	return s
}

// ---------- projects ----------

const projectCols = `id, user_id, event_type, title, event_date, event_end_date, prep_start_date, timezone,
	target_budget, savings_target, status, is_sample, skipped_vendor_categories, invitation_id, metadata, created_at, updated_at`

func scanProject(row pgx.Row) (domain.Project, error) {
	var p domain.Project
	err := row.Scan(&p.ID, &p.UserID, &p.EventType, &p.Title, &p.EventDate, &p.EventEndDate, &p.PrepStartDate, &p.Timezone,
		&p.TargetBudget, &p.SavingsTarget, &p.Status, &p.IsSample, &p.SkippedVendorCategories, &p.InvitationID, &p.Metadata,
		&p.CreatedAt, &p.UpdatedAt)
	if p.Metadata == nil {
		p.Metadata = map[string]any{}
	}

	return p, err
}

func (r *PostgresRepository) InsertProject(ctx context.Context, p *domain.Project, clientToken *string) (bool, error) {
	meta, err := jsonString(p.Metadata)
	if err != nil {
		return false, err
	}

	conflict := ""
	switch {
	case clientToken != nil:
		conflict = " ON CONFLICT (user_id, client_token) WHERE client_token IS NOT NULL DO NOTHING"
	case p.IsSample:
		conflict = " ON CONFLICT (user_id) WHERE is_sample DO NOTHING"
	}

	err = r.q(ctx).QueryRow(ctx, `
		INSERT INTO planner_projects (id, user_id, event_type, title, event_date, event_end_date, prep_start_date, timezone,
			target_budget, savings_target, status, is_sample, skipped_vendor_categories, invitation_id, client_token, metadata)
		VALUES ($1,$2,$3,$4,$5::date,$6::date,$7::date,$8,$9,$10,$11,$12,$13::text[],$14,$15,$16::jsonb)`+conflict+`
		RETURNING created_at, updated_at`,
		p.ID, p.UserID, string(p.EventType), p.Title, dateStr(p.EventDate), dateStrPtr(p.EventEndDate), dateStr(p.PrepStartDate),
		p.Timezone, p.TargetBudget, p.SavingsTarget, string(p.Status), p.IsSample, nonNil(p.SkippedVendorCategories),
		p.InvitationID, clientToken, meta).Scan(&p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, mapErr(err)
	}

	return true, nil
}

func (r *PostgresRepository) ProjectForUser(ctx context.Context, id, userID uuid.UUID) (domain.Project, error) {
	p, err := scanProject(r.q(ctx).QueryRow(ctx, `SELECT `+projectCols+` FROM planner_projects WHERE id=$1 AND user_id=$2`, id, userID))

	return p, notFound(err)
}

func (r *PostgresRepository) LockProject(ctx context.Context, id uuid.UUID) (domain.Project, error) {
	p, err := scanProject(r.q(ctx).QueryRow(ctx, `SELECT `+projectCols+` FROM planner_projects WHERE id=$1 FOR UPDATE`, id))

	return p, notFound(err)
}

func (r *PostgresRepository) ProjectByClientToken(ctx context.Context, userID uuid.UUID, token string) (domain.Project, error) {
	p, err := scanProject(r.q(ctx).QueryRow(ctx, `SELECT `+projectCols+` FROM planner_projects WHERE user_id=$1 AND client_token=$2`, userID, token))

	return p, notFound(err)
}

func (r *PostgresRepository) SampleProject(ctx context.Context, userID uuid.UUID) (domain.Project, error) {
	p, err := scanProject(r.q(ctx).QueryRow(ctx, `SELECT `+projectCols+` FROM planner_projects WHERE user_id=$1 AND is_sample`, userID))

	return p, notFound(err)
}

func (r *PostgresRepository) CountProjects(ctx context.Context, userID uuid.UUID) (int, error) {
	var n int
	err := r.q(ctx).QueryRow(ctx, `SELECT count(*) FROM planner_projects WHERE user_id=$1 AND NOT is_sample`, userID).Scan(&n)

	return n, err
}

// The phase is computed here and in domain.ComputePhase with one definition;
// "today" is the calendar date in each project's own zone. now is a parameter
// (not now()) so the midnight boundaries can be tested.
const phaseLateral = `
	CROSS JOIN LATERAL (SELECT ($3::timestamptz AT TIME ZONE p.timezone)::date AS today) n
	CROSS JOIN LATERAL (SELECT p.event_date - n.today AS days_remaining,
	       CASE WHEN n.today < p.event_date THEN 'upcoming'
	            WHEN n.today <= COALESCE(p.event_end_date, p.event_date) THEN 'ongoing'
	            ELSE 'finished' END AS phase,
	       CASE WHEN n.today BETWEEN p.event_date AND COALESCE(p.event_end_date, p.event_date)
	            THEN n.today - p.event_date + 1 ELSE 0 END AS day_number,
	       COALESCE(p.event_end_date, p.event_date) - p.event_date + 1 AS total_days) d`

// Active first (nearest date first), then completed, then archived (latest first).
const projectOrder = `
	ORDER BY CASE p.status WHEN 'active' THEN 0 WHEN 'completed' THEN 1 ELSE 2 END,
	         CASE WHEN p.status = 'active' THEN p.event_date END ASC,
	         p.event_date DESC, p.id`

func (r *PostgresRepository) ListCards(ctx context.Context, userID uuid.UUID, status string, now time.Time) ([]domain.ProjectCardResponse, error) {
	rows, err := r.q(ctx).Query(ctx, `
		SELECT p.id, p.event_type, p.title, p.event_date, p.event_end_date, p.status, p.is_sample,
		       p.target_budget, p.metadata,
		       d.days_remaining, d.phase, d.day_number, d.total_days,
		       COALESCE(f.estimate, 0), COALESCE(f.paid, 0),
		       COALESCE(t.total, 0), COALESCE(t.completed, 0),
		       COALESCE(g.pax, 0)
		FROM planner_projects p`+phaseLateral+`
		LEFT JOIN LATERAL (SELECT sum(estimate)::bigint AS estimate, sum(paid)::bigint AS paid
		                   FROM planner_budget_lines WHERE project_id = p.id) f ON true
		LEFT JOIN LATERAL (SELECT count(*) AS total, count(*) FILTER (WHERE status = 'selesai') AS completed
		                   FROM planner_tasks WHERE project_id = p.id) t ON true
		LEFT JOIN LATERAL (SELECT sum(pax_count)::bigint AS pax FROM planner_guests WHERE project_id = p.id) g ON true
		WHERE p.user_id = $1 AND ($2 = 'all' OR p.status = $2)`+projectOrder, userID, status, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cards := []domain.ProjectCardResponse{}
	for rows.Next() {
		var c domain.ProjectCardResponse
		var eventDate time.Time
		var endDate *time.Time
		if err := rows.Scan(&c.ID, &c.EventType, &c.Title, &eventDate, &endDate, &c.Status, &c.IsSample,
			&c.TargetBudget, &c.Metadata,
			&c.DaysRemaining, &c.Phase, &c.DayNumber, &c.TotalDays,
			&c.TotalEstimate, &c.TotalPaid, &c.TotalTasks, &c.CompletedTasks, &c.GuestPaxCount); err != nil {
			return nil, err
		}
		c.EventDate, c.EventEndDate = dateStr(eventDate), dateStrPtr(endDate)
		if c.Metadata == nil {
			c.Metadata = map[string]any{}
		}
		cards = append(cards, c)
	}

	return cards, rows.Err()
}

func (r *PostgresRepository) ListLite(ctx context.Context, userID uuid.UUID, status string, now time.Time) ([]domain.ProjectLiteResponse, error) {
	rows, err := r.q(ctx).Query(ctx, `
		SELECT p.id, p.event_type, p.title, p.event_date, p.event_end_date, p.status, p.is_sample,
		       d.days_remaining, d.phase
		FROM planner_projects p`+phaseLateral+`
		WHERE p.user_id = $1 AND ($2 = 'all' OR p.status = $2)`+projectOrder, userID, status, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.ProjectLiteResponse{}
	for rows.Next() {
		var l domain.ProjectLiteResponse
		var eventDate time.Time
		var endDate *time.Time
		if err := rows.Scan(&l.ID, &l.EventType, &l.Title, &eventDate, &endDate, &l.Status, &l.IsSample, &l.DaysRemaining, &l.Phase); err != nil {
			return nil, err
		}
		l.EventDate, l.EventEndDate = dateStr(eventDate), dateStrPtr(endDate)
		out = append(out, l)
	}

	return out, rows.Err()
}

func (r *PostgresRepository) UpdateProject(ctx context.Context, p *domain.Project) error {
	meta, err := jsonString(p.Metadata)
	if err != nil {
		return err
	}
	err = r.q(ctx).QueryRow(ctx, `
		UPDATE planner_projects
		SET title=$2, event_date=$3::date, event_end_date=$4::date, prep_start_date=$5::date, timezone=$6,
		    target_budget=$7, savings_target=$8, skipped_vendor_categories=$9::text[], metadata=$10::jsonb, updated_at=now()
		WHERE id=$1
		RETURNING updated_at`,
		p.ID, p.Title, dateStr(p.EventDate), dateStrPtr(p.EventEndDate), dateStr(p.PrepStartDate), p.Timezone,
		p.TargetBudget, p.SavingsTarget, nonNil(p.SkippedVendorCategories), meta).Scan(&p.UpdatedAt)

	return notFound(err)
}

func (r *PostgresRepository) SetProjectStatus(ctx context.Context, id uuid.UUID, status domain.ProjectStatus) error {
	tag, err := r.q(ctx).Exec(ctx, `UPDATE planner_projects SET status=$2, updated_at=now() WHERE id=$1`, id, string(status))
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *PostgresRepository) DeleteProject(ctx context.Context, id uuid.UUID) error {
	tag, err := r.q(ctx).Exec(ctx, `DELETE FROM planner_projects WHERE id=$1`, id)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// ---------- coupling with other modules' tables ----------

func (r *PostgresRepository) CountShiftableTasks(ctx context.Context, projectID uuid.UUID) (int, error) {
	var n int
	err := r.q(ctx).QueryRow(ctx,
		`SELECT count(*) FROM planner_tasks WHERE project_id=$1 AND status <> 'selesai' AND template_key IS NOT NULL`, projectID).Scan(&n)

	return n, err
}

func (r *PostgresRepository) ShiftTemplateTasks(ctx context.Context, projectID uuid.UUID, days int) (int, error) {
	tag, err := r.q(ctx).Exec(ctx, `
		UPDATE planner_tasks
		SET start_date = start_date + $2::int, end_date = end_date + $2::int, updated_at = now()
		WHERE project_id=$1 AND status <> 'selesai' AND template_key IS NOT NULL`, projectID, days)
	if err != nil {
		return 0, mapErr(err)
	}

	return int(tag.RowsAffected()), nil
}

func (r *PostgresRepository) CountOverdueTemplateTasks(ctx context.Context, projectID uuid.UUID, today time.Time) (int, error) {
	var n int
	err := r.q(ctx).QueryRow(ctx, `
		SELECT count(*) FROM planner_tasks
		WHERE project_id=$1 AND status <> 'selesai' AND template_key IS NOT NULL AND end_date < $2::date`,
		projectID, dateStr(today)).Scan(&n)

	return n, err
}

func (r *PostgresRepository) ShiftSessionDates(ctx context.Context, projectID uuid.UUID, from, to time.Time) error {
	_, err := r.q(ctx).Exec(ctx, `
		UPDATE planner_event_sessions SET session_date = $3::date, updated_at = now()
		WHERE project_id=$1 AND session_date = $2::date`, projectID, dateStr(from), dateStr(to))

	return mapErr(err)
}

func (r *PostgresRepository) CountSessionsOnOtherDates(ctx context.Context, projectID uuid.UUID, date time.Time) (int, error) {
	var n int
	err := r.q(ctx).QueryRow(ctx, `
		SELECT count(*) FROM planner_event_sessions
		WHERE project_id=$1 AND session_date IS NOT NULL AND session_date <> $2::date`, projectID, dateStr(date)).Scan(&n)

	return n, err
}

// ---------- sessions ----------

const sessionCols = `id, project_id, scope_key, name, icon, is_system, sort_order, session_date,
	to_char(start_time, 'HH24:MI'), to_char(end_time, 'HH24:MI'), location_name`

// The same columns qualified with the alias s, for queries that join or
// correlate against planner_event_sessions.
const sessionColsS = `s.id, s.project_id, s.scope_key, s.name, s.icon, s.is_system, s.sort_order, s.session_date,
	to_char(s.start_time, 'HH24:MI'), to_char(s.end_time, 'HH24:MI'), s.location_name`

func scanSession(row pgx.Row, extra ...any) (domain.Session, error) {
	var s domain.Session
	dest := append([]any{&s.ID, &s.ProjectID, &s.ScopeKey, &s.Name, &s.Icon, &s.IsSystem, &s.SortOrder, &s.SessionDate,
		&s.StartTime, &s.EndTime, &s.LocationName}, extra...)
	err := row.Scan(dest...)

	return s, err
}

func (r *PostgresRepository) ListSessions(ctx context.Context, projectID uuid.UUID) ([]domain.Session, error) {
	rows, err := r.q(ctx).Query(ctx, `SELECT `+sessionCols+` FROM planner_event_sessions WHERE project_id=$1 ORDER BY sort_order, created_at, id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.Session{}
	for rows.Next() {
		s, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}

	return out, rows.Err()
}

func (r *PostgresRepository) ListSessionsWithUsage(ctx context.Context, projectID uuid.UUID) ([]domain.SessionWithUsage, error) {
	rows, err := r.q(ctx).Query(ctx, `
		SELECT `+sessionColsS+`,
		  (SELECT count(*) FROM planner_tasks x WHERE x.project_id = s.project_id AND x.sub_event_scope = s.scope_key),
		  (SELECT count(*) FROM planner_vendors x WHERE x.project_id = s.project_id AND x.sub_event_scope = s.scope_key),
		  (SELECT count(*) FROM planner_budget_items x WHERE x.project_id = s.project_id AND x.sub_event_scope = s.scope_key),
		  (SELECT count(*) FROM planner_budget_scenario_items x WHERE x.project_id = s.project_id AND x.sub_event_scope = s.scope_key),
		  (SELECT count(*) FROM planner_shopping_items x WHERE x.project_id = s.project_id AND x.sub_event_scope = s.scope_key),
		  (SELECT count(*) FROM planner_guests x WHERE x.project_id = s.project_id AND x.sub_event_scope = s.scope_key),
		  (SELECT count(*) FROM planner_rundowns x WHERE x.session_id = s.id)
		FROM planner_event_sessions s WHERE s.project_id = $1 ORDER BY s.sort_order, s.created_at, s.id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.SessionWithUsage{}
	for rows.Next() {
		var u domain.SessionUsage
		s, err := scanSession(rows, &u.Tasks, &u.Vendors, &u.BudgetItems, &u.ScenarioItems, &u.ShoppingItems, &u.Guests, &u.Agendas)
		if err != nil {
			return nil, err
		}
		out = append(out, domain.SessionWithUsage{Session: s, Usage: u})
	}

	return out, rows.Err()
}

func (r *PostgresRepository) SessionByID(ctx context.Context, projectID, sessionID uuid.UUID) (domain.Session, error) {
	s, err := scanSession(r.q(ctx).QueryRow(ctx, `SELECT `+sessionCols+` FROM planner_event_sessions WHERE id=$1 AND project_id=$2`, sessionID, projectID))

	return s, notFound(err)
}

func (r *PostgresRepository) InsertSession(ctx context.Context, s *domain.Session) error {
	_, err := r.q(ctx).Exec(ctx, `
		INSERT INTO planner_event_sessions (id, project_id, scope_key, name, icon, is_system, sort_order, session_date, start_time, end_time, location_name)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8::date,$9::time,$10::time,$11)`,
		s.ID, s.ProjectID, s.ScopeKey, s.Name, s.Icon, s.IsSystem, s.SortOrder, dateStrPtr(s.SessionDate), s.StartTime, s.EndTime, s.LocationName)

	return mapErr(err)
}

func (r *PostgresRepository) UpdateSession(ctx context.Context, s *domain.Session) error {
	tag, err := r.q(ctx).Exec(ctx, `
		UPDATE planner_event_sessions
		SET name=$3, icon=$4, session_date=$5::date, start_time=$6::time, end_time=$7::time, location_name=$8, updated_at=now()
		WHERE id=$1 AND project_id=$2 AND NOT is_system`,
		s.ID, s.ProjectID, s.Name, s.Icon, dateStrPtr(s.SessionDate), s.StartTime, s.EndTime, s.LocationName)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *PostgresRepository) ReorderSessions(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) error {
	idStrs := make([]string, len(ids))
	orders := make([]int32, len(ids))
	for i, id := range ids {
		idStrs[i], orders[i] = id.String(), int32(i+1)
	}
	_, err := r.q(ctx).Exec(ctx, `
		UPDATE planner_event_sessions s SET sort_order = v.ord, updated_at = now()
		FROM unnest($2::text[]::uuid[], $3::int[]) AS v(id, ord)
		WHERE s.id = v.id AND s.project_id = $1 AND NOT s.is_system`, projectID, idStrs, orders)

	return mapErr(err)
}

func (r *PostgresRepository) SessionUsage(ctx context.Context, projectID uuid.UUID, s domain.Session) (domain.SessionUsage, error) {
	var u domain.SessionUsage
	err := r.q(ctx).QueryRow(ctx, `
		SELECT
		  (SELECT count(*) FROM planner_tasks WHERE project_id = $1 AND sub_event_scope = $2),
		  (SELECT count(*) FROM planner_vendors WHERE project_id = $1 AND sub_event_scope = $2),
		  (SELECT count(*) FROM planner_budget_items WHERE project_id = $1 AND sub_event_scope = $2),
		  (SELECT count(*) FROM planner_budget_scenario_items WHERE project_id = $1 AND sub_event_scope = $2),
		  (SELECT count(*) FROM planner_shopping_items WHERE project_id = $1 AND sub_event_scope = $2),
		  (SELECT count(*) FROM planner_guests WHERE project_id = $1 AND sub_event_scope = $2),
		  (SELECT count(*) FROM planner_rundowns WHERE session_id = $3)`,
		projectID, s.ScopeKey, s.ID).
		Scan(&u.Tasks, &u.Vendors, &u.BudgetItems, &u.ScenarioItems, &u.ShoppingItems, &u.Guests, &u.Agendas)

	return u, err
}

// moveSessionStatements re-point everything that references scope $2 of
// project $1 to scope $3 and drop the session (01 BE §4.6). Why this is not
// one simple UPDATE per table:
//   - budget items of two sessions can share an item_key (catering in both
//     Lamaran and Resepsi); moving them as is breaks the unique index, so the
//     colliding item loses its key and becomes a plain line;
//   - scenario items also point at a session and are moved the same way;
//   - planner_budget_scenarios.active_sub_events is an array without a foreign
//     key and would keep naming a session that no longer exists.
//
// pgx rejects arguments a statement does not use, hence the explicit count.
var moveSessionStatements = []struct {
	sql   string
	nargs int
}{
	{`UPDATE planner_tasks          SET sub_event_scope = $3, updated_at = now() WHERE project_id = $1 AND sub_event_scope = $2`, 3},
	{`UPDATE planner_vendors        SET sub_event_scope = $3, updated_at = now() WHERE project_id = $1 AND sub_event_scope = $2`, 3},
	{`UPDATE planner_shopping_items SET sub_event_scope = $3, updated_at = now() WHERE project_id = $1 AND sub_event_scope = $2`, 3},
	{`UPDATE planner_guests         SET sub_event_scope = $3, updated_at = now() WHERE project_id = $1 AND sub_event_scope = $2`, 3},
	{`UPDATE planner_budget_items b
	 SET sub_event_scope = $3, updated_at = now(),
	     item_key = CASE WHEN b.item_key IS NOT NULL AND EXISTS (
	                  SELECT 1 FROM planner_budget_items o
	                  WHERE o.project_id = b.project_id AND o.sub_event_scope = $3 AND o.item_key = b.item_key)
	                THEN NULL ELSE b.item_key END
	 WHERE b.project_id = $1 AND b.sub_event_scope = $2`, 3},
	{`UPDATE planner_budget_scenario_items s
	 SET sub_event_scope = $3,
	     item_key = CASE WHEN s.item_key IS NOT NULL AND EXISTS (
	                  SELECT 1 FROM planner_budget_scenario_items o
	                  WHERE o.scenario_id = s.scenario_id AND o.sub_event_scope = $3 AND o.item_key = s.item_key)
	                THEN NULL ELSE s.item_key END
	 WHERE s.project_id = $1 AND s.sub_event_scope = $2`, 3},
	{`UPDATE planner_budget_scenarios
	 SET active_sub_events = array_remove(active_sub_events, $2)
	         || CASE WHEN $3 <> 'all' AND NOT ($3 = ANY(active_sub_events)) THEN ARRAY[$3] ELSE ARRAY[]::text[] END,
	     updated_at = now()
	 WHERE project_id = $1 AND $2 = ANY(active_sub_events)`, 3},
	{`DELETE FROM planner_event_sessions WHERE project_id = $1 AND scope_key = $2 AND NOT is_system`, 2},
}

func (r *PostgresRepository) MoveSessionData(ctx context.Context, projectID uuid.UUID, from, to string) error {
	q := r.q(ctx)
	args := []any{projectID, from, to}
	for _, stmt := range moveSessionStatements {
		if _, err := q.Exec(ctx, stmt.sql, args[:stmt.nargs]...); err != nil {
			return mapErr(err)
		}
	}

	return nil
}
