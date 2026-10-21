package http_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func itemPath(id, item string) string { return base(id) + "/budget/items/" + item }

func addVendor(e *env, id, scope, name string) string {
	e.t.Helper()
	vendor := uuid.NewString()
	exec(e, `INSERT INTO planner_vendors (id, project_id, sub_event_scope, category_key, category_label, vendor_name)
	         VALUES ($1, $2, $3, 'catering', 'Catering', $4)`, vendor, id, scope, name)

	return vendor
}

func pay(e *env, tok, id, item string, body map[string]any) resp {
	e.t.Helper()

	return e.do(tok, "PATCH", itemPath(id, item)+"/payment", body)
}

func TestCreateAndReadBudgetItems(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)

	wantError(t, e.do(tok, "POST", base(id)+"/budget/items", map[string]any{"category": "x"}), 400, "invalid_input", "item_name: must not be empty")
	wantError(t, e.do(tok, "POST", base(id)+"/budget/items", map[string]any{"item_name": "x"}), 400, "invalid_input", "category: must not be empty")
	wantError(t, e.do(tok, "POST", base(id)+"/budget/items", map[string]any{"category": "x", "item_name": "x", "initial_estimate": -1}), 400, "invalid_input", "initial_estimate: must be between 0 and 10000000000000")
	wantError(t, e.do(tok, "POST", base(id)+"/budget/items", map[string]any{"category": "x", "item_name": "x", "sub_event_scope": "nope"}), 400, "invalid_scope", "")
	wantError(t, e.do(tok, "POST", base(id)+"/budget/items", map[string]any{"category": "x", "item_name": "x", "pricing_mode": "weekly"}), 400, "invalid_input", "pricing_mode: must be one of lump, unit")
	wantError(t, e.do(tok, "POST", base(id)+"/budget/items", map[string]any{"category": "x", "item_name": "x", "vendor_id": uuid.NewString()}), 400, "invalid_input", "vendor_id: must be a vendor of this project")
	wantError(t, e.do(tok, "POST", base(id)+"/budget/items", map[string]any{"category": "x", "item_name": strings.Repeat("a", 151)}), 400, "invalid_input", "item_name: must be at most 150 characters")

	a := mustStatus(t, e.do(tok, "POST", base(id)+"/budget/items", map[string]any{
		"sub_event_scope": "resepsi", "category": " Hiburan ", "item_name": " Photobooth ", "initial_estimate": 1_500_000, "notes": "baris 1\r\nbaris 2",
	}), 201).data()
	if a["item_name"] != "Photobooth" || a["category"] != "Hiburan" || a["source"] != "manual" || a["pricing_mode"] != "lump" || num(a["estimate"]) != 1_500_000 ||
		a["payment_status"] != "unpaid" || num(a["remaining"]) != 1_500_000 || a["notes"] != "baris 1\nbaris 2" || a["item_key"] != nil || a["sort_order"] == nil {
		t.Errorf("created = %v", a)
	}
	for _, k := range []string{"item_key", "vendor_id", "vendor_name", "admin_doc_id", "rollup_source", "quantity", "unit_label", "unit_price", "dp_paid_at", "dp_receipt_url"} {
		if _, has := a[k]; has {
			t.Errorf("optional key %q must be omitted when empty", k)
		}
	}
	b := mustStatus(t, e.do(tok, "POST", base(id)+"/budget/items", map[string]any{"category": "x", "item_name": "Umum"}), 201).data()
	if b["sub_event_scope"] != "all" || num(b["sort_order"]) != num(a["sort_order"])+1 {
		t.Errorf("a blank scope is the general session and the line goes last: %v", b)
	}

	// Scope filter: none = every session, "all" = the general lines only, a key = that session.
	count := func(q string) int {
		return len(list(get(e, tok, base(id)+"/budget/items"+q)["items"]))
	}
	if count("") != 2 || count("?scope=all") != 1 || count("?scope=resepsi") != 1 || count("?scope=akad") != 0 {
		t.Errorf("scope filter: %d %d %d %d", count(""), count("?scope=all"), count("?scope=resepsi"), count("?scope=akad"))
	}
	wantError(t, e.do(tok, "GET", base(id)+"/budget/items?scope=Bad%20Scope", nil), 400, "invalid_input", "scope: must be a session scope_key")
}

