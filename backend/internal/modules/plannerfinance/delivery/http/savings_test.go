package http_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func createContributor(e *env, tok, id, name, role string) map[string]any {
	e.t.Helper()
	body := map[string]any{"name": name}
	if role != "" {
		body["role_label"] = role
	}

	return mustStatus(e.t, e.do(tok, "POST", base(id)+"/contributors", body), 201).data()
}

func deposit(e *env, tok, id, contributor, date string, amount int64) resp {
	e.t.Helper()

	return e.do(tok, "POST", base(id)+"/savings", map[string]any{"contributor_id": contributor, "entry_date": date, "amount": amount})
}

func TestContributors(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)

	if got := list(mustStatus(t, e.do(tok, "GET", base(id)+"/contributors", nil), 200).json()["data"]); len(got) != 0 {
		t.Fatalf("a new project has no contributor: %v", got)
	}
	arif := createContributor(e, tok, id, "  Arif  ", "groom")
	sinta := createContributor(e, tok, id, "Sinta", "")
	if arif["name"] != "Arif" || arif["role_label"] != "groom" || num(arif["total"]) != 0 || num(arif["entry_count"]) != 0 || sinta["role_label"] != "other" {
		t.Errorf("contributors = %v %v", arif, sinta)
	}

	wantError(t, e.do(tok, "POST", base(id)+"/contributors", map[string]any{"name": ""}), 400, "invalid_input", "name: must not be empty")
	wantError(t, e.do(tok, "POST", base(id)+"/contributors", map[string]any{"name": strings.Repeat("a", 101)}), 400, "invalid_input", "name: must be at most 100 characters")
	// The roles belong to the event type: "host" is not a wedding role.
	wantError(t, e.do(tok, "POST", base(id)+"/contributors", map[string]any{"name": "x", "role_label": "host"}), 400, "invalid_input", "role_label: must be one of the event type roles")

	// PUT replaces: a blank role becomes "other".
	up := mustStatus(t, e.do(tok, "PUT", base(id)+"/contributors/"+arif["id"].(string), map[string]any{"name": "Arif P."}), 200).data()
	if up["name"] != "Arif P." || up["role_label"] != "other" {
		t.Errorf("updated = %v", up)
	}
	wantError(t, e.do(tok, "PUT", base(id)+"/contributors/"+uuid.NewString(), map[string]any{"name": "x"}), 404, "not_found", "contributor not found")

	// A contributor with deposits cannot be deleted; one without can.
	mustStatus(t, deposit(e, tok, id, arif["id"].(string), "2026-10-01", 1_000_000), 201)
	wantError(t, e.do(tok, "DELETE", base(id)+"/contributors/"+arif["id"].(string), nil), 409, "contributor_in_use", "")
	mustStatus(t, e.do(tok, "DELETE", base(id)+"/contributors/"+sinta["id"].(string), nil), 204)
	wantError(t, e.do(tok, "DELETE", base(id)+"/contributors/"+sinta["id"].(string), nil), 404, "not_found", "contributor not found")
	if got := list(mustStatus(t, e.do(tok, "GET", base(id)+"/contributors", nil), 200).json()["data"]); len(got) != 1 || num(got[0]["total"]) != 1_000_000 || num(got[0]["entry_count"]) != 1 {
		t.Errorf("contributors = %v", got)
	}

	// A seminar has other roles.
	sem := createProject(e, tok, map[string]any{"event_type": "seminar", "metadata": map[string]any{"organizer_name": "BEM"}})
	wantError(t, e.do(tok, "POST", base(sem)+"/contributors", map[string]any{"name": "x", "role_label": "groom"}), 400, "invalid_input", "role_label: must be one of the event type roles")
	if c := createContributor(e, tok, sem, "Panitia", "internal"); c["role_label"] != "internal" {
		t.Errorf("seminar contributor = %v", c)
	}
}

func TestContributorLimit(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	exec(e, `INSERT INTO planner_contributors (id, project_id, name, sort_order) SELECT gen_random_uuid(), $1, 'k' || n, n FROM generate_series(1, 20) n`, id)

	wantError(t, e.do(tok, "POST", base(id)+"/contributors", map[string]any{"name": "ke-21"}), 409, "contributor_limit_reached", "")
	for _, s := range concurrently(4, func(int) int { return e.do(tok, "POST", base(id)+"/contributors", map[string]any{"name": "x"}).status }) {
		if s != 409 {
			t.Fatalf("a full project accepted another contributor")
		}
	}
}

