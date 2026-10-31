package usecase

import (
	"context"

	"github.com/google/uuid"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerrundown/domain"
)

// ---------- reads ----------

// sessionsAndKPI is what both overviews carry: the sessions with their span and the headline numbers, always of the
// whole project.
func (s *Service) sessionsAndKPI(ctx context.Context, p pd.Project) ([]domain.RundownSessionSummary, domain.RundownKPI, error) {
	sessions, err := s.repo.SessionSummaries(ctx, p.ID)
	if err != nil {
		return nil, domain.RundownKPI{}, err
	}
	pics, err := s.repo.DistinctPICs(ctx, p.ID)
	if err != nil {
		return nil, domain.RundownKPI{}, err
	}

	return sessions, domain.BuildKPI(sessions, pics), nil
}

// ListRundowns is GET /rundowns (06 BE 2.8): the sessions and the KPI describe the whole project; only the items follow
// ?session_id=. sessionID is the raw query value, empty for none.
func (s *Service) ListRundowns(ctx context.Context, p pd.Project, sessionID string) (domain.RundownOverview, error) {
	var filter *uuid.UUID
	if sessionID != "" {
		id, err := parseSessionQuery(sessionID)
		if err != nil {
			return domain.RundownOverview{}, err
		}
		if _, err := s.session(ctx, p.ID, id); err != nil {
			return domain.RundownOverview{}, err
		}
		filter = &id
	}

	sessions, kpi, err := s.sessionsAndKPI(ctx, p)
	if err != nil {
		return domain.RundownOverview{}, err
	}
	items, err := s.repo.ListAgendas(ctx, p.ID, filter)
	if err != nil {
		return domain.RundownOverview{}, err
	}

	return domain.RundownOverview{Sessions: sessions, KPI: kpi, Items: items}, nil
}

// Live is GET /rundowns/summary: what the Hari-H screen polls every 5 seconds. It carries no agenda text, only the
// status of every agenda, and the server's clock so the client need not trust the device's.
func (s *Service) Live(ctx context.Context, p pd.Project) (domain.RundownLive, error) {
	sessions, kpi, err := s.sessionsAndKPI(ctx, p)
	if err != nil {
		return domain.RundownLive{}, err
	}
	items, err := s.repo.LiveItems(ctx, p.ID)
	if err != nil {
		return domain.RundownLive{}, err
	}

	return domain.RundownLive{KPI: kpi, Sessions: sessions, Items: items, ServerTime: s.now().UTC()}, nil
}

func (s *Service) GetRundown(ctx context.Context, p pd.Project, id uuid.UUID) (domain.Rundown, error) {
	a, err := s.repo.AgendaByID(ctx, p.ID, id)

	return a, mapErr(err, "agenda")
}

// ---------- writes ----------

// agendaWrite validates a POST or PUT body in the order of 06 BE 3.4 and returns the row to store (without the id). The
// checks that need the transaction (the limit) run there.
func (s *Service) agendaWrite(ctx context.Context, p pd.Project, req domain.RundownRequest) (domain.AgendaWrite, error) {
	var w domain.AgendaWrite
	sess, err := s.session(ctx, p.ID, req.SessionID)
	if err != nil {
		return w, err
	}
	w.SessionID = sess.ID

	times, ferr := domain.ValidateTimes(req.StartTime, req.EndTime, req.EndsNextDay)
	if ferr != nil {
		return w, invalidInput(ferr.Field, ferr.Reason)
	}
	w.Times = times

	if w.ActivityName, err = cleanText("activity_name", req.ActivityName, maxActivityRunes, true); err != nil {
		return w, err
	}
	if w.PICName, err = cleanText("pic_name", req.PICName, maxPICRunes, false); err != nil {
		return w, err
	}
	w.VendorID = req.VendorID
	if w.Description, err = cleanNote("description", req.Description, maxNoteRunes); err != nil {
		return w, err
	}
	w.Highlights, err = cleanNote("highlights", req.Highlights, maxNoteRunes)

	return w, err
}

