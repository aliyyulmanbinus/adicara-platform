package http_test

import (
	"fmt"
	"testing"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerfinance/domain"
)

const universalBudget = 50_000_000

// metadataFor fills the required metadata fields of an event type, whatever they are.
func metadataFor(spec pd.EventTypeSpec) map[string]any {
	m := map[string]any{}
	for _, f := range spec.MetadataFields {
		switch {
		case !f.Required:
		case f.Type == pd.FieldInt && f.Min != nil:
			m[f.Key] = *f.Min
		case f.Type == pd.FieldInt:
			m[f.Key] = 1
		case f.Type == pd.FieldDate:
			m[f.Key] = inDays(30)
		default:
			m[f.Key] = "Uji"
		}
	}

	return m
}

func sumEstimates(ls map[string]map[string]any) (sum int64) {
	for _, l := range ls {
		sum += num(l["estimate"])
	}

	return sum
}

// Menu 03 is not a wedding feature: every event type of the registry goes through the whole
// flow (wizard template, preset, apply, contributors with their own roles, deposits, payment,
// cashflow, dashboard, export) on the same endpoints, with nothing but the registry to tell them apart.
func TestEveryEventTypeRunsTheWholeFinanceFlow(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	defer func() { e.t = t }()

	for _, spec := range pd.DefaultRegistry().All() {
		t.Run(string(spec.Key), func(t *testing.T) {
			e.t = t
			id := createProject(e, tok, map[string]any{
				"event_type": string(spec.Key), "title": "Uji " + spec.Label, "event_date": inDays(120), "target_budget": universalBudget,
				"metadata": metadataFor(spec), "templates": map[string]any{"budget_lines": true},
			})

			templateLinesOf(t, e, tok, id, spec)
			applied := applyFirstPreset(t, e, tok, id, spec)
			collected := contributeAndSave(t, e, tok, id, spec)
			paymentAndViews(t, e, tok, id, collected, applied)
			documentCostReachesTheBudget(t, e, tok, id)
		})
	}
}

// templateLinesOf: the wizard template is what the registry says; a percentage template spends the whole budget.
func templateLinesOf(t *testing.T, e *env, tok, id string, spec pd.EventTypeSpec) {
	t.Helper()
	want := domain.TemplateLines(spec.Key, universalBudget)
	ls := lines(e, tok, id)
	if len(ls) != len(want) || len(want) == 0 {
		t.Fatalf("%d lines, want %d", len(ls), len(want))
	}
	for _, l := range ls {
		if l["source"] != "template" {
			t.Errorf("line %v has source %v", l["item_name"], l["source"])
		}
	}
	wantSum := int64(universalBudget)
	if spec.Key == pd.EventWedding {
		wantSum = 0
	}
	if got := sumEstimates(ls); got != wantSum {
		t.Errorf("the template adds up to %d, want %d", got, wantSum)
	}
}

// applyFirstPreset creates a scenario from the first preset of the type and applies it. The scenario items pair
// with the template lines by key, so a percentage preset over the percentage template creates nothing new.
// It returns the lines after the application.
func applyFirstPreset(t *testing.T, e *env, tok, id string, spec pd.EventTypeSpec) map[string]map[string]any {
	t.Helper()
	before := len(lines(e, tok, id))
	presets := list(get(e, tok, base(id)+"/scenarios")["presets"])
	if len(presets) == 0 {
		t.Fatal("no preset offered")
	}
	body := map[string]any{"name": "Dari preset", "preset_key": presets[0]["key"]}
	if presets[0]["kind"] == "percent" {
		body["total_target"] = universalBudget
	}
	sc := createScenario(e, tok, id, body)
	report := mustStatus(t, e.do(tok, "POST", base(id)+"/scenarios/apply", map[string]any{"scenario_id": sc["id"], "mode": "smart_adjust"}), 200).data()

	after := lines(e, tok, id)
	sum := sumEstimates(after)
	if sum != num(report["estimate_after"]) {
		t.Errorf("Σ estimate = %d, the report says %d", sum, num(report["estimate_after"]))
	}
	if spec.Key != pd.EventWedding && (len(list(report["created"])) != 0 || len(after) != before || sum != universalBudget) {
		t.Errorf("percent preset over the percent template: created %v, %d lines (was %d), Σ %d", report["created"], len(after), before, sum)
	}
	if get(e, tok, base(id)+"/budget/matrix")["active_scenario"] == nil {
		t.Error("applying a scenario must make it the active one")
	}

	return after
}

