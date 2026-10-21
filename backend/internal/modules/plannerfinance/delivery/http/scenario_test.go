package http_test

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
)

func scenarioPath(id, scenario string) string { return base(id) + "/scenarios/" + scenario }

func createScenario(e *env, tok, id string, body map[string]any) map[string]any {
	e.t.Helper()
	r := mustStatus(e.t, e.do(tok, "POST", base(id)+"/scenarios", body), 201)

	return r.data()["scenario"].(map[string]any)
}

func itemsByKey(sc map[string]any) map[string]map[string]any {
	out := map[string]map[string]any{}
	for _, it := range list(sc["items"]) {
		k, _ := it["item_key"].(string)
		out[it["sub_event_scope"].(string)+"/"+k] = it
	}

	return out
}

// A project with the 14 template lines (all estimate 0) and no scenario.
func templateProject(e *env, tok string) string {
	return createProject(e, tok, map[string]any{"templates": map[string]any{"budget_lines": true}})
}

func TestCreateScenarioFromPresets(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)

	// An empty scenario needs a total.
	wantError(t, e.do(tok, "POST", base(id)+"/scenarios", map[string]any{"name": "Kosong"}), 400, "invalid_input", "total_target: required")
	wantError(t, e.do(tok, "POST", base(id)+"/scenarios", map[string]any{"name": "Nol", "total_target": 0}), 400, "invalid_input", "total_target: must be between 1 and 10000000000000")
	wantError(t, e.do(tok, "POST", base(id)+"/scenarios", map[string]any{"name": "  ", "total_target": 1}), 400, "invalid_input", "name: must not be empty")
	empty := createScenario(e, tok, id, map[string]any{"name": "  Kosong  ", "total_target": 20_000_000})
	if empty["name"] != "Kosong" || len(list(empty["items"])) != 0 || empty["is_active"] != false || len(empty["active_sub_events"].([]any)) != 0 || num(empty["allocated"]) != 0 {
		t.Errorf("empty scenario = %v", empty)
	}

	// A fixed preset: the total may be left out or repeated, never different.
	wantError(t, e.do(tok, "POST", base(id)+"/scenarios", map[string]any{"name": "x", "preset_key": "wedding_50jt", "total_target": 49_000_000}),
		400, "invalid_input", "total_target: must match the preset total")
	wantError(t, e.do(tok, "POST", base(id)+"/scenarios", map[string]any{"name": "x", "preset_key": "wedding_pct"}), 400, "invalid_template", "")
	wantError(t, e.do(tok, "POST", base(id)+"/scenarios", map[string]any{"name": "x", "preset_key": "seminar_pct", "total_target": 10_000_000}), 400, "invalid_template", "")
	hemat := createScenario(e, tok, id, map[string]any{"name": "Hemat", "preset_key": "wedding_50jt", "active_sub_events": []string{"akad", "resepsi", "akad"}})
	if num(hemat["total_target"]) != 50_000_000 || hemat["preset_key"] != "wedding_50jt" || len(list(hemat["items"])) != 10 || num(hemat["allocated"]) != 50_000_000 {
		t.Errorf("preset scenario = %v", hemat)
	}
	if got := hemat["active_sub_events"].([]any); len(got) != 2 {
		t.Errorf("duplicates in active_sub_events must be dropped: %v", got)
	}
	it := itemsByKey(hemat)["resepsi/catering"]
	if it == nil || num(it["cost_amount"]) != 20_000_000 || num(it["preset_amount"]) != 20_000_000 || it["included"] != true || it["item_name"] != "Catering" {
		t.Errorf("resepsi/catering = %v", it)
	}

	// Unknown sessions are refused, and so is the general one: it always takes part.
	wantError(t, e.do(tok, "POST", base(id)+"/scenarios", map[string]any{"name": "x", "total_target": 1, "active_sub_events": []string{"all"}}), 400, "invalid_scope", "")
	wantError(t, e.do(tok, "POST", base(id)+"/scenarios", map[string]any{"name": "x", "total_target": 1, "active_sub_events": []string{"nope"}}), 400, "invalid_scope", "")

	same := createScenario(e, tok, id, map[string]any{"name": "Sama", "preset_key": "wedding_75jt", "total_target": 75_000_000})
	if num(same["total_target"]) != 75_000_000 {
		t.Errorf("repeated preset total: %v", same)
	}
}

