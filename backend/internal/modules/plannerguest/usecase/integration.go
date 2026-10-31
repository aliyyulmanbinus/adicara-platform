package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerguest/domain"
)

// The module sits on two of menu 01's plug-in points and offers one interface to the invitation module. Each key of
// the host belongs to one role, so every role is a small type that wraps the Service (as in menus 02-04). Register
// them in internal/app/app.go; the module owns no wizard template (the wizard creates no guests):
//
//	provider   "guests"  the dashboard widget
//	overlay    "guests"  the 20 sample guests
//	RSVPWriter           pd.GuestRSVPWriter for the invitation module (not registered anywhere yet)

// ---------- dashboard widget ----------

type provider struct{ s *Service }

func (provider) Key() string { return "guests" }

// Summarize is the guests widget (01 BE 5.2). It is built from the same overview as the endpoints, so the dashboard and
// the page cannot show different numbers. Sides is empty for an event type without sides.
func (p provider) Summarize(ctx context.Context, project pd.Project) (any, error) {
	o, err := p.s.overview(ctx, project)
	if err != nil {
		return nil, err
	}

	return pd.GuestSummary{Invitations: o.Total, TotalPax: o.TotalPax, Sides: o.Sides, Sent: o.Invitation.Sent, RSVP: o.RSVP}, nil
}

func (s *Service) Provider() pd.SummaryProvider { return provider{s} }

// ---------- sample project (Wedding_Planner.pdf) ----------

type guestsOverlay struct{ s *Service }

func (guestsOverlay) Key() string { return "guests" }

// ApplySample writes the 20 guests of 05 BE 6 in table order (sort_order = position). It depends on no other overlay.
// Attending guests get pax_actual = pax_count; an invitation that was sent and an answer that was given
// are stamped with the creation time of the project.
func (o guestsOverlay) ApplySample(ctx context.Context, p pd.Project) error {
	s := o.s
	now := s.now()
	guests := make([]domain.Guest, 0, len(domain.SampleGuests))
	for _, sg := range domain.SampleGuests {
		phone := sg.Phone
		wa, err := pd.NormalizeWA(phone)
		if err != nil {
			return fmt.Errorf("sample guest %q: %w", sg.Name, err)
		}
		g, err := newGuest(guestFields{name: sg.Name, side: &sg.Side, category: sg.Category, scope: sg.Scope, table: &sg.Table, pax: sg.Pax, phone: &phone, wa: &wa})
		if err != nil {
			return err
		}
		if sg.Sent {
			g.InvitationSentStatus, g.InvitationSentAt = domain.SentSent, &now
		}
		if sg.RSVP != domain.RSVPUnconfirmed {
			g.RSVPStatus, g.RSVPSubmittedAt = sg.RSVP, &now
		}
		if sg.RSVP == domain.RSVPAttending {
			g.PaxActual = sg.Pax
		}
		guests = append(guests, g)
	}

	return s.repo.InsertGuests(ctx, p.ID, p.InvitationID, guests)
}

func (s *Service) GuestsOverlay() pd.SampleOverlay { return guestsOverlay{s} }

// ---------- invitation module ----------

type rsvpWriter struct{ s *Service }

// RSVPWriter is the interface the invitation module calls when a guest answers on the public page (05 BE 5.1). The
// guest is found by invitation and ticket token, never by a project: the module does not know the planner's ids.
func (s *Service) RSVPWriter() pd.GuestRSVPWriter { return rsvpWriter{s} }

// ApplyRSVP is idempotent. A guest who has checked in is left alone and the call succeeds. pd.ErrNotFound when no
// guest has that token and invitation (the caller may errors.Is it); a bad status or pax is an apierr 400.
func (w rsvpWriter) ApplyRSVP(ctx context.Context, invitationID uuid.UUID, qrToken string, status pd.RSVPStatus, pax int) error {
	stored, err := checkRSVP(status, pax)
	if err != nil {
		return err
	}

	return w.s.repo.ApplyRSVPByToken(ctx, invitationID, qrToken, status, stored)
}
