package usecase

import (
	"testing"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	plannerusecase "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/usecase"
)

// The host panics at start on a key it does not know or a key registered twice (a wiring bug must stop the process, not surface
// later as an empty widget). The four roles of this module use keys the host accepts.
func TestVendorWiring(t *testing.T) {
	registry := pd.DefaultRegistry()
	svc := New(nil, nil, registry, Options{})
	host := plannerusecase.New(nil, nil, registry, plannerusecase.Options{})

	host.RegisterSummary(svc.Provider())
	host.RegisterTemplateAppliers(svc.Applier())
	host.RegisterSampleOverlays(svc.VendorsOverlay(), svc.ShoppingOverlay())

	if svc.Provider().Key() != "vendors" || svc.Applier().Key() != pd.TemplateShopping || svc.VendorsOverlay().Key() != "vendors" || svc.ShoppingOverlay().Key() != "shopping" {
		t.Errorf("keys = %q %q %q %q", svc.Provider().Key(), svc.Applier().Key(), svc.VendorsOverlay().Key(), svc.ShoppingOverlay().Key())
	}
	for name, register := range map[string]func(){
		"a second provider": func() { host.RegisterSummary(svc.Provider()) },
		"a second applier":  func() { host.RegisterTemplateAppliers(svc.Applier()) },
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

	// The wizard now offers the shopping list for the event type that owns one: the wedding.
	for _, et := range host.EventTypes() {
		if want := et.Key == pd.EventWedding; et.TemplatesAvailable.ShoppingList != want {
			t.Errorf("%s: shopping_list = %v, want %v", et.Key, et.TemplatesAvailable.ShoppingList, want)
		}
	}
}
