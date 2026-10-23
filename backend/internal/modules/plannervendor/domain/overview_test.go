package domain

import (
	"testing"

	"github.com/google/uuid"
)

var weddingCats = []CategorySpec{
	{"venue", "Venue", []string{"venue_dekor"}}, {"dekorasi", "Decoration", []string{"venue_dekor"}},
	{"wo", "Wedding Organizer", nil}, {"mua", "MUA", []string{"rias_busana"}}, {"catering", "Catering", []string{"catering"}},
	{"fotografi", "Photography", []string{"dokumentasi"}}, {"attire", "Attire", []string{"rias_busana"}},
	{"hand_bouquet", "Hand Bouquet", nil}, {"mc", "MC", []string{"hiburan"}}, {"entertainment", "Entertainment", []string{"hiburan"}},
}

// The sample project as the real backend of menus 01-03 builds it (04 BE 2.1): the active scenario of 75 Jt holds
// catering 30.000.000, dokumentasi 3.000.000, venue_dekor 20.000.000, rias_busana 5.000.000, hiburan 3.400.000 in `resepsi`.
func sampleData() (vendors []VendorRow, scenario []ScenarioRow, lines []LineRow) {
	for i, sv := range SampleVendors {
		vendors = append(vendors, VendorRow{
			ID: uuid.New(), SubEventScope: SampleScope, CategoryKey: sv.CategoryKey, CategoryLabel: sv.CategoryKey, VendorName: sv.Name,
			AgreedPrice: sv.Price, Status: sv.Status, SortOrder: i + 1,
		})
	}
	for _, r := range []struct {
		key  string
		cost int64
	}{{"catering", 30_000_000}, {"dokumentasi", 3_000_000}, {"venue_dekor", 20_000_000}, {"rias_busana", 5_000_000}, {"hiburan", 3_400_000}} {
		scenario = append(scenario, ScenarioRow{SubEventScope: "resepsi", ItemKey: r.key, ItemName: r.key, Cost: r.cost})
	}
	vendorOf := func(category string) *uuid.UUID {
		for _, v := range vendors {
			if v.CategoryKey == category {
				id := v.ID

				return &id
			}
		}

		return nil
	}
	lines = []LineRow{
		{uuid.New(), "resepsi", "catering", "Catering", vendorOf("catering"), 25_000_000, 15_000_000, 10_000_000, "partial"},
		{uuid.New(), "resepsi", "dokumentasi", "Dokumentasi", vendorOf("fotografi"), 3_000_000, 600_000, 2_400_000, "dp_paid"},
		{uuid.New(), "resepsi", "venue_dekor", "Sewa Gedung & Dekorasi", nil, 20_000_000, 900_000, 19_100_000, "dp_paid"},
		{uuid.New(), "resepsi", "rias_busana", "Rias & Baju Pengantin", nil, 5_000_000, 600_000, 4_400_000, "dp_paid"},
		{uuid.New(), "resepsi", "hiburan", "Hiburan & Pengisi acara", nil, 3_400_000, 3_400_000, 0, "settled"},
	}

	return vendors, scenario, lines
}

