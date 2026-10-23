package domain

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

const specRoot = "../../../../../docs/APP_SPEC/PLANNER_SPEC/"

func readSpec(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(specRoot + rel)
	if err != nil {
		t.Skipf("spec not available: %v", err)
	}

	return string(b)
}

func section(t *testing.T, text, from, to string) string {
	t.Helper()
	i := strings.Index(text, from)
	if i < 0 {
		t.Fatalf("section %q not found", from)
	}
	rest := text[i+len(from):]
	if j := strings.Index(rest, to); j >= 0 {
		rest = rest[:j]
	}

	return rest
}

func money(t *testing.T, s string) int64 {
	t.Helper()
	n, err := strconv.ParseInt(strings.ReplaceAll(s, ".", ""), 10, 64)
	if err != nil {
		t.Fatalf("amount %q: %v", s, err)
	}

	return n
}

// TestWeddingShoppingMatchesSpec compares the 37 items with the table of 04 BE 5.3 (which was compared with the PDF), so a
// change to one without the other fails.
func TestWeddingShoppingMatchesSpec(t *testing.T) {
	be := readSpec(t, "04_VENDOR_DAN_BELANJA/BE.md")
	rows := regexp.MustCompile(`(?m)^\| (\d+) \| (.+?) \| (.+?) \| ([\d.]+) \| (selesai|proses) \|$`).
		FindAllStringSubmatch(section(t, be, "### 5.3 Daftar belanja", "Subtotal kategori"), -1)
	if len(rows) != 37 || len(WeddingShopping) != 37 {
		t.Fatalf("spec has %d items, code %d; want 37", len(rows), len(WeddingShopping))
	}

	var total, done int64
	seen := map[string]bool{}
	for i, r := range rows {
		it := WeddingShopping[i]
		if r[2] != it.Category || r[3] != it.Name || money(t, r[4]) != it.Price || (r[5] == "selesai") != it.Done {
			t.Errorf("item %d: spec %v, code %+v", i+1, r[2:], it)
		}
		total += it.Price
		if it.Done {
			done += it.Price
		}
		key := it.Key()
		if seen[key] || len(key) > 44 || !strings.HasPrefix(key, "wedding.seserahan_") || strings.Trim(strings.TrimPrefix(key, "wedding.seserahan_"), "abcdefghijklmnopqrstuvwxyz0123456789_") != "" {
			t.Errorf("template key %q is duplicated, too long or not a plain key", key)
		}
		seen[key] = true
	}
	if total != 4_578_500 || done != 2_063_800 { // PDF p. 9-10: total, and the 16 finished items
		t.Errorf("total %d, done %d", total, done)
	}

	// the categories of the template are the registry's shopping categories of the wedding
	spec, _ := pd.DefaultRegistry().Get(pd.EventWedding)
	for _, it := range WeddingShopping {
		found := false
		for _, c := range spec.ShoppingCategories {
			found = found || c == it.Category
		}
		if !found {
			t.Errorf("%q is not a shopping category of the wedding", it.Category)
		}
	}
}

// TestSampleVendorsMatchSpec compares the sample vendors with the table of 04 BE 5.2.
func TestSampleVendorsMatchSpec(t *testing.T) {
	be := readSpec(t, "04_VENDOR_DAN_BELANJA/BE.md")
	rows := regexp.MustCompile("(?m)^\\| (\\w+) \\| (.+?) \\| (0812\\d{8}) \\| ([\\d.]+) \\| (fix|rencana|negosiasi) \\|").
		FindAllStringSubmatch(section(t, be, "### 5.2 Vendor Proyek Contoh", "### 5.3"), -1)
	if len(rows) != len(SampleVendors) || len(rows) != 10 {
		t.Fatalf("spec has %d vendors, code %d", len(rows), len(SampleVendors))
	}
	fixed := 0
	for i, r := range rows {
		sv := SampleVendors[i]
		if r[1] != sv.CategoryKey || r[2] != sv.Name || r[3] != sv.Phone || money(t, r[4]) != sv.Price || r[5] != string(sv.Status) {
			t.Errorf("vendor %d: spec %v, code %+v", i+1, r[1:], sv)
		}
		if sv.Status == VendorFixed {
			fixed++
		}
	}
	if fixed != 7 {
		t.Errorf("%d fixed vendors, the PDF has 7", fixed)
	}
}

