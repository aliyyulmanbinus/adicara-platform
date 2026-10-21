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

func money(t *testing.T, s string) int64 {
	t.Helper()
	n, err := strconv.ParseInt(strings.ReplaceAll(s, ".", ""), 10, 64)
	if err != nil {
		t.Fatalf("amount %q: %v", s, err)
	}

	return n
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

var nonWedding = []pd.EventType{
	pd.EventKhitanan, pd.EventAqiqah, pd.EventPeresmian, pd.EventLaunching, pd.EventBirthday, pd.EventWisuda, pd.EventSeminar, pd.EventOther,
}

// TestTemplatesMatchSpec reads the tables of 03 BE and TEMPLATE_EVENT and
// compares them with the Go data, so a change to one without the other fails.
func TestTemplatesMatchSpec(t *testing.T) {
	be := readSpec(t, "03_KEUANGAN/BE.md")

	// 6.1 the six packages
	rows := regexp.MustCompile("(?m)^\\| (\\d+) \\| `(\\w+)` \\| (.+?) \\| (.+?) \\| `(\\w+)` \\| ([\\d.]+) \\| ([\\d.]+) \\| ([\\d.]+) \\| ([\\d.]+) \\| ([\\d.]+) \\| ([\\d.]+) \\|$").
		FindAllStringSubmatch(section(t, be, "### 6.1 Enam preset skenario", "### 6.2"), -1)
	if len(rows) != len(weddingPresetLines) {
		t.Fatalf("spec has %d preset lines, code %d", len(rows), len(weddingPresetLines))
	}
	for i, r := range rows {
		l := weddingPresetLines[i]
		if r[2] != l.Key || r[3] != l.Name || r[4] != l.Category || r[5] != l.Scope {
			t.Errorf("preset line %d: spec %v, code %+v", i+1, r[2:6], l)
		}
		for j := 0; j < 6; j++ {
			if got := money(t, r[6+j]); got != l.Amounts[j] {
				t.Errorf("preset line %s, package %s: spec %d, code %d", l.Key, weddingPresetKeys[j], got, l.Amounts[j])
			}
		}
	}

	// 6.2 the 14 lines
	rows = regexp.MustCompile("(?m)^\\| (\\d+) \\| `(\\w+)` \\| `(\\w+)` \\| (.+?) \\| (.+?) \\|$").
		FindAllStringSubmatch(section(t, be, "### 6.2 Template 14 pos", "### 6.3"), -1)
	if len(rows) != len(weddingBudgetLines) {
		t.Fatalf("spec has %d template lines, code %d", len(rows), len(weddingBudgetLines))
	}
	for i, r := range rows {
		l := weddingBudgetLines[i]
		if r[2] != l.Scope || r[3] != l.Key || r[4] != l.Name || r[5] != l.Category || l.Estimate != 0 {
			t.Errorf("template line %d: spec %v, code %+v", i+1, r[2:6], l)
		}
	}

	// 6.3 the payments of the sample project
	rows = regexp.MustCompile("(?m)^\\| (\\d+) \\| (\\w+) \\| (\\w+) \\| ([\\d.]+) \\| ([\\d.]+) \\| ([\\d.]+) \\| ([\\d.]+) \\| ([\\d.]+) \\| ([\\d.]+) \\| (\\w+) \\|$").
		FindAllStringSubmatch(section(t, be, "### 6.3 Dataset Acuan PDF", "### 6.4"), -1)
	if len(rows) != len(samplePayments) {
		t.Fatalf("spec has %d sample payments, code %d", len(rows), len(samplePayments))
	}
	for i, r := range rows {
		p := samplePayments[i]
		got := []int64{p.Estimate, p.DP, p.Term1, p.Term2}
		for j, w := range []string{r[4], r[5], r[6], r[7]} {
			if money(t, w) != got[j] {
				t.Errorf("sample payment %s/%s column %d: spec %s, code %d", p.Scope, p.Key, j, w, got[j])
			}
		}
		if r[2] != p.Scope || r[3] != p.Key {
			t.Errorf("sample payment %d: spec %s/%s, code %s/%s", i+1, r[2], r[3], p.Scope, p.Key)
		}
		if paid := p.DP + p.Term1 + p.Term2; money(t, r[8]) != paid || money(t, r[9]) != p.Estimate-paid {
			t.Errorf("sample payment %s/%s: paid and remaining of the spec do not follow from its columns", p.Scope, p.Key)
		}
	}

	// 6.4 the deposits
	months := []string{"Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}
	s64 := section(t, be, "### 6.4 Tabungan", "### 6.5")
	if len(sampleSavings) != len(months) {
		t.Fatalf("code has %d months", len(sampleSavings))
	}
	for i, m := range months {
		r := regexp.MustCompile("(?m)^\\| " + m + " \\| ([\\d.]+) \\| ([\\d.]+) \\| ([\\d.]+) \\|$").FindStringSubmatch(s64)
		if r == nil {
			t.Fatalf("spec has no row for %s", m)
		}
		s := sampleSavings[i]
		if s.Month != i+2 || money(t, r[1]) != s.Groom || money(t, r[2]) != s.Bride || money(t, r[3]) != s.Groom+s.Bride {
			t.Errorf("%s: spec %v, code %+v", m, r[1:], s)
		}
	}

	// 6.6 the sample scenarios
	s66 := section(t, be, "### 6.6 Skenario Proyek Contoh", "## 7. Pengujian")
	for _, sc := range SampleScenarios {
		active, name := "–", sc.Name
		if sc.Active {
			active, name = "✔", "**"+sc.Name+"**"
		}
		row := regexp.MustCompile("(?m)^\\| " + regexp.QuoteMeta(name) + " \\| `" + sc.PresetKey + "` \\| ([\\d.]+) \\| (.) \\|$").FindStringSubmatch(s66)
		if row == nil || row[2] != active {
			t.Errorf("scenario %q (%s, active %v) is not in 6.6: %v", sc.Name, sc.PresetKey, sc.Active, row)
			continue
		}
		if _, fixed, _ := PresetItems(pd.EventWedding, sc.PresetKey, 0); fixed == nil || money(t, row[1]) != *fixed {
			t.Errorf("scenario %q: total of 6.6 is not the total of its preset", sc.Name)
		}
	}

	// TEMPLATE_EVENT 15.1 the percentages
	te := readSpec(t, "00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md")
	rows = regexp.MustCompile("(?m)^\\| `(\\w+)` \\| `(\\w+)` \\| (.+?) \\| (.+?) \\| (\\d+) \\|$").
		FindAllStringSubmatch(section(t, te, "### 15.1 Pos dan persentase", "### 15.2"), -1)
	var total int
	for _, ty := range nonWedding {
		total += len(percentTemplates[ty])
	}
	if len(rows) != total {
		t.Fatalf("spec has %d percentage lines, code %d", len(rows), total)
	}
	next := map[pd.EventType]int{}
	for _, r := range rows {
		ty := pd.EventType(r[1])
		i := next[ty]
		next[ty]++
		if i >= len(percentTemplates[ty]) {
			t.Fatalf("%s: spec has more lines than the code", ty)
		}
		l := percentTemplates[ty][i]
		pct, _ := strconv.Atoi(r[5])
		if l.Key != r[2] || l.Name != r[3] || l.Category != r[4] || l.Percent != int64(pct) {
			t.Errorf("%s line %d: spec %v, code %+v", ty, i+1, r[2:], l)
		}
	}
}

func TestWeddingPresetsSumToPackage(t *testing.T) {
	var sums [6]int64
	for _, l := range weddingPresetLines {
		for i, a := range l.Amounts {
			sums[i] += a
		}
	}
	if sums != weddingPresetTotals {
		t.Errorf("column sums %v, package totals %v", sums, weddingPresetTotals)
	}
	keys := map[string]bool{}
	for _, l := range weddingPresetLines {
		if keys[l.Key] {
			t.Errorf("duplicate key %s in a package", l.Key)
		}
		keys[l.Key] = true
	}
}

func TestPercentTemplatesAddUpTo100(t *testing.T) {
	want := map[pd.EventType]int{
		pd.EventKhitanan: 8, pd.EventAqiqah: 7, pd.EventPeresmian: 8, pd.EventLaunching: 8, pd.EventBirthday: 7,
		pd.EventWisuda: 7, pd.EventSeminar: 8, pd.EventOther: 6,
	}
	for _, ty := range nonWedding {
		var sum int64
		keys := map[string]bool{}
		for _, l := range percentTemplates[ty] {
			sum += l.Percent
			if keys[l.Key] {
				t.Errorf("%s: duplicate key %s", ty, l.Key)
			}
			keys[l.Key] = true
			if l.Category == "" || l.Name == "" || len([]rune(l.Category)) > 60 || len([]rune(l.Name)) > 150 {
				t.Errorf("%s/%s: name or category out of range", ty, l.Key)
			}
		}
		if sum != 100 || len(percentTemplates[ty]) != want[ty] {
			t.Errorf("%s: %d lines adding up to %d, want %d lines and 100", ty, len(percentTemplates[ty]), sum, want[ty])
		}
	}
}

func TestPercentAmountsAddUpAndNeverGoNegative(t *testing.T) {
	check := func(ty pd.EventType, total int64) {
		t.Helper()
		var sum int64
		for i, a := range PercentAmounts(percentTemplates[ty], total) {
			if a < 0 {
				t.Fatalf("%s total %d: line %d is negative (%d)", ty, total, i, a)
			}
			sum += a
		}
		if sum != total {
			t.Fatalf("%s total %d: the lines add up to %d", ty, total, sum)
		}
	}
	// Every total from the minimum up to Rp 400.000 (the range where rounding hurts most), then a spread of large ones.
	for _, ty := range nonWedding {
		for total := int64(MinPercentTotal); total <= 400_000; total++ {
			check(ty, total)
		}
		for _, total := range []int64{1_234_567, 33_333_333, 75_000_000, 999_999_999, 10_000_000_000, MaxMoney} {
			check(ty, total)
		}
	}

	// Below the minimum, rounding would drive dana_darurat negative: every amount is 0.
	for _, total := range []int64{0, 1, 19_000, 99_999} {
		for _, a := range PercentAmounts(percentTemplates[pd.EventSeminar], total) {
			if a != 0 {
				t.Errorf("total %d must give zeros, got %d", total, a)
			}
		}
	}

	// A known case: 75 Jt over the seminar template, every line a multiple of Rp 1.000.
	got := PercentAmounts(percentTemplates[pd.EventSeminar], 75_000_000)
	want := []int64{18_750_000, 15_000_000, 11_250_000, 9_000_000, 6_000_000, 6_000_000, 3_750_000, 5_250_000}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("seminar 75 Jt line %d = %d, want %d", i, got[i], want[i])
		}
	}
	// Rounding to Rp 1.000 is half up (25% of 102.000 = 25.500 becomes 26.000, 20% = 20.400 becomes 20.000) and the
	// difference lands on dana_darurat, the last line of the seminar template.
	odd := PercentAmounts(percentTemplates[pd.EventSeminar], 102_000)
	if odd[0] != 26_000 || odd[1] != 20_000 || odd[7] != 102_000-(26_000+20_000+15_000+12_000+8_000+8_000+5_000) {
		t.Errorf("seminar 102.000 = %v", odd)
	}
}

