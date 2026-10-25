package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerguest/domain"
)

func wantAPIError(t *testing.T, err error, code, message string) {
	t.Helper()
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.Code != code || (message != "" && ae.Message != message) {
		t.Fatalf("got %v, want %s %q", err, code, message)
	}
}

func ptr(s string) *string { return &s }

func spec(t *testing.T, key pd.EventType) domain.GuestSpec {
	t.Helper()
	et, ok := pd.DefaultRegistry().Get(key)
	if !ok {
		t.Fatalf("no event type %s", key)
	}

	return domain.GuestSpec{Sides: et.GuestSides, Categories: et.GuestCategories}
}

var scopes = map[string]bool{"all": true, "akad": true, "resepsi": true}

func TestCheckGuestNormalisesAndValidatesInTheOrderOfTheSpec(t *testing.T) {
	w := spec(t, pd.EventWedding)
	f, err := checkGuest(w, scopes, domain.GuestRequest{
		Name: "  Budi \t Santoso\n", GuestSide: ptr(" Pihak Wanita "), Category: "teman orang tua", SubEventScope: " akad ", TableNumber: ptr("  T-01  "),
		PaxCount: 3, PhoneNumber: ptr(" 0812 3456 7890 "), Notes: ptr("  a\r\nb  "),
	}, "pax_count")
	if err != nil {
		t.Fatal(err)
	}
	if f.name != "Budi Santoso" || *f.side != "wanita" || f.category != "teman_ortu" || f.scope != "akad" || *f.table != "T-01" || f.pax != 3 ||
		*f.phone != "0812 3456 7890" || *f.wa != "6281234567890" || *f.notes != "a\nb" {
		t.Errorf("fields = %+v", f)
	}
	// Blank optional values become absent, not empty strings.
	f, err = checkGuest(w, scopes, domain.GuestRequest{Name: "X", GuestSide: ptr("  "), TableNumber: ptr(" "), PhoneNumber: ptr(""), Notes: ptr(" \n "), PaxCount: 1}, "pax_count")
	if err != nil || f.side != nil || f.table != nil || f.phone != nil || f.wa != nil || f.notes != nil || f.category != "lainnya" || f.scope != "all" {
		t.Errorf("blanks = %+v, %v", f, err)
	}

	// The first problem in the order of the table is the one reported.
	_, err = checkGuest(w, scopes, domain.GuestRequest{Name: "", GuestSide: ptr("x"), PaxCount: 0}, "pax_count")
	wantAPIError(t, err, "invalid_input", "name: required")
	_, err = checkGuest(w, scopes, domain.GuestRequest{Name: "X", GuestSide: ptr("x"), PaxCount: 0, SubEventScope: "nope"}, "pax_count")
	wantAPIError(t, err, "invalid_input", "guest_side: must be a registry side key")
	_, err = checkGuest(w, scopes, domain.GuestRequest{Name: "X", PaxCount: 0, SubEventScope: "nope"}, "pax_count")
	wantAPIError(t, err, "invalid_scope", "sub_event_scope: unknown session")
	_, err = checkGuest(w, scopes, domain.GuestRequest{Name: "X", PaxCount: 0, PhoneNumber: ptr("12")}, "pax_count")
	wantAPIError(t, err, "invalid_input", "pax_count: must be between 1 and 20")
	_, err = checkGuest(w, scopes, domain.GuestRequest{Name: "X", PaxCount: 0, PhoneNumber: ptr("12")}, "pax_actual")
	wantAPIError(t, err, "invalid_input", "pax_actual: must be between 1 and 20")
	_, err = checkGuest(w, scopes, domain.GuestRequest{Name: "X", PaxCount: 1, PhoneNumber: ptr("12")}, "pax_count")
	wantAPIError(t, err, "invalid_phone", "phone_number: must be an Indonesian phone number (08…, +62…)")
}

