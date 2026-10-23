package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannervendor/domain"
)

func shoppingLimitReached() *apierr.Error {
	return apierr.Conflict("shopping_item_limit_reached", fmt.Sprintf("at most %d shopping items per project", maxShoppingItems))
}

// ShoppingFilter is the query of GET /shopping-items. Scope: empty = every session, "all" = the general items only;
// Category is the category text exactly as stored.
type ShoppingFilter struct{ Scope, Category, Status string }

func validShoppingStatus(s string) bool {
	return s == domain.ShoppingNotStarted || s == domain.ShoppingInProgress || s == domain.ShoppingDone
}

func (f ShoppingFilter) validate() error {
	if f.Scope != "" && !keyPattern.MatchString(f.Scope) {
		return invalidInput("scope", "must be a session scope_key")
	}
	if f.Status != "" && !validShoppingStatus(f.Status) {
		return invalidInput("status", "must be one of belum, proses, selesai")
	}

	return nil
}

func (f ShoppingFilter) keep(it domain.ShoppingItem) bool {
	return (f.Scope == "" || it.SubEventScope == f.Scope) && (f.Category == "" || it.Category == f.Category) && (f.Status == "" || it.Status == f.Status)
}

// ListShopping is GET /shopping-items: the KPI, the rollup lines and the default rollup target describe the whole
// project; only the item list is filtered.
func (s *Service) ListShopping(ctx context.Context, p pd.Project, f ShoppingFilter) (domain.ShoppingOverview, error) {
	if err := f.validate(); err != nil {
		return domain.ShoppingOverview{}, err
	}
	all, err := s.repo.ListShopping(ctx, p.ID)
	if err != nil {
		return domain.ShoppingOverview{}, err
	}
	rollups, err := s.repo.RollupLines(ctx, p.ID)
	if err != nil {
		return domain.ShoppingOverview{}, err
	}
	target, err := s.defaultTarget(ctx, p)
	if err != nil {
		return domain.ShoppingOverview{}, err
	}

	items := make([]domain.ShoppingItem, 0, len(all))
	for _, it := range all {
		if f.keep(it) {
			items = append(items, it)
		}
	}

	return domain.ShoppingOverview{KPI: domain.ShoppingKPIOf(all), Rollups: rollups, DefaultTarget: target, Items: items}, nil
}

// defaultTarget is the line behind the button "compute the line from this list": the line with the key of the registry
// in any session (or the fallback line when that one cannot roll up, see fallbackRollup), or the line that would be made.
func (s *Service) defaultTarget(ctx context.Context, p pd.Project) (domain.RollupTarget, error) {
	spec, ref, found, err := s.defaultRollup(ctx, p)
	if err != nil {
		return domain.RollupTarget{}, err
	}
	if !found {
		return domain.RollupTarget{ItemKey: spec.Key, ItemName: spec.Name, SubEventScope: pd.AllScope}, nil
	}

	return domain.RollupTarget{
		BudgetItemID: &ref.ID, ItemKey: spec.Key, ItemName: ref.ItemName, SubEventScope: ref.SubEventScope,
		Rollup: ref.Rollup, HasPayments: ref.HasPayments, Estimate: ref.Estimate, Paid: ref.Paid,
	}, nil
}

// shoppingInput is what POST and PUT share.
type shoppingInput struct {
	scope, category, name string
	quantity              *float64
	unit, location, notes *string
	estimated, actual     int64
	vendorID              *uuid.UUID
	purpose               *string
	status                string
}

func inputOfCreate(r domain.ShoppingItemRequest) shoppingInput {
	return shoppingInput{
		scope: r.SubEventScope, category: r.Category, name: r.ItemName, quantity: r.Quantity, unit: r.Unit, location: r.PurchaseLocation,
		notes: r.Notes, estimated: r.EstimatedPrice, actual: r.ActualPrice, vendorID: r.VendorID, purpose: r.Purpose, status: r.Status,
	}
}

