// Package usecase implements planner menu 05: the guest list with RSVP and invitation status, the reception desk of the
// event day (QR scan, check-in, walk-ins, souvenirs), the catering projection, and the import and export of the list.
// It never imports pgx; transactions travel in the context (pd.TxManager). The module plugs into menu 01 through the
// provider and the overlay in integration.go, and offers GuestRSVPWriter to the invitation module.
package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerguest/domain"
)

type Options struct {
	// Now is the clock; nil means time.Now. Tests inject a fixed one.
	Now func() time.Time
	// Invitations finds the public slug of an invitation. nil (the invitation module does not exist yet) means no
	// guest has a personalized_url.
	Invitations pd.InvitationLookup
	// PublicBaseURL is the origin of the public pages ({PublicBaseURL}/i/{slug}); empty = no personal links.
	PublicBaseURL string
}

type Service struct {
	repo        domain.Repository
	tx          pd.TxManager
	registry    *pd.Registry
	now         func() time.Time
	invitations pd.InvitationLookup
	baseURL     string
}

func New(repo domain.Repository, tx pd.TxManager, registry *pd.Registry, opts Options) *Service {
	s := &Service{repo: repo, tx: tx, registry: registry, now: opts.Now, invitations: opts.Invitations, baseURL: opts.PublicBaseURL}
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

// spec is the guest part of the registry for the event type of p, and the whole spec for the labels.
func (s *Service) spec(p pd.Project) (domain.GuestSpec, pd.EventTypeSpec) {
	et, _ := s.registry.Get(p.EventType)

	return domain.GuestSpec{Sides: et.GuestSides, Categories: et.GuestCategories}, et
}

// mapErr turns a repository sentinel into the API error; unknown errors pass through and become 500.
func mapErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domain.ErrNotFound):
		return apierr.NotFound("guest")
	case errors.Is(err, domain.ErrInvalidScope):
		return errUnknownScope()
	case errors.Is(err, domain.ErrAlreadyCheckedIn):
		return apierr.Conflict("already_checked_in", "the guest is already checked in")
	case errors.Is(err, domain.ErrConflict):
		return apierr.Conflict("conflict", "the project was changed concurrently, try again")
	}

	return err
}

func errUnknownScope() *apierr.Error {
	return apierr.BadRequest("invalid_scope", "sub_event_scope: unknown session")
}

func errLimit() *apierr.Error {
	return apierr.Conflict("guest_limit_reached", fmt.Sprintf("at most %d guests per project", maxGuests))
}

// newToken is the ticket token of a guest: 16 random bytes as 32 hex characters (05 BE 2.1). A collision is
// practically impossible; if it ever happened the unique index would fail the statement with a 409 and a retry of the
// request would get a new token (inside a transaction the statement cannot simply be repeated).
func newToken() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}

	return hex.EncodeToString(b[:]), nil
}

// sessionScopes returns the scope keys of the project's sessions (the foreign key is deferred, so a wrong one would
// only surface at COMMIT) and the sessions themselves.
func (s *Service) sessionScopes(ctx context.Context, projectID uuid.UUID) (map[string]bool, []domain.SessionRef, error) {
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

// linkFunc fills personalized_url of the guests it is given. It is built once per request: the slug is looked up once,
// not once per guest.
type linkFunc func(g *domain.Guest)

func noLinks(*domain.Guest) {}

func (s *Service) links(ctx context.Context, p pd.Project) (linkFunc, error) {
	if s.invitations == nil || s.baseURL == "" || p.InvitationID == nil {
		return noLinks, nil
	}
	slug, err := s.invitations.PublicSlug(ctx, p.UserID, *p.InvitationID)
	if errors.Is(err, pd.ErrNotFound) {
		return noLinks, nil // the invitation was deleted: the guests keep their tickets, they just have no link
	}
	if err != nil {
		return nil, err
	}

	return func(g *domain.Guest) {
		u := pd.PersonalURL(s.baseURL, slug, g.Name, g.QRCodeToken)
		g.PersonalizedURL = &u
	}, nil
}

func (l linkFunc) all(gs []domain.Guest) {
	for i := range gs {
		l(&gs[i])
	}
}
