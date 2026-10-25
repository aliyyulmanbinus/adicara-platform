package http_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

type route struct {
	method, path string
	body         any
}

// guestRoutes lists the 19 routes of the module with a body that is valid for a project that has the guest g (so a 404
// can only come from the guard).
func guestRoutes(g, token string) []route {
	return []route{
		{"GET", "/guests", nil},
		{"POST", "/guests", guestBody("x", nil)},
		{"GET", "/guests/summary", nil},
		{"GET", "/guests/catering-projection", nil},
		{"GET", "/guests/export?format=xlsx", nil},
		{"GET", "/guests/import-template?format=xlsx", nil},
		{"POST", "/guests/bulk-import", importBody("a.csv", []byte("Nama Tamu\nx\n"))},
		{"GET", "/guests/reception-overview", nil},
		{"GET", "/guests/export-attendance?format=xlsx", nil},
		{"POST", "/guests/scan-qr", map[string]any{"qr_code_token": token}},
		{"POST", "/guests/walk-in", map[string]any{"name": "x", "pax_actual": 1}},
		{"GET", "/guests/" + g, nil},
		{"PUT", "/guests/" + g, guestBody("x", nil)},
		{"DELETE", "/guests/" + g, nil},
		{"PATCH", "/guests/" + g + "/sent-status", map[string]any{"sent_status": "sent"}},
		{"PATCH", "/guests/" + g + "/rsvp", map[string]any{"rsvp_status": "declined"}},
		{"PATCH", "/guests/" + g + "/souvenir", map[string]any{"souvenir_received": true}},
		{"POST", "/guests/" + g + "/checkin", map[string]any{"pax_actual": 1}},
		{"DELETE", "/guests/" + g + "/checkin", nil},
	}
}

func TestEveryGuestRouteNeedsABearerToken(t *testing.T) {
	e := newEnv(t)
	id := uuid.NewString()

	routes := guestRoutes(uuid.NewString(), "t")
	if len(routes) != 19 {
		t.Fatalf("the module has 19 routes, the test lists %d", len(routes))
	}
	for _, rt := range routes {
		if r := e.do("", rt.method, base(id)+rt.path, rt.body); r.status != 401 || r.code() != "unauthorized" {
			t.Errorf("%s %s without a token: %d %s", rt.method, rt.path, r.status, r.body)
		}
	}
}

// B sees A's project as not existing, on every one of the 19 routes, and A's data is untouched.
func TestUserBGetsNotFoundOnEveryGuestEndpoint(t *testing.T) {
	e := newEnv(t)
	a, b := e.login("owner"), e.login("intruder")
	id := createSample(e, a)
	g := guestNamed(t, e, a, id, "Fajar Nugraha")
	before := jsonOf([]any{guestList(e, a, id, "limit=500"), reception(e, a, id, "")})

	for _, rt := range guestRoutes(g["id"].(string), g["qr_code_token"].(string)) {
		wantError(t, e.do(b, rt.method, base(id)+rt.path, rt.body), 404, "not_found", "project not found")
	}

	// B's own project does not know A's guest or A's ticket either.
	own := createProject(e, b, nil)
	for _, rt := range []route{
		{"GET", "/guests/" + g["id"].(string), nil},
		{"PUT", "/guests/" + g["id"].(string), guestBody("x", nil)},
		{"DELETE", "/guests/" + g["id"].(string), nil},
		{"PATCH", "/guests/" + g["id"].(string) + "/sent-status", map[string]any{"sent_status": "sent"}},
		{"PATCH", "/guests/" + g["id"].(string) + "/rsvp", map[string]any{"rsvp_status": "declined"}},
		{"PATCH", "/guests/" + g["id"].(string) + "/souvenir", map[string]any{"souvenir_received": true}},
		{"POST", "/guests/" + g["id"].(string) + "/checkin", map[string]any{"pax_actual": 1}},
		{"DELETE", "/guests/" + g["id"].(string) + "/checkin", nil},
		{"POST", "/guests/scan-qr", map[string]any{"qr_code_token": g["qr_code_token"]}},
	} {
		if r := e.do(b, rt.method, base(own)+rt.path, rt.body); r.status != 404 {
			t.Errorf("%s %s with another project's id: %d %s", rt.method, rt.path, r.status, r.body)
		}
	}

	// ids that are not canonical UUIDs are 404 as well, never a query
	for _, bad := range []string{"nope", "{" + g["id"].(string) + "}", strings.ToUpper(strings.ReplaceAll(g["id"].(string), "-", "")), "summary2"} {
		wantError(t, e.do(a, "GET", guests(id)+"/"+bad, nil), 404, "not_found", "guest not found")
		wantError(t, e.do(a, "POST", guests(id)+"/"+bad+"/checkin", map[string]any{"pax_actual": 1}), 404, "not_found", "guest not found")
		wantError(t, e.do(a, "DELETE", guests(id)+"/"+bad, nil), 404, "not_found", "guest not found")
	}

	if after := jsonOf([]any{guestList(e, a, id, "limit=500"), reception(e, a, id, "")}); after != before {
		t.Error("the intruder changed something")
	}
	if num(guestOverview(e, b, own)["total"]) != 0 {
		t.Error("B's project holds something of A's")
	}
}

