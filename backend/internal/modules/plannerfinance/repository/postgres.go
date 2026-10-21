// Package repository is the PostgreSQL side of planner menu 03. The pool runs
// pgx in QueryExecModeSimpleProtocol, so results arrive as text: NULL columns
// scan into pointers, DATE into time.Time (00:00 UTC), and arrays and dates go
// in as strings with explicit casts. pgx also rejects an argument a statement
// does not use, so every query is given exactly the arguments it mentions.
//
// All figures of budget lines are read from the view planner_budget_lines,
// never from the table: a line that rolls up the shopping list has its estimate
// and paid there only. The view cannot be locked (FOR UPDATE on a nullable side
// of an outer join), so writers lock the base row first (LockBudgetItem).
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
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerfinance/domain"
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
// sentinels. The foreign keys into planner_event_sessions, planner_contributors
// and planner_vendors-less ones are deferred, so a wrong reference normally shows
// up at COMMIT (the shared TxManager maps that one); this covers the
// statement-time case.
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
	case "23505":
		switch pg.ConstraintName {
		case "planner_budget_items_vendor_idx":
			return domain.ErrVendorAlreadyLinked
		case "planner_budget_items_key_idx", "planner_budget_scenario_items_key_idx":
			return domain.ErrKeyTaken
		}

		return domain.ErrConflict
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

func affected(tag pgconn.CommandTag, err error) error {
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func dateStr(t time.Time) string { return t.Format(pd.DateLayout) }

func dateStrPtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := dateStr(*t)

	return &s
}

// ---------- project, sessions, vendors ----------

func (r *PostgresRepository) LockProject(ctx context.Context, projectID uuid.UUID) error {
	var id uuid.UUID
	err := r.q(ctx).QueryRow(ctx, `SELECT id FROM planner_projects WHERE id=$1 FOR UPDATE`, projectID).Scan(&id)

	return notFound(err)
}

func (r *PostgresRepository) Sessions(ctx context.Context, projectID uuid.UUID) ([]domain.SessionRef, error) {
	rows, err := r.q(ctx).Query(ctx,
		`SELECT scope_key, name, is_system FROM planner_event_sessions WHERE project_id=$1 ORDER BY sort_order, created_at, id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.SessionRef{}
	for rows.Next() {
		var s domain.SessionRef
		if err := rows.Scan(&s.ScopeKey, &s.Name, &s.IsSystem); err != nil {
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

func (r *PostgresRepository) SetProjectTargets(ctx context.Context, projectID uuid.UUID, targetBudget *int64, clearSavings bool) error {
	tag, err := r.q(ctx).Exec(ctx, `
		UPDATE planner_projects
		SET target_budget = COALESCE($2::bigint, target_budget),
		    savings_target = CASE WHEN $3::boolean THEN NULL ELSE savings_target END,
		    updated_at = now()
		WHERE id = $1`, projectID, targetBudget, clearSavings)

	return affected(tag, err)
}

// ---------- scenarios ----------

const scenarioCols = `id, name, total_target, is_active, active_sub_events, preset_key`

func scanScenario(row pgx.Row) (domain.Scenario, error) {
	var s domain.Scenario
	err := row.Scan(&s.ID, &s.Name, &s.TotalTarget, &s.IsActive, &s.ActiveSubEvents, &s.PresetKey)
	if s.ActiveSubEvents == nil {
		s.ActiveSubEvents = []string{}
	}
	s.Items = []domain.ScenarioItem{}

	return s, err
}

func (r *PostgresRepository) scenarioItems(ctx context.Context, projectID uuid.UUID, scenarioID *uuid.UUID) (map[uuid.UUID][]domain.ScenarioItem, error) {
	sql := `SELECT scenario_id, id, sub_event_scope, item_key, category, item_name, cost_amount
	        FROM planner_budget_scenario_items WHERE project_id = $1`
	args := []any{projectID}
	if scenarioID != nil {
		sql += ` AND scenario_id = $2`
		args = append(args, *scenarioID)
	}
	sql += ` ORDER BY scenario_id, sort_order, id`

	rows, err := r.q(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[uuid.UUID][]domain.ScenarioItem{}
	for rows.Next() {
		var sid uuid.UUID
		var it domain.ScenarioItem
		if err := rows.Scan(&sid, &it.ID, &it.SubEventScope, &it.ItemKey, &it.Category, &it.ItemName, &it.CostAmount); err != nil {
			return nil, err
		}
		out[sid] = append(out[sid], it)
	}

	return out, rows.Err()
}

func (r *PostgresRepository) ListScenarios(ctx context.Context, projectID uuid.UUID) ([]domain.Scenario, error) {
	rows, err := r.q(ctx).Query(ctx,
		`SELECT `+scenarioCols+` FROM planner_budget_scenarios WHERE project_id=$1 ORDER BY total_target, created_at, id`, projectID)
	if err != nil {
		return nil, err
	}
	out := []domain.Scenario{}
	for rows.Next() {
		s, err := scanScenario(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, s)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	items, err := r.scenarioItems(ctx, projectID, nil)
	if err != nil {
		return nil, err
	}
	for i := range out {
		if its := items[out[i].ID]; its != nil {
			out[i].Items = its
		}
	}

	return out, nil
}

func (r *PostgresRepository) ScenarioByID(ctx context.Context, projectID, id uuid.UUID) (domain.Scenario, error) {
	s, err := scanScenario(r.q(ctx).QueryRow(ctx,
		`SELECT `+scenarioCols+` FROM planner_budget_scenarios WHERE id=$1 AND project_id=$2`, id, projectID))
	if err != nil {
		return domain.Scenario{}, notFound(err)
	}
	items, err := r.scenarioItems(ctx, projectID, &id)
	if err != nil {
		return domain.Scenario{}, err
	}
	if its := items[id]; its != nil {
		s.Items = its
	}

	return s, nil
}

func (r *PostgresRepository) CountScenarios(ctx context.Context, projectID uuid.UUID) (int, error) {
	var n int
	err := r.q(ctx).QueryRow(ctx, `SELECT count(*) FROM planner_budget_scenarios WHERE project_id=$1`, projectID).Scan(&n)

	return n, err
}

func (r *PostgresRepository) insertScenarioItem(ctx context.Context, projectID, scenarioID uuid.UUID, it *domain.ScenarioItem, sortOrder int) error {
	_, err := r.q(ctx).Exec(ctx, `
		INSERT INTO planner_budget_scenario_items (id, scenario_id, project_id, sub_event_scope, item_key, category, item_name, cost_amount, sort_order)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		it.ID, scenarioID, projectID, it.SubEventScope, it.ItemKey, it.Category, it.ItemName, it.CostAmount, sortOrder)

	return mapErr(err)
}

