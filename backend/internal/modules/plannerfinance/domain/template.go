package domain

import (
	"time"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

// MinPercentTotal is the smallest total a percentage template or preset is
// computed for. Below roughly Rp 19.000 the rounding to Rp 1.000 would make the
// remainder line negative; Rp 100.000 leaves a wide margin (03 BE 3.9).
const MinPercentTotal = 100_000

// LineTemplate is one budget line a template creates. Estimate is 0 for the
// wedding template and a share of the budget for the other types.
type LineTemplate struct {
	Scope, Key, Name, Category string
	Estimate                   int64
}

// PercentLine is one line of the template of a non-wedding event type.
type PercentLine struct {
	Key, Name, Category string
	Percent             int64
}

type presetLine struct {
	Scope, Key, Name, Category string
	Amounts                    [6]int64
}

// PresetItem is one item of a scenario preset.
type PresetItem struct {
	Scope, Key, Name, Category string
	Amount                     int64
}

// PercentPresetKey is the preset key of the percentage template of an event type, for example "seminar_pct".
func PercentPresetKey(t pd.EventType) string { return string(t) + "_pct" }

// HasBudgetTemplate reports whether the event type has a budget template.
func HasBudgetTemplate(t pd.EventType) bool {
	if t == pd.EventWedding {
		return true
	}
	_, ok := percentTemplates[t]

	return ok
}

// Presets lists the scenario presets of an event type: six fixed packages for
// a wedding, one percentage preset for every other type.
func Presets(t pd.EventType) []PresetInfo {
	if t == pd.EventWedding {
		out := make([]PresetInfo, len(weddingPresetKeys))
		for i, k := range weddingPresetKeys {
			total := weddingPresetTotals[i]
			out[i] = PresetInfo{Key: k, Kind: KindFixed, TotalTarget: &total}
		}

		return out
	}
	if _, ok := percentTemplates[t]; ok {
		return []PresetInfo{{Key: PercentPresetKey(t), Kind: KindPercent}}
	}

	return []PresetInfo{}
}

// PresetItems builds the items of a preset of the event type. A fixed preset
// has its own total (fixedTotal); a percent preset is computed from total.
// ok is false for a key the event type does not own.
func PresetItems(t pd.EventType, key string, total int64) (items []PresetItem, fixedTotal *int64, ok bool) {
	if t == pd.EventWedding {
		for i, k := range weddingPresetKeys {
			if k != key {
				continue
			}
			t := weddingPresetTotals[i]
			items = make([]PresetItem, len(weddingPresetLines))
			for j, l := range weddingPresetLines {
				items[j] = PresetItem{Scope: l.Scope, Key: l.Key, Name: l.Name, Category: l.Category, Amount: l.Amounts[i]}
			}

			return items, &t, true
		}

		return nil, nil, false
	}

	lines, found := percentTemplates[t]
	if !found || key != PercentPresetKey(t) {
		return nil, nil, false
	}
	amounts := PercentAmounts(lines, total)
	items = make([]PresetItem, len(lines))
	for i, l := range lines {
		items[i] = PresetItem{Scope: pd.AllScope, Key: l.Key, Name: l.Name, Category: l.Category, Amount: amounts[i]}
	}

	return items, nil, true
}

// PercentAmounts splits total over the lines by their percentage: every line is
// round(percent × total ÷ 100) to Rp 1.000 (half up) and the rounding
// difference goes to dana_darurat, or to the last line when the type has none,
// so the amounts always add up to total. Below MinPercentTotal every amount is 0.
func PercentAmounts(lines []PercentLine, total int64) []int64 {
	out := make([]int64, len(lines))
	if total < MinPercentTotal || len(lines) == 0 {
		return out
	}
	rest := len(lines) - 1
	for i, l := range lines {
		if l.Key == "dana_darurat" {
			rest = i
		}
	}
	var sum int64
	for i, l := range lines {
		if i == rest {
			continue
		}
		out[i] = (l.Percent*total + 50_000) / 100_000 * 1000
		sum += out[i]
	}
	out[rest] = total - sum

	return out
}

// TemplateLines are the lines the budget template of an event type creates.
// The wedding template has 14 lines of estimate 0 (Wedding_Planner.pdf p. 5-6);
// the other types get percentage lines in the general session, sized from
// targetBudget (all 0 while the budget is below MinPercentTotal).
func TemplateLines(t pd.EventType, targetBudget int64) []LineTemplate {
	if t == pd.EventWedding {
		out := make([]LineTemplate, len(weddingBudgetLines))
		copy(out, weddingBudgetLines)

		return out
	}
	lines := percentTemplates[t]
	amounts := PercentAmounts(lines, targetBudget)
	out := make([]LineTemplate, len(lines))
	for i, l := range lines {
		out[i] = LineTemplate{Scope: pd.AllScope, Key: l.Key, Name: l.Name, Category: l.Category, Estimate: amounts[i]}
	}

	return out
}

// ---------- sample project (Wedding_Planner.pdf p. 5-7) ----------

type sampleMonth struct {
	Month        int // February = 2 … December = 12, of 2026
	Groom, Bride int64
}

// SampleContributor is a savings contributor of the sample project.
type SampleContributor struct{ Name, Role string }

// SampleEntry is a deposit of the sample project; Contributor is an index into SampleContributors.
type SampleEntry struct {
	Contributor int
	Date        time.Time
	Amount      int64
}

// SampleScenario is a scenario of the sample project, laid over the presets of 03 BE 6.6.
type SampleScenario struct {
	Name      string
	PresetKey string
	Active    bool
}

// SampleContributors are Arif (groom) and Sinta (bride).
var SampleContributors = []SampleContributor{{Name: "Arif", Role: "groom"}, {Name: "Sinta", Role: "bride"}}

// SampleScenarios are the three scenarios of the sample project; the 75 Jt one is active.
var SampleScenarios = []SampleScenario{
	{Name: "Opsi Hemat 50 Jt", PresetKey: "wedding_50jt"},
	{Name: "Opsi Ideal 75 Jt", PresetKey: "wedding_75jt", Active: true},
	{Name: "Opsi 100 Jt", PresetKey: "wedding_100jt"},
}

// SampleActiveSessions are the sessions the scenarios of the sample project simulate.
var SampleActiveSessions = []string{"akad", "resepsi"}

// SamplePayments returns the realisation of the 14 template lines.
func SamplePayments() []SamplePayment { return append([]SamplePayment(nil), samplePayments...) }

// SampleEntries returns the 22 deposits: one per contributor per month, on the 1st.
func SampleEntries() []SampleEntry {
	out := make([]SampleEntry, 0, 2*len(sampleSavings))
	for _, m := range sampleSavings {
		d := time.Date(2026, time.Month(m.Month), 1, 0, 0, 0, 0, time.UTC)
		out = append(out, SampleEntry{Contributor: 0, Date: d, Amount: m.Groom}, SampleEntry{Contributor: 1, Date: d, Amount: m.Bride})
	}

	return out
}
