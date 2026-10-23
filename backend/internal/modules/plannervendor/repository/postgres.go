// Package repository is the PostgreSQL side of planner menu 04. The pool runs pgx in QueryExecModeSimpleProtocol, so
// results arrive as text: NULL columns scan into pointers, NUMERIC quantities are read as hundredths
// ((quantity * 100)::bigint) and written back with $n::bigint::numeric / 100, and arrays go in as []string with an explicit
// cast. pgx also rejects an argument a statement does not use, so every query is given exactly the arguments it mentions.
//
// Figures of budget lines are read from the view planner_budget_lines, never from the table: a line that rolls up
// the shopping list has its estimate and paid there only. Writers that touch lines or the item counts hold the project
// lock (SELECT … FOR UPDATE on planner_projects, LockProject).
package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/db"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannervendor/domain"
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
// one); this covers the statement-time cases: the composite foreign keys to vendors and budget lines are immediate.
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
		case strings.HasSuffix(pg.ConstraintName, "_budget_item_id_project_id_fkey"):
			return domain.ErrInvalidBudgetItem
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

// ---------- project, sessions ----------

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

// ---------- vendors ----------

const vendorCols = `id, sub_event_scope, category_key, category_label, vendor_name, pic_name, phone_number, formatted_wa_number,
	portfolio_url, budget_limit, agreed_price, status, notes, sort_order`

func scanVendor(row pgx.Row) (domain.VendorRow, error) {
	var v domain.VendorRow
	var status string
	err := row.Scan(&v.ID, &v.SubEventScope, &v.CategoryKey, &v.CategoryLabel, &v.VendorName, &v.PICName, &v.PhoneNumber,
		&v.FormattedWANumber, &v.PortfolioURL, &v.BudgetLimit, &v.AgreedPrice, &status, &v.Notes, &v.SortOrder)
	v.Status = domain.VendorStatus(status)

	return v, err
}

