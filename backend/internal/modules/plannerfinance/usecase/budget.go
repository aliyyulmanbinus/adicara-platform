package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerfinance/domain"
)

// ---------- reading ----------

// ListItems returns the budget lines in sort_order. No scope = every session;
// scope "all" = the general lines only; any other scope_key = that session
// (the rule of menu 02).
func (s *Service) ListItems(ctx context.Context, p pd.Project, scope string) (domain.BudgetLineList, error) {
	var filter *string
	if scope != "" {
		if !scopePattern.MatchString(scope) {
			return domain.BudgetLineList{}, invalidInput("scope", "must be a session scope_key")
		}
		filter = &scope
	}
	lines, err := s.repo.ListLines(ctx, p.ID, filter)
	if err != nil {
		return domain.BudgetLineList{}, err
	}

	return domain.BudgetLineList{Items: lines}, nil
}

// Cashflow is GET /budget/cashflow-summary (03 BE 3.5). remaining_debt adds up
// the positive part of every line: an overpaid line does not reduce what
// another line still owes.
func (s *Service) Cashflow(ctx context.Context, p pd.Project) (domain.CashflowSummary, error) {
	t, err := s.repo.LineTotals(ctx, p.ID)
	if err != nil {
		return domain.CashflowSummary{}, err
	}
	collected, err := s.repo.SavingsCollected(ctx, p.ID)
	if err != nil {
		return domain.CashflowSummary{}, err
	}

	return cashflow(t, collected), nil
}

func cashflow(t domain.LineTotals, collected int64) domain.CashflowSummary {
	available := collected - t.Paid

	return domain.CashflowSummary{
		TotalSavingsCollected: collected, TotalPaid: t.Paid, AvailableCash: available,
		RemainingDebt: t.Remaining, Overpaid: t.Overpaid, CashDeficit: max(t.Remaining-available, 0),
	}
}

// Matrix is GET /budget/matrix: the KPI, one row per session that has lines
// (Umum last), their total and the active scenario with what it leaves unallocated.
func (s *Service) Matrix(ctx context.Context, p pd.Project) (domain.BudgetMatrix, error) {
	rows, err := s.repo.MatrixRows(ctx, p.ID)
	if err != nil {
		return domain.BudgetMatrix{}, err
	}
	t, err := s.repo.LineTotals(ctx, p.ID)
	if err != nil {
		return domain.BudgetMatrix{}, err
	}
	id, name, total, found, err := s.repo.ActiveScenario(ctx, p.ID)
	if err != nil {
		return domain.BudgetMatrix{}, err
	}

	m := buildMatrix(rows, t)
	if found {
		m.ActiveScenario = &domain.ActiveScenario{ID: id, Name: name, TotalTarget: total, Unallocated: total - t.Estimate}
	}

	return m, nil
}

func buildMatrix(rows []domain.MatrixRow, t domain.LineTotals) domain.BudgetMatrix {
	total := domain.MatrixRow{}
	for _, r := range rows {
		total.Estimate += r.Estimate
		total.DP += r.DP
		total.Term1 += r.Term1
		total.Term2 += r.Term2
		total.RollupPaid += r.RollupPaid
		total.Paid += r.Paid
		total.Remaining += r.Remaining
		total.Overpaid += r.Overpaid
		total.LineCount += r.LineCount
	}

	return domain.BudgetMatrix{
		KPI: domain.MatrixKPI{
			TotalEstimate: t.Estimate, TotalPaid: t.Paid, TotalRemaining: t.Remaining, TotalOverpaid: t.Overpaid,
			LineCount: t.Lines, LineStatus: t.Status,
		},
		Rows: rows, Total: total,
	}
}

// ---------- writing a line ----------

