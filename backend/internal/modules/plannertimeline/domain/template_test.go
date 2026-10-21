package domain

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

// slug is the rule for the template_key of the non-wedding templates (02 BE 5.5).
func slug(s string) string {
	s = regexp.MustCompile(`[^a-z0-9]`).ReplaceAllString(strings.ToLower(s), "_")
	s = regexp.MustCompile(`_+`).ReplaceAllString(s, "_")

	return strings.Trim(s, "_")
}

func phaseByOffset(start int) int {
	switch {
	case start >= 47:
		return 1
	case start >= 35:
		return 2
	case start >= 8:
		return 3
	}

	return 4
}

var nonWedding = []pd.EventType{
	pd.EventKhitanan, pd.EventAqiqah, pd.EventPeresmian, pd.EventLaunching, pd.EventBirthday, pd.EventWisuda, pd.EventSeminar, pd.EventOther,
}

func TestWeddingSchedules(t *testing.T) {
	for _, c := range []struct {
		preset  string
		adicara bool
		want    int
		ok      bool
	}{
		{PresetWeddingPDF44, false, 44, true},
		{PresetWeddingPDF44, true, 50, true},
		{PresetWeddingEssential22, false, 22, true},
		{PresetWeddingEssential22, true, 28, true},
		{"", true, 6, true},
		{"", false, 0, false},
		{PresetGeneral, true, 0, false},
		{"nonsense", false, 0, false},
	} {
		s, ok := BuildSchedule(c.preset, pd.EventWedding, c.adicara)
		if ok != c.ok || len(s.Tasks) != c.want {
			t.Errorf("%q adicara=%v: ok=%v tasks=%d, want ok=%v tasks=%d", c.preset, c.adicara, ok, len(s.Tasks), c.ok, c.want)
		}
	}

	s, _ := BuildSchedule(PresetWeddingPDF44, pd.EventWedding, true)
	if s.Horizon != WeddingHorizon || s.PhaseName(1) != "Persiapan Awal & Lamaran" || s.PhaseName(4) != "Finalisasi & Menuju Hari-H" || s.PhaseName(5) != "" {
		t.Errorf("horizon/phase names: %d %q", s.Horizon, s.PhaseName(1))
	}
	for i, tt := range s.Tasks {
		wantOrigin := OriginPDF
		if i >= 44 {
			wantOrigin = OriginAdicara
		}
		if tt.Origin != wantOrigin {
			t.Errorf("task %d origin %s", i+1, tt.Origin)
		}
	}
}

// PDF p. 3: 41 tasks last one day and 3 (3, 4, 12) two days; the offsets only
// invent the dates, so the durations must come out of them unchanged.
func TestWeddingTemplateKeepsPDFDurationsAndOrder(t *testing.T) {
	twoDays := map[int]bool{3: true, 4: true, 12: true}
	prev := 1 << 30
	for i, tt := range weddingTasks {
		no := i + 1
		want := 1
		if twoDays[no] {
			want = 2
		}
		if got := max(tt.Start-tt.End, 1); got != want {
			t.Errorf("task %d (%s): duration %d, want %d", no, tt.Title, got, want)
		}
		if tt.Start > prev {
			t.Errorf("task %d starts before task %d", no, no-1)
		}
		prev = tt.Start
		if tt.Phase < 1 || tt.Phase > 4 {
			t.Errorf("task %d phase %d", no, tt.Phase)
		}
	}
	essential := 0
	for _, tt := range weddingTasks {
		if tt.Essential {
			essential++
		}
	}
	if len(weddingTasks) != 44 || essential != 22 {
		t.Errorf("%d tasks, %d essential", len(weddingTasks), essential)
	}
}

func TestEveryTemplateOffsetIsWithinItsHorizon(t *testing.T) {
	check := func(name string, s Schedule) {
		for _, tt := range s.Tasks {
			if tt.Start > s.Horizon || tt.Start < tt.End || tt.End < 0 {
				t.Errorf("%s: %q H-%d..H-%d outside 0..%d", name, tt.Title, tt.Start, tt.End, s.Horizon)
			}
		}
	}
	w, _ := BuildSchedule(PresetWeddingPDF44, pd.EventWedding, true)
	check("wedding", w)
	for _, et := range nonWedding {
		s, ok := BuildSchedule(PresetGeneral, et, false)
		if !ok {
			t.Fatalf("%s: no general schedule", et)
		}
		check(string(et), s)
	}
}

