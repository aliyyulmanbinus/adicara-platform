package domain

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// sql_aggregates.json is the output of the SQL of 05 BE 2.8 run on PostgreSQL against the 20 guests of the sample project
// (the HTTP golden test of delivery/http computes the same figures live).
func loadSQLAggregates(t *testing.T) Aggregates {
	raw, err := os.ReadFile("testdata/sql_aggregates.json")
	if err != nil {
		t.Fatal(err)
	}
	var j struct {
		Total        int `json:"total"`
		TotalPax     int `json:"total_pax"`
		Sent         int `json:"sent"`
		Attending    int `json:"attending"`
		Declined     int `json:"declined"`
		Maybe        int `json:"maybe"`
		Unconfirmed  int `json:"unconfirmed"`
		AttendingPax int `json:"attending_pax"`
		Cells        []struct {
			Side        *string `json:"side"`
			Category    string  `json:"category"`
			Invitations int     `json:"invitations"`
			Pax         int     `json:"pax"`
		} `json:"cells"`
	}
	if err := json.Unmarshal(raw, &j); err != nil {
		t.Fatal(err)
	}
	a := Aggregates{Total: j.Total, TotalPax: j.TotalPax, Sent: j.Sent, Attending: j.Attending, Declined: j.Declined, Maybe: j.Maybe, Unconfirmed: j.Unconfirmed, AttendingPax: j.AttendingPax}
	for _, c := range j.Cells {
		a.Cells = append(a.Cells, Cell{Side: c.Side, Category: c.Category, Invitations: c.Invitations, Pax: c.Pax})
	}
	return a
}

func sum(n int, f func(i int) int) int {
	s := 0
	for i := 0; i < n; i++ {
		s += f(i)
	}
	return s
}

func TestSampleProjectOverview(t *testing.T) {
	spec := specsFromTemplateEvent(t)["wedding"]
	o := spec.AssembleOverview(loadSQLAggregates(t))

	if o.Total != 20 || o.TotalPax != 43 {
		t.Fatalf("total = %d/%d, want 20/43", o.Total, o.TotalPax)
	}
	if o.Invitation.Sent != 14 || o.Invitation.Unsent != 6 {
		t.Errorf("invitation = %+v, want 14/6", o.Invitation)
	}
	if o.RSVP != (RSVPSummary{Attending: 12, Declined: 2, Maybe: 2, Unconfirmed: 4, AttendingPax: 27}) {
		t.Errorf("rsvp = %+v", o.RSVP)
	}
	if got := o.RSVP.Attending + o.RSVP.Declined + o.RSVP.Maybe + o.RSVP.Unconfirmed; got != o.Total {
		t.Errorf("RSVP statuses sum to %d, want %d", got, o.Total)
	}

	if len(o.Sides) != 2 || *o.Sides[0].Key != "pria" || o.Sides[0].Invitations != 10 || o.Sides[0].Pax != 22 ||
		*o.Sides[1].Key != "wanita" || o.Sides[1].Invitations != 10 || o.Sides[1].Pax != 21 {
		t.Errorf("sides = %+v", o.Sides)
	}
	want := []struct {
		key string
		inv int
	}{{"teman", 6}, {"kolega", 3}, {"keluarga", 4}, {"tetangga", 2}, {"teman_ortu", 3}, {"vip", 2}}
	if len(o.Categories) != len(want) {
		t.Fatalf("categories = %+v", o.Categories)
	}
	for i, w := range want {
		if o.Categories[i].Key != w.key || o.Categories[i].Invitations != w.inv {
			t.Errorf("category %d = %+v, want %s %d", i, o.Categories[i], w.key, w.inv)
		}
	}
	if o.Categories[0].Label != "Teman" || o.Categories[4].Label != "Teman Orang Tua" {
		t.Errorf("labels = %q, %q", o.Categories[0].Label, o.Categories[4].Label)
	}
	// Tabel PDF: tiap pihak punya enam penghitungnya sendiri; jumlah semua sel = total.
	if got := sum(len(o.SideCategories), func(i int) int { return o.SideCategories[i].Invitations }); got != o.Total {
		t.Errorf("side_categories sum = %d, want %d", got, o.Total)
	}
	cell := func(side, cat string) (int, int) {
		for _, c := range o.SideCategories {
			if c.Side != nil && *c.Side == side && c.Category == cat {
				return c.Invitations, c.Pax
			}
		}
		return 0, 0
	}
	if inv, pax := cell("pria", "teman"); inv != 3 || pax != 5 { // #2, #7, #9
		t.Errorf("pria/teman = %d/%d, want 3/5", inv, pax)
	}
	if inv, _ := cell("wanita", "teman"); inv != 3 { // #12, #17, #19
		t.Errorf("wanita/teman = %d, want 3", inv)
	}
	if inv, _ := cell("pria", "vip"); inv != 1 {
		t.Errorf("pria/vip = %d", inv)
	}
	// urutan: pihak registry dulu, lalu kategori registry
	if first := o.SideCategories[0]; *first.Side != "pria" || first.Category != "teman" {
		t.Errorf("first cell = %+v", first)
	}
}

