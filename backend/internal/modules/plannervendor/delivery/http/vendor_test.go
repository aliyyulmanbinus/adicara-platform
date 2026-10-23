package http_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestVendorLifecycle(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, map[string]any{"templates": map[string]any{"budget_lines": true}})

	// a candidate: status rencana, no limit (nothing active yet), no payment
	v := addVendor(e, tok, id, vendorBody("catering", "Katering A", map[string]any{
		"pic_name": "Bu Ani", "phone_number": "0812-9876-5432", "portfolio_url": "https://example.com/a", "notes": "baris 1\nbaris 2",
	}))
	vid := v["id"].(string)
	if v["status"] != "rencana" || v["category_label"] != "Catering" || v["limit_source"] != "none" || num(v["agreed_price"]) != 0 || v["payment"] != nil ||
		v["formatted_wa_number"] != "6281298765432" || v["phone_number"] != "0812-9876-5432" || num(v["sort_order"]) != 1 || v["notes"] != "baris 1\nbaris 2" {
		t.Fatalf("vendor = %v", v)
	}
	if r := e.do(tok, "POST", base(id)+"/vendors", vendorBody("mua", "MUA B", nil)); r.status != 201 || r.data()["warnings"] == nil || len(list(r.data()["warnings"])) != 0 {
		t.Errorf("warnings must be an empty array: %s", r.body)
	}

	// read it back alone and in the list
	if got := get(e, tok, base(id)+"/vendors/"+vid); got["vendor_name"] != "Katering A" {
		t.Errorf("GET = %v", got)
	}
	ov := vendorOverview(e, tok, id)
	if len(list(ov["vendors"])) != 2 || len(list(ov["categories"])) != 10 || list(ov["limit_groups"]) == nil {
		t.Errorf("overview = %v", ov)
	}

	// PUT replaces: what is not sent is cleared
	r := mustStatus(t, e.do(tok, "PUT", base(id)+"/vendors/"+vid, map[string]any{
		"sub_event_scope": "akad", "category_key": "catering", "vendor_name": "Katering A+", "agreed_price": 18_000_000, "status": "negosiasi",
	}), 200)
	pv := r.data()["vendor"].(map[string]any)
	if pv["vendor_name"] != "Katering A+" || pv["sub_event_scope"] != "akad" || pv["status"] != "negosiasi" || num(pv["agreed_price"]) != 18_000_000 ||
		pv["pic_name"] != nil || pv["phone_number"] != nil || pv["formatted_wa_number"] != nil || pv["portfolio_url"] != nil || pv["notes"] != nil {
		t.Errorf("after PUT = %v", pv)
	}
	if len(list(r.data()["warnings"])) != 0 {
		t.Errorf("warnings = %v", r.data()["warnings"])
	}

	// the pipeline: fix needs a deal, one click moves it
	mustStatus(t, e.do(tok, "PATCH", base(id)+"/vendors/"+vid+"/status", map[string]any{"status": "fix"}), 200)
	zero := addVendor(e, tok, id, vendorBody("mc", "MC C", nil))["id"].(string)
	wantError(t, e.do(tok, "PATCH", base(id)+"/vendors/"+zero+"/status", map[string]any{"status": "fix"}), 400, "invalid_input", "agreed_price: must be greater than 0 when status is fix")
	wantError(t, e.do(tok, "PATCH", base(id)+"/vendors/"+zero+"/status", map[string]any{"status": "done"}), 400, "invalid_input", "status: must be one of rencana, negosiasi, fix, batal")
	wantError(t, e.do(tok, "PATCH", base(id)+"/vendors/"+zero+"/status", map[string]any{}), 400, "invalid_input", "status: must be one of rencana, negosiasi, fix, batal")
	mustStatus(t, e.do(tok, "PATCH", base(id)+"/vendors/"+zero+"/status", map[string]any{"status": "batal"}), 200)

	// filters: the vendor list is filtered, the KPI and the categories describe the whole project
	for query, want := range map[string]int{"?status=fix": 1, "?status=batal": 1, "?scope=akad": 1, "?scope=all": 0, "?scope=resepsi": 2, "?category=catering": 1, "?category=mua&status=rencana": 1, "?category=photo": 0} {
		got := get(e, tok, base(id)+"/vendors"+query)
		if n := len(list(got["vendors"])); n != want {
			t.Errorf("%s: %d vendors, want %d", query, n, want)
		}
		if num(got["kpi"].(map[string]any)["fixed_vendors"]) != 1 || len(list(got["categories"])) != 10 {
			t.Errorf("%s: the kpi must stay whole-project: %v", query, got["kpi"])
		}
	}
	for _, bad := range []string{"?status=x", "?scope=A%20B", "?category=Has%20Space"} {
		if r := e.do(tok, "GET", base(id)+"/vendors"+bad, nil); r.status != 400 || r.code() != "invalid_input" {
			t.Errorf("%s: %d %s", bad, r.status, r.body)
		}
	}

	// delete: 204, then 404
	mustStatus(t, e.do(tok, "DELETE", base(id)+"/vendors/"+zero, nil), 204)
	wantError(t, e.do(tok, "DELETE", base(id)+"/vendors/"+zero, nil), 404, "not_found", "vendor not found")
	wantError(t, e.do(tok, "GET", base(id)+"/vendors/"+zero, nil), 404, "not_found", "vendor not found")
	wantError(t, e.do(tok, "PUT", base(id)+"/vendors/"+uuid.NewString(), vendorBody("mua", "x", nil)), 404, "not_found", "vendor not found")
	wantError(t, e.do(tok, "PATCH", base(id)+"/vendors/"+uuid.NewString()+"/status", map[string]any{"status": "fix"}), 404, "not_found", "vendor not found")
	wantError(t, e.do(tok, "POST", base(id)+"/vendors/"+uuid.NewString()+"/lock-deal", map[string]any{"agreed_price": 1}), 404, "not_found", "vendor not found")
}