var typeHeading = regexp.MustCompile("(?m)^## \\d+\\. .*\\(`(\\w+)`\\)$")

// typeSection returns the text of the chapter of an event type (TEMPLATE_EVENT 7-14).
func typeSection(t *testing.T, doc string, et pd.EventType) string {
	t.Helper()
	locs := typeHeading.FindAllStringSubmatchIndex(doc, -1)
	for i, l := range locs {
		if doc[l[2]:l[3]] != string(et) {
			continue
		}
		end := len(doc)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		if j := strings.Index(doc[l[1]:end], "\n---"); j >= 0 {
			end = l[1] + j
		}

		return doc[l[1]:end]
	}
	t.Fatalf("no chapter for %s", et)

	return ""
}

func bullet(t *testing.T, sec, name string) string {
	t.Helper()
	m := regexp.MustCompile(`(?m)^- \*\*` + regexp.QuoteMeta(name) + `:\*\* (.+)$`).FindStringSubmatch(sec)
	if m == nil {
		t.Fatalf("bullet %q not found", name)
	}

	return strings.TrimSuffix(strings.TrimSpace(m[1]), ".")
}

// TestVendorCategoriesMatchTemplateEvent keeps the registry in step with TEMPLATE_EVENT: the category keys (and the labels the
// document gives) of chapters 7-14, the budget_item_keys of 15.2, the item_keys of 15.1, the shopping categories and the
// rollup lines of 15.4. The wedding is compared with 04 BE 5.1.
func TestVendorCategoriesMatchTemplateEvent(t *testing.T) {
	doc := readSpec(t, "00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md")
	reg := pd.DefaultRegistry()

	// 15.1: item_key per type; 15.2: budget_item_keys per (type, category); 15.4: rollup per type
	lineKeys := map[string]map[string]bool{}
	for _, r := range regexp.MustCompile("(?m)^\\| `(\\w+)` \\| `(\\w+)` \\| .+ \\| .+ \\| \\d+ \\|$").
		FindAllStringSubmatch(section(t, doc, "### 15.1 Pos dan persentase", "### 15.2"), -1) {
		if lineKeys[r[1]] == nil {
			lineKeys[r[1]] = map[string]bool{}
		}
		lineKeys[r[1]][r[2]] = true
	}
	mapped := map[string]map[string]string{}
	for _, r := range regexp.MustCompile("(?m)^\\| `(\\w+)` \\| `(\\w+)` \\| `(\\w+)` \\|$").
		FindAllStringSubmatch(section(t, doc, "### 15.2 Pemetaan", "### 15.3"), -1) {
		if mapped[r[1]] == nil {
			mapped[r[1]] = map[string]string{}
		}
		mapped[r[1]][r[2]] = r[3]
	}
	rollups := map[string][3]string{}
	for _, r := range regexp.MustCompile("(?m)^\\| `(\\w+)` \\| `(\\w+)` \\| (.+?) \\| (.+?) \\| (.+?) \\|$").
		FindAllStringSubmatch(section(t, doc, "### 15.4 Pos gulungan belanja", "Kunci, nama, dan kategori"), -1) {
		rollups[r[1]] = [3]string{r[2], r[3], r[4]}
	}

	cell := regexp.MustCompile("`(\\w+)`(?: ([^·]+?))?(?: · |$)")
	for _, et := range reg.All() {
		spec := et.VendorCategories
		if len(spec) == 0 || len(et.ShoppingCategories) == 0 {
			t.Fatalf("%s: no vendor or shopping categories", et.Key)
		}

		// shopping categories and rollup line
		r, ok := rollups[string(et.Key)]
		if !ok || et.ShoppingRollup.Key != r[0] || et.ShoppingRollup.Name != r[1] || et.ShoppingRollup.Category != r[2] {
			t.Errorf("%s: rollup %+v, spec %v", et.Key, et.ShoppingRollup, r)
		}
		if et.Key != pd.EventWedding && et.Key != pd.EventOther && !lineKeys[string(et.Key)][et.ShoppingRollup.Key] {
			t.Errorf("%s: rollup key %q is not a line of 15.1", et.Key, et.ShoppingRollup.Key)
		}

		if et.Key == pd.EventWedding {
			continue // 04 BE 5.1 below
		}
		sec := typeSection(t, doc, et.Key)

		var keys []string
		for _, m := range cell.FindAllStringSubmatch(bullet(t, sec, "Kategori vendor"), -1) {
			keys = append(keys, m[1])
			i := len(keys) - 1
			if i >= len(spec) || spec[i].Key != m[1] || (strings.TrimSpace(m[2]) != "" && spec[i].Label != strings.TrimSpace(m[2])) {
				t.Errorf("%s: vendor category %d: spec %q %q, code %+v", et.Key, i+1, m[1], m[2], spec)
			}
		}
		if len(keys) != len(spec) {
			t.Errorf("%s: spec has %d vendor categories, code %d", et.Key, len(keys), len(spec))
		}
		for _, c := range spec {
			want := mapped[string(et.Key)][c.Key]
			if want == "" || len(c.BudgetItemKeys) != 1 || c.BudgetItemKeys[0] != want || !lineKeys[string(et.Key)][want] {
				t.Errorf("%s/%s: budget_item_keys %v, 15.2 says %q (must be a line of 15.1)", et.Key, c.Key, c.BudgetItemKeys, want)
			}
		}

		if want := strings.Split(bullet(t, sec, "Kategori belanja"), " · "); strings.Join(want, "|") != strings.Join(et.ShoppingCategories, "|") {
			t.Errorf("%s: shopping categories %v, spec %v", et.Key, et.ShoppingCategories, want)
		}
	}

	// the wedding: 04 BE 5.1 (ten categories, the keys of the 14 PDF lines) and the six shopping categories
	be := readSpec(t, "04_VENDOR_DAN_BELANJA/BE.md")
	var rows [][]string
	for _, r := range regexp.MustCompile("(?m)^\\| `(\\w+)` \\| (.+?) \\| (–|`\\w+`) \\| .+ \\|$").
		FindAllStringSubmatch(section(t, be, "### 5.1 Kategori vendor pernikahan", "**Kategori belanja pernikahan**"), -1) {
		if r[1] != "category_key" { // the header row
			rows = append(rows, r)
		}
	}
	wedding, _ := reg.Get(pd.EventWedding)
	if len(rows) != 10 || len(wedding.VendorCategories) != 10 {
		t.Fatalf("wedding: spec %d categories, code %d", len(rows), len(wedding.VendorCategories))
	}
	for i, r := range rows {
		c := wedding.VendorCategories[i]
		label := strings.TrimSpace(strings.SplitN(r[2], " (PDF", 2)[0])
		want := strings.Trim(r[3], "`–")
		var got string
		if len(c.BudgetItemKeys) > 0 {
			got = c.BudgetItemKeys[0]
		}
		if c.Key != r[1] || c.Label != label || got != want {
			t.Errorf("wedding category %d: spec %v, code %+v", i+1, r[1:], c)
		}
	}
}

