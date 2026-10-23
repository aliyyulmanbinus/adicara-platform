package http_test

import (
	"testing"

	"github.com/google/uuid"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

func linkToBudget(e *env, tok, id string, body map[string]any) resp {
	e.t.Helper()

	return e.do(tok, "POST", base(id)+"/shopping-items/link-to-budget", body)
}

func unlinkFromBudget(e *env, tok, id, lineID string) resp {
	e.t.Helper()

	return e.do(tok, "POST", base(id)+"/shopping-items/unlink-from-budget", map[string]any{"budget_item_id": lineID})
}

// weddingWithShopping is a wedding with the 14 template lines and the 37 template items (nothing paid, nothing rolled up).
func weddingWithShopping(e *env, tok string) string {
	return createProject(e, tok, map[string]any{"templates": map[string]any{"budget_lines": true, "shopping_list": true}})
}

func lineOf(e *env, tok, id, key string) map[string]any { return budgetLines(e, tok, id)[key] }

// The line "seserahan" computes its estimate and its paid total from the list: the total is never copied, the view sums
// the items that point at the line (04 BE 2.4).
func TestLinkToBudgetRollsTheListUpIntoTheDefaultLine(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := weddingWithShopping(e, tok)

	dt := shoppingOverview(e, tok, id)["default_target"].(map[string]any)
	line := lineOf(e, tok, id, "resepsi/seserahan")
	if dt["budget_item_id"] != line["id"] || dt["rollup"] != false || dt["has_payments"] != false || num(dt["estimate"]) != 0 || dt["item_name"] != line["item_name"] {
		t.Fatalf("default_target = %v", dt)
	}

	r := mustStatus(t, linkToBudget(e, tok, id, map[string]any{}), 200)
	res := r.data()
	l := res["line"].(map[string]any)
	if res["created"] != false || num(res["linked"]) != 37 || l["budget_item_id"] != line["id"] || num(l["estimate"]) != 4_578_500 || num(l["paid"]) != 0 || num(l["linked_items"]) != 37 {
		t.Fatalf("result = %v", res)
	}
	if got := lineOf(e, tok, id, "resepsi/seserahan"); got["rollup_source"] != "shopping" || num(got["estimate"]) != 4_578_500 || got["payment_status"] != "unpaid" {
		t.Errorf("line = %v", got)
	}
	if sum := estimateSum(budgetLines(e, tok, id)); sum != 4_578_500 {
		t.Errorf("the budget now holds %d", sum)
	}
	ov := shoppingOverview(e, tok, id)
	if roll := list(ov["rollups"]); len(roll) != 1 || roll[0]["budget_item_id"] != line["id"] || num(roll[0]["linked_items"]) != 37 || num(roll[0]["estimate"]) != 4_578_500 {
		t.Errorf("rollups = %v", ov["rollups"])
	}
	if dt := ov["default_target"].(map[string]any); dt["rollup"] != true {
		t.Errorf("default_target = %v", dt)
	}
	for _, it := range list(ov["items"]) {
		if it["budget_item_id"] != line["id"] {
			t.Fatalf("an item is not linked: %v", it)
		}
	}

	// again: idempotent
	again := mustStatus(t, linkToBudget(e, tok, id, map[string]any{"budget_item_id": line["id"]}), 200).data()
	if num(again["linked"]) != 0 || num(again["line"].(map[string]any)["estimate"]) != 4_578_500 {
		t.Errorf("second call = %v", again)
	}

	// an item made later joins only when it says which line (the banner of the screen fills that in)
	loose := addItem(e, tok, id, itemBody("Kebutuhan Lain", "Tas Belanja", 100_000, nil))
	if num(lineOf(e, tok, id, "resepsi/seserahan")["estimate"]) != 4_578_500 {
		t.Error("an item without a link must stay out of the line")
	}
	joined := addItem(e, tok, id, itemBody("Kebutuhan Lain", "Rak", 50_000, map[string]any{"budget_item_id": line["id"]}))
	if joined["budget_item_id"] != line["id"] || num(lineOf(e, tok, id, "resepsi/seserahan")["estimate"]) != 4_628_500 {
		t.Errorf("joined = %v, line %v", joined, lineOf(e, tok, id, "resepsi/seserahan")["estimate"])
	}
	// PUT does not touch the link, the line follows the price
	put := mustStatus(t, e.do(tok, "PUT", base(id)+"/shopping-items/"+joined["id"].(string), itemBody("Kebutuhan Lain", "Rak", 70_000, nil)), 200).data()
	if put["budget_item_id"] != line["id"] || num(lineOf(e, tok, id, "resepsi/seserahan")["estimate"]) != 4_648_500 {
		t.Errorf("PUT: %v", put)
	}
	// a plain "link everything" picks up the loose item
	if n := num(mustStatus(t, linkToBudget(e, tok, id, nil), 200).data()["linked"]); n != 1 {
		t.Errorf("linked %d, want the one loose item", n)
	}
	if got := list(shoppingOverview(e, tok, id)["items"]); got[len(got)-2]["id"] != loose["id"] || got[len(got)-2]["budget_item_id"] != line["id"] {
		t.Errorf("the loose item did not join: %v", got[len(got)-2])
	}

	// finishing items moves the paid total of the line; the status of the line is computed by the view
	first := list(shoppingOverview(e, tok, id)["items"])[0]
	mustStatus(t, e.do(tok, "PATCH", base(id)+"/shopping-items/"+first["id"].(string)+"/status", map[string]any{"status": "selesai"}), 200)
	if l := lineOf(e, tok, id, "resepsi/seserahan"); num(l["paid"]) != 194_000 || l["payment_status"] != "partial" {
		t.Errorf("paid after one finished item: %v", l)
	}
	// deleting an item recomputes the line
	mustStatus(t, e.do(tok, "DELETE", base(id)+"/shopping-items/"+joined["id"].(string), nil), 204)
	if got := num(lineOf(e, tok, id, "resepsi/seserahan")["estimate"]); got != 4_678_500 { // 4.578.500 + the loose item of 100.000
		t.Errorf("after the delete: %d", got)
	}
}

// Unlinking releases every item and switches the rollup off; the line goes back to the figures stored in it.
func TestUnlinkReleasesEveryItem(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := weddingWithShopping(e, tok)
	line := lineOf(e, tok, id, "resepsi/seserahan")
	exec(e, `UPDATE planner_budget_items SET initial_estimate = 3500000 WHERE id = $1`, line["id"])

	mustStatus(t, linkToBudget(e, tok, id, nil), 200)
	if num(lineOf(e, tok, id, "resepsi/seserahan")["estimate"]) != 4_578_500 {
		t.Fatal("rollup did not apply")
	}
	r := mustStatus(t, unlinkFromBudget(e, tok, id, line["id"].(string)), 200).data()
	if num(r["unlinked"]) != 37 {
		t.Errorf("unlinked = %v", r)
	}
	if n := count(e, `SELECT count(*) FROM planner_shopping_items WHERE budget_item_id IS NOT NULL`); n != 0 {
		t.Errorf("%d items still point at the line", n)
	}
	l := lineOf(e, tok, id, "resepsi/seserahan")
	if l["rollup_source"] != nil || num(l["estimate"]) != 3_500_000 || len(list(shoppingOverview(e, tok, id)["rollups"])) != 0 {
		t.Errorf("line after the unlink = %v", l)
	}
	// and again: nothing left to release, no error
	if n := num(mustStatus(t, unlinkFromBudget(e, tok, id, line["id"].(string)), 200).data()["unlinked"]); n != 0 {
		t.Errorf("second unlink released %d", n)
	}
	// "every item that is not linked" is unambiguous again
	if n := num(mustStatus(t, linkToBudget(e, tok, id, nil), 200).data()["linked"]); n != 37 {
		t.Errorf("relink: %d", n)
	}
}

// Menu 03 owns the payments of a budget line, but a line that rolls up the shopping list takes its figures from the list:
// the lock is menu 03's (409 rollup_locked), the switch is this menu's, so the two have to agree through the real endpoints.
func TestMenu03LocksThePaymentsOfARolledUpLine(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := weddingWithShopping(e, tok)
	line := lineOf(e, tok, id, "resepsi/seserahan")
	payment := base(id) + "/budget/items/" + line["id"].(string) + "/payment"
	body := map[string]any{"payment_stage": "dp", "amount": 100_000, "paid_at": inDays(0)}

	mustStatus(t, linkToBudget(e, tok, id, nil), 200)
	wantError(t, e.do(tok, "PATCH", payment, body), 409, "rollup_locked", "")
	wantError(t, e.do(tok, "DELETE", payment+"/dp", nil), 409, "rollup_locked", "")

	// letting go of the rollup gives the payments back to menu 03
	mustStatus(t, unlinkFromBudget(e, tok, id, line["id"].(string)), 200)
	mustStatus(t, e.do(tok, "PATCH", payment, body), 200)
	if got := lineOf(e, tok, id, "resepsi/seserahan"); num(got["paid"]) != 100_000 {
		t.Errorf("line after the payment = %v", got)
	}
}

// The sample project has payments on the line (Rp 3.500.000 settled): switching the rollup on ignores them, so it needs confirm.
func TestLinkToBudgetNeedsConfirmWhenTheLineHoldsPayments(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)

	r := linkToBudget(e, tok, id, map[string]any{})
	wantError(t, r, 409, "rollup_conflict", "")
	if n := count(e, `SELECT count(*) FROM planner_shopping_items WHERE budget_item_id IS NOT NULL`); n != 0 {
		t.Errorf("a refused call linked %d items", n)
	}
	if l := lineOf(e, tok, id, "resepsi/seserahan"); l["rollup_source"] != nil || num(l["estimate"]) != 3_500_000 {
		t.Errorf("a refused call changed the line: %v", l)
	}

	res := mustStatus(t, linkToBudget(e, tok, id, map[string]any{"confirm": true}), 200).data()
	l := res["line"].(map[string]any)
	if num(res["linked"]) != 37 || num(l["estimate"]) != 4_578_500 || num(l["paid"]) != 2_063_800 {
		t.Fatalf("result = %v", res)
	}
	// the figures of the PDF, now computed from the list: estimate 4.578.500, paid 2.063.800, left 2.514.700, partial
	line := lineOf(e, tok, id, "resepsi/seserahan")
	if num(line["remaining"]) != 2_514_700 || line["payment_status"] != "partial" {
		t.Errorf("line = %v", line)
	}
	m := get(e, tok, base(id)+"/budget/matrix")["kpi"].(map[string]any)
	if num(m["total_estimate"]) != 75_228_500 || num(m["total_paid"]) != 32_113_800 {
		t.Errorf("matrix = %v", m)
	}
	// once on, it is not asked again
	mustStatus(t, linkToBudget(e, tok, id, map[string]any{}), 200)
}