func TestVendorValidation(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	post := func(over map[string]any) resp {
		return e.do(tok, "POST", base(id)+"/vendors", vendorBody("mua", "MUA Ayu", over))
	}
	long := func(n int) string { return strings.Repeat("a", n) }

	for _, c := range []struct {
		name string
		body resp
		code string
		msg  string
	}{
		{"unknown session", post(map[string]any{"sub_event_scope": "nope"}), "invalid_scope", "sub_event_scope: unknown session"},
		{"bad session key", post(map[string]any{"sub_event_scope": "A B"}), "invalid_scope", "sub_event_scope: unknown session"},
		{"category not in the registry", post(map[string]any{"category_key": "kuliner"}), "invalid_input", "category_key: must be a registry category key"},
		{"no category at all", post(map[string]any{"category_key": ""}), "invalid_input", "category_label: required when category_key is empty"},
		{"category label too long", post(map[string]any{"category_key": "", "category_label": long(61)}), "invalid_input", "category_label: must be at most 60 characters"},
		{"empty name", post(map[string]any{"vendor_name": "  "}), "invalid_input", "vendor_name: must not be empty"},
		{"name 256", post(map[string]any{"vendor_name": long(256)}), "invalid_input", "vendor_name: must be at most 255 characters"},
		{"control in name", post(map[string]any{"vendor_name": "a\x00b"}), "invalid_input", "vendor_name: must not contain control characters"},
		{"pic too long", post(map[string]any{"pic_name": long(256)}), "invalid_input", "pic_name: must be at most 255 characters"},
		{"phone with letters", post(map[string]any{"phone_number": "0812abc"}), "invalid_phone", "phone_number: must be an Indonesian phone number (08…, +62…)"},
		{"foreign phone", post(map[string]any{"phone_number": "+65 9123 4567"}), "invalid_phone", ""},
		{"phone too long", post(map[string]any{"phone_number": strings.Repeat("1", 41)}), "invalid_input", "phone_number: must be at most 40 characters"},
		{"url without scheme", post(map[string]any{"portfolio_url": "example.com"}), "invalid_input", "portfolio_url: must be an http or https URL"},
		{"url with javascript", post(map[string]any{"portfolio_url": "javascript:alert(1)"}), "invalid_input", "portfolio_url: must be an http or https URL"},
		{"url too long", post(map[string]any{"portfolio_url": "https://example.com/" + long(500)}), "invalid_input", "portfolio_url: must be an http or https URL"},
		{"negative limit", post(map[string]any{"budget_limit": -1}), "invalid_input", "budget_limit: must be between 0 and 10000000000000"},
		{"price above the cap", post(map[string]any{"agreed_price": 10_000_000_000_001}), "invalid_input", "agreed_price: must be between 0 and 10000000000000"},
		{"bad status", post(map[string]any{"status": "lunas"}), "invalid_input", "status: must be one of rencana, negosiasi, fix, batal"},
		{"fix without a deal", post(map[string]any{"status": "fix"}), "invalid_input", "agreed_price: must be greater than 0 when status is fix"},
		{"notes too long", post(map[string]any{"notes": long(2001)}), "invalid_input", "notes: must be at most 2000 characters"},
		{"notes with a control character", post(map[string]any{"notes": "a\x07b"}), "invalid_input", "notes: must not contain control characters"},
		{"malformed json", e.do(tok, "POST", base(id)+"/vendors", "{not json"), "invalid_json", "invalid JSON body"},
		{"wrong type", e.do(tok, "POST", base(id)+"/vendors", `{"vendor_name": 5}`), "invalid_json", "invalid JSON body"},
		{"price as a fraction", e.do(tok, "POST", base(id)+"/vendors", `{"vendor_name": "x", "category_key": "mua", "agreed_price": 1.5}`), "invalid_json", ""},
	} {
		if c.body.status != 400 || c.body.code() != c.code || (c.msg != "" && c.body.message() != c.msg) {
			t.Errorf("%s: got %d %s %q; body %s", c.name, c.body.status, c.body.code(), c.body.message(), c.body.body)
		}
	}
	if n := count(e, `SELECT count(*) FROM planner_vendors`); n != 0 {
		t.Errorf("%d vendors were stored by rejected requests", n)
	}

	// the boundaries are accepted
	ok := post(map[string]any{"vendor_name": long(255), "pic_name": long(255), "notes": long(2000), "budget_limit": 10_000_000_000_000, "agreed_price": 10_000_000_000_000})
	mustStatus(t, ok, 201)
	// 255 rune names: counted in characters, not bytes
	mustStatus(t, post(map[string]any{"vendor_name": strings.Repeat("é", 255)}), 201)
}

