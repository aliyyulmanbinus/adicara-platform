package http_test

import (
	"fmt"
	"testing"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

// Menu 04 is not a wedding feature. Every event type of the registry goes through the whole flow on the same endpoints with
// nothing but the registry to tell them apart: the checklist, the limits read from the scenario, the groups of categories that
// share a line, lock-deal without a second line, the shopping list and its rollup, the dashboard widget and the exports.
func TestEveryEventTypeRunsTheWholeVendorFlow(t *testing.T) {
	e := newEnv(t)
	defer func() { e.t = t }()

	sharedGroups := map[pd.EventType]int{
		pd.EventWedding: 3, pd.EventPeresmian: 1, pd.EventLaunching: 1, pd.EventBirthday: 1, pd.EventSeminar: 1,
	}
	for _, spec := range pd.DefaultRegistry().All() {
		t.Run(string(spec.Key), func(t *testing.T) {
			e.t = t
			tok := e.login("user_" + string(spec.Key)) // the exports are limited per user
			id := createProject(e, tok, map[string]any{
				"event_type": string(spec.Key), "title": "Uji " + spec.Label, "event_date": inDays(120), "target_budget": 50_000_000,
				"metadata": metadataFor(spec), "templates": map[string]any{"budget_lines": true},
			})
			applyFirstPreset(t, e, tok, id)

			scope := spec.Sessions[0].ScopeKey
			if spec.Key == pd.EventWedding {
				scope = "resepsi" // the sessions of the wedding scenario lines
			}
			ids := addOneVendorPerCategory(t, e, tok, id, spec, scope)

			ov := vendorOverview(e, tok, id)
			checkLimits(t, spec, ov, sharedGroups[spec.Key])
			linesBefore := len(budgetLines(e, tok, id))

			keyless := lockEveryDeal(t, e, tok, id, spec, ids)
			checkNoDuplicateLines(t, e, tok, id, spec, ids, linesBefore, keyless)

			k := get(e, tok, base(id)+"/dashboard")["vendors"].(map[string]any)
			if int(num(k["categories_total"])) != len(spec.VendorCategories) || int(num(k["categories_fixed"])) != len(spec.VendorCategories) || int(num(k["fixed_vendors"])) != len(spec.VendorCategories) || num(k["pipeline"]) != 0 {
				t.Errorf("dashboard.vendors = %v", k)
			}

			shoppingAndRollup(t, e, tok, id, spec)
			for _, path := range []string{"/vendors/export-contacts?format=xlsx", "/shopping-items/export-shopping-list?format=xlsx"} {
				if r := mustStatus(t, e.do(tok, "GET", base(id)+path, nil), 200); len(r.body) < 1000 || string(r.body[:2]) != "PK" {
					t.Errorf("%s: %d bytes, not a workbook", path, len(r.body))
				}
			}
		})
	}
}

// applyFirstPreset creates a scenario from the first preset of the type and applies it, which makes it the active one.
func applyFirstPreset(t *testing.T, e *env, tok, id string) {
	t.Helper()
	presets := list(get(e, tok, base(id)+"/scenarios")["presets"])
	if len(presets) == 0 {
		t.Fatal("no preset offered")
	}
	body := map[string]any{"name": "Dari preset", "preset_key": presets[0]["key"]}
	if presets[0]["kind"] == "percent" {
		body["total_target"] = 50_000_000
	}
	sc := mustStatus(t, e.do(tok, "POST", base(id)+"/scenarios", body), 201).data()["scenario"].(map[string]any)
	mustStatus(t, e.do(tok, "POST", base(id)+"/scenarios/apply", map[string]any{"scenario_id": sc["id"], "mode": "smart_adjust"}), 200)
}

// addOneVendorPerCategory puts a vendor with an offer of 1.000.000 in every category of the checklist, in a real session.
func addOneVendorPerCategory(t *testing.T, e *env, tok, id string, spec pd.EventTypeSpec, scope string) map[string]string {
	t.Helper()
	ids := map[string]string{}
	for i, c := range spec.VendorCategories {
		v := addVendor(e, tok, id, map[string]any{
			"sub_event_scope": scope, "category_key": c.Key, "vendor_name": fmt.Sprintf("Vendor %s %d", c.Label, i), "agreed_price": 1_000_000,
			"phone_number": fmt.Sprintf("0812000%05d", i),
		})
		if v["category_label"] != c.Label || v["sub_event_scope"] != scope {
			t.Fatalf("vendor = %v", v)
		}
		ids[c.Key] = v["id"].(string)
	}

	return ids
}

// checkLimits: a category with a line gets its limit from the active scenario - alone or as a member of a group - and a
// category without a line gets none, whatever the session of the vendor is.
func checkLimits(t *testing.T, spec pd.EventTypeSpec, ov map[string]any, wantGroups int) {
	t.Helper()
	if k := ov["kpi"].(map[string]any); int(num(k["categories_total"])) != len(spec.VendorCategories) || int(num(k["pipeline"])) != len(spec.VendorCategories) || num(k["fixed_vendors"]) != 0 {
		t.Errorf("kpi = %v", k)
	}
	if got := len(list(ov["limit_groups"])); got != wantGroups {
		t.Errorf("%d limit groups, want %d: %v", got, wantGroups, ov["limit_groups"])
	}
	byCat := map[string]map[string]any{}
	for _, v := range list(ov["vendors"]) {
		byCat[v["category_key"].(string)] = v
	}
	for _, c := range spec.VendorCategories {
		v := byCat[c.Key]
		switch {
		case len(c.BudgetItemKeys) == 0:
			if v["limit_source"] != "none" {
				t.Errorf("%s has no line but limit_source = %v", c.Key, v["limit_source"])
			}
		case v["limit_source"] != "scenario" && v["limit_source"] != "group":
			t.Errorf("%s (line %v) has limit_source %v: a vendor in a real session must still find the line of the scenario", c.Key, c.BudgetItemKeys, v["limit_source"])
		case v["limit_source"] == "scenario" && (num(v["budget_limit"]) > 0) != (v["price_delta"] != nil):
			// a scenario line of 0 (the 10 Jt package of the wedding has no photography) gives no limit to compare with
			t.Errorf("%s: %v", c.Key, v)
		}
	}
}

// lockEveryDeal locks all deals with the budget sync; it returns how many categories have no line of their own.
func lockEveryDeal(t *testing.T, e *env, tok, id string, spec pd.EventTypeSpec, ids map[string]string) (keyless int) {
	t.Helper()
	for _, c := range spec.VendorCategories {
		r := mustStatus(t, lockDeal(e, tok, id, ids[c.Key], 1_000_000, true), 200)
		if len(c.BudgetItemKeys) == 0 {
			keyless++
			if r.data()["created"] != true {
				t.Errorf("%s has no line: lock-deal must make one, got %v", c.Key, r.data())
			}
		}
	}

	return keyless
}

// checkNoDuplicateLines: after all deals the budget holds one line per dedicated category, none for the shared ones, and one
// new line for each category that had no line - never a second `venue` line next to the one of the template.
func checkNoDuplicateLines(t *testing.T, e *env, tok, id string, spec pd.EventTypeSpec, ids map[string]string, before, keyless int) {
	t.Helper()
	lines := list(get(e, tok, base(id)+"/budget/items")["items"])
	if len(lines) != before+keyless {
		t.Errorf("%d lines after the deals, want %d (+%d for the categories without a line)", len(lines), before, keyless)
	}
	holder := map[string]map[string]any{}
	for _, l := range lines {
		if vid, _ := l["vendor_id"].(string); vid != "" {
			holder[vid] = l
		}
	}
	shared := map[string]int{}
	for _, c := range spec.VendorCategories {
		if len(c.BudgetItemKeys) > 0 {
			shared[c.BudgetItemKeys[0]]++
		}
	}
	for _, c := range spec.VendorCategories {
		l := holder[ids[c.Key]]
		switch {
		case len(c.BudgetItemKeys) > 0 && shared[c.BudgetItemKeys[0]] > 1:
			if l != nil {
				t.Errorf("%s shares the line %s but holds a line: %v", c.Key, c.BudgetItemKeys[0], l)
			}
		case l == nil || num(l["estimate"]) != 1_000_000:
			t.Errorf("%s: line %v", c.Key, l)
		case len(c.BudgetItemKeys) > 0 && l["item_key"] != c.BudgetItemKeys[0]:
			t.Errorf("%s took the line %v, want %s", c.Key, l["item_key"], c.BudgetItemKeys[0])
		}
	}
	ov := vendorOverview(e, tok, id)
	for _, g := range list(ov["limit_groups"]) {
		if g["line"] == nil || num(g["total_agreed"]) != num(g["members"])*1_000_000 || int(num(g["members"])) != len(list(g["vendor_ids"])) {
			t.Errorf("group %v", g)
		}
	}
}

// rollupKey is the line the list rolls up into for this type once every vendor is locked: the default line of the type, unless
// a vendor category owns it (then the vendor holds it and the list goes to the fallback line).
func rollupKey(spec pd.EventTypeSpec) (key string, made bool) {
	if spec.Key == pd.EventOther {
		return spec.ShoppingRollup.Key, true
	}
	for _, c := range spec.VendorCategories {
		for _, k := range c.BudgetItemKeys {
			if k == spec.ShoppingRollup.Key {
				return "perlengkapan", true
			}
		}
	}

	return spec.ShoppingRollup.Key, false
}

// shoppingAndRollup: an item of the first shopping category rolls up into a line of the type, whatever the type, and the
// button never ends in an error because a vendor holds the default line.
func shoppingAndRollup(t *testing.T, e *env, tok, id string, spec pd.EventTypeSpec) {
	t.Helper()
	addItem(e, tok, id, itemBody(spec.ShoppingCategories[0], "Barang uji", 125_000, map[string]any{"quantity": 4, "purpose": "souvenir", "status": "selesai"}))
	if k := shoppingOverview(e, tok, id)["kpi"].(map[string]any); num(k["souvenir_stock"]) != 4 || num(k["estimated_total"]) != 500_000 {
		t.Errorf("kpi = %v", k)
	}
	key, made := rollupKey(spec)
	if dt := shoppingOverview(e, tok, id)["default_target"].(map[string]any); dt["item_key"] != key || (dt["budget_item_id"] == nil) != made {
		t.Errorf("default_target = %v, want the key %s (made: %v)", dt, key, made)
	}

	before := len(budgetLines(e, tok, id))
	r := mustStatus(t, linkToBudget(e, tok, id, map[string]any{"confirm": true}), 200).data()
	line := r["line"].(map[string]any)
	if num(line["estimate"]) != 500_000 || num(r["linked"]) != 1 || line["item_name"] == "" {
		t.Errorf("rollup = %v", r)
	}
	lines := budgetLines(e, tok, id)
	if r["created"] != made || len(lines) != before+map[bool]int{true: 1, false: 0}[made] {
		t.Errorf("created %v (want %v), %d lines (was %d)", r["created"], made, len(lines), before)
	}
	if lines[line["sub_event_scope"].(string)+"/"+key]["rollup_source"] != "shopping" {
		t.Errorf("the line %q is not rolling up: %v", key, lines)
	}
}

// The registry reaches the client: the checklist and the shopping categories of every type, and nothing of the line keys.
func TestEventTypesCarryTheVendorRegistry(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")

	types := list(mustStatus(t, e.do(tok, "GET", "/v1/planner/event-types", nil), 200).json()["data"])
	if len(types) != 9 {
		t.Fatalf("%d types", len(types))
	}
	wantCount := map[string]int{"wedding": 10, "khitanan": 5, "aqiqah": 4, "peresmian": 7, "launching": 7, "birthday": 6, "wisuda": 6, "seminar": 6, "other": 5}
	for _, et := range types {
		cats := list(et["vendor_categories"])
		if len(cats) != wantCount[et["key"].(string)] {
			t.Errorf("%v: %d vendor categories", et["key"], len(cats))
		}
		for _, c := range cats {
			if len(c) != 2 || c["key"] == nil || c["label"] == nil {
				t.Errorf("%v: a category must be {key, label} only: %v", et["key"], c)
			}
		}
		if sc, ok := et["shopping_categories"].([]any); !ok || len(sc) < 3 {
			t.Errorf("%v: shopping_categories = %v", et["key"], et["shopping_categories"])
		}
		if _, leaked := et["shopping_rollup"]; leaked {
			t.Errorf("%v: the rollup line stays on the server", et["key"])
		}
	}
	first := list(types[0]["vendor_categories"])
	if types[0]["key"] != "wedding" || first[0]["key"] != "venue" || first[0]["label"] != "Venue" || first[1]["label"] != "Decoration" {
		t.Errorf("wedding = %v", first)
	}
}