func TestPercentPresetForOtherEventTypes(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, map[string]any{
		"event_type": "seminar", "metadata": map[string]any{"organizer_name": "BEM"}, "target_budget": 50_000_000,
	})

	list0 := get(e, tok, base(id)+"/scenarios")
	presets := list(list0["presets"])
	if len(presets) != 1 || presets[0]["key"] != "seminar_pct" || presets[0]["kind"] != "percent" || presets[0]["total_target"] != nil {
		t.Fatalf("presets = %v", presets)
	}
	if len(list(list0["scenarios"])) != 0 {
		t.Fatal("a new project has no scenario")
	}

	wantError(t, e.do(tok, "POST", base(id)+"/scenarios", map[string]any{"name": "x", "preset_key": "seminar_pct"}), 400, "invalid_input", "total_target: required")
	wantError(t, e.do(tok, "POST", base(id)+"/scenarios", map[string]any{"name": "x", "preset_key": "seminar_pct", "total_target": 99_999}),
		400, "invalid_input", "total_target: must be at least 100000")
	sc := createScenario(e, tok, id, map[string]any{"name": "Seminar 50", "preset_key": "seminar_pct", "total_target": 50_000_000})
	its := itemsByKey(sc)
	if len(its) != 8 || num(its["all/venue"]["cost_amount"]) != 12_500_000 || num(its["all/katering_coffee_break"]["cost_amount"]) != 10_000_000 ||
		num(its["all/dana_darurat"]["cost_amount"]) != 3_500_000 || num(sc["allocated"]) != 50_000_000 {
		t.Errorf("seminar preset = %v", sc)
	}
	wantError(t, e.do(tok, "POST", base(id)+"/scenarios", map[string]any{"name": "x", "preset_key": "wedding_50jt"}), 400, "invalid_template", "")
}

func TestScenarioPresetFallsBackToTheGeneralSession(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	for _, s := range sessionsOf(e, tok, id) {
		if s["scope_key"] == "resepsi" {
			mustStatus(t, e.do(tok, "DELETE", base(id)+"/sessions/"+s["id"].(string), nil), 204)
		}
	}

	r := mustStatus(t, e.do(tok, "POST", base(id)+"/scenarios", map[string]any{"name": "Hemat", "preset_key": "wedding_50jt"}), 201)
	warnings := list(r.data()["warnings"])
	if len(warnings) != 1 || warnings[0]["code"] != "scope_fallback" || num(warnings[0]["count"]) != 8 {
		t.Fatalf("warnings = %v", warnings)
	}
	its := itemsByKey(r.data()["scenario"].(map[string]any))
	if its["all/catering"] == nil || its["akad/biaya_kua"] == nil || its["resepsi/catering"] != nil {
		t.Errorf("items = %v", its)
	}
}

func TestScenarioLimitCloneAndDeleteActive(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)

	a := createScenario(e, tok, id, map[string]any{"name": "A", "preset_key": "wedding_30jt"})
	createScenario(e, tok, id, map[string]any{"name": "B", "total_target": 40_000_000})
	clone := mustStatus(t, e.do(tok, "POST", scenarioPath(id, a["id"].(string))+"/clone", map[string]any{"new_name": "Salinan A"}), 201).data()["scenario"].(map[string]any)
	if clone["name"] != "Salinan A" || num(clone["total_target"]) != 30_000_000 || clone["preset_key"] != "wedding_30jt" || clone["is_active"] != false ||
		len(list(clone["items"])) != 10 || clone["id"] == a["id"] {
		t.Errorf("clone = %v", clone)
	}
	ca, cb := itemsByKey(a), itemsByKey(clone)
	if ca["resepsi/catering"]["id"] == cb["resepsi/catering"]["id"] {
		t.Error("a clone must get its own item ids")
	}

	// The fourth scenario is refused: new, clone and also a clone of the clone.
	wantError(t, e.do(tok, "POST", base(id)+"/scenarios", map[string]any{"name": "D", "total_target": 1}), 409, "max_scenarios_reached", "")
	wantError(t, e.do(tok, "POST", scenarioPath(id, a["id"].(string))+"/clone", map[string]any{"new_name": "Lagi"}), 409, "max_scenarios_reached", "")
	wantError(t, e.do(tok, "POST", scenarioPath(id, a["id"].(string))+"/clone", map[string]any{"new_name": " "}), 400, "invalid_input", "new_name: must not be empty")

	// Delete the active scenario: no other one is activated, and its items are gone with it.
	mustStatus(t, e.do(tok, "POST", base(id)+"/budget/apply-template", map[string]any{}), 200)
	mustStatus(t, e.do(tok, "POST", base(id)+"/scenarios/apply", map[string]any{"scenario_id": a["id"], "mode": "smart_adjust"}), 200)
	if n := count(e, `SELECT count(*) FROM planner_budget_scenarios WHERE project_id=$1 AND is_active`, id); n != 1 {
		t.Fatalf("%d active scenarios", n)
	}
	mustStatus(t, e.do(tok, "DELETE", scenarioPath(id, a["id"].(string)), nil), 204)
	if n := count(e, `SELECT count(*) FROM planner_budget_scenarios WHERE project_id=$1 AND is_active`, id); n != 0 {
		t.Errorf("%d active scenarios after deleting the active one, want 0", n)
	}
	if n := count(e, `SELECT count(*) FROM planner_budget_scenario_items WHERE scenario_id=$1`, a["id"]); n != 0 {
		t.Errorf("%d items left of the deleted scenario", n)
	}
	wantError(t, e.do(tok, "GET", scenarioPath(id, a["id"].(string)), nil), 404, "not_found", "scenario not found")
	wantError(t, e.do(tok, "DELETE", scenarioPath(id, a["id"].(string)), nil), 404, "not_found", "scenario not found")
	createScenario(e, tok, id, map[string]any{"name": "Baru", "total_target": 1})
}