func TestPresets(t *testing.T) {
	w := Presets(pd.EventWedding)
	if len(w) != 6 || w[0].Key != "wedding_10jt" || w[0].Kind != KindFixed || *w[0].TotalTarget != 10_000_000 || *w[5].TotalTarget != 100_000_000 {
		t.Errorf("wedding presets = %+v", w)
	}
	for _, ty := range nonWedding {
		p := Presets(ty)
		if len(p) != 1 || p[0].Key != string(ty)+"_pct" || p[0].Kind != KindPercent || p[0].TotalTarget != nil {
			t.Errorf("%s presets = %+v", ty, p)
		}
	}
	if p := Presets("martian"); len(p) != 0 || p == nil {
		t.Errorf("an unknown type has no presets but a non-nil list: %#v", p)
	}

	items, fixed, ok := PresetItems(pd.EventWedding, "wedding_75jt", 1)
	var sum int64
	for _, it := range items {
		sum += it.Amount
	}
	if !ok || fixed == nil || *fixed != 75_000_000 || sum != 75_000_000 || len(items) != 10 {
		t.Errorf("wedding_75jt: ok=%v fixed=%v sum=%d items=%d", ok, fixed, sum, len(items))
	}
	if _, _, ok := PresetItems(pd.EventWedding, "seminar_pct", 1_000_000); ok {
		t.Error("a wedding must not accept a percentage preset")
	}
	if _, _, ok := PresetItems(pd.EventSeminar, "wedding_75jt", 1_000_000); ok {
		t.Error("a seminar must not accept a wedding package")
	}
	if _, _, ok := PresetItems(pd.EventSeminar, "khitanan_pct", 1_000_000); ok {
		t.Error("a seminar must not accept the preset of another type")
	}
	items, fixed, ok = PresetItems(pd.EventSeminar, "seminar_pct", 40_000_000)
	if !ok || fixed != nil || len(items) != 8 || items[0].Scope != pd.AllScope {
		t.Errorf("seminar_pct: ok=%v fixed=%v items=%d", ok, fixed, len(items))
	}
}