// lineFields is the part of BudgetItemRequest every line has, rollup or not.
func (s *Service) lineFields(ctx context.Context, p pd.Project, req domain.BudgetItemRequest) (domain.BudgetItem, error) {
	scope, err := normalizeScope(req.SubEventScope)
	if err != nil {
		return domain.BudgetItem{}, err
	}
	category, err := cleanText("category", req.Category, maxCategoryRunes, true)
	if err != nil {
		return domain.BudgetItem{}, err
	}
	name, err := cleanText("item_name", req.ItemName, maxItemNameRunes, true)
	if err != nil {
		return domain.BudgetItem{}, err
	}
	notes, err := optionalNote("notes", req.Notes)
	if err != nil {
		return domain.BudgetItem{}, err
	}
	if err := s.checkScope(ctx, p.ID, scope); err != nil {
		return domain.BudgetItem{}, err
	}
	if req.VendorID != nil {
		ok, err := s.repo.VendorInProject(ctx, p.ID, *req.VendorID)
		if err != nil {
			return domain.BudgetItem{}, err
		}
		if !ok {
			return domain.BudgetItem{}, invalidInput("vendor_id", "must be a vendor of this project")
		}
	}

	return domain.BudgetItem{ProjectID: p.ID, SubEventScope: scope, Category: category, ItemName: name, VendorID: req.VendorID, Notes: notes}, nil
}

// pricing validates and applies the pricing fields. A unit-priced line has its
// estimate worked out here, from quantity × unit price, whatever the client sent.
func pricing(it *domain.BudgetItem, req domain.BudgetItemRequest) error {
	mode := domain.PricingLump
	if req.PricingMode != nil && strings.TrimSpace(*req.PricingMode) != "" {
		mode = strings.TrimSpace(*req.PricingMode)
	}
	label, err := cleanText("unit_label", deref(req.UnitLabel), maxUnitRunes, false)
	if err != nil {
		return err
	}

	switch mode {
	case domain.PricingLump:
		var estimate int64
		if req.InitialEstimate != nil {
			estimate = *req.InitialEstimate
		}
		if err := validateMoney("initial_estimate", estimate); err != nil {
			return err
		}
		it.PricingMode, it.InitialEstimate = mode, estimate
		it.QuantityX100, it.UnitLabel, it.UnitPrice = nil, nil, nil

	case domain.PricingUnit:
		if req.Quantity == nil {
			return invalidInput("quantity", "required when pricing_mode is unit")
		}
		if req.UnitPrice == nil {
			return invalidInput("unit_price", "required when pricing_mode is unit")
		}
		q, ok := domain.QuantityX100(*req.Quantity)
		if !ok || q < domain.MinQuantityX100 || q > domain.MaxQuantityX100 {
			return invalidInput("quantity", "must be between 0.01 and 1000000")
		}
		if err := validateMoney("unit_price", *req.UnitPrice); err != nil {
			return err
		}
		estimate, ok := domain.UnitEstimate(q, *req.UnitPrice)
		if !ok {
			return invalidInput("unit_price", fmt.Sprintf("the estimate (quantity × unit_price) must be at most %d", domain.MaxMoney))
		}
		it.PricingMode, it.InitialEstimate, it.QuantityX100, it.UnitPrice = mode, estimate, &q, req.UnitPrice
		it.UnitLabel = nil
		if label != "" {
			it.UnitLabel = &label
		}

	default:
		return invalidInput("pricing_mode", "must be one of lump, unit")
	}

	return nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}

// CreateItem adds a line the user made (no item_key, source manual) at the end
// of the order. It runs under the project lock so a double click cannot pass the
// limit of 300.
func (s *Service) CreateItem(ctx context.Context, p pd.Project, req domain.BudgetItemRequest) (domain.BudgetLine, error) {
	var out domain.BudgetLine
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		it, err := s.lineFields(ctx, p, req)
		if err != nil {
			return err
		}
		if err := pricing(&it, req); err != nil {
			return err
		}
		it.ID, it.Source = uuid.New(), domain.SourceManual

		if err := s.repo.LockProject(ctx, p.ID); err != nil {
			return err
		}
		n, err := s.repo.CountLines(ctx, p.ID)
		if err != nil {
			return err
		}
		if n >= maxLines {
			return lineLimit()
		}
		if it.SortOrder, err = s.repo.NextLineSortOrder(ctx, p.ID); err != nil {
			return err
		}
		if err := s.repo.InsertBudgetItem(ctx, &it); err != nil {
			return err
		}
		out, err = s.repo.LineByID(ctx, p.ID, it.ID)

		return err
	})
	if err != nil {
		return domain.BudgetLine{}, mapErr(err, "budget item")
	}

	return out, nil
}

