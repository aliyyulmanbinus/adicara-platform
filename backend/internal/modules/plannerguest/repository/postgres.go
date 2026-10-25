// Package repository is the PostgreSQL side of planner menu 05. The pool runs pgx in QueryExecModeSimpleProtocol, so
// results arrive as text: NULL columns scan into pointers, NUMERIC is cast to bigint in SQL, and pgx rejects an argument
// a statement does not use, so every query is given exactly the arguments it mentions. Every query filters by
// project_id; a guest with project_id NULL (it belongs to the invitation module alone) is never read or written here,
// except by ApplyRSVPByToken, which is keyed by the invitation.
//
// The writers that add rows (InsertGuest, InsertGuests) run under the project lock (SELECT … FOR UPDATE on
// planner_projects, LockProject) taken by the usecase, so the 5.000-guest limit and sort_order cannot race. Check-in,
// RSVP and the other single-row writers are one statement each and need no lock.
package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/db"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerguest/domain"
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
// one); the usecase also checks the scope before it writes.
func mapErr(err error) error {
	var pg *pgconn.PgError
	if !errors.As(err, &pg) {
		return err
	}
	switch pg.Code {
	case "23503":
		if strings.HasSuffix(pg.ConstraintName, "_sub_event_scope_fkey") {
			return domain.ErrInvalidScope
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

func (r *PostgresRepository) LockProject(ctx context.Context, projectID uuid.UUID) error {
	var id uuid.UUID
	err := r.q(ctx).QueryRow(ctx, `SELECT id FROM planner_projects WHERE id=$1 FOR UPDATE`, projectID).Scan(&id)

	return notFound(err)
}

func (r *PostgresRepository) Sessions(ctx context.Context, projectID uuid.UUID) ([]domain.SessionRef, error) {
	rows, err := r.q(ctx).Query(ctx,
		`SELECT scope_key, name FROM planner_event_sessions WHERE project_id=$1 ORDER BY sort_order, created_at, id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.SessionRef{}
	for rows.Next() {
		var s domain.SessionRef
		if err := rows.Scan(&s.ScopeKey, &s.Name); err != nil {
			return nil, err
		}
		out = append(out, s)
	}

	return out, rows.Err()
}

// ---------- guests ----------

const guestCols = `id, name, guest_side, category, sub_event_scope, table_number, pax_count, phone_number, formatted_wa_number,
	invitation_sent_status, invitation_sent_at, rsvp_status, pax_actual, rsvp_submitted_at, qr_code_token, is_checked_in,
	checked_in_at, souvenir_received, notes`

func scanGuest(row pgx.Row) (domain.Guest, error) {
	var g domain.Guest
	var sent, rsvp string
	err := row.Scan(&g.ID, &g.Name, &g.GuestSide, &g.Category, &g.SubEventScope, &g.TableNumber, &g.PaxCount, &g.PhoneNumber,
		&g.FormattedWANumber, &sent, &g.InvitationSentAt, &rsvp, &g.PaxActual, &g.RSVPSubmittedAt, &g.QRCodeToken, &g.IsCheckedIn,
		&g.CheckedInAt, &g.SouvenirReceived, &g.Notes)
	g.InvitationSentStatus, g.RSVPStatus = domain.SentStatus(sent), domain.RSVPStatus(rsvp)

	return g, err
}

func scanGuests(rows pgx.Rows) ([]domain.Guest, error) {
	defer rows.Close()
	out := []domain.Guest{}
	for rows.Next() {
		g, err := scanGuest(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}

	return out, rows.Err()
}

// nilIfEmpty is the argument of an optional filter: the statements test `$n::text IS NULL`.
func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}

	return &s
}

// listWhere is the filter of 05 BE 2.7, with the project as $1 and the filters as $2..$7. Search is a substring match
// (strpos), never LIKE, so `%` and `_` need no escaping.
const listWhere = `project_id = $1
	AND ($2::text IS NULL OR ($2 = 'none' AND guest_side IS NULL) OR guest_side = $2)
	AND ($3::text IS NULL OR category = $3)
	AND ($4::text IS NULL OR rsvp_status = $4)
	AND ($5::text IS NULL OR sub_event_scope = $5)
	AND ($6::text IS NULL OR (CASE WHEN $6 = 'sent' THEN invitation_sent_status <> 'unsent' ELSE invitation_sent_status = 'unsent' END))
	AND ($7::text IS NULL OR strpos(lower(name), lower($7)) > 0 OR strpos(lower(COALESCE(table_number, '')), lower($7)) > 0
	     OR strpos(COALESCE(phone_number, ''), $7) > 0 OR strpos(COALESCE(formatted_wa_number, ''), $7) > 0)`

func (r *PostgresRepository) ListGuests(ctx context.Context, projectID uuid.UUID, q domain.GuestQuery) ([]domain.Guest, int, error) {
	args := []any{projectID, nilIfEmpty(q.Side), nilIfEmpty(q.Category), nilIfEmpty(q.RSVP), nilIfEmpty(q.Scope), nilIfEmpty(q.Sent), nilIfEmpty(q.Search)}

	var total int
	if err := r.q(ctx).QueryRow(ctx, `SELECT count(*) FROM planner_guests WHERE `+listWhere, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.q(ctx).Query(ctx, `SELECT `+guestCols+` FROM planner_guests WHERE `+listWhere+
		` ORDER BY sort_order, created_at, id LIMIT $8 OFFSET $9`, append(args, q.Limit, q.Offset)...)
	if err != nil {
		return nil, 0, err
	}
	gs, err := scanGuests(rows)

	return gs, total, err
}

func (r *PostgresRepository) AllGuests(ctx context.Context, projectID uuid.UUID) ([]domain.Guest, error) {
	rows, err := r.q(ctx).Query(ctx,
		`SELECT `+guestCols+` FROM planner_guests WHERE project_id=$1 ORDER BY sort_order, created_at, id`, projectID)
	if err != nil {
		return nil, err
	}

	return scanGuests(rows)
}

func (r *PostgresRepository) GuestByID(ctx context.Context, projectID, id uuid.UUID) (domain.Guest, error) {
	g, err := scanGuest(r.q(ctx).QueryRow(ctx, `SELECT `+guestCols+` FROM planner_guests WHERE id=$1 AND project_id=$2`, id, projectID))

	return g, notFound(err)
}

func (r *PostgresRepository) GuestByToken(ctx context.Context, projectID uuid.UUID, token string) (domain.Guest, error) {
	g, err := scanGuest(r.q(ctx).QueryRow(ctx,
		`SELECT `+guestCols+` FROM planner_guests WHERE project_id=$1 AND qr_code_token=$2`, projectID, token))

	return g, notFound(err)
}

func (r *PostgresRepository) CountGuests(ctx context.Context, projectID uuid.UUID) (int, error) {
	var n int
	err := r.q(ctx).QueryRow(ctx, `SELECT count(*) FROM planner_guests WHERE project_id=$1`, projectID).Scan(&n)

	return n, err
}

func (r *PostgresRepository) ExistingForImport(ctx context.Context, projectID uuid.UUID) ([]domain.ExistingGuest, error) {
	rows, err := r.q(ctx).Query(ctx,
		`SELECT id, name, COALESCE(formatted_wa_number, '') FROM planner_guests WHERE project_id=$1 ORDER BY sort_order, created_at, id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.ExistingGuest{}
	for rows.Next() {
		var g domain.ExistingGuest
		if err := rows.Scan(&g.ID, &g.Name, &g.WA); err != nil {
			return nil, err
		}
		out = append(out, g)
	}

	return out, rows.Err()
}

// ---------- summaries ----------

// Aggregates runs the two statements of 05 BE 2.8. The first is the one of the dashboard provider of menu 01 (01 BE 5.2).
func (r *PostgresRepository) Aggregates(ctx context.Context, projectID uuid.UUID) (domain.Aggregates, error) {
	var a domain.Aggregates
	err := r.q(ctx).QueryRow(ctx, `
		SELECT count(*), COALESCE(sum(pax_count), 0),
		       count(*) FILTER (WHERE invitation_sent_status <> 'unsent'),
		       count(*) FILTER (WHERE rsvp_status = 'attending'),
		       count(*) FILTER (WHERE rsvp_status = 'declined'),
		       count(*) FILTER (WHERE rsvp_status = 'maybe'),
		       count(*) FILTER (WHERE rsvp_status = 'unconfirmed'),
		       COALESCE(sum(CASE WHEN rsvp_status = 'attending' THEN COALESCE(NULLIF(pax_actual, 0), pax_count) END), 0)
		FROM planner_guests WHERE project_id = $1`, projectID).
		Scan(&a.Total, &a.TotalPax, &a.Sent, &a.Attending, &a.Declined, &a.Maybe, &a.Unconfirmed, &a.AttendingPax)
	if err != nil {
		return a, err
	}

	rows, err := r.q(ctx).Query(ctx, `
		SELECT guest_side, category, count(*), sum(pax_count)::bigint
		FROM planner_guests WHERE project_id = $1 GROUP BY guest_side, category`, projectID)
	if err != nil {
		return a, err
	}
	defer rows.Close()
	for rows.Next() {
		var c domain.Cell
		if err := rows.Scan(&c.Side, &c.Category, &c.Invitations, &c.Pax); err != nil {
			return a, err
		}
		a.Cells = append(a.Cells, c)
	}

	return a, rows.Err()
}

func (r *PostgresRepository) Catering(ctx context.Context, projectID uuid.UUID) (domain.CateringProjection, error) {
	var p domain.CateringProjection
	err := r.q(ctx).QueryRow(ctx, `
		SELECT
		  COALESCE(sum(CASE WHEN rsvp_status = 'attending' THEN COALESCE(NULLIF(pax_actual, 0), pax_count) END), 0),
		  COALESCE(sum(CASE WHEN rsvp_status IN ('maybe', 'unconfirmed') THEN pax_count END), 0),
		  COALESCE(sum(CASE WHEN rsvp_status = 'declined' THEN pax_count END), 0),
		  count(*) FILTER (WHERE rsvp_status = 'attending'),
		  count(*) FILTER (WHERE rsvp_status = 'declined'),
		  count(*) FILTER (WHERE rsvp_status = 'maybe'),
		  count(*) FILTER (WHERE rsvp_status = 'unconfirmed')
		FROM planner_guests WHERE project_id = $1`, projectID).
		Scan(&p.ConfirmedPax, &p.PotentialPax, &p.DeclinedPax, &p.AttendingGuests, &p.DeclinedGuests, &p.MaybeGuests, &p.PendingGuests)

	return p, err
}

// ---------- writes ----------

// insertCols are the columns InsertGuest(s) write; sort_order is the last one.
const insertCols = `id, project_id, invitation_id, name, guest_side, category, sub_event_scope, table_number, pax_count, phone_number,
	formatted_wa_number, invitation_sent_status, invitation_sent_at, rsvp_status, pax_actual, rsvp_submitted_at, qr_code_token,
	is_checked_in, checked_in_at, souvenir_received, notes, sort_order`

const insertColCount = 22

func insertArgs(projectID uuid.UUID, invitationID *uuid.UUID, g domain.Guest, sortOrder int) []any {
	return []any{g.ID, projectID, invitationID, g.Name, g.GuestSide, g.Category, g.SubEventScope, g.TableNumber, g.PaxCount, g.PhoneNumber,
		g.FormattedWANumber, string(g.InvitationSentStatus), g.InvitationSentAt, string(g.RSVPStatus), g.PaxActual, g.RSVPSubmittedAt, g.QRCodeToken,
		g.IsCheckedIn, g.CheckedInAt, g.SouvenirReceived, g.Notes, sortOrder}
}

func (r *PostgresRepository) maxSort(ctx context.Context, projectID uuid.UUID) (int, error) {
	var n int
	err := r.q(ctx).QueryRow(ctx, `SELECT COALESCE(max(sort_order), 0) FROM planner_guests WHERE project_id=$1`, projectID).Scan(&n)

	return n, err
}

func (r *PostgresRepository) InsertGuest(ctx context.Context, projectID uuid.UUID, invitationID *uuid.UUID, g *domain.Guest) error {
	return r.InsertGuests(ctx, projectID, invitationID, []domain.Guest{*g})
}

// insertBatch is how many rows one INSERT carries (22 columns each; the simple protocol puts the values in the text).
const insertBatch = 500

func (r *PostgresRepository) InsertGuests(ctx context.Context, projectID uuid.UUID, invitationID *uuid.UUID, gs []domain.Guest) error {
	base, err := r.maxSort(ctx, projectID)
	if err != nil {
		return err
	}
	for from := 0; from < len(gs); from += insertBatch {
		to := min(from+insertBatch, len(gs))
		var sql strings.Builder
		sql.WriteString(`INSERT INTO planner_guests (` + insertCols + `) VALUES `)
		args := make([]any, 0, (to-from)*insertColCount)
		for i, g := range gs[from:to] {
			if i > 0 {
				sql.WriteByte(',')
			}
			sql.WriteByte('(')
			for c := 0; c < insertColCount; c++ {
				if c > 0 {
					sql.WriteByte(',')
				}
				sql.WriteString("$" + strconv.Itoa(len(args)+c+1))
			}
			sql.WriteByte(')')
			args = append(args, insertArgs(projectID, invitationID, g, base+from+i+1)...)
		}
		if _, err := r.q(ctx).Exec(ctx, sql.String(), args...); err != nil {
			return mapErr(err)
		}
	}

	return nil
}

func (r *PostgresRepository) UpdateGuest(ctx context.Context, projectID uuid.UUID, g *domain.Guest) error {
	updated, err := scanGuest(r.q(ctx).QueryRow(ctx, `
		UPDATE planner_guests
		SET name=$3, guest_side=$4, category=$5, sub_event_scope=$6, table_number=$7, pax_count=$8, phone_number=$9,
		    formatted_wa_number=$10, notes=$11, updated_at=now()
		WHERE id=$1 AND project_id=$2
		RETURNING `+guestCols,
		g.ID, projectID, g.Name, g.GuestSide, g.Category, g.SubEventScope, g.TableNumber, g.PaxCount, g.PhoneNumber,
		g.FormattedWANumber, g.Notes))
	if err != nil {
		return notFound(err)
	}
	*g = updated

	return nil
}

func (r *PostgresRepository) SetSentStatus(ctx context.Context, projectID, id uuid.UUID, status domain.SentStatus) (domain.Guest, error) {
	g, err := scanGuest(r.q(ctx).QueryRow(ctx, `
		UPDATE planner_guests
		SET invitation_sent_status=$3::text,
		    invitation_sent_at = CASE WHEN $3::text = 'unsent' THEN NULL ELSE now() END,
		    updated_at=now()
		WHERE id=$1 AND project_id=$2
		RETURNING `+guestCols, id, projectID, string(status)))

	return g, notFound(err)
}

// rsvpAssign is the assignment of an RSVP (status and pax are the placeholders): the stamp is cleared for
// `unconfirmed`, kept when the same answer comes again (a retried request or a repeated public form), and renewed otherwise.
func rsvpAssign(status, pax string) string {
	return fmt.Sprintf(`rsvp_status=%[1]s::text, pax_actual=%[2]s::int,
	rsvp_submitted_at = CASE WHEN %[1]s::text = 'unconfirmed' THEN NULL
	                         WHEN rsvp_status = %[1]s::text AND pax_actual = %[2]s::int AND rsvp_submitted_at IS NOT NULL THEN rsvp_submitted_at
	                         ELSE now() END,
	updated_at=now()`, status, pax)
}

var (
	setRSVPSQL = `UPDATE planner_guests SET ` + rsvpAssign("$3", "$4") +
		` WHERE id=$1 AND project_id=$2 AND is_checked_in = FALSE RETURNING ` + guestCols
	applyRSVPSQL = `UPDATE planner_guests SET ` + rsvpAssign("$3", "$4") +
		` WHERE invitation_id=$1 AND qr_code_token=$2 AND is_checked_in = FALSE`
)

func (r *PostgresRepository) SetRSVP(ctx context.Context, projectID, id uuid.UUID, status domain.RSVPStatus, paxActual int) (domain.Guest, error) {
	g, err := scanGuest(r.q(ctx).QueryRow(ctx, setRSVPSQL, id, projectID, string(status), paxActual))
	if errors.Is(err, pgx.ErrNoRows) {
		return g, r.whyNotUpdated(ctx, projectID, id)
	}

	return g, mapErr(err)
}

// whyNotUpdated tells an unknown guest (ErrNotFound) from one that has checked in already (ErrAlreadyCheckedIn) after an
// UPDATE … WHERE is_checked_in = FALSE touched no row.
func (r *PostgresRepository) whyNotUpdated(ctx context.Context, projectID, id uuid.UUID) error {
	var in bool
	err := r.q(ctx).QueryRow(ctx, `SELECT is_checked_in FROM planner_guests WHERE id=$1 AND project_id=$2`, id, projectID).Scan(&in)
	if err != nil {
		return notFound(err)
	}
	if in {
		return domain.ErrAlreadyCheckedIn
	}

	// The row was there and not checked in: somebody undid the check-in between the two statements. Let the caller retry.
	return domain.ErrConflict
}

func (r *PostgresRepository) ApplyRSVPByToken(ctx context.Context, invitationID uuid.UUID, token string, status domain.RSVPStatus, paxActual int) error {
	tag, err := r.q(ctx).Exec(ctx, applyRSVPSQL, invitationID, token, string(status), paxActual)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() > 0 {
		return nil
	}

	// No row changed: either there is no such guest, or the guest has checked in (success, nothing to change).
	var one int
	err = r.q(ctx).QueryRow(ctx, `SELECT 1 FROM planner_guests WHERE invitation_id=$1 AND qr_code_token=$2`, invitationID, token).Scan(&one)

	return notFound(err)
}

func (r *PostgresRepository) SetSouvenir(ctx context.Context, projectID, id uuid.UUID, received bool) (domain.Guest, error) {
	g, err := scanGuest(r.q(ctx).QueryRow(ctx,
		`UPDATE planner_guests SET souvenir_received=$3, updated_at=now() WHERE id=$1 AND project_id=$2 RETURNING `+guestCols,
		id, projectID, received))

	return g, notFound(err)
}

func (r *PostgresRepository) DeleteGuest(ctx context.Context, projectID, id uuid.UUID) error {
	tag, err := r.q(ctx).Exec(ctx, `DELETE FROM planner_guests WHERE id=$1 AND project_id=$2`, id, projectID)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// ---------- reception ----------

func (r *PostgresRepository) Checkin(ctx context.Context, projectID, id uuid.UUID, paxActual int, souvenir bool) (domain.Guest, error) {
	g, err := scanGuest(r.q(ctx).QueryRow(ctx, `
		UPDATE planner_guests
		SET is_checked_in = TRUE, checked_in_at = now(), pax_actual = $3, souvenir_received = $4, updated_at = now()
		WHERE id = $1 AND project_id = $2 AND is_checked_in = FALSE
		RETURNING `+guestCols, id, projectID, paxActual, souvenir))
	if errors.Is(err, pgx.ErrNoRows) {
		return g, r.whyNotUpdated(ctx, projectID, id)
	}

	return g, mapErr(err)
}

func (r *PostgresRepository) CancelCheckin(ctx context.Context, projectID, id uuid.UUID) (domain.Guest, error) {
	g, err := scanGuest(r.q(ctx).QueryRow(ctx, `
		UPDATE planner_guests
		SET is_checked_in = FALSE, checked_in_at = NULL, souvenir_received = FALSE, updated_at = now()
		WHERE id = $1 AND project_id = $2 AND is_checked_in = TRUE
		RETURNING `+guestCols, id, projectID))
	if errors.Is(err, pgx.ErrNoRows) {
		return r.GuestByID(ctx, projectID, id) // not checked in: as it is (or ErrNotFound)
	}

	return g, mapErr(err)
}

func (r *PostgresRepository) ReceptionCounts(ctx context.Context, projectID uuid.UUID) (domain.ReceptionCounts, error) {
	out := domain.ReceptionCounts{ByCategory: map[string]int{}}
	k := &out.KPI
	err := r.q(ctx).QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE is_checked_in),
		       count(*) FILTER (WHERE rsvp_status <> 'declined' OR is_checked_in),
		       COALESCE(sum(pax_actual) FILTER (WHERE is_checked_in), 0),
		       COALESCE(sum(CASE WHEN is_checked_in THEN pax_actual
		                         WHEN rsvp_status <> 'declined' THEN COALESCE(NULLIF(pax_actual, 0), pax_count) END), 0),
		       count(*) FILTER (WHERE souvenir_received)
		FROM planner_guests WHERE project_id = $1`, projectID).
		Scan(&k.CheckedInGuests, &k.ExpectedGuests, &k.ActualPax, &k.ExpectedPax, &k.SouvenirDistributed)
	if err != nil {
		return out, err
	}
	k.PendingGuests = k.ExpectedGuests - k.CheckedInGuests

	rows, err := r.q(ctx).Query(ctx, `
		SELECT category, count(*) FROM planner_guests
		WHERE project_id = $1 AND (rsvp_status <> 'declined' OR is_checked_in) GROUP BY category`, projectID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var cat string
		var n int
		if err := rows.Scan(&cat, &n); err != nil {
			return out, err
		}
		out.ByCategory[cat] = n
	}

	return out, rows.Err()
}

// queueWhere is the filter of the reception queue ($2 status, $3 category, $4 search). The queue shows the expected
// guests; a search widens it to the guests who declined and have not arrived, because one of them may stand at the door.
const queueWhere = `project_id = $1
	AND (CASE $2::text
	       WHEN 'in' THEN is_checked_in
	       WHEN 'pending' THEN NOT is_checked_in AND ($4::text IS NOT NULL OR rsvp_status <> 'declined')
	       ELSE ($4::text IS NOT NULL OR rsvp_status <> 'declined' OR is_checked_in) END)
	AND ($3::text IS NULL OR category = $3)
	AND ($4::text IS NULL OR strpos(lower(name), lower($4)) > 0 OR strpos(lower(COALESCE(table_number, '')), lower($4)) > 0
	     OR strpos(COALESCE(phone_number, ''), $4) > 0 OR strpos(COALESCE(formatted_wa_number, ''), $4) > 0)`

func (r *PostgresRepository) ReceptionQueue(ctx context.Context, projectID uuid.UUID, q domain.ReceptionQuery) ([]domain.Guest, int, error) {
	args := []any{projectID, q.Status, nilIfEmpty(q.Category), nilIfEmpty(q.Search)}

	var total int
	if err := r.q(ctx).QueryRow(ctx, `SELECT count(*) FROM planner_guests WHERE `+queueWhere, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	order := `lower(name), id`
	if q.Status == domain.StatusIn {
		order = `checked_in_at DESC, id`
	}
	rows, err := r.q(ctx).Query(ctx, `SELECT `+guestCols+` FROM planner_guests WHERE `+queueWhere+
		` ORDER BY `+order+` LIMIT $5 OFFSET $6`, append(args, q.Limit, q.Offset)...)
	if err != nil {
		return nil, 0, err
	}
	gs, err := scanGuests(rows)

	return gs, total, err
}

func (r *PostgresRepository) RecentCheckins(ctx context.Context, projectID uuid.UUID, n int) ([]domain.Guest, error) {
	rows, err := r.q(ctx).Query(ctx, `SELECT `+guestCols+` FROM planner_guests WHERE project_id=$1 AND is_checked_in
		ORDER BY checked_in_at DESC, id LIMIT $2`, projectID, n)
	if err != nil {
		return nil, err
	}

	return scanGuests(rows)
}

// SouvenirStock reads the shopping list of menu 04 (read only): the quantity of the finished items marked souvenir,
// rounded down. found is false when no item is marked.
func (r *PostgresRepository) SouvenirStock(ctx context.Context, projectID uuid.UUID) (int, bool, error) {
	var items, stock int
	err := r.q(ctx).QueryRow(ctx, `
		SELECT count(*), COALESCE(floor(sum(quantity) FILTER (WHERE status = 'selesai')), 0)::bigint
		FROM planner_shopping_items WHERE project_id = $1 AND purpose = 'souvenir'`, projectID).Scan(&items, &stock)

	return stock, items > 0, err
}
