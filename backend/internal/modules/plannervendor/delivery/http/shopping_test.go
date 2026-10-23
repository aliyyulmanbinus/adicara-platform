package http_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestShoppingLifecycle(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)

	// defaults: quantity 1, status belum, nothing linked, the totals worked out by the database
	a := addItem(e, tok, id, itemBody("Souvenir", "Gantungan Kunci", 5_000, map[string]any{"quantity": 300, "unit": "pcs", "purpose": "souvenir", "notes": "a\nb"}))
	b := addItem(e, tok, id, itemBody("Dekorasi", "Balon", 12_500, nil))
	if num(a["quantity"]) != 300 || a["unit"] != "pcs" || num(a["estimated_total"]) != 1_500_000 || a["status"] != "belum" || num(a["actual_price"]) != 0 || a["price_delta"] != nil ||
		a["purpose"] != "souvenir" || a["budget_item_id"] != nil || a["template_key"] != nil || num(a["sort_order"]) != 1 || a["notes"] != "a\nb" || a["vendor_id"] != nil {
		t.Fatalf("a = %v", a)
	}
	if num(b["quantity"]) != 1 || num(b["estimated_total"]) != 12_500 || num(b["sort_order"]) != 2 || b["unit"] != nil || b["purpose"] != nil {
		t.Fatalf("b = %v", b)
	}

	// a vendor of the project can be named; the list shows its name
	vendor := addVendor(e, tok, id, vendorBody("mua", "Sanggar Ayu", nil))
	r := e.do(tok, "PUT", base(id)+"/shopping-items/"+b["id"].(string), map[string]any{
		"sub_event_scope": "resepsi", "category": "Dekorasi", "item_name": "Balon Metalik", "quantity": 1.5, "estimated_price": 20_000, "actual_price": 18_000,
		"purchase_location": "https://shopee.co.id/x\nToko Balon, Jl. Mawar 3", "vendor_id": vendor["id"], "status": "proses",
	})
	pb := mustStatus(t, r, 200).data()
	if pb["item_name"] != "Balon Metalik" || pb["quantity"] != 1.5 || num(pb["estimated_total"]) != 30_000 || num(pb["actual_total"]) != 27_000 || num(pb["price_delta"]) != 3_000 ||
		pb["vendor_name"] != "Sanggar Ayu" || pb["sub_event_scope"] != "resepsi" || pb["status"] != "proses" || num(pb["sort_order"]) != 2 ||
		pb["purchase_location"] != "https://shopee.co.id/x\nToko Balon, Jl. Mawar 3" {
		t.Errorf("after PUT = %v", pb)
	}
	// PUT replaces: what is not sent is cleared (the quantity falls back to 1)
	pa := mustStatus(t, e.do(tok, "PUT", base(id)+"/shopping-items/"+a["id"].(string), itemBody("Souvenir", "Gantungan Kunci", 5_000, nil)), 200).data()
	if num(pa["quantity"]) != 1 || pa["unit"] != nil || pa["purpose"] != nil || pa["notes"] != nil {
		t.Errorf("PUT must replace: %v", pa)
	}

	// the overview: the kpi is the whole project, the list is filtered
	ov := shoppingOverview(e, tok, id)
	if k := ov["kpi"].(map[string]any); num(k["total"]) != 2 || num(k["estimated_total"]) != 35_000 || num(k["not_started"]) != 1 || num(k["in_progress"]) != 1 {
		t.Errorf("kpi = %v", k)
	}
	for query, want := range map[string]int{"?scope=resepsi": 1, "?scope=all": 1, "?category=Dekorasi": 1, "?category=dekorasi": 0, "?status=belum": 1, "?status=selesai": 0, "?scope=all&status=proses": 0} {
		got := get(e, tok, base(id)+"/shopping-items"+query)
		if n := len(list(got["items"])); n != want || num(got["kpi"].(map[string]any)["total"]) != 2 {
			t.Errorf("%s: %d items, want %d", query, n, want)
		}
	}
	for _, bad := range []string{"?status=x", "?scope=A%20B"} {
		if r := e.do(tok, "GET", base(id)+"/shopping-items"+bad, nil); r.status != 400 || r.code() != "invalid_input" {
			t.Errorf("%s: %d %s", bad, r.status, r.body)
		}
	}

	mustStatus(t, e.do(tok, "DELETE", base(id)+"/shopping-items/"+a["id"].(string), nil), 204)
	wantError(t, e.do(tok, "DELETE", base(id)+"/shopping-items/"+a["id"].(string), nil), 404, "not_found", "shopping item not found")
	wantError(t, e.do(tok, "PUT", base(id)+"/shopping-items/"+uuid.NewString(), itemBody("x", "y", 1, nil)), 404, "not_found", "shopping item not found")
	wantError(t, e.do(tok, "PATCH", base(id)+"/shopping-items/"+uuid.NewString()+"/status", map[string]any{"status": "selesai"}), 404, "not_found", "shopping item not found")

	// deleting the vendor keeps the item
	mustStatus(t, e.do(tok, "DELETE", base(id)+"/vendors/"+vendor["id"].(string), nil), 204)
	if got := list(shoppingOverview(e, tok, id)["items"]); len(got) != 1 || got[0]["vendor_id"] != nil || got[0]["vendor_name"] != nil {
		t.Errorf("after the vendor was deleted: %v", got)
	}
}