func TestTemplateLines(t *testing.T) {
	w := TemplateLines(pd.EventWedding, 75_000_000)
	if len(w) != 14 || w[0].Scope != "lamaran" || w[13].Key != "hiburan" {
		t.Fatalf("wedding lines = %+v", w)
	}
	for _, l := range w {
		if l.Estimate != 0 {
			t.Errorf("a wedding template line starts at 0, got %+v", l)
		}
	}
	// The template is a copy: changing the result does not change the data.
	w[0].Name = "changed"
	if weddingBudgetLines[0].Name == "changed" {
		t.Error("TemplateLines must return a copy")
	}

	for _, ty := range nonWedding {
		var sum int64
		for _, l := range TemplateLines(ty, 60_000_000) {
			sum += l.Estimate
			if l.Scope != pd.AllScope {
				t.Errorf("%s/%s is in %s", ty, l.Key, l.Scope)
			}
		}
		if sum != 60_000_000 {
			t.Errorf("%s: estimates add up to %d, want the budget", ty, sum)
		}
		for _, l := range TemplateLines(ty, 0) {
			if l.Estimate != 0 {
				t.Errorf("%s: a project without budget must get zeros", ty)
			}
		}
		if !HasBudgetTemplate(ty) {
			t.Errorf("%s has a budget template", ty)
		}
	}
	if HasBudgetTemplate("martian") || len(TemplateLines("martian", 1_000_000)) != 0 {
		t.Error("an unknown type has no template")
	}
}

