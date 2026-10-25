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
// (a build of the backend alone).
const specRoot = "../../../../../docs/APP_SPEC/PLANNER_SPEC/"

func readSpec(t testing.TB, rel string) string {
	t.Helper()
	b, err := os.ReadFile(specRoot + rel)
	if err != nil {
		t.Skipf("spec not available: %v", err)
	}

	return string(b)
}

const templateEvent = "00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md"

var rowKey = regexp.MustCompile("^\\| `([a-z]+)` \\|")

func options(cell string) []Option {
	cell = strings.TrimSpace(cell)
	if cell == "–" || cell == "" {
		return nil
	}
	var out []Option
	for _, part := range strings.Split(cell, ";") {
		part = strings.Trim(strings.TrimSpace(part), "`")
		kv := strings.SplitN(part, "=", 2)
		out = append(out, Option{Key: strings.Trim(kv[0], "` "), Label: strings.TrimSpace(kv[1])})
	}
	return out
}

// specsFromTemplateEvent membaca blok 4.2 TEMPLATE_EVENT (sumber tunggal label).
func specsFromTemplateEvent(t testing.TB) map[string]GuestSpec {
	doc := readSpec(t, templateEvent)
	i := strings.Index(doc, "### 4.2 ")
	j := strings.Index(doc, "## 5. ")
	if i < 0 || j < i {
		t.Fatal("section 4.2 not found")
	}
	out := map[string]GuestSpec{}
	for _, line := range strings.Split(doc[i:j], "\n") {
		m := rowKey.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		cells := strings.Split(line, "|")
		out[m[1]] = GuestSpec{Sides: options(cells[2]), Categories: options(cells[3])}
	}
	return out
}

func TestRegistryHasNineTypesWithTheDocumentedShape(t *testing.T) {
	specs := specsFromTemplateEvent(t)
	if len(specs) != 9 {
		t.Fatalf("types = %d, want 9", len(specs))
	}
	sides := map[string]int{"wedding": 2, "khitanan": 2, "aqiqah": 2, "peresmian": 2, "launching": 2, "birthday": 0, "wisuda": 0, "seminar": 2, "other": 0}
	cats := map[string]int{"wedding": 6, "khitanan": 5, "aqiqah": 4, "peresmian": 6, "launching": 6, "birthday": 5, "wisuda": 5, "seminar": 5, "other": 2}
	for k, s := range specs {
		if len(s.Sides) != sides[k] || len(s.Categories) != cats[k] {
			t.Errorf("%s: sides=%d categories=%d, want %d/%d", k, len(s.Sides), len(s.Categories), sides[k], cats[k])
		}
		for _, c := range s.Categories {
			if c.Key == FallbackCategory {
				t.Errorf("%s: the fallback key must not be a registry category", k)
			}
		}
	}
}

// Bab 4 (kunci) dan 4.2 (label) harus memuat kunci yang sama.
func TestKeysOfSection4MatchTheLabelsOfSection42(t *testing.T) {
	doc := readSpec(t, templateEvent)
	i := strings.Index(doc, "## 4. Peran, Sisi Tamu")
	j := strings.Index(doc, "### 4.1 ")
	keyRe := regexp.MustCompile("`([a-z_]+)`")
	specs := specsFromTemplateEvent(t)
	for _, line := range strings.Split(doc[i:j], "\n") {
		m := rowKey.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		cells := strings.Split(line, "|")
		side := keyRe.FindAllStringSubmatch(strings.SplitN(cells[2], "Pihak", 2)[0], -1)
		_ = side
		var wantSides, wantCats []string
		for _, k := range keyRe.FindAllStringSubmatch(cells[2], -1) {
			wantSides = append(wantSides, k[1])
		}
		cat := strings.SplitN(cells[3], "*(", 2)[0]
		for _, k := range keyRe.FindAllStringSubmatch(cat, -1) {
			wantCats = append(wantCats, k[1])
		}
		got := specs[m[1]]
		var gs, gc []string
		for _, o := range got.Sides {
			gs = append(gs, o.Key)
		}
		for _, o := range got.Categories {
			gc = append(gc, o.Key)
		}
		if strings.Join(wantSides, ",") != strings.Join(gs, ",") || strings.Join(wantCats, ",") != strings.Join(gc, ",") {
			t.Errorf("%s: bab 4 sides=%v categories=%v; 4.2 sides=%v categories=%v", m[1], wantSides, wantCats, gs, gc)
		}
	}
}

const readme = "05_BUKU_TAMU/README.md"