func TestVendorCategoriesAreRegistryOrCustom(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)

	// a custom category: the key is derived from the label
	c := addVendor(e, tok, id, map[string]any{"category_label": "Dekorasi & Tenda", "vendor_name": "Tenda Jaya"})
	if c["category_key"] != "dekorasi_tenda" || c["category_label"] != "Dekorasi & Tenda" || c["limit_source"] != "none" || c["sub_event_scope"] != "all" {
		t.Errorf("custom = %v", c)
	}
	// a label that derives a registry key merges with that category and takes its label
	m := addVendor(e, tok, id, map[string]any{"category_label": " venue ", "vendor_name": "Aula"})
	if m["category_key"] != "venue" || m["category_label"] != "Venue" {
		t.Errorf("merged = %v", m)
	}
	// a registry key wins over a label that came with it
	r := addVendor(e, tok, id, map[string]any{"category_key": "mua", "category_label": "Salon", "vendor_name": "Ayu"})
	if r["category_key"] != "mua" || r["category_label"] != "MUA" {
		t.Errorf("registry = %v", r)
	}
	// emoji-only label: the key falls back to "kategori"
	if x := addVendor(e, tok, id, map[string]any{"category_label": "🎉", "vendor_name": "Pesta"}); x["category_key"] != "kategori" {
		t.Errorf("emoji = %v", x)
	}

	ov := vendorOverview(e, tok, id)
	cats := list(ov["categories"])
	if len(cats) != 12 || cats[10]["key"] != "dekorasi_tenda" || cats[10]["custom"] != true || cats[11]["key"] != "kategori" || cats[0]["custom"] != false {
		t.Errorf("categories = %v", cats)
	}
	if k := ov["kpi"].(map[string]any); num(k["categories_total"]) != 12 || num(k["categories_fixed"]) != 0 {
		t.Errorf("kpi = %v", k)
	}

	// skipping a category is Menu 01's PATCH; the overview follows
	mustStatus(t, e.do(tok, "PATCH", base(id), map[string]any{"skipped_vendor_categories": []string{"wo", "hand_bouquet"}}), 200)
	ov = vendorOverview(e, tok, id)
	if k := ov["kpi"].(map[string]any); num(k["categories_total"]) != 10 || num(k["skipped"]) != 2 {
		t.Errorf("kpi after skipping = %v", k)
	}
	var skipped int
	for _, c := range list(ov["categories"]) {
		if c["skipped"] == true {
			skipped++
		}
	}
	if skipped != 2 {
		t.Errorf("%d categories flagged as skipped", skipped)
	}
}