func TestUnitPricing(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	post := func(extra map[string]any) resp {
		body := map[string]any{"category": "Catering", "item_name": "Katering", "pricing_mode": "unit"}
		for k, v := range extra {
			body[k] = v
		}

		return e.do(tok, "POST", base(id)+"/budget/items", body)
	}

	l := mustStatus(t, post(map[string]any{"quantity": 250.5, "unit_price": 85_000, "unit_label": "porsi", "initial_estimate": 1}), 201).data()
	// 250,5 × 85.000 = 21.292.500; the estimate the client sent is ignored.
	if num(l["estimate"]) != 21_292_500 || l["pricing_mode"] != "unit" || l["quantity"] != 250.5 || num(l["unit_price"]) != 85_000 || l["unit_label"] != "porsi" {
		t.Errorf("unit line = %v", l)
	}
	if r := mustStatus(t, post(map[string]any{"quantity": 0.01, "unit_price": 150}), 201).data(); num(r["estimate"]) != 2 {
		t.Errorf("0,01 × 150 = 1,5 rounds half up to 2: %v", r)
	}

	wantError(t, post(map[string]any{"unit_price": 1}), 400, "invalid_input", "quantity: required when pricing_mode is unit")
	wantError(t, post(map[string]any{"quantity": 1}), 400, "invalid_input", "unit_price: required when pricing_mode is unit")
	wantError(t, post(map[string]any{"quantity": 0, "unit_price": 1}), 400, "invalid_input", "quantity: must be between 0.01 and 1000000")
	wantError(t, post(map[string]any{"quantity": 1.234, "unit_price": 1}), 400, "invalid_input", "quantity: must be between 0.01 and 1000000")
	wantError(t, post(map[string]any{"quantity": 1_000_000.01, "unit_price": 1}), 400, "invalid_input", "quantity: must be between 0.01 and 1000000")
	wantError(t, post(map[string]any{"quantity": 1, "unit_price": -1}), 400, "invalid_input", "unit_price: must be between 0 and 10000000000000")
	wantError(t, post(map[string]any{"quantity": 1, "unit_price": 1, "unit_label": strings.Repeat("a", 31)}), 400, "invalid_input", "unit_label: must be at most 30 characters")
	// The product is checked before it is written: PostgreSQL would answer "bigint out of range" (a 500).
	wantError(t, post(map[string]any{"quantity": 999_999.99, "unit_price": 10_000_000_000_000}), 400, "invalid_input",
		"unit_price: the estimate (quantity × unit_price) must be at most 10000000000000")
	if r := post(map[string]any{"quantity": 1, "unit_price": 10_000_000_000_000}); r.status != 201 {
		t.Errorf("exactly 10^13 is allowed: %d %s", r.status, r.body)
	}

	// Switching a line back to lump forgets the quantity and the price.
	back := mustStatus(t, e.do(tok, "PUT", itemPath(id, l["id"].(string)), map[string]any{"category": "Catering", "item_name": "Katering", "initial_estimate": 5_000_000}), 200).data()
	if back["pricing_mode"] != "lump" || num(back["estimate"]) != 5_000_000 || back["quantity"] != nil || back["unit_price"] != nil || back["unit_label"] != nil {
		t.Errorf("back to lump = %v", back)
	}
}

func TestRecordPayment(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	line := mustStatus(t, e.do(tok, "POST", base(id)+"/budget/items", map[string]any{"sub_event_scope": "resepsi", "category": "Dokumentasi", "item_name": "Foto", "initial_estimate": 1_000_000}), 201).data()
	item := line["id"].(string)

	// DP.
	r := mustStatus(t, pay(e, tok, id, item, map[string]any{"payment_stage": "dp", "amount": 500_000, "paid_at": "2026-10-01", "receipt_url": "https://example.com/bukti"}), 200).data()
	if num(r["dp_amount"]) != 500_000 || r["dp_paid_at"] != "2026-10-01" || r["dp_receipt_url"] != "https://example.com/bukti" ||
		r["payment_status"] != "dp_paid" || num(r["paid"]) != 500_000 || num(r["remaining"]) != 500_000 {
		t.Errorf("after dp = %v", r)
	}
	// "full" pays the remainder into the first empty stage (term_1) whatever amount is sent.
	r = mustStatus(t, pay(e, tok, id, item, map[string]any{"payment_stage": "full", "amount": 1, "paid_at": "2026-10-02"}), 200).data()
	if num(r["term_1_amount"]) != 500_000 || r["term_1_paid_at"] != "2026-10-02" || r["payment_status"] != "settled" || num(r["remaining"]) != 0 || num(r["paid"]) != 1_000_000 {
		t.Errorf("after full = %v", r)
	}
	// Nothing left: a second "full" (double click) is refused and pays nothing.
	wantError(t, pay(e, tok, id, item, map[string]any{"payment_stage": "full", "paid_at": "2026-10-02"}), 400, "invalid_stage", "")
	if n := count(e, `SELECT term_2_amount FROM planner_budget_items WHERE id=$1`, item); n != 0 {
		t.Errorf("term_2 = %d after a refused full", n)
	}

	// A stage replaces everything in it: a correction, and the receipt is cleared when it is not sent again.
	r = mustStatus(t, pay(e, tok, id, item, map[string]any{"payment_stage": "dp", "amount": 600_000, "paid_at": "2026-10-03"}), 200).data()
	if num(r["dp_amount"]) != 600_000 || r["dp_receipt_url"] != nil || num(r["remaining"]) != -100_000 || r["payment_status"] != "settled" {
		t.Errorf("after correcting dp (overpaid) = %v", r)
	}
	// Taking a payment back.
	r = mustStatus(t, e.do(tok, "DELETE", itemPath(id, item)+"/payment/term_1", nil), 200).data()
	if num(r["term_1_amount"]) != 0 || r["term_1_paid_at"] != nil || num(r["remaining"]) != 400_000 || r["payment_status"] != "dp_paid" {
		t.Errorf("after clearing term_1 = %v", r)
	}
	mustStatus(t, e.do(tok, "DELETE", itemPath(id, item)+"/payment/dp", nil), 200)
	if first := list(get(e, tok, base(id)+"/budget/items")["items"])[0]; first["payment_status"] != "unpaid" {
		t.Errorf("all stages cleared: %v", first)
	}

	// Validation.
	wantError(t, pay(e, tok, id, item, map[string]any{"payment_stage": "cicilan", "amount": 1, "paid_at": today}), 400, "invalid_input", "payment_stage: must be one of dp, term_1, term_2, full")
	wantError(t, pay(e, tok, id, item, map[string]any{"payment_stage": "dp", "paid_at": today}), 400, "invalid_input", "amount: required")
	wantError(t, pay(e, tok, id, item, map[string]any{"payment_stage": "dp", "amount": 0, "paid_at": today}), 400, "invalid_input", "amount: must be greater than 0")
	wantError(t, pay(e, tok, id, item, map[string]any{"payment_stage": "dp", "amount": 1}), 400, "invalid_date", "paid_at: required")
	wantError(t, pay(e, tok, id, item, map[string]any{"payment_stage": "dp", "amount": 1, "paid_at": "2010-01-01"}), 400, "invalid_date", "paid_at: must be within the project schedule window")
	for _, bad := range []string{"ftp://example.com/x", "javascript:alert(1)", "bukti.pdf", "https://" + strings.Repeat("a", 500)} {
		wantError(t, pay(e, tok, id, item, map[string]any{"payment_stage": "dp", "amount": 1, "paid_at": today, "receipt_url": bad}), 400, "invalid_input", "receipt_url: must be an http or https URL")
	}
	wantError(t, e.do(tok, "DELETE", itemPath(id, item)+"/payment/full", nil), 400, "invalid_input", "stage: must be one of dp, term_1, term_2")
	wantError(t, pay(e, tok, id, uuid.NewString(), map[string]any{"payment_stage": "dp", "amount": 1, "paid_at": today}), 404, "not_found", "budget item not found")
	wantError(t, pay(e, tok, id, "nope", map[string]any{"payment_stage": "dp", "amount": 1, "paid_at": today}), 404, "not_found", "budget item not found")

	// An estimate of 0 is "not budgeted yet": a payment makes it settled only when there is an estimate.
	zero := mustStatus(t, e.do(tok, "POST", base(id)+"/budget/items", map[string]any{"category": "x", "item_name": "Nol"}), 201).data()["id"].(string)
	if z := mustStatus(t, pay(e, tok, id, zero, map[string]any{"payment_stage": "dp", "amount": 100, "paid_at": today}), 200).data(); z["payment_status"] != "dp_paid" || num(z["remaining"]) != -100 {
		t.Errorf("paid line without an estimate = %v", z)
	}
}