func (r *PostgresRepository) InsertScenario(ctx context.Context, projectID uuid.UUID, sc *domain.Scenario) error {
	_, err := r.q(ctx).Exec(ctx, `
		INSERT INTO planner_budget_scenarios (id, project_id, name, total_target, is_active, active_sub_events, preset_key)
		VALUES ($1,$2,$3,$4,$5,$6::text[],$7)`,
		sc.ID, projectID, sc.Name, sc.TotalTarget, sc.IsActive, sc.ActiveSubEvents, sc.PresetKey)
	if err != nil {
		return mapErr(err)
	}
	for i := range sc.Items {
		if err := r.insertScenarioItem(ctx, projectID, sc.ID, &sc.Items[i], i+1); err != nil {
			return err
		}
	}

	return nil
}

func (r *PostgresRepository) UpdateScenario(ctx context.Context, projectID, id uuid.UUID, name string, total int64, active []string) error {
	tag, err := r.q(ctx).Exec(ctx, `
		UPDATE planner_budget_scenarios SET name=$3, total_target=$4, active_sub_events=$5::text[], updated_at=now()
		WHERE id=$1 AND project_id=$2`, id, projectID, name, total, active)

	return affected(tag, err)
}

func (r *PostgresRepository) DeleteScenario(ctx context.Context, projectID, id uuid.UUID) error {
	tag, err := r.q(ctx).Exec(ctx, `DELETE FROM planner_budget_scenarios WHERE id=$1 AND project_id=$2`, id, projectID)

	return affected(tag, err)
}

func (r *PostgresRepository) SetScenarioPreset(ctx context.Context, projectID, id uuid.UUID, presetKey *string, total int64) error {
	tag, err := r.q(ctx).Exec(ctx,
		`UPDATE planner_budget_scenarios SET preset_key=$3, total_target=$4, updated_at=now() WHERE id=$1 AND project_id=$2`, id, projectID, presetKey, total)

	return affected(tag, err)
}

func (r *PostgresRepository) ReplaceScenarioItems(ctx context.Context, projectID, scenarioID uuid.UUID, items []domain.ScenarioItem) error {
	if _, err := r.q(ctx).Exec(ctx, `DELETE FROM planner_budget_scenario_items WHERE scenario_id=$1 AND project_id=$2`, scenarioID, projectID); err != nil {
		return mapErr(err)
	}
	for i := range items {
		if err := r.insertScenarioItem(ctx, projectID, scenarioID, &items[i], i+1); err != nil {
			return err
		}
	}
	_, err := r.q(ctx).Exec(ctx, `UPDATE planner_budget_scenarios SET updated_at=now() WHERE id=$1 AND project_id=$2`, scenarioID, projectID)

	return mapErr(err)
}

func (r *PostgresRepository) CountScenarioItems(ctx context.Context, projectID, scenarioID uuid.UUID) (int, error) {
	var n int
	err := r.q(ctx).QueryRow(ctx,
		`SELECT count(*) FROM planner_budget_scenario_items WHERE scenario_id=$1 AND project_id=$2`, scenarioID, projectID).Scan(&n)

	return n, err
}

func (r *PostgresRepository) InsertScenarioItem(ctx context.Context, projectID, scenarioID uuid.UUID, it *domain.ScenarioItem) error {
	_, err := r.q(ctx).Exec(ctx, `
		INSERT INTO planner_budget_scenario_items (id, scenario_id, project_id, sub_event_scope, item_key, category, item_name, cost_amount, sort_order)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,
		        (SELECT COALESCE(max(sort_order), 0) + 1 FROM planner_budget_scenario_items WHERE scenario_id = $2))`,
		it.ID, scenarioID, projectID, it.SubEventScope, it.ItemKey, it.Category, it.ItemName, it.CostAmount)

	return mapErr(err)
}