func TestShoppingValidation(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	other := createProject(e, tok, map[string]any{"title": "Lain"})
	foreign := addVendor(e, tok, other, vendorBody("mua", "Vendor proyek lain", nil))["id"]
	post := func(over map[string]any) resp {
		return e.do(tok, "POST", base(id)+"/shopping-items", itemBody("Dekorasi", "Balon", 1_000, over))
	}
	long := func(n int) string { return strings.Repeat("a", n) }

	for _, c := range []struct {
		name string
		body resp
		code string
		msg  string
	}{
		{"unknown session", post(map[string]any{"sub_event_scope": "nope"}), "invalid_scope", "sub_event_scope: unknown session"},
		{"empty category", post(map[string]any{"category": " "}), "invalid_input", "category: must not be empty"},
		{"category 61", post(map[string]any{"category": long(61)}), "invalid_input", "category: must be at most 60 characters"},
		{"empty name", post(map[string]any{"item_name": ""}), "invalid_input", "item_name: must not be empty"},
		{"name 256", post(map[string]any{"item_name": long(256)}), "invalid_input", "item_name: must be at most 255 characters"},
		{"quantity zero", post(map[string]any{"quantity": 0}), "invalid_input", "quantity: must be between 0.01 and 1000000"},
		{"quantity three decimals", post(map[string]any{"quantity": 1.234}), "invalid_input", "quantity: must be between 0.01 and 1000000"},
		{"quantity too big", post(map[string]any{"quantity": 1_000_000.01}), "invalid_input", "quantity: must be between 0.01 and 1000000"},
		{"negative quantity", post(map[string]any{"quantity": -2}), "invalid_input", "quantity: must be between 0.01 and 1000000"},
		{"unit 31", post(map[string]any{"unit": long(31)}), "invalid_input", "unit: must be at most 30 characters"},
		{"negative price", post(map[string]any{"estimated_price": -1}), "invalid_input", "estimated_price: must be between 0 and 10000000000000"},
		{"actual above the cap", post(map[string]any{"actual_price": 10_000_000_000_001}), "invalid_input", "actual_price: must be between 0 and 10000000000000"},
		{"location 501", post(map[string]any{"purchase_location": long(501)}), "invalid_input", "purchase_location: must be at most 500 characters"},
		{"purpose", post(map[string]any{"purpose": "gift"}), "invalid_input", "purpose: must be one of souvenir"},
		{"status", post(map[string]any{"status": "done"}), "invalid_input", "status: must be one of belum, proses, selesai"},
		{"notes control", post(map[string]any{"notes": "a\x00"}), "invalid_input", "notes: must not contain control characters"},
		{"vendor of another project", post(map[string]any{"vendor_id": foreign}), "invalid_input", "vendor_id: must be a vendor of this project"},
		{"unknown vendor", post(map[string]any{"vendor_id": uuid.NewString()}), "invalid_input", "vendor_id: must be a vendor of this project"},
		{"link to nothing", post(map[string]any{"budget_item_id": uuid.NewString()}), "invalid_input", "budget_item_id: must be a shopping rollup line of this project"},
		{"bad vendor id", e.do(tok, "POST", base(id)+"/shopping-items", `{"category":"a","item_name":"b","vendor_id":"nope"}`), "invalid_json", ""},
		{"quantity as text", e.do(tok, "POST", base(id)+"/shopping-items", `{"category":"a","item_name":"b","quantity":"2"}`), "invalid_json", ""},
	} {
		if c.body.status != 400 || c.body.code() != c.code || (c.msg != "" && c.body.message() != c.msg) {
			t.Errorf("%s: got %d %s %q; body %s", c.name, c.body.status, c.body.code(), c.body.message(), c.body.body)
		}
	}
	if n := count(e, `SELECT count(*) FROM planner_shopping_items`); n != 0 {
		t.Errorf("%d items were stored by rejected requests", n)
	}

	// the boundaries are accepted
	mustStatus(t, post(map[string]any{"item_name": long(255), "category": long(60), "unit": long(30), "quantity": 0.01, "estimated_price": 100}), 201)
	mustStatus(t, post(map[string]any{"quantity": 1_000_000, "estimated_price": 10_000_000}), 201) // exactly 10^13
}