func TestTwoSimultaneousCreatesCannotPassTheScenarioLimit(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	createScenario(e, tok, id, map[string]any{"name": "A", "total_target": 1})
	createScenario(e, tok, id, map[string]any{"name": "B", "total_target": 2})

	for round := 0; round < 3; round++ {
		exec(e, `DELETE FROM planner_budget_scenarios WHERE project_id=$1 AND name NOT IN ('A','B')`, id)
		statuses := concurrently(6, func(i int) int {
			return e.do(tok, "POST", base(id)+"/scenarios", map[string]any{"name": fmt.Sprintf("C%d", i), "total_target": 3}).status
		})
		created := 0
		for _, s := range statuses {
			if s == 201 {
				created++
			} else if s != 409 {
				t.Fatalf("round %d: unexpected status %d in %v", round, s, statuses)
			}
		}
		if n := count(e, `SELECT count(*) FROM planner_budget_scenarios WHERE project_id=$1`, id); n != 3 || created != 1 {
			t.Fatalf("round %d: %d scenarios, %d created, statuses %v; want 3 and 1", round, n, created, statuses)
		}
	}
}

func TestUpdateScenarioReplacesTheHeader(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	sc := createScenario(e, tok, id, map[string]any{"name": "A", "preset_key": "wedding_50jt", "active_sub_events": []string{"akad"}})
	path := scenarioPath(id, sc["id"].(string))

	r := mustStatus(t, e.do(tok, "PUT", path, map[string]any{"name": "Ubah", "total_target": 60_000_000, "active_sub_events": []string{"resepsi"}}), 200).data()
	if r["name"] != "Ubah" || num(r["total_target"]) != 60_000_000 || len(list(r["items"])) != 10 || num(r["allocated"]) == 50_000_000 {
		t.Errorf("updated = %v", r)
	}
	// Changing the total does not scale the items; only the resepsi and general items take part now.
	if its := itemsByKey(r); its["akad/biaya_kua"]["included"] != false || its["resepsi/catering"]["included"] != true || its["all/dana_darurat"]["included"] != true {
		t.Errorf("included flags = %v", its)
	}
	// PUT replaces: no active_sub_events means every session again.
	r = mustStatus(t, e.do(tok, "PUT", path, map[string]any{"name": "Ubah", "total_target": 60_000_000}), 200).data()
	if len(r["active_sub_events"].([]any)) != 0 || num(r["allocated"]) != 50_000_000 {
		t.Errorf("after PUT without sessions = %v", r)
	}
	wantError(t, e.do(tok, "PUT", path, map[string]any{"name": "", "total_target": 1}), 400, "invalid_input", "name: must not be empty")
	wantError(t, e.do(tok, "PUT", path, map[string]any{"name": "x", "total_target": 0}), 400, "invalid_input", "")
	wantError(t, e.do(tok, "PUT", scenarioPath(id, uuid.NewString()), map[string]any{"name": "x", "total_target": 1}), 404, "not_found", "scenario not found")
}

func TestLoadPreset(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	sc := createScenario(e, tok, id, map[string]any{"name": "A", "total_target": 20_000_000})
	path := scenarioPath(id, sc["id"].(string)) + "/load-preset"

	// An empty scenario takes a package without confirmation; the package sets the total and the preset.
	r := mustStatus(t, e.do(tok, "POST", path, map[string]any{"preset_key": "wedding_30jt"}), 200).data()["scenario"].(map[string]any)
	if len(list(r["items"])) != 10 || num(r["total_target"]) != 30_000_000 || r["preset_key"] != "wedding_30jt" {
		t.Errorf("loaded = %v", r)
	}
	// A scenario that has items needs confirm before they are replaced.
	wantError(t, e.do(tok, "POST", path, map[string]any{"preset_key": "wedding_75jt"}), 400, "invalid_input", "confirm: required")
	r = mustStatus(t, e.do(tok, "POST", path, map[string]any{"preset_key": "wedding_75jt", "confirm": true}), 200).data()["scenario"].(map[string]any)
	if num(r["total_target"]) != 75_000_000 || num(itemsByKey(r)["resepsi/catering"]["cost_amount"]) != 30_000_000 || count(e, `SELECT count(*) FROM planner_budget_scenario_items WHERE scenario_id=$1`, sc["id"]) != 10 {
		t.Errorf("replaced = %v", r)
	}
	wantError(t, e.do(tok, "POST", path, map[string]any{"preset_key": "khitanan_pct", "confirm": true}), 400, "invalid_template", "")
	wantError(t, e.do(tok, "POST", path, map[string]any{"confirm": true}), 400, "invalid_input", "preset_key: required")
}