func TestSampleDataset(t *testing.T) {
	var estimate, paid int64
	for _, p := range SamplePayments() {
		estimate += p.Estimate
		paid += p.DP + p.Term1 + p.Term2
	}
	if estimate != 74_150_000 || paid != 33_550_000 || len(SamplePayments()) != 14 {
		t.Errorf("sample payments: estimate %d, paid %d", estimate, paid)
	}

	entries := SampleEntries()
	var perContributor [2]int64
	for _, e := range entries {
		perContributor[e.Contributor] += e.Amount
		if e.Date.Day() != 1 || e.Date.Year() != 2026 {
			t.Errorf("deposit date %v", e.Date)
		}
	}
	if len(entries) != 22 || perContributor != [2]int64{30_000_000, 17_500_000} {
		t.Errorf("%d deposits, per contributor %v", len(entries), perContributor)
	}
	if len(SampleContributors) != 2 || SampleContributors[0].Role != "groom" || SampleContributors[1].Role != "bride" {
		t.Errorf("contributors = %+v", SampleContributors)
	}
	active := 0
	for _, s := range SampleScenarios {
		if s.Active {
			active++
		}
		if _, _, ok := PresetItems(pd.EventWedding, s.PresetKey, 0); !ok {
			t.Errorf("sample scenario %q uses an unknown preset", s.Name)
		}
	}
	if active != 1 || len(SampleScenarios) != 3 {
		t.Errorf("%d scenarios, %d active", len(SampleScenarios), active)
	}

	// Every payment lands on a template line of the sample, by (session, key).
	lines := map[string]bool{}
	for _, l := range weddingBudgetLines {
		lines[l.Scope+"/"+l.Key] = true
	}
	for _, p := range SamplePayments() {
		if !lines[p.Scope+"/"+p.Key] {
			t.Errorf("sample payment %s/%s matches no template line", p.Scope, p.Key)
		}
	}
}