func (r *PostgresRepository) ScenarioItemByID(ctx context.Context, projectID, scenarioID, itemID uuid.UUID) (domain.ScenarioItem, error) {
	var it domain.ScenarioItem
	err := r.q(ctx).QueryRow(ctx, `
		SELECT id, sub_event_scope, item_key, category, item_name, cost_amount
		FROM planner_budget_scenario_items WHERE id=$1 AND scenario_id=$2 AND project_id=$3`, itemID, scenarioID, projectID).
		Scan(&it.ID, &it.SubEventScope, &it.ItemKey, &it.Category, &it.ItemName, &it.CostAmount)

	return it, notFound(err)
}

func (r *PostgresRepository) UpdateScenarioItem(ctx context.Context, projectID, scenarioID uuid.UUID, it *domain.ScenarioItem) error {
	tag, err := r.q(ctx).Exec(ctx, `
		UPDATE planner_budget_scenario_items
		SET sub_event_scope=$4, item_key=$5, category=$6, item_name=$7, cost_amount=$8
		WHERE id=$1 AND scenario_id=$2 AND project_id=$3`,
		it.ID, scenarioID, projectID, it.SubEventScope, it.ItemKey, it.Category, it.ItemName, it.CostAmount)

	return affected(tag, err)
}

func (r *PostgresRepository) ScenarioKeyTaken(ctx context.Context, scenarioID uuid.UUID, scope, key string, except uuid.UUID) (bool, error) {
	var one int
	err := r.q(ctx).QueryRow(ctx, `
		SELECT 1 FROM planner_budget_scenario_items WHERE scenario_id=$1 AND sub_event_scope=$2 AND item_key=$3 AND id<>$4`,
		scenarioID, scope, key, except).Scan(&one)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}

	return err == nil, err
}

func (r *PostgresRepository) DeleteScenarioItem(ctx context.Context, projectID, scenarioID, itemID uuid.UUID) error {
	tag, err := r.q(ctx).Exec(ctx,
		`DELETE FROM planner_budget_scenario_items WHERE id=$1 AND scenario_id=$2 AND project_id=$3`, itemID, scenarioID, projectID)

	return affected(tag, err)
}

func (r *PostgresRepository) DeactivateScenarios(ctx context.Context, projectID uuid.UUID) error {
	_, err := r.q(ctx).Exec(ctx, `UPDATE planner_budget_scenarios SET is_active=false, updated_at=now() WHERE project_id=$1 AND is_active`, projectID)

	return mapErr(err)
}

func (r *PostgresRepository) ActivateScenario(ctx context.Context, projectID, id uuid.UUID) error {
	tag, err := r.q(ctx).Exec(ctx, `UPDATE planner_budget_scenarios SET is_active=true, updated_at=now() WHERE id=$1 AND project_id=$2`, id, projectID)

	return affected(tag, err)
}

func (r *PostgresRepository) ActiveScenario(ctx context.Context, projectID uuid.UUID) (uuid.UUID, string, int64, bool, error) {
	var id uuid.UUID
	var name string
	var total int64
	err := r.q(ctx).QueryRow(ctx,
		`SELECT id, name, total_target FROM planner_budget_scenarios WHERE project_id=$1 AND is_active`, projectID).Scan(&id, &name, &total)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, "", 0, false, nil
	}

	return id, name, total, err == nil, err
}