func TestScenarioItems(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	sc := createScenario(e, tok, id, map[string]any{"name": "A", "preset_key": "wedding_10jt"})
	path := scenarioPath(id, sc["id"].(string)) + "/items"

	body := map[string]any{"sub_event_scope": "resepsi", "item_key": "photobooth", "category": "Hiburan", "item_name": "Photobooth", "cost_amount": 1_500_000}
	it := mustStatus(t, e.do(tok, "POST", path, body), 201).data()
	if it["item_key"] != "photobooth" || num(it["cost_amount"]) != 1_500_000 || it["included"] != true || it["sub_event_scope"] != "resepsi" {
		t.Errorf("created item = %v", it)
	}
	// The same key twice in a session is a conflict; the same key in another session is not.
	wantError(t, e.do(tok, "POST", path, body), 409, "conflict", "item_key: already used in this session")
	body2 := map[string]any{"sub_event_scope": "akad", "item_key": "photobooth", "category": "Hiburan", "item_name": "Photobooth", "cost_amount": 1}
	mustStatus(t, e.do(tok, "POST", path, body2), 201)
	wantError(t, e.do(tok, "POST", path, map[string]any{"item_key": "Bad Key", "category": "x", "item_name": "x"}), 400, "invalid_input", "item_key: must match ^[a-z0-9_]{1,40}$")
	wantError(t, e.do(tok, "POST", path, map[string]any{"sub_event_scope": "nope", "category": "x", "item_name": "x"}), 400, "invalid_scope", "")
	wantError(t, e.do(tok, "POST", path, map[string]any{"category": "", "item_name": "x"}), 400, "invalid_input", "category: must not be empty")
	wantError(t, e.do(tok, "POST", path, map[string]any{"category": "x", "item_name": "x", "cost_amount": -1}), 400, "invalid_input", "cost_amount: must be between 0 and 10000000000000")

	// PUT replaces name, category, cost and session; item_key never changes and sending it is refused.
	itemPath := path + "/" + it["id"].(string)
	wantError(t, e.do(tok, "PUT", itemPath, map[string]any{"sub_event_scope": "resepsi", "category": "x", "item_name": "x", "cost_amount": 1, "item_key": "other"}),
		400, "invalid_input", "item_key: cannot be changed")
	up := mustStatus(t, e.do(tok, "PUT", itemPath, map[string]any{"sub_event_scope": "resepsi", "category": "Hiburan", "item_name": "Photobooth 360", "cost_amount": 2_000_000}), 200).data()
	if up["item_name"] != "Photobooth 360" || num(up["cost_amount"]) != 2_000_000 || up["item_key"] != "photobooth" {
		t.Errorf("updated item = %v", up)
	}
	// Moving it to a session that already has the key makes it a keyless item instead of failing (01 BE 4.6).
	moved := mustStatus(t, e.do(tok, "PUT", itemPath, map[string]any{"sub_event_scope": "akad", "category": "Hiburan", "item_name": "Photobooth 360", "cost_amount": 2_000_000}), 200).data()
	if moved["item_key"] != nil || moved["sub_event_scope"] != "akad" {
		t.Errorf("moved item = %v", moved)
	}
	mustStatus(t, e.do(tok, "DELETE", itemPath, nil), 204)
	wantError(t, e.do(tok, "DELETE", itemPath, nil), 404, "not_found", "scenario item not found")
	wantError(t, e.do(tok, "PUT", itemPath, map[string]any{"category": "x", "item_name": "x"}), 404, "not_found", "scenario item not found")
	wantError(t, e.do(tok, "POST", scenarioPath(id, uuid.NewString())+"/items", body), 404, "not_found", "scenario not found")
}

func TestScenarioItemLimit(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	sc := createScenario(e, tok, id, map[string]any{"name": "A", "total_target": 1})
	exec(e, `INSERT INTO planner_budget_scenario_items (id, scenario_id, project_id, sub_event_scope, category, item_name, cost_amount, sort_order)
	         SELECT gen_random_uuid(), $1, $2, 'all', 'x', 'item ' || n, 0, n FROM generate_series(1, 100) n`, sc["id"], id)

	body := map[string]any{"category": "x", "item_name": "ke-101"}
	wantError(t, e.do(tok, "POST", scenarioPath(id, sc["id"].(string))+"/items", body), 409, "scenario_item_limit_reached", "")
	statuses := concurrently(5, func(int) int { return e.do(tok, "POST", scenarioPath(id, sc["id"].(string))+"/items", body).status })
	for _, s := range statuses {
		if s != 409 {
			t.Fatalf("a full scenario accepted another item: %v", statuses)
		}
	}
}