func TestGeneralSchedulePerType(t *testing.T) {
	for _, et := range nonWedding {
		s, ok := BuildSchedule(PresetGeneral, et, true) // include_adicara_tasks is ignored here
		want := 18
		if et == pd.EventOther {
			want = 15
		}
		if !ok || len(s.Tasks) != want {
			t.Fatalf("%s: ok=%v tasks=%d, want %d", et, ok, len(s.Tasks), want)
		}

		keys := map[string]bool{}
		general := []string{}
		prevStart, prevPhase := 1<<30, 0
		for _, tt := range s.Tasks {
			if keys[tt.Key] {
				t.Errorf("%s: duplicate key %s", et, tt.Key)
			}
			keys[tt.Key] = true
			if tt.Scope != "all" || tt.Origin != OriginAdicara {
				t.Errorf("%s: %s scope %q origin %q", et, tt.Key, tt.Scope, tt.Origin)
			}
			if tt.Start > prevStart {
				t.Errorf("%s: %q is not chronological", et, tt.Title)
			}
			if tt.Phase < prevPhase {
				t.Errorf("%s: phase goes %d -> %d at %q", et, prevPhase, tt.Phase, tt.Title)
			}
			if tt.Phase != phaseByOffset(tt.Start) {
				t.Errorf("%s: %q phase %d, the offset rule gives %d", et, tt.Title, tt.Phase, phaseByOffset(tt.Start))
			}
			prefix := string(et) + "."
			if strings.HasPrefix(tt.Key, "general.") {
				prefix = "general."
				general = append(general, tt.Title)
			}
			if tt.Key != prefix+slug(tt.Title) {
				t.Errorf("%s: key %q does not follow the slug rule (%q)", et, tt.Key, prefix+slug(tt.Title))
			}
			prevStart, prevPhase = tt.Start, tt.Phase
		}
		if len(general) != 15 {
			t.Errorf("%s: %d general tasks", et, len(general))
		}
		for i, tt := range generalTasks {
			if general[i] != tt.Title {
				t.Errorf("%s: general task %d moved: %q", et, i+1, general[i])
			}
		}
	}

	// A preset of the other family is refused.
	if _, ok := BuildSchedule(PresetGeneral, pd.EventWedding, false); ok {
		t.Error("the wedding has no general preset")
	}
	if _, ok := BuildSchedule(PresetWeddingPDF44, pd.EventSeminar, false); ok {
		t.Error("the seminar has no wedding preset")
	}
}

func TestSeminarSpeakerTaskSitsRightAfterTheCommitteeTasks(t *testing.T) {
	s, _ := BuildSchedule(PresetGeneral, pd.EventSeminar, false)
	if s.Tasks[2].Title != "Undang dan konfirmasi pembicara" || s.Tasks[2].Start != 58 || s.Tasks[2].End != 50 {
		t.Errorf("third task: %+v", s.Tasks[2])
	}
	if s.Tasks[1].Title != "Tetapkan anggaran total" {
		t.Errorf("a general task starting the same day must come first: %q", s.Tasks[1].Title)
	}
}

func TestDocTemplates(t *testing.T) {
	reg := pd.DefaultRegistry()
	want := map[pd.EventType]int{
		pd.EventWedding: 15, pd.EventKhitanan: 3, pd.EventAqiqah: 3, pd.EventPeresmian: 4, pd.EventLaunching: 4,
		pd.EventBirthday: 1, pd.EventWisuda: 3, pd.EventSeminar: 4, pd.EventOther: 0,
	}
	for et, n := range want {
		docs := DocsFor(et)
		if len(docs) != n {
			t.Errorf("%s: %d documents, want %d", et, len(docs), n)
		}
		spec, _ := reg.Get(et)
		if spec.Templates.AdminDocs != (n > 0) {
			t.Errorf("%s: the registry says admin_docs=%v for %d documents", et, spec.Templates.AdminDocs, n)
		}
		subjects := map[string]bool{}
		for _, o := range spec.AdminSubjects {
			subjects[o.Key] = true
		}
		keys := map[string]bool{}
		for _, d := range docs {
			if !subjects[d.Subject] {
				t.Errorf("%s: %q uses subject %q, which the registry does not list", et, d.Name, d.Subject)
			}
			if keys[d.Key] {
				t.Errorf("%s: duplicate key %s", et, d.Key)
			}
			keys[d.Key] = true
			if et != pd.EventWedding {
				if d.Key != string(et)+"."+slug(d.Name) || d.Quantity != "1 berkas" || d.Cost != 0 || d.Origin != OriginAdicara {
					t.Errorf("%s: %+v breaks the non-wedding rules", et, d)
				}
			} else if d.Origin != OriginPDF {
				t.Errorf("wedding doc %q origin %q", d.Name, d.Origin)
			}
		}
	}

	if got := len(SampleDocsDone()); got != 9 {
		t.Errorf("sample documents done: %d, want 9", got)
	}
	w := DocsFor(pd.EventWedding)
	if w[0].Key != KUADocKey || w[0].Cost != 600_000 || w[1].Subject != "cpp" || w[2].Subject != "cpw" || w[14].Name != "Lain Lain" {
		t.Errorf("wedding documents: %+v ... %+v", w[0], w[14])
	}
}

