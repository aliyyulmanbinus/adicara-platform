package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerguest/domain"
)

// ---------- list and summary ----------

// overview computes the figures of the whole project (05 BE 2.8). One function serves the dashboard provider, the
// summary endpoint and the list, so the numbers cannot differ.
func (s *Service) overview(ctx context.Context, p pd.Project) (domain.GuestOverview, error) {
	a, err := s.repo.Aggregates(ctx, p.ID)
	if err != nil {
		return domain.GuestOverview{}, err
	}
	spec, _ := s.spec(p)

	return spec.AssembleOverview(a), nil
}

func (s *Service) Summary(ctx context.Context, p pd.Project) (domain.GuestOverview, error) {
	return s.overview(ctx, p)
}

func (s *Service) ListGuests(ctx context.Context, p pd.Project, f GuestFilter) (domain.GuestList, error) {
	q, err := f.parse()
	if err != nil {
		return domain.GuestList{}, err
	}
	link, err := s.links(ctx, p)
	if err != nil {
		return domain.GuestList{}, err
	}
	guests, total, err := s.repo.ListGuests(ctx, p.ID, q)
	if err != nil {
		return domain.GuestList{}, err
	}
	ov, err := s.overview(ctx, p)
	if err != nil {
		return domain.GuestList{}, err
	}
	link.all(guests)

	return domain.GuestList{Overview: ov, Guests: guests, Total: total, Limit: q.Limit, Offset: q.Offset}, nil
}

func (s *Service) GetGuest(ctx context.Context, p pd.Project, id uuid.UUID) (domain.Guest, error) {
	g, err := s.repo.GuestByID(ctx, p.ID, id)
	if err != nil {
		return domain.Guest{}, mapErr(err)
	}

	return s.withLink(ctx, p, g)
}

// withLink adds the personal link to one guest.
func (s *Service) withLink(ctx context.Context, p pd.Project, g domain.Guest) (domain.Guest, error) {
	link, err := s.links(ctx, p)
	if err != nil {
		return domain.Guest{}, err
	}
	link(&g)

	return g, nil
}

// ---------- create, update, delete ----------

// newGuest is a guest nobody has answered or visited yet.
func newGuest(f guestFields) (domain.Guest, error) {
	token, err := newToken()
	if err != nil {
		return domain.Guest{}, err
	}

	return domain.Guest{
		ID: uuid.New(), Name: f.name, GuestSide: f.side, Category: f.category, SubEventScope: f.scope, TableNumber: f.table,
		PaxCount: f.pax, PhoneNumber: f.phone, FormattedWANumber: f.wa, Notes: f.notes, QRCodeToken: token,
		InvitationSentStatus: domain.SentUnsent, RSVPStatus: domain.RSVPUnconfirmed,
	}, nil
}

// reserve checks the guest limit for n new guests under the project lock; the caller holds the transaction.
func (s *Service) reserve(ctx context.Context, projectID uuid.UUID, n int) error {
	if err := s.repo.LockProject(ctx, projectID); err != nil {
		return mapErr(err)
	}
	count, err := s.repo.CountGuests(ctx, projectID)
	if err != nil {
		return err
	}
	if count+n > maxGuests {
		return errLimit()
	}

	return nil
}