func TestCompareScenarios(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	a := createScenario(e, tok, id, map[string]any{"name": "Hemat", "preset_key": "wedding_50jt"})
	b := createScenario(e, tok, id, map[string]any{"name": "Ideal", "preset_key": "wedding_75jt"})
	mustStatus(t, e.do(tok, "POST", scenarioPath(id, b["id"].(string))+"/items", map[string]any{"sub_event_scope": "resepsi", "category": "Hiburan", "item_name": "Photobooth", "cost_amount": 1_500_000}), 201)
	mustStatus(t, e.do(tok, "POST", scenarioPath(id, a["id"].(string))+"/items", map[string]any{"sub_event_scope": "resepsi", "category": "Lain", "item_name": "Parkir", "cost_amount": 500_000}), 201)

	r := mustStatus(t, e.do(tok, "POST", base(id)+"/scenarios/compare", map[string]any{"a": a["id"], "b": b["id"]}), 200).data()
	rows := list(r["rows"])
	if len(rows) != 12 {
		t.Fatalf("%d rows, want 10 shared + 2 one-sided", len(rows))
	}
	byKey := map[string]map[string]any{}
	for _, row := range rows {
		k, _ := row["item_key"].(string)
		if k == "" {
			k = row["item_name"].(string)
		}
		byKey[k] = row
	}
	if c := byKey["catering"]; num(c["a_amount"]) != 20_000_000 || num(c["b_amount"]) != 30_000_000 || num(c["delta"]) != 10_000_000 {
		t.Errorf("catering = %v", c)
	}
	if p := byKey["Photobooth"]; num(p["a_amount"]) != 0 || num(p["b_amount"]) != 1_500_000 || num(p["delta"]) != 1_500_000 {
		t.Errorf("photobooth = %v", p)
	}
	if p := byKey["Parkir"]; num(p["a_amount"]) != 500_000 || num(p["b_amount"]) != 0 || num(p["delta"]) != -500_000 {
		t.Errorf("parkir = %v", p)
	}
	tot := r["totals"].(map[string]any)
	if num(tot["a"]) != 50_500_000 || num(tot["b"]) != 76_500_000 || num(tot["delta"]) != 26_000_000 {
		t.Errorf("totals = %v", tot)
	}
	if r["a"].(map[string]any)["name"] != "Hemat" || num(r["b"].(map[string]any)["total_target"]) != 75_000_000 {
		t.Errorf("sides = %v %v", r["a"], r["b"])
	}

	wantError(t, e.do(tok, "POST", base(id)+"/scenarios/compare", map[string]any{"a": a["id"], "b": a["id"]}), 400, "invalid_input", "b: must differ from a")
	wantError(t, e.do(tok, "POST", base(id)+"/scenarios/compare", map[string]any{"b": a["id"]}), 400, "invalid_input", "a: required")
	wantError(t, e.do(tok, "POST", base(id)+"/scenarios/compare", map[string]any{"a": a["id"]}), 400, "invalid_input", "b: required")
	wantError(t, e.do(tok, "POST", base(id)+"/scenarios/compare", map[string]any{"a": a["id"], "b": uuid.NewString()}), 404, "not_found", "scenario not found")
}

// ---------- applying a scenario ----------

