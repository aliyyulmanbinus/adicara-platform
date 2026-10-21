package usecase

import (
	"testing"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	plannerusecase "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/usecase"
)

// The host panics at start on a key it does not know or a key registered twice (a wiring bug must stop the
// process, not surface later as an empty widget). The three roles of this module use keys the host accepts.
func TestFinanceWiring(t *testing.T) {
	registry := pd.DefaultRegistry()
	fin := New(nil, nil, registry, Options{})
	host := plannerusecase.New(nil, nil, registry, plannerusecase.Options{})

	host.RegisterSummary(fin.Provider())
	host.RegisterTemplateAppliers(fin.Applier())
	host.RegisterSampleOverlays(fin.Overlay())

	if fin.Provider().Key() != "finance" || fin.Applier().Key() != pd.TemplateBudget || fin.Overlay().Key() != "finance" {
		t.Errorf("keys = %q %q %q", fin.Provider().Key(), fin.Applier().Key(), fin.Overlay().Key())
	}
	for name, register := range map[string]func(){
		"a second provider": func() { host.RegisterSummary(fin.Provider()) },
		"a second applier":  func() { host.RegisterTemplateAppliers(fin.Applier()) },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s must panic", name)
				}
			}()
			register()
		}()
	}

	// The wizard now offers the budget template for every event type that owns one.
	for _, et := range host.EventTypes() {
		if !et.TemplatesAvailable.BudgetLines {
			t.Errorf("%s: budget_lines is not offered", et.Key)
		}
	}
}