func TestPaymentsOfAllThreeStagesAndTheFullStageOrder(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	item := mustStatus(t, e.do(tok, "POST", base(id)+"/budget/items", map[string]any{"category": "x", "item_name": "Catering", "initial_estimate": 10_000_000}), 201).data()["id"].(string)

	// term_2 first, then "full" fills dp (the first EMPTY stage in the order dp, term_1, term_2), then term_1.
	mustStatus(t, pay(e, tok, id, item, map[string]any{"payment_stage": "term_2", "amount": 1_000_000, "paid_at": today}), 200)
	r := mustStatus(t, pay(e, tok, id, item, map[string]any{"payment_stage": "full", "paid_at": today}), 200).data()
	if num(r["dp_amount"]) != 9_000_000 || num(r["term_1_amount"]) != 0 || num(r["term_2_amount"]) != 1_000_000 || r["payment_status"] != "settled" {
		t.Errorf("full into the first empty stage = %v", r)
	}
	// Clearing dp leaves a gap that "full" fills again.
	mustStatus(t, e.do(tok, "DELETE", itemPath(id, item)+"/payment/dp", nil), 200)
	mustStatus(t, pay(e, tok, id, item, map[string]any{"payment_stage": "term_1", "amount": 2_000_000, "paid_at": today}), 200)
	r = mustStatus(t, pay(e, tok, id, item, map[string]any{"payment_stage": "full", "paid_at": today}), 200).data()
	if num(r["dp_amount"]) != 7_000_000 || num(r["paid"]) != 10_000_000 {
		t.Errorf("second full = %v", r)
	}
	// Three stages are filled but the line is overpaid? "full" has no stage left and nothing to pay either way.
	wantError(t, pay(e, tok, id, item, map[string]any{"payment_stage": "full", "paid_at": today}), 400, "invalid_stage", "")
	// Overpaying through a stage is allowed; "full" on an overpaid line has nothing to settle.
	mustStatus(t, pay(e, tok, id, item, map[string]any{"payment_stage": "term_2", "amount": 5_000_000, "paid_at": today}), 200)
	if get(e, tok, base(id)+"/budget/cashflow-summary")["overpaid"] == nil {
		t.Fatal("no overpaid in the cashflow")
	}
	cash := get(e, tok, base(id)+"/budget/cashflow-summary")
	if num(cash["overpaid"]) != 4_000_000 || num(cash["remaining_debt"]) != 0 {
		t.Errorf("cashflow = %v", cash)
	}
}

