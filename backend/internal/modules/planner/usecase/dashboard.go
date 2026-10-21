package usecase

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

const (
	// The pool is small and shared with BearerAuth (one connection per
	// request), so at most this many providers query at once.
	dashboardParallelism    = 3
	defaultDashboardTimeout = 3 * time.Second
)

// Dashboard assembles the workspace dashboard of p. Each registered provider
// fills one widget; a provider that fails, panics or is too slow only affects
// its own widget (it becomes {"error": "internal" | "timeout"}).
func (s *Service) Dashboard(ctx context.Context, p domain.Project) (domain.WorkspaceDashboard, error) {
	phase, err := s.Phase(p)
	if err != nil {
		return domain.WorkspaceDashboard{}, err
	}
	sessions, err := s.repo.ListSessions(ctx, p.ID)
	if err != nil {
		return domain.WorkspaceDashboard{}, err
	}

	briefs := make([]domain.SessionBrief, 0, len(sessions))
	for _, sess := range sessions {
		if !sess.IsSystem {
			briefs = append(briefs, sess.Brief())
		}
	}

	d := domain.WorkspaceDashboard{
		ServerTime: s.now().UTC(),
		Project: domain.DashboardProject{
			ID: p.ID, EventType: p.EventType, Title: p.Title, EventDate: p.EventDate.Format(domain.DateLayout),
			PhaseInfo: phase, Status: p.Status, IsSample: p.IsSample, InvitationLinked: p.InvitationID != nil,
			TargetBudget: p.TargetBudget, SavingsTarget: p.SavingsTarget, Timezone: p.Timezone, Metadata: p.Metadata,
		},
		Sessions: briefs,
	}
	if p.EventEndDate != nil {
		end := p.EventEndDate.Format(domain.DateLayout)
		d.Project.EventEndDate = &end
	}

	widgets := s.runProviders(ctx, p)
	d.Timeline, d.Finance, d.Vendors = widgets["timeline"], widgets["finance"], widgets["vendors"]
	d.Guests, d.AdminDocs, d.Rundown = widgets["guests"], widgets["admin_docs"], widgets["rundown"]

	return d, nil
}

// runProviders returns one value per registered provider key: its summary or a
// domain.WidgetError. Keys without a provider are absent (rendered as null).
func (s *Service) runProviders(ctx context.Context, p domain.Project) map[string]any {
	out := make(map[string]any, len(s.providers))
	if len(s.providers) == 0 {
		return out
	}

	ctx, cancel := context.WithTimeout(ctx, s.dashTimeout)
	defer cancel()

	var mu sync.Mutex
	var g errgroup.Group
	g.SetLimit(dashboardParallelism)
	for key, provider := range s.providers {
		g.Go(func() error {
			value := s.runProvider(ctx, key, provider, p)
			mu.Lock()
			out[key] = value
			mu.Unlock()

			return nil
		})
	}
	_ = g.Wait()

	return out
}

// runProvider shields the process: gin.Recovery only protects the handler
// goroutine, so a panic in a goroutine of ours would kill the whole API.
func (s *Service) runProvider(ctx context.Context, key string, provider domain.SummaryProvider, p domain.Project) any {
	// The provider runs in its own goroutine, with its own recover, so that
	// one that ignores ctx cannot hold the response past the deadline either.
	type result struct {
		v   any
		err error
	}
	done := make(chan result, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("planner dashboard: provider %s panicked: %v", key, r)
				done <- result{err: errors.New("panic")}
			}
		}()
		v, err := provider.Summarize(ctx, p)
		done <- result{v: v, err: err}
	}()

	select {
	case r := <-done:
		if r.err != nil {
			if errors.Is(r.err, context.DeadlineExceeded) || ctx.Err() == context.DeadlineExceeded {
				return domain.WidgetError{Error: "timeout"}
			}
			log.Printf("planner dashboard: provider %s failed: %v", key, r.err)

			return domain.WidgetError{Error: "internal"}
		}

		return r.v
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return domain.WidgetError{Error: "timeout"}
		}

		return domain.WidgetError{Error: "internal"} // the client went away; nobody reads this
	}
}
