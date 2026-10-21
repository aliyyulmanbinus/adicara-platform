package usecase

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerfinance/domain"
)

func ptr[T any](v T) *T { return &v }

func date(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func wantAPIError(t *testing.T, err error, status int, code, message string) {
	t.Helper()
	ae, ok := err.(*apierr.Error)
	if !ok {
		t.Fatalf("error = %v (%T), want %d %s %q", err, err, status, code, message)
	}
	if ae.Status != status || ae.Code != code || (message != "" && ae.Message != message) {
		t.Fatalf("error = %d %s %q, want %d %s %q", ae.Status, ae.Code, ae.Message, status, code, message)
	}
}

func item(scope, key, name string, cost int64) domain.ScenarioItem {
	it := domain.ScenarioItem{ID: uuid.New(), SubEventScope: scope, ItemName: name, Category: "Umum", CostAmount: cost, Included: true}
	if key != "" {
		it.ItemKey = ptr(key)
	}

	return it
}

// ---------- Smart Adjust ----------

func TestSmartAdjustPlanDecidesPerPairedLine(t *testing.T) {
	line := func() *uuid.UUID { id := uuid.New(); return &id }
	rows := []domain.AdjustRow{
		{Item: item("resepsi", "catering", "Catering", 30_000_000), BudgetID: line(), From: 0},                // update
		{Item: item("resepsi", "mahar", "Mahar", 5_000_000), BudgetID: line(), From: 5_000_000},               // same: not an update
		{Item: item("resepsi", "seserahan", "Seserahan", 3_500_000), BudgetID: line(), From: 0, Rollup: true}, // skipped: rollup
		{Item: item("resepsi", "hiburan", "Hiburan", 3_400_000), BudgetID: line(), From: 0, Unit: true},       // skipped: unit pricing
		{Item: item("all", "dana_darurat", "Dana darurat", 1_000_000)},                                        // create
		{Item: item("resepsi", "dokumentasi", "Dokumentasi", 3_000_000), BudgetID: line(), From: 4_000_000},   // update downwards
	}

	p := smartAdjustPlan(rows, 12, 20_000_000)
	r := p.report
	if r.Mode != domain.ApplySmartAdjust || len(r.Updated) != 2 || len(r.Created) != 1 || len(r.Skipped) != 2 || r.Deleted != nil || r.NewTargetBudget != nil {
		t.Fatalf("report = %+v", r)
	}
	if r.Updated[0].From != 0 || r.Updated[0].To != 30_000_000 || r.Updated[1].From != 4_000_000 || r.Updated[1].To != 3_000_000 {
		t.Errorf("updated = %+v", r.Updated)
	}
	if r.Skipped[0].Reason != domain.SkipRollup || r.Skipped[1].Reason != domain.SkipUnitPricing || r.Skipped[0].ItemKey == nil || *r.Skipped[0].ItemKey != "seserahan" {
		t.Errorf("skipped = %+v", r.Skipped)
	}
	// 12 lines, 5 are paired (the new item has none): 7 are left alone.
	if r.UntouchedCount != 12-5 {
		t.Errorf("untouched = %d", r.UntouchedCount)
	}
	// 20.000.000 now, +30.000.000 (update), −1.000.000 (update), +1.000.000 (created).
	if r.EstimateAfter != 20_000_000+30_000_000-1_000_000+1_000_000 {
		t.Errorf("estimate_after = %d", r.EstimateAfter)
	}
	if len(p.ids) != 2 || len(p.amounts) != 2 || len(p.creates) != 1 || p.amounts[0] != 30_000_000 {
		t.Errorf("plan = %+v", p)
	}
}

func TestSmartAdjustPlanIsIdempotent(t *testing.T) {
	id := uuid.New()
	rows := []domain.AdjustRow{{Item: item("akad", "biaya_kua", "Biaya KUA", 600_000), BudgetID: &id, From: 600_000}}
	p := smartAdjustPlan(rows, 3, 1_000_000)
	if len(p.report.Updated) != 0 || len(p.report.Created) != 0 || len(p.ids) != 0 || len(p.creates) != 0 || p.report.EstimateAfter != 1_000_000 || p.report.UntouchedCount != 2 {
		t.Errorf("a line that already has the amount must not change: %+v", p.report)
	}
}

// Two keyless items of one name pair with the same line; the second becomes a line of its own
// instead of overwriting the first (an UPDATE … FROM with a duplicate target is undefined).
func TestSmartAdjustPlanDoesNotPairTwoItemsWithOneLine(t *testing.T) {
	id := uuid.New()
	rows := []domain.AdjustRow{
		{Item: item("all", "", "Photobooth", 1_000_000), BudgetID: &id, From: 0},
		{Item: item("all", "", "photobooth", 2_000_000), BudgetID: &id, From: 0},
	}
	p := smartAdjustPlan(rows, 1, 0)
	if len(p.report.Updated) != 1 || p.report.Updated[0].To != 1_000_000 || len(p.report.Created) != 1 || p.report.Created[0].Amount != 2_000_000 {
		t.Errorf("report = %+v", p.report)
	}
	if p.report.UntouchedCount != 0 || p.report.EstimateAfter != 3_000_000 {
		t.Errorf("untouched %d, estimate_after %d", p.report.UntouchedCount, p.report.EstimateAfter)
	}
}

func TestCleanOverwritePlan(t *testing.T) {
	out := item("lamaran", "cincin", "Cincin", 999)
	out.Included = false
	sc := domain.Scenario{Items: []domain.ScenarioItem{item("resepsi", "catering", "Catering", 30_000_000), out, item("all", "dana_darurat", "Dana darurat", 1_000_000)}}

	p := cleanOverwritePlan(sc, domain.LineTotals{Lines: 16, Paid: 500_000})
	r := p.report
	if r.Mode != domain.ApplyCleanOverwrite || r.Deleted == nil || r.Deleted.Lines != 16 || r.Deleted.PaidTotal != 500_000 ||
		len(r.Created) != 2 || len(r.Updated) != 0 || len(r.Skipped) != 0 || r.UntouchedCount != 0 || r.EstimateAfter != 31_000_000 || len(p.creates) != 2 {
		t.Errorf("report = %+v", r)
	}
	if r.Updated == nil || r.Skipped == nil {
		t.Error("the lists of a report are never null")
	}
}

func TestParseApplyMode(t *testing.T) {
	for in, want := range map[string]string{"": "smart_adjust", " smart_adjust ": "smart_adjust", "clean_overwrite": "clean_overwrite"} {
		if got, err := parseApplyMode(in); err != nil || got != want {
			t.Errorf("parseApplyMode(%q) = %q, %v", in, got, err)
		}
	}
	_, err := parseApplyMode("merge")
	wantAPIError(t, err, 400, "invalid_input", "mode: must be one of smart_adjust, clean_overwrite")
}

// ---------- scenarios ----------

func TestEnrichComputesIncludedAllocatedAndPresetAmount(t *testing.T) {
	sc := domain.Scenario{
		TotalTarget: 50_000_000, PresetKey: ptr("wedding_50jt"), ActiveSubEvents: []string{"resepsi"},
		Items: []domain.ScenarioItem{
			item("akad", "biaya_kua", "Biaya KUA", 600_000), item("resepsi", "catering", "Catering", 21_000_000),
			item("all", "dana_darurat", "Dana darurat", 1_000_000), item("resepsi", "", "Kustom", 5),
		},
	}
	enrich(&sc, pd.EventWedding)

	if sc.Items[0].Included || !sc.Items[1].Included || !sc.Items[2].Included || !sc.Items[3].Included {
		t.Errorf("included = %v %v %v %v", sc.Items[0].Included, sc.Items[1].Included, sc.Items[2].Included, sc.Items[3].Included)
	}
	if sc.Allocated != 21_000_000+1_000_000+5 {
		t.Errorf("allocated = %d", sc.Allocated)
	}
	// The preset value is kept next to the edited one: catering was 20.000.000 in the package, the user made it 21.000.000.
	if a := sc.Items[1].PresetAmount; a == nil || *a != 20_000_000 {
		t.Errorf("catering preset_amount = %v", a)
	}
	if sc.Items[3].PresetAmount != nil {
		t.Error("an item without a key has no preset amount")
	}

	// Empty active_sub_events: every session takes part.
	sc.ActiveSubEvents = nil
	enrich(&sc, pd.EventWedding)
	if !sc.Items[0].Included || sc.Allocated != 600_000+21_000_000+1_000_000+5 {
		t.Errorf("without sessions: %v, allocated %d", sc.Items[0].Included, sc.Allocated)
	}

	// A percentage preset whose total was lowered below the minimum has no preset amounts (they would all be 0).
	pct := domain.Scenario{TotalTarget: 50_000, PresetKey: ptr("seminar_pct"), Items: []domain.ScenarioItem{item("all", "venue", "Venue", 1)}}
	enrich(&pct, pd.EventSeminar)
	if pct.Items[0].PresetAmount != nil {
		t.Errorf("preset_amount = %v", *pct.Items[0].PresetAmount)
	}
	pct.TotalTarget = 40_000_000
	enrich(&pct, pd.EventSeminar)
	if a := pct.Items[0].PresetAmount; a == nil || *a != 10_000_000 {
		t.Errorf("seminar venue preset_amount = %v", a)
	}
}

func TestPresetPlan(t *testing.T) {
	items, total, err := presetPlan(pd.EventWedding, "wedding_30jt", nil)
	if err != nil || total != 30_000_000 || len(items) != 10 {
		t.Fatalf("fixed preset: %d items, total %d, %v", len(items), total, err)
	}
	_, _, err = presetPlan(pd.EventWedding, "wedding_30jt", ptr(int64(31_000_000)))
	wantAPIError(t, err, 400, "invalid_input", "total_target: must match the preset total")
	_, _, err = presetPlan(pd.EventWedding, "seminar_pct", ptr(int64(1_000_000)))
	wantAPIError(t, err, 400, "invalid_template", "preset_key: not available for this event type")
	_, _, err = presetPlan(pd.EventSeminar, "seminar_pct", nil)
	wantAPIError(t, err, 400, "invalid_input", "total_target: required")
	_, _, err = presetPlan(pd.EventSeminar, "seminar_pct", ptr(int64(99_999)))
	wantAPIError(t, err, 400, "invalid_input", "total_target: must be at least 100000")
	_, _, err = presetPlan(pd.EventSeminar, "seminar_pct", ptr(domain.MaxMoney+1))
	wantAPIError(t, err, 400, "invalid_input", "total_target: must be between 1 and 10000000000000")
	items, total, err = presetPlan(pd.EventSeminar, "seminar_pct", ptr(int64(100_000)))
	if err != nil || total != 100_000 || len(items) != 8 {
		t.Errorf("percent preset at the minimum: %d items, %d, %v", len(items), total, err)
	}
}

func TestItemsFromPresetFallBackToTheGeneralSession(t *testing.T) {
	preset, _, _ := domain.PresetItems(pd.EventWedding, "wedding_50jt", 0)
	items, fallback := itemsFromPreset(preset, map[string]bool{"all": true, "akad": true})
	if fallback != 8 || len(items) != 10 {
		t.Fatalf("fallback %d, items %d", fallback, len(items))
	}
	scopes := map[string]int{}
	ids := map[uuid.UUID]bool{}
	for _, it := range items {
		scopes[it.SubEventScope]++
		ids[it.ID] = true
		if it.ItemKey == nil {
			t.Error("preset items keep their key")
		}
	}
	if scopes["all"] != 9 || scopes["akad"] != 1 || len(ids) != 10 {
		t.Errorf("scopes %v, %d distinct ids", scopes, len(ids))
	}
	if w := fallbackWarnings(0); w == nil || len(w) != 0 {
		t.Error("no fallback is an empty list, not null")
	}
	if w := fallbackWarnings(3); len(w) != 1 || w[0].Code != "scope_fallback" || w[0].Count != 3 {
		t.Errorf("warnings = %+v", w)
	}
}

func TestCompare(t *testing.T) {
	a := domain.Scenario{ID: uuid.New(), Name: "A", TotalTarget: 10, Items: []domain.ScenarioItem{
		item("resepsi", "catering", "Catering", 20), item("resepsi", "", "Photobooth", 5), item("all", "dana_darurat", "Dana darurat", 1), item("akad", "x", "Hanya A", 7),
	}}
	b := domain.Scenario{ID: uuid.New(), Name: "B", TotalTarget: 20, Items: []domain.ScenarioItem{
		item("resepsi", "catering", "Catering besar", 30), item("resepsi", "", "PHOTOBOOTH", 9), item("lamaran", "x", "Hanya B", 2), item("akad", "x", "Hanya A", 7),
	}}

	c := compare(a, b)
	if len(c.Rows) != 5 || c.A.Name != "A" || c.B.TotalTarget != 20 {
		t.Fatalf("comparison = %+v", c)
	}
	want := []struct {
		name        string
		a, b, delta int64
	}{{"Catering", 20, 30, 10}, {"Photobooth", 5, 9, 4}, {"Dana darurat", 1, 0, -1}, {"Hanya A", 7, 7, 0}, {"Hanya B", 0, 2, 2}}
	for i, w := range want {
		r := c.Rows[i]
		if r.ItemName != w.name || r.AAmount != w.a || r.BAmount != w.b || r.Delta != w.delta {
			t.Errorf("row %d = %+v, want %+v", i, r, w)
		}
	}
	// The same key in another session is another item.
	if c.Rows[3].SubEventScope != "akad" || c.Rows[4].SubEventScope != "lamaran" {
		t.Errorf("rows = %+v", c.Rows)
	}
	if c.Totals.A != 33 || c.Totals.B != 48 || c.Totals.Delta != 15 {
		t.Errorf("totals = %+v", c.Totals)
	}
	if e := compare(domain.Scenario{}, domain.Scenario{}); e.Rows == nil || len(e.Rows) != 0 {
		t.Error("rows are never null")
	}
}

// ---------- savings ----------

func TestRecommendationAcrossTime(t *testing.T) {
	event := date(2027, 3, 28)
	last := []domain.Milestone{{SessionDate: "2027-03-28"}}

	for _, c := range []struct {
		name         string
		today        time.Time
		shortfall    int64
		contributors int
		milestones   []domain.Milestone
		months       int
		monthly      int64
		per          *int64
		basis        string
	}{
		{"the day of the PDF (175 days)", date(2026, 10, 4), 27_500_000, 2, last, 6, 4_584_000, ptr(int64(2_292_000)), "session"},
		{"17 days later (158 days, still 6 months)", date(2026, 10, 21), 27_500_000, 2, last, 6, 4_584_000, ptr(int64(2_292_000)), "session"},
		{"150 days is 5 months", event.AddDate(0, 0, -150), 27_500_000, 2, last, 5, 5_500_000, ptr(int64(2_750_000)), "session"},
		{"151 days is 6 months", event.AddDate(0, 0, -151), 27_500_000, 2, last, 6, 4_584_000, ptr(int64(2_292_000)), "session"},
		{"30 days is 1 month", event.AddDate(0, 0, -30), 27_500_000, 2, last, 1, 27_500_000, ptr(int64(13_750_000)), "session"},
		{"31 days is 2 months", event.AddDate(0, 0, -31), 27_500_000, 2, last, 2, 13_750_000, ptr(int64(6_875_000)), "session"},
		{"the day before (1 day)", date(2027, 3, 27), 27_500_000, 2, last, 1, 27_500_000, ptr(int64(13_750_000)), "session"},
		{"Hari-H itself", event, 27_500_000, 2, last, 1, 27_500_000, ptr(int64(13_750_000)), "session"},
		{"six days after", date(2027, 4, 2), 27_500_000, 2, last, 1, 27_500_000, ptr(int64(13_750_000)), "session"},
		{"nothing missing", date(2026, 10, 4), 0, 2, last, 6, 0, ptr(int64(0)), "session"},
		{"no contributor", date(2026, 10, 4), 27_500_000, 0, last, 6, 4_584_000, nil, "session"},
		{"three contributors", date(2026, 10, 4), 27_500_000, 3, last, 6, 4_584_000, ptr(int64(1_528_000)), "session"},
		{"no dated session: Hari-H", date(2026, 10, 4), 27_500_000, 2, nil, 6, 4_584_000, ptr(int64(2_292_000)), "event_date"},
		{"rounds every figure up to 1.000", date(2026, 10, 4), 1, 1, last, 6, 1_000, ptr(int64(1_000)), "session"},
	} {
		got := recommend(c.shortfall, c.contributors, c.milestones, event, c.today)
		if got.MonthsLeft != c.months || got.RecommendedMonthly != c.monthly || got.Basis != c.basis || got.TargetDate != "2027-03-28" {
			t.Errorf("%s: %+v, want %d months, %d a month, basis %s", c.name, got, c.months, c.monthly, c.basis)
		}
		if (got.PerContributor == nil) != (c.per == nil) || (c.per != nil && *got.PerContributor != *c.per) {
			t.Errorf("%s: per_contributor = %v, want %v", c.name, got.PerContributor, c.per)
		}
	}

	// The last dated session decides, not Hari-H.
	early := []domain.Milestone{{SessionDate: "2026-11-15"}, {SessionDate: "2027-01-10"}}
	if got := recommend(10_000_000, 1, early, event, date(2026, 10, 4)); got.TargetDate != "2027-01-10" || got.MonthsLeft != 4 || got.RecommendedMonthly != 2_500_000 {
		t.Errorf("last session: %+v", got)
	}
}

func TestSavingsMatrixFoldsCellsIntoMonths(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	m := savingsMatrix([]domain.MonthCell{
		{Month: "2026-02", ContributorID: a, Amount: 2_500_000}, {Month: "2026-02", ContributorID: b, Amount: 500_000},
		{Month: "2026-03", ContributorID: a}, {Month: "2026-03", ContributorID: b},
		{Month: "2026-04", ContributorID: a, Amount: 1}, {Month: "2026-04", ContributorID: b, Amount: 2},
	})
	if len(m.Months) != 3 || m.Months[1] != "2026-03" || len(m.Rows) != 3 {
		t.Fatalf("matrix = %+v", m)
	}
	if m.Rows[0].Total != 3_000_000 || m.Rows[1].Total != 0 || m.Rows[2].Total != 3 {
		t.Errorf("totals = %d %d %d", m.Rows[0].Total, m.Rows[1].Total, m.Rows[2].Total)
	}
	if len(m.Rows[1].ByContributor) != 2 || m.Rows[1].ByContributor[a.String()] != 0 {
		t.Errorf("an empty month still lists every contributor: %v", m.Rows[1].ByContributor)
	}
	if e := savingsMatrix(nil); e.Months == nil || e.Rows == nil || len(e.Months) != 0 {
		t.Error("an empty matrix has empty lists, not null")
	}
}

// ---------- realisation ----------

func TestCashflow(t *testing.T) {
	// Data of the PDF: 47.500.000 collected, 33.550.000 paid, 40.600.000 still owed.
	c := cashflow(domain.LineTotals{Paid: 33_550_000, Remaining: 40_600_000}, 47_500_000)
	if c.AvailableCash != 13_950_000 || c.CashDeficit != 26_650_000 || c.RemainingDebt != 40_600_000 || c.Overpaid != 0 || c.TotalSavingsCollected != 47_500_000 {
		t.Errorf("cashflow = %+v", c)
	}
	// Plenty of cash: no deficit. Not enough paid yet: the available cash is negative.
	if c := cashflow(domain.LineTotals{Paid: 10, Remaining: 50}, 100); c.AvailableCash != 90 || c.CashDeficit != 0 {
		t.Errorf("surplus = %+v", c)
	}
	if c := cashflow(domain.LineTotals{Paid: 7_000_000, Remaining: 10_000_000, Overpaid: 5_000_000}, 0); c.AvailableCash != -7_000_000 || c.CashDeficit != 17_000_000 || c.Overpaid != 5_000_000 {
		t.Errorf("overpaid = %+v", c)
	}
}

func TestFinanceSummaryMatchesTheDashboardContract(t *testing.T) {
	project := pd.Project{TargetBudget: 75_000_000}
	totals := domain.LineTotals{Estimate: 74_150_000, Paid: 33_550_000, Remaining: 40_600_000, Lines: 14, Status: domain.LineStatus{Settled: 7, Partial: 7}}

	s := financeSummary(project, totals, 47_500_000)
	if s.TargetBudget != 75_000_000 || s.TotalEstimate != 74_150_000 || s.TotalPaid != 33_550_000 || s.Remaining != 40_600_000 || s.Overpaid != 0 ||
		s.SavingsTarget != 75_000_000 || s.SavingsCollected != 47_500_000 || s.FreeCash != 13_950_000 || s.CashDeficit != 26_650_000 ||
		s.LineStatus.Settled != 7 || s.LineStatus.Partial != 7 || s.LineStatus.Unpaid != 0 {
		t.Errorf("summary = %+v", s)
	}
	// An explicit savings target replaces the budget as the savings target.
	project.SavingsTarget = ptr(int64(60_000_000))
	if s := financeSummary(project, totals, 0); s.SavingsTarget != 60_000_000 || s.FreeCash != -33_550_000 {
		t.Errorf("summary with a savings target = %+v", s)
	}
}

func TestBuildMatrix(t *testing.T) {
	rows := []domain.MatrixRow{
		{SubEventScope: "akad", Estimate: 600, DP: 600, Paid: 600, LineCount: 1},
		{SubEventScope: "resepsi", Estimate: 1000, DP: 100, Term1: 200, Term2: 50, RollupPaid: 30, Paid: 380, Remaining: 650, Overpaid: 30, LineCount: 3},
	}
	m := buildMatrix(rows, domain.LineTotals{Estimate: 1600, Paid: 980, Remaining: 650, Overpaid: 30, Lines: 4, Status: domain.LineStatus{Settled: 1, Partial: 2, Unpaid: 1}})
	if m.Total.SubEventScope != "" || m.Total.Estimate != 1600 || m.Total.DP != 700 || m.Total.Term1 != 200 || m.Total.RollupPaid != 30 || m.Total.Paid != 980 || m.Total.Remaining != 650 || m.Total.LineCount != 4 {
		t.Errorf("total = %+v", m.Total)
	}
	if m.KPI.TotalEstimate != 1600 || m.KPI.TotalRemaining != 650 || m.KPI.TotalOverpaid != 30 || m.KPI.LineCount != 4 || m.KPI.LineStatus.Unpaid != 1 || m.ActiveScenario != nil {
		t.Errorf("kpi = %+v", m.KPI)
	}
	if b := buildMatrix(nil, domain.LineTotals{}); b.Rows != nil && len(b.Rows) != 0 {
		t.Error("an empty matrix has no rows")
	}
}

func TestFirstEmptyStage(t *testing.T) {
	for _, c := range []struct {
		l    domain.BudgetLine
		want string
	}{
		{domain.BudgetLine{}, "dp"},
		{domain.BudgetLine{DPAmount: 1}, "term_1"},
		{domain.BudgetLine{DPAmount: 1, Term1Amount: 1}, "term_2"},
		{domain.BudgetLine{DPAmount: 1, Term2Amount: 1}, "term_1"},
		{domain.BudgetLine{Term1Amount: 1, Term2Amount: 1}, "dp"},
		{domain.BudgetLine{DPAmount: 1, Term1Amount: 1, Term2Amount: 1}, ""},
	} {
		if got := firstEmptyStage(c.l); got != c.want {
			t.Errorf("%+v: %q, want %q", c.l, got, c.want)
		}
	}
}

// ---------- pricing and validation ----------

func TestPricing(t *testing.T) {
	lump := func(est *int64) domain.BudgetItemRequest { return domain.BudgetItemRequest{InitialEstimate: est} }

	var it domain.BudgetItem
	if err := pricing(&it, lump(nil)); err != nil || it.PricingMode != "lump" || it.InitialEstimate != 0 || it.QuantityX100 != nil {
		t.Errorf("default lump: %+v %v", it, err)
	}
	it = domain.BudgetItem{QuantityX100: ptr(int64(5)), UnitPrice: ptr(int64(5)), UnitLabel: ptr("x")}
	if err := pricing(&it, lump(ptr(int64(7)))); err != nil || it.InitialEstimate != 7 || it.QuantityX100 != nil || it.UnitPrice != nil || it.UnitLabel != nil {
		t.Errorf("lump forgets the unit fields: %+v %v", it, err)
	}

	unit := domain.BudgetItemRequest{PricingMode: ptr("unit"), Quantity: ptr(250.5), UnitPrice: ptr(int64(85_000)), UnitLabel: ptr(" porsi "), InitialEstimate: ptr(int64(1))}
	it = domain.BudgetItem{}
	if err := pricing(&it, unit); err != nil || it.InitialEstimate != 21_292_500 || *it.QuantityX100 != 25_050 || *it.UnitPrice != 85_000 || *it.UnitLabel != "porsi" || it.PricingMode != "unit" {
		t.Errorf("unit: %+v %v", it, err)
	}
	unit.UnitLabel = ptr("   ")
	it = domain.BudgetItem{}
	if err := pricing(&it, unit); err != nil || it.UnitLabel != nil {
		t.Errorf("a blank unit label is none: %+v %v", it, err)
	}

	for _, c := range []struct {
		name    string
		req     domain.BudgetItemRequest
		message string
	}{
		{"no quantity", domain.BudgetItemRequest{PricingMode: ptr("unit"), UnitPrice: ptr(int64(1))}, "quantity: required when pricing_mode is unit"},
		{"no price", domain.BudgetItemRequest{PricingMode: ptr("unit"), Quantity: ptr(1.0)}, "unit_price: required when pricing_mode is unit"},
		{"zero quantity", domain.BudgetItemRequest{PricingMode: ptr("unit"), Quantity: ptr(0.0), UnitPrice: ptr(int64(1))}, "quantity: must be between 0.01 and 1000000"},
		{"three decimals", domain.BudgetItemRequest{PricingMode: ptr("unit"), Quantity: ptr(1.005), UnitPrice: ptr(int64(1))}, "quantity: must be between 0.01 and 1000000"},
		{"too many", domain.BudgetItemRequest{PricingMode: ptr("unit"), Quantity: ptr(1_000_000.01), UnitPrice: ptr(int64(1))}, "quantity: must be between 0.01 and 1000000"},
		{"negative price", domain.BudgetItemRequest{PricingMode: ptr("unit"), Quantity: ptr(1.0), UnitPrice: ptr(int64(-1))}, "unit_price: must be between 0 and 10000000000000"},
		{"overflow", domain.BudgetItemRequest{PricingMode: ptr("unit"), Quantity: ptr(999_999.99), UnitPrice: ptr(domain.MaxMoney)}, "unit_price: the estimate (quantity × unit_price) must be at most 10000000000000"},
		{"mode", domain.BudgetItemRequest{PricingMode: ptr("weekly")}, "pricing_mode: must be one of lump, unit"},
		{"negative estimate", lump(ptr(int64(-1))), "initial_estimate: must be between 0 and 10000000000000"},
		{"long label", domain.BudgetItemRequest{PricingMode: ptr("unit"), Quantity: ptr(1.0), UnitPrice: ptr(int64(1)), UnitLabel: ptr(strings.Repeat("a", 31))}, "unit_label: must be at most 30 characters"},
	} {
		err := pricing(&domain.BudgetItem{}, c.req)
		if ae, ok := err.(*apierr.Error); !ok || ae.Code != "invalid_input" || ae.Message != c.message {
			t.Errorf("%s: %v, want invalid_input %q", c.name, err, c.message)
		}
	}
}

func TestCleanReceiptURL(t *testing.T) {
	for _, ok := range []string{"https://example.com/bukti.jpg", "http://x.id/a?b=c#d", " https://example.com "} {
		if got, err := cleanReceiptURL(&ok); err != nil || got == nil || *got != strings.TrimSpace(ok) {
			t.Errorf("%q: %v %v", ok, got, err)
		}
	}
	for _, none := range []*string{nil, ptr(""), ptr("   ")} {
		if got, err := cleanReceiptURL(none); err != nil || got != nil {
			t.Errorf("%v must mean no receipt: %v %v", none, got, err)
		}
	}
	for _, bad := range []string{"ftp://example.com", "javascript:alert(1)", "//example.com", "bukti.jpg", "https://", "https://exa mple.com/\x00", "data:text/html,x", "https://" + strings.Repeat("a", 500)} {
		if _, err := cleanReceiptURL(&bad); err == nil {
			t.Errorf("%q must be refused", bad)
		} else {
			wantAPIError(t, err, 400, "invalid_input", "receipt_url: must be an http or https URL")
		}
	}
}

func TestValidationHelpers(t *testing.T) {
	if s, err := cleanText("name", "  Arif  ", 10, true); err != nil || s != "Arif" {
		t.Errorf("cleanText = %q %v", s, err)
	}
	_, err := cleanText("name", "   ", 10, true)
	wantAPIError(t, err, 400, "invalid_input", "name: must not be empty")
	_, err = cleanText("name", "abc\x07", 10, true)
	wantAPIError(t, err, 400, "invalid_input", "name: must not contain control characters")
	_, err = cleanText("name", "ééééé", 4, true)
	wantAPIError(t, err, 400, "invalid_input", "name: must be at most 4 characters")
	if s, err := cleanNote("notes", " a\r\nb ", 10); err != nil || s != "a\nb" {
		t.Errorf("cleanNote = %q %v", s, err)
	}
	if _, err := cleanNote("notes", "a\tb", 10); err == nil {
		t.Error("a tab is a control character")
	}
	if n, err := optionalNote("notes", ptr("  ")); err != nil || n != nil {
		t.Errorf("a blank note is none: %v %v", n, err)
	}

	for _, v := range []int64{-1, domain.MaxMoney + 1} {
		if validateMoney("amount", v) == nil {
			t.Errorf("%d must be refused", v)
		}
	}
	if validateMoney("amount", 0) != nil || validateMoney("amount", domain.MaxMoney) != nil || validateTotal("t", 0) == nil || validateTotal("t", 1) != nil {
		t.Error("money bounds")
	}

	if k, err := cleanKey("item_key", ptr(" catering ")); err != nil || *k != "catering" {
		t.Errorf("cleanKey = %v %v", k, err)
	}
	if k, err := cleanKey("item_key", ptr(" ")); err != nil || k != nil {
		t.Errorf("a blank key is none: %v %v", k, err)
	}
	for _, bad := range []string{"Catering", "a b", "a-b", strings.Repeat("a", 41)} {
		if _, err := cleanKey("item_key", &bad); err == nil {
			t.Errorf("key %q must be refused", bad)
		}
	}

	if s, err := normalizeScope(" "); err != nil || s != "all" {
		t.Errorf("a blank scope is the general session: %q %v", s, err)
	}
	if _, err := normalizeScope("Bad Scope"); err == nil {
		t.Error("a malformed scope must be refused")
	}

	ids := []uuid.UUID{uuid.New(), uuid.New()}
	if checkIDs(ids, 2) != nil || checkIDs(ids, 3) == nil || checkIDs([]uuid.UUID{ids[0], ids[0]}, 2) == nil {
		t.Error("checkIDs")
	}

	// The window is Hari-H − 1095 days … the last day + 365 days, both ends included (2028 is a leap year).
	p := pd.Project{EventDate: date(2027, 3, 28)}
	for d, ok := range map[time.Time]bool{
		date(2027, 3, 28): true, date(2024, 3, 28): true, date(2024, 3, 27): false, date(2028, 3, 27): true, date(2028, 3, 28): false, date(2030, 1, 1): false,
	} {
		if (checkWindow(p, "paid_at", d) == nil) != ok {
			t.Errorf("window %s: allowed = %v, want %v", d.Format(pd.DateLayout), !ok, ok)
		}
	}
	end := date(2027, 3, 30)
	p.EventEndDate = &end
	if checkWindow(p, "paid_at", date(2028, 3, 29)) != nil || checkWindow(p, "paid_at", date(2028, 3, 30)) == nil {
		t.Error("the window ends a year after the last day of a multi-day event")
	}
}

func TestMapErr(t *testing.T) {
	for _, c := range []struct {
		err          error
		status       int
		code, prefix string
	}{
		{domain.ErrNotFound, 404, "not_found", "scenario not found"},
		{domain.ErrInvalidScope, 400, "invalid_scope", "sub_event_scope"},
		{domain.ErrInvalidVendor, 400, "invalid_input", "vendor_id: must be a vendor"},
		{domain.ErrVendorAlreadyLinked, 409, "vendor_already_linked", "vendor_id"},
		{domain.ErrKeyTaken, 409, "conflict", "item_key"},
		{domain.ErrConflict, 409, "conflict", "the project was changed"},
	} {
		ae, ok := mapErr(c.err, "scenario").(*apierr.Error)
		if !ok || ae.Status != c.status || ae.Code != c.code || !strings.HasPrefix(ae.Message, c.prefix) {
			t.Errorf("%v -> %+v", c.err, ae)
		}
	}
	if mapErr(nil, "x") != nil {
		t.Error("nil stays nil")
	}
	other := &apierr.Error{Status: 409, Code: "max_scenarios_reached"}
	if mapErr(other, "x") != error(other) {
		t.Error("an API error passes through untouched")
	}
}
