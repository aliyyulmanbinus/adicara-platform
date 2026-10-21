// Package usecase implements planner menu 02: the task timeline and the
// administrative documents. It never imports pgx; transactions travel in the
// context (pd.TxManager). The module plugs into menu 01 through the providers,
// appliers and overlays in integration.go.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannertimeline/domain"
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

// clock returns the project's zone and "now" for the read queries, and the
// loaded location. A zone that cannot be loaded is a server problem (500).
func (s *Service) clock(p pd.Project) (domain.At, *time.Location, error) {
	loc, err := pd.LoadLocation(p.Timezone)
	if err != nil {
		return domain.At{}, nil, fmt.Errorf("project %s: %w", p.ID, err)
	}

	return domain.At{Zone: p.Timezone, Now: s.now()}, loc, nil
}

// today is the calendar date in the project's zone.
func (s *Service) today(p pd.Project) (time.Time, error) {
	_, loc, err := s.clock(p)
	if err != nil {
		return time.Time{}, err
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

func errUnknownScope() *apierr.Error {
	return apierr.BadRequest("invalid_scope", "sub_event_scope: unknown session")
}

func notice(code, message string, count int) pd.Notice {
	return pd.Notice{Code: code, Message: message, Count: count}
}