// quantity × price beyond 10^13 is a 400 with a field message, never the 500 that the generated column's
// "bigint out of range" would be.
func TestShoppingOverflowIsA400(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	const msg = "the total (quantity × price) must be at most 10000000000000"

	for field, body := range map[string]map[string]any{
		"estimated_price": itemBody("a", "b", 10_000_001, map[string]any{"quantity": 1_000_000}),
		"actual_price":    itemBody("a", "b", 1, map[string]any{"quantity": 1_000_000, "actual_price": 10_000_001}),
	} {
		wantError(t, e.do(tok, "POST", base(id)+"/shopping-items", body), 400, "invalid_input", field+": "+msg)
	}
	it := addItem(e, tok, id, itemBody("a", "b", 10_000_000, map[string]any{"quantity": 1_000_000}))
	if num(it["estimated_total"]) != 10_000_000_000_000 {
		t.Errorf("the boundary: %v", it)
	}
	put := itemBody("a", "b", 1_000_000_000_000, map[string]any{"quantity": 100})
	wantError(t, e.do(tok, "PUT", base(id)+"/shopping-items/"+it["id"].(string), put), 400, "invalid_input", "estimated_price: "+msg)
	wantError(t, e.do(tok, "PUT", base(id)+"/shopping-items/"+it["id"].(string), itemBody("a", "b", 100, map[string]any{"quantity": 0.5, "actual_price": 20_000_000_000_000})),
		400, "invalid_input", "actual_price: must be between 0 and 10000000000000")
	if got := shoppingOverview(e, tok, id)["items"].([]any)[0].(map[string]any); num(got["estimated_total"]) != 10_000_000_000_000 {
		t.Errorf("a refused PUT changed the item: %v", got)
	}

	// rounding is the database's: 0,5 × 1 = 1 and 0,01 × 49 = 0
	for _, c := range []struct {
		q     float64
		price int64
		want  int64
	}{{0.5, 1, 1}, {0.01, 49, 0}, {2.5, 3, 8}, {0.29, 100, 29}} {
		if got := addItem(e, tok, id, itemBody("a", "b", c.price, map[string]any{"quantity": c.q})); num(got["estimated_total"]) != c.want {
			t.Errorf("%v × %d = %v, want %d", c.q, c.price, got["estimated_total"], c.want)
		}
	}
}

// Finishing an item without an actual price copies the estimate into it; a price that is there stays, going back keeps it.
func TestShoppingStatusFillsTheActualPrice(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	it := addItem(e, tok, id, itemBody("a", "Mukena", 194_000, map[string]any{"quantity": 2}))
	path := base(id) + "/shopping-items/" + it["id"].(string)

	r := mustStatus(t, e.do(tok, "PATCH", path+"/status", map[string]any{"status": "proses"}), 200).data()
	if r["status"] != "proses" || num(r["actual_price"]) != 0 || r["price_delta"] != nil {
		t.Errorf("proses: %v", r)
	}
	r = mustStatus(t, e.do(tok, "PATCH", path+"/status", map[string]any{"status": "selesai"}), 200).data()
	if r["status"] != "selesai" || num(r["actual_price"]) != 194_000 || num(r["actual_total"]) != 388_000 || num(r["price_delta"]) != 0 {
		t.Errorf("selesai: %v", r)
	}
	// edit the actual price, go back and finish again: the price the user typed is not overwritten
	mustStatus(t, e.do(tok, "PUT", path, itemBody("a", "Mukena", 194_000, map[string]any{"quantity": 2, "actual_price": 150_000, "status": "selesai"})), 200)
	mustStatus(t, e.do(tok, "PATCH", path+"/status", map[string]any{"status": "belum"}), 200)
	r = mustStatus(t, e.do(tok, "PATCH", path+"/status", map[string]any{"status": "selesai"}), 200).data()
	if num(r["actual_price"]) != 150_000 || num(r["price_delta"]) != 88_000 {
		t.Errorf("a typed price must stay: %v", r)
	}
	wantError(t, e.do(tok, "PATCH", path+"/status", map[string]any{"status": "lunas"}), 400, "invalid_input", "status: must be one of belum, proses, selesai")
	wantError(t, e.do(tok, "PATCH", path+"/status", map[string]any{}), 400, "invalid_input", "status: must be one of belum, proses, selesai")
}