// UpdateItem replaces the line (PUT): an optional field that is not sent is
// cleared, including vendor_id, which releases the vendor. source, item_key,
// the document link, the rollup and the payments stay. A line that rolls up the
// shopping list keeps its pricing and estimate (they come from the list). Moving a
// line to a session that already has its item_key makes it a keyless line.
func (s *Service) UpdateItem(ctx context.Context, p pd.Project, id uuid.UUID, req domain.BudgetItemRequest) (domain.BudgetLine, error) {
	var out domain.BudgetLine
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		if err := s.repo.LockBudgetItem(ctx, p.ID, id); err != nil {
			return err
		}
		cur, err := s.repo.BudgetItemByID(ctx, p.ID, id)
		if err != nil {
			return err
		}
		next, err := s.lineFields(ctx, p, req)
		if err != nil {
			return err
		}

		next.ID, next.Source, next.ItemKey, next.SortOrder, next.Rollup = cur.ID, cur.Source, cur.ItemKey, cur.SortOrder, cur.Rollup
		if cur.Rollup {
			next.PricingMode, next.QuantityX100, next.UnitLabel, next.UnitPrice, next.InitialEstimate =
				cur.PricingMode, cur.QuantityX100, cur.UnitLabel, cur.UnitPrice, cur.InitialEstimate
		} else if err := pricing(&next, req); err != nil {
			return err
		}

		if cur.ItemKey != nil && next.SubEventScope != cur.SubEventScope {
			taken, err := s.repo.BudgetKeyTaken(ctx, p.ID, next.SubEventScope, *cur.ItemKey, cur.ID)
			if err != nil {
				return err
			}
			if taken {
				next.ItemKey = nil
			}
		}

		if err := s.repo.UpdateBudgetItem(ctx, &next); err != nil {
			return err
		}
		out, err = s.repo.LineByID(ctx, p.ID, id)

		return err
	})
	if err != nil {
		return domain.BudgetLine{}, mapErr(err, "budget item")
	}

	return out, nil
}

// DeleteItem removes a line together with its payments; shopping items and
// documents that pointed at it keep their figures and lose the link.
func (s *Service) DeleteItem(ctx context.Context, p pd.Project, id uuid.UUID) error {
	return mapErr(s.repo.DeleteBudgetItem(ctx, p.ID, id), "budget item")
}

// ReorderItems sets sort_order = 1…n in the given order; ids must list every
// line of the project exactly once.
func (s *Service) ReorderItems(ctx context.Context, p pd.Project, ids []uuid.UUID) (domain.BudgetLineList, error) {
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		if err := s.repo.LockProject(ctx, p.ID); err != nil {
			return err
		}
		n, err := s.repo.CountLines(ctx, p.ID)
		if err != nil {
			return err
		}
		if err := checkIDs(ids, n); err != nil {
			return err
		}
		changed, err := s.repo.ReorderBudgetItems(ctx, p.ID, ids)
		if err != nil {
			return err
		}
		if changed != n {
			return invalidInput("ids", "must list every id exactly once")
		}

		return nil
	})
	if err != nil {
		return domain.BudgetLineList{}, mapErr(err, "budget item")
	}

	return s.ListItems(ctx, p, "")
}

// ---------- payments ----------

func invalidStage() *apierr.Error {
	return apierr.BadRequest("invalid_stage", "payment_stage: there is no stage left to pay")
}

func rollupLocked() *apierr.Error {
	return apierr.Conflict("rollup_locked", "the figures of this line come from the shopping list")
}

