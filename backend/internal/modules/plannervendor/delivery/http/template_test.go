package http_test

import (
	"testing"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannervendor/domain"
)

func applyTemplate(e *env, tok, id string, body any) resp {
	e.t.Helper()

	return e.do(tok, "POST", base(id)+"/shopping-items/apply-template", body)
}

// The wizard creates the 37 items of the wedding, once, inside the project transaction; the other types are not offered one.
func TestWizardCreatesTheShoppingTemplate(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")

	r := mustStatus(t, e.do(tok, "POST", "/v1/planner/projects", project(map[string]any{"templates": map[string]any{"budget_lines": true, "shopping_list": true}})), 201)
	applied := r.data()["applied"].(map[string]any)["shopping"].(map[string]any)
	id := r.data()["project"].(map[string]any)["id"].(string)
	if num(applied["created"]) != 37 || num(applied["skipped"]) != 0 || len(list(applied["warnings"])) != 0 {
		t.Fatalf("applied = %v", applied)
	}
	ov := shoppingOverview(e, tok, id)
	k := ov["kpi"].(map[string]any)
	if num(k["estimated_total"]) != 4_578_500 || num(k["actual_total"]) != 0 || num(k["not_started"]) != 37 || num(k["total"]) != 37 {
		t.Errorf("kpi = %v", k)
	}
	items := list(ov["items"])
	for i, it := range items {
		want := domain.WeddingShopping[i]
		if it["item_name"] != want.Name || it["category"] != want.Category || num(it["estimated_price"]) != want.Price || it["status"] != "belum" || num(it["actual_price"]) != 0 ||
			it["template_key"] != want.Key() || it["sub_event_scope"] != "all" || num(it["quantity"]) != 1 || num(it["sort_order"]) != int64(i+1) || it["purchase_location"] != nil {
			t.Fatalf("item %d = %v", i, it)
		}
	}

	// no vendor is created by the wizard
	if n := len(list(vendorOverview(e, tok, id)["vendors"])); n != 0 {
		t.Errorf("%d vendors", n)
	}

	// a type without a shopping template: the wizard refuses (it is not offered), and no project remains
	wantError(t, e.do(tok, "POST", "/v1/planner/projects", project(map[string]any{
		"event_type": "seminar", "title": "Seminar", "metadata": map[string]any{"organizer_name": "Panitia"}, "templates": map[string]any{"shopping_list": true},
	})), 400, "invalid_template", "")
	if n := count(e, `SELECT count(*) FROM planner_projects WHERE title = 'Seminar'`); n != 0 {
		t.Errorf("%d projects remain", n)
	}
}

func TestShoppingTemplateAppendIsIdempotentAndFillsGaps(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)

	r := mustStatus(t, applyTemplate(e, tok, id, map[string]any{}), 200).data() // mode defaults to append
	if num(r["created"]) != 37 || num(r["skipped"]) != 0 || num(r["deleted"]) != 0 || len(list(r["warnings"])) != 0 {
		t.Fatalf("first = %v", r)
	}
	if r := mustStatus(t, applyTemplate(e, tok, id, map[string]any{"mode": "append"}), 200).data(); num(r["created"]) != 0 || num(r["skipped"]) != 37 {
		t.Errorf("second = %v", r)
	}
	if r := mustStatus(t, applyTemplate(e, tok, id, nil), 200).data(); num(r["skipped"]) != 37 { // no body at all
		t.Errorf("without a body = %v", r)
	}

	// three items deleted by the user come back, the other 34 are skipped, and the user's own items are not touched
	own := addItem(e, tok, id, itemBody("Lain", "Barang sendiri", 10_000, nil))
	items := list(shoppingOverview(e, tok, id)["items"])
	for _, it := range items[:3] {
		mustStatus(t, e.do(tok, "DELETE", base(id)+"/shopping-items/"+it["id"].(string), nil), 204)
	}
	r = mustStatus(t, applyTemplate(e, tok, id, map[string]any{"mode": "append"}), 200).data()
	if num(r["created"]) != 3 || num(r["skipped"]) != 34 {
		t.Errorf("fill gaps = %v", r)
	}
	if k := shoppingOverview(e, tok, id)["kpi"].(map[string]any); num(k["total"]) != 38 || num(k["estimated_total"]) != 4_588_500 {
		t.Errorf("kpi = %v", k)
	}
	_ = own
}

