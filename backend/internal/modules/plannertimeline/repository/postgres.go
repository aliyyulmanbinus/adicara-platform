// Package repository is the PostgreSQL side of planner menu 02. The pool runs
// pgx in QueryExecModeSimpleProtocol, so results arrive as text: NULL columns
// scan into pointers, DATE into time.Time (00:00 UTC), and arrays and dates go
// in as strings with explicit casts. pgx also rejects an argument a statement
// does not use, so every query is given exactly the arguments it mentions.
package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/db"
	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannertimeline/domain"
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
// sentinels. The foreign keys into planner_event_sessions are deferred, so
// the scope error normally shows up at COMMIT (the shared TxManager maps that
// one); this covers the statement-time case.
func mapErr(err error) error {
	var pg *pgconn.PgError
	if !errors.As(err, &pg) {
		return err
	}
	switch pg.Code {
	case "23503":
		switch {
		case strings.HasSuffix(pg.ConstraintName, "_sub_event_scope_fkey"):
			return domain.ErrInvalidScope
		case strings.HasSuffix(pg.ConstraintName, "_vendor_id_project_id_fkey"):
			return domain.ErrInvalidVendor
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

func dateStr(t time.Time) string { return t.Format(pd.DateLayout) }

// zoned renders "now in the project's zone" for the queries: $zone and $now.
func localDate(zoneArg, nowArg int) string {
	return fmt.Sprintf("($%d::timestamptz AT TIME ZONE $%d)::date", nowArg, zoneArg)
}

// ---------- project, sessions, vendors ----------

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

func (r *PostgresRepository) VendorInProject(ctx context.Context, projectID, vendorID uuid.UUID) (bool, error) {
	var one int
	err := r.q(ctx).QueryRow(ctx, `SELECT 1 FROM planner_vendors WHERE id=$1 AND project_id=$2`, vendorID, projectID).Scan(&one)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}

	return err == nil, err
}

// ---------- tasks ----------

// taskCols: the second line is the joined vendor and the overdue flag, which
// need $zone and $now (argument numbers are passed in).
func taskSelect(zoneArg, nowArg int) string {
	return `SELECT t.id, t.project_id, t.title, t.sub_event_scope, t.category, t.phase_order, t.phase_name,
	       t.start_date, t.end_date, t.duration_days, t.status, t.pic,
	       t.vendor_id, v.vendor_name, v.formatted_wa_number, t.sort_order, t.notes, t.template_key, t.origin,
	       (t.status <> 'selesai' AND t.end_date < ` + localDate(zoneArg, nowArg) + `) AS overdue
	FROM planner_tasks t
	LEFT JOIN planner_vendors v ON v.id = t.vendor_id AND v.project_id = t.project_id`
}

func scanTask(row pgx.Row) (domain.Task, error) {
	var t domain.Task
	err := row.Scan(&t.ID, &t.ProjectID, &t.Title, &t.SubEventScope, &t.Category, &t.PhaseOrder, &t.PhaseName,
		&t.StartDate, &t.EndDate, &t.DurationDays, &t.Status, &t.PIC,
		&t.VendorID, &t.VendorName, &t.VendorWA, &t.SortOrder, &t.Notes, &t.TemplateKey, &t.Origin, &t.Overdue)

	return t, err
}

func (r *PostgresRepository) ListTasks(ctx context.Context, projectID uuid.UUID, f domain.TaskFilter, at domain.At) ([]domain.Task, error) {
	sql := taskSelect(2, 3) + ` WHERE t.project_id = $1`
	args := []any{projectID, at.Zone, at.Now}
	add := func(cond string, v any) {
		args = append(args, v)
		sql += fmt.Sprintf(" AND "+cond, len(args))
	}
	if f.Status != "" {
		add("t.status = $%d", f.Status)
	}
	if f.Category != "" {
		add("t.category = $%d", f.Category)
	}
	if f.Scope != nil {
		add("t.sub_event_scope = $%d", *f.Scope)
	}
	if f.Phase != nil {
		add("t.phase_order = $%d", *f.Phase)
	}
	sql += ` ORDER BY t.sort_order, t.created_at, t.id`

	rows, err := r.q(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.Task{}
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}

	return out, rows.Err()
}

func (r *PostgresRepository) SummarizeTasks(ctx context.Context, projectID uuid.UUID, at domain.At) (domain.TaskSummary, error) {
	var s domain.TaskSummary
	err := r.q(ctx).QueryRow(ctx, `
		SELECT count(*),
		       count(*) FILTER (WHERE status = 'selesai'),
		       count(*) FILTER (WHERE status = 'proses'),
		       count(*) FILTER (WHERE status = 'belum'),
		       count(*) FILTER (WHERE status <> 'selesai' AND end_date < `+localDate(2, 3)+`)
		FROM planner_tasks WHERE project_id = $1`, projectID, at.Zone, at.Now).
		Scan(&s.Total, &s.Completed, &s.InProgress, &s.NotStarted, &s.Overdue)

	return s, err
}

func (r *PostgresRepository) TaskByID(ctx context.Context, projectID, id uuid.UUID, at domain.At) (domain.Task, error) {
	t, err := scanTask(r.q(ctx).QueryRow(ctx, taskSelect(3, 4)+` WHERE t.project_id = $1 AND t.id = $2`, projectID, id, at.Zone, at.Now))

	return t, notFound(err)
}

func (r *PostgresRepository) CountTasks(ctx context.Context, projectID uuid.UUID) (int, error) {
	var n int
	err := r.q(ctx).QueryRow(ctx, `SELECT count(*) FROM planner_tasks WHERE project_id=$1`, projectID).Scan(&n)

	return n, err
}

func (r *PostgresRepository) NextTaskSortOrder(ctx context.Context, projectID uuid.UUID) (int, error) {
	var n int
	err := r.q(ctx).QueryRow(ctx, `SELECT COALESCE(max(sort_order), 0) + 1 FROM planner_tasks WHERE project_id=$1`, projectID).Scan(&n)

	return n, err
}

func (r *PostgresRepository) LastTaskPhase(ctx context.Context, projectID uuid.UUID) (int, string, bool, error) {
	var order int
	var name string
	err := r.q(ctx).QueryRow(ctx, `
		SELECT phase_order, phase_name FROM planner_tasks WHERE project_id=$1
		ORDER BY sort_order DESC, created_at DESC, id LIMIT 1`, projectID).Scan(&order, &name)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, "", false, nil
	}

	return order, name, err == nil, err
}

func (r *PostgresRepository) PhaseName(ctx context.Context, projectID uuid.UUID, order int) (string, bool, error) {
	var name string
	err := r.q(ctx).QueryRow(ctx, `
		SELECT phase_name FROM planner_tasks WHERE project_id=$1 AND phase_order=$2 AND phase_name <> ''
		ORDER BY sort_order LIMIT 1`, projectID, order).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}

	return name, err == nil, err
}