// RecordPayment writes one payment stage (03 BE 3.3). dp, term_1 and term_2
// replace everything in their stage (amount, date, receipt), which also corrects
// a wrong entry. full pays the remainder into the first empty stage and ignores
// the amount, so a double click cannot pay twice. An overpayment is allowed.
func (s *Service) RecordPayment(ctx context.Context, p pd.Project, id uuid.UUID, req domain.PaymentRequest) (domain.BudgetLine, error) {
	stage := strings.TrimSpace(req.PaymentStage)
	switch stage {
	case domain.StageDP, domain.StageTerm1, domain.StageTerm2, domain.StageFull:
	default:
		return domain.BudgetLine{}, invalidInput("payment_stage", "must be one of dp, term_1, term_2, full")
	}

	if strings.TrimSpace(req.PaidAt) == "" {
		return domain.BudgetLine{}, invalidDate("paid_at", "required")
	}
	paidAt, err := parseDate("paid_at", req.PaidAt)
	if err != nil {
		return domain.BudgetLine{}, err
	}
	if err := checkWindow(p, "paid_at", paidAt); err != nil {
		return domain.BudgetLine{}, err
	}
	receipt, err := cleanReceiptURL(req.ReceiptURL)
	if err != nil {
		return domain.BudgetLine{}, err
	}

	var amount int64
	if stage != domain.StageFull {
		if req.Amount == nil {
			return domain.BudgetLine{}, invalidInput("amount", "required")
		}
		if *req.Amount <= 0 {
			return domain.BudgetLine{}, invalidInput("amount", "must be greater than 0")
		}
		if err := validateMoney("amount", *req.Amount); err != nil {
			return domain.BudgetLine{}, err
		}
		amount = *req.Amount
	}

	var out domain.BudgetLine
	err = s.tx.InTx(ctx, func(ctx context.Context) error {
		if err := s.repo.LockBudgetItem(ctx, p.ID, id); err != nil {
			return err
		}
		line, err := s.repo.LineByID(ctx, p.ID, id)
		if err != nil {
			return err
		}
		if line.RollupSource != nil {
			return rollupLocked()
		}

		if stage == domain.StageFull {
			if line.Remaining <= 0 {
				return invalidStage()
			}
			if stage = firstEmptyStage(line); stage == "" {
				return invalidStage()
			}
			amount = line.Remaining
		}
		if err := s.repo.SetPayment(ctx, p.ID, id, stage, amount, &paidAt, receipt); err != nil {
			return err
		}
		out, err = s.repo.LineByID(ctx, p.ID, id)

		return err
	})
	if err != nil {
		return domain.BudgetLine{}, mapErr(err, "budget item")
	}

	return out, nil
}

// firstEmptyStage is the first of dp, term_1, term_2 whose amount is 0, or "".
func firstEmptyStage(l domain.BudgetLine) string {
	switch {
	case l.DPAmount == 0:
		return domain.StageDP
	case l.Term1Amount == 0:
		return domain.StageTerm1
	case l.Term2Amount == 0:
		return domain.StageTerm2
	}

	return ""
}

// ClearPayment empties one stage (amount 0, no date, no receipt): the way to
// take back a payment that was entered by mistake.
func (s *Service) ClearPayment(ctx context.Context, p pd.Project, id uuid.UUID, stage string) (domain.BudgetLine, error) {
	switch stage {
	case domain.StageDP, domain.StageTerm1, domain.StageTerm2:
	default:
		return domain.BudgetLine{}, invalidInput("stage", "must be one of dp, term_1, term_2")
	}

	var out domain.BudgetLine
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		if err := s.repo.LockBudgetItem(ctx, p.ID, id); err != nil {
			return err
		}
		line, err := s.repo.LineByID(ctx, p.ID, id)
		if err != nil {
			return err
		}
		if line.RollupSource != nil {
			return rollupLocked()
		}
		if err := s.repo.SetPayment(ctx, p.ID, id, stage, 0, nil, nil); err != nil {
			return err
		}
		out, err = s.repo.LineByID(ctx, p.ID, id)

		return err
	})
	if err != nil {
		return domain.BudgetLine{}, mapErr(err, "budget item")
	}

	return out, nil
}

// ---------- template ----------

// ApplyTemplate is POST /budget/apply-template.
func (s *Service) ApplyTemplate(ctx context.Context, p pd.Project, req domain.ApplyTemplateRequest) (domain.ApplyTemplateResult, error) {
	mode := strings.TrimSpace(req.Mode)
	switch mode {
	case "":
		mode = domain.ModeAppend
	case domain.ModeAppend, domain.ModeReplaceTemplate:
	default:
		return domain.ApplyTemplateResult{}, invalidInput("mode", "must be one of append, replace_template")
	}

	var scope *string
	if req.SubEventScope != nil && strings.TrimSpace(*req.SubEventScope) != "" {
		sc, err := normalizeScope(*req.SubEventScope)
		if err != nil {
			return domain.ApplyTemplateResult{}, err
		}
		scope = &sc
	}

	return s.applyTemplate(ctx, p, mode, scope, req.Confirm)
}

