package http_test

import (
	"testing"
)

// TestSampleProjectVendorGolden checks the figures of Wedding_Planner.pdf p. 9-10 and 12 through the whole stack: the
// sample project is created by menu 01 with the templates of menus 02-04 and the overlays on top, then read back through
// the read endpoints of this module and the dashboard (04 BE 7.1, the golden test of the contract chapter 9).
func TestSampleProjectVendorGolden(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)

	ov := vendorOverview(e, tok, id)
	k := ov["kpi"].(map[string]any)
	for key, want := range map[string]int64{
		"categories_total": 10, "categories_fixed": 7, "skipped": 0, "fixed_vendors": 7, "pipeline": 3,
		"fixed_total": 58_000_000, "pipeline_offers": 4_300_000, "groups_total": 3, "groups_over": 0,
	} {
		if num(k[key]) != want {
			t.Errorf("kpi.%s = %d, want %d", key, num(k[key]), want)
		}
	}
	if len(list(ov["vendors"])) != 10 || len(list(ov["categories"])) != 10 {
		t.Fatalf("%d vendors, %d categories", len(list(ov["vendors"])), len(list(ov["categories"])))
	}

	// the limit of each vendor comes from the active scenario (75 Jt) - nothing is stored on the vendor
	cat := vendorOf(t, ov, "Katering Bu Minah (500 pax)")
	if cat["limit_source"] != "scenario" || num(cat["budget_limit"]) != 30_000_000 || num(cat["agreed_price"]) != 25_000_000 || num(cat["price_delta"]) != 5_000_000 {
		t.Errorf("catering = %v", cat)
	}
	pay := cat["payment"].(map[string]any)
	if pay["payment_status"] != "partial" || num(pay["estimate"]) != 25_000_000 || num(pay["paid"]) != 15_000_000 || num(pay["remaining"]) != 10_000_000 {
		t.Errorf("catering payment = %v", pay)
	}
	photo := vendorOf(t, ov, "Studio Cahaya")
	if photo["limit_source"] != "scenario" || num(photo["budget_limit"]) != 3_000_000 || num(photo["price_delta"]) != 0 || photo["payment"].(map[string]any)["payment_status"] != "dp_paid" {
		t.Errorf("photography = %v", photo)
	}
	venue := vendorOf(t, ov, "Sasana Kriya Ballroom")
	if venue["limit_source"] != "group" || venue["limit_group"] != "venue_dekor" || venue["payment"] != nil || num(venue["budget_limit"]) != 0 || venue["price_delta"] != nil {
		t.Errorf("venue = %v", venue)
	}
	if wo := vendorOf(t, ov, "Harmony Wedding Organizer"); wo["limit_source"] != "none" || wo["status"] != "fix" {
		t.Errorf("wedding organizer = %v", wo)
	}
	if v := vendorOf(t, ov, "Kang Dedi Master MC"); v["status"] != "negosiasi" || v["formatted_wa_number"] != "6281200000009" || v["phone_number"] != "081200000009" || v["pic_name"] != nil {
		t.Errorf("mc = %v", v)
	}

	// the shared lines are read at group level, with the payment status of the line the PDF shows
	groups := map[string]map[string]any{}
	for _, g := range list(ov["limit_groups"]) {
		groups[g["item_key"].(string)] = g
	}
	for key, want := range map[string][5]any{
		"venue_dekor": {int64(20_000_000), int64(20_000_000), "dp_paid", int64(900_000), 2},
		"rias_busana": {int64(5_000_000), int64(5_000_000), "dp_paid", int64(600_000), 2},
		"hiburan":     {int64(3_400_000), int64(3_400_000), "settled", int64(3_400_000), 2},
	} {
		g, line := groups[key], map[string]any(nil)
		if g == nil {
			t.Fatalf("no group %s in %v", key, ov["limit_groups"])
		}
		line, _ = g["line"].(map[string]any)
		if num(g["limit"]) != want[0] || num(g["total_agreed"]) != want[1] || num(g["delta"]) != 0 || int(num(g["members"])) != want[4] || int(num(g["priced"])) != want[4] ||
			line == nil || line["payment_status"] != want[2] || num(line["paid"]) != want[3].(int64) {
			t.Errorf("group %s = %v", key, g)
		}
	}

	// the budget keeps the PDF figures: lock-deal was not run by the overlay
	if sum := estimateSum(budgetLines(e, tok, id)); sum != 74_150_000 {
		t.Errorf("budget estimate = %d, want 74.150.000", sum)
	}
	lines := budgetLines(e, tok, id)
	if lines["resepsi/catering"]["vendor_id"] != cat["id"] || lines["resepsi/dokumentasi"]["vendor_id"] != photo["id"] || lines["resepsi/venue_dekor"]["vendor_id"] != nil {
		t.Errorf("links: catering %v, dokumentasi %v, venue_dekor %v", lines["resepsi/catering"]["vendor_id"], lines["resepsi/dokumentasi"]["vendor_id"], lines["resepsi/venue_dekor"]["vendor_id"])
	}

	// the task "Visit Venue Pernikahan" points at the venue vendor (menu 02 shows its name and WhatsApp)
	found := false
	for _, task := range list(get(e, tok, base(id)+"/tasks")["tasks"]) {
		if task["template_key"] == "wedding.visit_venue_pernikahan" {
			found = true
			if task["vendor_id"] != venue["id"] || task["vendor_name"] != "Sasana Kriya Ballroom" {
				t.Errorf("task = %v", task)
			}
		}
	}
	if !found {
		t.Error("the task Visit Venue Pernikahan is missing")
	}

	// the shopping list: 37 items, Rp 4.578.500; 16 finished = 2.063.800, 21 in progress
	sh := shoppingOverview(e, tok, id)
	sk := sh["kpi"].(map[string]any)
	for key, want := range map[string]int64{"estimated_total": 4_578_500, "actual_total": 2_063_800, "done": 16, "in_progress": 21, "not_started": 0, "total": 37} {
		if num(sk[key]) != want {
			t.Errorf("shopping kpi.%s = %d, want %d", key, num(sk[key]), want)
		}
	}
	if sk["souvenir_stock"] != nil {
		t.Errorf("the sample project has no souvenir item: %v", sk["souvenir_stock"])
	}
	items := list(sh["items"])
	if len(items) != 37 || items[0]["item_name"] != "Mukena" || items[0]["status"] != "selesai" || num(items[0]["actual_price"]) != 194_000 ||
		items[1]["item_name"] != "Al-Quran" || items[1]["status"] != "proses" || num(items[1]["actual_price"]) != 0 || items[36]["item_name"] != "Box Seserahan (10 Kotak)" {
		t.Errorf("items: %d, first %v, second %v", len(items), items[0], items[1])
	}
	if items[0]["template_key"] != "wedding.seserahan_mukena" || items[0]["purchase_location"] != nil || items[0]["sub_event_scope"] != "all" || num(items[0]["quantity"]) != 1 {
		t.Errorf("first item = %v", items[0])
	}
	subtotal := map[string]int64{}
	for _, it := range items {
		subtotal[it["category"].(string)] += num(it["estimated_total"])
	}
	for cat, want := range map[string]int64{
		"Perangkat Alat Solat": 597_300, "Make Up & Skin Care": 1_268_900, "Peralatan Mandi": 990_700,
		"Pakaian Dalam": 517_400, "Pakaian Luar": 820_200, "Kebutuhan Lain": 384_000,
	} {
		if subtotal[cat] != want {
			t.Errorf("subtotal %s = %d, want %d", cat, subtotal[cat], want)
		}
	}

	// the rollup is off: the "seserahan" line keeps the PDF figures and says it holds payments
	if len(list(sh["rollups"])) != 0 {
		t.Errorf("rollups = %v", sh["rollups"])
	}
	dt := sh["default_target"].(map[string]any)
	if dt["item_key"] != "seserahan" || dt["rollup"] != false || dt["has_payments"] != true || num(dt["estimate"]) != 3_500_000 || num(dt["paid"]) != 3_500_000 || dt["budget_item_id"] == nil || dt["sub_event_scope"] != "resepsi" {
		t.Errorf("default_target = %v", dt)
	}

	// the dashboard widget of menu 01 is filled, and the wizard offers the shopping template for the wedding only
	dash := get(e, tok, base(id)+"/dashboard")["vendors"].(map[string]any)
	for key, want := range map[string]int64{"categories_total": 10, "categories_fixed": 7, "skipped": 0, "fixed_vendors": 7, "pipeline": 3} {
		if num(dash[key]) != want {
			t.Errorf("dashboard.vendors.%s = %d, want %d", key, num(dash[key]), want)
		}
	}
	for _, et := range list(mustStatus(t, e.do(tok, "GET", "/v1/planner/event-types", nil), 200).json()["data"]) {
		avail := et["templates_available"].(map[string]any)
		if (et["key"] == "wedding") != (avail["shopping_list"] == true) {
			t.Errorf("%v: shopping_list = %v", et["key"], avail["shopping_list"])
		}
	}
}