func TestSavingsEntries(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	arif := createContributor(e, tok, id, "Arif", "groom")["id"].(string)
	sinta := createContributor(e, tok, id, "Sinta", "bride")["id"].(string)

	r := mustStatus(t, e.do(tok, "POST", base(id)+"/savings", map[string]any{"contributor_id": arif, "entry_date": "2026-09-30", "amount": 2_500_000, "notes": "  gaji  "}), 201).data()
	if r["contributor_name"] != "Arif" || r["role_label"] != "groom" || r["entry_date"] != "2026-09-30" || num(r["amount"]) != 2_500_000 || r["notes"] != "gaji" {
		t.Errorf("entry = %v", r)
	}
	if r2 := mustStatus(t, deposit(e, tok, id, sinta, "2026-10-01", 500_000), 201).data(); r2["notes"] != nil {
		t.Errorf("an entry without notes must not send the key: %v", r2)
	}

	wantError(t, deposit(e, tok, id, arif, "2026-10-01", 0), 400, "invalid_input", "amount: must be greater than 0")
	wantError(t, deposit(e, tok, id, arif, "2026-10-01", -5), 400, "invalid_input", "amount: must be greater than 0")
	wantError(t, deposit(e, tok, id, arif, "2026-10-01", 10_000_000_000_001), 400, "invalid_input", "amount: must be between 0 and 10000000000000")
	wantError(t, deposit(e, tok, id, arif, "01-10-2026", 1), 400, "invalid_date", "entry_date: must be a valid date in YYYY-MM-DD format")
	wantError(t, deposit(e, tok, id, arif, "2010-01-01", 1), 400, "invalid_date", "entry_date: must be within the project schedule window")
	wantError(t, deposit(e, tok, id, arif, "2099-01-01", 1), 400, "invalid_date", "entry_date: must be within the project schedule window")
	wantError(t, deposit(e, tok, id, uuid.NewString(), "2026-10-01", 1), 400, "invalid_input", "contributor_id: must be a contributor of this project")
	// A future date is allowed (and counts as collected, 03 BE chapter 9 item 7).
	mustStatus(t, deposit(e, tok, id, arif, "2026-12-01", 1_000_000), 201)

	// A contributor of another project cannot receive a deposit.
	other := createProject(e, tok, nil)
	foreign := createContributor(e, tok, other, "Asing", "groom")["id"].(string)
	wantError(t, deposit(e, tok, id, foreign, "2026-10-01", 1), 400, "invalid_input", "contributor_id: must be a contributor of this project")

	// PUT corrects a deposit and may move it to another contributor.
	entry := r["id"].(string)
	up := mustStatus(t, e.do(tok, "PUT", base(id)+"/savings/"+entry, map[string]any{"contributor_id": sinta, "entry_date": "2026-09-29", "amount": 3_000_000}), 200).data()
	if up["contributor_name"] != "Sinta" || num(up["amount"]) != 3_000_000 || up["notes"] != nil {
		t.Errorf("updated = %v", up)
	}
	wantError(t, e.do(tok, "PUT", base(id)+"/savings/"+uuid.NewString(), map[string]any{"contributor_id": sinta, "entry_date": "2026-09-29", "amount": 1}), 404, "not_found", "deposit not found")
	mustStatus(t, e.do(tok, "DELETE", base(id)+"/savings/"+entry, nil), 204)
	wantError(t, e.do(tok, "DELETE", base(id)+"/savings/"+entry, nil), 404, "not_found", "deposit not found")

	// The filter narrows the deposits and nothing else.
	all := get(e, tok, base(id)+"/savings")
	filtered := get(e, tok, base(id)+"/savings?contributor_id="+arif)
	if len(list(all["entries"])) != 2 || len(list(filtered["entries"])) != 1 {
		t.Errorf("entries: %d all, %d filtered", len(list(all["entries"])), len(list(filtered["entries"])))
	}
	for _, key := range []string{"kpi", "contributors", "milestones", "recommendation", "matrix", "unscheduled_estimate"} {
		if jsonOf(all[key]) != jsonOf(filtered[key]) {
			t.Errorf("%s changed with the filter:\n%s\n%s", key, jsonOf(all[key]), jsonOf(filtered[key]))
		}
	}
	wantError(t, e.do(tok, "GET", base(id)+"/savings?contributor_id=nope", nil), 400, "invalid_input", "contributor_id: must be a valid id")
}

func TestSavingsEntryLimit(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	c := createContributor(e, tok, id, "Arif", "groom")["id"].(string)
	exec(e, `INSERT INTO planner_savings_entries (id, project_id, contributor_id, entry_date, amount)
	         SELECT gen_random_uuid(), $1, $2, DATE '2026-10-01', 1000 FROM generate_series(1, 2000)`, id, c)

	wantError(t, deposit(e, tok, id, c, "2026-10-02", 1), 409, "savings_entry_limit_reached", "")
	for _, s := range concurrently(4, func(int) int { return deposit(e, tok, id, c, "2026-10-02", 1).status }) {
		if s != 409 {
			t.Fatalf("a full project accepted another deposit")
		}
	}
}