func TestEveryRegistryTypeHasASubjectListGuideAndLabel(t *testing.T) {
	for _, spec := range pd.DefaultRegistry().All() {
		if len(spec.AdminSubjects) == 0 || spec.AdminLabel == "" || spec.AdminGuide.Title == "" || len(spec.AdminGuide.Steps) != 4 {
			t.Errorf("%s: subjects=%d label=%q guide=%q steps=%d", spec.Key, len(spec.AdminSubjects), spec.AdminLabel, spec.AdminGuide.Title, len(spec.AdminGuide.Steps))
		}
		// Every timeline preset the registry offers must resolve.
		for _, preset := range spec.Templates.TimelinePresets {
			if _, ok := BuildSchedule(preset, spec.Key, false); !ok {
				t.Errorf("%s: preset %q does not resolve", spec.Key, preset)
			}
		}
	}
}

// ---------- the templates against the spec text ----------

const specRoot = "../../../../../docs/APP_SPEC/PLANNER_SPEC/"

func readSpec(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(specRoot + rel)
	if err != nil {
		t.Skipf("spec not available: %v", err)
	}

	return string(b)
}

// TestTemplatesMatchSpec reads the tables of 02 BE and TEMPLATE_EVENT and
// compares them with the Go data, so a change to one without the other fails.
func TestTemplatesMatchSpec(t *testing.T) {
	be := readSpec(t, "02_TIMELINE_DAN_TUGAS/BE.md")

	rows := regexp.MustCompile("(?m)^\\| (\\d+) \\| `(wedding\\.[a-z0-9_]+)` \\| (.+?) \\| `(\\w+)` \\| (.+?) \\| (\\d) \\| (\\d) hari \\| H-(\\d+) s/d H-(\\d+) \\| (✓?) \\|").FindAllStringSubmatch(be, -1)
	if len(rows) != len(weddingTasks) {
		t.Fatalf("spec has %d wedding tasks, code %d", len(rows), len(weddingTasks))
	}
	for i, r := range rows {
		tt := weddingTasks[i]
		phase, _ := strconv.Atoi(r[6])
		start, _ := strconv.Atoi(r[8])
		end, _ := strconv.Atoi(r[9])
		if tt.Key != r[2] || tt.Title != strings.TrimSpace(r[3]) || tt.Scope != r[4] || tt.Category != r[5] ||
			tt.Phase != phase || tt.Start != start || tt.End != end || tt.Essential != (r[10] == "✓") {
			t.Errorf("wedding task %s differs from the spec: %+v", r[1], tt)
		}
	}

	adicara := regexp.MustCompile("(?m)^\\| `(wedding\\.adicara_[a-z_]+)` \\| (.+?) \\| `(\\w+)` \\| (.+?) \\| H-(\\d+) s/d H-(\\d+) \\|").FindAllStringSubmatch(be, -1)
	if len(adicara) != len(weddingAdicaraTasks) {
		t.Fatalf("spec has %d Adicara tasks, code %d", len(adicara), len(weddingAdicaraTasks))
	}
	for i, r := range adicara {
		tt := weddingAdicaraTasks[i]
		start, _ := strconv.Atoi(r[5])
		end, _ := strconv.Atoi(r[6])
		if tt.Key != r[1] || tt.Title != strings.TrimSpace(r[2]) || tt.Scope != r[3] || tt.Category != strings.TrimSpace(r[4]) || tt.Phase != 4 || tt.Start != start || tt.End != end {
			t.Errorf("Adicara task %s differs from the spec: %+v", r[1], tt)
		}
	}

	docSection := be[strings.Index(be, "### 5.3"):strings.Index(be, "### 5.4")]
	docs := regexp.MustCompile("(?m)^\\| (\\d+) \\| `(wedding\\.[a-z0-9_]+)` \\| (.+?) \\| `(\\w+)` \\| (.+?) \\| ([\\d.]+) \\| ").FindAllStringSubmatch(docSection, -1)
	if len(docs) != len(weddingDocs) {
		t.Fatalf("spec has %d wedding documents, code %d", len(docs), len(weddingDocs))
	}
	for i, r := range docs {
		d := weddingDocs[i]
		cost, _ := strconv.ParseInt(strings.ReplaceAll(r[6], ".", ""), 10, 64)
		if d.Key != r[2] || d.Name != strings.TrimSpace(r[3]) || d.Subject != r[4] || d.Quantity != strings.TrimSpace(r[5]) || d.Cost != cost {
			t.Errorf("wedding document %s differs from the spec: %+v", r[2], d)
		}
	}

	te := readSpec(t, "00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md")
	general := regexp.MustCompile(`(?m)^\| (\d+) \| (.+?) \| (.+?) \| (\d) (.+?) \| H-(\d+) s/d H-(\d+) \|`).FindAllStringSubmatch(te, -1)
	if len(general) != len(generalTasks) {
		t.Fatalf("spec has %d general tasks, code %d", len(general), len(generalTasks))
	}
	for i, r := range general {
		tt := generalTasks[i]
		phase, _ := strconv.Atoi(r[4])
		start, _ := strconv.Atoi(r[6])
		end, _ := strconv.Atoi(r[7])
		if tt.Title != strings.TrimSpace(r[2]) || tt.Category != strings.TrimSpace(r[3]) || tt.Phase != phase || tt.Start != start || tt.End != end {
			t.Errorf("general task %d differs from the spec: %+v", i+1, tt)
		}
	}

	// The extra tasks and the documents of each type.
	for _, et := range nonWedding {
		section := regexp.MustCompile(`(?s)\n## \d+\. [^\n]*\(` + "`" + string(et) + "`" + `\)\n(.*?)(\n## |\n---\n)`).FindStringSubmatch(te)
		if section == nil {
			t.Fatalf("no section for %s in TEMPLATE_EVENT", et)
		}
		var specTasks, specDocs []string
		for _, line := range strings.Split(section[1], "\n") {
			switch {
			case strings.HasPrefix(line, "- **Tugas tambahan"):
				if !strings.Contains(line, "tidak ada") {
					body := line[strings.Index(line, "):")+2:]
					if !strings.Contains(line, "(di atas") {
						body = line[strings.Index(line, ":**")+3:]
					}
					for _, part := range strings.Split(strings.TrimSuffix(strings.TrimSpace(body), "."), " · ") {
						m := regexp.MustCompile(`^(.+?) H-(\d+)…H-(\d+)$`).FindStringSubmatch(strings.TrimSpace(part))
						specTasks = append(specTasks, m[1]+"|"+m[2]+"|"+m[3])
					}
				}
			case strings.HasPrefix(line, "- **Dokumen"):
				if !strings.Contains(line, "tidak ada") {
					body := line[strings.Index(line, "):")+2:]
					for _, part := range strings.Split(strings.TrimSuffix(strings.TrimSpace(body), "."), " · ") {
						m := regexp.MustCompile("^(.+) \\(`(\\w+)`\\)$").FindStringSubmatch(strings.TrimSpace(part))
						specDocs = append(specDocs, m[1]+"|"+m[2])
					}
				}
			}
		}

		var codeTasks, codeDocs []string
		for _, tt := range typeTasks[et] {
			codeTasks = append(codeTasks, tt.Title+"|"+strconv.Itoa(tt.Start)+"|"+strconv.Itoa(tt.End))
		}
		for _, d := range typeDocs[et] {
			codeDocs = append(codeDocs, d.Name+"|"+d.Subject)
		}
		if strings.Join(specTasks, ";") != strings.Join(codeTasks, ";") {
			t.Errorf("%s extra tasks differ:\n spec %v\n code %v", et, specTasks, codeTasks)
		}
		if strings.Join(specDocs, ";") != strings.Join(codeDocs, ";") {
			t.Errorf("%s documents differ:\n spec %v\n code %v", et, specDocs, codeDocs)
		}
	}
}