func (r *PostgresRepository) ListVendors(ctx context.Context, projectID uuid.UUID) ([]domain.VendorRow, error) {
	rows, err := r.q(ctx).Query(ctx,
		`SELECT `+vendorCols+` FROM planner_vendors WHERE project_id=$1 ORDER BY sort_order, created_at, id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.VendorRow{}
	for rows.Next() {
		v, err := scanVendor(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}

	return out, rows.Err()
}

func (r *PostgresRepository) VendorByID(ctx context.Context, projectID, id uuid.UUID) (domain.VendorRow, error) {
	v, err := scanVendor(r.q(ctx).QueryRow(ctx,
		`SELECT `+vendorCols+` FROM planner_vendors WHERE id=$1 AND project_id=$2`, id, projectID))

	return v, notFound(err)
}

func (r *PostgresRepository) CountVendors(ctx context.Context, projectID uuid.UUID) (int, error) {
	var n int
	err := r.q(ctx).QueryRow(ctx, `SELECT count(*) FROM planner_vendors WHERE project_id=$1`, projectID).Scan(&n)

	return n, err
}

func (r *PostgresRepository) InsertVendor(ctx context.Context, projectID uuid.UUID, v *domain.VendorRow) error {
	err := r.q(ctx).QueryRow(ctx, `
		INSERT INTO planner_vendors (id, project_id, sub_event_scope, category_key, category_label, vendor_name, pic_name,
		       phone_number, formatted_wa_number, portfolio_url, budget_limit, agreed_price, status, notes, sort_order)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,
		        (SELECT COALESCE(max(sort_order), 0) + 1 FROM planner_vendors WHERE project_id = $2))
		RETURNING sort_order`,
		v.ID, projectID, v.SubEventScope, v.CategoryKey, v.CategoryLabel, v.VendorName, v.PICName, v.PhoneNumber,
		v.FormattedWANumber, v.PortfolioURL, v.BudgetLimit, v.AgreedPrice, string(v.Status), v.Notes).Scan(&v.SortOrder)

	return mapErr(err)
}

func (r *PostgresRepository) UpdateVendor(ctx context.Context, projectID uuid.UUID, v *domain.VendorRow) error {
	tag, err := r.q(ctx).Exec(ctx, `
		UPDATE planner_vendors
		SET sub_event_scope=$3, category_key=$4, category_label=$5, vendor_name=$6, pic_name=$7, phone_number=$8,
		    formatted_wa_number=$9, portfolio_url=$10, budget_limit=$11, agreed_price=$12, status=$13, notes=$14, updated_at=now()
		WHERE id=$1 AND project_id=$2`,
		v.ID, projectID, v.SubEventScope, v.CategoryKey, v.CategoryLabel, v.VendorName, v.PICName, v.PhoneNumber,
		v.FormattedWANumber, v.PortfolioURL, v.BudgetLimit, v.AgreedPrice, string(v.Status), v.Notes)

	return affected(tag, err)
}

func (r *PostgresRepository) SetVendorStatus(ctx context.Context, projectID, id uuid.UUID, status domain.VendorStatus) error {
	tag, err := r.q(ctx).Exec(ctx,
		`UPDATE planner_vendors SET status=$3, updated_at=now() WHERE id=$1 AND project_id=$2`, id, projectID, string(status))

	return affected(tag, err)
}

func (r *PostgresRepository) SetVendorDeal(ctx context.Context, projectID, id uuid.UUID, price int64) error {
	tag, err := r.q(ctx).Exec(ctx,
		`UPDATE planner_vendors SET status='fix', agreed_price=$3, updated_at=now() WHERE id=$1 AND project_id=$2`, id, projectID, price)

	return affected(tag, err)
}

func (r *PostgresRepository) DeleteVendor(ctx context.Context, projectID, id uuid.UUID) error {
	tag, err := r.q(ctx).Exec(ctx, `DELETE FROM planner_vendors WHERE id=$1 AND project_id=$2`, id, projectID)

	return affected(tag, err)
}

// ActiveScenarioRows: the lines of the active scenario that take part in the simulation, with the session rule of
// Smart Adjust (an empty active_sub_events means every session).
func (r *PostgresRepository) ActiveScenarioRows(ctx context.Context, projectID uuid.UUID) ([]domain.ScenarioRow, error) {
	rows, err := r.q(ctx).Query(ctx, `
		SELECT si.sub_event_scope, si.item_key, si.item_name, si.cost_amount
		FROM planner_budget_scenario_items si
		JOIN planner_budget_scenarios s ON s.id = si.scenario_id AND s.is_active
		WHERE s.project_id = $1 AND si.item_key IS NOT NULL
		  AND (cardinality(s.active_sub_events) = 0 OR si.sub_event_scope = 'all' OR si.sub_event_scope = ANY (s.active_sub_events))
		ORDER BY si.sort_order, si.id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.ScenarioRow{}
	for rows.Next() {
		var s domain.ScenarioRow
		if err := rows.Scan(&s.SubEventScope, &s.ItemKey, &s.ItemName, &s.Cost); err != nil {
			return nil, err
		}
		out = append(out, s)
	}

	return out, rows.Err()
}

