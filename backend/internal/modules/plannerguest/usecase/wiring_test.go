package usecase

import (
	"testing"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	plannerusecase "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/usecase"
)

// The host panics at start on a key it does not know or a key registered twice (a wiring bug must stop the process, not surface
// later as an empty widget). The two roles of this module use keys the host accepts; it owns no template.
func TestGuestWiring(t *testing.T) {
	registry := pd.DefaultRegistry()
	svc := New(nil, nil, registry, Options{})
	host := plannerusecase.New(nil, nil, registry, plannerusecase.Options{})

	host.RegisterSummary(svc.Provider())
	host.RegisterSampleOverlays(svc.GuestsOverlay())

	if svc.Provider().Key() != "guests" || svc.GuestsOverlay().Key() != "guests" {
		t.Errorf("keys = %q %q", svc.Provider().Key(), svc.GuestsOverlay().Key())
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Error("a second provider must panic")
			}
		}()
		host.RegisterSummary(svc.Provider())
	}()

	// The wizard offers no guest template for any type: there is none.
	for _, et := range host.EventTypes() {
		if et.GuestSides == nil || et.GuestCategories == nil {
			t.Errorf("%s: the registry lists are nil", et.Key)
		}
	}
	var _ pd.GuestRSVPWriter = svc.RSVPWriter()
}