func TestCleanSideRefusesAnySideForATypeWithoutSides(t *testing.T) {
	for _, key := range []pd.EventType{pd.EventBirthday, pd.EventWisuda, pd.EventOther} {
		s := spec(t, key)
		_, err := cleanSide(s, ptr("pria"))
		wantAPIError(t, err, "invalid_input", "guest_side: must be empty for an event type without sides")
		if got, err := cleanSide(s, ptr(" ")); err != nil || got != nil {
			t.Errorf("%s: a blank side = %v, %v", key, got, err)
		}
	}
	// A side belongs to its type: the wedding's "pria" is not a side of a seminar, whose sides are its own.
	_, err := cleanSide(spec(t, pd.EventSeminar), ptr("pria"))
	wantAPIError(t, err, "invalid_input", "guest_side: must be a registry side key")
	if got, err := cleanSide(spec(t, pd.EventSeminar), ptr("Peserta Eksternal")); err != nil || *got != "eksternal" {
		t.Errorf("seminar side = %v, %v", got, err)
	}
}

func TestTextLimitsCountRunes(t *testing.T) {
	w := spec(t, pd.EventWedding)
	if _, err := cleanName("name", strings.Repeat("é", 255), 255); err != nil {
		t.Errorf("255 runes of 2 bytes: %v", err)
	}
	_, err := cleanName("name", strings.Repeat("é", 256), 255)
	wantAPIError(t, err, "invalid_input", "name: must be at most 255 characters")
	if _, err := cleanCategory(w, strings.Repeat("日", 60)); err != nil {
		t.Errorf("60 runes: %v", err)
	}
	_, err = cleanCategory(w, strings.Repeat("日", 61))
	wantAPIError(t, err, "invalid_input", "category: must be at most 60 characters")
	_, err = cleanCategory(w, "a\tb\x07")
	wantAPIError(t, err, "invalid_input", "category: must not contain control characters")
	_, err = optionalNote("notes", ptr("a\x00b"), 2000)
	wantAPIError(t, err, "invalid_input", "notes: must not contain control characters")
	if n, err := optionalNote("notes", ptr("a\r\n\r\nb"), 2000); err != nil || *n != "a\n\nb" {
		t.Errorf("note = %v, %v", n, err)
	}
}

func TestPageParameters(t *testing.T) {
	for _, c := range []struct {
		limit, offset string
		def, max      int
		wantLimit     int
		wantOffset    int
		field, msg    string
	}{
		{"", "", 100, 500, 100, 0, "", ""},
		{"1", "0", 100, 500, 1, 0, "", ""},
		{"500", "4999", 100, 500, 500, 4999, "", ""},
		{"0", "", 100, 500, 0, 0, "limit", "limit: must be between 1 and 500"},
		{"501", "", 100, 500, 0, 0, "limit", "limit: must be between 1 and 500"},
		{"1.5", "", 100, 500, 0, 0, "limit", "limit: must be between 1 and 500"},
		{"", "-1", 100, 500, 0, 0, "offset", "offset: must be at least 0"},
		{"201", "", 50, 200, 0, 0, "limit", "limit: must be between 1 and 200"},
	} {
		limit, offset, err := page(c.limit, c.offset, c.def, c.max)
		if c.field != "" {
			wantAPIError(t, err, "invalid_input", c.msg)

			continue
		}
		if err != nil || limit != c.wantLimit || offset != c.wantOffset {
			t.Errorf("page(%q, %q) = %d, %d, %v", c.limit, c.offset, limit, offset, err)
		}
	}
}

func TestBufferParameters(t *testing.T) {
	if p, x, err := bufferOf("", ""); err != nil || p != 25 || x != 0 {
		t.Errorf("defaults = %d, %d, %v", p, x, err)
	}
	if p, x, err := bufferOf("0", "1000"); err != nil || p != 0 || x != 1000 {
		t.Errorf("limits = %d, %d, %v", p, x, err)
	}
	for _, c := range []struct{ percent, pax, msg string }{
		{"101", "", "buffer_percent: must be between 0 and 100"}, {"-1", "", "buffer_percent: must be between 0 and 100"},
		{"1.25", "", "buffer_percent: must be between 0 and 100"}, {"", "1001", "buffer_pax: must be between 0 and 1000"}, {"", "x", "buffer_pax: must be between 0 and 1000"},
	} {
		_, _, err := bufferOf(c.percent, c.pax)
		wantAPIError(t, err, "invalid_input", c.msg)
	}
}

