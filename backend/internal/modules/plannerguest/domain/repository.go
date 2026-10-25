package domain

import (
	"context"
	"errors"

	"github.com/google/uuid"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

// ErrNotFound, ErrInvalidScope and ErrConflict are the sentinels of the planner module (the shared TxManager already
// returns pd.ErrConflict for a failed COMMIT); they are reused so one mapping serves every planner module.
var (
	ErrNotFound     = pd.ErrNotFound
	ErrInvalidScope = pd.ErrInvalidScope
	ErrConflict     = pd.ErrConflict

	// ErrAlreadyCheckedIn: the guest exists but has checked in already (a second check-in, or an RSVP change after it).
	ErrAlreadyCheckedIn = errors.New("guest already checked in")
)

// GuestQuery is the filter of GET /guests (05 BE 2.7). An empty field is "no filter"; Side may be SideNone.
type GuestQuery struct {
	Side, Category, RSVP, Sent, Scope, Search string
	Limit, Offset                             int
}

// Reception statuses (05 BE 2.5).
const (
	StatusAll     = "all"     // expected guests
	StatusIn      = "in"      // checked in
	StatusPending = "pending" // expected and not checked in yet
)

// ReceptionQuery is the filter of the reception queue. A non-empty Search also reaches the guests who declined and
// have not arrived.
type ReceptionQuery struct {
	Status, Category, Search string
	Limit, Offset            int
}

// ReceptionCounts is what the KPI and the filter chips are built from (05 BE 2.5): the summary row and the number of
// expected guests per category.
type ReceptionCounts struct {
	KPI        ReceptionKPI // without SouvenirStock
	ByCategory map[string]int
}

// Repository is the PostgreSQL side of the module. Every method that takes a projectID filters by it; ids that come
// from a URL are always looked up with both (WHERE id = $x AND project_id = $y). Guests come back without
// PersonalizedURL: the usecase adds it.
type Repository interface {
	// LockProject takes SELECT … FOR UPDATE on the project row. Needs a transaction.
	LockProject(ctx context.Context, projectID uuid.UUID) error
	Sessions(ctx context.Context, projectID uuid.UUID) ([]SessionRef, error)

	// ---------- reads ----------
	// ListGuests returns one page in creation order and the number of guests matching the filters.
	ListGuests(ctx context.Context, projectID uuid.UUID, q GuestQuery) ([]Guest, int, error)
	// AllGuests returns every guest in creation order (the exports).
	AllGuests(ctx context.Context, projectID uuid.UUID) ([]Guest, error)
	GuestByID(ctx context.Context, projectID, id uuid.UUID) (Guest, error)
	GuestByToken(ctx context.Context, projectID uuid.UUID, token string) (Guest, error)
	// Aggregates is the summary row and the side x category cells of 05 BE 2.8.
	Aggregates(ctx context.Context, projectID uuid.UUID) (Aggregates, error)
	CountGuests(ctx context.Context, projectID uuid.UUID) (int, error)
	// Catering is the projection of 05 BE 2.3 without the buffer (confirmed, potential, declined and the four counts).
	Catering(ctx context.Context, projectID uuid.UUID) (CateringProjection, error)
	// ExistingForImport lists name and WhatsApp of every guest, to find duplicates.
	ExistingForImport(ctx context.Context, projectID uuid.UUID) ([]ExistingGuest, error)

	// ---------- writes ----------
	// InsertGuest appends the guest (sort_order = max + 1). invitationID is copied from the project (nil = standalone).
	InsertGuest(ctx context.Context, projectID uuid.UUID, invitationID *uuid.UUID, g *Guest) error
	// InsertGuests appends the guests in order, many rows per statement.
	InsertGuests(ctx context.Context, projectID uuid.UUID, invitationID *uuid.UUID, gs []Guest) error
	// UpdateGuest writes name, side, category, scope, table, pax_count, phone and notes (never RSVP, check-in,
	// souvenir, token or order) and refreshes g from the row.
	UpdateGuest(ctx context.Context, projectID uuid.UUID, g *Guest) error
	SetSentStatus(ctx context.Context, projectID, id uuid.UUID, status SentStatus) (Guest, error)
	// SetRSVP returns ErrAlreadyCheckedIn when the guest has checked in.
	SetRSVP(ctx context.Context, projectID, id uuid.UUID, status RSVPStatus, paxActual int) (Guest, error)
	// ApplyRSVPByToken is the public RSVP of the invitation module: the guest is found by invitation and ticket
	// token. A guest who has checked in is left alone and the call succeeds; ErrNotFound when there is no such guest.
	ApplyRSVPByToken(ctx context.Context, invitationID uuid.UUID, token string, status RSVPStatus, paxActual int) error
	SetSouvenir(ctx context.Context, projectID, id uuid.UUID, received bool) (Guest, error)
	DeleteGuest(ctx context.Context, projectID, id uuid.UUID) error

	// ---------- reception ----------
	// Checkin is one atomic UPDATE … WHERE is_checked_in = FALSE: ErrNotFound for an unknown guest, ErrAlreadyCheckedIn
	// when somebody got there first.
	Checkin(ctx context.Context, projectID, id uuid.UUID, paxActual int, souvenir bool) (Guest, error)
	// CancelCheckin clears the check-in and the souvenir mark but keeps pax_actual; a guest who has not checked in is returned as is.
	CancelCheckin(ctx context.Context, projectID, id uuid.UUID) (Guest, error)
	ReceptionCounts(ctx context.Context, projectID uuid.UUID) (ReceptionCounts, error)
	// ReceptionQueue returns one page of the queue and the number of guests matching.
	ReceptionQueue(ctx context.Context, projectID uuid.UUID, q ReceptionQuery) ([]Guest, int, error)
	// RecentCheckins returns the newest check-ins, newest first.
	RecentCheckins(ctx context.Context, projectID uuid.UUID, n int) ([]Guest, error)
	// SouvenirStock is the stock of the shopping items marked souvenir (05 BE 2.5); found is false when there is none.
	SouvenirStock(ctx context.Context, projectID uuid.UUID) (stock int, found bool, err error)
}