func TestSampleOverview(t *testing.T) {
	vs, ss, ls := sampleData()
	o := BuildOverview(weddingCats, nil, vs, ss, ls)

	k := o.KPI
	if k.CategoriesTotal != 10 || k.CategoriesFixed != 7 || k.Skipped != 0 || k.FixedVendors != 7 || k.Pipeline != 3 ||
		k.FixedTotal != 58_000_000 || k.PipelineOffers != 4_300_000 || k.GroupsTotal != 3 || k.GroupsOver != 0 {
		t.Fatalf("kpi = %+v", k)
	}

	ptr := func(v int64) *int64 { return &v }
	type limit struct {
		src   string
		limit int64
		delta *int64
	}
	want := map[string]limit{
		"venue":     {LimitSourceGroup, 0, nil},
		"catering":  {LimitSourceScenario, 30_000_000, ptr(5_000_000)}, // 75 Jt scenario: 30 Jt, deal 25 Jt (the old wireframe said 25 Jt)
		"fotografi": {LimitSourceScenario, 3_000_000, ptr(0)},
		"wo":        {LimitSourceNone, 0, nil},
	}
	for _, v := range o.Vendors {
		if w, ok := want[v.CategoryKey]; ok {
			if v.LimitSource != w.src || v.BudgetLimit != w.limit || (v.PriceDelta == nil) != (w.delta == nil) || (w.delta != nil && *v.PriceDelta != *w.delta) {
				t.Errorf("%s: %+v", v.CategoryKey, v)
			}
		}
	}

	if len(o.LimitGroups) != 3 {
		t.Fatalf("groups = %d", len(o.LimitGroups))
	}
	groups := map[string][4]int64{"venue_dekor": {20_000_000, 20_000_000, 0, 2}, "rias_busana": {5_000_000, 5_000_000, 0, 2}, "hiburan": {3_400_000, 3_400_000, 0, 2}}
	for _, g := range o.LimitGroups {
		w := groups[g.ItemKey]
		if g.Limit != w[0] || g.TotalAgreed != w[1] || g.Delta == nil || *g.Delta != w[2] || int64(g.Members) != w[3] || g.Priced != g.Members || g.Line == nil {
			t.Errorf("group %+v", g)
		}
	}

	// the payment of a vendor is read from the line that points at it; the shared lines are read at group level
	for _, v := range o.Vendors {
		switch v.CategoryKey {
		case "catering":
			if v.Payment == nil || v.Payment.Paid != 15_000_000 || v.Payment.PaymentStatus != "partial" {
				t.Errorf("catering payment %+v", v.Payment)
			}
		case "fotografi":
			if v.Payment == nil || v.Payment.PaymentStatus != "dp_paid" {
				t.Errorf("fotografi payment %+v", v.Payment)
			}
		default:
			if v.Payment != nil {
				t.Errorf("%s should have no payment", v.CategoryKey)
			}
		}
	}
	states := map[string]string{}
	for _, c := range o.Categories {
		states[c.Key] = c.State
	}
	if states["hand_bouquet"] != "pipeline" || states["mc"] != "pipeline" || states["venue"] != "fix" {
		t.Errorf("states %v", states)
	}
}

