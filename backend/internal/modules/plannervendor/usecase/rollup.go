package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannervendor/domain"
)

// LinkToBudget is POST /shopping-items/link-to-budget (04 BE 2.4): a budget line computes its estimate and its paid
// total from the shopping list instead of from the figures typed into it. The total is never copied; the view
// planner_budget_lines sums the items that point at the line. Everything runs under the project lock.
//
//   - The line: budget_item_id when it is sent, else the default line of the event type (looked up by key in any
//     session, made when it does not exist).
//   - The items: item_ids when sent, else every item that is not linked to a line yet. Items linked to another line stay.
//   - A line that already holds stored payments (deposit/terms) ignores them while it rolls up; that needs confirm.
func (s *Service) LinkToBudget(ctx context.Context, p pd.Project, req domain.LinkToBudgetRequest) (domain.LinkToBudgetResult, error) {
	itemIDs := uniqueIDs(req.ItemIDs)

	var res domain.LinkToBudgetResult
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		res = domain.LinkToBudgetResult{}
		if err := s.repo.LockProject(ctx, p.ID); err != nil {
			return err
		}
		line, created, err := s.rollupLine(ctx, p, req.BudgetItemID)
		if err != nil {
			return err
		}
		if !line.Rollup && line.HasPayments && !req.Confirm {
			return apierr.Conflict("rollup_conflict", "the line has stored payments that are ignored while it computes from the shopping list; send confirm: true to continue")
		}
		if len(itemIDs) > 0 {
			n, err := s.repo.CountShoppingIDs(ctx, p.ID, itemIDs)
			if err != nil {
				return err
			}
			if n != len(itemIDs) {
				return invalidInput("item_ids", "must be items of this project")
			}
		}

		if !line.Rollup {
			if err := s.repo.SetRollup(ctx, p.ID, line.ID, true); err != nil {
				return err
			}
		}
		var only []uuid.UUID
		if len(itemIDs) > 0 {
			only = itemIDs
		}
		linked, err := s.repo.LinkShoppingItems(ctx, p.ID, line.ID, only)
		if err != nil {
			return err
		}

		rolled, err := s.rollupLineOf(ctx, p.ID, line.ID)
		if err != nil {
			return err
		}
		res = domain.LinkToBudgetResult{Line: rolled, Created: created, Linked: linked}

		return nil
	})
	if err != nil {
		return domain.LinkToBudgetResult{}, mapErr(err, "budget line")
	}

	return res, nil
}

// fallbackRollup is the line the list rolls up into when the default line of the event type exists but cannot compute from
// a list (a vendor, a document or a unit price holds it). Three types have such a collision: the default line of peresmian
// (prosesi_vip), wisuda (toga_atribut) and seminar (kit_sertifikat) is also the dedicated line of a vendor category (mc,
// toga, percetakan), which lock-deal links to the vendor. Without a fallback "compute the line from this list" would end
// in an error for exactly the users who locked that vendor (04 BE chapter 9).
var fallbackRollup = pd.ShoppingRollup{Key: "perlengkapan", Name: "Perlengkapan", Category: "Perlengkapan"}

func rollable(ref domain.BudgetItemRef) bool { return !ref.Unit && !ref.VendorLinked && !ref.DocLinked }

// defaultRollup is the line the button "compute the line from this list" acts on: the default line of the event type
// (in any session, lowest sort_order) or, when that line cannot roll up, the fallback line. found is false when the line
// does not exist yet; spec then says which line would be made.
func (s *Service) defaultRollup(ctx context.Context, p pd.Project) (spec pd.ShoppingRollup, ref domain.BudgetItemRef, found bool, err error) {
	_, et := s.specs(p)
	spec = et.ShoppingRollup
	if ref, found, err = s.repo.BudgetItemByKey(ctx, p.ID, spec.Key); err != nil {
		return spec, ref, false, err
	}
	if found && !rollable(ref) && spec.Key != fallbackRollup.Key {
		spec = fallbackRollup
		ref, found, err = s.repo.BudgetItemByKey(ctx, p.ID, spec.Key)
	}

	return spec, ref, found, err
}