func TestUnitEstimate(t *testing.T) {
	for _, c := range []struct {
		qx100, price, want int64
		ok                 bool
	}{
		{25_050, 85_000, 21_292_500, true},             // 250,5 × 85.000
		{1, 150, 2, true},                              // 0,01 × 150 = 1,5 → 2 (half up)
		{1, 149, 1, true},                              // 1,49 → 1
		{100, 0, 0, true},                              // free
		{100, MaxMoney, MaxMoney, true},                // exactly 10^13
		{101, MaxMoney, 0, false},                      // 10^13 × 1,01
		{MaxQuantityX100, MaxMoney, 0, false},          // 10^13 × 10^6: 10^19, far over int64
		{MaxQuantityX100, 9_223_372_036_854, 0, false}, // product over int64 inside big.Int, answer over the cap
		{MaxQuantityX100, 100_000_000_000, 0, false},   // 10^6 × 10^11 = 10^17 > 10^13
		{MaxQuantityX100, 10_000_000, MaxMoney, true},  // 10^6 × 10^7 = 10^13
	} {
		got, ok := UnitEstimate(c.qx100, c.price)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("UnitEstimate(%d, %d) = %d, %v; want %d, %v", c.qx100, c.price, got, ok, c.want, c.ok)
		}
	}
}

func TestQuantityX100(t *testing.T) {
	for _, c := range []struct {
		in   float64
		want int64
		ok   bool
	}{
		{1, 100, true}, {0.01, 1, true}, {250.5, 25_050, true}, {0.07, 7, true}, {1_000_000, 100_000_000, true}, {19.99, 1_999, true},
		{1.234, 0, false}, {0.005, 0, false}, {-1, 0, false}, {3_000_000, 0, false},
	} {
		got, ok := QuantityX100(c.in)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("QuantityX100(%v) = %d, %v; want %d, %v", c.in, got, ok, c.want, c.ok)
		}
	}
	for _, bad := range []float64{nan(), inf()} {
		if _, ok := QuantityX100(bad); ok {
			t.Errorf("QuantityX100(%v) must be refused", bad)
		}
	}
	if QuantityFloat(25_050) != 250.5 {
		t.Error("QuantityFloat")
	}
}

func TestCeilDiv(t *testing.T) {
	for _, c := range [][3]int64{{27_500_000, 6_000, 4584}, {7, 7, 1}, {8, 7, 2}, {0, 5, 0}, {-3, 5, 0}} {
		if got := CeilDiv(c[0], c[1]); got != c[2] {
			t.Errorf("CeilDiv(%d, %d) = %d, want %d", c[0], c[1], got, c[2])
		}
	}
}

func nan() float64 { z := 0.0; return z / z }
func inf() float64 { z := 0.0; return 1 / z }