// contributeAndSave: contributors take the roles of this type and no others; it returns what was saved.
func contributeAndSave(t *testing.T, e *env, tok, id string, spec pd.EventTypeSpec) int64 {
	t.Helper()
	for i, role := range spec.ContributorRoles {
		c := createContributor(e, tok, id, fmt.Sprintf("Kontributor %d", i), role.Key)
		if c["role_label"] != role.Key {
			t.Errorf("role %s came back as %v", role.Key, c["role_label"])
		}
		mustStatus(t, deposit(e, tok, id, c["id"].(string), today, 1_000_000), 201)
	}
	wantError(t, e.do(tok, "POST", base(id)+"/contributors", map[string]any{"name": "Salah", "role_label": "groom_xx"}),
		400, "invalid_input", "role_label: must be one of the event type roles")

	collected := int64(len(spec.ContributorRoles)) * 1_000_000
	sv := get(e, tok, base(id)+"/savings")
	kpi := sv["kpi"].(map[string]any)
	if num(kpi["collected"]) != collected || num(kpi["target"]) != universalBudget || num(kpi["shortfall"]) != universalBudget-collected {
		t.Errorf("savings kpi = %v", kpi)
	}
	if rec := sv["recommendation"].(map[string]any); num(rec["months_left"]) < 1 {
		t.Errorf("recommendation = %v", rec)
	}
	if len(list(sv["milestones"])) == 0 {
		t.Errorf("a dated project has at least one milestone: %v", sv["milestones"])
	}

	return collected
}

// paymentAndViews pays a line; the cashflow, the dashboard widget and the export follow.
func paymentAndViews(t *testing.T, e *env, tok, id string, collected int64, ls map[string]map[string]any) {
	t.Helper()
	var item string
	for _, l := range ls {
		if num(l["estimate"]) > 0 {
			item = l["id"].(string)

			break
		}
	}
	mustStatus(t, pay(e, tok, id, item, map[string]any{"payment_stage": "dp", "amount": 500_000, "paid_at": today}), 200)

	cash := get(e, tok, base(id)+"/budget/cashflow-summary")
	if num(cash["total_paid"]) != 500_000 || num(cash["available_cash"]) != collected-500_000 {
		t.Errorf("cashflow = %v", cash)
	}
	fin := get(e, tok, base(id)+"/dashboard")["finance"].(map[string]any)
	if num(fin["total_estimate"]) != sumEstimates(ls) || num(fin["total_paid"]) != 500_000 || num(fin["savings_collected"]) != collected {
		t.Errorf("dashboard finance = %v", fin)
	}
	r := mustStatus(t, e.do(tok, "GET", base(id)+"/budget/export?format=xlsx", nil), 200)
	if len(r.body) < 1000 || string(r.body[:2]) != "PK" {
		t.Errorf("export is %d bytes, not a workbook", len(r.body))
	}
}

// documentCostReachesTheBudget: the cost of a document of menu 02 becomes a line of this module in the general
// session, once, without touching what the user already planned.
func documentCostReachesTheBudget(t *testing.T, e *env, tok, id string) {
	t.Helper()
	before := lines(e, tok, id)
	doc := mustStatus(t, e.do(tok, "POST", base(id)+"/admin-docs", map[string]any{"document_name": "Izin keramaian", "nominal_cost": 750_000}), 201).data()["id"].(string)

	first := mustStatus(t, e.do(tok, "POST", base(id)+"/admin-docs/"+doc+"/sync-budget", nil), 201).data()
	again := mustStatus(t, e.do(tok, "POST", base(id)+"/admin-docs/"+doc+"/sync-budget", nil), 200).data()
	after := lines(e, tok, id)
	line := after["all/Izin keramaian"]
	if first["created"] != true || again["created"] != false || again["budget_item_id"] != first["budget_item_id"] || len(after) != len(before)+1 {
		t.Fatalf("sync = %v then %v, %d lines (was %d)", first, again, len(after), len(before))
	}
	if line["source"] != "admin_doc" || line["admin_doc_id"] != doc || num(line["estimate"]) != 750_000 || sumEstimates(after) != sumEstimates(before)+750_000 {
		t.Errorf("the line of the document = %v", line)
	}
}