func TestOverviewEdgeCases(t *testing.T) {
	vs, ss, ls := sampleData()
	with := func(mut func(v *VendorRow)) []VendorRow {
		out := append([]VendorRow(nil), vs...)
		for i := range out {
			mut(&out[i])
		}

		return out
	}

	// a cancelled vendor counts nowhere: its category is empty again, the KPI and the group follow
	o := BuildOverview(weddingCats, nil, with(func(v *VendorRow) {
		if v.CategoryKey == "dekorasi" {
			v.Status = VendorCancelled
		}
	}), ss, ls)
	if o.KPI.FixedVendors != 6 || o.KPI.CategoriesFixed != 6 || o.KPI.FixedTotal != 50_000_000 || o.KPI.Pipeline != 3 || o.KPI.PipelineOffers != 4_300_000 {
		t.Errorf("cancelled kpi %+v", o.KPI)
	}
	for _, c := range o.Categories {
		if c.Key == "dekorasi" && (c.State != "empty" || c.Vendors != 0) {
			t.Errorf("a category whose only vendor is cancelled is empty again: %+v", c)
		}
	}
	for _, g := range o.LimitGroups {
		if g.ItemKey == "venue_dekor" && (g.Members != 1 || g.TotalAgreed != 12_000_000 || g.Delta == nil || *g.Delta != 8_000_000) {
			t.Errorf("venue_dekor %+v", g)
		}
	}

	// a member without an offer: 2 members, 1 priced; the difference is what is left of the limit, not a saving
	o = BuildOverview(weddingCats, nil, with(func(v *VendorRow) {
		if v.CategoryKey == "dekorasi" {
			v.AgreedPrice = 0
		}
	}), ss, ls)
	for _, g := range o.LimitGroups {
		if g.ItemKey == "venue_dekor" && (g.Members != 2 || g.Priced != 1 || g.TotalAgreed != 12_000_000 || *g.Delta != 8_000_000) {
			t.Errorf("unpriced member %+v", g)
		}
	}

	// a manual limit beats the scenario and takes the vendor out of the group
	o = BuildOverview(weddingCats, nil, with(func(v *VendorRow) {
		if v.CategoryKey == "venue" {
			v.BudgetLimit = 10_000_000
		}
	}), ss, ls)
	for _, v := range o.Vendors {
		if v.CategoryKey == "venue" && (v.LimitSource != LimitSourceManual || v.PriceDelta == nil || *v.PriceDelta != -2_000_000 || v.LimitGroup != nil) {
			t.Errorf("manual venue %+v", v)
		}
	}
	for _, g := range o.LimitGroups {
		if g.ItemKey == "venue_dekor" && (g.Members != 1 || g.TotalAgreed != 8_000_000) {
			t.Errorf("group without the manual member %+v", g)
		}
	}

	// a group that is over its limit is counted
	o = BuildOverview(weddingCats, nil, with(func(v *VendorRow) {
		if v.CategoryKey == "venue" {
			v.AgreedPrice = 15_000_000
		}
	}), ss, ls)
	if o.KPI.GroupsOver != 1 {
		t.Errorf("groups over = %d", o.KPI.GroupsOver)
	}

	// a skipped category leaves the count; a custom category in use joins it
	custom := append(append([]VendorRow(nil), vs...),
		VendorRow{ID: uuid.New(), SubEventScope: "resepsi", CategoryKey: "photobooth", CategoryLabel: "Photobooth", VendorName: "Booth A", Status: VendorPlanned, SortOrder: 99})
	o = BuildOverview(weddingCats, []string{"hand_bouquet"}, custom, ss, ls)
	if o.KPI.CategoriesTotal != 10 || o.KPI.CategoriesFixed != 7 || o.KPI.Skipped != 1 || len(o.Categories) != 11 || !o.Categories[10].Custom {
		t.Errorf("custom/skipped %+v / %d", o.KPI, len(o.Categories))
	}

	// switching the active scenario moves the limit with it: nothing stale is stored
	o = BuildOverview(weddingCats, nil, vs, []ScenarioRow{{"resepsi", "catering", "catering", 50_000_000}, {"resepsi", "venue_dekor", "venue", 24_000_000}}, ls)
	for _, v := range o.Vendors {
		if v.CategoryKey == "catering" && (v.BudgetLimit != 50_000_000 || *v.PriceDelta != 25_000_000) {
			t.Errorf("catering after the switch %+v", v)
		}
	}
	// no active scenario: no limits, no groups
	o = BuildOverview(weddingCats, nil, vs, nil, ls)
	if len(o.LimitGroups) != 0 {
		t.Errorf("groups without a scenario: %d", len(o.LimitGroups))
	}
	for _, v := range o.Vendors {
		if v.LimitSource != LimitSourceNone || v.PriceDelta != nil {
			t.Errorf("%s without a scenario: %+v", v.CategoryKey, v)
		}
	}

	// the input is not changed
	shuffled := []VendorRow{vs[3], vs[0]}
	BuildOverview(weddingCats, nil, shuffled, ss, ls)
	if shuffled[0].ID != vs[3].ID {
		t.Error("BuildOverview reordered its input")
	}
}