// TestApplySmartAdjust follows 03 BE 3.2: the 14 template lines (estimate 0)
// receive the 75 Jt package: 9 are updated, 1 is created, the 5 Lamaran lines
// are not touched, and a payment already made stays.
func TestApplySmartAdjust(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := templateProject(e, tok)
	sc := createScenario(e, tok, id, map[string]any{"name": "Ideal", "preset_key": "wedding_75jt", "active_sub_events": []string{"akad", "resepsi"}})
	foto := lines(e, tok, id)["lamaran/fotografer"]
	mustStatus(t, e.do(tok, "PATCH", base(id)+"/budget/items/"+foto["id"].(string)+"/payment",
		map[string]any{"payment_stage": "dp", "amount": 500_000, "paid_at": today}), 200)
	catering := lines(e, tok, id)["resepsi/catering"]
	mustStatus(t, e.do(tok, "PATCH", base(id)+"/budget/items/"+catering["id"].(string)+"/payment",
		map[string]any{"payment_stage": "dp", "amount": 5_000_000, "paid_at": today}), 200)

	body := map[string]any{"scenario_id": sc["id"], "mode": "smart_adjust", "sync_target_budget": false}
	preview := mustStatus(t, e.do(tok, "POST", base(id)+"/scenarios/apply-preview", body), 200).data()
	if n := count(e, `SELECT count(*) FROM planner_budget_items WHERE project_id=$1`, id); n != 14 {
		t.Fatalf("apply-preview wrote: %d lines", n)
	}
	if count(e, `SELECT count(*) FROM planner_budget_scenarios WHERE project_id=$1 AND is_active`, id) != 0 {
		t.Fatal("apply-preview activated the scenario")
	}

	applied := mustStatus(t, e.do(tok, "POST", base(id)+"/scenarios/apply", body), 200).data()
	if fmt.Sprint(preview) != fmt.Sprint(applied) {
		t.Errorf("apply-preview and apply differ:\n%v\n%v", preview, applied)
	}
	if len(list(applied["updated"])) != 9 || len(list(applied["created"])) != 1 || len(list(applied["skipped"])) != 0 ||
		num(applied["untouched_count"]) != 5 || num(applied["estimate_after"]) != 75_000_000 || applied["deleted"] != nil || applied["new_target_budget"] != nil {
		t.Errorf("report = %v", applied)
	}
	created := list(applied["created"])[0]
	if created["sub_event_scope"] != "all" || created["item_key"] != "dana_darurat" || num(created["amount"]) != 1_000_000 {
		t.Errorf("created = %v", created)
	}
	for _, u := range list(applied["updated"]) {
		if num(u["from"]) != 0 || num(u["to"]) <= 0 {
			t.Errorf("updated = %v", u)
		}
	}

	after := lines(e, tok, id)
	if len(after) != 15 || num(after["resepsi/catering"]["estimate"]) != 30_000_000 || num(after["akad/biaya_kua"]["estimate"]) != 600_000 ||
		after["all/dana_darurat"]["source"] != "scenario" || after["resepsi/catering"]["source"] != "template" {
		t.Errorf("lines after apply: %d, catering %v", len(after), after["resepsi/catering"])
	}
	// Payments stay, estimates of the Lamaran lines stay 0.
	if num(after["resepsi/catering"]["dp_amount"]) != 5_000_000 || num(after["lamaran/fotografer"]["dp_amount"]) != 500_000 || num(after["lamaran/fotografer"]["estimate"]) != 0 {
		t.Errorf("payments or untouched lines changed: %v %v", after["resepsi/catering"], after["lamaran/fotografer"])
	}
	m := get(e, tok, base(id)+"/budget/matrix")
	if as := m["active_scenario"].(map[string]any); as["name"] != "Ideal" || num(as["unallocated"]) != 0 {
		t.Errorf("active_scenario = %v", as)
	}

	// Applying the same scenario again changes nothing (idempotent).
	again := mustStatus(t, e.do(tok, "POST", base(id)+"/scenarios/apply", body), 200).data()
	if len(list(again["updated"])) != 0 || len(list(again["created"])) != 0 || num(again["estimate_after"]) != 75_000_000 || num(again["untouched_count"]) != 5 {
		t.Errorf("second apply = %v", again)
	}
	if n := count(e, `SELECT count(*) FROM planner_budget_items WHERE project_id=$1`, id); n != 15 {
		t.Errorf("%d lines after the second apply", n)
	}
}

func TestApplySyncsTheTargets(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := templateProject(e, tok)
	mustStatus(t, e.do(tok, "PATCH", base(id), map[string]any{"savings_target": 60_000_000}), 200)
	sc := createScenario(e, tok, id, map[string]any{"name": "Hemat", "preset_key": "wedding_50jt"})

	r := mustStatus(t, e.do(tok, "POST", base(id)+"/scenarios/apply", map[string]any{"scenario_id": sc["id"], "mode": "smart_adjust"}), 200).data()
	if r["new_target_budget"] != nil {
		t.Errorf("new_target_budget without sync: %v", r)
	}
	var target int64
	var savings *int64
	read := func() {
		if err := e.pool.QueryRow(t.Context(), `SELECT target_budget, savings_target FROM planner_projects WHERE id=$1`, id).Scan(&target, &savings); err != nil {
			t.Fatal(err)
		}
	}
	if read(); target != 75_000_000 || savings == nil || *savings != 60_000_000 {
		t.Fatalf("targets changed without sync: %d %v", target, savings)
	}

	r = mustStatus(t, e.do(tok, "POST", base(id)+"/scenarios/apply", map[string]any{"scenario_id": sc["id"], "mode": "smart_adjust", "sync_target_budget": true, "sync_target_savings": true}), 200).data()
	if num(r["new_target_budget"]) != 50_000_000 {
		t.Errorf("report = %v", r)
	}
	if read(); target != 50_000_000 || savings != nil {
		t.Errorf("after sync: target %d, savings %v (want 50M and NULL)", target, savings)
	}
	if sv := get(e, tok, base(id)+"/savings")["kpi"].(map[string]any); num(sv["target"]) != 50_000_000 || sv["target_follows_budget"] != true {
		t.Errorf("savings kpi = %v", sv)
	}
}