func TestVendorLimitOf200(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	exec(e, `INSERT INTO planner_vendors (id, project_id, sub_event_scope, category_key, category_label, vendor_name, sort_order)
	         SELECT gen_random_uuid(), $1, 'all', 'venue', 'Venue', 'V' || g, g FROM generate_series(1, 199) g`, id)

	mustStatus(t, e.do(tok, "POST", base(id)+"/vendors", vendorBody("venue", "ke-200", nil)), 201)
	wantError(t, e.do(tok, "POST", base(id)+"/vendors", vendorBody("venue", "ke-201", nil)), 409, "vendor_limit_reached", "")
	if n := count(e, `SELECT count(*) FROM planner_vendors`); n != 200 {
		t.Errorf("%d vendors", n)
	}

	// the limit holds under concurrency: 10 racing requests at 190 vendors leave exactly 200
	id2 := createProject(e, tok, map[string]any{"title": "Dua"})
	exec(e, `INSERT INTO planner_vendors (id, project_id, sub_event_scope, category_key, category_label, vendor_name, sort_order)
	         SELECT gen_random_uuid(), $1, 'all', 'venue', 'Venue', 'V' || g, g FROM generate_series(1, 195) g`, id2)
	statuses := concurrently(10, func(i int) int {
		return e.do(tok, "POST", base(id2)+"/vendors", vendorBody("venue", "R", nil)).status
	})
	ok, full := 0, 0
	for _, s := range statuses {
		switch s {
		case 201:
			ok++
		case 409:
			full++
		}
	}
	if ok != 5 || full != 5 || count(e, `SELECT count(*) FROM planner_vendors WHERE project_id=$1`, id2) != 200 {
		t.Errorf("created %d, refused %d: %v", ok, full, statuses)
	}
	// and the order stays unique under the same race
	if n := count(e, `SELECT count(DISTINCT sort_order) FROM planner_vendors WHERE project_id=$1`, id2); n != 200 {
		t.Errorf("%d distinct sort_order values for 200 vendors", n)
	}
}

// Deleting a vendor leaves the things that pointed at it: tasks, budget lines, shopping items (FK SET NULL).
func TestDeleteVendorKeepsReferrers(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)

	ov := vendorOverview(e, tok, id)
	cat := vendorOf(t, ov, "Katering Bu Minah (500 pax)")["id"].(string)
	venue := vendorOf(t, ov, "Sasana Kriya Ballroom")["id"].(string)
	item := list(shoppingOverview(e, tok, id)["items"])[0]["id"].(string)
	mustStatus(t, e.do(tok, "PUT", base(id)+"/shopping-items/"+item, map[string]any{
		"sub_event_scope": "all", "category": "Perangkat Alat Solat", "item_name": "Mukena", "estimated_price": 194000, "vendor_id": cat, "status": "selesai",
	}), 200)

	linesBefore := budgetLines(e, tok, id)
	mustStatus(t, e.do(tok, "DELETE", base(id)+"/vendors/"+cat, nil), 204)
	mustStatus(t, e.do(tok, "DELETE", base(id)+"/vendors/"+venue, nil), 204)

	after := budgetLines(e, tok, id)
	if len(after) != len(linesBefore) || after["resepsi/catering"] == nil || after["resepsi/catering"]["vendor_id"] != nil || num(after["resepsi/catering"]["estimate"]) != 25_000_000 {
		t.Errorf("the catering line after the delete: %v", after["resepsi/catering"])
	}
	if got := list(shoppingOverview(e, tok, id)["items"])[0]; got["vendor_id"] != nil || got["vendor_name"] != nil || got["item_name"] != "Mukena" {
		t.Errorf("shopping item after the delete: %v", got)
	}
	for _, task := range list(get(e, tok, base(id)+"/tasks")["tasks"]) {
		if task["template_key"] == "wedding.visit_venue_pernikahan" && (task["vendor_id"] != nil || task["title"] != "Visit Venue Pernikahan") {
			t.Errorf("task after the delete: %v", task)
		}
	}
	if k := vendorOverview(e, tok, id)["kpi"].(map[string]any); num(k["fixed_vendors"]) != 5 {
		t.Errorf("kpi = %v", k)
	}
}

