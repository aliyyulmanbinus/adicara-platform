package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

const scopeKeyRetries = 3

var errSystemSession = invalidInput("session", "the system session cannot be changed or deleted")

// ListSessions returns every session including the system one, with usage.
func (s *Service) ListSessions(ctx context.Context, projectID uuid.UUID) ([]domain.SessionWithUsage, error) {
	return s.repo.ListSessionsWithUsage(ctx, projectID)
}

// CreateSession adds a session. The project row is locked so the 20-session
// limit and the sort order cannot race with another create or delete.
func (s *Service) CreateSession(ctx context.Context, projectID uuid.UUID, req domain.CreateSessionRequest) (domain.Session, error) {
	f, err := validateSessionFields(req)
	if err != nil {
		return domain.Session{}, err
	}

	var created domain.Session
	for attempt := 0; ; attempt++ {
		err = s.tx.InTx(ctx, func(ctx context.Context) error {
			if _, err := s.repo.LockProject(ctx, projectID); err != nil {
				return err
			}
			existing, err := s.repo.ListSessions(ctx, projectID)
			if err != nil {
				return err
			}

			taken := make(map[string]bool, len(existing))
			custom, maxOrder := 0, 0
			for _, e := range existing {
				taken[e.ScopeKey] = true
				if !e.IsSystem {
					custom++
					maxOrder = max(maxOrder, e.SortOrder)
				}
			}
			if custom >= maxSessions {
				return apierr.Conflict("session_limit_reached", fmt.Sprintf("at most %d sessions per project", maxSessions))
			}

			key, ok := domain.UniqueScopeKey(domain.ScopeKeyFromName(f.name), taken)
			if !ok {
				return apierr.Conflict("session_limit_reached", "no free scope key for this name")
			}
			created = domain.Session{
				ID: uuid.New(), ProjectID: projectID, ScopeKey: key, Name: f.name, Icon: f.icon,
				SortOrder: maxOrder + 1, SessionDate: f.date, StartTime: f.startTime, EndTime: f.endTime, LocationName: f.place,
			}

			return s.repo.InsertSession(ctx, &created)
		})

		// A scope_key race can only happen across processes, since the project
		// row is locked; retry with a fresh view.
		if errors.Is(err, domain.ErrScopeKeyTaken) && attempt < scopeKeyRetries-1 {
			continue
		}

		break
	}
	if err != nil {
		return domain.Session{}, mapSessionErr(err)
	}

	return created, nil
}

func mapSessionErr(err error) error {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return apierr.NotFound("session")
	case errors.Is(err, domain.ErrSessionNameTaken):
		return apierr.Conflict("session_name_taken", "a session with this name already exists in the project")
	case errors.Is(err, domain.ErrScopeKeyTaken):
		return apierr.Conflict("conflict", "the sessions were changed concurrently, try again")
	}

	return mapRepoErr(err)
}

// UpdateSession replaces name, icon, date, times and location. scope_key never changes.
func (s *Service) UpdateSession(ctx context.Context, projectID, sessionID uuid.UUID, req domain.UpdateSessionRequest) (domain.Session, error) {
	f, err := validateSessionFields(req)
	if err != nil {
		return domain.Session{}, err
	}

	sess, err := s.repo.SessionByID(ctx, projectID, sessionID)
	if err != nil {
		return domain.Session{}, mapSessionErr(err)
	}
	if sess.IsSystem {
		return domain.Session{}, errSystemSession
	}

	sess.Name, sess.Icon, sess.SessionDate = f.name, f.icon, f.date
	sess.StartTime, sess.EndTime, sess.LocationName = f.startTime, f.endTime, f.place
	if err := s.repo.UpdateSession(ctx, &sess); err != nil {
		return domain.Session{}, mapSessionErr(err)
	}

	return sess, nil
}

// ReorderSessions sets sort_order = 1..n. The list must name every non-system
// session of the project exactly once.
func (s *Service) ReorderSessions(ctx context.Context, projectID uuid.UUID, req domain.ReorderSessionsRequest) ([]domain.Session, error) {
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		if _, err := s.repo.LockProject(ctx, projectID); err != nil {
			return err
		}
		existing, err := s.repo.ListSessions(ctx, projectID)
		if err != nil {
			return err
		}

		want := map[uuid.UUID]bool{}
		for _, e := range existing {
			if !e.IsSystem {
				want[e.ID] = true
			}
		}
		if len(req.SessionIDs) != len(want) {
			return invalidInput("session_ids", "must list every session of the project exactly once")
		}
		seen := map[uuid.UUID]bool{}
		for _, id := range req.SessionIDs {
			if !want[id] || seen[id] {
				return invalidInput("session_ids", "must list every session of the project exactly once")
			}
			seen[id] = true
		}

		return s.repo.ReorderSessions(ctx, projectID, req.SessionIDs)
	})
	if err != nil {
		return nil, mapRepoErr(err)
	}

	return s.repo.ListSessions(ctx, projectID)
}

// DeleteSession moves everything that points at the session to moveTo (default
// 'all') and deletes it. A session that still has rundown agendas needs
// deleteAgendas, otherwise 409 session_in_use with the usage counts.
func (s *Service) DeleteSession(ctx context.Context, projectID, sessionID uuid.UUID, moveTo string, deleteAgendas bool) error {
	if moveTo == "" {
		moveTo = domain.AllScope
	}

	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		// Serialises with every create/delete of sessions and of rows that
		// point at one.
		if _, err := s.repo.LockProject(ctx, projectID); err != nil {
			return err
		}

		sess, err := s.repo.SessionByID(ctx, projectID, sessionID)
		if err != nil {
			return err
		}
		if sess.IsSystem {
			return errSystemSession
		}
		if moveTo == sess.ScopeKey {
			return apierr.BadRequest("invalid_scope", "move_to: must be a different session")
		}
		sessions, err := s.repo.ListSessions(ctx, projectID)
		if err != nil {
			return err
		}
		found := false
		for _, e := range sessions {
			found = found || e.ScopeKey == moveTo
		}
		if !found {
			return apierr.BadRequest("invalid_scope", "move_to: unknown session")
		}

		usage, err := s.repo.SessionUsage(ctx, projectID, sess)
		if err != nil {
			return err
		}
		if usage.Agendas > 0 && !deleteAgendas {
			return apierr.Conflict("session_in_use", "the session still has rundown agendas").
				WithDetails(map[string]any{"usage": usage})
		}

		return s.repo.MoveSessionData(ctx, projectID, sess.ScopeKey, moveTo)
	})

	return mapSessionErr(err)
}