func TestLinkToBudgetPartialItemsAndTheirRules(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := weddingWithShopping(e, tok)
	items := list(shoppingOverview(e, tok, id)["items"])
	a, b := items[0]["id"].(string), items[1]["id"].(string) // Mukena 194.000 and Al-Quran 70.500

	res := mustStatus(t, linkToBudget(e, tok, id, map[string]any{"item_ids": []string{a, b, a}}), 200).data()
	if num(res["linked"]) != 2 || num(res["line"].(map[string]any)["estimate"]) != 264_500 || num(res["line"].(map[string]any)["linked_items"]) != 2 {
		t.Errorf("partial = %v", res)
	}

	// items that belong to another line stay there when "everything" is linked to the default line
	manual := mustStatus(t, e.do(tok, "POST", base(id)+"/budget/items", map[string]any{"category": "Lain", "item_name": "Pos belanja lain", "sub_event_scope": "all"}), 201).data()
	c := items[2]["id"].(string)
	mustStatus(t, linkToBudget(e, tok, id, map[string]any{"budget_item_id": manual["id"], "item_ids": []string{c}}), 200)
	res = mustStatus(t, linkToBudget(e, tok, id, nil), 200).data()
	if num(res["linked"]) != 34 || count(e, `SELECT count(*) FROM planner_shopping_items WHERE budget_item_id = $1`, manual["id"]) != 1 {
		t.Errorf("everything: %v", res)
	}

	// the rules
	wantError(t, linkToBudget(e, tok, id, map[string]any{"item_ids": []string{uuid.NewString()}}), 400, "invalid_input", "item_ids: must be items of this project")
	other := createProject(e, tok, map[string]any{"title": "Lain"})
	foreign := addItem(e, tok, other, itemBody("a", "b", 1, nil))["id"]
	wantError(t, linkToBudget(e, tok, id, map[string]any{"item_ids": []any{foreign}}), 400, "invalid_input", "item_ids: must be items of this project")
	wantError(t, linkToBudget(e, tok, id, map[string]any{"budget_item_id": uuid.NewString()}), 400, "invalid_input", "budget_item_id: must be a budget line of this project")
	foreignLine := list(get(e, tok, base(other)+"/budget/items")["items"])
	if len(foreignLine) == 0 {
		exec(e, `INSERT INTO planner_budget_items (id, project_id, sub_event_scope, category, item_name, source, sort_order) VALUES (gen_random_uuid(), $1, 'all', 'x', 'Pos lain', 'manual', 1)`, other)
		foreignLine = list(get(e, tok, base(other)+"/budget/items")["items"])
	}
	wantError(t, linkToBudget(e, tok, id, map[string]any{"budget_item_id": foreignLine[0]["id"]}), 400, "invalid_input", "budget_item_id: must be a budget line of this project")
	wantError(t, e.do(tok, "POST", base(id)+"/shopping-items/link-to-budget", `{"budget_item_id": "nope"}`), 400, "invalid_json", "")

	// a unit-priced line, a line that a vendor holds and a line that a document holds cannot roll up
	exec(e, `UPDATE planner_budget_items SET pricing_mode='unit', quantity=2, unit_price=500, initial_estimate=1000 WHERE id=$1`, manual["id"])
	wantError(t, linkToBudget(e, tok, id, map[string]any{"budget_item_id": manual["id"]}), 400, "invalid_input", "budget_item_id: must not be a unit-priced line")
	v := addVendor(e, tok, id, vendorBody("catering", "Katering", nil))
	mustStatus(t, lockDeal(e, tok, id, v["id"].(string), 1_000_000, true), 200)
	cateringLine := lineOf(e, tok, id, "resepsi/catering")["id"]
	wantError(t, linkToBudget(e, tok, id, map[string]any{"budget_item_id": cateringLine}), 400, "invalid_input", "budget_item_id: must not be linked to a vendor or document")
	doc := mustStatus(t, e.do(tok, "POST", base(id)+"/admin-docs", map[string]any{"document_name": "Izin", "nominal_cost": 100_000}), 201).data()["id"]
	mustStatus(t, e.do(tok, "POST", base(id)+"/admin-docs/"+doc.(string)+"/sync-budget", nil), 201)
	docLine := lineOf(e, tok, id, "all/Izin")["id"]
	wantError(t, linkToBudget(e, tok, id, map[string]any{"budget_item_id": docLine}), 400, "invalid_input", "budget_item_id: must not be linked to a vendor or document")

	// unlink: the line must be of this project
	wantError(t, unlinkFromBudget(e, tok, id, uuid.NewString()), 400, "invalid_input", "budget_item_id: must be a budget line of this project")
	wantError(t, unlinkFromBudget(e, tok, id, foreignLine[0]["id"].(string)), 400, "invalid_input", "budget_item_id: must be a budget line of this project")
	wantError(t, e.do(tok, "POST", base(id)+"/shopping-items/unlink-from-budget", `{}`), 400, "invalid_input", "budget_item_id: must be a budget line of this project")

	// creating an item "into" a line that is not a rollup line is refused
	wantError(t, e.do(tok, "POST", base(id)+"/shopping-items", itemBody("a", "b", 1, map[string]any{"budget_item_id": cateringLine})), 400, "invalid_input", "budget_item_id: must be a shopping rollup line of this project")
}

