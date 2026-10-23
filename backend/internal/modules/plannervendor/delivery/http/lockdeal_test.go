package http_test

import (
	"strings"
	"testing"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

// weddingWithBudget is a wedding with the 14 template lines (estimate 0) and no vendor.
func weddingWithBudget(e *env, tok string) string {
	return createProject(e, tok, map[string]any{"templates": map[string]any{"budget_lines": true}})
}

func warningCodes(r resp) []string {
	var out []string
	for _, w := range list(r.data()["warnings"]) {
		out = append(out, w["code"].(string))
	}

	return out
}

func hasWarning(r resp, code string) bool {
	for _, c := range warningCodes(r) {
		if c == code {
			return true
		}
	}

	return false
}

// A dedicated line is linked to the vendor and takes the deal as its estimate, and no second line is made (04 BE 2.2, 2c).
func TestLockDealLinksTheDedicatedLine(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := weddingWithBudget(e, tok)
	line := budgetLines(e, tok, id)["resepsi/catering"]
	v := addVendor(e, tok, id, vendorBody("catering", "Katering Bu Minah", nil))

	r := mustStatus(t, lockDeal(e, tok, id, v["id"].(string), 25_000_000, true), 200)
	d := r.data()
	if d["budget_item_id"] != line["id"] || d["created"] != false || d["shared_item"] != nil || len(warningCodes(r)) != 0 {
		t.Fatalf("result = %v", d)
	}
	vd := d["vendor"].(map[string]any)
	if vd["status"] != "fix" || num(vd["agreed_price"]) != 25_000_000 {
		t.Errorf("vendor = %v", vd)
	}
	if p := vd["payment"].(map[string]any); p["budget_item_id"] != line["id"] || num(p["estimate"]) != 25_000_000 || p["payment_status"] != "unpaid" || num(p["remaining"]) != 25_000_000 {
		t.Errorf("payment = %v", p)
	}
	lines := budgetLines(e, tok, id)
	if len(lines) != 14 || lines["resepsi/catering"]["vendor_id"] != v["id"] || num(lines["resepsi/catering"]["estimate"]) != 25_000_000 || estimateSum(lines) != 25_000_000 {
		t.Errorf("lines: %d, catering %v", len(lines), lines["resepsi/catering"])
	}
	if lines["resepsi/catering"]["source"] != "template" {
		t.Errorf("the line keeps its source: %v", lines["resepsi/catering"]["source"])
	}

	// locking again re-prices the same line: no second line, no warning
	r = mustStatus(t, lockDeal(e, tok, id, v["id"].(string), 24_000_000, true), 200)
	lines = budgetLines(e, tok, id)
	if r.data()["budget_item_id"] != line["id"] || r.data()["created"] != false || len(lines) != 14 || num(lines["resepsi/catering"]["estimate"]) != 24_000_000 {
		t.Errorf("reprice: %v, %d lines, estimate %v", r.data(), len(lines), lines["resepsi/catering"]["estimate"])
	}

	// without sync_to_budget_items only the vendor changes
	w := addVendor(e, tok, id, vendorBody("fotografi", "Studio X", nil))
	r = mustStatus(t, lockDeal(e, tok, id, w["id"].(string), 3_000_000, false), 200)
	if r.data()["budget_item_id"] != nil || r.data()["created"] != false || r.data()["vendor"].(map[string]any)["status"] != "fix" {
		t.Errorf("no sync: %v", r.data())
	}
	if l := budgetLines(e, tok, id)["resepsi/dokumentasi"]; l["vendor_id"] != nil || num(l["estimate"]) != 0 {
		t.Errorf("no sync touched the line: %v", l)
	}
}

// A category without a line of its own gets a new line; the name is cut to the 150 characters a line name allows.
func TestLockDealMakesANewLine(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := weddingWithBudget(e, tok)
	before := estimateSum(budgetLines(e, tok, id))

	wo := addVendor(e, tok, id, vendorBody("wo", "Harmony Wedding Organizer", nil))
	r := mustStatus(t, lockDeal(e, tok, id, wo["id"].(string), 5_000_000, true), 200)
	lines := budgetLines(e, tok, id)
	l := lines["resepsi/wo"]
	if r.data()["created"] != true || r.data()["budget_item_id"] != l["id"] || len(lines) != 15 || l["vendor_id"] != wo["id"] || l["source"] != "vendor" ||
		l["category"] != "Wedding Organizer" || l["item_name"] != "Harmony Wedding Organizer" || num(l["estimate"]) != 5_000_000 || estimateSum(lines) != before+5_000_000 {
		t.Fatalf("new line: %v / %v", r.data(), l)
	}

	// a 255-character name (valid for a vendor) is cut to 150 for the line (a longer line name is a database error)
	long := strings.Repeat("N", 255)
	big := addVendor(e, tok, id, vendorBody("hand_bouquet", long, nil))
	mustStatus(t, lockDeal(e, tok, id, big["id"].(string), 900_000, true), 200)
	if l := budgetLines(e, tok, id)["resepsi/hand_bouquet"]; l == nil || len([]rune(l["item_name"].(string))) != 150 || l["item_name"] != long[:150] {
		t.Errorf("long name: %v", l)
	}
	if v := get(e, tok, base(id)+"/vendors/"+big["id"].(string)); v["vendor_name"] != long {
		t.Error("the vendor keeps its full name")
	}

	// a second vendor of a category whose line is taken gets a line without a key: nothing is overwritten
	cat1 := addVendor(e, tok, id, vendorBody("catering", "Katering A", nil))
	cat2 := addVendor(e, tok, id, vendorBody("catering", "Katering B", nil))
	mustStatus(t, lockDeal(e, tok, id, cat1["id"].(string), 20_000_000, true), 200)
	r = mustStatus(t, lockDeal(e, tok, id, cat2["id"].(string), 22_000_000, true), 200)
	if r.data()["created"] != true {
		t.Fatalf("second vendor: %v", r.data())
	}
	var a, b map[string]any
	for _, l := range list(get(e, tok, base(id)+"/budget/items")["items"]) {
		switch l["vendor_id"] {
		case cat1["id"]:
			a = l
		case cat2["id"]:
			b = l
		}
	}
	if a == nil || b == nil || a["item_key"] != "catering" || b["item_key"] != nil || b["item_name"] != "Katering B" || num(a["estimate"]) != 20_000_000 || num(b["estimate"]) != 22_000_000 {
		t.Errorf("A = %v, B = %v", a, b)
	}

	// a custom category: its derived key is the line key
	tenda := addVendor(e, tok, id, map[string]any{"sub_event_scope": "akad", "category_label": "Sewa Tenda", "vendor_name": "Tenda Jaya"})
	mustStatus(t, lockDeal(e, tok, id, tenda["id"].(string), 2_000_000, true), 200)
	if l := budgetLines(e, tok, id)["akad/sewa_tenda"]; l == nil || l["category"] != "Sewa Tenda" || num(l["estimate"]) != 2_000_000 {
		t.Errorf("custom line: %v", l)
	}
}

// A line that several categories share stands for all of them: it is reported, not linked and not changed.
func TestLockDealLeavesASharedLineAlone(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := weddingWithBudget(e, tok)
	exec(e, `UPDATE planner_budget_items SET initial_estimate = 20000000 WHERE project_id = $1 AND item_key = 'venue_dekor'`, id)
	line := budgetLines(e, tok, id)["resepsi/venue_dekor"]

	venue := addVendor(e, tok, id, vendorBody("venue", "Sasana Kriya Ballroom", nil))
	r := mustStatus(t, lockDeal(e, tok, id, venue["id"].(string), 12_000_000, true), 200)
	si, _ := r.data()["shared_item"].(map[string]any)
	if r.data()["budget_item_id"] != nil || r.data()["created"] != false || !hasWarning(r, "shared_budget_item") || len(warningCodes(r)) != 1 {
		t.Fatalf("result = %v", r.data())
	}
	if si["item_key"] != "venue_dekor" || si["budget_item_id"] != line["id"] || si["item_name"] != line["item_name"] || num(si["plan"]) != 20_000_000 || num(si["total_agreed"]) != 12_000_000 {
		t.Errorf("shared_item = %v", si)
	}
	if l := budgetLines(e, tok, id)["resepsi/venue_dekor"]; l["vendor_id"] != nil || num(l["estimate"]) != 20_000_000 {
		t.Errorf("the shared line changed: %v", l)
	}

	// the second member: the total of the group grows, the line is still untouched
	deco := addVendor(e, tok, id, vendorBody("dekorasi", "Pelaminan Mega", nil))
	r = mustStatus(t, lockDeal(e, tok, id, deco["id"].(string), 8_000_000, true), 200)
	if si := r.data()["shared_item"].(map[string]any); num(si["total_agreed"]) != 20_000_000 || num(si["plan"]) != 20_000_000 {
		t.Errorf("shared_item = %v", si)
	}
	if n := len(budgetLines(e, tok, id)); n != 14 {
		t.Errorf("%d lines, a shared deal must not add one", n)
	}

	// no line at all (deleted): the scenario line gives the name and the amount, and there is no line id
	sc := mustStatus(t, e.do(tok, "POST", base(id)+"/scenarios", map[string]any{"name": "75", "preset_key": "wedding_75jt"}), 201).data()["scenario"].(map[string]any)
	mustStatus(t, e.do(tok, "POST", base(id)+"/scenarios/apply", map[string]any{"scenario_id": sc["id"], "mode": "smart_adjust"}), 200)
	exec(e, `DELETE FROM planner_budget_items WHERE project_id = $1 AND item_key = 'rias_busana'`, id)
	mua := addVendor(e, tok, id, vendorBody("mua", "Sanggar Rias Ayu", nil))
	r = mustStatus(t, lockDeal(e, tok, id, mua["id"].(string), 3_500_000, true), 200)
	si, _ = r.data()["shared_item"].(map[string]any)
	if si["budget_item_id"] != nil || si["item_key"] != "rias_busana" || num(si["plan"]) != 5_000_000 || si["item_name"] == "" {
		t.Errorf("shared_item without a line = %v", si)
	}
}

// A line priced per unit keeps quantity × price as its estimate: the deal does not replace it.
func TestLockDealKeepsTheUnitPricedEstimate(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := weddingWithBudget(e, tok)
	exec(e, `UPDATE planner_budget_items SET pricing_mode='unit', quantity=500, unit_label='pax', unit_price=45000, initial_estimate=22500000
	         WHERE project_id = $1 AND item_key = 'catering'`, id)

	v := addVendor(e, tok, id, vendorBody("catering", "Katering Bu Minah", nil))
	r := mustStatus(t, lockDeal(e, tok, id, v["id"].(string), 25_000_000, true), 200)
	l := budgetLines(e, tok, id)["resepsi/catering"]
	if !hasWarning(r, "unit_pricing") || r.data()["budget_item_id"] != l["id"] || l["vendor_id"] != v["id"] || num(l["estimate"]) != 22_500_000 || l["pricing_mode"] != "unit" {
		t.Errorf("result %v, line %v", r.data(), l)
	}
	if num(r.data()["vendor"].(map[string]any)["agreed_price"]) != 25_000_000 {
		t.Error("the vendor keeps the deal")
	}
	// locking again (the vendor already has the line): still the same estimate
	r = mustStatus(t, lockDeal(e, tok, id, v["id"].(string), 26_000_000, true), 200)
	if !hasWarning(r, "unit_pricing") || num(budgetLines(e, tok, id)["resepsi/catering"]["estimate"]) != 22_500_000 {
		t.Errorf("second lock: %v", r.data())
	}
}

// Every scenario line and template line of a non-wedding type is in 'all'. A vendor in a real session still finds its line
// there and does not get a second one (the old rule made a second `venue` line in `registrasi`: 120 Jt on a 100 Jt budget).
func TestLockDealFallsBackToTheGeneralSession(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	spec, _ := pd.DefaultRegistry().Get(pd.EventSeminar)
	id := createProject(e, tok, map[string]any{
		"event_type": "seminar", "title": "Seminar", "target_budget": 100_000_000, "metadata": metadataFor(spec), "templates": map[string]any{"budget_lines": true},
	})
	before := budgetLines(e, tok, id)
	if before["all/venue"] == nil || num(before["all/venue"]["estimate"]) != 25_000_000 {
		t.Fatalf("venue line = %v", before["all/venue"])
	}

	venue := addVendor(e, tok, id, map[string]any{"sub_event_scope": "registrasi", "category_key": "venue", "vendor_name": "Aula Kampus"})
	r := mustStatus(t, lockDeal(e, tok, id, venue["id"].(string), 20_000_000, true), 200)
	after := budgetLines(e, tok, id)
	if r.data()["created"] != false || r.data()["budget_item_id"] != before["all/venue"]["id"] || len(after) != len(before) ||
		after["all/venue"]["vendor_id"] != venue["id"] || num(after["all/venue"]["estimate"]) != 20_000_000 {
		t.Errorf("result %v, %d lines (was %d), venue %v", r.data(), len(after), len(before), after["all/venue"])
	}
	if sum := estimateSum(after); sum != 95_000_000 {
		t.Errorf("Σ estimate = %d: the budget must not grow beyond its 100 Jt", sum)
	}

	// the line of the vendor's own session wins over the one in 'all'
	own := addVendor(e, tok, id, map[string]any{"sub_event_scope": "sesi_pagi", "category_key": "dokumentasi", "vendor_name": "Foto Sesi Pagi"})
	exec(e, `INSERT INTO planner_budget_items (id, project_id, sub_event_scope, category, item_name, item_key, source, initial_estimate, sort_order)
	         VALUES (gen_random_uuid(), $1, 'sesi_pagi', 'Dokumentasi', 'Dokumentasi sesi pagi', 'dokumentasi', 'manual', 0, 99)`, id)
	r = mustStatus(t, lockDeal(e, tok, id, own["id"].(string), 4_000_000, true), 200)
	if l := budgetLines(e, tok, id); l["sesi_pagi/dokumentasi"]["vendor_id"] != own["id"] || l["all/dokumentasi"]["vendor_id"] != nil || r.data()["created"] != false {
		t.Errorf("own session first: %v / %v", l["sesi_pagi/dokumentasi"], l["all/dokumentasi"])
	}

	// AV and streaming share one line of the seminar: reported, not linked
	av := addVendor(e, tok, id, map[string]any{"sub_event_scope": "all", "category_key": "av", "vendor_name": "AV Pro"})
	r = mustStatus(t, lockDeal(e, tok, id, av["id"].(string), 7_000_000, true), 200)
	if si, _ := r.data()["shared_item"].(map[string]any); si["item_key"] != "av_streaming" || !hasWarning(r, "shared_budget_item") {
		t.Errorf("av: %v", r.data())
	}
}

func TestLockDealValidationAndTheLineLimit(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := weddingWithBudget(e, tok)
	v := addVendor(e, tok, id, vendorBody("wo", "WO", nil))
	vid := v["id"].(string)

	for _, price := range []int64{0, -5, 10_000_000_000_001} {
		wantError(t, lockDeal(e, tok, id, vid, price, true), 400, "invalid_input", "agreed_price: must be between 1 and 10000000000000")
	}
	wantError(t, e.do(tok, "POST", base(id)+"/vendors/"+vid+"/lock-deal", `{"agreed_price": "1"}`), 400, "invalid_json", "")
	if got := get(e, tok, base(id)+"/vendors/"+vid); got["status"] != "rencana" {
		t.Errorf("a refused deal changed the vendor: %v", got)
	}
	mustStatus(t, lockDeal(e, tok, id, vid, 10_000_000_000_000, false), 200)

	// 300 budget lines is the limit of menu 03; a new line made here counts, and the whole call rolls back
	full := weddingWithBudget(e, tok)
	exec(e, `INSERT INTO planner_budget_items (id, project_id, sub_event_scope, category, item_name, source, sort_order)
	         SELECT gen_random_uuid(), $1, 'all', 'Lain', 'Pos ' || g, 'manual', 100 + g FROM generate_series(1, 286) g`, full)
	w := addVendor(e, tok, full, vendorBody("wo", "WO penuh", nil))
	wantError(t, lockDeal(e, tok, full, w["id"].(string), 5_000_000, true), 409, "budget_item_limit_reached", "")
	if got := get(e, tok, base(full)+"/vendors/"+w["id"].(string)); got["status"] != "rencana" || num(got["agreed_price"]) != 0 {
		t.Errorf("the vendor must be untouched after the refusal: %v", got)
	}
	// without sync the deal needs no line
	mustStatus(t, lockDeal(e, tok, full, w["id"].(string), 5_000_000, false), 200)
}

// A vendor that left fix lost its link; locking it again takes the same line back.
func TestLockDealAfterLeavingFix(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := weddingWithBudget(e, tok)
	v := addVendor(e, tok, id, vendorBody("catering", "Katering", nil))
	vid := v["id"].(string)
	first := mustStatus(t, lockDeal(e, tok, id, vid, 25_000_000, true), 200).data()["budget_item_id"]

	r := mustStatus(t, e.do(tok, "PATCH", base(id)+"/vendors/"+vid+"/status", map[string]any{"status": "negosiasi"}), 200)
	if !hasWarning(r, "budget_link_released") {
		t.Fatalf("warnings = %v", r.data()["warnings"])
	}
	if l := budgetLines(e, tok, id)["resepsi/catering"]; l["vendor_id"] != nil || num(l["estimate"]) != 25_000_000 || l["id"] != first {
		t.Errorf("after the release: %v", l)
	}

	again := mustStatus(t, lockDeal(e, tok, id, vid, 26_000_000, true), 200).data()
	if again["budget_item_id"] != first || again["created"] != false || len(budgetLines(e, tok, id)) != 14 || num(budgetLines(e, tok, id)["resepsi/catering"]["estimate"]) != 26_000_000 {
		t.Errorf("relock: %v", again)
	}
}

// Eight deals for vendors of one category at the same time: exactly one takes the dedicated line, the others make a line each,
// and every deal is in the budget. Without the project lock the vendors that raced for the line overwrote each other's
// link and were still told "linked".
func TestConcurrentLockDealNeverLosesALink(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := weddingWithBudget(e, tok)

	const n = 8
	ids := make([]string, n)
	for i := range ids {
		ids[i] = addVendor(e, tok, id, vendorBody("catering", "Katering "+string(rune('A'+i)), nil))["id"].(string)
	}
	statuses := concurrently(n, func(i int) int { return lockDeal(e, tok, id, ids[i], int64(10_000_000+i*1_000_000), true).status })
	for i, s := range statuses {
		if s != 200 {
			t.Fatalf("deal %d answered %d", i, s)
		}
	}

	lines := list(get(e, tok, base(id)+"/budget/items")["items"])
	if len(lines) != 14+n-1 {
		t.Errorf("%d lines, want %d", len(lines), 14+n-1)
	}
	byVendor := map[string]map[string]any{}
	var total int64
	for _, l := range lines {
		if vid, _ := l["vendor_id"].(string); vid != "" {
			byVendor[vid] = l
			total += num(l["estimate"])
		}
	}
	var want int64
	for i := range ids {
		want += int64(10_000_000 + i*1_000_000)
		l := byVendor[ids[i]]
		if l == nil || num(l["estimate"]) != int64(10_000_000+i*1_000_000) {
			t.Errorf("vendor %d: line %v", i, l)
		}
	}
	if len(byVendor) != n || total != want {
		t.Errorf("%d vendors have a line, Σ %d, want %d", len(byVendor), total, want)
	}
	keyed := 0
	for _, l := range byVendor {
		if l["item_key"] == "catering" {
			keyed++
		}
	}
	if keyed != 1 {
		t.Errorf("%d lines took the dedicated key", keyed)
	}
}

// The same race when no line is there for the category (it was deleted): every deal makes a line, and only the first may take the
// key. Under the project lock the others see the key taken and make a line without one; without it they all try to insert the
// same (session, item_key) and all but one fail.
func TestConcurrentLockDealWithoutADedicatedLine(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := weddingWithBudget(e, tok)
	exec(e, `DELETE FROM planner_budget_items WHERE project_id = $1 AND item_key = 'catering'`, id)

	const n = 8
	ids := make([]string, n)
	for i := range ids {
		ids[i] = addVendor(e, tok, id, vendorBody("catering", "Katering "+string(rune('A'+i)), nil))["id"].(string)
	}
	statuses := concurrently(n, func(i int) int { return lockDeal(e, tok, id, ids[i], 5_000_000, true).status })
	for i, s := range statuses {
		if s != 200 {
			t.Fatalf("deal %d answered %d", i, s)
		}
	}
	if got := len(budgetLines(e, tok, id)); got != 12+n { // the template has a catering line in lamaran and one in resepsi; both were deleted
		t.Errorf("%d lines, want %d", got, 12+n)
	}
	if k := count(e, `SELECT count(*) FROM planner_budget_items WHERE project_id = $1 AND sub_event_scope = 'resepsi' AND item_key = 'catering'`, id); k != 1 {
		t.Errorf("%d lines took the key", k)
	}
	if k := count(e, `SELECT count(*) FROM planner_budget_items WHERE project_id = $1 AND vendor_id IS NOT NULL`, id); k != n {
		t.Errorf("%d vendors have a line", k)
	}
}

// The 300-line limit of menu 03 holds for the lines made here, also under concurrency: of eight deals that each need a new
// line at 296 lines exactly four fit, and the other four are refused whole.
func TestConcurrentLockDealRespectsTheLineLimit(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := weddingWithBudget(e, tok)
	exec(e, `INSERT INTO planner_budget_items (id, project_id, sub_event_scope, category, item_name, source, sort_order)
	         SELECT gen_random_uuid(), $1, 'all', 'Lain', 'Pos ' || g, 'manual', 100 + g FROM generate_series(1, 282) g`, id)

	const n = 8
	ids := make([]string, n)
	for i := range ids {
		ids[i] = addVendor(e, tok, id, map[string]any{"sub_event_scope": "resepsi", "category_label": "Lain " + string(rune('A'+i)), "vendor_name": "Vendor " + string(rune('A'+i))})["id"].(string)
	}
	statuses := concurrently(n, func(i int) int { return lockDeal(e, tok, id, ids[i], 1_000_000, true).status })
	ok, refused := 0, 0
	for _, s := range statuses {
		switch s {
		case 200:
			ok++
		case 409:
			refused++
		}
	}
	if ok != 4 || refused != 4 || count(e, `SELECT count(*) FROM planner_budget_items WHERE project_id = $1`, id) != 300 {
		t.Errorf("%d deals went through, %d refused: %v", ok, refused, statuses)
	}
	if k := count(e, `SELECT count(*) FROM planner_vendors WHERE project_id = $1 AND status = 'fix'`, id); k != 4 {
		t.Errorf("%d vendors are fix, a refused deal must leave its vendor alone", k)
	}
}