const insertTaskSQL = `
	INSERT INTO planner_tasks (id, project_id, title, sub_event_scope, category, phase_order, phase_name, start_date, end_date,
		status, pic, vendor_id, sort_order, notes, template_key, origin)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8::date,$9::date,$10,$11,$12,$13,$14,$15,$16)`

func taskInsertArgs(t *domain.Task) []any {
	return []any{t.ID, t.ProjectID, t.Title, t.SubEventScope, t.Category, t.PhaseOrder, t.PhaseName, dateStr(t.StartDate), dateStr(t.EndDate),
		string(t.Status), t.PIC, t.VendorID, t.SortOrder, t.Notes, t.TemplateKey, string(t.Origin)}
}

func (r *PostgresRepository) InsertTask(ctx context.Context, t *domain.Task) error {
	_, err := r.q(ctx).Exec(ctx, insertTaskSQL, taskInsertArgs(t)...)

	return mapErr(err)
}

func (r *PostgresRepository) UpdateTask(ctx context.Context, t *domain.Task) error {
	tag, err := r.q(ctx).Exec(ctx, `
		UPDATE planner_tasks
		SET title=$3, sub_event_scope=$4, category=$5, phase_order=$6, phase_name=$7, start_date=$8::date, end_date=$9::date,
		    status=$10, pic=$11, vendor_id=$12, notes=$13, updated_at=now()
		WHERE id=$1 AND project_id=$2`,
		t.ID, t.ProjectID, t.Title, t.SubEventScope, t.Category, t.PhaseOrder, t.PhaseName, dateStr(t.StartDate), dateStr(t.EndDate),
		string(t.Status), t.PIC, t.VendorID, t.Notes)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *PostgresRepository) SetTaskStatus(ctx context.Context, projectID, id uuid.UUID, status domain.TaskStatus) error {
	tag, err := r.q(ctx).Exec(ctx, `UPDATE planner_tasks SET status=$3, updated_at=now() WHERE id=$1 AND project_id=$2`, id, projectID, string(status))
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *PostgresRepository) DeleteTask(ctx context.Context, projectID, id uuid.UUID) error {
	tag, err := r.q(ctx).Exec(ctx, `DELETE FROM planner_tasks WHERE id=$1 AND project_id=$2`, id, projectID)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func reorderArgs(ids []uuid.UUID) ([]string, []int32) {
	idStrs := make([]string, len(ids))
	orders := make([]int32, len(ids))
	for i, id := range ids {
		idStrs[i], orders[i] = id.String(), int32(i+1)
	}

	return idStrs, orders
}

func (r *PostgresRepository) ReorderTasks(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) (int, error) {
	idStrs, orders := reorderArgs(ids)
	tag, err := r.q(ctx).Exec(ctx, `
		UPDATE planner_tasks t SET sort_order = v.ord, updated_at = now()
		FROM unnest($2::text[]::uuid[], $3::int[]) AS v(id, ord)
		WHERE t.project_id = $1 AND t.id = v.id`, projectID, idStrs, orders)
	if err != nil {
		return 0, mapErr(err)
	}

	return int(tag.RowsAffected()), nil
}

// shiftable: template tasks that are not done (the same rule as menu 01's date change).
const shiftable = `project_id=$1 AND status <> 'selesai' AND template_key IS NOT NULL`

func (r *PostgresRepository) ShiftBounds(ctx context.Context, projectID uuid.UUID) (int, time.Time, time.Time, error) {
	var n int
	var lo, hi *time.Time
	err := r.q(ctx).QueryRow(ctx, `SELECT count(*), min(start_date), max(end_date) FROM planner_tasks WHERE `+shiftable, projectID).Scan(&n, &lo, &hi)
	if err != nil || n == 0 || lo == nil || hi == nil {
		return n, time.Time{}, time.Time{}, err
	}

	return n, *lo, *hi, nil
}

func (r *PostgresRepository) ShiftTasks(ctx context.Context, projectID uuid.UUID, days int) (int, error) {
	tag, err := r.q(ctx).Exec(ctx, `
		UPDATE planner_tasks SET start_date = start_date + $2::int, end_date = end_date + $2::int, updated_at = now()
		WHERE `+shiftable, projectID, days)
	if err != nil {
		return 0, mapErr(err)
	}

	return int(tag.RowsAffected()), nil
}

func (r *PostgresRepository) CountOverdueShifted(ctx context.Context, projectID uuid.UUID, today time.Time) (int, error) {
	var n int
	err := r.q(ctx).QueryRow(ctx, `SELECT count(*) FROM planner_tasks WHERE `+shiftable+` AND end_date < $2::date`, projectID, dateStr(today)).Scan(&n)

	return n, err
}

func (r *PostgresRepository) TemplateTaskKeys(ctx context.Context, projectID uuid.UUID) (map[string]bool, error) {
	return r.keys(ctx, `SELECT template_key FROM planner_tasks WHERE project_id=$1 AND template_key IS NOT NULL`, projectID)
}

func (r *PostgresRepository) keys(ctx context.Context, sql string, projectID uuid.UUID) (map[string]bool, error) {
	rows, err := r.q(ctx).Query(ctx, sql, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]bool{}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		out[k] = true
	}

	return out, rows.Err()
}