func TestArchivedProjectIsReadOnlyButStillReadable(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)
	g := guestNamed(t, e, tok, id, "Fajar Nugraha")
	before := jsonOf([]any{guestList(e, tok, id, "limit=500"), reception(e, tok, id, "")})
	mustStatus(t, e.do(tok, "PATCH", base(id)+"/status", map[string]any{"status": "archived"}), 200)

	for _, rt := range guestRoutes(g["id"].(string), g["qr_code_token"].(string)) {
		r := e.do(tok, rt.method, base(id)+rt.path, rt.body)
		if rt.method == "GET" { // the list, the summary, the exports and the template stay available
			if r.status != 200 {
				t.Errorf("GET %s on an archived project: %d %s", rt.path, r.status, r.body)
			}

			continue
		}
		wantError(t, r, 409, "project_read_only", "")
	}
	if after := jsonOf([]any{guestList(e, tok, id, "limit=500"), reception(e, tok, id, "")}); after != before {
		t.Error("an archived project changed")
	}
}

func TestGuestEndpointsRejectMalformedBodies(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	g := addGuest(e, tok, id, guestBody("x", nil))["id"].(string)

	for _, rt := range []route{
		{"POST", "/guests", "{not json"}, {"POST", "/guests", `[]`}, {"POST", "/guests", `{"name": 7}`}, {"POST", "/guests", `{"name": "x", "pax_count": "2"}`},
		{"POST", "/guests", `{"name": "x", "pax_count": 2.5}`}, {"PUT", "/guests/" + g, `{"guest_side": 1}`}, {"PUT", "/guests/" + g, `{"notes": ["a"]}`},
		{"PATCH", "/guests/" + g + "/sent-status", `{"sent_status": 1}`}, {"PATCH", "/guests/" + g + "/rsvp", `{"rsvp_status": ["x"]}`},
		{"PATCH", "/guests/" + g + "/rsvp", `{"rsvp_status": "attending", "pax_actual": "2"}`}, {"PATCH", "/guests/" + g + "/souvenir", `{"souvenir_received": "ya"}`},
		{"POST", "/guests/scan-qr", `{"qr_code_token": 5}`}, {"POST", "/guests/walk-in", `{"name": "x", "pax_actual": "1"}`},
		{"POST", "/guests/" + g + "/checkin", `{"pax_actual": true}`}, {"POST", "/guests/bulk-import", `{"filename": ["a"]}`},
	} {
		wantError(t, e.do(tok, rt.method, base(id)+rt.path, rt.body), 400, "invalid_json", "invalid JSON body")
	}
	big := `{"name": "` + strings.Repeat("a", 70<<10) + `"}`
	wantError(t, e.do(tok, "POST", guests(id), big), 400, "invalid_json", "invalid JSON body")
	// Unknown keys are ignored, so a client that sends more than it should (a whole guest back on PUT) still works.
	mustStatus(t, e.do(tok, "PUT", guests(id)+"/"+g, `{"name": "y", "pax_count": 1, "rsvp_status": "declined", "is_checked_in": true, "qr_code_token": "x"}`), 200)
	if got := get(e, tok, guests(id)+"/"+g); got["rsvp_status"] != "unconfirmed" || got["is_checked_in"] != false || got["qr_code_token"] == "x" {
		t.Errorf("PUT accepted a field it does not own: %v", got)
	}
}
