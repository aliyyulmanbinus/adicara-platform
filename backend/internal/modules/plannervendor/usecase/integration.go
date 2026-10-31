package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannervendor/domain"
)

// The module sits on four of menu 01's plug-in points. Each key of the host belongs to one role, so every role is a
// small type that wraps the Service (as in menu 02). Register them in internal/app/app.go:
//
//	provider   "vendors"   the dashboard widget
//	applier    "shopping"  the wizard template (the wedding's 37 seserahan items)
//	overlay    "vendors"   the 10 sample vendors, linked to two budget lines and one task
//	overlay    "shopping"  the statuses and actual prices of the 37 sample items

// ---------- dashboard widget ----------

type provider struct{ s *Service }

func (provider) Key() string { return "vendors" }

// Summarize is the vendors widget (01 BE 5.2): the KPI part of the overview, so it counts exactly what the tab counts.
// It needs the vendors only (the limits come from the scenario and are not part of the widget).
func (p provider) Summarize(ctx context.Context, project pd.Project) (any, error) {
	vendors, err := p.s.repo.ListVendors(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	cats, _ := p.s.specs(project)
	k := domain.BuildOverview(cats, project.SkippedVendorCategories, vendors, nil, nil).KPI

	return pd.VendorSummary{
		CategoriesTotal: k.CategoriesTotal, CategoriesFixed: k.CategoriesFixed, Skipped: k.Skipped,
		FixedVendors: k.FixedVendors, Pipeline: k.Pipeline,
	}, nil
}

func (s *Service) Provider() pd.SummaryProvider { return provider{s} }

// ---------- project wizard template ----------

type applier struct{ s *Service }

func (applier) Key() string { return pd.TemplateShopping }

// Apply runs inside the project-creation transaction; the wizard only appends. The wizard creates no vendors.
func (a applier) Apply(ctx context.Context, _ uuid.UUID, req pd.ApplyRequest) (pd.ApplyResult, error) {
	r, err := a.s.applyTemplate(ctx, req.Project, domain.ModeAppend, false)
	if err != nil {
		return pd.ApplyResult{}, err
	}

	return pd.ApplyResult{Created: r.Created, Skipped: r.Skipped, Warnings: r.Warnings}, nil
}

func (s *Service) Applier() pd.TemplateApplier { return applier{s} }

// ---------- sample project (Wedding_Planner.pdf) ----------

type vendorsOverlay struct{ s *Service }

func (vendorsOverlay) Key() string { return "vendors" }

// ApplySample writes the 10 vendors of 04 BE 5.2 and points two budget lines and one task at them. It needs the budget
// lines and the task that the budget and timeline appliers made, which the host runs before any overlay. It does not run lock-deal: Wedding Organizer
// would make a 15th line and change the total of the PDF; the two lines it links already hold the deal as estimate.
func (o vendorsOverlay) ApplySample(ctx context.Context, p pd.Project) error {
	s := o.s
	cats, _ := s.specs(p)
	byKey := make(map[string]domain.CategorySpec, len(cats))
	for _, c := range cats {
		byKey[c.Key] = c
	}

	ids := make(map[string]uuid.UUID, len(domain.SampleVendors))
	for _, sv := range domain.SampleVendors {
		c, ok := byKey[sv.CategoryKey]
		if !ok {
			return fmt.Errorf("sample vendor %q: unknown category %q", sv.Name, sv.CategoryKey)
		}
		phone := sv.Phone
		wa, err := pd.NormalizeWA(phone)
		if err != nil {
			return fmt.Errorf("sample vendor %q: %w", sv.Name, err)
		}
		row := domain.VendorRow{
			ID: uuid.New(), SubEventScope: domain.SampleScope, CategoryKey: c.Key, CategoryLabel: c.Label, VendorName: sv.Name,
			PhoneNumber: &phone, FormattedWANumber: &wa, AgreedPrice: sv.Price, Status: sv.Status,
		}
		if err := s.repo.InsertVendor(ctx, p.ID, &row); err != nil {
			return err
		}
		ids[sv.CategoryKey] = row.ID
	}

	for _, l := range domain.SampleLineLinks {
		if err := s.repo.LinkLineByKey(ctx, p.ID, l.Scope, l.Key, ids[l.VendorCategory]); err != nil {
			return fmt.Errorf("sample budget line %s/%s: %w", l.Scope, l.Key, err)
		}
	}
	if err := s.repo.LinkTaskByTemplateKey(ctx, p.ID, domain.SampleTaskLink.TemplateKey, ids[domain.SampleTaskLink.VendorCategory]); err != nil {
		return fmt.Errorf("sample task %s: %w", domain.SampleTaskLink.TemplateKey, err)
	}

	return nil
}

func (s *Service) VendorsOverlay() pd.SampleOverlay { return vendorsOverlay{s} }

type shoppingOverlay struct{ s *Service }

func (shoppingOverlay) Key() string { return "shopping" }

// ApplySample gives the 37 template items the status of the PDF: 16 finished with their reference price as actual
// price (Rp 2.063.800), 21 in progress (Rp 2.514.700). Rollup stays off, so the budget keeps the PDF figures.
func (o shoppingOverlay) ApplySample(ctx context.Context, p pd.Project) error {
	for _, it := range domain.TemplateFor(p.EventType) {
		status, actual := domain.ShoppingInProgress, int64(0)
		if it.Done {
			status, actual = domain.ShoppingDone, it.Price
		}
		if err := o.s.repo.SetSampleShopping(ctx, p.ID, it.Key(), status, actual); err != nil {
			return fmt.Errorf("sample shopping item %s: %w", it.Name, err)
		}
	}

	return nil
}

func (s *Service) ShoppingOverlay() pd.SampleOverlay { return shoppingOverlay{s} }