func (r *PostgresRepository) InsertTemplateTask(ctx context.Context, t *domain.Task) (bool, error) {
	tag, err := r.q(ctx).Exec(ctx, insertTaskSQL+` ON CONFLICT (project_id, template_key) WHERE template_key IS NOT NULL DO NOTHING`, taskInsertArgs(t)...)
	if err != nil {
		return false, mapErr(err)
	}

	return tag.RowsAffected() == 1, nil
}

func (r *PostgresRepository) CountProgressTasks(ctx context.Context, projectID uuid.UUID) (int, error) {
	var n int
	err := r.q(ctx).QueryRow(ctx, `
		SELECT count(*) FROM planner_tasks
		WHERE project_id=$1 AND template_key IS NOT NULL AND (status <> 'belum' OR vendor_id IS NOT NULL)`, projectID).Scan(&n)

	return n, err
}

func (r *PostgresRepository) DeleteTemplateTasks(ctx context.Context, projectID uuid.UUID) (int, error) {
	tag, err := r.q(ctx).Exec(ctx, `DELETE FROM planner_tasks WHERE project_id=$1 AND template_key IS NOT NULL`, projectID)
	if err != nil {
		return 0, mapErr(err)
	}

	return int(tag.RowsAffected()), nil
}

func (r *PostgresRepository) DeleteAllTasks(ctx context.Context, projectID uuid.UUID) (int, error) {
	tag, err := r.q(ctx).Exec(ctx, `DELETE FROM planner_tasks WHERE project_id=$1`, projectID)
	if err != nil {
		return 0, mapErr(err)
	}

	return int(tag.RowsAffected()), nil
}