// Tabel 9 jenis di README harus sama dengan TEMPLATE_EVENT 4.2 (kunci sisi dan kategori, urutan, sebutan tamu).
func TestReadmeNineTypeTableMatchesTemplateEvent(t *testing.T) {
	doc := readSpec(t, readme)
	i := strings.Index(doc, "## 5. Penerapan 9 Jenis Acara")
	j := strings.Index(doc, "## 6. Integrasi")
	if i < 0 || j < i {
		t.Fatal("README section 5 not found")
	}
	order := []string{"wedding", "khitanan", "aqiqah", "peresmian", "launching", "birthday", "wisuda", "seminar", "other"}
	specs := specsFromTemplateEvent(t)
	keyRe := regexp.MustCompile("`([a-z_]+)`")
	keys := func(cell string) string {
		var out []string
		for _, m := range keyRe.FindAllStringSubmatch(cell, -1) {
			out = append(out, m[1])
		}
		return strings.Join(out, ",")
	}
	join := func(o []Option) string {
		var out []string
		for _, x := range o {
			out = append(out, x.Key)
		}
		return strings.Join(out, ",")
	}
	n := 0
	for _, line := range strings.Split(doc[i:j], "\n") {
		if !strings.HasPrefix(line, "| ") || strings.HasPrefix(line, "| Jenis") || strings.HasPrefix(line, "|---") {
			continue
		}
		cells := strings.Split(line, "|")
		typ := order[n]
		n++
		word := "Tamu"
		if typ == "seminar" {
			word = "Peserta"
		}
		if strings.TrimSpace(cells[2]) != word {
			t.Errorf("%s: sebutan %q, want %q", typ, strings.TrimSpace(cells[2]), word)
		}
		if got, want := keys(cells[3]), join(specs[typ].Sides); got != want {
			t.Errorf("%s: sisi %q, want %q", typ, got, want)
		}
		if got, want := keys(cells[4]), join(specs[typ].Categories); got != want {
			t.Errorf("%s: kategori %q, want %q", typ, got, want)
		}
	}
	if n != 9 {
		t.Errorf("rows = %d, want 9", n)
	}
}

// The registry of the code is the table of TEMPLATE_EVENT 4.2: keys, labels and order, for all nine types.
func TestEventTypesCarryTheGuestRegistry(t *testing.T) {
	specs := specsFromTemplateEvent(t)
	all := pd.DefaultRegistry().All()
	if len(all) != 9 || len(specs) != 9 {
		t.Fatalf("registry has %d types, the document %d", len(all), len(specs))
	}
	for _, et := range all {
		want := specs[string(et.Key)]
		if !sameOptions(et.GuestSides, want.Sides) || !sameOptions(et.GuestCategories, want.Categories) {
			t.Errorf("%s: code sides %v categories %v; document sides %v categories %v", et.Key, et.GuestSides, et.GuestCategories, want.Sides, want.Categories)
		}
		if et.GuestSides == nil || et.GuestCategories == nil {
			t.Errorf("%s: GET /event-types must send [] and not null", et.Key)
		}
	}
}

func sameOptions(a, b []Option) bool {
	return len(a) == len(b) && (len(a) == 0 || reflect.DeepEqual(a, b))
}

// The registry hands out copies: a caller that edits what it got does not change the next project.
func TestGuestRegistryIsNotShared(t *testing.T) {
	first, _ := pd.DefaultRegistry().Get(pd.EventWedding)
	first.GuestSides[0].Label = "diubah"
	first.GuestCategories[0].Key = "diubah"
	second, _ := pd.DefaultRegistry().Get(pd.EventWedding)
	if second.GuestSides[0].Label != "Pihak Pria (CPP)" || second.GuestCategories[0].Key != "teman" {
		t.Errorf("the registry was changed through a copy: %v %v", second.GuestSides, second.GuestCategories)
	}
}

// The README table is the registry too (kunci sisi dan kategori, urutan, sebutan tamu).
func TestReadmeNineTypeTableMatchesRegistry(t *testing.T) {
	doc := readSpec(t, readme)
	i := strings.Index(doc, "## 5. Penerapan 9 Jenis Acara")
	j := strings.Index(doc, "## 6. Integrasi")
	if i < 0 || j < i {
		t.Fatal("README section 5 not found")
	}
	keyRe := regexp.MustCompile("`([a-z_]+)`")
	keys := func(cell string) string {
		var out []string
		for _, m := range keyRe.FindAllStringSubmatch(cell, -1) {
			out = append(out, m[1])
		}
		return strings.Join(out, ",")
	}
	join := func(o []pd.Option) string {
		var out []string
		for _, x := range o {
			out = append(out, x.Key)
		}
		return strings.Join(out, ",")
	}
	all := pd.DefaultRegistry().All()
	n := 0
	for _, line := range strings.Split(doc[i:j], "\n") {
		if !strings.HasPrefix(line, "| ") || strings.HasPrefix(line, "| Jenis") || strings.HasPrefix(line, "|---") {
			continue
		}
		cells := strings.Split(line, "|")
		et := all[n]
		n++
		if got := strings.TrimSpace(cells[2]); got != et.Labels.Guest {
			t.Errorf("%s: sebutan %q, registry %q", et.Key, got, et.Labels.Guest)
		}
		if got := keys(cells[3]); got != join(et.GuestSides) {
			t.Errorf("%s: sisi %q, registry %q", et.Key, got, join(et.GuestSides))
		}
		if got := keys(cells[4]); got != join(et.GuestCategories) {
			t.Errorf("%s: kategori %q, registry %q", et.Key, got, join(et.GuestCategories))
		}
	}
	if n != 9 {
		t.Errorf("rows = %d, want 9", n)
	}
}
