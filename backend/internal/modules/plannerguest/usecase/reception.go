package usecase

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerguest/domain"
)

// ReceptionOverview is the one packet the reception tab polls every 5 seconds (05 BE 2.5): the KPI, the filter counts,
// the newest check-ins and one page of the queue.
func (s *Service) ReceptionOverview(ctx context.Context, p pd.Project, f ReceptionFilter) (domain.ReceptionOverview, error) {
	q, err := f.parse()
	if err != nil {
		return domain.ReceptionOverview{}, err
	}
	link, err := s.links(ctx, p)
	if err != nil {
		return domain.ReceptionOverview{}, err
	}
	counts, err := s.repo.ReceptionCounts(ctx, p.ID)
	if err != nil {
		return domain.ReceptionOverview{}, err
	}
	kpi := counts.KPI
	stock, found, err := s.repo.SouvenirStock(ctx, p.ID)
	if err != nil {
		return domain.ReceptionOverview{}, err
	}
	if found {
		kpi.SouvenirStock = &stock
	}
	recent, err := s.repo.RecentCheckins(ctx, p.ID, recentCheckins)
	if err != nil {
		return domain.ReceptionOverview{}, err
	}
	guests, total, err := s.repo.ReceptionQueue(ctx, p.ID, q)
	if err != nil {
		return domain.ReceptionOverview{}, err
	}
	link.all(recent)
	link.all(guests)
	spec, _ := s.spec(p)

	return domain.ReceptionOverview{
		KPI:            kpi,
		Filters:        spec.ReceptionFilters(kpi.ExpectedGuests, kpi.CheckedInGuests, kpi.PendingGuests, counts.ByCategory),
		RecentCheckins: recent,
		Guests:         guests,
		Total:          total,
		Limit:          q.Limit,
		Offset:         q.Offset,
	}, nil
}

// maxTokenLen bounds what is looked up; a longer string cannot be a token.
const maxTokenLen = 64

// ScanQR finds the guest of a ticket in this project. A token that is malformed, unknown, or belongs to another project
// is the same 404; a guest who has checked in is returned like any other (the screen shows the entry time).
func (s *Service) ScanQR(ctx context.Context, p pd.Project, req domain.ScanQRRequest) (domain.Guest, error) {
	token := strings.TrimSpace(req.QRCodeToken)
	if token == "" || utf8.RuneCountInString(token) > maxTokenLen {
		return domain.Guest{}, mapErr(domain.ErrNotFound)
	}
	g, err := s.repo.GuestByToken(ctx, p.ID, token)
	if err != nil {
		return domain.Guest{}, mapErr(err)
	}

	return s.withLink(ctx, p, g)
}

// Checkin is one atomic UPDATE: of two simultaneous taps one wins and the other gets 409 already_checked_in. It does
// not change the RSVP: a guest who declined and came is counted present at the desk, and stays "declined" on the list.
func (s *Service) Checkin(ctx context.Context, p pd.Project, id uuid.UUID, req domain.CheckinRequest) (domain.Guest, error) {
	if err := cleanPax("pax_actual", req.PaxActual, minPax); err != nil {
		return domain.Guest{}, err
	}
	g, err := s.repo.Checkin(ctx, p.ID, id, req.PaxActual, req.SouvenirReceived)
	if err != nil {
		return domain.Guest{}, mapErr(err)
	}

	return s.withLink(ctx, p, g)
}

// CancelCheckin undoes a check-in and the souvenir mark; pax_actual stays. It is idempotent.
func (s *Service) CancelCheckin(ctx context.Context, p pd.Project, id uuid.UUID) (domain.Guest, error) {
	g, err := s.repo.CancelCheckin(ctx, p.ID, id)
	if err != nil {
		return domain.Guest{}, mapErr(err)
	}

	return s.withLink(ctx, p, g)
}

// SetSouvenir marks whether the souvenir was handed over: one mark per invitation, however many people came (05 BE 10.2).
func (s *Service) SetSouvenir(ctx context.Context, p pd.Project, id uuid.UUID, req domain.SouvenirRequest) (domain.Guest, error) {
	g, err := s.repo.SetSouvenir(ctx, p.ID, id, req.SouvenirReceived)
	if err != nil {
		return domain.Guest{}, mapErr(err)
	}

	return s.withLink(ctx, p, g)
}

// WalkIn adds a guest who was not invited: attending, checked in now, pax_count = pax_actual, session "all".
func (s *Service) WalkIn(ctx context.Context, p pd.Project, req domain.WalkInRequest) (domain.Guest, error) {
	spec, _ := s.spec(p)
	var out domain.Guest
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		f, err := checkGuest(spec, map[string]bool{pd.AllScope: true}, domain.GuestRequest{
			Name: req.Name, GuestSide: req.GuestSide, Category: req.Category, TableNumber: req.TableNumber,
			PaxCount: req.PaxActual, Notes: req.Notes,
		}, "pax_actual")
		if err != nil {
			return err
		}
		g, err := newGuest(f)
		if err != nil {
			return err
		}
		now := s.now()
		g.RSVPStatus, g.PaxActual, g.RSVPSubmittedAt = domain.RSVPAttending, f.pax, &now
		g.IsCheckedIn, g.CheckedInAt, g.SouvenirReceived = true, &now, req.SouvenirReceived
		if err := s.reserve(ctx, p.ID, 1); err != nil {
			return err
		}
		if err := s.repo.InsertGuest(ctx, p.ID, p.InvitationID, &g); err != nil {
			return mapErr(err)
		}
		out = g

		return nil
	})
	if err != nil {
		return domain.Guest{}, mapErr(err)
	}

	return s.withLink(ctx, p, out)
}
