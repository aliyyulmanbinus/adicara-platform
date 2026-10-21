package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerfinance/domain"
)

func lineLimit() *apierr.Error {
	return apierr.Conflict("budget_item_limit_reached", fmt.Sprintf("at most %d budget lines per project", maxLines))
}

// applyPlan is what an application of a scenario does, worked out before
// anything is written: apply-preview returns its report, apply executes it and
// returns the same report.
type applyPlan struct {
	report  domain.ApplyReport
	ids     []uuid.UUID // lines whose estimate changes
	amounts []int64     // …and their new estimates
	creates []domain.ScenarioItem
}

func newReport(mode string) domain.ApplyReport {
	return domain.ApplyReport{
		Mode: mode, Updated: []domain.AppliedUpdate{}, Created: []domain.AppliedCreate{}, Skipped: []domain.AppliedSkip{},
	}
}

// smartAdjustPlan is the Smart Adjust of 03 BE 3.2 over the scenario items
// already paired with the project's lines. totalLines and currentEstimate are
// the project's line count and Σ estimate before the application.
//
// A paired line is skipped when it rolls up the shopping list or is priced per
// unit, left alone when it already has the amount (so applying twice changes
// nothing), and otherwise gets the scenario amount as its estimate: payments,
// vendor and document are never touched. An item without a line is created. Two
// items paired with the same line (two keyless items of one name): the second one
// is created as a line of its own. Lines no item pairs with stay as they are.
func smartAdjustPlan(rows []domain.AdjustRow, totalLines int, currentEstimate int64) applyPlan {
	plan := applyPlan{report: newReport(domain.ApplySmartAdjust)}
	used := map[uuid.UUID]bool{}
	after := currentEstimate

	for _, r := range rows {
		it := r.Item
		if r.BudgetID == nil || used[*r.BudgetID] {
			plan.creates = append(plan.creates, it)
			plan.report.Created = append(plan.report.Created, domain.AppliedCreate{
				SubEventScope: it.SubEventScope, ItemKey: it.ItemKey, ItemName: it.ItemName, Amount: it.CostAmount,
			})
			after += it.CostAmount

			continue
		}
		used[*r.BudgetID] = true

		switch {
		case r.Rollup:
			plan.report.Skipped = append(plan.report.Skipped, skip(it, domain.SkipRollup))
		case r.Unit:
			plan.report.Skipped = append(plan.report.Skipped, skip(it, domain.SkipUnitPricing))
		case r.From != it.CostAmount:
			plan.ids = append(plan.ids, *r.BudgetID)
			plan.amounts = append(plan.amounts, it.CostAmount)
			plan.report.Updated = append(plan.report.Updated, domain.AppliedUpdate{
				SubEventScope: it.SubEventScope, ItemKey: it.ItemKey, ItemName: it.ItemName, From: r.From, To: it.CostAmount,
			})
			after += it.CostAmount - r.From
		}
	}

	plan.report.UntouchedCount = totalLines - len(used)
	plan.report.EstimateAfter = after

	return plan
}

func skip(it domain.ScenarioItem, reason string) domain.AppliedSkip {
	return domain.AppliedSkip{SubEventScope: it.SubEventScope, ItemKey: it.ItemKey, ItemName: it.ItemName, Reason: reason}
}

// cleanOverwritePlan deletes every line of the project and copies the items
// that take part (included) as new lines.
func cleanOverwritePlan(sc domain.Scenario, totals domain.LineTotals) applyPlan {
	plan := applyPlan{report: newReport(domain.ApplyCleanOverwrite)}
	plan.report.Deleted = &domain.AppliedDeleted{Lines: totals.Lines, PaidTotal: totals.Paid}
	for _, it := range sc.Items {
		if !it.Included {
			continue
		}
		plan.creates = append(plan.creates, it)
		plan.report.Created = append(plan.report.Created, domain.AppliedCreate{
			SubEventScope: it.SubEventScope, ItemKey: it.ItemKey, ItemName: it.ItemName, Amount: it.CostAmount,
		})
		plan.report.EstimateAfter += it.CostAmount
	}

	return plan
}

func checkScenarioID(id uuid.UUID) error {
	if id == uuid.Nil {
		return invalidInput("scenario_id", "required")
	}

	return nil
}

func parseApplyMode(raw string) (string, error) {
	switch mode := strings.TrimSpace(raw); mode {
	case "", domain.ApplySmartAdjust:
		return domain.ApplySmartAdjust, nil
	case domain.ApplyCleanOverwrite:
		return mode, nil
	}

	return "", invalidInput("mode", "must be one of smart_adjust, clean_overwrite")
}