func TestRollupLinesCannotBePaid(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := templateProject(e, tok)
	sese := lines(e, tok, id)["resepsi/seserahan"]["id"].(string)
	// A payment made before the line started to roll up the shopping list is stale: the view ignores it.
	mustStatus(t, pay(e, tok, id, sese, map[string]any{"payment_stage": "dp", "amount": 200_000, "paid_at": today}), 200)
	exec(e, `UPDATE planner_budget_items SET rollup_source='shopping' WHERE id=$1`, sese)
	exec(e, `INSERT INTO planner_shopping_items (id, project_id, sub_event_scope, category, item_name, quantity, estimated_price, actual_price, status, budget_item_id)
	         VALUES (gen_random_uuid(), $1, 'resepsi', 'Hantaran', 'Tas', 1, 1000000, 900000, 'selesai', $2),
	                (gen_random_uuid(), $1, 'resepsi', 'Hantaran', 'Sepatu', 2, 500000, 0, 'belum', $2)`, id, sese)

	wantError(t, pay(e, tok, id, sese, map[string]any{"payment_stage": "dp", "amount": 1, "paid_at": today}), 409, "rollup_locked", "")
	wantError(t, pay(e, tok, id, sese, map[string]any{"payment_stage": "full", "paid_at": today}), 409, "rollup_locked", "")
	wantError(t, e.do(tok, "DELETE", itemPath(id, sese)+"/payment/dp", nil), 409, "rollup_locked", "")

	// The figures come from the shopping list: estimate 1.000.000 + 2 × 500.000, paid only for the finished item.
	l := lines(e, tok, id)["resepsi/seserahan"]
	if num(l["estimate"]) != 2_000_000 || num(l["paid"]) != 900_000 || num(l["remaining"]) != 1_100_000 || l["rollup_source"] != "shopping" || l["payment_status"] != "partial" {
		t.Errorf("rollup line = %v", l)
	}
	m := get(e, tok, base(id)+"/budget/matrix")
	for _, r := range list(m["rows"]) {
		if r["sub_event_scope"] == "resepsi" && (num(r["rollup_paid"]) != 900_000 || num(r["paid"]) != 900_000 || num(r["dp"]) != 0) {
			t.Errorf("matrix resepsi = %v", r)
		}
		// paid = dp + term_1 + term_2 + rollup_paid must hold in every row.
		if num(r["paid"]) != num(r["dp"])+num(r["term_1"])+num(r["term_2"])+num(r["rollup_paid"]) {
			t.Errorf("the paid identity breaks in %v", r)
		}
	}

	// A PUT cannot change what the shopping list owns, but renames and notes go through.
	up := mustStatus(t, e.do(tok, "PUT", itemPath(id, sese), map[string]any{
		"sub_event_scope": "resepsi", "category": "Hantaran", "item_name": "Seserahan Baru", "initial_estimate": 42, "pricing_mode": "unit", "quantity": 3, "unit_price": 7, "notes": "ok",
	}), 200).data()
	if up["item_name"] != "Seserahan Baru" || up["notes"] != "ok" || num(up["estimate"]) != 2_000_000 || up["pricing_mode"] != "lump" || up["rollup_source"] != "shopping" {
		t.Errorf("updated rollup line = %v", up)
	}

	// Deleting a rollup line is allowed; the shopping items stay and lose the link.
	mustStatus(t, e.do(tok, "DELETE", itemPath(id, sese), nil), 204)
	if n := count(e, `SELECT count(*) FROM planner_shopping_items WHERE project_id=$1 AND budget_item_id IS NULL`, id); n != 2 {
		t.Errorf("%d shopping items without a link, want 2 (both stay)", n)
	}
}

func TestUpdateBudgetItem(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := templateProject(e, tok)
	all := lines(e, tok, id)
	cat := all["resepsi/catering"]["id"].(string)
	vendorA, vendorB := addVendor(e, id, "resepsi", "Dapur Ibu"), addVendor(e, id, "resepsi", "Bunda Catering")

	body := func(extra map[string]any) map[string]any {
		m := map[string]any{"sub_event_scope": "resepsi", "category": "Catering", "item_name": "Catering", "initial_estimate": 20_000_000}
		for k, v := range extra {
			m[k] = v
		}

		return m
	}

	r := mustStatus(t, e.do(tok, "PUT", itemPath(id, cat), body(map[string]any{"vendor_id": vendorA, "notes": "DP minggu depan"})), 200).data()
	if r["vendor_id"] != vendorA || r["vendor_name"] != "Dapur Ibu" || num(r["estimate"]) != 20_000_000 || r["notes"] != "DP minggu depan" || r["item_key"] != "catering" || r["source"] != "template" {
		t.Errorf("after PUT = %v", r)
	}
	// PUT replaces: leaving vendor_id and notes out releases the vendor and clears the notes (03 BE 3.10).
	r = mustStatus(t, e.do(tok, "PUT", itemPath(id, cat), body(nil)), 200).data()
	if r["vendor_id"] != nil || r["notes"] != nil {
		t.Errorf("an omitted vendor_id must release the vendor: %v", r)
	}

	// A vendor can belong to one line only; a vendor of another project is refused.
	mustStatus(t, e.do(tok, "PUT", itemPath(id, cat), body(map[string]any{"vendor_id": vendorA})), 200)
	doc := all["resepsi/dokumentasi"]["id"].(string)
	wantError(t, e.do(tok, "PUT", itemPath(id, doc), body(map[string]any{"category": "Dokumentasi", "item_name": "Dokumentasi", "vendor_id": vendorA})), 409, "vendor_already_linked", "")
	mustStatus(t, e.do(tok, "PUT", itemPath(id, doc), body(map[string]any{"category": "Dokumentasi", "item_name": "Dokumentasi", "vendor_id": vendorB})), 200)
	other := createProject(e, tok, nil)
	foreign := addVendor(e, other, "all", "Asing")
	wantError(t, e.do(tok, "PUT", itemPath(id, doc), body(map[string]any{"vendor_id": foreign})), 400, "invalid_input", "vendor_id: must be a vendor of this project")

	// Payments are not touched by PUT.
	mustStatus(t, pay(e, tok, id, cat, map[string]any{"payment_stage": "dp", "amount": 5_000_000, "paid_at": today}), 200)
	r = mustStatus(t, e.do(tok, "PUT", itemPath(id, cat), body(map[string]any{"vendor_id": vendorA, "initial_estimate": 25_000_000})), 200).data()
	if num(r["dp_amount"]) != 5_000_000 || num(r["estimate"]) != 25_000_000 {
		t.Errorf("payment after PUT = %v", r)
	}

	wantError(t, e.do(tok, "PUT", itemPath(id, uuid.NewString()), body(nil)), 404, "not_found", "budget item not found")
	wantError(t, e.do(tok, "PUT", itemPath(id, cat), body(map[string]any{"item_name": ""})), 400, "invalid_input", "item_name: must not be empty")
}