// AdjustRows is step 1 of the Smart Adjust algorithm (03 BE 3.2) as a plain
// SELECT, so apply-preview can run it without writing. A scenario item pairs
// with the line of the same (session, item_key); an item without a key pairs
// with the keyless line of the same name (the lowest sort_order when several).
func (r *PostgresRepository) AdjustRows(ctx context.Context, projectID, scenarioID uuid.UUID) ([]domain.AdjustRow, error) {
	rows, err := r.q(ctx).Query(ctx, `
		SELECT i.id, i.sub_event_scope, i.item_key, i.category, i.item_name, i.cost_amount,
		       b.id, b.initial_estimate,
		       COALESCE(b.rollup_source IS NOT NULL, false), COALESCE(b.pricing_mode = 'unit', false)
		FROM planner_budget_scenario_items i
		JOIN planner_budget_scenarios sc ON sc.id = i.scenario_id AND sc.project_id = i.project_id
		LEFT JOIN LATERAL (
		    SELECT x.id, x.initial_estimate, x.rollup_source, x.pricing_mode
		    FROM planner_budget_items x
		    WHERE x.project_id = $1 AND x.sub_event_scope = i.sub_event_scope
		      AND ((i.item_key IS NOT NULL AND x.item_key = i.item_key)
		        OR (i.item_key IS NULL AND x.item_key IS NULL AND lower(x.item_name) = lower(i.item_name)))
		    ORDER BY x.sort_order, x.id LIMIT 1) b ON true
		WHERE i.scenario_id = $2 AND i.project_id = $1
		  AND (cardinality(sc.active_sub_events) = 0 OR i.sub_event_scope = 'all' OR i.sub_event_scope = ANY (sc.active_sub_events))
		ORDER BY i.sort_order, i.id`, projectID, scenarioID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.AdjustRow{}
	for rows.Next() {
		var a domain.AdjustRow
		var from *int64
		if err := rows.Scan(&a.Item.ID, &a.Item.SubEventScope, &a.Item.ItemKey, &a.Item.Category, &a.Item.ItemName, &a.Item.CostAmount,
			&a.BudgetID, &from, &a.Rollup, &a.Unit); err != nil {
			return nil, err
		}
		if from != nil {
			a.From = *from
		}
		a.Item.Included = true
		out = append(out, a)
	}

	return out, rows.Err()
}

// ---------- budget lines ----------

func (r *PostgresRepository) LockBudgetItem(ctx context.Context, projectID, id uuid.UUID) error {
	var got uuid.UUID
	err := r.q(ctx).QueryRow(ctx, `SELECT id FROM planner_budget_items WHERE id=$2 AND project_id=$1 FOR UPDATE`, projectID, id).Scan(&got)

	return notFound(err)
}

const lineSelect = `
	SELECT l.id, l.sub_event_scope, l.category, l.item_name, l.item_key, l.source, l.vendor_id, v.vendor_name,
	       l.admin_doc_id, l.rollup_source, l.pricing_mode, (l.quantity * 100)::bigint, l.unit_label, l.unit_price, l.estimate,
	       l.dp_amount, l.term_1_amount, l.term_2_amount, l.dp_paid_at, l.term_1_paid_at, l.term_2_paid_at,
	       l.dp_receipt_url, l.term_1_receipt_url, l.term_2_receipt_url,
	       l.paid, l.remaining, l.payment_status, l.sort_order, l.notes
	FROM planner_budget_lines l
	LEFT JOIN planner_vendors v ON v.id = l.vendor_id AND v.project_id = l.project_id`

func scanLine(row pgx.Row) (domain.BudgetLine, error) {
	var l domain.BudgetLine
	var qx100 *int64
	var dpAt, t1At, t2At *time.Time
	err := row.Scan(&l.ID, &l.SubEventScope, &l.Category, &l.ItemName, &l.ItemKey, &l.Source, &l.VendorID, &l.VendorName,
		&l.AdminDocID, &l.RollupSource, &l.PricingMode, &qx100, &l.UnitLabel, &l.UnitPrice, &l.Estimate,
		&l.DPAmount, &l.Term1Amount, &l.Term2Amount, &dpAt, &t1At, &t2At,
		&l.DPReceiptURL, &l.Term1ReceiptURL, &l.Term2ReceiptURL,
		&l.Paid, &l.Remaining, &l.PaymentStatus, &l.SortOrder, &l.Notes)
	if qx100 != nil {
		q := domain.QuantityFloat(*qx100)
		l.Quantity = &q
	}
	l.DPPaidAt, l.Term1PaidAt, l.Term2PaidAt = dateStrPtr(dpAt), dateStrPtr(t1At), dateStrPtr(t2At)

	return l, err
}

func (r *PostgresRepository) ListLines(ctx context.Context, projectID uuid.UUID, scope *string) ([]domain.BudgetLine, error) {
	sql := lineSelect + ` WHERE l.project_id = $1`
	args := []any{projectID}
	if scope != nil {
		sql += ` AND l.sub_event_scope = $2`
		args = append(args, *scope)
	}
	sql += ` ORDER BY l.sort_order, l.created_at, l.id`

	rows, err := r.q(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.BudgetLine{}
	for rows.Next() {
		l, err := scanLine(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}

	return out, rows.Err()
}

func (r *PostgresRepository) LineByID(ctx context.Context, projectID, id uuid.UUID) (domain.BudgetLine, error) {
	l, err := scanLine(r.q(ctx).QueryRow(ctx, lineSelect+` WHERE l.project_id = $1 AND l.id = $2`, projectID, id))

	return l, notFound(err)
}

func (r *PostgresRepository) BudgetItemByID(ctx context.Context, projectID, id uuid.UUID) (domain.BudgetItem, error) {
	var it domain.BudgetItem
	err := r.q(ctx).QueryRow(ctx, `
		SELECT id, project_id, sub_event_scope, category, item_name, item_key, source, vendor_id, pricing_mode,
		       (quantity * 100)::bigint, unit_label, unit_price, initial_estimate, notes, sort_order, rollup_source IS NOT NULL
		FROM planner_budget_items WHERE id = $1 AND project_id = $2`, id, projectID).
		Scan(&it.ID, &it.ProjectID, &it.SubEventScope, &it.Category, &it.ItemName, &it.ItemKey, &it.Source, &it.VendorID, &it.PricingMode,
			&it.QuantityX100, &it.UnitLabel, &it.UnitPrice, &it.InitialEstimate, &it.Notes, &it.SortOrder, &it.Rollup)

	return it, notFound(err)
}

func (r *PostgresRepository) CountLines(ctx context.Context, projectID uuid.UUID) (int, error) {
	var n int
	err := r.q(ctx).QueryRow(ctx, `SELECT count(*) FROM planner_budget_items WHERE project_id=$1`, projectID).Scan(&n)

	return n, err
}

func (r *PostgresRepository) NextLineSortOrder(ctx context.Context, projectID uuid.UUID) (int, error) {
	var n int
	err := r.q(ctx).QueryRow(ctx, `SELECT COALESCE(max(sort_order), 0) + 1 FROM planner_budget_items WHERE project_id=$1`, projectID).Scan(&n)

	return n, err
}

const insertItemSQL = `
	INSERT INTO planner_budget_items (id, project_id, sub_event_scope, category, item_name, item_key, source, vendor_id,
		pricing_mode, quantity, unit_label, unit_price, initial_estimate, notes, sort_order)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10::bigint::numeric / 100,$11,$12,$13,$14,$15)`

func itemInsertArgs(it *domain.BudgetItem) []any {
	return []any{it.ID, it.ProjectID, it.SubEventScope, it.Category, it.ItemName, it.ItemKey, it.Source, it.VendorID,
		it.PricingMode, it.QuantityX100, it.UnitLabel, it.UnitPrice, it.InitialEstimate, it.Notes, it.SortOrder}
}

func (r *PostgresRepository) InsertBudgetItem(ctx context.Context, it *domain.BudgetItem) error {
	_, err := r.q(ctx).Exec(ctx, insertItemSQL, itemInsertArgs(it)...)

	return mapErr(err)
}

func (r *PostgresRepository) UpdateBudgetItem(ctx context.Context, it *domain.BudgetItem) error {
	tag, err := r.q(ctx).Exec(ctx, `
		UPDATE planner_budget_items
		SET sub_event_scope=$3, category=$4, item_name=$5, item_key=$6, vendor_id=$7, pricing_mode=$8,
		    quantity=$9::bigint::numeric / 100, unit_label=$10, unit_price=$11, initial_estimate=$12, notes=$13, updated_at=now()
		WHERE id=$1 AND project_id=$2`,
		it.ID, it.ProjectID, it.SubEventScope, it.Category, it.ItemName, it.ItemKey, it.VendorID, it.PricingMode,
		it.QuantityX100, it.UnitLabel, it.UnitPrice, it.InitialEstimate, it.Notes)

	return affected(tag, err)
}

func (r *PostgresRepository) BudgetKeyTaken(ctx context.Context, projectID uuid.UUID, scope, key string, except uuid.UUID) (bool, error) {
	var one int
	err := r.q(ctx).QueryRow(ctx, `
		SELECT 1 FROM planner_budget_items WHERE project_id=$1 AND sub_event_scope=$2 AND item_key=$3 AND id<>$4`,
		projectID, scope, key, except).Scan(&one)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}

	return err == nil, err
}

func (r *PostgresRepository) UpdateEstimates(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID, amounts []int64) error {
	if len(ids) == 0 {
		return nil
	}
	idStrs := make([]string, len(ids))
	for i, id := range ids {
		idStrs[i] = id.String()
	}
	_, err := r.q(ctx).Exec(ctx, `
		UPDATE planner_budget_items b SET initial_estimate = v.amount, updated_at = now()
		FROM unnest($2::text[]::uuid[], $3::bigint[]) AS v(id, amount)
		WHERE b.project_id = $1 AND b.id = v.id`, projectID, idStrs, amounts)

	return mapErr(err)
}

// stageColumns is the closed list the payment stage is chosen from; the column
// names are never built from client text.
var stageColumns = map[string][3]string{
	domain.StageDP:    {"dp_amount", "dp_paid_at", "dp_receipt_url"},
	domain.StageTerm1: {"term_1_amount", "term_1_paid_at", "term_1_receipt_url"},
	domain.StageTerm2: {"term_2_amount", "term_2_paid_at", "term_2_receipt_url"},
}

func (r *PostgresRepository) SetPayment(ctx context.Context, projectID, id uuid.UUID, stage string, amount int64, paidAt *time.Time, receiptURL *string) error {
	cols, ok := stageColumns[stage]
	if !ok {
		return fmt.Errorf("payment stage %q", stage)
	}
	tag, err := r.q(ctx).Exec(ctx, fmt.Sprintf(
		`UPDATE planner_budget_items SET %s=$3, %s=$4::date, %s=$5, updated_at=now() WHERE id=$1 AND project_id=$2`,
		cols[0], cols[1], cols[2]), id, projectID, amount, dateStrPtr(paidAt), receiptURL)

	return affected(tag, err)
}

func (r *PostgresRepository) DeleteBudgetItem(ctx context.Context, projectID, id uuid.UUID) error {
	tag, err := r.q(ctx).Exec(ctx, `DELETE FROM planner_budget_items WHERE id=$1 AND project_id=$2`, id, projectID)

	return affected(tag, err)
}

func (r *PostgresRepository) DeleteAllBudgetItems(ctx context.Context, projectID uuid.UUID) error {
	_, err := r.q(ctx).Exec(ctx, `DELETE FROM planner_budget_items WHERE project_id=$1`, projectID)

	return mapErr(err)
}

func (r *PostgresRepository) ReorderBudgetItems(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) (int, error) {
	idStrs := make([]string, len(ids))
	orders := make([]int32, len(ids))
	for i, id := range ids {
		idStrs[i], orders[i] = id.String(), int32(i+1)
	}
	tag, err := r.q(ctx).Exec(ctx, `
		UPDATE planner_budget_items b SET sort_order = v.ord, updated_at = now()
		FROM unnest($2::text[]::uuid[], $3::int[]) AS v(id, ord)
		WHERE b.project_id = $1 AND b.id = v.id`, projectID, idStrs, orders)
	if err != nil {
		return 0, mapErr(err)
	}

	return int(tag.RowsAffected()), nil
}

func (r *PostgresRepository) InsertTemplateLine(ctx context.Context, it *domain.BudgetItem) (bool, error) {
	tag, err := r.q(ctx).Exec(ctx, insertItemSQL+` ON CONFLICT (project_id, sub_event_scope, item_key) WHERE item_key IS NOT NULL DO NOTHING`, itemInsertArgs(it)...)
	if err != nil {
		return false, mapErr(err)
	}

	return tag.RowsAffected() == 1, nil
}

// templateWhere selects the template lines (optionally of one session) from the view alias l.
func templateWhere(scope *string) (string, []any) {
	if scope == nil {
		return `l.project_id = $1 AND l.source = 'template'`, nil
	}

	return `l.project_id = $1 AND l.source = 'template' AND l.sub_event_scope = $2`, []any{*scope}
}

const linkedPredicate = `(l.vendor_id IS NOT NULL OR l.admin_doc_id IS NOT NULL OR l.rollup_source IS NOT NULL
	OR EXISTS (SELECT 1 FROM planner_shopping_items s WHERE s.budget_item_id = l.id))`

func (r *PostgresRepository) TemplateStats(ctx context.Context, projectID uuid.UUID, scope *string) (domain.TemplateStats, error) {
	where, extra := templateWhere(scope)
	var s domain.TemplateStats
	err := r.q(ctx).QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE l.paid = 0),
		       count(*) FILTER (WHERE l.paid > 0),
		       count(*) FILTER (WHERE l.paid = 0 AND (l.estimate > 0 OR `+linkedPredicate+`)),
		       count(*) FILTER (WHERE l.paid = 0 AND `+linkedPredicate+`)
		FROM planner_budget_lines l WHERE `+where, append([]any{projectID}, extra...)...).
		Scan(&s.Deletable, &s.SkippedWithPayment, &s.AtRisk, &s.Linked)

	return s, err
}

func (r *PostgresRepository) DeleteTemplateLines(ctx context.Context, projectID uuid.UUID, scope *string) (int, error) {
	where, extra := templateWhere(scope)
	tag, err := r.q(ctx).Exec(ctx, `
		DELETE FROM planner_budget_items b USING planner_budget_lines l
		WHERE l.id = b.id AND b.project_id = l.project_id AND l.paid = 0 AND `+where, append([]any{projectID}, extra...)...)
	if err != nil {
		return 0, mapErr(err)
	}

	return int(tag.RowsAffected()), nil
}

// ---------- aggregates ----------

func (r *PostgresRepository) LineTotals(ctx context.Context, projectID uuid.UUID) (domain.LineTotals, error) {
	var t domain.LineTotals
	err := r.q(ctx).QueryRow(ctx, `
		SELECT COALESCE(sum(estimate), 0)::bigint, COALESCE(sum(paid), 0)::bigint,
		       COALESCE(sum(GREATEST(remaining, 0)), 0)::bigint, COALESCE(sum(GREATEST(-remaining, 0)), 0)::bigint,
		       count(*),
		       count(*) FILTER (WHERE estimate > 0 AND payment_status = 'settled'),
		       count(*) FILTER (WHERE estimate > 0 AND payment_status IN ('dp_paid', 'partial')),
		       count(*) FILTER (WHERE estimate > 0 AND payment_status = 'unpaid')
		FROM planner_budget_lines WHERE project_id = $1`, projectID).
		Scan(&t.Estimate, &t.Paid, &t.Remaining, &t.Overpaid, &t.Lines, &t.Status.Settled, &t.Status.Partial, &t.Status.Unpaid)

	return t, err
}

func (r *PostgresRepository) SavingsCollected(ctx context.Context, projectID uuid.UUID) (int64, error) {
	var n int64
	err := r.q(ctx).QueryRow(ctx, `SELECT COALESCE(sum(amount), 0)::bigint FROM planner_savings_entries WHERE project_id=$1`, projectID).Scan(&n)

	return n, err
}

func (r *PostgresRepository) MatrixRows(ctx context.Context, projectID uuid.UUID) ([]domain.MatrixRow, error) {
	rows, err := r.q(ctx).Query(ctx, `
		SELECT l.sub_event_scope,
		       sum(l.estimate)::bigint,
		       COALESCE(sum(l.dp_amount)     FILTER (WHERE l.rollup_source IS NULL), 0)::bigint,
		       COALESCE(sum(l.term_1_amount) FILTER (WHERE l.rollup_source IS NULL), 0)::bigint,
		       COALESCE(sum(l.term_2_amount) FILTER (WHERE l.rollup_source IS NULL), 0)::bigint,
		       COALESCE(sum(l.paid)          FILTER (WHERE l.rollup_source IS NOT NULL), 0)::bigint,
		       sum(l.paid)::bigint, sum(GREATEST(l.remaining, 0))::bigint, sum(GREATEST(-l.remaining, 0))::bigint,
		       count(*)
		FROM planner_budget_lines l
		LEFT JOIN planner_event_sessions s ON s.project_id = l.project_id AND s.scope_key = l.sub_event_scope
		WHERE l.project_id = $1
		GROUP BY l.sub_event_scope
		ORDER BY min(CASE WHEN s.is_system THEN 1000000 ELSE s.sort_order END), l.sub_event_scope`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.MatrixRow{}
	for rows.Next() {
		var m domain.MatrixRow
		if err := rows.Scan(&m.SubEventScope, &m.Estimate, &m.DP, &m.Term1, &m.Term2, &m.RollupPaid, &m.Paid, &m.Remaining, &m.Overpaid, &m.LineCount); err != nil {
			return nil, err
		}
		out = append(out, m)
	}

	return out, rows.Err()
}

// ---------- contributors and savings ----------

const contributorSelect = `
	SELECT c.id, c.name, c.role_label, COALESCE(sum(e.amount), 0)::bigint, count(e.id)
	FROM planner_contributors c
	LEFT JOIN planner_savings_entries e ON e.contributor_id = c.id AND e.project_id = c.project_id`

func (r *PostgresRepository) ListContributors(ctx context.Context, projectID uuid.UUID) ([]domain.Contributor, error) {
	rows, err := r.q(ctx).Query(ctx, contributorSelect+`
		WHERE c.project_id = $1 GROUP BY c.id ORDER BY c.sort_order, c.created_at, c.id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.Contributor{}
	for rows.Next() {
		var c domain.Contributor
		if err := rows.Scan(&c.ID, &c.Name, &c.RoleLabel, &c.Total, &c.EntryCount); err != nil {
			return nil, err
		}
		out = append(out, c)
	}

	return out, rows.Err()
}

func (r *PostgresRepository) ContributorByID(ctx context.Context, projectID, id uuid.UUID) (domain.Contributor, error) {
	var c domain.Contributor
	err := r.q(ctx).QueryRow(ctx, contributorSelect+` WHERE c.project_id = $1 AND c.id = $2 GROUP BY c.id`, projectID, id).
		Scan(&c.ID, &c.Name, &c.RoleLabel, &c.Total, &c.EntryCount)

	return c, notFound(err)
}

func (r *PostgresRepository) CountContributors(ctx context.Context, projectID uuid.UUID) (int, error) {
	var n int
	err := r.q(ctx).QueryRow(ctx, `SELECT count(*) FROM planner_contributors WHERE project_id=$1`, projectID).Scan(&n)

	return n, err
}

func (r *PostgresRepository) InsertContributor(ctx context.Context, projectID uuid.UUID, c *domain.Contributor) error {
	_, err := r.q(ctx).Exec(ctx, `
		INSERT INTO planner_contributors (id, project_id, name, role_label, sort_order)
		VALUES ($1,$2,$3,$4,(SELECT COALESCE(max(sort_order), 0) + 1 FROM planner_contributors WHERE project_id = $2))`,
		c.ID, projectID, c.Name, c.RoleLabel)

	return mapErr(err)
}

func (r *PostgresRepository) UpdateContributor(ctx context.Context, projectID uuid.UUID, c *domain.Contributor) error {
	tag, err := r.q(ctx).Exec(ctx,
		`UPDATE planner_contributors SET name=$3, role_label=$4, updated_at=now() WHERE id=$1 AND project_id=$2`, c.ID, projectID, c.Name, c.RoleLabel)

	return affected(tag, err)
}

func (r *PostgresRepository) DeleteContributor(ctx context.Context, projectID, id uuid.UUID) error {
	tag, err := r.q(ctx).Exec(ctx, `DELETE FROM planner_contributors WHERE id=$1 AND project_id=$2`, id, projectID)

	return affected(tag, err)
}

const entrySelect = `
	SELECT e.id, e.contributor_id, c.name, c.role_label, e.entry_date, e.amount, e.notes
	FROM planner_savings_entries e
	JOIN planner_contributors c ON c.id = e.contributor_id AND c.project_id = e.project_id`

func scanEntry(row pgx.Row) (domain.SavingsEntry, error) {
	var e domain.SavingsEntry
	var date time.Time
	err := row.Scan(&e.ID, &e.ContributorID, &e.ContributorName, &e.RoleLabel, &date, &e.Amount, &e.Notes)
	e.EntryDate = dateStr(date)

	return e, err
}

func (r *PostgresRepository) ListEntries(ctx context.Context, projectID uuid.UUID, contributor *uuid.UUID) ([]domain.SavingsEntry, error) {
	sql := entrySelect + ` WHERE e.project_id = $1`
	args := []any{projectID}
	if contributor != nil {
		sql += ` AND e.contributor_id = $2`
		args = append(args, *contributor)
	}
	sql += ` ORDER BY e.entry_date, e.created_at, e.id`

	rows, err := r.q(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.SavingsEntry{}
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}

	return out, rows.Err()
}

func (r *PostgresRepository) EntryByID(ctx context.Context, projectID, id uuid.UUID) (domain.SavingsEntry, error) {
	e, err := scanEntry(r.q(ctx).QueryRow(ctx, entrySelect+` WHERE e.project_id = $1 AND e.id = $2`, projectID, id))

	return e, notFound(err)
}

func (r *PostgresRepository) CountEntries(ctx context.Context, projectID uuid.UUID) (int, error) {
	var n int
	err := r.q(ctx).QueryRow(ctx, `SELECT count(*) FROM planner_savings_entries WHERE project_id=$1`, projectID).Scan(&n)

	return n, err
}

func (r *PostgresRepository) InsertEntry(ctx context.Context, projectID uuid.UUID, e *domain.SavingsEntry) error {
	_, err := r.q(ctx).Exec(ctx, `
		INSERT INTO planner_savings_entries (id, project_id, contributor_id, entry_date, amount, notes)
		VALUES ($1,$2,$3,$4::date,$5,$6)`, e.ID, projectID, e.ContributorID, e.EntryDate, e.Amount, e.Notes)

	return mapErr(err)
}

func (r *PostgresRepository) UpdateEntry(ctx context.Context, projectID uuid.UUID, e *domain.SavingsEntry) error {
	tag, err := r.q(ctx).Exec(ctx, `
		UPDATE planner_savings_entries SET contributor_id=$3, entry_date=$4::date, amount=$5, notes=$6, updated_at=now()
		WHERE id=$1 AND project_id=$2`, e.ID, projectID, e.ContributorID, e.EntryDate, e.Amount, e.Notes)

	return affected(tag, err)
}

func (r *PostgresRepository) DeleteEntry(ctx context.Context, projectID, id uuid.UUID) error {
	tag, err := r.q(ctx).Exec(ctx, `DELETE FROM planner_savings_entries WHERE id=$1 AND project_id=$2`, id, projectID)

	return affected(tag, err)
}

// Milestones is the FIFO allocation of 03 BE 3.6: the savings fill the dated,
// non-system sessions in date order, each up to the estimate of its lines.
func (r *PostgresRepository) Milestones(ctx context.Context, projectID uuid.UUID) ([]domain.Milestone, error) {
	rows, err := r.q(ctx).Query(ctx, `
		WITH col AS (SELECT COALESCE(sum(amount), 0)::bigint AS c FROM planner_savings_entries WHERE project_id = $1),
		ms AS (
		  SELECT s.scope_key, s.name, s.session_date, s.sort_order,
		         COALESCE((SELECT sum(estimate) FROM planner_budget_lines l WHERE l.project_id = s.project_id AND l.sub_event_scope = s.scope_key), 0)::bigint AS target
		  FROM planner_event_sessions s WHERE s.project_id = $1 AND NOT s.is_system AND s.session_date IS NOT NULL),
		cum AS (SELECT ms.*, COALESCE(sum(target) OVER (ORDER BY session_date, sort_order ROWS BETWEEN UNBOUNDED PRECEDING AND 1 PRECEDING), 0)::bigint AS before FROM ms)
		SELECT scope_key, name, session_date, target,
		       LEAST(GREATEST(col.c - before, 0), target),
		       LEAST(GREATEST(col.c - before, 0), target) = target
		FROM cum, col ORDER BY session_date, sort_order`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.Milestone{}
	for rows.Next() {
		var m domain.Milestone
		var date time.Time
		if err := rows.Scan(&m.SubEventScope, &m.Name, &date, &m.TargetAmount, &m.SecuredAmount, &m.IsSecured); err != nil {
			return nil, err
		}
		m.SessionDate = dateStr(date)
		out = append(out, m)
	}

	return out, rows.Err()
}

func (r *PostgresRepository) UnscheduledEstimate(ctx context.Context, projectID uuid.UUID) (int64, error) {
	var n int64
	err := r.q(ctx).QueryRow(ctx, `
		SELECT COALESCE(sum(l.estimate), 0)::bigint FROM planner_budget_lines l
		LEFT JOIN planner_event_sessions s ON s.project_id = l.project_id AND s.scope_key = l.sub_event_scope
		WHERE l.project_id = $1 AND (s.session_date IS NULL OR s.is_system)`, projectID).Scan(&n)

	return n, err
}

// MonthCells lists every month from the first to the last deposit, with no gap,
// for every contributor of the project (0 where there is no deposit).
func (r *PostgresRepository) MonthCells(ctx context.Context, projectID uuid.UUID) ([]domain.MonthCell, error) {
	rows, err := r.q(ctx).Query(ctx, `
		WITH r AS (SELECT date_trunc('month', min(entry_date))::date AS a, date_trunc('month', max(entry_date))::date AS z
		           FROM planner_savings_entries WHERE project_id = $1),
		m AS (SELECT generate_series(r.a, r.z, interval '1 month')::date AS month FROM r)
		SELECT to_char(m.month, 'YYYY-MM'), c.id, COALESCE(sum(e.amount), 0)::bigint
		FROM m CROSS JOIN planner_contributors c
		LEFT JOIN planner_savings_entries e ON e.contributor_id = c.id AND date_trunc('month', e.entry_date) = m.month
		WHERE c.project_id = $1
		GROUP BY m.month, c.id, c.sort_order, c.created_at
		ORDER BY m.month, c.sort_order, c.created_at, c.id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.MonthCell{}
	for rows.Next() {
		var c domain.MonthCell
		if err := rows.Scan(&c.Month, &c.ContributorID, &c.Amount); err != nil {
			return nil, err
		}
		out = append(out, c)
	}

	return out, rows.Err()
}

// ---------- sample dataset ----------

func (r *PostgresRepository) ApplySamplePayment(ctx context.Context, projectID uuid.UUID, p domain.SamplePayment) error {
	tag, err := r.q(ctx).Exec(ctx, `
		UPDATE planner_budget_items
		SET initial_estimate = $4, dp_amount = $5, term_1_amount = $6, term_2_amount = $7, updated_at = now()
		WHERE project_id = $1 AND sub_event_scope = $2 AND item_key = $3 AND source = 'template'`,
		projectID, p.Scope, p.Key, p.Estimate, p.DP, p.Term1, p.Term2)

	return affected(tag, err)
}