// CreateRundown is POST /rundowns. The agenda goes last in its session; the limit is checked under the project lock.
func (s *Service) CreateRundown(ctx context.Context, p pd.Project, req domain.RundownRequest) (domain.Rundown, error) {
	w, err := s.agendaWrite(ctx, p, req)
	if err != nil {
		return domain.Rundown{}, err
	}
	w.ID = uuid.New()

	var out domain.Rundown
	err = s.tx.InTx(ctx, func(ctx context.Context) error {
		if err := s.repo.LockProject(ctx, p.ID); err != nil {
			return err
		}
		n, err := s.repo.CountAgendas(ctx, p.ID)
		if err != nil {
			return err
		}
		if n >= maxAgendas {
			return agendaLimitReached()
		}
		if err := s.repo.InsertAgenda(ctx, p.ID, &w); err != nil {
			return err
		}
		out, err = s.repo.AgendaByID(ctx, p.ID, w.ID)

		return err
	})
	if err != nil {
		return domain.Rundown{}, mapErr(err, "agenda")
	}

	return out, nil
}

// UpdateRundown is PUT /rundowns/:id: the whole body replaces the agenda, except the status and the template key. An
// agenda moved to another session goes last there, which changes an order and so takes the project lock.
func (s *Service) UpdateRundown(ctx context.Context, p pd.Project, id uuid.UUID, req domain.RundownRequest) (domain.Rundown, error) {
	w, err := s.agendaWrite(ctx, p, req)
	if err != nil {
		return domain.Rundown{}, err
	}
	w.ID = id

	var out domain.Rundown
	err = s.tx.InTx(ctx, func(ctx context.Context) error {
		cur, err := s.repo.AgendaByID(ctx, p.ID, id)
		if err != nil {
			return err
		}
		if cur.SessionID != w.SessionID {
			if err := s.repo.LockProject(ctx, p.ID); err != nil {
				return err
			}
		}
		if err := s.repo.UpdateAgenda(ctx, p.ID, &w); err != nil {
			return err
		}
		out, err = s.repo.AgendaByID(ctx, p.ID, id)

		return err
	})
	if err != nil {
		return domain.Rundown{}, mapErr(err, "agenda")
	}

	return out, nil
}

// SetStatus is PATCH /rundowns/:id/status (06 BE 2.3): one statement writes the status and its two time stamps, so they
// cannot disagree. Every move is allowed; the order pending → live → done is only how the screen presents it.
func (s *Service) SetStatus(ctx context.Context, p pd.Project, id uuid.UUID, req domain.RundownStatusRequest) (domain.Rundown, error) {
	if !req.Status.Valid() {
		return domain.Rundown{}, invalidInput("status", "must be one of pending, in_progress, completed")
	}

	var out domain.Rundown
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetAgendaStatus(ctx, p.ID, id, req.Status); err != nil {
			return err
		}
		var err error
		out, err = s.repo.AgendaByID(ctx, p.ID, id)

		return err
	})
	if err != nil {
		return domain.Rundown{}, mapErr(err, "agenda")
	}

	return out, nil
}

// DeleteRundown is DELETE /rundowns/:id. The songs that pointed at the agenda stay, without a link (the foreign key
// clears rundown_id).
func (s *Service) DeleteRundown(ctx context.Context, p pd.Project, id uuid.UUID) error {
	return mapErr(s.repo.DeleteAgenda(ctx, p.ID, id), "agenda")
}

// ReorderRundowns is POST /rundowns/reorder (06 BE 2.5): ids must be exactly the agendas of the session, each once. The
// check is needed: a repeated id would pass the update (one row takes one of the positions) and an incomplete list would
// leave two agendas with the same sort_order.
func (s *Service) ReorderRundowns(ctx context.Context, p pd.Project, req domain.ReorderRundownsRequest) error {
	sess, err := s.session(ctx, p.ID, req.SessionID)
	if err != nil {
		return err
	}

	err = s.tx.InTx(ctx, func(ctx context.Context) error {
		if err := s.repo.LockProject(ctx, p.ID); err != nil {
			return err
		}
		have, err := s.repo.AgendaIDs(ctx, p.ID, sess.ID)
		if err != nil {
			return err
		}
		if !sameIDs(have, req.IDs) {
			return invalidInput("ids", "must list every agenda of the session exactly once")
		}

		return s.repo.ReorderAgendas(ctx, p.ID, sess.ID, req.IDs)
	})

	return mapErr(err, "agenda")
}
