package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerfinance/domain"
)

// The module has one key on each of menu 01's plug-in points (finance for the
// widget and the overlay, budget for the template), so every role is a small
// type that wraps the Service. Register them in internal/app/app.go.

// ---------- dashboard widget ----------

type provider struct{ s *Service }

func (provider) Key() string { return "finance" }

// Summarize is the finance widget (01 BE 5.2): the figures of the view
// planner_budget_lines and the savings. Remaining and Overpaid add up the
// positive and the negative part of every line, so one overpaid line never hides
// what another still owes.
func (p provider) Summarize(ctx context.Context, project pd.Project) (any, error) {
	t, err := p.s.repo.LineTotals(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	collected, err := p.s.repo.SavingsCollected(ctx, project.ID)
	if err != nil {
		return nil, err
	}

	return financeSummary(project, t, collected), nil
}

func financeSummary(project pd.Project, t domain.LineTotals, collected int64) pd.FinanceSummary {
	savingsTarget := project.TargetBudget
	if project.SavingsTarget != nil {
		savingsTarget = *project.SavingsTarget
	}
	free := collected - t.Paid

	return pd.FinanceSummary{
		TargetBudget: project.TargetBudget, TotalEstimate: t.Estimate, TotalPaid: t.Paid,
		Remaining: t.Remaining, Overpaid: t.Overpaid,
		SavingsTarget: savingsTarget, SavingsCollected: collected,
		FreeCash: free, CashDeficit: max(t.Remaining-free, 0),
		LineStatus: t.Status,
	}
}

func (s *Service) Provider() pd.SummaryProvider { return provider{s} }

// ---------- project wizard template ----------

type applier struct{ s *Service }

func (applier) Key() string { return pd.TemplateBudget }

// Apply runs inside the project-creation transaction; the wizard only appends.
func (a applier) Apply(ctx context.Context, _ uuid.UUID, req pd.ApplyRequest) (pd.ApplyResult, error) {
	r, err := a.s.applyTemplate(ctx, req.Project, domain.ModeAppend, nil, false)
	if err != nil {
		return pd.ApplyResult{}, err
	}

	return pd.ApplyResult{Created: r.Created, Skipped: r.Skipped, Warnings: r.Warnings}, nil
}

func (s *Service) Applier() pd.TemplateApplier { return applier{s} }

// ---------- sample project (Wedding_Planner.pdf) ----------

type overlay struct{ s *Service }

func (overlay) Key() string { return "finance" }

// ApplySample lays the dataset of Wedding_Planner.pdf p. 4-7 over the sample
// project, inside the same transaction and after the budget template was
// applied: the estimates and payments of the 14 lines, Arif and Sinta with their
// 22 deposits, and the three scenarios (the 75 Jt one active). The scenarios are
// not applied to the lines: the PDF estimates differ from the presets, so the
// sample shows 850.000 still unallocated, as the PDF does.
func (o overlay) ApplySample(ctx context.Context, p pd.Project) error {
	s := o.s
	for _, pay := range domain.SamplePayments() {
		if err := s.repo.ApplySamplePayment(ctx, p.ID, pay); err != nil {
			return fmt.Errorf("sample budget line %s/%s: %w", pay.Scope, pay.Key, err)
		}
	}

	ids := make([]uuid.UUID, len(domain.SampleContributors))
	for i, c := range domain.SampleContributors {
		ids[i] = uuid.New()
		if err := s.repo.InsertContributor(ctx, p.ID, &domain.Contributor{ID: ids[i], Name: c.Name, RoleLabel: c.Role}); err != nil {
			return err
		}
	}
	for _, e := range domain.SampleEntries() {
		entry := domain.SavingsEntry{ID: uuid.New(), ContributorID: ids[e.Contributor], EntryDate: e.Date.Format(pd.DateLayout), Amount: e.Amount}
		if err := s.repo.InsertEntry(ctx, p.ID, &entry); err != nil {
			return err
		}
	}

	for _, sc := range domain.SampleScenarios {
		items, total, err := presetPlan(p.EventType, sc.PresetKey, nil)
		if err != nil {
			return err
		}
		key := sc.PresetKey
		made, _, err := s.insertScenario(ctx, p, sc.Name, &key, total, items, domain.SampleActiveSessions)
		if err != nil {
			return err
		}
		if sc.Active {
			if err := s.repo.DeactivateScenarios(ctx, p.ID); err != nil {
				return err
			}
			if err := s.repo.ActivateScenario(ctx, p.ID, made.ID); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Service) Overlay() pd.SampleOverlay { return overlay{s} }
