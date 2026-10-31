package domain

import (
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

// specRoot is the PLANNER_SPEC folder; the tests that compare code with the documents skip when it is not available
// (a build of the backend alone). Run them with -v to see that they did not skip.
const specRoot = "../../../../../docs/APP_SPEC/PLANNER_SPEC/"

func readSpec(t testing.TB, rel string) string {
	t.Helper()
	b, err := os.ReadFile(specRoot + rel)
	if err != nil {
		t.Skipf("spec not available: %v", err)
	}

	return string(b)
}

// tableRows returns the cells of the rows of the first table after marker, up to the line that is not a table row.
func tableRows(t testing.TB, doc, marker string) [][]string {
	t.Helper()
	i := strings.Index(doc, marker)
	if i < 0 {
		t.Fatalf("%q not found", marker)
	}
	var rows [][]string
	started := false
	for _, line := range strings.Split(doc[i:], "\n") {
		if !strings.HasPrefix(line, "|") {
			if started {
				break
			}

			continue
		}
		started = true
		cells := strings.Split(strings.Trim(strings.TrimSpace(line), "|"), "|")
		for k := range cells {
			cells[k] = strings.TrimSpace(cells[k])
		}
		rows = append(rows, cells)
	}

	return rows[2:] // the header and its rule
}

func dash(s string) string {
	if s == "–" || s == "-" || s == "—" {
		return ""
	}

	return s
}

func clockOf(t testing.TB, s string) string {
	t.Helper()
	_, hhmm, ok := ParseClock(s)
	if !ok {
		t.Fatalf("%q is not a time", s)
	}

	return hhmm
}

// span reads "09.00 - 09.10", "10.40 - selesai", "09.15" and "08.00–08.30".
func span(t testing.TB, s string) (start, end string) {
	t.Helper()
	parts := regexp.MustCompile(`\s*[–-]\s*`).Split(strings.TrimSpace(s), -1)
	start = clockOf(t, parts[0])
	if len(parts) == 2 && parts[1] != "selesai" {
		end = clockOf(t, parts[1])
	}

	return start, end
}

func highlights(cell string) string {
	var lines []string
	for _, h := range strings.Split(dash(cell), ";") {
		if h = strings.TrimSpace(h); h != "" {
			lines = append(lines, "- "+h)
		}
	}

	return strings.Join(lines, "\n")
}

// The wedding template is the table of 06 BE 5.2 and 5.3, row by row.
func TestWeddingTemplateMatchesSpec(t *testing.T) {
	doc := readSpec(t, "06_RUNDOWN_DAN_ACARA/BE.md")

	lamaran := tableRows(t, doc, "### 5.2")
	if len(lamaran) != 11 || len(WeddingLamaran) != 11 {
		t.Fatalf("lamaran: %d rows in the spec, %d in the code, want 11", len(lamaran), len(WeddingLamaran))
	}
	for i, r := range lamaran { // No | key | Waktu | Agenda | PIC | Keterangan | Highlights
		start, end := span(t, r[2])
		want := TemplateAgenda{
			Key: strings.Trim(r[1], "`"), ScopeKey: "lamaran", Start: start, End: end, Name: r[3], PIC: dash(r[4]), Description: dash(r[5]), Highlights: highlights(r[6]),
		}
		if !reflect.DeepEqual(WeddingLamaran[i], want) {
			t.Errorf("lamaran %d:\n code %+v\n spec %+v", i+1, WeddingLamaran[i], want)
		}
	}

	rundown := tableRows(t, doc, "### 5.3")
	if len(rundown) != 22 || len(WeddingAkadResepsi) != 22 {
		t.Fatalf("akad and resepsi: %d rows in the spec, %d in the code, want 22", len(rundown), len(WeddingAkadResepsi))
	}
	perScope := map[string]int{}
	for i, r := range rundown { // No | key | Sesi | Waktu | Agenda | PIC | Keterangan
		start, end := span(t, r[3])
		scope := strings.Trim(r[2], "`")
		perScope[scope]++
		want := TemplateAgenda{Key: strings.Trim(r[1], "`"), ScopeKey: scope, Start: start, End: end, Name: r[4], PIC: dash(r[5]), Description: dash(r[6])}
		if !reflect.DeepEqual(WeddingAkadResepsi[i], want) {
			t.Errorf("rundown %d:\n code %+v\n spec %+v", i+1, WeddingAkadResepsi[i], want)
		}
	}
	if perScope["akad"] != 8 || perScope["resepsi"] != 14 {
		t.Errorf("sessions of the second table: %v, want akad 8 and resepsi 14", perScope)
	}
	if len(AgendasFor(pd.EventWedding)) != 33 {
		t.Errorf("the wedding has %d agendas, want 33", len(AgendasFor(pd.EventWedding)))
	}
}

func TestWeddingPlaylistMatchesSpec(t *testing.T) {
	rows := tableRows(t, readSpec(t, "06_RUNDOWN_DAN_ACARA/BE.md"), "### 5.4")
	if len(rows) != 30 || len(WeddingPlaylist) != 30 {
		t.Fatalf("%d rows in the spec, %d in the code, want 30", len(rows), len(WeddingPlaylist))
	}
	for i, r := range rows { // No | key | Judul | Penyanyi | Moment tag
		want := TemplateSong{Key: strings.Trim(r[1], "`"), Title: r[2], Artist: r[3], MomentTag: r[4]}
		if WeddingPlaylist[i] != want {
			t.Errorf("song %d: code %+v, spec %+v", i+1, WeddingPlaylist[i], want)
		}
	}
	// every moment tag of the template is one the registry suggests for the wedding
	suggested := map[string]bool{}
	for _, tag := range pd.DefaultRegistry().All()[0].MomentTags {
		suggested[tag] = true
	}
	for _, s := range WeddingPlaylist {
		if !suggested[s.MomentTag] {
			t.Errorf("song %q has the moment tag %q, which the registry does not suggest", s.Title, s.MomentTag)
		}
	}
	if len(PlaylistFor(pd.EventSeminar)) != 0 || len(PlaylistFor(pd.EventWedding)) != 30 {
		t.Error("only the wedding has a playlist template")
	}
}

// The agendas of the eight other types are the tables of TEMPLATE_EVENT 7-14, with the keys "<type>.agenda_<NN>".
func TestGeneralTemplateMatchesTemplateEvent(t *testing.T) {
	doc := readSpec(t, "00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md")
	headings := map[string]string{
		"khitanan": "## 7.", "aqiqah": "## 8.", "peresmian": "## 9.", "launching": "## 10.",
		"birthday": "## 11.", "wisuda": "## 12.", "seminar": "## 13.", "other": "## 14.",
	}
	wantCount := map[string]int{"khitanan": 9, "aqiqah": 7, "peresmian": 7, "launching": 9, "birthday": 7, "wisuda": 9, "seminar": 9, "other": 3}

	total := 0
	for key, heading := range headings {
		section := doc[strings.Index(doc, "\n"+heading):]
		rows := tableRows(t, section, "**Agenda bawaan:**")
		got := GeneralAgendas[key]
		if len(rows) != wantCount[key] || len(got) != len(rows) {
			t.Errorf("%s: %d rows in the spec, %d in the code, want %d", key, len(rows), len(got), wantCount[key])
			continue
		}
		for i, r := range rows { // Sesi | Waktu | Agenda | PIC
			start, end := span(t, r[1])
			want := TemplateAgenda{Key: key + ".agenda_" + twoDigits(i+1), ScopeKey: r[0], Start: start, End: end, Name: r[2], PIC: r[3]}
			if !reflect.DeepEqual(got[i], want) {
				t.Errorf("%s row %d:\n code %+v\n spec %+v", key, i+1, got[i], want)
			}
		}
		total += len(got)
	}
	if total != 60 || len(GeneralAgendas) != 8 {
		t.Errorf("%d agendas in %d types, want 60 in 8", total, len(GeneralAgendas))
	}
}

func twoDigits(n int) string {
	if n < 10 {
		return "0" + string(rune('0'+n))
	}

	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}

// The suggested moment tags of the registry are the table of TEMPLATE_EVENT 6.
func TestMomentTagsMatchTemplateEvent(t *testing.T) {
	rows := tableRows(t, readSpec(t, "00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md"), "## 6. Moment Tag Audio")
	if len(rows) != 9 {
		t.Fatalf("%d event types in the spec, want 9", len(rows))
	}
	reg := pd.DefaultRegistry()
	for _, r := range rows { // Key | Moment tag
		spec, ok := reg.Get(pd.EventType(strings.Trim(r[0], "`")))
		if !ok {
			t.Errorf("%s is not an event type of the registry", r[0])
			continue
		}
		var want []string
		for _, tag := range strings.Split(r[1], "·") {
			want = append(want, strings.TrimSpace(tag))
		}
		if !reflect.DeepEqual(spec.MomentTags, want) {
			t.Errorf("%s: moment tags %q, spec %q", spec.Key, spec.MomentTags, want)
		}
	}
	// a registry gets its own copy of the shared table: changing the tags of one leaves the next registry alone
	spec, _ := reg.Get(pd.EventWedding)
	spec.MomentTags[0] = "diubah"
	if again, _ := pd.DefaultRegistry().Get(pd.EventWedding); again.MomentTags[0] != "Akad Khidmat" {
		t.Error("the shared table of moment tags was changed through a registry")
	}
}

// What the template promises about itself, whatever the documents say.
func TestTemplateIsConsistentWithTheRegistry(t *testing.T) {
	key := regexp.MustCompile(`^[a-z]+\.(agenda|lamaran|rundown|lagu)_[0-9]{2}$`)
	reg := pd.DefaultRegistry()
	seen := map[string]bool{}
	for _, spec := range reg.All() {
		scopes := map[string]bool{}
		for _, s := range spec.Sessions {
			scopes[s.ScopeKey] = true
		}
		agendas := AgendasFor(spec.Key)
		if len(agendas) == 0 || !spec.Templates.Rundown {
			t.Errorf("%s: %d agendas, Templates.Rundown = %v", spec.Key, len(agendas), spec.Templates.Rundown)
		}
		if (len(PlaylistFor(spec.Key)) > 0) != spec.Templates.Playlist {
			t.Errorf("%s: playlist template and registry disagree", spec.Key)
		}
		covered := map[string]bool{}
		for _, a := range agendas {
			switch {
			case !key.MatchString(a.Key) || seen[a.Key]:
				t.Errorf("%s: key %q is malformed or used twice", spec.Key, a.Key)
			case !scopes[a.ScopeKey]:
				t.Errorf("%s: %q belongs to session %q, which the registry does not seed", spec.Key, a.Key, a.ScopeKey)
			case a.Name == "" || len([]rune(a.Name)) > 255 || len([]rune(a.PIC)) > 150 || len([]rune(a.Description)) > 2000 || len([]rune(a.Highlights)) > 2000:
				t.Errorf("%s: %q is longer than the limits of the API", spec.Key, a.Key)
			}
			if _, err := ValidateTimes(a.Start, ptrOrNil(a.End), false); err != nil {
				t.Errorf("%s: %q has bad times: %v", spec.Key, a.Key, err)
			}
			seen[a.Key], covered[a.ScopeKey] = true, true
		}
		for scope := range scopes {
			if !covered[scope] {
				t.Errorf("%s: no agenda for the session %q", spec.Key, scope)
			}
		}
	}
	for _, s := range WeddingPlaylist {
		if !key.MatchString(s.Key) || seen[s.Key] || s.Title == "" || s.Artist == "" {
			t.Errorf("song %q: key %q", s.Title, s.Key)
		}
		seen[s.Key] = true
	}
	if len(seen) != 33+60+30 {
		t.Errorf("%d template keys, want %d", len(seen), 33+60+30)
	}
}

func ptrOrNil(s string) *string {
	if s == "" {
		return nil
	}

	return &s
}