// TestEveryEventTypeHasItsSharedGroups: the groups TEMPLATE_EVENT and 04 BE 2.1 name (wedding: venue_dekor, rias_busana,
// hiburan; peresmian venue_dekor; launching produksi_av; birthday katering_kue; seminar av_streaming).
func TestEveryEventTypeHasItsSharedGroups(t *testing.T) {
	want := map[pd.EventType][]string{
		pd.EventWedding: {"hiburan", "rias_busana", "venue_dekor"}, pd.EventKhitanan: nil, pd.EventAqiqah: nil,
		pd.EventPeresmian: {"venue_dekor"}, pd.EventLaunching: {"produksi_av"}, pd.EventBirthday: {"katering_kue"},
		pd.EventWisuda: nil, pd.EventSeminar: {"av_streaming"}, pd.EventOther: nil,
	}
	for _, et := range pd.DefaultRegistry().All() {
		var cats []CategorySpec
		for _, c := range et.VendorCategories {
			cats = append(cats, CategorySpec{Key: c.Key, Label: c.Label, BudgetItemKeys: c.BudgetItemKeys})
		}
		var got []string
		for k := range SharedKeys(cats) {
			got = append(got, k)
		}
		sortStrings(got)
		if strings.Join(got, ",") != strings.Join(want[et.Key], ",") {
			t.Errorf("%s: shared keys %v, want %v", et.Key, got, want[et.Key])
		}
	}
}

