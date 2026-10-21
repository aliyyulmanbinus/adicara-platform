// Package usecase implements menu 01 of the planner: projects, sessions, the
// event type registry and the dashboard. It never imports pgx; transactions
// travel in the context (domain.TxManager) and the other planner modules plug
// in through SummaryProvider, TemplateApplier and SampleOverlay.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

// DefaultMaxProjects is PLANNER_MAX_PROJECTS when it is not set.
const DefaultMaxProjects = 100

type Options struct {
	// MaxProjects caps a user's real (non-sample) projects; <= 0 means DefaultMaxProjects.
	MaxProjects int
	// Now is the clock; nil means time.Now. Tests inject a fixed one.
	Now func() time.Time
	// DashboardTimeout bounds the whole dashboard fan-out; <= 0 means 3s.
	DashboardTimeout time.Duration
}

type Service struct {
	repo        domain.Repository
	tx          domain.TxManager
	registry    *domain.Registry
	maxProjects int
	now         func() time.Time
	dashTimeout time.Duration

	// Filled during wiring, before the server starts; read-only afterwards.
	appliers  map[string]domain.TemplateApplier
	providers map[string]domain.SummaryProvider
	overlays  []domain.SampleOverlay
}

func New(repo domain.Repository, tx domain.TxManager, registry *domain.Registry, opts Options) *Service {
	s := &Service{
		repo: repo, tx: tx, registry: registry,
		maxProjects: opts.MaxProjects, now: opts.Now, dashTimeout: opts.DashboardTimeout,
		appliers:  map[string]domain.TemplateApplier{},
		providers: map[string]domain.SummaryProvider{},
	}
	if s.maxProjects <= 0 {
		s.maxProjects = DefaultMaxProjects
	}
	if s.now == nil {
		s.now = time.Now
	}
	if s.dashTimeout <= 0 {
		s.dashTimeout = defaultDashboardTimeout
	}

	return s
}

func contains(keys []string, key string) bool {
	for _, k := range keys {
		if k == key {
			return true
		}
	}

	return false
}

// RegisterSummary adds the dashboard providers of the other planner modules.
// It panics on an unknown or duplicate key: that is a wiring bug and must stop
// the process at start, not surface as an empty widget later.
func (s *Service) RegisterSummary(providers ...domain.SummaryProvider) {
	for _, p := range providers {
		if !contains(domain.SummaryKeys, p.Key()) {
			panic(fmt.Sprintf("planner: unknown summary provider key %q", p.Key()))
		}
		if _, dup := s.providers[p.Key()]; dup {
			panic(fmt.Sprintf("planner: summary provider %q registered twice", p.Key()))
		}
		s.providers[p.Key()] = p
	}
}

// RegisterTemplateAppliers adds the template appliers; same panic rule.
func (s *Service) RegisterTemplateAppliers(appliers ...domain.TemplateApplier) {
	for _, a := range appliers {
		if !contains(domain.TemplateKeys, a.Key()) {
			panic(fmt.Sprintf("planner: unknown template applier key %q", a.Key()))
		}
		if _, dup := s.appliers[a.Key()]; dup {
			panic(fmt.Sprintf("planner: template applier %q registered twice", a.Key()))
		}
		s.appliers[a.Key()] = a
	}
}

// RegisterSampleOverlays adds the dataset overlays of the sample project.
func (s *Service) RegisterSampleOverlays(overlays ...domain.SampleOverlay) {
	s.overlays = append(s.overlays, overlays...)
}

func (s *Service) hasApplier(key string) bool {
	_, ok := s.appliers[key]

	return ok
}

// EventTypeView is one entry of GET /event-types.
type EventTypeView struct {
	domain.EventTypeSpec
	TemplatesAvailable domain.TemplatesAvailable `json:"templates_available"`
}

// EventTypes returns the registry with the templates that can be requested
// right now (owned by the type AND having a registered applier).
func (s *Service) EventTypes() []EventTypeView {
	all := s.registry.All()
	out := make([]EventTypeView, len(all))
	for i, spec := range all {
		out[i] = EventTypeView{EventTypeSpec: spec, TemplatesAvailable: spec.Templates.Available(s.hasApplier)}
	}

	return out
}

// ---------- project access ----------

// ProjectForUser backs ProjectGuard: unknown and foreign projects are both 404.
func (s *Service) ProjectForUser(ctx context.Context, id, userID uuid.UUID) (domain.Project, error) {
	p, err := s.repo.ProjectForUser(ctx, id, userID)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.Project{}, apierr.NotFound("project")
	}

	return p, err
}

// Phase computes the phase of p now, in its own time zone.
func (s *Service) Phase(p domain.Project) (domain.PhaseInfo, error) {
	loc, err := domain.LoadLocation(p.Timezone)
	if err != nil {
		return domain.PhaseInfo{}, fmt.Errorf("project %s: %w", p.ID, err)
	}

	return domain.ComputePhase(s.now(), loc, p.EventDate, p.EventEndDate), nil
}

// mapRepoErr turns a repository sentinel into the API error every endpoint
// answers with; unknown errors pass through and become 500.
func mapRepoErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domain.ErrNotFound):
		return apierr.NotFound("resource")
	case errors.Is(err, domain.ErrConflict):
		return apierr.Conflict("conflict", "the project was changed concurrently, try again")
	case errors.Is(err, domain.ErrInvalidTimezone):
		return apierr.BadRequest("invalid_timezone", "timezone: unknown time zone")
	case errors.Is(err, domain.ErrInvalidScope):
		return apierr.BadRequest("invalid_scope", "scope: unknown session")
	}

	return err
}