func TestSavingsWithoutContributorsHasNoPerContributor(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)

	sv := get(e, tok, base(id)+"/savings")
	rec := sv["recommendation"].(map[string]any)
	if _, has := rec["per_contributor"]; has {
		t.Errorf("per_contributor without contributors: %v", rec)
	}
	// Nothing collected: the whole target, in 6 months to the last session (175 days).
	if num(rec["recommended_monthly"]) != 12_500_000 || num(rec["months_left"]) != 6 || rec["basis"] != "session" || rec["target_date"] != inDays(175) {
		t.Errorf("recommendation = %v", rec)
	}
	if len(list(sv["contributors"])) != 0 || len(list(sv["entries"])) != 0 || len(sv["matrix"].(map[string]any)["months"].([]any)) != 0 {
		t.Errorf("a new project must have empty lists: %v", sv)
	}
	// The sessions are dated at Hari-H but have no lines yet: milestones with nothing to fund.
	if ms := list(sv["milestones"]); len(ms) != 2 || num(ms[0]["target_amount"]) != 0 || ms[0]["is_secured"] != true {
		t.Errorf("milestones = %v", ms)
	}

	// Without any dated session the target date is Hari-H itself.
	exec(e, `UPDATE planner_event_sessions SET session_date = NULL WHERE project_id = $1`, id)
	sv = get(e, tok, base(id)+"/savings")
	rec = sv["recommendation"].(map[string]any)
	if rec["basis"] != "event_date" || rec["target_date"] != inDays(175) || len(list(sv["milestones"])) != 0 {
		t.Errorf("recommendation without dated sessions = %v", rec)
	}
	if kpi := sv["kpi"].(map[string]any); num(kpi["target"]) != 75_000_000 || num(kpi["collected"]) != 0 || num(kpi["shortfall"]) != 75_000_000 {
		t.Errorf("kpi = %v", kpi)
	}
}

func TestSavingsMatrixHasNoGapsAndMilestonesFollowSessionDates(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := templateProject(e, tok)
	a := createContributor(e, tok, id, "Arif", "groom")["id"].(string)
	b := createContributor(e, tok, id, "Sinta", "bride")["id"].(string)
	mustStatus(t, deposit(e, tok, id, a, "2026-02-15", 1_000_000), 201)
	mustStatus(t, deposit(e, tok, id, a, "2026-02-20", 500_000), 201)
	mustStatus(t, deposit(e, tok, id, b, "2026-05-01", 2_000_000), 201)

	mx := get(e, tok, base(id)+"/savings")["matrix"].(map[string]any)
	months := mx["months"].([]any)
	if len(months) != 4 || months[0] != "2026-02" || months[1] != "2026-03" || months[2] != "2026-04" || months[3] != "2026-05" {
		t.Fatalf("months = %v (March and April have no deposit but must be there)", months)
	}
	rows := list(mx["rows"])
	if num(rows[0]["total"]) != 1_500_000 || num(rows[1]["total"]) != 0 || num(rows[3]["total"]) != 2_000_000 {
		t.Errorf("rows = %v", rows)
	}
	for _, r := range rows {
		if len(r["by_contributor"].(map[string]any)) != 2 {
			t.Errorf("every contributor must be in every month: %v", r)
		}
	}

	// Milestones are the dated sessions in date order; lines in an undated session are "unscheduled".
	lamaran := ""
	for _, s := range sessionsOf(e, tok, id) {
		if s["scope_key"] == "lamaran" {
			lamaran = s["id"].(string)
		}
	}
	mustStatus(t, e.do(tok, "PUT", base(id)+"/budget/items/"+lines(e, tok, id)["lamaran/catering"]["id"].(string), map[string]any{
		"sub_event_scope": "lamaran", "category": "Catering", "item_name": "Catering", "initial_estimate": 2_000_000,
	}), 200)
	sv := get(e, tok, base(id)+"/savings")
	if len(list(sv["milestones"])) != 2 || num(sv["unscheduled_estimate"]) != 2_000_000 {
		t.Errorf("milestones %v, unscheduled %v: Lamaran has no date yet", sv["milestones"], sv["unscheduled_estimate"])
	}
	mustStatus(t, e.do(tok, "PUT", base(id)+"/sessions/"+lamaran, map[string]any{"name": "Lamaran", "session_date": "2026-11-15"}), 200)
	sv = get(e, tok, base(id)+"/savings")
	ms := list(sv["milestones"])
	if len(ms) != 3 || ms[0]["sub_event_scope"] != "lamaran" || ms[0]["session_date"] != "2026-11-15" || num(ms[0]["target_amount"]) != 2_000_000 ||
		num(ms[0]["secured_amount"]) != 2_000_000 || ms[0]["is_secured"] != true || num(sv["unscheduled_estimate"]) != 0 {
		t.Errorf("milestones = %v", ms)
	}
	// Collected 3.500.000: Lamaran (2.000.000) is secured, the next dated session (all estimates 0) too.
	if rec := sv["recommendation"].(map[string]any); rec["basis"] != "session" || rec["target_date"] != inDays(175) {
		t.Errorf("recommendation = %v", rec)
	}
}

func jsonOf(v any) string {
	b, _ := json.Marshal(v)

	return string(b)
}
