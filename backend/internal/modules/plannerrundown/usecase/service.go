// Package usecase implements planner menu 06: the rundown (agendas minute by minute per session, with the live status
// of Hari-H) and the playlist (songs tied to the agendas), with the WhatsApp texts built on the server. It never imports
// pgx; transactions travel in the context (pd.TxManager). The module plugs into menu 01 through the provider and the two
// appliers in integration.go.
package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerrundown/domain"
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
		return errUnknownSession()
	case errors.Is(err, domain.ErrInvalidVendor):
		return invalidInput("vendor_id", reasonVendorOfProject)
	case errors.Is(err, domain.ErrInvalidAgenda):
		return invalidInput("rundown_id", reasonAgendaOfProject)
	case errors.Is(err, domain.ErrConflict):
		return apierr.Conflict("conflict", "the project was changed concurrently, try again")
	}

	return err
}

func errUnknownSession() *apierr.Error {
	return apierr.BadRequest("invalid_scope", "session_id: must be a session of this project")
}

func notice(code, message string, count int) pd.Notice {
	return pd.Notice{Code: code, Message: message, Count: count}
}

func invalidTemplate(field string) *apierr.Error {
	return apierr.BadRequest("invalid_template", field+": not available for this event type")
}

func confirmRequired() *apierr.Error { return invalidInput("confirm", "required") }

// session finds a session of the project that agendas may belong to: not the system session `all`.
func (s *Service) session(ctx context.Context, projectID, id uuid.UUID) (domain.SessionRef, error) {
	if id == uuid.Nil {
		return domain.SessionRef{}, invalidInput("session_id", "required")
	}
	sessions, err := s.repo.Sessions(ctx, projectID)
	if err != nil {
		return domain.SessionRef{}, err
	}
	for _, sess := range sessions {
		if sess.ID != id {
			continue
		}
		if sess.IsSystem {
			return domain.SessionRef{}, invalidInput("session_id", "must not be the system session")
		}

		return sess, nil
	}

	return domain.SessionRef{}, errUnknownSession()
}

// parseSessionQuery reads the ?session_id= of the list and the export; anything that is not the id of a session of the
// project is the same invalid_scope as in a body.
func parseSessionQuery(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil || id.String() != raw {
		return uuid.Nil, errUnknownSession()
	}

	return id, nil
}

// sameIDs reports whether got is exactly want: the same ids, each once, in any order.
func sameIDs(want, got []uuid.UUID) bool {
	if len(want) != len(got) {
		return false
	}
	seen := make(map[uuid.UUID]bool, len(want))
	for _, id := range want {
		seen[id] = true
	}
	for _, id := range got {
		if !seen[id] {
			return false
		}
		delete(seen, id) // a repeated id finds nothing the second time
	}

	return true
}