func (r *PostgresRepository) Lines(ctx context.Context, projectID uuid.UUID) ([]domain.LineRow, error) {
	rows, err := r.q(ctx).Query(ctx, `
		SELECT id, sub_event_scope, COALESCE(item_key, ''), item_name, vendor_id, estimate, paid, remaining, payment_status
		FROM planner_budget_lines
		WHERE project_id = $1 AND (item_key IS NOT NULL OR vendor_id IS NOT NULL)
		ORDER BY sort_order, id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.LineRow{}
	for rows.Next() {
		var l domain.LineRow
		if err := rows.Scan(&l.ID, &l.SubEventScope, &l.ItemKey, &l.ItemName, &l.VendorID, &l.Estimate, &l.Paid, &l.Remaining, &l.PaymentStatus); err != nil {
			return nil, err
		}
		out = append(out, l)
	}

	return out, rows.Err()
}

// ---------- lock-deal ----------

func (r *PostgresRepository) lineRef(row pgx.Row) (domain.BudgetLineRef, bool, error) {
	var l domain.BudgetLineRef
	err := row.Scan(&l.ID, &l.Unit)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.BudgetLineRef{}, false, nil
	}

	return l, err == nil, err
}

func (r *PostgresRepository) VendorLine(ctx context.Context, projectID, vendorID uuid.UUID) (domain.BudgetLineRef, bool, error) {
	return r.lineRef(r.q(ctx).QueryRow(ctx,
		`SELECT id, pricing_mode = 'unit' FROM planner_budget_items WHERE project_id=$1 AND vendor_id=$2`, projectID, vendorID))
}

func (r *PostgresRepository) FreeLine(ctx context.Context, projectID uuid.UUID, keys []string, scope string) (domain.BudgetLineRef, bool, error) {
	return r.lineRef(r.q(ctx).QueryRow(ctx, `
		SELECT id, pricing_mode = 'unit' FROM planner_budget_items
		WHERE project_id = $1 AND item_key = ANY ($2::text[]) AND sub_event_scope IN ($3, 'all')
		  AND vendor_id IS NULL AND rollup_source IS NULL
		ORDER BY (sub_event_scope = $3) DESC, sort_order, id
		LIMIT 1`, projectID, keys, scope))
}

func (r *PostgresRepository) LinkVendorToLine(ctx context.Context, projectID, lineID, vendorID uuid.UUID, price int64) (bool, error) {
	tag, err := r.q(ctx).Exec(ctx, `
		UPDATE planner_budget_items
		SET vendor_id = $3, initial_estimate = CASE WHEN pricing_mode = 'unit' THEN initial_estimate ELSE $4 END, updated_at = now()
		WHERE id = $2 AND project_id = $1 AND vendor_id IS NULL`, projectID, lineID, vendorID, price)
	if err != nil {
		return false, mapErr(err)
	}

	return tag.RowsAffected() == 1, nil
}

func (r *PostgresRepository) RepriceVendorLines(ctx context.Context, projectID, vendorID uuid.UUID, price int64) error {
	_, err := r.q(ctx).Exec(ctx, `
		UPDATE planner_budget_items
		SET initial_estimate = CASE WHEN pricing_mode = 'unit' THEN initial_estimate ELSE $3 END, updated_at = now()
		WHERE project_id = $1 AND vendor_id = $2`, projectID, vendorID, price)

	return mapErr(err)
}

func (r *PostgresRepository) CountBudgetLines(ctx context.Context, projectID uuid.UUID) (int, error) {
	var n int
	err := r.q(ctx).QueryRow(ctx, `SELECT count(*) FROM planner_budget_items WHERE project_id=$1`, projectID).Scan(&n)

	return n, err
}

// InsertBudgetLine appends a line (sort_order = max + 1). ItemKey is a wish: it is left NULL when the session already has
// a line with that key, so the unique index (project, session, item_key) cannot make the insert fail.
func (r *PostgresRepository) InsertBudgetLine(ctx context.Context, projectID uuid.UUID, l domain.NewBudgetLine) (uuid.UUID, error) {
	id := uuid.New()
	_, err := r.q(ctx).Exec(ctx, `
		INSERT INTO planner_budget_items (id, project_id, sub_event_scope, category, item_name, item_key, source, vendor_id, initial_estimate, sort_order)
		VALUES ($1, $2, $3, $4, $5,
		        CASE WHEN $6::text IS NOT NULL AND NOT EXISTS (
		            SELECT 1 FROM planner_budget_items WHERE project_id = $2 AND sub_event_scope = $3 AND item_key = $6::text)
		        THEN $6::text END,
		        $7, $8, $9, (SELECT COALESCE(max(sort_order), 0) + 1 FROM planner_budget_items WHERE project_id = $2))`,
		id, projectID, l.SubEventScope, l.Category, l.ItemName, l.ItemKey, l.Source, l.VendorID, l.Estimate)
	if err != nil {
		return uuid.Nil, mapErr(err)
	}

	return id, nil
}

func (r *PostgresRepository) ReleaseVendorLines(ctx context.Context, projectID, vendorID uuid.UUID) (int, error) {
	tag, err := r.q(ctx).Exec(ctx,
		`UPDATE planner_budget_items SET vendor_id = NULL, updated_at = now() WHERE project_id = $1 AND vendor_id = $2`, projectID, vendorID)
	if err != nil {
		return 0, mapErr(err)
	}

	return int(tag.RowsAffected()), nil
}

// ---------- shopping ----------

func (r *PostgresRepository) VendorInProject(ctx context.Context, projectID, vendorID uuid.UUID) (bool, error) {
	var one int
	err := r.q(ctx).QueryRow(ctx, `SELECT 1 FROM planner_vendors WHERE id=$1 AND project_id=$2`, vendorID, projectID).Scan(&one)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}

	return err == nil, err
}

const shoppingSelect = `
	SELECT s.id, s.sub_event_scope, s.category, s.item_name, (s.quantity * 100)::bigint, s.unit, s.estimated_price, s.actual_price,
	       s.estimated_total, s.actual_total, s.price_delta, s.purchase_location, s.vendor_id, v.vendor_name, s.budget_item_id,
	       s.purpose, s.status, s.sort_order, s.template_key, s.notes
	FROM planner_shopping_items s
	LEFT JOIN planner_vendors v ON v.id = s.vendor_id AND v.project_id = s.project_id`

func scanShopping(row pgx.Row) (domain.ShoppingItem, error) {
	var it domain.ShoppingItem
	var q100 int64
	err := row.Scan(&it.ID, &it.SubEventScope, &it.Category, &it.ItemName, &q100, &it.Unit, &it.EstimatedPrice, &it.ActualPrice,
		&it.EstimatedTotal, &it.ActualTotal, &it.PriceDelta, &it.PurchaseLocation, &it.VendorID, &it.VendorName, &it.BudgetItemID,
		&it.Purpose, &it.Status, &it.SortOrder, &it.TemplateKey, &it.Notes)
	it.Quantity = domain.QuantityFloat(q100)

	return it, err
}

func (r *PostgresRepository) ListShopping(ctx context.Context, projectID uuid.UUID) ([]domain.ShoppingItem, error) {
	rows, err := r.q(ctx).Query(ctx, shoppingSelect+` WHERE s.project_id=$1 ORDER BY s.sort_order, s.created_at, s.id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.ShoppingItem{}
	for rows.Next() {
		it, err := scanShopping(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, it)
	}

	return out, rows.Err()
}

func (r *PostgresRepository) ShoppingByID(ctx context.Context, projectID, id uuid.UUID) (domain.ShoppingItem, error) {
	it, err := scanShopping(r.q(ctx).QueryRow(ctx, shoppingSelect+` WHERE s.id=$1 AND s.project_id=$2`, id, projectID))

	return it, notFound(err)
}

func (r *PostgresRepository) CountShopping(ctx context.Context, projectID uuid.UUID) (int, error) {
	var n int
	err := r.q(ctx).QueryRow(ctx, `SELECT count(*) FROM planner_shopping_items WHERE project_id=$1`, projectID).Scan(&n)

	return n, err
}

func (r *PostgresRepository) InsertShopping(ctx context.Context, projectID uuid.UUID, it *domain.ShoppingWrite) error {
	_, err := r.q(ctx).Exec(ctx, `
		INSERT INTO planner_shopping_items (id, project_id, sub_event_scope, category, item_name, quantity, unit, estimated_price, actual_price,
		       purchase_location, vendor_id, budget_item_id, purpose, status, notes, template_key, sort_order)
		VALUES ($1,$2,$3,$4,$5,$6::bigint::numeric / 100,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,
		        (SELECT COALESCE(max(sort_order), 0) + 1 FROM planner_shopping_items WHERE project_id = $2))`,
		it.ID, projectID, it.SubEventScope, it.Category, it.ItemName, it.QuantityX100, it.Unit, it.EstimatedPrice, it.ActualPrice,
		it.PurchaseLocation, it.VendorID, it.BudgetItemID, it.Purpose, it.Status, it.Notes, it.TemplateKey)

	return mapErr(err)
}

func (r *PostgresRepository) UpdateShopping(ctx context.Context, projectID uuid.UUID, it *domain.ShoppingWrite) error {
	tag, err := r.q(ctx).Exec(ctx, `
		UPDATE planner_shopping_items
		SET sub_event_scope=$3, category=$4, item_name=$5, quantity=$6::bigint::numeric / 100, unit=$7, estimated_price=$8, actual_price=$9,
		    purchase_location=$10, vendor_id=$11, purpose=$12, status=$13, notes=$14, updated_at=now()
		WHERE id=$1 AND project_id=$2`,
		it.ID, projectID, it.SubEventScope, it.Category, it.ItemName, it.QuantityX100, it.Unit, it.EstimatedPrice, it.ActualPrice,
		it.PurchaseLocation, it.VendorID, it.Purpose, it.Status, it.Notes)

	return affected(tag, err)
}

// SetShoppingStatus: finishing an item whose actual price is 0 fills it with the estimate; an actual price that is there
// stays, and moving back to proses or belum does not clear it.
func (r *PostgresRepository) SetShoppingStatus(ctx context.Context, projectID, id uuid.UUID, status string) error {
	tag, err := r.q(ctx).Exec(ctx, `
		UPDATE planner_shopping_items
		SET status = $3,
		    actual_price = CASE WHEN $3 = 'selesai' AND actual_price = 0 THEN estimated_price ELSE actual_price END,
		    updated_at = now()
		WHERE id = $2 AND project_id = $1`, projectID, id, status)

	return affected(tag, err)
}

func (r *PostgresRepository) DeleteShopping(ctx context.Context, projectID, id uuid.UUID) error {
	tag, err := r.q(ctx).Exec(ctx, `DELETE FROM planner_shopping_items WHERE id=$1 AND project_id=$2`, id, projectID)

	return affected(tag, err)
}

func idStrings(ids []uuid.UUID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}

	return out
}