// applyTemplate lays the budget template of the event type over the project:
// both the wizard (append, inside the project-creation transaction) and
// POST /budget/apply-template come here (03 BE 3.9).
//
//   - append adds the template lines whose (session, item_key) is not there yet;
//   - replace_template first deletes the template lines that have no payment (the
//     paid ones stay and count as skipped). confirm is needed when a line to
//     delete holds user work: an estimate, a vendor, a document or a shopping
//     link. The deleted lines lose those links (warning budget_link_lost).
//
// A session the user deleted sends its lines to the general session
// (warning scope_fallback). The limit of 300 lines is checked on the result.
func (s *Service) applyTemplate(ctx context.Context, p pd.Project, mode string, scope *string, confirm bool) (domain.ApplyTemplateResult, error) {
	if !domain.HasBudgetTemplate(p.EventType) {
		return domain.ApplyTemplateResult{}, invalidTemplate("budget_lines")
	}

	res := domain.ApplyTemplateResult{Warnings: []pd.Notice{}}
	linkLost := 0
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		res = domain.ApplyTemplateResult{Warnings: []pd.Notice{}}
		linkLost = 0

		if err := s.repo.LockProject(ctx, p.ID); err != nil {
			return err
		}
		set, _, err := s.scopeSet(ctx, p.ID)
		if err != nil {
			return err
		}
		if scope != nil && !set[*scope] {
			return errUnknownScope()
		}

		var lines []domain.LineTemplate
		fallback := 0
		for _, l := range domain.TemplateLines(p.EventType, p.TargetBudget) {
			if !set[l.Scope] {
				l.Scope = pd.AllScope
				if scope == nil || *scope == pd.AllScope {
					fallback++
				}
			}
			if scope == nil || l.Scope == *scope {
				lines = append(lines, l)
			}
		}

		if mode == domain.ModeReplaceTemplate {
			stats, err := s.repo.TemplateStats(ctx, p.ID, scope)
			if err != nil {
				return err
			}
			if stats.AtRisk > 0 && !confirm {
				return confirmRequired()
			}
			if res.Deleted, err = s.repo.DeleteTemplateLines(ctx, p.ID, scope); err != nil {
				return err
			}
			linkLost = stats.Linked
		}

		next, err := s.repo.NextLineSortOrder(ctx, p.ID)
		if err != nil {
			return err
		}
		for i, l := range lines {
			key := l.Key
			it := domain.BudgetItem{
				ID: uuid.New(), ProjectID: p.ID, SubEventScope: l.Scope, Category: l.Category, ItemName: l.Name, ItemKey: &key,
				Source: domain.SourceTemplate, PricingMode: domain.PricingLump, InitialEstimate: l.Estimate, SortOrder: next + i,
			}
			inserted, err := s.repo.InsertTemplateLine(ctx, &it)
			if err != nil {
				return err
			}
			if inserted {
				res.Created++
			} else {
				res.Skipped++
			}
		}

		total, err := s.repo.CountLines(ctx, p.ID)
		if err != nil {
			return err
		}
		if total > maxLines {
			return lineLimit()
		}
		if fallback > 0 {
			res.Warnings = append(res.Warnings, notice("scope_fallback", "lines of sessions the project no longer has were placed in the general session", fallback))
		}
		if linkLost > 0 {
			res.Warnings = append(res.Warnings, notice("budget_link_lost", "deleted lines that were linked to a vendor, a document or the shopping list lost the link; their figures stay where they were", linkLost))
		}

		return nil
	})
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ApplyTemplateResult{}, apierr.NotFound("project")
		}

		return domain.ApplyTemplateResult{}, mapErr(err, "budget item")
	}

	return res, nil
}

// Export gathers everything of GET /budget/export: every line (not a filtered
// view), the session matrix and the savings overview of the whole project.
func (s *Service) Export(ctx context.Context, p pd.Project) (domain.ExportData, error) {
	today, err := s.today(p)
	if err != nil {
		return domain.ExportData{}, err
	}
	_, sessions, err := s.scopeSet(ctx, p.ID)
	if err != nil {
		return domain.ExportData{}, err
	}
	lines, err := s.repo.ListLines(ctx, p.ID, nil)
	if err != nil {
		return domain.ExportData{}, err
	}
	matrix, err := s.Matrix(ctx, p)
	if err != nil {
		return domain.ExportData{}, err
	}
	savings, err := s.savings(ctx, p, today, nil)
	if err != nil {
		return domain.ExportData{}, err
	}

	return domain.ExportData{Project: p, Today: today, Sessions: sessions, Lines: lines, Matrix: matrix, Savings: savings}, nil
}