// Every scenario line and template line of a non-wedding type is in 'all', so a vendor in a real session must still get its
// limit: its own session is tried first and then 'all'. Without that fallback a Venue in `registrasi` has no limit.
func TestNonWeddingScopeFallback(t *testing.T) {
	cats := []CategorySpec{{"venue", "Venue", []string{"venue"}}, {"av", "Audio Visual", []string{"av_streaming"}}, {"streaming", "Live Streaming", []string{"av_streaming"}}}
	sc := []ScenarioRow{{"all", "venue", "Venue", 25_000_000}, {"all", "av_streaming", "AV dan streaming", 12_000_000}}
	vs := []VendorRow{
		{ID: uuid.New(), SubEventScope: "registrasi", CategoryKey: "venue", CategoryLabel: "Venue", VendorName: "Aula", AgreedPrice: 20_000_000, Status: VendorFixed},
		{ID: uuid.New(), SubEventScope: "all", CategoryKey: "av", CategoryLabel: "Audio Visual", VendorName: "AV", AgreedPrice: 7_000_000, Status: VendorNegotiating, SortOrder: 2},
	}
	o := BuildOverview(cats, nil, vs, sc, nil)
	if v := o.Vendors[0]; v.LimitSource != LimitSourceScenario || v.BudgetLimit != 25_000_000 || *v.PriceDelta != 5_000_000 {
		t.Errorf("venue in a real session: %+v", v)
	}
	if len(o.LimitGroups) != 1 || o.LimitGroups[0].SubEventScope != "all" || o.LimitGroups[0].Members != 1 {
		t.Errorf("groups %+v", o.LimitGroups)
	}
	// a line of the vendor's own session wins over the one in 'all'
	own := append(sc, ScenarioRow{"registrasi", "venue", "Venue", 40_000_000})
	if v := BuildOverview(cats, nil, vs, own, nil).Vendors[0]; v.BudgetLimit != 40_000_000 {
		t.Errorf("own session first: %+v", v)
	}
}

func TestSharedItemOf(t *testing.T) {
	vs, ss, ls := sampleData()

	item, ok := SharedItemOf(weddingCats, "venue", "resepsi", ss, ls, vs)
	if !ok || item.ItemKey != "venue_dekor" || item.ItemName != "Sewa Gedung & Dekorasi" || item.BudgetItemID == nil || item.Plan != 20_000_000 || item.TotalAgreed != 20_000_000 {
		t.Errorf("venue: %+v %v", item, ok)
	}
	if _, ok := SharedItemOf(weddingCats, "catering", "resepsi", ss, ls, vs); ok {
		t.Error("catering is not shared")
	}
	if _, ok := SharedItemOf(weddingCats, "wo", "resepsi", ss, ls, vs); ok {
		t.Error("a category without a line is not shared")
	}
	if _, ok := SharedItemOf(weddingCats, "no_such", "resepsi", ss, ls, vs); ok {
		t.Error("a custom category is not shared")
	}

	// no budget line yet: the scenario line gives the name and the amount, and there is no line id
	item, ok = SharedItemOf(weddingCats, "mua", "resepsi", ss, nil, vs)
	if !ok || item.BudgetItemID != nil || item.Plan != 5_000_000 || item.ItemName != "rias_busana" || item.TotalAgreed != 5_000_000 {
		t.Errorf("mua without a line: %+v %v", item, ok)
	}
	// a line that a vendor holds is not the shared line
	held := append([]LineRow(nil), ls...)
	vid := vs[0].ID
	for i := range held {
		if held[i].ItemKey == "venue_dekor" {
			held[i].VendorID = &vid
		}
	}
	item, _ = SharedItemOf(weddingCats, "venue", "resepsi", ss, held, vs)
	if item.BudgetItemID != nil {
		t.Errorf("a vendor-held line was taken as the shared line: %+v", item)
	}
}

func TestSharedKeysPerEventType(t *testing.T) {
	got := SharedKeys(weddingCats)
	if len(got) != 3 || !got["venue_dekor"] || !got["rias_busana"] || !got["hiburan"] {
		t.Errorf("wedding shared keys = %v", got)
	}
}