// PUT warns when the deal moves away from the estimate of the linked line, and releases the link when the vendor leaves fix.
func TestVendorUpdateWarnings(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)
	cat := vendorOf(t, vendorOverview(e, tok, id), "Katering Bu Minah (500 pax)")
	cid := cat["id"].(string)
	body := func(price int64, status string) map[string]any {
		return map[string]any{"sub_event_scope": "resepsi", "category_key": "catering", "vendor_name": "Katering Bu Minah (500 pax)", "agreed_price": price, "status": status}
	}

	// the price moves to 27 Jt but the line still says 25 Jt: a warning, and the line is not touched
	r := mustStatus(t, e.do(tok, "PUT", base(id)+"/vendors/"+cid, body(27_000_000, "fix")), 200)
	w := list(r.data()["warnings"])
	if len(w) != 1 || w[0]["code"] != "estimate_differs" || num(w[0]["count"]) != 1 {
		t.Errorf("warnings = %v", w)
	}
	if l := budgetLines(e, tok, id)["resepsi/catering"]; num(l["estimate"]) != 25_000_000 || l["vendor_id"] != cid {
		t.Errorf("the line must stay as it was: %v", l)
	}
	// the same price again: nothing changed, nothing to warn about
	if w := list(mustStatus(t, e.do(tok, "PUT", base(id)+"/vendors/"+cid, body(27_000_000, "fix")), 200).data()["warnings"]); len(w) != 0 {
		t.Errorf("unchanged price warned: %v", w)
	}

	// leaving fix releases the line (it stays, as a plain line) and says so
	r = mustStatus(t, e.do(tok, "PUT", base(id)+"/vendors/"+cid, body(27_000_000, "negosiasi")), 200)
	w = list(r.data()["warnings"])
	if len(w) != 1 || w[0]["code"] != "budget_link_released" || num(w[0]["count"]) != 1 || r.data()["vendor"].(map[string]any)["payment"] != nil {
		t.Errorf("warnings = %v, vendor = %v", w, r.data()["vendor"])
	}
	if l := budgetLines(e, tok, id)["resepsi/catering"]; l["vendor_id"] != nil || num(l["paid"]) != 15_000_000 {
		t.Errorf("the released line: %v", l)
	}

	// PATCH …/status does the same, and has nothing to release the second time
	if w := list(mustStatus(t, e.do(tok, "PATCH", base(id)+"/vendors/"+cid+"/status", map[string]any{"status": "rencana"}), 200).data()["warnings"]); len(w) != 0 {
		t.Errorf("nothing to release: %v", w)
	}
	photo := vendorOf(t, vendorOverview(e, tok, id), "Studio Cahaya")["id"].(string)
	w = list(mustStatus(t, e.do(tok, "PATCH", base(id)+"/vendors/"+photo+"/status", map[string]any{"status": "batal"}), 200).data()["warnings"])
	if len(w) != 1 || w[0]["code"] != "budget_link_released" {
		t.Errorf("PATCH warnings = %v", w)
	}
}