// Without a line of the registry key the default line is made: 'other' has no line that fits, so `perlengkapan` is created.
func TestLinkToBudgetMakesTheDefaultLineWhenMissing(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	spec, _ := pd.DefaultRegistry().Get(pd.EventOther)
	id := createProject(e, tok, map[string]any{
		"event_type": "other", "title": "Acara", "target_budget": 10_000_000, "metadata": metadataFor(spec), "templates": map[string]any{"budget_lines": true},
	})
	before := budgetLines(e, tok, id)
	addItem(e, tok, id, itemBody("Perlengkapan", "Kursi", 15_000, map[string]any{"quantity": 100}))
	addItem(e, tok, id, itemBody("Cetak", "Banner", 250_000, nil))

	dt := shoppingOverview(e, tok, id)["default_target"].(map[string]any)
	if dt["budget_item_id"] != nil || dt["item_key"] != "perlengkapan" || dt["item_name"] != "Perlengkapan" || dt["sub_event_scope"] != "all" || dt["rollup"] != false || num(dt["estimate"]) != 0 {
		t.Fatalf("default_target = %v", dt)
	}
	r := mustStatus(t, linkToBudget(e, tok, id, nil), 200).data()
	l := r["line"].(map[string]any)
	if r["created"] != true || num(r["linked"]) != 2 || num(l["estimate"]) != 1_750_000 || l["item_name"] != "Perlengkapan" || l["sub_event_scope"] != "all" {
		t.Fatalf("result = %v", r)
	}
	after := budgetLines(e, tok, id)
	made := after["all/perlengkapan"]
	if len(after) != len(before)+1 || made["source"] != "manual" || made["category"] != "Perlengkapan" || made["rollup_source"] != "shopping" || made["id"] != l["budget_item_id"] {
		t.Errorf("made = %v (%d lines, was %d)", made, len(after), len(before))
	}
	// the second call finds it
	if r := mustStatus(t, linkToBudget(e, tok, id, nil), 200).data(); r["created"] != false || len(budgetLines(e, tok, id)) != len(before)+1 {
		t.Errorf("second call = %v", r)
	}
}