func TestApplySkipsRollupAndUnitPricingAndPairsKeylessLines(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := templateProject(e, tok)
	all := lines(e, tok, id)

	// seserahan rolls up a shopping item; catering (resepsi) is priced per unit.
	exec(e, `UPDATE planner_budget_items SET rollup_source='shopping' WHERE id=$1`, all["resepsi/seserahan"]["id"])
	exec(e, `INSERT INTO planner_shopping_items (id, project_id, sub_event_scope, category, item_name, quantity, estimated_price, actual_price, status, budget_item_id)
	         VALUES (gen_random_uuid(), $1, 'resepsi', 'Hantaran', 'Tas', 1, 1000000, 900000, 'selesai', $2)`, id, all["resepsi/seserahan"]["id"])
	mustStatus(t, e.do(tok, "PUT", base(id)+"/budget/items/"+all["resepsi/catering"]["id"].(string), map[string]any{
		"sub_event_scope": "resepsi", "category": "Catering", "item_name": "Catering", "pricing_mode": "unit", "quantity": 100, "unit_price": 150_000,
	}), 200)
	// A keyless line named like a scenario item without a key pairs by name.
	mustStatus(t, e.do(tok, "POST", base(id)+"/budget/items", map[string]any{"sub_event_scope": "resepsi", "category": "Hiburan", "item_name": "Photobooth", "initial_estimate": 750_000}), 201)

	sc := createScenario(e, tok, id, map[string]any{"name": "Ideal", "preset_key": "wedding_75jt"})
	mustStatus(t, e.do(tok, "POST", scenarioPath(id, sc["id"].(string))+"/items", map[string]any{"sub_event_scope": "resepsi", "category": "Hiburan", "item_name": "PHOTOBOOTH", "cost_amount": 1_500_000}), 201)

	r := mustStatus(t, e.do(tok, "POST", base(id)+"/scenarios/apply", map[string]any{"scenario_id": sc["id"], "mode": "smart_adjust"}), 200).data()
	skipped := map[string]string{}
	for _, s := range list(r["skipped"]) {
		skipped[s["item_key"].(string)] = s["reason"].(string)
	}
	if len(skipped) != 2 || skipped["seserahan"] != "rollup" || skipped["catering"] != "unit_pricing" {
		t.Errorf("skipped = %v", skipped)
	}
	after := lines(e, tok, id)
	if num(after["resepsi/seserahan"]["estimate"]) != 1_000_000 || num(after["resepsi/catering"]["estimate"]) != 15_000_000 {
		t.Errorf("rollup and unit lines changed: %v %v", after["resepsi/seserahan"], after["resepsi/catering"])
	}
	if num(after["resepsi/Photobooth"]["estimate"]) != 1_500_000 || len(after) != 16 {
		t.Errorf("keyless pairing: %v, %d lines", after["resepsi/Photobooth"], len(after))
	}
	if len(list(r["updated"])) != 8 || len(list(r["created"])) != 1 {
		t.Errorf("report = %v", r)
	}
}

func TestActivateScenarioSwitchesWithoutUniqueViolation(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := templateProject(e, tok)
	a := createScenario(e, tok, id, map[string]any{"name": "A", "preset_key": "wedding_50jt"})
	b := createScenario(e, tok, id, map[string]any{"name": "B", "preset_key": "wedding_75jt"})
	c := createScenario(e, tok, id, map[string]any{"name": "C", "preset_key": "wedding_100jt"})

	// Back and forth many times: a single UPDATE … SET is_active = (id = x) fails on the partial unique index on some of these.
	for i, sc := range []map[string]any{a, b, c, a, c, b, a, b, c, a} {
		mustStatus(t, e.do(tok, "POST", base(id)+"/scenarios/apply", map[string]any{"scenario_id": sc["id"], "mode": "smart_adjust"}), 200)
		if n := count(e, `SELECT count(*) FROM planner_budget_scenarios WHERE project_id=$1 AND is_active`, id); n != 1 {
			t.Fatalf("step %d: %d active scenarios", i, n)
		}
		if count(e, `SELECT count(*) FROM planner_budget_scenarios WHERE id=$1 AND is_active`, sc["id"]) != 1 {
			t.Fatalf("step %d: the applied scenario is not the active one", i)
		}
	}
	wantError(t, e.do(tok, "POST", base(id)+"/scenarios/apply", map[string]any{"scenario_id": uuid.NewString(), "mode": "smart_adjust"}), 404, "not_found", "scenario not found")
	wantError(t, e.do(tok, "POST", base(id)+"/scenarios/apply", map[string]any{"scenario_id": a["id"], "mode": "merge"}), 400, "invalid_input", "mode: must be one of smart_adjust, clean_overwrite")
	wantError(t, e.do(tok, "POST", base(id)+"/scenarios/apply", map[string]any{"mode": "smart_adjust"}), 400, "invalid_input", "scenario_id: required")
	wantError(t, e.do(tok, "POST", base(id)+"/scenarios/apply-preview", map[string]any{}), 400, "invalid_input", "scenario_id: required")
}