func TestShoppingTemplateReplaceNeedsConfirmWhenItCarriesWork(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := weddingWithShopping(e, tok)
	own := addItem(e, tok, id, itemBody("Lain", "Barang sendiri", 10_000, nil))
	first := list(shoppingOverview(e, tok, id)["items"])[0]

	// nothing done yet: no confirm needed, all 37 are replaced, the user's own item stays
	r := mustStatus(t, applyTemplate(e, tok, id, map[string]any{"mode": "replace_template"}), 200).data()
	if num(r["deleted"]) != 37 || num(r["created"]) != 37 || num(r["skipped"]) != 0 || len(list(r["warnings"])) != 0 {
		t.Fatalf("replace = %v", r)
	}
	if k := shoppingOverview(e, tok, id)["kpi"].(map[string]any); num(k["total"]) != 38 {
		t.Errorf("kpi = %v", k)
	}
	if n := count(e, `SELECT count(*) FROM planner_shopping_items WHERE id = $1`, own["id"]); n != 1 {
		t.Error("an item of the user was deleted")
	}
	if n := count(e, `SELECT count(*) FROM planner_shopping_items WHERE id = $1`, first["id"]); n != 0 {
		t.Error("the old template item must be gone")
	}

	// work of the user: a status, an actual price, a link, the souvenir mark - each one needs confirm
	for name, mutate := range map[string]string{
		"a status":   `UPDATE planner_shopping_items SET status = 'proses' WHERE template_key = 'wedding.seserahan_mukena'`,
		"a price":    `UPDATE planner_shopping_items SET actual_price = 1 WHERE template_key = 'wedding.seserahan_mukena'`,
		"a souvenir": `UPDATE planner_shopping_items SET purpose = 'souvenir' WHERE template_key = 'wedding.seserahan_mukena'`,
	} {
		exec(e, mutate)
		wantError(t, applyTemplate(e, tok, id, map[string]any{"mode": "replace_template"}), 400, "invalid_input", "confirm: required")
		if n := count(e, `SELECT count(*) FROM planner_shopping_items WHERE status <> 'belum' OR actual_price > 0 OR purpose IS NOT NULL`); n != 1 {
			t.Errorf("%s: a refused replace changed the list (%d)", name, n)
		}
		r = mustStatus(t, applyTemplate(e, tok, id, map[string]any{"mode": "replace_template", "confirm": true}), 200).data()
		if num(r["deleted"]) != 37 || num(r["created"]) != 37 {
			t.Errorf("%s: confirmed replace = %v", name, r)
		}
		if n := count(e, `SELECT count(*) FROM planner_shopping_items WHERE status <> 'belum' OR actual_price > 0 OR purpose IS NOT NULL`); n != 0 {
			t.Errorf("%s: the replaced template must start clean (%d)", name, n)
		}
	}

	// rolled-up items: the replace says the link was lost, the line recomputes from what is there
	mustStatus(t, linkToBudget(e, tok, id, nil), 200)
	wantError(t, applyTemplate(e, tok, id, map[string]any{"mode": "replace_template"}), 400, "invalid_input", "confirm: required")
	r = mustStatus(t, applyTemplate(e, tok, id, map[string]any{"mode": "replace_template", "confirm": true}), 200).data()
	w := list(r["warnings"])
	if len(w) != 1 || w[0]["code"] != "budget_link_lost" || num(w[0]["count"]) != 37 {
		t.Errorf("warnings = %v", w)
	}
	// the 37 template items were linked and are gone; the user's own item was linked by "link everything" too and survives
	if l := lineOf(e, tok, id, "resepsi/seserahan"); num(l["estimate"]) != 10_000 || l["rollup_source"] != "shopping" {
		t.Errorf("the line after the replace: %v", l)
	}
}

func TestShoppingTemplateIsAskedForTheWeddingOnly(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	defer func() { e.t = t }()

	for _, spec := range pd.DefaultRegistry().All() {
		if spec.Key == pd.EventWedding {
			continue
		}
		e.t = t
		id := createProject(e, tok, map[string]any{"event_type": string(spec.Key), "title": spec.Label, "metadata": metadataFor(spec), "event_date": inDays(60)})
		for _, mode := range []string{"append", "replace_template"} {
			wantError(t, applyTemplate(e, tok, id, map[string]any{"mode": mode}), 400, "invalid_template", "shopping_list: not available for this event type")
		}
		if n := len(list(shoppingOverview(e, tok, id)["items"])); n != 0 {
			t.Errorf("%s: %d items", spec.Key, n)
		}
	}

	id := createProject(e, tok, map[string]any{"title": "Wedding"})
	wantError(t, applyTemplate(e, tok, id, map[string]any{"mode": "merge"}), 400, "invalid_input", "mode: must be one of append, replace_template")
	wantError(t, applyTemplate(e, tok, id, `{"mode": 3}`), 400, "invalid_json", "")
}