func (r *PostgresRepository) CountShoppingIDs(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) (int, error) {
	var n int
	err := r.q(ctx).QueryRow(ctx,
		`SELECT count(*) FROM planner_shopping_items WHERE project_id=$1 AND id = ANY ($2::text[]::uuid[])`, projectID, idStrings(ids)).Scan(&n)

	return n, err
}

// ---------- rollup ----------

func (r *PostgresRepository) RollupLines(ctx context.Context, projectID uuid.UUID) ([]domain.RollupLine, error) {
	rows, err := r.q(ctx).Query(ctx, `
		SELECT l.id, l.item_name, l.sub_event_scope, l.estimate, l.paid,
		       (SELECT count(*) FROM planner_shopping_items i WHERE i.budget_item_id = l.id)
		FROM planner_budget_lines l
		WHERE l.project_id = $1 AND l.rollup_source = 'shopping'
		ORDER BY l.sort_order, l.id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.RollupLine{}
	for rows.Next() {
		var l domain.RollupLine
		if err := rows.Scan(&l.BudgetItemID, &l.ItemName, &l.SubEventScope, &l.Estimate, &l.Paid, &l.LinkedItems); err != nil {
			return nil, err
		}
		out = append(out, l)
	}

	return out, rows.Err()
}

const itemRefSelect = `
	SELECT id, sub_event_scope, COALESCE(item_key, ''), item_name, pricing_mode = 'unit', rollup_source IS NOT NULL,
	       vendor_id IS NOT NULL, admin_doc_id IS NOT NULL, (dp_amount + term_1_amount + term_2_amount) > 0, estimate, paid
	FROM planner_budget_lines`

func scanItemRef(row pgx.Row) (domain.BudgetItemRef, error) {
	var b domain.BudgetItemRef
	err := row.Scan(&b.ID, &b.SubEventScope, &b.ItemKey, &b.ItemName, &b.Unit, &b.Rollup, &b.VendorLinked, &b.DocLinked, &b.HasPayments, &b.Estimate, &b.Paid)

	return b, err
}

func (r *PostgresRepository) BudgetItem(ctx context.Context, projectID, id uuid.UUID) (domain.BudgetItemRef, error) {
	b, err := scanItemRef(r.q(ctx).QueryRow(ctx, itemRefSelect+` WHERE id=$1 AND project_id=$2`, id, projectID))

	return b, notFound(err)
}

func (r *PostgresRepository) BudgetItemByKey(ctx context.Context, projectID uuid.UUID, key string) (domain.BudgetItemRef, bool, error) {
	b, err := scanItemRef(r.q(ctx).QueryRow(ctx, itemRefSelect+` WHERE project_id=$1 AND item_key=$2 ORDER BY sort_order, id LIMIT 1`, projectID, key))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.BudgetItemRef{}, false, nil
	}

	return b, err == nil, err
}

func (r *PostgresRepository) SetRollup(ctx context.Context, projectID, lineID uuid.UUID, on bool) error {
	if on {
		tag, err := r.q(ctx).Exec(ctx,
			`UPDATE planner_budget_items SET rollup_source='shopping', updated_at=now() WHERE id=$1 AND project_id=$2`, lineID, projectID)

		return affected(tag, err)
	}
	_, err := r.q(ctx).Exec(ctx,
		`UPDATE planner_budget_items SET rollup_source=NULL, updated_at=now() WHERE id=$1 AND project_id=$2 AND rollup_source IS NOT NULL`, lineID, projectID)

	return mapErr(err)
}

func (r *PostgresRepository) LinkShoppingItems(ctx context.Context, projectID, lineID uuid.UUID, ids []uuid.UUID) (int, error) {
	sql := `UPDATE planner_shopping_items SET budget_item_id=$2, updated_at=now() WHERE project_id=$1 AND budget_item_id IS NULL`
	args := []any{projectID, lineID}
	if ids != nil {
		sql += ` AND id = ANY ($3::text[]::uuid[])`
		args = append(args, idStrings(ids))
	}
	tag, err := r.q(ctx).Exec(ctx, sql, args...)
	if err != nil {
		return 0, mapErr(err)
	}

	return int(tag.RowsAffected()), nil
}

func (r *PostgresRepository) UnlinkShoppingItems(ctx context.Context, projectID, lineID uuid.UUID) (int, error) {
	tag, err := r.q(ctx).Exec(ctx,
		`UPDATE planner_shopping_items SET budget_item_id=NULL, updated_at=now() WHERE project_id=$1 AND budget_item_id=$2`, projectID, lineID)
	if err != nil {
		return 0, mapErr(err)
	}

	return int(tag.RowsAffected()), nil
}

// ---------- template ----------

func (r *PostgresRepository) TemplateImpact(ctx context.Context, projectID uuid.UUID) (domain.TemplateImpact, error) {
	var t domain.TemplateImpact
	err := r.q(ctx).QueryRow(ctx, `
		SELECT count(*),
		       count(*) FILTER (WHERE status <> 'belum' OR actual_price > 0 OR budget_item_id IS NOT NULL OR vendor_id IS NOT NULL OR purpose IS NOT NULL),
		       count(*) FILTER (WHERE budget_item_id IS NOT NULL)
		FROM planner_shopping_items WHERE project_id = $1 AND template_key IS NOT NULL`, projectID).Scan(&t.Removed, &t.CarryWork, &t.RollupLinks)

	return t, err
}

func (r *PostgresRepository) InsertTemplateItem(ctx context.Context, projectID uuid.UUID, it *domain.ShoppingWrite) (bool, error) {
	tag, err := r.q(ctx).Exec(ctx, `
		INSERT INTO planner_shopping_items (id, project_id, sub_event_scope, category, item_name, quantity, estimated_price, status, template_key, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6::bigint::numeric / 100, $7, $8, $9,
		        (SELECT COALESCE(max(sort_order), 0) + 1 FROM planner_shopping_items WHERE project_id = $2))
		ON CONFLICT (project_id, template_key) WHERE template_key IS NOT NULL DO NOTHING`,
		it.ID, projectID, it.SubEventScope, it.Category, it.ItemName, it.QuantityX100, it.EstimatedPrice, it.Status, it.TemplateKey)
	if err != nil {
		return false, mapErr(err)
	}

	return tag.RowsAffected() == 1, nil
}

func (r *PostgresRepository) DeleteTemplateItems(ctx context.Context, projectID uuid.UUID) (int, error) {
	tag, err := r.q(ctx).Exec(ctx, `DELETE FROM planner_shopping_items WHERE project_id=$1 AND template_key IS NOT NULL`, projectID)
	if err != nil {
		return 0, mapErr(err)
	}

	return int(tag.RowsAffected()), nil
}

// ---------- sample dataset ----------

func (r *PostgresRepository) LinkLineByKey(ctx context.Context, projectID uuid.UUID, scope, key string, vendorID uuid.UUID) error {
	tag, err := r.q(ctx).Exec(ctx, `
		UPDATE planner_budget_items SET vendor_id=$4, updated_at=now()
		WHERE project_id=$1 AND sub_event_scope=$2 AND item_key=$3`, projectID, scope, key, vendorID)

	return affected(tag, err)
}

func (r *PostgresRepository) LinkTaskByTemplateKey(ctx context.Context, projectID uuid.UUID, templateKey string, vendorID uuid.UUID) error {
	tag, err := r.q(ctx).Exec(ctx,
		`UPDATE planner_tasks SET vendor_id=$3, updated_at=now() WHERE project_id=$1 AND template_key=$2`, projectID, templateKey, vendorID)

	return affected(tag, err)
}

func (r *PostgresRepository) SetSampleShopping(ctx context.Context, projectID uuid.UUID, templateKey, status string, actualPrice int64) error {
	tag, err := r.q(ctx).Exec(ctx, `
		UPDATE planner_shopping_items SET status=$3, actual_price=$4, updated_at=now()
		WHERE project_id=$1 AND template_key=$2`, projectID, templateKey, status, actualPrice)

	return affected(tag, err)
}