func TestApplyCleanOverwriteNeedsReset(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := templateProject(e, tok)
	foto := lines(e, tok, id)["lamaran/fotografer"]
	mustStatus(t, e.do(tok, "PATCH", base(id)+"/budget/items/"+foto["id"].(string)+"/payment", map[string]any{"payment_stage": "dp", "amount": 500_000, "paid_at": today}), 200)
	mustStatus(t, e.do(tok, "POST", base(id)+"/budget/items", map[string]any{"sub_event_scope": "all", "category": "x", "item_name": "Kustom", "initial_estimate": 10}), 201)
	sc := createScenario(e, tok, id, map[string]any{"name": "Ideal", "preset_key": "wedding_75jt", "active_sub_events": []string{"akad", "resepsi"}})
	body := func(confirm any) map[string]any {
		m := map[string]any{"scenario_id": sc["id"], "mode": "clean_overwrite"}
		if confirm != nil {
			m["confirm"] = confirm
		}
		return m
	}

	wantError(t, e.do(tok, "POST", base(id)+"/scenarios/apply", body(nil)), 400, "invalid_input", "confirm: required")
	wantError(t, e.do(tok, "POST", base(id)+"/scenarios/apply", body("reset")), 400, "invalid_input", "confirm: must be RESET")
	if n := count(e, `SELECT count(*) FROM planner_budget_items WHERE project_id=$1`, id); n != 15 {
		t.Fatalf("a refused clean overwrite deleted lines: %d", n)
	}

	// The preview needs no confirmation and shows what would be lost: 15 lines and the payment.
	prev := mustStatus(t, e.do(tok, "POST", base(id)+"/scenarios/apply-preview", body(nil)), 200).data()
	if d := prev["deleted"].(map[string]any); num(d["lines"]) != 15 || num(d["paid_total"]) != 500_000 || len(list(prev["created"])) != 10 || num(prev["estimate_after"]) != 75_000_000 {
		t.Errorf("preview = %v", prev)
	}

	r := mustStatus(t, e.do(tok, "POST", base(id)+"/scenarios/apply", body("RESET")), 200).data()
	if d := r["deleted"].(map[string]any); num(d["lines"]) != 15 || num(d["paid_total"]) != 500_000 || len(list(r["created"])) != 10 || len(list(r["updated"])) != 0 || num(r["untouched_count"]) != 0 {
		t.Errorf("report = %v", r)
	}
	after := lines(e, tok, id)
	if len(after) != 10 || num(after["resepsi/catering"]["estimate"]) != 30_000_000 || num(after["resepsi/catering"]["paid"]) != 0 || after["resepsi/catering"]["source"] != "scenario" {
		t.Errorf("lines after clean overwrite: %d", len(after))
	}
}

func TestApplyStaysUnderTheLineLimit(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := templateProject(e, tok)
	exec(e, `INSERT INTO planner_budget_items (id, project_id, sub_event_scope, category, item_name, source, sort_order)
	         SELECT gen_random_uuid(), $1, 'all', 'x', 'pos ' || n, 'manual', 100 + n FROM generate_series(1, 286) n`, id)
	sc := createScenario(e, tok, id, map[string]any{"name": "Ideal", "preset_key": "wedding_75jt"})

	// 300 lines already: the one new line (dana_darurat) would be the 301st, so nothing at all is applied.
	wantError(t, e.do(tok, "POST", base(id)+"/scenarios/apply", map[string]any{"scenario_id": sc["id"], "mode": "smart_adjust"}), 409, "budget_item_limit_reached", "")
	if n := count(e, `SELECT count(*) FROM planner_budget_items WHERE project_id=$1`, id); n != 300 {
		t.Errorf("%d lines", n)
	}
	if n := count(e, `SELECT count(*) FROM planner_budget_items WHERE project_id=$1 AND initial_estimate>0`, id); n != 0 {
		t.Errorf("a refused application still updated %d estimates", n)
	}
	if count(e, `SELECT count(*) FROM planner_budget_scenarios WHERE project_id=$1 AND is_active`, id) != 0 {
		t.Error("a refused application still activated the scenario")
	}
}
