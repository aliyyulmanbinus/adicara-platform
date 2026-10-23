// Package usecase implements planner menu 04: the vendor pipeline (with the limit of each category read from the
// active scenario and the lock-deal that ties a deal to the budget) and the shopping list (with its rollup into a
// budget line). It never imports pgx; transactions travel in the context (pd.TxManager). The module plugs into menu 01
// through the provider, applier and overlays in integration.go.
package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannervendor/domain"
)

type Options struct {
	// Now is the clock; nil means time.Now. Tests inject a fixed one.
	Now func() time.Time
}

type Service struct {
	repo     domain.Repository
	tx       pd.TxManager
	registry *pd.Registry
	now      func() time.Time
}

func New(repo domain.Repository, tx pd.TxManager, registry *pd.Registry, opts Options) *Service {
	s := &Service{repo: repo, tx: tx, registry: registry, now: opts.Now}
	if s.now == nil {
		s.now = time.Now
	}

	return s
}

// today is the calendar date in the project's zone. A zone that cannot be loaded is a server problem (500).
func (s *Service) today(p pd.Project) (time.Time, error) {
	loc, err := pd.LoadLocation(p.Timezone)
	if err != nil {
		return time.Time{}, err
	}

	return pd.CalendarDay(s.now(), loc), nil
}

// mapErr turns a repository sentinel into the API error; unknown errors pass through and become 500.
// resource names what a 404 refers to.
func mapErr(err error, resource string) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domain.ErrNotFound):
		return apierr.NotFound(resource)
	case errors.Is(err, domain.ErrInvalidScope):
		return errUnknownScope()
	case errors.Is(err, domain.ErrInvalidVendor):
		return invalidInput("vendor_id", reasonVendorOfProject)
	case errors.Is(err, domain.ErrInvalidBudgetItem):
		return invalidInput("budget_item_id", reasonBudgetLineOfProject)
	case errors.Is(err, domain.ErrConflict):
		return apierr.Conflict("conflict", "the project was changed concurrently, try again")
	}

	return err
}

func errUnknownScope() *apierr.Error {
	return apierr.BadRequest("invalid_scope", "sub_event_scope: unknown session")
}

func notice(code, message string, count int) pd.Notice {
	return pd.Notice{Code: code, Message: message, Count: count}
}

func invalidTemplate(field string) *apierr.Error {
	return apierr.BadRequest("invalid_template", field+": not available for this event type")
}

func confirmRequired() *apierr.Error { return invalidInput("confirm", "required") }

// scopeSet loads the scope keys of the project's sessions.
func (s *Service) scopeSet(ctx context.Context, projectID uuid.UUID) (map[string]bool, []domain.SessionRef, error) {
	sessions, err := s.repo.Sessions(ctx, projectID)
	if err != nil {
		return nil, nil, err
	}
	set := make(map[string]bool, len(sessions))
	for _, sess := range sessions {
		set[sess.ScopeKey] = true
	}

	return set, sessions, nil
}

// checkScope rejects a scope that is not a session of the project. The foreign key is deferred, so without this the
// error would only surface at COMMIT.
func (s *Service) checkScope(ctx context.Context, projectID uuid.UUID, scope string) error {
	set, _, err := s.scopeSet(ctx, projectID)
	if err != nil {
		return err
	}
	if !set[scope] {
		return errUnknownScope()
	}

	return nil
}

// specs is the vendor checklist of the event type of p.
func (s *Service) specs(p pd.Project) ([]domain.CategorySpec, pd.EventTypeSpec) {
	et, _ := s.registry.Get(p.EventType)
	out := make([]domain.CategorySpec, len(et.VendorCategories))
	for i, c := range et.VendorCategories {
		out[i] = domain.CategorySpec{Key: c.Key, Label: c.Label, BudgetItemKeys: c.BudgetItemKeys}
	}

	return out, et
}

func isNotFound(err error) bool { return errors.Is(err, domain.ErrNotFound) }