// Stock of souvenirs = floor(Σ quantity) of the finished souvenir items: 300 + 50,5 finished, 20 in progress → 350.
func TestSouvenirStock(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)

	if k := shoppingOverview(e, tok, id)["kpi"].(map[string]any); k["souvenir_stock"] != nil {
		t.Errorf("no item: %v", k)
	}
	addItem(e, tok, id, itemBody("a", "Kipas", 1_000, nil))
	if k := shoppingOverview(e, tok, id)["kpi"].(map[string]any); k["souvenir_stock"] != nil {
		t.Errorf("an item that is not a souvenir does not make a stock: %v", k)
	}
	one := addItem(e, tok, id, itemBody("Souvenir", "Gantungan", 1_000, map[string]any{"quantity": 300, "purpose": "souvenir"}))
	two := addItem(e, tok, id, itemBody("Souvenir", "Pulpen", 1_000, map[string]any{"quantity": 50.5, "purpose": "souvenir"}))
	addItem(e, tok, id, itemBody("Souvenir", "Tas", 1_000, map[string]any{"quantity": 20, "purpose": "souvenir", "status": "proses"}))
	if k := shoppingOverview(e, tok, id)["kpi"].(map[string]any); k["souvenir_stock"] == nil || num(k["souvenir_stock"]) != 0 {
		t.Errorf("souvenir items that are not finished: a stock of 0, got %v", k["souvenir_stock"])
	}
	for _, it := range []map[string]any{one, two} {
		mustStatus(t, e.do(tok, "PATCH", base(id)+"/shopping-items/"+it["id"].(string)+"/status", map[string]any{"status": "selesai"}), 200)
	}
	if k := shoppingOverview(e, tok, id)["kpi"].(map[string]any); num(k["souvenir_stock"]) != 350 {
		t.Errorf("stock = %v, want 350", k["souvenir_stock"])
	}
}

func TestShoppingLimitOf1000(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	exec(e, `INSERT INTO planner_shopping_items (id, project_id, sub_event_scope, category, item_name, sort_order)
	         SELECT gen_random_uuid(), $1, 'all', 'x', 'Barang ' || g, g FROM generate_series(1, 999) g`, id)

	mustStatus(t, e.do(tok, "POST", base(id)+"/shopping-items", itemBody("x", "ke-1000", 1, nil)), 201)
	wantError(t, e.do(tok, "POST", base(id)+"/shopping-items", itemBody("x", "ke-1001", 1, nil)), 409, "shopping_item_limit_reached", "")
	if n := count(e, `SELECT count(*) FROM planner_shopping_items`); n != 1000 {
		t.Errorf("%d items", n)
	}
	// a template that would pass the limit is refused whole: nothing is inserted
	wedding := createProject(e, tok, map[string]any{"title": "Dua"})
	exec(e, `INSERT INTO planner_shopping_items (id, project_id, sub_event_scope, category, item_name, sort_order)
	         SELECT gen_random_uuid(), $1, 'all', 'x', 'Barang ' || g, g FROM generate_series(1, 980) g`, wedding)
	wantError(t, e.do(tok, "POST", base(wedding)+"/shopping-items/apply-template", map[string]any{}), 409, "shopping_item_limit_reached", "")
	if n := count(e, `SELECT count(*) FROM planner_shopping_items WHERE project_id=$1`, wedding); n != 980 {
		t.Errorf("%d items after the refused template, want 980", n)
	}
}
