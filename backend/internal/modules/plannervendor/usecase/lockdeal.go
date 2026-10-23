package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannervendor/domain"
)

func budgetLimitReached() *apierr.Error {
	return apierr.Conflict("budget_item_limit_reached", "at most 300 budget lines per project")
}

// LockDeal is POST /vendors/:id/lock-deal (04 BE 2.2): the vendor becomes fix with the agreed price and, with
// sync_to_budget_items, the deal is tied to the budget without ever making a second line for the same cost. The whole
// call runs under the project lock; without it two deals for vendors of one category both reported "linked" while the
// second overwrote the first link.
func (s *Service) LockDeal(ctx context.Context, p pd.Project, id uuid.UUID, req domain.LockDealRequest) (domain.LockDealResult, error) {
	if err := validateDeal("agreed_price", req.AgreedPrice); err != nil {
		return domain.LockDealResult{}, err
	}

	res := domain.LockDealResult{Warnings: []pd.Notice{}}
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		res = domain.LockDealResult{Warnings: []pd.Notice{}}
		if err := s.repo.LockProject(ctx, p.ID); err != nil {
			return err
		}
		v, err := s.repo.VendorByID(ctx, p.ID, id)
		if err != nil {
			return err
		}
		if err := s.repo.SetVendorDeal(ctx, p.ID, id, req.AgreedPrice); err != nil {
			return err
		}
		if req.SyncToBudgetItems {
			if err := s.syncDeal(ctx, p, v, req.AgreedPrice, &res); err != nil {
				return err
			}
		}
		res.Vendor, err = s.vendorDTO(ctx, p, id)

		return err
	})
	if err != nil {
		return domain.LockDealResult{}, mapErr(err, "vendor")
	}

	return res, nil
}

// syncDeal ties the deal to a budget line, in this order: the line that already points at the vendor (2a), the line the
// category shares with others (2b, reported but left alone), a dedicated line that nothing holds yet (2c), a new line (2d).
func (s *Service) syncDeal(ctx context.Context, p pd.Project, v domain.VendorRow, price int64, res *domain.LockDealResult) error {
	cats, _ := s.specs(p)

	// 2a. The vendor already has a line: bring its estimate to the deal.
	if line, found, err := s.repo.VendorLine(ctx, p.ID, v.ID); err != nil {
		return err
	} else if found {
		if err := s.repo.RepriceVendorLines(ctx, p.ID, v.ID, price); err != nil {
			return err
		}
		res.BudgetItemID = &line.ID
		unitPricing(res, line)

		return nil
	}

	// 2b. A line that several categories share stands for all of them: replacing its estimate with one vendor's price
	// would lower the plan before the others are locked.
	d, err := s.readVendorData(ctx, p)
	if err != nil {
		return err
	}
	if item, shared := domain.SharedItemOf(cats, v.CategoryKey, v.SubEventScope, d.scenario, d.lines, d.vendors); shared {
		res.SharedItem = &item
		res.Warnings = append(res.Warnings, notice("shared_budget_item", "the budget line is shared by several vendor categories and was not changed", 1))

		return nil
	}

	// 2c. The dedicated line of the category, when nothing holds it yet (the conditional update closes the race).
	if keys := categoryKeys(cats, v.CategoryKey); len(keys) > 0 {
		line, found, err := s.repo.FreeLine(ctx, p.ID, keys, v.SubEventScope)
		if err != nil {
			return err
		}
		if found {
			linked, err := s.repo.LinkVendorToLine(ctx, p.ID, line.ID, v.ID, price)
			if err != nil {
				return err
			}
			if linked {
				res.BudgetItemID = &line.ID
				unitPricing(res, line)

				return nil
			}
		}
	}

	// 2d. Nothing fits: a new line. The vendor name is cut to the 150 characters of a line name.
	n, err := s.repo.CountBudgetLines(ctx, p.ID)
	if err != nil {
		return err
	}
	if n >= maxBudgetLines {
		return budgetLimitReached()
	}
	key := v.CategoryKey
	lineID, err := s.repo.InsertBudgetLine(ctx, p.ID, domain.NewBudgetLine{
		SubEventScope: v.SubEventScope, Category: truncRunes(v.CategoryLabel, maxLineCategoryRunes),
		ItemName: truncRunes(v.VendorName, maxLineNameRunes), ItemKey: &key, Source: "vendor", VendorID: &v.ID, Estimate: price,
	})
	if err != nil {
		return err
	}
	res.BudgetItemID, res.Created = &lineID, true

	return nil
}

func unitPricing(res *domain.LockDealResult, line domain.BudgetLineRef) {
	if line.Unit {
		res.Warnings = append(res.Warnings, notice("unit_pricing", "the budget line is priced per unit, so its estimate was not replaced by the deal", 1))
	}
}

func categoryKeys(cats []domain.CategorySpec, key string) []string {
	for _, c := range cats {
		if c.Key == key {
			return c.BudgetItemKeys
		}
	}

	return nil
}