func TestMovingALineNeutralisesTheKeyWhenTheSessionHasIt(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := templateProject(e, tok)
	all := lines(e, tok, id)
	lam := all["lamaran/catering"]["id"].(string)

	// Both Lamaran and Resepsi have a "catering" line. Moving one to the other session must not fail.
	r := mustStatus(t, e.do(tok, "PUT", itemPath(id, lam), map[string]any{"sub_event_scope": "resepsi", "category": "Catering", "item_name": "Catering lamaran", "initial_estimate": 2_000_000}), 200).data()
	if r["sub_event_scope"] != "resepsi" || r["item_key"] != nil || r["item_name"] != "Catering lamaran" {
		t.Errorf("moved line = %v", r)
	}
	if got := lines(e, tok, id); got["resepsi/catering"] == nil || got["resepsi/catering"]["item_key"] != "catering" {
		t.Errorf("the resepsi line must keep its key: %v", got["resepsi/catering"])
	}
	// A line moves with its key when the target session does not have it.
	fotografer := all["lamaran/fotografer"]["id"].(string)
	r = mustStatus(t, e.do(tok, "PUT", itemPath(id, fotografer), map[string]any{"sub_event_scope": "akad", "category": "Dokumentasi", "item_name": "Fotografer"}), 200).data()
	if r["sub_event_scope"] != "akad" || r["item_key"] != "fotografer" {
		t.Errorf("moved line = %v", r)
	}
}

func TestDeleteAndReorderBudgetItems(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := templateProject(e, tok)
	all := lines(e, tok, id)

	// Deleting a line removes its payments; a document and a shopping item that pointed at it keep their data and lose the link.
	cincin := all["lamaran/cincin"]["id"].(string)
	mustStatus(t, pay(e, tok, id, cincin, map[string]any{"payment_stage": "dp", "amount": 600_000, "paid_at": today}), 200)
	mustStatus(t, e.do(tok, "DELETE", itemPath(id, cincin), nil), 204)
	wantError(t, e.do(tok, "DELETE", itemPath(id, cincin), nil), 404, "not_found", "budget item not found")
	if len(lines(e, tok, id)) != 13 {
		t.Fatal("the line is not gone")
	}

	// Reorder: the whole list, once each.
	var ids []string
	for _, l := range list(get(e, tok, base(id)+"/budget/items")["items"]) {
		ids = append(ids, l["id"].(string))
	}
	rev := make([]string, len(ids))
	for i, v := range ids {
		rev[len(ids)-1-i] = v
	}
	out := list(mustStatus(t, e.do(tok, "POST", base(id)+"/budget/items/reorder", map[string]any{"ids": rev}), 200).data()["items"])
	if len(out) != 13 || out[0]["id"] != rev[0] || out[12]["id"] != rev[12] || num(out[0]["sort_order"]) != 1 || num(out[12]["sort_order"]) != 13 {
		t.Errorf("reordered = %v", out[0])
	}
	wantError(t, e.do(tok, "POST", base(id)+"/budget/items/reorder", map[string]any{"ids": rev[:12]}), 400, "invalid_input", "ids: must list every id exactly once")
	dup := append([]string{rev[0]}, rev[:12]...)
	wantError(t, e.do(tok, "POST", base(id)+"/budget/items/reorder", map[string]any{"ids": dup}), 400, "invalid_input", "ids: must list every id exactly once")
	foreign := append([]string{uuid.NewString()}, rev[:12]...)
	wantError(t, e.do(tok, "POST", base(id)+"/budget/items/reorder", map[string]any{"ids": foreign}), 400, "invalid_input", "ids: must list every id exactly once")
}

func TestBudgetLineLimit(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	exec(e, `INSERT INTO planner_budget_items (id, project_id, sub_event_scope, category, item_name, source, sort_order)
	         SELECT gen_random_uuid(), $1, 'all', 'x', 'pos ' || n, 'manual', n FROM generate_series(1, 300) n`, id)

	body := map[string]any{"category": "x", "item_name": "ke-301"}
	wantError(t, e.do(tok, "POST", base(id)+"/budget/items", body), 409, "budget_item_limit_reached", "")
	for _, s := range concurrently(5, func(int) int { return e.do(tok, "POST", base(id)+"/budget/items", body).status }) {
		if s != 409 {
			t.Fatalf("a full project accepted another line")
		}
	}
	if n := count(e, `SELECT count(*) FROM planner_budget_items WHERE project_id=$1`, id); n != 300 {
		t.Errorf("%d lines", n)
	}
}