func TestOverviewOfAnEmptyProject(t *testing.T) {
	for k, spec := range specsFromTemplateEvent(t) {
		o := spec.AssembleOverview(Aggregates{})
		if o.Total != 0 || len(o.Categories) != len(spec.Categories) || len(o.Sides) != len(spec.Sides) {
			t.Errorf("%s: %+v", k, o)
		}
		if o.SideCategories == nil || o.Sides == nil || o.Categories == nil {
			t.Errorf("%s: lists must be empty arrays, not null", k)
		}
		b, _ := json.Marshal(o)
		if strings.Contains(string(b), "null") {
			t.Errorf("%s: JSON contains null: %s", k, b)
		}
	}
}

func TestSeminarShowsItsOwnSidesAndNeverPriaWanita(t *testing.T) {
	spec := specsFromTemplateEvent(t)["seminar"]
	ext, none := "eksternal", (*string)(nil)
	o := spec.AssembleOverview(Aggregates{Total: 7, TotalPax: 7, Cells: []Cell{
		{Side: &ext, Category: "peserta", Invitations: 4, Pax: 4},
		{Side: none, Category: "peserta", Invitations: 3, Pax: 3},
	}})
	var keys []string
	for _, s := range o.Sides {
		if s.Key == nil {
			keys = append(keys, "(nil)")
		} else {
			keys = append(keys, *s.Key)
		}
	}
	if strings.Join(keys, ",") != "internal,eksternal,(nil)" {
		t.Errorf("sides = %v", keys)
	}
	if o.Sides[2].Label != "Belum ditentukan" || o.Sides[2].Invitations != 3 {
		t.Errorf("unset side = %+v", o.Sides[2])
	}
	if o.Sides[1].Label != "Peserta Eksternal" {
		t.Errorf("label = %q", o.Sides[1].Label)
	}
	if got := sum(len(o.Sides), func(i int) int { return o.Sides[i].Invitations }); got != o.Total {
		t.Errorf("sides sum = %d, want %d", got, o.Total)
	}
}

func TestEventTypesWithoutSidesNeverEmitSides(t *testing.T) {
	for _, k := range []string{"birthday", "wisuda", "other"} {
		spec := specsFromTemplateEvent(t)[k]
		o := spec.AssembleOverview(Aggregates{Total: 2, TotalPax: 3, Cells: []Cell{{Side: nil, Category: "vip", Invitations: 2, Pax: 3}}})
		if len(o.Sides) != 0 || len(o.SideCategories) != 0 {
			t.Errorf("%s: sides=%v side_categories=%v", k, o.Sides, o.SideCategories)
		}
		if o.Total != 2 {
			t.Errorf("%s: total", k)
		}
	}
}

func TestCategoryOrderIsRegistryThenFallbackThenCustomAlphabetical(t *testing.T) {
	spec := specsFromTemplateEvent(t)["other"] // umum, vip
	o := spec.AssembleOverview(Aggregates{Total: 6, TotalPax: 6, Cells: []Cell{
		{Category: "Zebra", Invitations: 1, Pax: 1}, {Category: "lainnya", Invitations: 2, Pax: 2},
		{Category: "alpha", Invitations: 1, Pax: 1}, {Category: "vip", Invitations: 2, Pax: 2},
	}})
	var keys []string
	for _, c := range o.Categories {
		keys = append(keys, c.Key)
	}
	if strings.Join(keys, ",") != "umum,vip,lainnya,alpha,Zebra" {
		t.Errorf("order = %v", keys)
	}
	if o.Categories[0].Custom || o.Categories[1].Custom || !o.Categories[2].Custom || !o.Categories[3].Custom || !o.Categories[4].Custom {
		t.Errorf("custom flags: %+v", o.Categories)
	}
	if o.Categories[2].Label != "Lainnya" || o.Categories[0].Invitations != 0 {
		t.Errorf("categories = %+v", o.Categories)
	}
	// lainnya tidak muncul bila tak dipakai
	o = spec.AssembleOverview(Aggregates{Total: 1, TotalPax: 1, Cells: []Cell{{Category: "vip", Invitations: 1, Pax: 1}}})
	if len(o.Categories) != 2 {
		t.Errorf("categories = %+v", o.Categories)
	}
}