// ApplyPreview reports what Apply would do without writing anything.
func (s *Service) ApplyPreview(ctx context.Context, p pd.Project, req domain.ApplyScenarioRequest) (domain.ApplyReport, error) {
	if err := checkScenarioID(req.ScenarioID); err != nil {
		return domain.ApplyReport{}, err
	}
	mode, err := parseApplyMode(req.Mode)
	if err != nil {
		return domain.ApplyReport{}, err
	}
	sc, err := s.scenario(ctx, p, req.ScenarioID)
	if err != nil {
		return domain.ApplyReport{}, err
	}
	plan, err := s.plan(ctx, p, sc, mode)
	if err != nil {
		return domain.ApplyReport{}, err
	}
	s.syncReport(&plan.report, sc, req)

	return plan.report, nil
}

func (s *Service) plan(ctx context.Context, p pd.Project, sc domain.Scenario, mode string) (applyPlan, error) {
	totals, err := s.repo.LineTotals(ctx, p.ID)
	if err != nil {
		return applyPlan{}, err
	}
	if mode == domain.ApplyCleanOverwrite {
		return cleanOverwritePlan(sc, totals), nil
	}
	rows, err := s.repo.AdjustRows(ctx, p.ID, sc.ID)
	if err != nil {
		return applyPlan{}, err
	}

	return smartAdjustPlan(rows, totals.Lines, totals.Estimate), nil
}

func (s *Service) syncReport(r *domain.ApplyReport, sc domain.Scenario, req domain.ApplyScenarioRequest) {
	if req.SyncTargetBudget {
		t := sc.TotalTarget
		r.NewTargetBudget = &t
	}
}

// Apply makes a scenario the active one and lays it over the budget lines, in
// one transaction under the project lock (03 BE 3.2).
func (s *Service) Apply(ctx context.Context, p pd.Project, req domain.ApplyScenarioRequest) (domain.ApplyReport, error) {
	if err := checkScenarioID(req.ScenarioID); err != nil {
		return domain.ApplyReport{}, err
	}
	mode, err := parseApplyMode(req.Mode)
	if err != nil {
		return domain.ApplyReport{}, err
	}
	if mode == domain.ApplyCleanOverwrite {
		switch {
		case req.Confirm == nil || strings.TrimSpace(*req.Confirm) == "":
			return domain.ApplyReport{}, confirmRequired()
		case *req.Confirm != domain.ConfirmReset:
			return domain.ApplyReport{}, invalidInput("confirm", "must be "+domain.ConfirmReset)
		}
	}

	var report domain.ApplyReport
	err = s.tx.InTx(ctx, func(ctx context.Context) error {
		if err := s.repo.LockProject(ctx, p.ID); err != nil {
			return err
		}
		sc, err := s.repo.ScenarioByID(ctx, p.ID, req.ScenarioID)
		if err != nil {
			return err
		}
		enrich(&sc, p.EventType)

		plan, err := s.plan(ctx, p, sc, mode)
		if err != nil {
			return err
		}
		if err := s.execute(ctx, p, mode, plan); err != nil {
			return err
		}

		// Two statements, in this order: one UPDATE … SET is_active = (id = $x)
		// trips the partial unique index depending on the physical row order.
		if err := s.repo.DeactivateScenarios(ctx, p.ID); err != nil {
			return err
		}
		if err := s.repo.ActivateScenario(ctx, p.ID, sc.ID); err != nil {
			return err
		}

		if req.SyncTargetBudget || req.SyncTargetSavings {
			var target *int64
			if req.SyncTargetBudget {
				target = &sc.TotalTarget
			}
			if err := s.repo.SetProjectTargets(ctx, p.ID, target, req.SyncTargetSavings); err != nil {
				return err
			}
		}

		s.syncReport(&plan.report, sc, req)
		report = plan.report

		return nil
	})
	if err != nil {
		return domain.ApplyReport{}, mapErr(err, "scenario")
	}

	return report, nil
}

// execute writes the plan: new estimates, then the new lines at the end of the
// order. The line limit is checked on the result, so a refused application
// leaves nothing behind.
func (s *Service) execute(ctx context.Context, p pd.Project, mode string, plan applyPlan) error {
	total, err := s.repo.CountLines(ctx, p.ID)
	if err != nil {
		return err
	}

	if mode == domain.ApplyCleanOverwrite {
		if err := s.repo.DeleteAllBudgetItems(ctx, p.ID); err != nil {
			return err
		}
		total = 0
	}
	if total+len(plan.creates) > maxLines {
		return lineLimit()
	}

	if err := s.repo.UpdateEstimates(ctx, p.ID, plan.ids, plan.amounts); err != nil {
		return err
	}
	next, err := s.repo.NextLineSortOrder(ctx, p.ID)
	if err != nil {
		return err
	}
	for i, it := range plan.creates {
		line := domain.BudgetItem{
			ID: uuid.New(), ProjectID: p.ID, SubEventScope: it.SubEventScope, Category: it.Category, ItemName: it.ItemName,
			ItemKey: it.ItemKey, Source: domain.SourceScenario, PricingMode: domain.PricingLump, InitialEstimate: it.CostAmount, SortOrder: next + i,
		}
		if err := s.repo.InsertBudgetItem(ctx, &line); err != nil {
			return err
		}
	}

	return nil
}
