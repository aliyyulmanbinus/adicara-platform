package domain

import (
	"time"

	"github.com/google/uuid"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

// The types below belong to menu 01 (planner/domain) and are aliases, not copies: the JSON shape is one and the same,
// and the dashboard provider returns the very types the guest list embeds.
type (
	Option      = pd.Option
	GuestSide   = pd.GuestSide
	RSVPSummary = pd.RSVPSummary
	RSVPStatus  = pd.RSVPStatus
)

const (
	RSVPUnconfirmed = pd.RSVPUnconfirmed
	RSVPAttending   = pd.RSVPAttending
	RSVPDeclined    = pd.RSVPDeclined
	RSVPMaybe       = pd.RSVPMaybe
)

type SentStatus string

const (
	SentUnsent    SentStatus = "unsent"
	SentSent      SentStatus = "sent"
	SentDelivered SentStatus = "delivered" // written by the invitation module only (05 BE 5); the planner writes unsent/sent
)

// ---------- requests (no binding tags: the usecase validates, with messages "<field>: <reason>") ----------

// GuestRequest is the body of POST and PUT. PUT replaces all of these fields; RSVP, invitation status, check-in,
// souvenir, token and order never change through PUT.
type GuestRequest struct {
	Name          string  `json:"name"`
	GuestSide     *string `json:"guest_side"`      // key of a registry side; empty = not decided yet
	Category      string  `json:"category"`        // registry key or label, or free text; empty = "lainnya"
	SubEventScope string  `json:"sub_event_scope"` // empty = "all"
	TableNumber   *string `json:"table_number"`
	PaxCount      int     `json:"pax_count"` // 1-20
	PhoneNumber   *string `json:"phone_number"`
	Notes         *string `json:"notes"`
}

type SentStatusRequest struct {
	SentStatus SentStatus `json:"sent_status"` // unsent | sent
}

// RSVPRequest: PaxActual only means something for attending (0 = follow pax_count); other statuses store 0.
type RSVPRequest struct {
	RSVPStatus RSVPStatus `json:"rsvp_status"`
	PaxActual  int        `json:"pax_actual"`
}

type ScanQRRequest struct {
	QRCodeToken string `json:"qr_code_token"`
}

type CheckinRequest struct {
	PaxActual        int  `json:"pax_actual"` // 1-20
	SouvenirReceived bool `json:"souvenir_received"`
}

type SouvenirRequest struct {
	SouvenirReceived bool `json:"souvenir_received"`
}

type WalkInRequest struct {
	Name             string  `json:"name"`
	GuestSide        *string `json:"guest_side"`
	Category         string  `json:"category"`
	TableNumber      *string `json:"table_number"`
	PaxActual        int     `json:"pax_actual"` // 1-20; also becomes pax_count
	SouvenirReceived bool    `json:"souvenir_received"`
	Notes            *string `json:"notes"`
}

// ---------- responses ----------

// Guest: optional keys are omitted when empty (there is no null in this menu, except GuestSide.Key of menu 01).
type Guest struct {
	ID                   uuid.UUID  `json:"id"`
	Name                 string     `json:"name"`
	GuestSide            *string    `json:"guest_side,omitempty"`
	Category             string     `json:"category"`
	SubEventScope        string     `json:"sub_event_scope"`
	TableNumber          *string    `json:"table_number,omitempty"`
	PaxCount             int        `json:"pax_count"`
	PhoneNumber          *string    `json:"phone_number,omitempty"`
	FormattedWANumber    *string    `json:"formatted_wa_number,omitempty"` // 62...; the wa.me link is built by the client
	PersonalizedURL      *string    `json:"personalized_url,omitempty"`    // only when an invitation is connected
	InvitationSentStatus SentStatus `json:"invitation_sent_status"`
	InvitationSentAt     *time.Time `json:"invitation_sent_at,omitempty"`
	RSVPStatus           RSVPStatus `json:"rsvp_status"`
	PaxActual            int        `json:"pax_actual"`
	RSVPSubmittedAt      *time.Time `json:"rsvp_submitted_at,omitempty"`
	QRCodeToken          string     `json:"qr_code_token"`
	IsCheckedIn          bool       `json:"is_checked_in"`
	CheckedInAt          *time.Time `json:"checked_in_at,omitempty"`
	SouvenirReceived     bool       `json:"souvenir_received"`
	Notes                *string    `json:"notes,omitempty"`
}

type CategoryCount struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Custom      bool   `json:"custom,omitempty"` // not a registry category of the event type ("lainnya" and free text)
	Invitations int    `json:"invitations"`
	Pax         int    `json:"pax"`
}

// SideCategoryCount is one cell of the side x category table (filled cells only); Side nil = side not decided yet.
type SideCategoryCount struct {
	Side        *string `json:"side"`
	Category    string  `json:"category"`
	Invitations int     `json:"invitations"`
	Pax         int     `json:"pax"`
}

type InvitationCounts struct {
	Sent   int `json:"sent"` // sent + delivered
	Unsent int `json:"unsent"`
}

