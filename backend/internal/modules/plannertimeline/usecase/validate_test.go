package usecase

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

func TestCleanNoteAllowsLineBreaksButNoOtherControlCharacters(t *testing.T) {
	// Menu 01's cleanText rejects "\n" because unicode.IsControl('\n') is true; notes need it.
	got, err := cleanNote("notes", "  baris 1\r\nbaris 2\n\nbaris 4  ", 100)
	if err != nil || got != "baris 1\nbaris 2\n\nbaris 4" {
		t.Fatalf("got %q, %v", got, err)
	}
	for _, bad := range []string{"a\tb", "a\x00b", "a\rb", "a\x1bb", "a\u0085b"} {
		if _, err := cleanNote("notes", bad, 100); err == nil || !strings.Contains(err.Error(), "notes: must not contain control characters") {
			t.Errorf("%q must be refused, got %v", bad, err)
		}
	}
	if _, err := cleanNote("notes", strings.Repeat("é", 101), 100); err == nil || !strings.Contains(err.Error(), "must be at most 100 characters") {
		t.Errorf("the limit counts characters, not bytes: %v", err)
	}
	if got, err := cleanNote("notes", strings.Repeat("é", 100), 100); err != nil || len([]rune(got)) != 100 {
		t.Errorf("100 characters fit: %v", err)
	}
	if _, err := cleanText("title", "a\nb", 10, true); err == nil {
		t.Error("cleanText must keep refusing a line break")
	}
}

func TestCheckIDs(t *testing.T) {
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	if err := checkIDs([]uuid.UUID{a, b, c}, 3); err != nil {
		t.Errorf("a complete list: %v", err)
	}
	for name, ids := range map[string][]uuid.UUID{"short": {a, b}, "long": {a, b, c, a}, "repeated": {a, a, b}, "empty": nil} {
		if err := checkIDs(ids, 3); err == nil || !strings.Contains(err.Error(), "ids: must list every id exactly once") {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestParseMode(t *testing.T) {
	for in, want := range map[string]string{"": "append", " append ": "append", "replace_template": "replace_template", "replace_all": "replace_all"} {
		if got, err := parseMode(in, true); err != nil || got != want {
			t.Errorf("parseMode(%q) = %q, %v", in, got, err)
		}
	}
	if _, err := parseMode("replace_all", false); err == nil || !strings.Contains(err.Error(), "must be one of append, replace_template") {
		t.Errorf("documents have no replace_all: %v", err)
	}
	if _, err := parseMode("merge", true); err == nil {
		t.Error("an unknown mode must be refused")
	}
}

func TestTaskQueryFilter(t *testing.T) {
	f, err := TaskQuery{Status: "proses", Category: " Vendor ", Scope: "resepsi", Phase: "3"}.filter()
	if err != nil || f.Status != "proses" || f.Category != "Vendor" || *f.Scope != "resepsi" || *f.Phase != 3 {
		t.Fatalf("filter = %+v, %v", f, err)
	}
	if f, err := (TaskQuery{}).filter(); err != nil || f.Scope != nil || f.Phase != nil {
		t.Fatalf("empty query: %+v, %v", f, err)
	}
	for name, q := range map[string]TaskQuery{
		"status": {Status: "done"}, "scope": {Scope: "Bad Scope"}, "phase text": {Phase: "x"}, "phase 0": {Phase: "0"}, "phase 21": {Phase: "21"},
		"category": {Category: strings.Repeat("c", 61)},
	} {
		if _, err := q.filter(); err == nil {
			t.Errorf("%s must be refused", name)
		}
	}
}

func TestScheduleWindow(t *testing.T) {
	end := time.Date(2027, 3, 29, 0, 0, 0, 0, time.UTC)
	p := pd.Project{EventDate: time.Date(2027, 3, 28, 0, 0, 0, 0, time.UTC), EventEndDate: &end}
	lo, hi := scheduleWindow(p)
	if lo.Format(pd.DateLayout) != "2024-03-28" || hi.Format(pd.DateLayout) != "2028-03-28" {
		t.Errorf("window = %s .. %s", lo.Format(pd.DateLayout), hi.Format(pd.DateLayout))
	}
	if checkWindow(p, "start_date", lo) != nil || checkWindow(p, "end_date", hi) != nil {
		t.Error("the edges are inside")
	}
	if checkWindow(p, "start_date", lo.AddDate(0, 0, -1)) == nil || checkWindow(p, "end_date", hi.AddDate(0, 0, 1)) == nil {
		t.Error("one day outside must be refused")
	}
}

func TestDefaultSubject(t *testing.T) {
	reg := pd.DefaultRegistry()
	wedding, _ := reg.Get(pd.EventWedding)
	seminar, _ := reg.Get(pd.EventSeminar)
	if got := defaultSubject(wedding); got != "bersama" {
		t.Errorf("wedding default = %q", got)
	}
	if got := defaultSubject(seminar); got != "umum" {
		t.Errorf("seminar default = %q", got)
	}
	if got := defaultSubject(pd.EventTypeSpec{AdminSubjects: []pd.Option{{Key: "x"}}}); got != "x" {
		t.Errorf("fallback = %q", got)
	}
}