func TestGuestFilterParsing(t *testing.T) {
	q, err := GuestFilter{Side: "none", Category: " vip ", RSVP: "declined", Sent: "unsent", Scope: "resepsi", Search: "  Bambang ", Limit: "20", Offset: "40"}.parse()
	if err != nil || q.Side != domain.SideNone || q.Category != "vip" || q.Search != "Bambang" || q.Limit != 20 || q.Offset != 40 {
		t.Errorf("query = %+v, %v", q, err)
	}
	for f, msg := range map[GuestFilter]string{
		{Side: "Pria"}: "side: must be a side key", {Scope: "a-b"}: "scope: must be a session scope_key", {RSVP: "x"}: "rsvp: must be one of unconfirmed, attending, declined, maybe",
		{Sent: "delivered"}: "sent: must be one of sent, unsent", {Search: strings.Repeat("s", 101)}: "search: must be at most 100 characters",
	} {
		_, err := f.parse()
		wantAPIError(t, err, "invalid_input", msg)
	}

	r, err := ReceptionFilter{}.parse()
	if err != nil || r.Status != domain.StatusAll || r.Limit != 50 || r.Offset != 0 {
		t.Errorf("defaults = %+v, %v", r, err)
	}
	_, err = ReceptionFilter{Status: "done"}.parse()
	wantAPIError(t, err, "invalid_input", "status: must be one of all, in, pending")
}

func TestCheckRSVPStoresPaxOnlyForAttending(t *testing.T) {
	for status, want := range map[domain.RSVPStatus]int{domain.RSVPAttending: 7, domain.RSVPDeclined: 0, domain.RSVPMaybe: 0, domain.RSVPUnconfirmed: 0} {
		if got, err := checkRSVP(status, 7); err != nil || got != want {
			t.Errorf("%s: %d, %v; want %d", status, got, err, want)
		}
	}
	_, err := checkRSVP("hadir", 1)
	wantAPIError(t, err, "invalid_input", "rsvp_status: must be one of unconfirmed, attending, declined, maybe")
	// The range is checked for every status, so a bad number is not hidden behind "declined".
	_, err = checkRSVP(domain.RSVPDeclined, 21)
	wantAPIError(t, err, "invalid_input", "pax_actual: must be between 0 and 20")
}

func TestNewTokenIsRandomHex(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		tk, err := newToken()
		if err != nil || len(tk) != 32 || strings.Trim(tk, "0123456789abcdef") != "" || seen[tk] {
			t.Fatalf("token %q: %v", tk, err)
		}
		seen[tk] = true
	}
}

type failingLookup struct{ err error }

func (f failingLookup) PublicSlug(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return "", f.err
}

// A broken invitation lookup is a server problem and must not be hidden as "no link"; a missing invitation is not.
func TestLinksDistinguishAMissingInvitationFromABrokenLookup(t *testing.T) {
	inv := uuid.New()
	p := pd.Project{ID: uuid.New(), UserID: uuid.New(), InvitationID: &inv}
	boom := errors.New("boom")

	s := New(nil, nil, pd.DefaultRegistry(), Options{Invitations: failingLookup{boom}, PublicBaseURL: "https://adicara.id"})
	if _, err := s.links(context.Background(), p); !errors.Is(err, boom) {
		t.Errorf("a broken lookup = %v", err)
	}
	s = New(nil, nil, pd.DefaultRegistry(), Options{Invitations: failingLookup{pd.ErrNotFound}, PublicBaseURL: "https://adicara.id"})
	if _, err := s.links(context.Background(), p); err != nil {
		t.Errorf("a missing invitation = %v", err)
	}
	// Without an invitation module, a base URL or a linked project the guest has no link and nothing is looked up.
	for name, svc := range map[string]*Service{
		"no module":   New(nil, nil, pd.DefaultRegistry(), Options{PublicBaseURL: "https://adicara.id"}),
		"no base URL": New(nil, nil, pd.DefaultRegistry(), Options{Invitations: failingLookup{boom}}),
	} {
		if _, err := svc.links(context.Background(), p); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	s = New(nil, nil, pd.DefaultRegistry(), Options{Invitations: failingLookup{boom}, PublicBaseURL: "https://adicara.id"})
	if _, err := s.links(context.Background(), pd.Project{ID: uuid.New()}); err != nil {
		t.Errorf("a standalone project looked something up: %v", err)
	}
}