// TestReadmeNineTypeTableMatchesRegistry compares the table "Penerapan 9 Jenis Acara" of the menu README (the page a
// reader sees first) with the registry: category counts, shared groups, shopping categories, template and rollup line.
func TestReadmeNineTypeTableMatchesRegistry(t *testing.T) {
	doc := readSpec(t, "04_VENDOR_DAN_BELANJA/README.md")
	rows := regexp.MustCompile("(?m)^\\| \\S+ (.+?) \\| (\\d+) \\| (.+?) \\| (\\d+) \\| (.+?) \\| `(\\w+)`.*\\|$").
		FindAllStringSubmatch(section(t, doc, "## 4. Penerapan 9 Jenis Acara", "## 5."), -1)
	labels := map[string]pd.EventType{}
	for _, et := range pd.DefaultRegistry().All() {
		labels[et.Label] = et.Key
	}
	if len(rows) != len(labels) {
		t.Fatalf("README has %d rows, the registry %d types", len(rows), len(labels))
	}
	for _, r := range rows {
		key, ok := labels[strings.TrimSpace(r[1])]
		if !ok {
			t.Errorf("README row %q is not an event type label", r[1])

			continue
		}
		et, _ := pd.DefaultRegistry().Get(key)
		var cats []CategorySpec
		for _, c := range et.VendorCategories {
			cats = append(cats, CategorySpec{Key: c.Key, Label: c.Label, BudgetItemKeys: c.BudgetItemKeys})
		}
		var shared []string
		for k := range SharedKeys(cats) {
			shared = append(shared, k)
		}
		sortStrings(shared)
		var docShared []string
		for _, m := range regexp.MustCompile("`(\\w+)`").FindAllStringSubmatch(r[3], -1) {
			docShared = append(docShared, m[1])
		}
		sortStrings(docShared)
		hasTemplate := strings.TrimSpace(r[5]) != "–"
		if r[2] != strconv.Itoa(len(et.VendorCategories)) || strings.Join(docShared, ",") != strings.Join(shared, ",") ||
			r[4] != strconv.Itoa(len(et.ShoppingCategories)) || hasTemplate != et.Templates.ShoppingList || r[6] != et.ShoppingRollup.Key {
			t.Errorf("%s: README %v; registry: %d categories, shared %v, %d shopping categories, template %v, rollup %s",
				key, r[2:], len(et.VendorCategories), shared, len(et.ShoppingCategories), et.Templates.ShoppingList, et.ShoppingRollup.Key)
		}
	}
}

func sortStrings(s []string) {
	for i := range s {
		for j := i + 1; j < len(s); j++ {
			if s[j] < s[i] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}