// rollupLine finds the line to roll up into (or makes the default one) and checks it can: not priced per unit, not
// linked to a vendor or a document.
func (s *Service) rollupLine(ctx context.Context, p pd.Project, requested *uuid.UUID) (ref domain.BudgetItemRef, created bool, err error) {
	switch {
	case requested != nil:
		ref, err = s.repo.BudgetItem(ctx, p.ID, *requested)
		if isNotFound(err) {
			return ref, false, invalidInput("budget_item_id", reasonBudgetLineOfProject)
		}
	default:
		var spec pd.ShoppingRollup
		var found bool
		if spec, ref, found, err = s.defaultRollup(ctx, p); err == nil && !found {
			ref, err = s.makeRollupLine(ctx, p, spec)
			created = true
		}
	}
	if err != nil {
		return ref, false, err
	}

	switch {
	case ref.Unit:
		return ref, false, invalidInput("budget_item_id", "must not be a unit-priced line")
	case ref.VendorLinked || ref.DocLinked:
		return ref, false, invalidInput("budget_item_id", "must not be linked to a vendor or document")
	}

	return ref, created, nil
}

// makeRollupLine creates the default line of the event type (a plain manual line in the general session, estimate 0).
func (s *Service) makeRollupLine(ctx context.Context, p pd.Project, spec pd.ShoppingRollup) (domain.BudgetItemRef, error) {
	n, err := s.repo.CountBudgetLines(ctx, p.ID)
	if err != nil {
		return domain.BudgetItemRef{}, err
	}
	if n >= maxBudgetLines {
		return domain.BudgetItemRef{}, budgetLimitReached()
	}
	key := spec.Key
	id, err := s.repo.InsertBudgetLine(ctx, p.ID, domain.NewBudgetLine{
		SubEventScope: pd.AllScope, Category: spec.Category, ItemName: spec.Name, ItemKey: &key, Source: "manual",
	})
	if err != nil {
		return domain.BudgetItemRef{}, err
	}

	return s.repo.BudgetItem(ctx, p.ID, id)
}

func (s *Service) rollupLineOf(ctx context.Context, projectID, lineID uuid.UUID) (domain.RollupLine, error) {
	lines, err := s.repo.RollupLines(ctx, projectID)
	if err != nil {
		return domain.RollupLine{}, err
	}
	for _, l := range lines {
		if l.BudgetItemID == lineID {
			return l, nil
		}
	}

	return domain.RollupLine{}, domain.ErrNotFound
}

// UnlinkFromBudget is POST /shopping-items/unlink-from-budget: it releases every item from the line and then switches the
// rollup off, so the line goes back to the figures stored in it. (Releasing only the switch would leave the items pointing
// at the line, which makes "every item that is not linked yet" ambiguous the next time.)
func (s *Service) UnlinkFromBudget(ctx context.Context, p pd.Project, req domain.UnlinkFromBudgetRequest) (domain.UnlinkFromBudgetResult, error) {
	var res domain.UnlinkFromBudgetResult
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		if err := s.repo.LockProject(ctx, p.ID); err != nil {
			return err
		}
		if _, err := s.repo.BudgetItem(ctx, p.ID, req.BudgetItemID); isNotFound(err) {
			return invalidInput("budget_item_id", reasonBudgetLineOfProject)
		} else if err != nil {
			return err
		}
		n, err := s.repo.UnlinkShoppingItems(ctx, p.ID, req.BudgetItemID)
		if err != nil {
			return err
		}
		res.Unlinked = n

		return s.repo.SetRollup(ctx, p.ID, req.BudgetItemID, false)
	})
	if err != nil {
		return domain.UnlinkFromBudgetResult{}, mapErr(err, "budget line")
	}

	return res, nil
}

func uniqueIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]bool, len(ids))
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}

	return out
}
