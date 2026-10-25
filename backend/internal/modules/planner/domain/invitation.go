package domain

import (
	"context"
	"net/url"
	"strings"

	"github.com/google/uuid"
)

// The contract between the guest module (menu 05) and the Digital Invitation module (05 BE chapter 5). It lives here
// because both sides may import this package and nothing else of each other. Until the invitation module exists the
// guest module is wired without a lookup, so no guest has a personal link.

// RSVPStatus is the answer of a guest (planner_guests.rsvp_status). It is defined here, not in the guest module,
// because GuestRSVPWriter has to name it.
type RSVPStatus string

const (
	RSVPUnconfirmed RSVPStatus = "unconfirmed"
	RSVPAttending   RSVPStatus = "attending"
	RSVPDeclined    RSVPStatus = "declined"
	RSVPMaybe       RSVPStatus = "maybe"
)

// InvitationLookup is implemented by the invitation module and used for personal links.
type InvitationLookup interface {
	// PublicSlug returns the public slug of the invitation of userID; ErrNotFound when the invitation does not exist
	// or belongs to somebody else.
	PublicSlug(ctx context.Context, userID, invitationID uuid.UUID) (string, error)
}

// GuestRSVPWriter is implemented by the guest module and called by the invitation module when a guest answers on the
// public page. The guest is found by the ticket token of the personal link (parameter g) and must belong to
// invitationID; the invitation module never reads or writes planner_guests.
type GuestRSVPWriter interface {
	// ApplyRSVP is idempotent: the same status and pax succeed without a change. A guest who has already checked in is
	// left alone and the call succeeds (the guest is at the door; the public form need not show an error). pax is
	// ignored unless the status is attending (0 = use the invitation's pax_count). ErrNotFound when no guest has that
	// token and invitation; a status or pax out of range is an apierr 400 invalid_input like in every usecase.
	ApplyRSVP(ctx context.Context, invitationID uuid.UUID, qrToken string, status RSVPStatus, pax int) error
}

// PersonalURL builds {baseURL}/i/{slug}?g={token}&to={name}. `to` is read by the invitation page to greet the guest;
// `g` identifies the guest for the public RSVP (a 128-bit token that cannot be guessed).
func PersonalURL(baseURL, slug, guestName, qrToken string) string {
	q := url.Values{"to": {guestName}, "g": {qrToken}}

	return strings.TrimRight(baseURL, "/") + "/i/" + url.PathEscape(slug) + "?" + q.Encode()
}