// ---------- documents ----------

// docSelect joins the budget line that points at the document, if any.
const docSelect = `SELECT d.id, d.project_id, d.document_name, d.subject_role, d.quantity_detail, d.nominal_cost, d.status,
	       d.sort_order, d.notes, d.template_key, d.origin, b.id
	FROM planner_admin_docs d
	LEFT JOIN LATERAL (SELECT id FROM planner_budget_items WHERE project_id = d.project_id AND admin_doc_id = d.id
	                   ORDER BY created_at, id LIMIT 1) b ON true`

func scanDoc(row pgx.Row) (domain.AdminDoc, error) {
	var d domain.AdminDoc
	err := row.Scan(&d.ID, &d.ProjectID, &d.DocumentName, &d.SubjectRole, &d.QuantityDetail, &d.NominalCost, &d.Status,
		&d.SortOrder, &d.Notes, &d.TemplateKey, &d.Origin, &d.BudgetItemID)

	return d, err
}

func (r *PostgresRepository) ListDocs(ctx context.Context, projectID uuid.UUID, f domain.DocFilter) ([]domain.AdminDoc, error) {
	sql := docSelect + ` WHERE d.project_id = $1`
	args := []any{projectID}
	if f.Subject != "" {
		args = append(args, f.Subject)
		sql += fmt.Sprintf(" AND d.subject_role = $%d", len(args))
	}
	if f.Status != "" {
		args = append(args, f.Status)
		sql += fmt.Sprintf(" AND d.status = $%d", len(args))
	}
	sql += ` ORDER BY d.sort_order, d.created_at, d.id`

	rows, err := r.q(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.AdminDoc{}
	for rows.Next() {
		d, err := scanDoc(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}

	return out, rows.Err()
}

func (r *PostgresRepository) SummarizeDocs(ctx context.Context, projectID uuid.UUID) (domain.DocSummary, error) {
	var s domain.DocSummary
	err := r.q(ctx).QueryRow(ctx, `
		SELECT count(*), count(*) FILTER (WHERE status = 'selesai') FROM planner_admin_docs WHERE project_id = $1`, projectID).
		Scan(&s.Total, &s.Completed)

	return s, err
}

func (r *PostgresRepository) DocByID(ctx context.Context, projectID, id uuid.UUID) (domain.AdminDoc, error) {
	d, err := scanDoc(r.q(ctx).QueryRow(ctx, docSelect+` WHERE d.project_id = $1 AND d.id = $2`, projectID, id))

	return d, notFound(err)
}

func (r *PostgresRepository) LockDoc(ctx context.Context, projectID, id uuid.UUID) (domain.AdminDoc, error) {
	var d domain.AdminDoc
	err := r.q(ctx).QueryRow(ctx, `
		SELECT id, project_id, document_name, subject_role, quantity_detail, nominal_cost, status, sort_order, notes, template_key, origin
		FROM planner_admin_docs WHERE id = $1 AND project_id = $2 FOR UPDATE`, id, projectID).
		Scan(&d.ID, &d.ProjectID, &d.DocumentName, &d.SubjectRole, &d.QuantityDetail, &d.NominalCost, &d.Status,
			&d.SortOrder, &d.Notes, &d.TemplateKey, &d.Origin)

	return d, notFound(err)
}

func (r *PostgresRepository) CountDocs(ctx context.Context, projectID uuid.UUID) (int, error) {
	var n int
	err := r.q(ctx).QueryRow(ctx, `SELECT count(*) FROM planner_admin_docs WHERE project_id=$1`, projectID).Scan(&n)

	return n, err
}

func (r *PostgresRepository) NextDocSortOrder(ctx context.Context, projectID uuid.UUID) (int, error) {
	var n int
	err := r.q(ctx).QueryRow(ctx, `SELECT COALESCE(max(sort_order), 0) + 1 FROM planner_admin_docs WHERE project_id=$1`, projectID).Scan(&n)

	return n, err
}

const insertDocSQL = `
	INSERT INTO planner_admin_docs (id, project_id, document_name, subject_role, quantity_detail, nominal_cost, status, sort_order, notes, template_key, origin)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`

func docInsertArgs(d *domain.AdminDoc) []any {
	return []any{d.ID, d.ProjectID, d.DocumentName, d.SubjectRole, d.QuantityDetail, d.NominalCost, string(d.Status), d.SortOrder, d.Notes, d.TemplateKey, string(d.Origin)}
}

func (r *PostgresRepository) InsertDoc(ctx context.Context, d *domain.AdminDoc) error {
	_, err := r.q(ctx).Exec(ctx, insertDocSQL, docInsertArgs(d)...)

	return mapErr(err)
}

func (r *PostgresRepository) UpdateDoc(ctx context.Context, d *domain.AdminDoc) error {
	tag, err := r.q(ctx).Exec(ctx, `
		UPDATE planner_admin_docs
		SET document_name=$3, subject_role=$4, quantity_detail=$5, nominal_cost=$6, notes=$7, updated_at=now()
		WHERE id=$1 AND project_id=$2`,
		d.ID, d.ProjectID, d.DocumentName, d.SubjectRole, d.QuantityDetail, d.NominalCost, d.Notes)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *PostgresRepository) SetDocStatus(ctx context.Context, projectID, id uuid.UUID, status domain.DocStatus) error {
	tag, err := r.q(ctx).Exec(ctx, `UPDATE planner_admin_docs SET status=$3, updated_at=now() WHERE id=$1 AND project_id=$2`, id, projectID, string(status))
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *PostgresRepository) DeleteDoc(ctx context.Context, projectID, id uuid.UUID) error {
	tag, err := r.q(ctx).Exec(ctx, `DELETE FROM planner_admin_docs WHERE id=$1 AND project_id=$2`, id, projectID)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *PostgresRepository) ReorderDocs(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) (int, error) {
	idStrs, orders := reorderArgs(ids)
	tag, err := r.q(ctx).Exec(ctx, `
		UPDATE planner_admin_docs d SET sort_order = v.ord, updated_at = now()
		FROM unnest($2::text[]::uuid[], $3::int[]) AS v(id, ord)
		WHERE d.project_id = $1 AND d.id = v.id`, projectID, idStrs, orders)
	if err != nil {
		return 0, mapErr(err)
	}

	return int(tag.RowsAffected()), nil
}

func (r *PostgresRepository) DocTemplateKeys(ctx context.Context, projectID uuid.UUID) (map[string]bool, error) {
	return r.keys(ctx, `SELECT template_key FROM planner_admin_docs WHERE project_id=$1 AND template_key IS NOT NULL`, projectID)
}

func (r *PostgresRepository) InsertTemplateDoc(ctx context.Context, d *domain.AdminDoc) (bool, error) {
	tag, err := r.q(ctx).Exec(ctx, insertDocSQL+` ON CONFLICT (project_id, template_key) WHERE template_key IS NOT NULL DO NOTHING`, docInsertArgs(d)...)
	if err != nil {
		return false, mapErr(err)
	}

	return tag.RowsAffected() == 1, nil
}

func (r *PostgresRepository) DocsProgress(ctx context.Context, projectID uuid.UUID) (int, int, error) {
	var finished, linked int
	err := r.q(ctx).QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE d.status = 'selesai'),
		       count(*) FILTER (WHERE EXISTS (SELECT 1 FROM planner_budget_items b WHERE b.admin_doc_id = d.id))
		FROM planner_admin_docs d WHERE d.project_id = $1 AND d.template_key IS NOT NULL`, projectID).Scan(&finished, &linked)

	return finished, linked, err
}

func (r *PostgresRepository) DeleteTemplateDocs(ctx context.Context, projectID uuid.UUID) (int, error) {
	tag, err := r.q(ctx).Exec(ctx, `DELETE FROM planner_admin_docs WHERE project_id=$1 AND template_key IS NOT NULL`, projectID)
	if err != nil {
		return 0, mapErr(err)
	}

	return int(tag.RowsAffected()), nil
}

// ---------- sync with the budget ----------
//
// planner_budget_items belongs to menu 03; this is the second deliberate SQL
// exception of the contract (kontrak 1.1, 02 BE 4.5). Menu 03 does not exist
// yet but its migration (008) is released, so it can be built and tested now.

func (r *PostgresRepository) LinkedBudgetItem(ctx context.Context, projectID, docID uuid.UUID) (domain.BudgetItemRef, bool, error) {
	var ref domain.BudgetItemRef
	err := r.q(ctx).QueryRow(ctx,
		`SELECT id, initial_estimate FROM planner_budget_items WHERE project_id=$1 AND admin_doc_id=$2 LIMIT 1`, projectID, docID).
		Scan(&ref.ID, &ref.Estimate)
	if errors.Is(err, pgx.ErrNoRows) {
		return ref, false, nil
	}

	return ref, err == nil, err
}

func (r *PostgresRepository) UnlinkedBudgetItem(ctx context.Context, projectID uuid.UUID, scope, itemKey string) (domain.BudgetItemRef, bool, error) {
	var ref domain.BudgetItemRef
	err := r.q(ctx).QueryRow(ctx, `
		SELECT id, initial_estimate FROM planner_budget_items
		WHERE project_id=$1 AND sub_event_scope=$2 AND item_key=$3 AND admin_doc_id IS NULL FOR UPDATE`, projectID, scope, itemKey).
		Scan(&ref.ID, &ref.Estimate)
	if errors.Is(err, pgx.ErrNoRows) {
		return ref, false, nil
	}

	return ref, err == nil, err
}

func (r *PostgresRepository) LinkBudgetItem(ctx context.Context, projectID, itemID, docID uuid.UUID, cost int64) error {
	tag, err := r.q(ctx).Exec(ctx, `
		UPDATE planner_budget_items
		SET admin_doc_id = $3, initial_estimate = CASE WHEN initial_estimate = 0 THEN $4 ELSE initial_estimate END, updated_at = now()
		WHERE id = $1 AND project_id = $2`, itemID, projectID, docID, cost)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// maxBudgetItemName is the CHECK on planner_budget_items.item_name; a document
// name can be 255 characters, so the name is cut (left() counts characters).
const maxBudgetItemName = 150

func (r *PostgresRepository) InsertBudgetItem(ctx context.Context, projectID uuid.UUID, scope, name string, docID uuid.UUID, cost int64) (uuid.UUID, error) {
	id := uuid.New()
	_, err := r.q(ctx).Exec(ctx, `
		INSERT INTO planner_budget_items (id, project_id, sub_event_scope, category, item_name, source, admin_doc_id, initial_estimate, sort_order)
		VALUES ($1, $2, $3, 'Legalitas', left($4, `+fmt.Sprint(maxBudgetItemName)+`), 'admin_doc', $5, $6,
		        (SELECT COALESCE(max(sort_order), 0) + 1 FROM planner_budget_items WHERE project_id = $2))`,
		id, projectID, scope, name, docID, cost)
	if err != nil {
		return uuid.Nil, mapErr(err)
	}

	return id, nil
}

// ---------- dashboard widgets and the sample dataset ----------

func (r *PostgresRepository) TimelineSummary(ctx context.Context, projectID uuid.UUID, at domain.At) (pd.TimelineSummary, error) {
	s := pd.TimelineSummary{Urgent: []pd.UrgentTask{}}
	err := r.q(ctx).QueryRow(ctx, `
		SELECT count(*), count(*) FILTER (WHERE status = 'selesai'), count(*) FILTER (WHERE status = 'proses'),
		       count(*) FILTER (WHERE status <> 'selesai' AND end_date < `+localDate(2, 3)+`)
		FROM planner_tasks WHERE project_id = $1`, projectID, at.Zone, at.Now).
		Scan(&s.Total, &s.Completed, &s.InProgress, &s.Overdue)
	if err != nil {
		return s, err
	}

	rows, err := r.q(ctx).Query(ctx, `
		SELECT id, title, end_date, end_date - `+localDate(2, 3)+` AS days_left
		FROM planner_tasks
		WHERE project_id = $1 AND status <> 'selesai' AND end_date <= `+localDate(2, 3)+` + 7
		ORDER BY end_date, sort_order LIMIT 3`, projectID, at.Zone, at.Now)
	if err != nil {
		return s, err
	}
	defer rows.Close()

	for rows.Next() {
		var u pd.UrgentTask
		var due time.Time
		if err := rows.Scan(&u.ID, &u.Title, &due, &u.DaysLeft); err != nil {
			return s, err
		}
		u.DueDate = dateStr(due)
		s.Urgent = append(s.Urgent, u)
	}

	return s, rows.Err()
}

func (r *PostgresRepository) DocSummaryWidget(ctx context.Context, projectID uuid.UUID) (int, int, []string, error) {
	var completed, total int
	err := r.q(ctx).QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE status = 'selesai'), count(*) FROM planner_admin_docs WHERE project_id = $1`, projectID).
		Scan(&completed, &total)
	if err != nil {
		return 0, 0, nil, err
	}

	rows, err := r.q(ctx).Query(ctx, `
		SELECT document_name FROM planner_admin_docs WHERE project_id = $1 AND status = 'belum' ORDER BY sort_order, created_at, id LIMIT 5`, projectID)
	if err != nil {
		return 0, 0, nil, err
	}
	defer rows.Close()

	pending := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return 0, 0, nil, err
		}
		pending = append(pending, name)
	}

	return completed, total, pending, rows.Err()
}