func TestTwoSimultaneousCreatesCannotPassTheLineLimit(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	exec(e, `INSERT INTO planner_budget_items (id, project_id, sub_event_scope, category, item_name, source, sort_order)
	         SELECT gen_random_uuid(), $1, 'all', 'x', 'pos ' || n, 'manual', n FROM generate_series(1, 297) n`, id)

	statuses := concurrently(8, func(i int) int {
		return e.do(tok, "POST", base(id)+"/budget/items", map[string]any{"category": "x", "item_name": "baru"}).status
	})
	created := 0
	for _, s := range statuses {
		if s == 201 {
			created++
		}
	}
	if n := count(e, `SELECT count(*) FROM planner_budget_items WHERE project_id=$1`, id); n != 300 || created != 3 {
		t.Errorf("%d lines, %d created (statuses %v), want 300 and 3", n, created, statuses)
	}
	// The order numbers handed out under the lock are distinct.
	if n := count(e, `SELECT count(DISTINCT sort_order) FROM planner_budget_items WHERE project_id=$1`, id); n != 300 {
		t.Errorf("%d distinct sort_order values among 300 lines", n)
	}
}

// ---------- the budget template ----------

func TestApplyTemplateAppend(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)

	r := mustStatus(t, e.do(tok, "POST", base(id)+"/budget/apply-template", map[string]any{"mode": "append"}), 200).data()
	if num(r["created"]) != 14 || num(r["skipped"]) != 0 || num(r["deleted"]) != 0 || len(list(r["warnings"])) != 0 {
		t.Errorf("first apply = %v", r)
	}
	all := lines(e, tok, id)
	if len(all) != 14 || num(all["resepsi/catering"]["estimate"]) != 0 || all["resepsi/catering"]["source"] != "template" || all["lamaran/dekorasi"]["category"] != "Dekorasi" {
		t.Errorf("template lines: %d", len(all))
	}
	// Idempotent; deleting two lines makes exactly those two come back.
	r = mustStatus(t, e.do(tok, "POST", base(id)+"/budget/apply-template", map[string]any{}), 200).data()
	if num(r["created"]) != 0 || num(r["skipped"]) != 14 {
		t.Errorf("second apply = %v", r)
	}
	mustStatus(t, e.do(tok, "DELETE", itemPath(id, all["akad/biaya_kua"]["id"].(string)), nil), 204)
	mustStatus(t, e.do(tok, "DELETE", itemPath(id, all["resepsi/hiburan"]["id"].(string)), nil), 204)
	r = mustStatus(t, e.do(tok, "POST", base(id)+"/budget/apply-template", map[string]any{"mode": "append"}), 200).data()
	if num(r["created"]) != 2 || num(r["skipped"]) != 12 {
		t.Errorf("third apply = %v", r)
	}

	// Only one session.
	id2 := createProject(e, tok, nil)
	r = mustStatus(t, e.do(tok, "POST", base(id2)+"/budget/apply-template", map[string]any{"mode": "append", "sub_event_scope": "lamaran"}), 200).data()
	if num(r["created"]) != 5 || len(lines(e, tok, id2)) != 5 {
		t.Errorf("lamaran only = %v", r)
	}
	wantError(t, e.do(tok, "POST", base(id2)+"/budget/apply-template", map[string]any{"sub_event_scope": "nope"}), 400, "invalid_scope", "")
	wantError(t, e.do(tok, "POST", base(id2)+"/budget/apply-template", map[string]any{"mode": "replace_all"}), 400, "invalid_input", "mode: must be one of append, replace_template")
}

func TestApplyTemplateReplace(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := templateProject(e, tok)
	all := lines(e, tok, id)

	// With no work in the template lines, replace needs no confirmation and recreates the same 14.
	r := mustStatus(t, e.do(tok, "POST", base(id)+"/budget/apply-template", map[string]any{"mode": "replace_template"}), 200).data()
	if num(r["deleted"]) != 14 || num(r["created"]) != 14 || num(r["skipped"]) != 0 || len(list(r["warnings"])) != 0 {
		t.Errorf("clean replace = %v", r)
	}

	// Work the user did: a scenario filled the estimates, one line has a payment, one a vendor.
	sc := createScenario(e, tok, id, map[string]any{"name": "Ideal", "preset_key": "wedding_75jt"})
	mustStatus(t, e.do(tok, "POST", base(id)+"/scenarios/apply", map[string]any{"scenario_id": sc["id"], "mode": "smart_adjust"}), 200)
	all = lines(e, tok, id)
	mustStatus(t, pay(e, tok, id, all["lamaran/fotografer"]["id"].(string), map[string]any{"payment_stage": "dp", "amount": 500_000, "paid_at": today}), 200)
	vendor := addVendor(e, id, "resepsi", "Dapur Ibu")
	mustStatus(t, e.do(tok, "PUT", itemPath(id, all["resepsi/catering"]["id"].(string)), map[string]any{
		"sub_event_scope": "resepsi", "category": "Catering", "item_name": "Catering", "initial_estimate": 30_000_000, "vendor_id": vendor,
	}), 200)

	wantError(t, e.do(tok, "POST", base(id)+"/budget/apply-template", map[string]any{"mode": "replace_template"}), 400, "invalid_input", "confirm: required")
	if len(lines(e, tok, id)) != 15 {
		t.Fatal("a refused replace deleted lines")
	}

	r = mustStatus(t, e.do(tok, "POST", base(id)+"/budget/apply-template", map[string]any{"mode": "replace_template", "confirm": true}), 200).data()
	// 13 template lines had no payment: deleted and recreated; the paid one stays and counts as skipped. The scenario's own line stays.
	if num(r["deleted"]) != 13 || num(r["created"]) != 13 || num(r["skipped"]) != 1 {
		t.Errorf("replace = %v", r)
	}
	warnings := list(r["warnings"])
	if len(warnings) != 1 || warnings[0]["code"] != "budget_link_lost" || num(warnings[0]["count"]) != 1 {
		t.Errorf("warnings = %v (one line with a vendor was deleted)", warnings)
	}
	after := lines(e, tok, id)
	if len(after) != 15 || num(after["resepsi/catering"]["estimate"]) != 0 || after["resepsi/catering"]["vendor_id"] != nil ||
		num(after["lamaran/fotografer"]["dp_amount"]) != 500_000 || after["all/dana_darurat"]["source"] != "scenario" {
		t.Errorf("lines after replace: %d", len(after))
	}
	if n := count(e, `SELECT count(*) FROM planner_vendors WHERE id=$1`, vendor); n != 1 {
		t.Error("deleting a line must not delete its vendor")
	}
}

