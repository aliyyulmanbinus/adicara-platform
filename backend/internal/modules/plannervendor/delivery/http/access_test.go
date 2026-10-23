package http_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

// vendorRoutes lists the 17 routes of the module with a body that is valid for a project that has the ids in ids (so a 404
// can only come from the guard).
type route struct {
	method, path string
	body         any
}

func vendorRoutes(ids map[string]string) []route {
	v, it, line := ids["vendor"], ids["item"], ids["line"]

	return []route{
		{"GET", "/vendors", nil},
		{"POST", "/vendors", vendorBody("mua", "x", nil)},
		{"GET", "/vendors/export-contacts?format=xlsx", nil},
		{"GET", "/vendors/" + v, nil},
		{"PUT", "/vendors/" + v, vendorBody("mua", "x", nil)},
		{"PATCH", "/vendors/" + v + "/status", map[string]any{"status": "negosiasi"}},
		{"POST", "/vendors/" + v + "/lock-deal", map[string]any{"agreed_price": 1}},
		{"DELETE", "/vendors/" + v, nil},

		{"GET", "/shopping-items", nil},
		{"POST", "/shopping-items", itemBody("x", "y", 1, nil)},
		{"POST", "/shopping-items/apply-template", map[string]any{}},
		{"POST", "/shopping-items/link-to-budget", map[string]any{"budget_item_id": line}},
		{"POST", "/shopping-items/unlink-from-budget", map[string]any{"budget_item_id": line}},
		{"GET", "/shopping-items/export-shopping-list?format=xlsx", nil},
		{"PUT", "/shopping-items/" + it, itemBody("x", "y", 1, nil)},
		{"PATCH", "/shopping-items/" + it + "/status", map[string]any{"status": "proses"}},
		{"DELETE", "/shopping-items/" + it, nil},
	}
}

func TestEveryVendorRouteNeedsABearerToken(t *testing.T) {
	e := newEnv(t)
	id := uuid.NewString()
	ids := map[string]string{"vendor": uuid.NewString(), "item": uuid.NewString(), "line": uuid.NewString()}

	routes := vendorRoutes(ids)
	if len(routes) != 17 {
		t.Fatalf("the module has 17 routes, the test lists %d", len(routes))
	}
	for _, rt := range routes {
		if r := e.do("", rt.method, base(id)+rt.path, rt.body); r.status != 401 || r.code() != "unauthorized" {
			t.Errorf("%s %s without a token: %d %s", rt.method, rt.path, r.status, r.body)
		}
	}
}