func TestAnUnknownSideKeyStaysVisibleSoTheSumsHold(t *testing.T) {
	spec := specsFromTemplateEvent(t)["wedding"]
	x := "mantan"
	o := spec.AssembleOverview(Aggregates{Total: 1, TotalPax: 2, Cells: []Cell{{Side: &x, Category: "teman", Invitations: 1, Pax: 2}}})
	if got := sum(len(o.Sides), func(i int) int { return o.Sides[i].Invitations }); got != 1 || len(o.Sides) != 3 || o.Sides[2].Label != "mantan" {
		t.Errorf("sides = %+v", o.Sides)
	}
}

// Sel dari SQL belum tentu terurut: pihak registry lebih dulu, "belum ditentukan" terakhir, lalu kategori registry.
func TestSideCategoryCellsAreSortedWhateverTheInputOrder(t *testing.T) {
	spec := specsFromTemplateEvent(t)["wedding"]
	pria, wanita := "pria", "wanita"
	o := spec.AssembleOverview(Aggregates{Total: 4, TotalPax: 4, Cells: []Cell{
		{Side: nil, Category: "vip", Invitations: 1, Pax: 1},
		{Side: &wanita, Category: "vip", Invitations: 1, Pax: 1},
		{Side: &pria, Category: "vip", Invitations: 1, Pax: 1},
		{Side: &pria, Category: "teman", Invitations: 1, Pax: 1},
	}})
	var got []string
	for _, c := range o.SideCategories {
		s := "(nil)"
		if c.Side != nil {
			s = *c.Side
		}
		got = append(got, s+"/"+c.Category)
	}
	if strings.Join(got, " ") != "pria/teman pria/vip wanita/vip (nil)/vip" {
		t.Errorf("%v", got)
	}
}

func TestASingleGuestWithoutASideStillShowsTheUndecidedEntry(t *testing.T) {
	spec := specsFromTemplateEvent(t)["wedding"]
	o := spec.AssembleOverview(Aggregates{Total: 1, TotalPax: 2, Cells: []Cell{{Category: "teman", Invitations: 1, Pax: 2}}})
	if len(o.Sides) != 3 || o.Sides[2].Key != nil || o.Sides[2].Invitations != 1 || o.Sides[2].Pax != 2 {
		t.Errorf("%+v", o.Sides)
	}
}

func TestReceptionFiltersListTheRegistryThenTheUsedFreeText(t *testing.T) {
	w := specsFromTemplateEvent(t)["wedding"]
	f := w.ReceptionFilters(9, 4, 5, map[string]int{"vip": 2, "teman": 3, "lainnya": 1, "Rekan Bisnis": 3, "kosong": 0})

	var keys []string
	for _, c := range f.Categories {
		keys = append(keys, c.Key)
	}
	if got := strings.Join(keys, ","); got != "teman,kolega,keluarga,tetangga,teman_ortu,vip,lainnya,Rekan Bisnis" {
		t.Errorf("categories = %s", got)
	}
	if f.All != 9 || f.CheckedIn != 4 || f.Pending != 5 {
		t.Errorf("filters = %+v", f)
	}
	byKey := map[string]CategoryCount{}
	for _, c := range f.Categories {
		byKey[c.Key] = c
	}
	if byKey["teman"].Invitations != 3 || byKey["kolega"].Invitations != 0 || byKey["teman"].Label != "Teman" || byKey["teman"].Custom || !byKey["lainnya"].Custom || !byKey["Rekan Bisnis"].Custom {
		t.Errorf("categories = %+v", byKey)
	}
	for _, c := range f.Categories {
		if c.Pax != 0 {
			t.Errorf("%s has pax %d, the chips carry none", c.Key, c.Pax)
		}
	}
	// A type without categories of its own still lists the free ones; the result is never nil.
	if empty := (GuestSpec{}).ReceptionFilters(0, 0, 0, nil); empty.Categories == nil || len(empty.Categories) != 0 {
		t.Errorf("empty = %+v", empty)
	}
}

func TestSideLabel(t *testing.T) {
	w := specsFromTemplateEvent(t)["wedding"]
	pria, unknown := "pria", "kiri"
	for in, want := range map[*string]string{&pria: "Pihak Pria (CPP)", nil: "Belum ditentukan", &unknown: "kiri"} {
		if got := w.SideLabel(in); got != want {
			t.Errorf("SideLabel = %q, want %q", got, want)
		}
	}
}
