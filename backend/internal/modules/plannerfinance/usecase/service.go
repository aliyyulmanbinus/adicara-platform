// Package usecase implements planner menu 03: the budget simulation
// (scenarios), the savings and the realisation of the budget lines. It never
// imports pgx; transactions travel in the context (pd.TxManager). The module
// plugs into menu 01 through the provider, applier and overlay in integration.go.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerfinance/domain"
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

// today is the calendar date in the project's zone. A zone that cannot be
// loaded is a server problem (500).
func (s *Service) today(p pd.Project) (time.Time, error) {
	loc, err := pd.LoadLocation(p.Timezone)
	if err != nil {
		return time.Time{}, fmt.Errorf("project %s: %w", p.ID, err)
	}

	return pd.CalendarDay(s.now(), loc), nil
}

// mapErr turns a repository sentinel into the API error; unknown errors pass
// through and become 500. resource names what a 404 refers to.
func mapErr(err error, resource string) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domain.ErrNotFound):
		return apierr.NotFound(resource)
	case errors.Is(err, domain.ErrInvalidScope):
		return errUnknownScope()
	case errors.Is(err, domain.ErrInvalidVendor):
		return invalidInput("vendor_id", "must be a vendor of this project")
	case errors.Is(err, domain.ErrVendorAlreadyLinked):
		return apierr.Conflict("vendor_already_linked", "vendor_id: already linked to another budget line")
	case errors.Is(err, domain.ErrKeyTaken):
		return apierr.Conflict("conflict", "item_key: already used in this session")
	case errors.Is(err, domain.ErrConflict):
		return apierr.Conflict("conflict", "the project was changed concurrently, try again")
	}

	return err
}

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

// checkScope rejects a scope that is not a session of the project. The foreign
// key is deferred, so without this the error would only surface at COMMIT.
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