func inputOfUpdate(r domain.UpdateShoppingItemRequest) shoppingInput {
	return shoppingInput{
		scope: r.SubEventScope, category: r.Category, name: r.ItemName, quantity: r.Quantity, unit: r.Unit, location: r.PurchaseLocation,
		notes: r.Notes, estimated: r.EstimatedPrice, actual: r.ActualPrice, vendorID: r.VendorID, purpose: r.Purpose, status: r.Status,
	}
}

// shoppingWrite validates the body in the order of the fields (04 BE 3.4) and returns the row to store. The checks that
// need the database (the session, the vendor, the rollup line, the limit of items) run in the transaction.
func shoppingWrite(in shoppingInput) (domain.ShoppingWrite, error) {
	var w domain.ShoppingWrite
	var err error

	if w.SubEventScope, err = normalizeScope(in.scope); err != nil {
		return w, err
	}
	if w.Category, err = cleanText("category", in.category, maxItemCategoryRunes, true); err != nil {
		return w, err
	}
	if w.ItemName, err = cleanText("item_name", in.name, maxItemNameRunes, true); err != nil {
		return w, err
	}
	if w.QuantityX100, err = quantityX100(in.quantity); err != nil {
		return w, err
	}
	if w.Unit, err = optionalText("unit", in.unit, maxUnitRunes); err != nil {
		return w, err
	}
	for _, price := range []struct {
		field string
		value int64
	}{{"estimated_price", in.estimated}, {"actual_price", in.actual}} {
		if err = validateMoney(price.field, price.value); err != nil {
			return w, err
		}
		// round(quantity × price) must fit: the generated column answers "bigint out of range" (a 500) beyond 10^13.
		if _, ok := domain.LineTotal(w.QuantityX100, price.value); !ok {
			return w, invalidInput(price.field, fmt.Sprintf("the total (quantity × price) must be at most %d", domain.MaxMoney))
		}
	}
	w.EstimatedPrice, w.ActualPrice = in.estimated, in.actual
	if w.PurchaseLocation, err = optionalNote("purchase_location", in.location, maxLocationRunes); err != nil {
		return w, err
	}
	w.VendorID = in.vendorID
	if in.purpose != nil && *in.purpose != "" && *in.purpose != domain.PurposeSouvenir {
		return w, invalidInput("purpose", "must be one of souvenir")
	}
	if in.purpose != nil && *in.purpose != "" {
		purpose := *in.purpose
		w.Purpose = &purpose
	}
	if w.Status = in.status; w.Status == "" {
		w.Status = domain.ShoppingNotStarted
	}
	if !validShoppingStatus(w.Status) {
		return w, invalidInput("status", "must be one of belum, proses, selesai")
	}
	w.Notes, err = optionalNote("notes", in.notes, maxNoteRunes)

	return w, err
}

// quantityX100 reads the optional quantity: empty = 1; at most two decimals; 0,01 … 1.000.000.
func quantityX100(q *float64) (int64, error) {
	if q == nil {
		return 100, nil
	}
	x, ok := domain.QuantityX100(*q)
	if !ok || x < domain.MinQuantityX100 || x > domain.MaxQuantityX100 {
		return 0, invalidInput("quantity", "must be between 0.01 and 1000000")
	}

	return x, nil
}

// checkVendor refuses a vendor of another project before the foreign key does.
func (s *Service) checkVendor(ctx context.Context, projectID uuid.UUID, id *uuid.UUID) error {
	if id == nil {
		return nil
	}
	ok, err := s.repo.VendorInProject(ctx, projectID, *id)
	if err != nil {
		return err
	}
	if !ok {
		return invalidInput("vendor_id", reasonVendorOfProject)
	}

	return nil
}

