package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerrundown/domain"
)

func wantAPIError(t *testing.T, err error, status int, code, message string) {
	t.Helper()
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.Status != status || ae.Code != code || ae.Message != message {
		t.Errorf("error = %v, want %d %s: %s", err, status, code, message)
	}
}

func TestMapErr(t *testing.T) {
	wantAPIError(t, mapErr(domain.ErrNotFound, "agenda"), 404, "not_found", "agenda not found")
	wantAPIError(t, mapErr(domain.ErrInvalidScope, "agenda"), 400, "invalid_scope", "session_id: must be a session of this project")
	wantAPIError(t, mapErr(domain.ErrInvalidVendor, "agenda"), 400, "invalid_input", "vendor_id: must be a vendor of this project")
	wantAPIError(t, mapErr(domain.ErrInvalidAgenda, "song"), 400, "invalid_input", "rundown_id: must be an agenda of this project")
	wantAPIError(t, mapErr(domain.ErrConflict, "agenda"), 409, "conflict", "the project was changed concurrently, try again")
	if mapErr(nil, "agenda") != nil {
		t.Error("nil must stay nil")
	}
	// an error that is already an API error, and one nobody knows, pass through (the latter becomes a 500)
	own := apierr.Conflict("agenda_limit_reached", "x")
	if mapErr(own, "agenda") != error(own) {
		t.Error("an API error was rewritten")
	}
	boom := errors.New("boom")
	if mapErr(boom, "agenda") != boom {
		t.Error("an unknown error was rewritten")
	}
}

func TestSameIDs(t *testing.T) {
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	for name, tc := range map[string]struct {
		want, got []uuid.UUID
		same      bool
	}{
		"same order":          {[]uuid.UUID{a, b}, []uuid.UUID{a, b}, true},
		"another order":       {[]uuid.UUID{a, b, c}, []uuid.UUID{c, a, b}, true},
		"both empty":          {nil, []uuid.UUID{}, true},
		"a repeated id":       {[]uuid.UUID{a, b}, []uuid.UUID{a, a}, false},
		"a repeat of three":   {[]uuid.UUID{a, b, c}, []uuid.UUID{a, b, b}, false},
		"missing":             {[]uuid.UUID{a, b}, []uuid.UUID{a}, false},
		"extra":               {[]uuid.UUID{a}, []uuid.UUID{a, b}, false},
		"another id":          {[]uuid.UUID{a, b}, []uuid.UUID{a, c}, false},
		"empty for non-empty": {[]uuid.UUID{a}, nil, false},
	} {
		if got := sameIDs(tc.want, tc.got); got != tc.same {
			t.Errorf("%s: sameIDs = %v, want %v", name, got, tc.same)
		}
	}
}

func TestParseSessionQuery(t *testing.T) {
	id := uuid.New()
	if got, err := parseSessionQuery(id.String()); err != nil || got != id {
		t.Errorf("a canonical id: %v %v", got, err)
	}
	for _, bad := range []string{"", "x", strings.ToUpper(id.String()), "{" + id.String() + "}", strings.ReplaceAll(id.String(), "-", "")} {
		wantAPIError(t, errOf(parseSessionQuery(bad)), 400, "invalid_scope", "session_id: must be a session of this project")
	}
}

func errOf(_ uuid.UUID, err error) error { return err }

func TestTemplateModeRules(t *testing.T) {
	for _, tc := range []struct {
		mode       string
		appendOnly bool
		want       string
		code, msg  string
	}{
		{"", false, "append", "", ""}, {"append", false, "append", "", ""}, {"replace_template", false, "replace_template", "", ""},
		{"", true, "append", "", ""}, {"append", true, "append", "", ""},
		{"replace_template", true, "", "invalid_input", "mode: must be append"},
		{"replace_all", false, "", "invalid_input", "mode: must be one of append, replace_template"},
		{"APPEND", true, "", "invalid_input", "mode: must be one of append, replace_template"},
	} {
		got, err := templateMode(tc.mode, tc.appendOnly)
		if tc.code == "" {
			if err != nil || got != tc.want {
				t.Errorf("templateMode(%q, %v) = %q, %v", tc.mode, tc.appendOnly, got, err)
			}

			continue
		}
		wantAPIError(t, err, 400, tc.code, tc.msg)
	}
}

// A type the registry has no template for is refused before anything is read or written (every real type has one, so
// only a unit test reaches this answer); the repository is not needed.
func TestApplyTemplateRefusesATypeWithoutTemplate(t *testing.T) {
	s := New(nil, nil, pd.DefaultRegistry(), Options{})
	_, err := s.applyAgendaTemplate(context.Background(), pd.Project{EventType: "unknown"}, domain.ModeAppend, false)
	wantAPIError(t, err, 400, "invalid_template", "rundown: not available for this event type")
	_, err = s.applyPlaylistTemplate(context.Background(), pd.Project{EventType: pd.EventSeminar})
	wantAPIError(t, err, 400, "invalid_template", "playlist: not available for this event type")
}

func TestClockDefaultsToTheWallClockAndKnowsTheProjectZone(t *testing.T) {
	s := New(nil, nil, pd.DefaultRegistry(), Options{})
	day, err := s.today(pd.Project{Timezone: "Asia/Jakarta"})
	if err != nil || time.Since(day) > 48*time.Hour || day.Hour() != 0 {
		t.Errorf("today = %v, %v", day, err)
	}
	if _, err := s.today(pd.Project{Timezone: "Mars/Olympus"}); err == nil {
		t.Error("an unknown zone must be an error")
	}
	fixed := New(nil, nil, pd.DefaultRegistry(), Options{Now: func() time.Time { return time.Date(2026, 10, 4, 20, 0, 0, 0, time.UTC) }})
	if d, _ := fixed.today(pd.Project{Timezone: "Asia/Jakarta"}); d.Format(pd.DateLayout) != "2026-10-05" {
		t.Errorf("20:00 UTC is already 5 October in Jakarta, got %v", d)
	}
}