func (r *PostgresRepository) ApplySampleTaskStatus(ctx context.Context, projectID uuid.UUID) error {
	_, err := r.q(ctx).Exec(ctx, `
		UPDATE planner_tasks
		SET status = CASE WHEN sort_order <= $2 THEN 'selesai' WHEN sort_order <= $3 THEN 'proses' ELSE 'belum' END, updated_at = now()
		WHERE project_id = $1 AND origin = 'pdf'`, projectID, domain.SampleTasksDoneUpTo, domain.SampleTasksInProgressUpTo)

	return mapErr(err)
}

func (r *PostgresRepository) ApplySampleDocStatus(ctx context.Context, projectID uuid.UUID, doneKeys []string) error {
	_, err := r.q(ctx).Exec(ctx, `
		UPDATE planner_admin_docs SET status = 'selesai', updated_at = now()
		WHERE project_id = $1 AND template_key = ANY($2::text[])`, projectID, doneKeys)

	return mapErr(err)
}

func (r *PostgresRepository) SetSessionDateFromTask(ctx context.Context, projectID uuid.UUID, scope, templateKey string) error {
	_, err := r.q(ctx).Exec(ctx, `
		UPDATE planner_event_sessions s SET session_date = t.start_date, updated_at = now()
		FROM planner_tasks t
		WHERE s.project_id = $1 AND s.scope_key = $2 AND t.project_id = $1 AND t.template_key = $3`, projectID, scope, templateKey)

	return mapErr(err)
}