func TestApplyTemplateForOtherEventTypes(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, map[string]any{
		"event_type": "seminar", "metadata": map[string]any{"organizer_name": "BEM"}, "target_budget": 50_000_000,
		"templates": map[string]any{"budget_lines": true},
	})
	all := lines(e, tok, id)
	if len(all) != 8 || num(all["all/venue"]["estimate"]) != 12_500_000 || num(all["all/dana_darurat"]["estimate"]) != 3_500_000 ||
		all["all/venue"]["category"] != "Venue & Dekorasi" || all["all/venue"]["source"] != "template" {
		t.Errorf("seminar template: %d lines", len(all))
	}
	var sum int64
	for _, l := range all {
		sum += num(l["estimate"])
	}
	if sum != 50_000_000 {
		t.Errorf("the template adds up to %d, want the budget 50.000.000", sum)
	}

	// No budget set (or a tiny one): the lines come with estimate 0 for the user to fill.
	zero := createProject(e, tok, map[string]any{"event_type": "aqiqah", "metadata": map[string]any{"baby_name": "Ayu"}, "target_budget": 0, "templates": map[string]any{"budget_lines": true}})
	tiny := createProject(e, tok, map[string]any{"event_type": "aqiqah", "metadata": map[string]any{"baby_name": "Ayu"}, "target_budget": 50_000, "templates": map[string]any{"budget_lines": true}})
	for _, p := range []string{zero, tiny} {
		l := lines(e, tok, p)
		if len(l) != 7 || num(l["all/kambing"]["estimate"]) != 0 {
			t.Errorf("project %s: %d lines, kambing %v", p, len(l), l["all/kambing"])
		}
	}
	// Once the budget is known the user can ask again: the lines hold no work yet, so replace needs no
	// confirmation and turns the zeros into figures (kambing is 45% of 40.000.000).
	exec(e, `UPDATE planner_projects SET target_budget=40000000 WHERE id=$1`, zero)
	r := mustStatus(t, e.do(tok, "POST", base(zero)+"/budget/apply-template", map[string]any{"mode": "replace_template"}), 200).data()
	if num(r["deleted"]) != 7 || num(r["created"]) != 7 || num(lines(e, tok, zero)["all/kambing"]["estimate"]) != 18_000_000 {
		t.Errorf("replace after the budget was set = %v", r)
	}
}

func TestBudgetTemplateScopeFallbackAndWizard(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")

	// Wizard: budget_lines creates the 14 lines in the same transaction.
	id := templateProject(e, tok)
	if got := get(e, tok, base(id)+"/budget/matrix"); num(got["kpi"].(map[string]any)["line_count"]) != 14 {
		t.Errorf("wizard lines: %v", got["kpi"])
	}
	if applied := mustStatus(t, e.do(tok, "POST", "/v1/planner/projects", project(map[string]any{"templates": map[string]any{"budget_lines": true}})), 201).data()["applied"].(map[string]any); num(applied["budget"].(map[string]any)["created"]) != 14 {
		t.Errorf("applied = %v", applied)
	}

	// A session the user deleted: its lines go to the general session with a warning.
	id2 := createProject(e, tok, nil)
	for _, s := range sessionsOf(e, tok, id2) {
		if s["scope_key"] == "lamaran" {
			mustStatus(t, e.do(tok, "DELETE", base(id2)+"/sessions/"+s["id"].(string), nil), 204)
		}
	}
	r := mustStatus(t, e.do(tok, "POST", base(id2)+"/budget/apply-template", map[string]any{}), 200).data()
	w := list(r["warnings"])
	if len(w) != 1 || w[0]["code"] != "scope_fallback" || num(w[0]["count"]) != 5 {
		t.Errorf("warnings = %v", w)
	}
	if got := lines(e, tok, id2); got["all/catering"] == nil || got["all/cincin"] == nil || got["resepsi/catering"] == nil || len(got) != 14 {
		t.Errorf("fallback lines: %d", len(got))
	}
}

func TestEventTypesOfferBudgetLinesAndContributorRoles(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	r := mustStatus(t, e.do(tok, "GET", "/v1/planner/event-types", nil), 200)
	types := list(r.json()["data"])
	if len(types) != 9 {
		t.Fatalf("%d event types", len(types))
	}
	for _, et := range types {
		if et["templates_available"].(map[string]any)["budget_lines"] != true {
			t.Errorf("%v: budget_lines must be offered now that the module is registered", et["key"])
		}
		roles := list(et["contributor_roles"])
		found := false
		for _, role := range roles {
			if role["key"] == "other" && role["label"] == "Lainnya" {
				found = true
			}
		}
		if !found || len(roles) < 3 {
			t.Errorf("%v: contributor_roles = %v", et["key"], roles)
		}
	}
	if w := list(types[0]["contributor_roles"]); len(w) != 5 || w[0]["key"] != "groom" || w[0]["label"] != "Mempelai Pria" {
		t.Errorf("wedding roles = %v", w)
	}
}