func (s *Service) CreateGuest(ctx context.Context, p pd.Project, req domain.GuestRequest) (domain.Guest, error) {
	spec, _ := s.spec(p)
	var out domain.Guest
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		scopes, _, err := s.sessionScopes(ctx, p.ID)
		if err != nil {
			return err
		}
		f, err := checkGuest(spec, scopes, req, "pax_count")
		if err != nil {
			return err
		}
		g, err := newGuest(f)
		if err != nil {
			return err
		}
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

// UpdateGuest replaces name, side, category, session, table, pax, phone and notes (a field that is not sent is cleared);
// RSVP, invitation status, check-in, souvenir, token and order stay as they are.
func (s *Service) UpdateGuest(ctx context.Context, p pd.Project, id uuid.UUID, req domain.GuestRequest) (domain.Guest, error) {
	spec, _ := s.spec(p)
	scopes, _, err := s.sessionScopes(ctx, p.ID)
	if err != nil {
		return domain.Guest{}, err
	}
	f, err := checkGuest(spec, scopes, req, "pax_count")
	if err != nil {
		return domain.Guest{}, err
	}
	g := domain.Guest{
		ID: id, Name: f.name, GuestSide: f.side, Category: f.category, SubEventScope: f.scope, TableNumber: f.table,
		PaxCount: f.pax, PhoneNumber: f.phone, FormattedWANumber: f.wa, Notes: f.notes,
	}
	if err := s.repo.UpdateGuest(ctx, p.ID, &g); err != nil {
		return domain.Guest{}, mapErr(err)
	}

	return s.withLink(ctx, p, g)
}

func (s *Service) DeleteGuest(ctx context.Context, p pd.Project, id uuid.UUID) error {
	return mapErr(s.repo.DeleteGuest(ctx, p.ID, id))
}

// ---------- invitation status and RSVP ----------

func (s *Service) SetSentStatus(ctx context.Context, p pd.Project, id uuid.UUID, req domain.SentStatusRequest) (domain.Guest, error) {
	if err := oneOf("sent_status", string(req.SentStatus), "unsent", "sent"); err != nil {
		return domain.Guest{}, err
	}
	g, err := s.repo.SetSentStatus(ctx, p.ID, id, req.SentStatus)
	if err != nil {
		return domain.Guest{}, mapErr(err)
	}

	return s.withLink(ctx, p, g)
}

// checkRSVP validates an answer and returns the pax to store: the head-count only means something for attending
// (0 = follow pax_count); every other status stores 0.
func checkRSVP(status domain.RSVPStatus, pax int) (int, error) {
	if err := oneOf("rsvp_status", string(status), "unconfirmed", "attending", "declined", "maybe"); err != nil {
		return 0, err
	}
	if err := cleanPax("pax_actual", pax, 0); err != nil {
		return 0, err
	}
	if status != domain.RSVPAttending {
		return 0, nil
	}

	return pax, nil
}

// SetRSVP records the answer of a guest by hand (standalone mode). A guest who has checked in cannot be changed:
// pax_actual is already the head-count at the door (409 already_checked_in; cancel the check-in first).
func (s *Service) SetRSVP(ctx context.Context, p pd.Project, id uuid.UUID, req domain.RSVPRequest) (domain.Guest, error) {
	pax, err := checkRSVP(req.RSVPStatus, req.PaxActual)
	if err != nil {
		return domain.Guest{}, err
	}
	g, err := s.repo.SetRSVP(ctx, p.ID, id, req.RSVPStatus, pax)
	if err != nil {
		return domain.Guest{}, mapErr(err)
	}

	return s.withLink(ctx, p, g)
}

// ---------- catering projection ----------

// CateringProjection is the portion advice of 05 BE 2.3: only attending guests count; the buffer is a whole percentage
// plus a number of extra portions.
func (s *Service) CateringProjection(ctx context.Context, p pd.Project, percentRaw, paxRaw string) (domain.CateringProjection, error) {
	percent, bufferPax, err := bufferOf(percentRaw, paxRaw)
	if err != nil {
		return domain.CateringProjection{}, err
	}
	out, err := s.repo.Catering(ctx, p.ID)
	if err != nil {
		return domain.CateringProjection{}, err
	}
	out.BufferPercent, out.BufferPax = percent, bufferPax
	if out.RecommendedPortions, err = domain.RecommendedPortions(out.ConfirmedPax, percent, bufferPax); err != nil {
		return domain.CateringProjection{}, apierr.BadRequest("invalid_input", "buffer: out of range")
	}

	return out, nil
}