// GuestOverview is always counted over the whole project, never affected by a filter, a search or a page.
type GuestOverview struct {
	Total          int                 `json:"total"`
	TotalPax       int                 `json:"total_pax"`
	Sides          []GuestSide         `json:"sides"`           // empty for a type without sides
	Categories     []CategoryCount     `json:"categories"`      // registry categories (also the empty ones), then the used "lainnya"/free ones
	SideCategories []SideCategoryCount `json:"side_categories"` // empty for a type without sides
	Invitation     InvitationCounts    `json:"invitation"`
	RSVP           RSVPSummary         `json:"rsvp"` // four statuses; they add up to Total
}

type GuestList struct {
	Overview GuestOverview `json:"overview"`
	Guests   []Guest       `json:"guests"`
	Total    int           `json:"total"` // matching the filters (for the pager), not Overview.Total
	Limit    int           `json:"limit"`
	Offset   int           `json:"offset"`
}

type ReceptionKPI struct {
	CheckedInGuests     int  `json:"checked_in_guests"`
	ExpectedGuests      int  `json:"expected_guests"` // has not declined, or declined but came
	PendingGuests       int  `json:"pending_guests"`  // ExpectedGuests - CheckedInGuests
	ActualPax           int  `json:"actual_pax"`
	ExpectedPax         int  `json:"expected_pax"`
	SouvenirDistributed int  `json:"souvenir_distributed"`
	SouvenirStock       *int `json:"souvenir_stock,omitempty"` // absent when no shopping item is marked souvenir (menu 04)
}

type ReceptionFilters struct {
	All        int             `json:"all"` // = ExpectedGuests
	CheckedIn  int             `json:"checked_in"`
	Pending    int             `json:"pending"`
	Categories []CategoryCount `json:"categories"` // expected guests only; Pax is not filled
}

type ReceptionOverview struct {
	KPI            ReceptionKPI     `json:"kpi"`
	Filters        ReceptionFilters `json:"filters"`
	RecentCheckins []Guest          `json:"recent_checkins"` // at most 10, newest first
	Guests         []Guest          `json:"guests"`
	Total          int              `json:"total"`
	Limit          int              `json:"limit"`
	Offset         int              `json:"offset"`
}

type CateringProjection struct {
	ConfirmedPax        int `json:"confirmed_pax"` // attending only
	PotentialPax        int `json:"potential_pax"` // maybe + unconfirmed (not part of the recommendation)
	DeclinedPax         int `json:"declined_pax"`
	RecommendedPortions int `json:"recommended_portions"`
	AttendingGuests     int `json:"attending_guests"`
	DeclinedGuests      int `json:"declined_guests"`
	MaybeGuests         int `json:"maybe_guests"`
	PendingGuests       int `json:"pending_guests"` // unconfirmed
	BufferPercent       int `json:"buffer_percent"`
	BufferPax           int `json:"buffer_pax"`
}

// ---------- import ----------

// ImportRequest is the body of POST bulk-import. The file travels as JSON, not multipart: the BFF forwards
// application/json only (contract 1.3) and Astro checks the origin of multipart bodies behind a TLS proxy.
// 5 MiB of file is about 6.7 MiB of base64, under client_max_body_size 8m.
type ImportRequest struct {
	Filename      string `json:"filename"`       // ends in .csv or .xlsx
	ContentBase64 string `json:"content_base64"` // the file, standard base64
}

type ImportRowError struct {
	Row    int    `json:"row"` // line in the file (the header is row 1)
	Field  string `json:"field"`
	Reason string `json:"reason"` // the fixed reasons of 05 BE 3.4
}

type ImportDuplicate struct {
	Row     int        `json:"row"`
	Name    string     `json:"name"`
	OfRow   *int       `json:"of_row,omitempty"`   // same as another row of this file
	OfGuest *uuid.UUID `json:"of_guest,omitempty"` // same as a guest that already exists
}

type ImportPreviewRow struct {
	Row               int     `json:"row"`
	Name              string  `json:"name"`
	GuestSide         *string `json:"guest_side,omitempty"`
	Category          string  `json:"category"`
	SubEventScope     string  `json:"sub_event_scope"`
	TableNumber       *string `json:"table_number,omitempty"`
	PaxCount          int     `json:"pax_count"`
	FormattedWANumber *string `json:"formatted_wa_number,omitempty"`
}

type ImportResult struct {
	DryRun      bool               `json:"dry_run"`
	TotalRows   int                `json:"total_rows"`
	ValidRows   int                `json:"valid_rows"`
	InvalidRows int                `json:"invalid_rows"`
	Created     int                `json:"created"`
	Skipped     int                `json:"skipped"` // invalid and duplicate rows that were left out
	Errors      []ImportRowError   `json:"errors"`
	Duplicates  []ImportDuplicate  `json:"duplicates"`
	Preview     []ImportPreviewRow `json:"preview"` // at most the first 5 valid rows
}