// CreateShopping is POST /shopping-items. budget_item_id (optional) is the rollup line the new item joins: a line that
// already computes its figures from the shopping list; items made without it stay out of that line until linked.
func (s *Service) CreateShopping(ctx context.Context, p pd.Project, req domain.ShoppingItemRequest) (domain.ShoppingItem, error) {
	w, err := shoppingWrite(inputOfCreate(req))
	if err != nil {
		return domain.ShoppingItem{}, err
	}
	w.ID, w.BudgetItemID = uuid.New(), req.BudgetItemID

	var out domain.ShoppingItem
	err = s.tx.InTx(ctx, func(ctx context.Context) error {
		if err := s.repo.LockProject(ctx, p.ID); err != nil {
			return err
		}
		n, err := s.repo.CountShopping(ctx, p.ID)
		if err != nil {
			return err
		}
		if n >= maxShoppingItems {
			return shoppingLimitReached()
		}
		if err := s.checkScope(ctx, p.ID, w.SubEventScope); err != nil {
			return err
		}
		if err := s.checkVendor(ctx, p.ID, w.VendorID); err != nil {
			return err
		}
		if err := s.checkRollupTarget(ctx, p.ID, w.BudgetItemID); err != nil {
			return err
		}
		if err := s.repo.InsertShopping(ctx, p.ID, &w); err != nil {
			return err
		}
		out, err = s.repo.ShoppingByID(ctx, p.ID, w.ID)

		return err
	})
	if err != nil {
		return domain.ShoppingItem{}, mapErr(err, "shopping item")
	}

	return out, nil
}

// checkRollupTarget: the line an item is created into must already compute its figures from the shopping list.
func (s *Service) checkRollupTarget(ctx context.Context, projectID uuid.UUID, id *uuid.UUID) error {
	if id == nil {
		return nil
	}
	ref, err := s.repo.BudgetItem(ctx, projectID, *id)
	if errors.Is(err, domain.ErrNotFound) || (err == nil && !ref.Rollup) {
		return invalidInput("budget_item_id", "must be a shopping rollup line of this project")
	}

	return err
}

// UpdateShopping is PUT /shopping-items/:id. It replaces the item but not its rollup link, template key or order.
func (s *Service) UpdateShopping(ctx context.Context, p pd.Project, id uuid.UUID, req domain.UpdateShoppingItemRequest) (domain.ShoppingItem, error) {
	w, err := shoppingWrite(inputOfUpdate(req))
	if err != nil {
		return domain.ShoppingItem{}, err
	}
	w.ID = id

	var out domain.ShoppingItem
	err = s.tx.InTx(ctx, func(ctx context.Context) error {
		if _, err := s.repo.ShoppingByID(ctx, p.ID, id); err != nil {
			return err
		}
		if err := s.checkScope(ctx, p.ID, w.SubEventScope); err != nil {
			return err
		}
		if err := s.checkVendor(ctx, p.ID, w.VendorID); err != nil {
			return err
		}
		if err := s.repo.UpdateShopping(ctx, p.ID, &w); err != nil {
			return err
		}
		out, err = s.repo.ShoppingByID(ctx, p.ID, id)

		return err
	})
	if err != nil {
		return domain.ShoppingItem{}, mapErr(err, "shopping item")
	}

	return out, nil
}

// SetShoppingStatus is PATCH /shopping-items/:id/status: finishing an item without an actual price copies the estimate
// into it (editable afterwards); a price that is there is never overwritten, and moving back keeps it.
func (s *Service) SetShoppingStatus(ctx context.Context, p pd.Project, id uuid.UUID, req domain.ShoppingStatusRequest) (domain.ShoppingItem, error) {
	if !validShoppingStatus(req.Status) {
		return domain.ShoppingItem{}, invalidInput("status", "must be one of belum, proses, selesai")
	}

	var out domain.ShoppingItem
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetShoppingStatus(ctx, p.ID, id, req.Status); err != nil {
			return err
		}
		var err error
		out, err = s.repo.ShoppingByID(ctx, p.ID, id)

		return err
	})
	if err != nil {
		return domain.ShoppingItem{}, mapErr(err, "shopping item")
	}

	return out, nil
}

// DeleteShopping is DELETE /shopping-items/:id. A rollup line recomputes its figures without the item.
func (s *Service) DeleteShopping(ctx context.Context, p pd.Project, id uuid.UUID) error {
	return mapErr(s.repo.DeleteShopping(ctx, p.ID, id), "shopping item")
}