// Every non-wedding type has a default line in its percentage template (TEMPLATE_EVENT 15.4): the rollup replaces the
// percentage estimate of that line with the sum of the list, and does not make a line beyond the budget.
func TestLinkToBudgetUsesALineOfThePercentTemplate(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	defer func() { e.t = t }()

	for _, spec := range pd.DefaultRegistry().All() {
		if spec.Key == pd.EventWedding || spec.Key == pd.EventOther {
			continue
		}
		t.Run(string(spec.Key), func(t *testing.T) {
			e.t = t
			id := createProject(e, tok, map[string]any{
				"event_type": string(spec.Key), "title": "Uji " + spec.Label, "target_budget": 50_000_000, "metadata": metadataFor(spec),
				"event_date": inDays(90), "templates": map[string]any{"budget_lines": true},
			})
			before := budgetLines(e, tok, id)
			line := before["all/"+spec.ShoppingRollup.Key]
			if line == nil || line["item_name"] != spec.ShoppingRollup.Name || line["category"] != spec.ShoppingRollup.Category {
				t.Fatalf("no default line %q in %v", spec.ShoppingRollup.Key, before)
			}
			addItem(e, tok, id, itemBody(spec.ShoppingCategories[0], "Barang uji", 125_000, map[string]any{"quantity": 4}))

			dt := shoppingOverview(e, tok, id)["default_target"].(map[string]any)
			if dt["budget_item_id"] != line["id"] || dt["has_payments"] != false || num(dt["estimate"]) != num(line["estimate"]) {
				t.Errorf("default_target = %v", dt)
			}
			r := mustStatus(t, linkToBudget(e, tok, id, nil), 200).data()
			after := budgetLines(e, tok, id)
			if r["created"] != false || len(after) != len(before) || num(after["all/"+spec.ShoppingRollup.Key]["estimate"]) != 500_000 {
				t.Errorf("result %v; %d lines (was %d); estimate %v", r, len(after), len(before), after["all/"+spec.ShoppingRollup.Key]["estimate"])
			}
		})
	}
}