func TestCashflowSummaryOverpaid(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	post := func(estimate int64) string {
		return mustStatus(t, e.do(tok, "POST", base(id)+"/budget/items", map[string]any{"category": "x", "item_name": uuid.NewString()[:8], "initial_estimate": estimate}), 201).data()["id"].(string)
	}
	a := post(2_000_000)
	post(10_000_000)
	mustStatus(t, pay(e, tok, id, a, map[string]any{"payment_stage": "dp", "amount": 7_000_000, "paid_at": today}), 200)

	// Line A is overpaid by 5.000.000 and line B owes 10.000.000: the debt is 10.000.000, not 5.000.000.
	cash := get(e, tok, base(id)+"/budget/cashflow-summary")
	if num(cash["remaining_debt"]) != 10_000_000 || num(cash["overpaid"]) != 5_000_000 || num(cash["total_paid"]) != 7_000_000 ||
		num(cash["available_cash"]) != -7_000_000 || num(cash["cash_deficit"]) != 17_000_000 {
		t.Errorf("cashflow = %v", cash)
	}
	m := get(e, tok, base(id)+"/budget/matrix")
	tot := m["total"].(map[string]any)
	// estimate − paid = remaining − overpaid, always.
	if num(tot["estimate"])-num(tot["paid"]) != num(tot["remaining"])-num(tot["overpaid"]) || m["active_scenario"] != nil {
		t.Errorf("matrix total = %v, active %v", tot, m["active_scenario"])
	}
}

// A double click on "pay the rest" must pay once: the row is locked, so the second request sees nothing left.
func TestSimultaneousFullPaymentsPayOnce(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	item := mustStatus(t, e.do(tok, "POST", base(id)+"/budget/items", map[string]any{"category": "x", "item_name": "Catering", "initial_estimate": 1_000_000}), 201).data()["id"].(string)

	for round := 0; round < 3; round++ {
		exec(e, `UPDATE planner_budget_items SET dp_amount=0, term_1_amount=0, term_2_amount=0 WHERE id=$1`, item)
		statuses := concurrently(8, func(int) int {
			return pay(e, tok, id, item, map[string]any{"payment_stage": "full", "paid_at": today}).status
		})
		paid := 0
		for _, s := range statuses {
			switch s {
			case 200:
				paid++
			case 400:
			default:
				t.Fatalf("round %d: unexpected status %d in %v", round, s, statuses)
			}
		}
		l := lines(e, tok, id)["all/Catering"]
		if paid != 1 || num(l["paid"]) != 1_000_000 || num(l["remaining"]) != 0 {
			t.Fatalf("round %d: %d payments accepted (%v), paid %v, remaining %v; want exactly one and a settled line", round, paid, statuses, l["paid"], l["remaining"])
		}
	}
}

// Menu 02 copies the cost of a document into the budget. Now that the budget template exists the
// document finds its real line (akad / biaya_kua) instead of the one a test inserted by hand.
func TestAdminDocSyncLinksTheTemplateLineOfThisModule(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, map[string]any{"templates": map[string]any{"admin_docs": true, "budget_lines": true}})
	docs := list(get(e, tok, base(id)+"/admin-docs")["docs"])
	kua := docs[0]["id"].(string)

	r := mustStatus(t, e.do(tok, "POST", base(id)+"/admin-docs/"+kua+"/sync-budget", nil), 200).data()
	line := lines(e, tok, id)["akad/biaya_kua"]
	if r["linked"] != true || r["created"] != false || r["budget_item_id"] != line["id"] || len(lines(e, tok, id)) != 14 {
		t.Fatalf("sync = %v, %d lines", r, len(lines(e, tok, id)))
	}
	if num(line["estimate"]) != 600_000 || line["admin_doc_id"] != kua || line["source"] != "template" {
		t.Errorf("the linked line = %v", line)
	}
	if list(get(e, tok, base(id)+"/admin-docs")["docs"])[0]["budget_item_id"] != line["id"] {
		t.Error("the document must show its line")
	}

	// The link is work of the user: replacing the template needs confirmation and then cuts the link, not the document.
	wantError(t, e.do(tok, "POST", base(id)+"/budget/apply-template", map[string]any{"mode": "replace_template"}), 400, "invalid_input", "confirm: required")
	res := mustStatus(t, e.do(tok, "POST", base(id)+"/budget/apply-template", map[string]any{"mode": "replace_template", "confirm": true}), 200).data()
	w := list(res["warnings"])
	if num(res["deleted"]) != 14 || len(w) != 1 || w[0]["code"] != "budget_link_lost" || num(w[0]["count"]) != 1 {
		t.Errorf("replace = %v", res)
	}
	if after := list(get(e, tok, base(id)+"/admin-docs")["docs"])[0]; after["budget_item_id"] != nil || after["id"] != kua {
		t.Errorf("the document after the replace = %v", after)
	}
	if num(lines(e, tok, id)["akad/biaya_kua"]["estimate"]) != 0 {
		t.Error("the recreated line starts at 0 again")
	}
}