// B sees A's project as not existing, on every one of the 17 routes, and A's data is untouched.
func TestUserBGetsNotFoundOnEveryVendorEndpoint(t *testing.T) {
	e := newEnv(t)
	a, b := e.login("owner"), e.login("intruder")
	id := createSample(e, a)

	ov := vendorOverview(e, a, id)
	sh := shoppingOverview(e, a, id)
	ids := map[string]string{
		"vendor": vendorOf(t, ov, "Katering Bu Minah (500 pax)")["id"].(string), "item": list(sh["items"])[0]["id"].(string),
		"line": sh["default_target"].(map[string]any)["budget_item_id"].(string),
	}
	before := jsonOf([]any{ov, sh, budgetLines(e, a, id)})

	for _, rt := range vendorRoutes(ids) {
		wantError(t, e.do(b, rt.method, base(id)+rt.path, rt.body), 404, "not_found", "project not found")
	}

	// B's own project does not know A's ids either
	own := createProject(e, b, nil)
	for _, rt := range []route{
		{"GET", "/vendors/" + ids["vendor"], nil},
		{"PUT", "/vendors/" + ids["vendor"], vendorBody("mua", "x", nil)},
		{"PATCH", "/vendors/" + ids["vendor"] + "/status", map[string]any{"status": "batal"}},
		{"POST", "/vendors/" + ids["vendor"] + "/lock-deal", map[string]any{"agreed_price": 1, "sync_to_budget_items": true}},
		{"DELETE", "/vendors/" + ids["vendor"], nil},
		{"PUT", "/shopping-items/" + ids["item"], itemBody("x", "y", 1, nil)},
		{"PATCH", "/shopping-items/" + ids["item"] + "/status", map[string]any{"status": "selesai"}},
		{"DELETE", "/shopping-items/" + ids["item"], nil},
	} {
		if r := e.do(b, rt.method, base(own)+rt.path, rt.body); r.status != 404 {
			t.Errorf("%s %s with another project's id: %d %s", rt.method, rt.path, r.status, r.body)
		}
	}
	wantError(t, e.do(b, "POST", base(own)+"/shopping-items", itemBody("x", "y", 1, map[string]any{"vendor_id": ids["vendor"]})), 400, "invalid_input", "vendor_id: must be a vendor of this project")
	wantError(t, e.do(b, "POST", base(own)+"/shopping-items", itemBody("x", "y", 1, map[string]any{"budget_item_id": ids["line"]})), 400, "invalid_input", "budget_item_id: must be a shopping rollup line of this project")
	wantError(t, linkToBudget(e, b, own, map[string]any{"budget_item_id": ids["line"]}), 400, "invalid_input", "budget_item_id: must be a budget line of this project")
	wantError(t, linkToBudget(e, b, own, map[string]any{"item_ids": []string{ids["item"]}}), 400, "invalid_input", "item_ids: must be items of this project")
	wantError(t, unlinkFromBudget(e, b, own, ids["line"]), 400, "invalid_input", "budget_item_id: must be a budget line of this project")

	// ids that are not canonical UUIDs are 404 as well, never a query
	for _, bad := range []string{"nope", "{" + ids["vendor"] + "}", strings.ToUpper(strings.ReplaceAll(ids["vendor"], "-", ""))} {
		wantError(t, e.do(a, "GET", base(id)+"/vendors/"+bad, nil), 404, "not_found", "vendor not found")
		wantError(t, e.do(a, "POST", base(id)+"/vendors/"+bad+"/lock-deal", map[string]any{"agreed_price": 1}), 404, "not_found", "vendor not found")
		wantError(t, e.do(a, "DELETE", base(id)+"/shopping-items/"+bad, nil), 404, "not_found", "shopping item not found")
	}

	if after := jsonOf([]any{vendorOverview(e, a, id), shoppingOverview(e, a, id), budgetLines(e, a, id)}); after != before {
		t.Error("the intruder changed something")
	}
	if len(list(vendorOverview(e, b, own)["vendors"])) != 0 || len(list(shoppingOverview(e, b, own)["items"])) != 0 {
		t.Error("B's project holds something of A's")
	}
}

func TestArchivedProjectIsReadOnlyButStillReadable(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)
	sh := shoppingOverview(e, tok, id)
	ids := map[string]string{
		"vendor": vendorOf(t, vendorOverview(e, tok, id), "Studio Cahaya")["id"].(string), "item": list(sh["items"])[0]["id"].(string),
		"line": sh["default_target"].(map[string]any)["budget_item_id"].(string),
	}
	before := jsonOf([]any{vendorOverview(e, tok, id), shoppingOverview(e, tok, id)})
	mustStatus(t, e.do(tok, "PATCH", base(id)+"/status", map[string]any{"status": "archived"}), 200)

	for _, rt := range vendorRoutes(ids) {
		r := e.do(tok, rt.method, base(id)+rt.path, rt.body)
		if rt.method == "GET" {
			if r.status != 200 {
				t.Errorf("GET %s on an archived project: %d %s", rt.path, r.status, r.body)
			}

			continue
		}
		wantError(t, r, 409, "project_read_only", "")
	}
	if after := jsonOf([]any{vendorOverview(e, tok, id), shoppingOverview(e, tok, id)}); after != before {
		t.Error("an archived project changed")
	}
}

func TestVendorEndpointsRejectMalformedBodies(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	v := addVendor(e, tok, id, vendorBody("mua", "x", nil))["id"].(string)

	for _, rt := range []route{
		{"POST", "/vendors", "{not json"}, {"POST", "/vendors", `[]`}, {"PUT", "/vendors/" + v, `{"vendor_name": 7}`},
		{"PATCH", "/vendors/" + v + "/status", `{"status": 1}`}, {"POST", "/vendors/" + v + "/lock-deal", `{"agreed_price": "x"}`},
		{"POST", "/shopping-items", `{"estimated_price": 1.5}`}, {"POST", "/shopping-items/apply-template", `{"confirm": "yes"}`},
		{"POST", "/shopping-items/link-to-budget", `{"item_ids": ["nope"]}`}, {"POST", "/shopping-items/unlink-from-budget", `{`},
	} {
		wantError(t, e.do(tok, rt.method, base(id)+rt.path, rt.body), 400, "invalid_json", "invalid JSON body")
	}
	big := `{"vendor_name": "` + strings.Repeat("a", 70<<10) + `"}`
	wantError(t, e.do(tok, "POST", base(id)+"/vendors", big), 400, "invalid_json", "invalid JSON body")
}