// peresmian: the default rollup line (prosesi_vip) is also the dedicated line of the MC. Whoever comes first holds it, and the
// other one is not left with an error: the vendor makes a line of its own, the list goes to the fallback line.
func TestTheDefaultLineIsNotFoughtOverByAVendorAndTheList(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	spec, _ := pd.DefaultRegistry().Get(pd.EventPeresmian)
	newProject := func(title string) string {
		return createProject(e, tok, map[string]any{
			"event_type": "peresmian", "title": title, "target_budget": 40_000_000, "metadata": metadataFor(spec), "event_date": inDays(90),
			"templates": map[string]any{"budget_lines": true},
		})
	}

	// the list first: the line rolls up, the MC then makes a line of its own and the rollup keeps its figures
	first := newProject("Daftar dulu")
	addItem(e, tok, first, itemBody("Souvenir", "Plakat", 200_000, map[string]any{"quantity": 5}))
	mustStatus(t, linkToBudget(e, tok, first, nil), 200)
	mc := addVendor(e, tok, first, map[string]any{"sub_event_scope": "all", "category_key": "mc", "vendor_name": "MC Handal"})
	r := mustStatus(t, lockDeal(e, tok, first, mc["id"].(string), 3_000_000, true), 200)
	lines := budgetLines(e, tok, first)
	if r.data()["created"] != true || lines["all/prosesi_vip"]["vendor_id"] != nil || num(lines["all/prosesi_vip"]["estimate"]) != 1_000_000 || lines["all/prosesi_vip"]["rollup_source"] != "shopping" {
		t.Errorf("rollup line: %v / %v", r.data(), lines["all/prosesi_vip"])
	}
	if l := lines["all/mc"]; l == nil || l["vendor_id"] != mc["id"] || num(l["estimate"]) != 3_000_000 {
		t.Errorf("the MC line: %v", l)
	}

	// the vendor first: the list goes to the fallback line, made once; the line of the MC is not touched
	second := newProject("Vendor dulu")
	mc2 := addVendor(e, tok, second, map[string]any{"sub_event_scope": "all", "category_key": "mc", "vendor_name": "MC Handal"})
	mustStatus(t, lockDeal(e, tok, second, mc2["id"].(string), 3_000_000, true), 200)
	addItem(e, tok, second, itemBody("Souvenir", "Plakat", 200_000, map[string]any{"quantity": 5}))
	dt := shoppingOverview(e, tok, second)["default_target"].(map[string]any)
	if dt["item_key"] != "perlengkapan" || dt["budget_item_id"] != nil || dt["item_name"] != "Perlengkapan" {
		t.Fatalf("default_target = %v", dt)
	}
	before := len(budgetLines(e, tok, second))
	res := mustStatus(t, linkToBudget(e, tok, second, nil), 200).data()
	lines = budgetLines(e, tok, second)
	if res["created"] != true || len(lines) != before+1 || lines["all/perlengkapan"]["rollup_source"] != "shopping" || num(lines["all/perlengkapan"]["estimate"]) != 1_000_000 ||
		lines["all/prosesi_vip"]["vendor_id"] != mc2["id"] || num(lines["all/prosesi_vip"]["estimate"]) != 3_000_000 {
		t.Errorf("result %v; %d lines", res, len(lines))
	}
	// and the second call finds the fallback line instead of making another one
	if again := mustStatus(t, linkToBudget(e, tok, second, nil), 200).data(); again["created"] != false || len(budgetLines(e, tok, second)) != before+1 {
		t.Errorf("again = %v", again)
	}
	// an explicit line that a vendor holds is still refused: only the default is swapped
	wantError(t, linkToBudget(e, tok, second, map[string]any{"budget_item_id": lines["all/prosesi_vip"]["id"]}), 400, "invalid_input", "budget_item_id: must not be linked to a vendor or document")
}
