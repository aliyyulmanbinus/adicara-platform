package domain

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func mustLoc(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := LoadLocation(name)
	if err != nil {
		t.Fatalf("LoadLocation(%s): %v", name, err)
	}

	return loc
}

func date(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func datep(y int, m time.Month, d int) *time.Time {
	t := date(y, m, d)

	return &t
}

func TestComputePhaseAtMidnightBoundariesInProjectZone(t *testing.T) {
	wib := mustLoc(t, "Asia/Jakarta")
	at := func(y int, m time.Month, d, h, min int) time.Time { return time.Date(y, m, d, h, min, 0, 0, wib) }

	// One-day event on 28 Mar 2027.
	one := []struct {
		name  string
		now   time.Time
		phase EventPhase
		day   int
		days  int
	}{
		{"23:59 the day before", at(2027, 3, 27, 23, 59), PhaseUpcoming, 0, 1},
		{"00:00 on Hari-H", at(2027, 3, 28, 0, 0), PhaseOngoing, 1, 0},
		{"23:59 on Hari-H", at(2027, 3, 28, 23, 59), PhaseOngoing, 1, 0},
		{"00:00 the day after", at(2027, 3, 29, 0, 0), PhaseFinished, 0, -1},
	}
	for _, tc := range one {
		t.Run("1 day "+tc.name, func(t *testing.T) {
			got := ComputePhase(tc.now, wib, date(2027, 3, 28), nil)
			if got.Phase != tc.phase || got.DayNumber != tc.day || got.DaysRemaining != tc.days || got.TotalDays != 1 {
				t.Fatalf("got %+v, want phase=%s day=%d days_remaining=%d total=1", got, tc.phase, tc.day, tc.days)
			}
		})
	}

	// Three-day event, 28–30 Mar 2027.
	end := datep(2027, 3, 30)
	three := []struct {
		name  string
		now   time.Time
		phase EventPhase
		day   int
	}{
		{"before", at(2027, 3, 27, 23, 59), PhaseUpcoming, 0},
		{"day 1", at(2027, 3, 28, 8, 0), PhaseOngoing, 1},
		{"day 2", at(2027, 3, 29, 12, 0), PhaseOngoing, 2},
		{"day 3 at 23:59", at(2027, 3, 30, 23, 59), PhaseOngoing, 3},
		{"00:00 after the last day", at(2027, 3, 31, 0, 0), PhaseFinished, 0},
	}
	for _, tc := range three {
		t.Run("3 days "+tc.name, func(t *testing.T) {
			got := ComputePhase(tc.now, wib, date(2027, 3, 28), end)
			if got.Phase != tc.phase || got.DayNumber != tc.day || got.TotalDays != 3 {
				t.Fatalf("got %+v, want phase=%s day=%d total=3", got, tc.phase, tc.day)
			}
		})
	}
}

func TestComputePhaseUsesTheProjectZoneNotTheServer(t *testing.T) {
	// 17:00Z on 27 Mar is already 00:00 on 28 Mar in Jakarta but still 27 Mar in UTC.
	now := time.Date(2027, 3, 27, 17, 0, 0, 0, time.UTC)

	if got := ComputePhase(now, mustLoc(t, "UTC"), date(2027, 3, 28), nil); got.Phase != PhaseUpcoming || got.DaysRemaining != 1 {
		t.Fatalf("UTC project: got %+v, want upcoming in 1 day", got)
	}
	if got := ComputePhase(now, mustLoc(t, "Asia/Jakarta"), date(2027, 3, 28), nil); got.Phase != PhaseOngoing {
		t.Fatalf("Jakarta project: got %+v, want ongoing", got)
	}
}

func TestDaysUntilIsStableAcrossTheWIBDay(t *testing.T) {
	// 6 Oct 2026 → 28 Mar 2027 is 173 days. Truncating now to 24h (UTC) gave 174
	// between 00:00 and 06:59 WIB.
	wib := mustLoc(t, "Asia/Jakarta")
	for _, hm := range [][2]int{{0, 0}, {6, 59}, {7, 0}, {12, 30}, {23, 59}} {
		now := time.Date(2026, 10, 6, hm[0], hm[1], 0, 0, wib)
		got, err := DaysUntil(now, "Asia/Jakarta", date(2027, 3, 28))
		if err != nil {
			t.Fatal(err)
		}
		if got != 173 {
			t.Errorf("%02d:%02d WIB: DaysUntil = %d, want 173", hm[0], hm[1], got)
		}
	}

	if _, err := DaysUntil(time.Now(), "Foo/Bar", date(2027, 3, 28)); err == nil {
		t.Error("an unknown zone must be an error")
	}
}

func TestScopeKeyFromName(t *testing.T) {
	cases := map[string]string{
		"Ngunduh Mantu":           "ngunduh_mantu",
		"Resepsi  Siang!":         "resepsi_siang",
		"  Sesi — Pagi  ":         "sesi_pagi",
		"Café Étoile":             "cafe_etoile",
		"Acara 2027":              "acara_2027",
		"🎉":                       "sesi",
		"سلام":                    "sesi",
		"":                        "sesi",
		"all":                     "all_sesi",
		"ALL":                     "all_sesi",
		"___":                     "sesi",
		strings.Repeat("a", 60):   strings.Repeat("a", 36),
		strings.Repeat("ab ", 30): "ab_ab_ab_ab_ab_ab_ab_ab_ab_ab_ab_ab",
		"Akad Nikah":              "akad_nikah", // the wedding seed is keyed `akad` explicitly, not derived
		"Sesi 1":                  "sesi_1",
		"Foto-foto & Video/Reels": "foto_foto_video_reels",
	}
	pattern := regexp.MustCompile(`^[a-z0-9_]{1,40}$`)
	for name, want := range cases {
		got := ScopeKeyFromName(name)
		if got != want {
			t.Errorf("ScopeKeyFromName(%q) = %q, want %q", name, got, want)
		}
		if !pattern.MatchString(got) {
			t.Errorf("ScopeKeyFromName(%q) = %q does not match the column CHECK", name, got)
		}
	}
}

func TestUniqueScopeKeyAddsSuffixAndStaysWithinTheColumnLimit(t *testing.T) {
	base := strings.Repeat("a", 36)
	taken := map[string]bool{base: true, base + "_2": true}
	got, ok := UniqueScopeKey(base, taken)
	if !ok || got != base+"_3" {
		t.Fatalf("got %q ok=%v, want %q", got, ok, base+"_3")
	}
	if len(got) > MaxScopeKeyLen {
		t.Fatalf("%q is longer than %d", got, MaxScopeKeyLen)
	}

	if k, ok := UniqueScopeKey("x", map[string]bool{}); !ok || k != "x" {
		t.Fatalf("free base must be returned as is, got %q", k)
	}

	all := map[string]bool{"x": true}
	for n := 2; n <= 999; n++ {
		all["x_"+strconv.Itoa(n)] = true
	}
	if _, ok := UniqueScopeKey("x", all); ok {
		t.Fatal("exhausted suffixes must report failure")
	}
	longest, _ := UniqueScopeKey(base, map[string]bool{base: true})
	if len(longest) > MaxScopeKeyLen {
		t.Fatalf("suffixed key %q exceeds %d", longest, MaxScopeKeyLen)
	}
}

func TestDefaultRegistryInvariants(t *testing.T) {
	reg := DefaultRegistry()

	wantKeys := []string{"wedding", "khitanan", "aqiqah", "peresmian", "launching", "birthday", "wisuda", "seminar", "other"}
	if got := reg.Keys(); strings.Join(got, ",") != strings.Join(wantKeys, ",") {
		t.Fatalf("registry keys = %v, want %v", got, wantKeys)
	}

	scope := regexp.MustCompile(`^[a-z0-9_]{1,40}$`)
	for _, spec := range reg.All() {
		if spec.Label == "" || spec.Icon == "" || spec.LeadTimeDays <= 0 {
			t.Errorf("%s: label, icon and lead time are required", spec.Key)
		}
		if spec.Labels.Guest == "" || spec.Labels.GuestMenu == "" {
			t.Errorf("%s: labels are required", spec.Key)
		}
		if len(spec.Sessions) == 0 {
			t.Errorf("%s: needs at least one default session", spec.Key)
		}

		keys, names := map[string]bool{}, map[string]bool{}
		for _, s := range spec.Sessions {
			if !scope.MatchString(s.ScopeKey) || s.ScopeKey == AllScope {
				t.Errorf("%s: bad session scope_key %q", spec.Key, s.ScopeKey)
			}
			if n := utf8.RuneCountInString(s.Icon); n < 1 || n > 8 {
				t.Errorf("%s/%s: icon %q must be 1..8 characters", spec.Key, s.ScopeKey, s.Icon)
			}
			if keys[s.ScopeKey] || names[strings.ToLower(s.Name)] {
				t.Errorf("%s: duplicate session %q", spec.Key, s.ScopeKey)
			}
			keys[s.ScopeKey], names[strings.ToLower(s.Name)] = true, true
		}

		fields := map[string]bool{}
		for _, f := range spec.MetadataFields {
			if fields[f.Key] {
				t.Errorf("%s: duplicate metadata field %q", spec.Key, f.Key)
			}
			fields[f.Key] = true
			if f.Type == FieldString && f.MaxLen == 0 {
				t.Errorf("%s.%s: string field needs MaxLen", spec.Key, f.Key)
			}
		}
		for _, common := range []string{"hero_title", "venue_name", "city"} {
			if !fields[common] {
				t.Errorf("%s: missing common metadata field %s", spec.Key, common)
			}
		}
	}
}

func TestRegistryMatchesTheSpecTable(t *testing.T) {
	reg := DefaultRegistry()

	wedding, _ := reg.Get(EventWedding)
	if got := []string{wedding.Sessions[0].ScopeKey, wedding.Sessions[1].ScopeKey, wedding.Sessions[2].ScopeKey}; strings.Join(got, ",") != "lamaran,akad,resepsi" {
		t.Errorf("wedding sessions = %v; the specs and golden data use lamaran, akad, resepsi", got)
	}
	if wedding.Sessions[0].SameDay || !wedding.Sessions[1].SameDay || !wedding.Sessions[2].SameDay {
		t.Error("wedding: lamaran has no default date, akad and resepsi are born on Hari-H")
	}
	if wedding.LeadTimeDays != 180 {
		t.Errorf("wedding lead time = %d, want 180", wedding.LeadTimeDays)
	}

	seminar, _ := reg.Get(EventSeminar)
	if seminar.Labels.Guest != "Peserta" || seminar.Labels.GuestMenu != "Peserta & RSVP" {
		t.Errorf("seminar labels = %+v", seminar.Labels)
	}
	if len(seminar.Sessions) != 4 {
		t.Errorf("seminar has %d sessions, want 4", len(seminar.Sessions))
	}

	if aqiqah, _ := reg.Get(EventAqiqah); aqiqah.LeadTimeDays != 21 {
		t.Errorf("aqiqah lead time = %d, want 21", aqiqah.LeadTimeDays)
	}
	if _, ok := reg.Get("nope"); ok {
		t.Error("an unknown type must not be found")
	}
}

func TestTemplatesAvailableFollowsRegisteredAppliers(t *testing.T) {
	wedding, _ := DefaultRegistry().Get(EventWedding)
	other, _ := DefaultRegistry().Get(EventOther)

	none := wedding.Templates.Available(func(string) bool { return false })
	if len(none.TimelinePresets) != 0 || none.AdminDocs || none.BudgetLines || none.ShoppingList || none.Rundown || none.Playlist || none.IncludeAdicaraTasks {
		t.Errorf("without appliers nothing may be offered, got %+v", none)
	}
	if none.TimelinePresets == nil {
		t.Error("timeline_presets must serialize as [] not null")
	}

	onlyTimeline := wedding.Templates.Available(func(k string) bool { return k == TemplateTimeline })
	if strings.Join(onlyTimeline.TimelinePresets, ",") != "wedding_pdf_44,wedding_essential_22" || !onlyTimeline.IncludeAdicaraTasks || onlyTimeline.Rundown {
		t.Errorf("timeline applier only: got %+v", onlyTimeline)
	}

	all := func(string) bool { return true }
	w := wedding.Templates.Available(all)
	if !w.AdminDocs || !w.BudgetLines || !w.ShoppingList || !w.Rundown || !w.Playlist {
		t.Errorf("wedding owns every template, got %+v", w)
	}
	o := other.Templates.Available(all)
	if o.AdminDocs || o.ShoppingList || o.Playlist || !o.BudgetLines || !o.Rundown || strings.Join(o.TimelinePresets, ",") != "general" {
		t.Errorf("other: general timeline, budget and rundown only, got %+v", o)
	}
}

// TestContributorRoles checks the roles of menu 03 against TEMPLATE_EVENT chapter 4:
// every type has "other" (the default) and only its own roles, with Indonesian labels.
func TestContributorRoles(t *testing.T) {
	want := map[EventType][]string{
		EventWedding:   {"groom", "bride", "family", "sponsor", "other"},
		EventKhitanan:  {"host", "family", "other"},
		EventAqiqah:    {"host", "family", "other"},
		EventPeresmian: {"internal", "sponsor", "other"},
		EventLaunching: {"internal", "sponsor", "other"},
		EventBirthday:  {"host", "family", "other"},
		EventWisuda:    {"internal", "family", "sponsor", "other"},
		EventSeminar:   {"internal", "sponsor", "other"},
		EventOther:     {"host", "internal", "sponsor", "other"},
	}
	r := DefaultRegistry()
	for _, spec := range r.All() {
		keys := make([]string, len(spec.ContributorRoles))
		for i, role := range spec.ContributorRoles {
			keys[i] = role.Key
			if role.Label == "" || role.Label == role.Key {
				t.Errorf("%s/%s has no Indonesian label", spec.Key, role.Key)
			}
		}
		if strings.Join(keys, ",") != strings.Join(want[spec.Key], ",") {
			t.Errorf("%s roles = %v, want %v", spec.Key, keys, want[spec.Key])
		}
	}
	if len(want) != len(r.All()) {
		t.Error("the table must cover every event type")
	}
	if wedding, _ := r.Get(EventWedding); wedding.ContributorRoles[0].Label != "Mempelai Pria" {
		t.Errorf("groom label = %q", wedding.ContributorRoles[0].Label)
	}
}
