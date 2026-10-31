package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-yaml"
	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/migrate"
)

// End-to-end through Build, the real composition root. The module tests under internal/modules/planner*/delivery/http
// wire the Planner modules by hand ("wired like internal/app/app.go"); nothing tied those copies to the original, so a
// provider, applier, overlay or mount that app.go forgot would have passed every one of them. These tests drive the
// handler Build returns.
//
// Like the other database tests they WIPE the database named by TEST_DATABASE_URL (the name must end in _test) and are
// skipped without it; run with `go test -p 1`.

const openapiPath = "../../docs/openapi.yaml"

type built struct {
	t     *testing.T
	h     http.Handler
	token string
}

func newBuilt(t *testing.T) *built {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if name := strings.TrimPrefix(u.Path, "/"); !strings.HasSuffix(name, "_test") {
		t.Fatalf("refusing to wipe database %q: its name must end in _test", name)
	}
	if err := migrate.Fresh(dsn); err != nil {
		t.Fatal(err)
	}

	t.Setenv("DATABASE_URL", dsn)
	t.Setenv("APP_ENV", "development")
	t.Setenv("MIGRATE_ON_START", "true")
	t.Setenv("AUTH_RATE_LIMIT", "0")
	gin.SetMode(gin.TestMode)
	a, err := Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(a.Pool.Close)

	b := &built{t: t, h: a.Handler}
	name := "u" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	b.mustStatus(b.do("", "POST", "/v1/auth/register", map[string]any{"username": name, "email": name + "@example.com", "password": "Password123"}), 201)
	login := b.mustStatus(b.do("", "POST", "/v1/auth/login", map[string]any{"username": name, "password": "Password123"}), 200)
	b.token, _ = login.at("tokens", "access_token").(string)
	if b.token == "" {
		t.Fatal("login returned no access token")
	}

	return b
}

type answer struct {
	status int
	body   map[string]any
}

// at walks nested objects; it returns nil when a key is missing.
func (a answer) at(path ...string) any {
	var cur any = a.body
	for _, k := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = m[k]
	}

	return cur
}

func (b *built) do(token, method, path string, body any) answer {
	b.t.Helper()
	var raw []byte
	if body != nil {
		raw, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	b.h.ServeHTTP(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)

	return answer{status: rec.Code, body: out}
}

func (b *built) mustStatus(a answer, want int) answer {
	b.t.Helper()
	if a.status != want {
		b.t.Fatalf("status %d, want %d: %v", a.status, want, a.body)
	}

	return a
}

func (b *built) withT(t *testing.T) *built {
	c := *b
	c.t = t

	return &c
}

func (b *built) get(path string) answer { return b.do(b.token, "GET", path, nil) }

func num(v any) int {
	f, _ := v.(float64)

	return int(f)
}

func objects(v any) []map[string]any {
	var out []map[string]any
	arr, _ := v.([]any)
	for _, x := range arr {
		if m, ok := x.(map[string]any); ok {
			out = append(out, m)
		}
	}

	return out
}

var pathParam = regexp.MustCompile(`\{[A-Za-z]+\}`)

// Every operation under /v1/planner in openapi.yaml must be mounted. A request without a token reaches BearerAuth only
// when the route exists, so 401 proves the mount and any 404 is a route (or a mux line in Build) that is missing.
func TestBuildMountsEveryDocumentedPlannerRoute(t *testing.T) {
	raw, err := os.ReadFile(openapiPath)
	if err != nil {
		t.Skipf("openapi.yaml not available: %v", err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	b := newBuilt(t)

	checked := 0
	for path, item := range doc["paths"].(map[string]any) {
		if !strings.HasPrefix(path, "/v1/planner") {
			continue
		}
		concrete := pathParam.ReplaceAllString(path, uuid.NewString())
		for method := range item.(map[string]any) {
			switch method {
			case "get", "post", "put", "patch", "delete":
			default:
				continue
			}
			checked++
			if got := b.do("", strings.ToUpper(method), concrete, map[string]any{}); got.status != 401 {
				t.Errorf("%s %s without a token: status %d, want 401 (route not mounted?)", strings.ToUpper(method), path, got.status)
			}
		}
	}
	if checked < 117 {
		t.Errorf("only %d planner operations found in openapi.yaml, expected at least 117", checked)
	}
}

// The wizard of every event type, asking for every template the registry offers, must leave a dashboard in which all six
// widgets exist and agree with the registry the modules read from. This is where a module that app.go registered under
// the wrong key, or forgot, shows up for one type only.
func TestBuildWizardFillsEverySummaryForEveryEventType(t *testing.T) {
	b := newBuilt(t)
	types := objects(b.mustStatus(b.get("/v1/planner/event-types"), 200).body["data"])
	if len(types) != 9 {
		t.Fatalf("event types: %d, want 9", len(types))
	}

	for _, et := range types {
		key, _ := et["key"].(string)
		t.Run(key, func(t *testing.T) {
			b := b.withT(t) // helpers call Fatalf on the test they hold, which must be the running subtest
			avail, _ := et["templates_available"].(map[string]any)
			for _, tpl := range []string{"rundown", "budget_lines"} {
				if avail[tpl] != true {
					t.Errorf("templates_available.%s is not offered for %s", tpl, key)
				}
			}
			templates := map[string]any{
				"admin_docs": avail["admin_docs"], "budget_lines": avail["budget_lines"], "shopping_list": avail["shopping_list"],
				"rundown": avail["rundown"], "playlist": avail["playlist"], "include_adicara_tasks": avail["include_adicara_tasks"],
			}
			if presets, _ := avail["timeline_presets"].([]any); len(presets) > 0 {
				templates["timeline_preset"] = presets[0]
			}
			metadata := map[string]any{}
			for _, f := range objects(et["metadata_fields"]) {
				if f["required"] == true {
					metadata[f["key"].(string)] = "Uji"
				}
			}
			created := b.mustStatus(b.do(b.token, "POST", "/v1/planner/projects", map[string]any{
				"event_type": key, "title": "Uji " + key, "event_date": "2030-03-20", "target_budget": 150000000,
				"metadata": metadata, "templates": templates,
			}), 201)
			id, _ := created.at("data", "id").(string)
			if id == "" {
				id, _ = created.at("data", "project", "id").(string)
			}
			base := "/v1/planner/projects/" + id

			d := b.mustStatus(b.get(base+"/dashboard"), 200)
			for _, w := range []string{"timeline", "admin_docs", "finance", "vendors", "guests", "rundown"} {
				if d.at("data", w) == nil {
					t.Errorf("dashboard widget %q is null: its provider is not registered in app.go", w)
				}
			}

			if num(d.at("data", "timeline", "total")) == 0 {
				t.Errorf("timeline widget has no tasks, the wizard applier did not run")
			}
			if wantDocs := avail["admin_docs"] == true; (num(d.at("data", "admin_docs", "total")) > 0) != wantDocs {
				t.Errorf("admin_docs total %d, but templates_available.admin_docs = %v", num(d.at("data", "admin_docs", "total")), wantDocs)
			}
			if got := num(d.at("data", "finance", "target_budget")); got != 150000000 {
				t.Errorf("finance widget target_budget %d, want 150000000", got)
			}
			if got, want := num(d.at("data", "vendors", "categories_total")), len(objects(et["vendor_categories"])); got != want {
				t.Errorf("vendors widget lists %d categories, the registry has %d", got, want)
			}
			if got, want := len(objects(d.at("data", "guests", "sides"))), len(objects(et["guest_sides"])); got != want {
				t.Errorf("guests widget has %d sides, the registry has %d", got, want)
			}
			if num(d.at("data", "rundown", "agendas")) == 0 {
				t.Errorf("rundown widget has no agendas, the rundown applier did not run")
			}
			if got, want := num(d.at("data", "rundown", "sessions")), len(objects(et["sessions"])); got != want {
				t.Errorf("rundown widget counts %d sessions, the registry has %d", got, want)
			}

			// The two lists that only some types own are filled exactly when the registry offers them.
			shopping := len(objects(b.mustStatus(b.get(base+"/shopping-items"), 200).at("data", "items")))
			if (shopping > 0) != (avail["shopping_list"] == true) {
				t.Errorf("shopping list has %d items, but templates_available.shopping_list = %v", shopping, avail["shopping_list"])
			}
			songs := len(objects(b.mustStatus(b.get(base+"/playlists"), 200).at("data", "items")))
			if (songs > 0) != (avail["playlist"] == true) {
				t.Errorf("playlist has %d songs, but templates_available.playlist = %v", songs, avail["playlist"])
			}
		})
	}
}

// The sample project is assembled by the overlays of five modules, in the order app.go registers them (finance before
// vendors, which link to the budget lines finance made). Its dashboard has to show the figures of Wedding_Planner.pdf.
func TestBuildSampleProjectShowsThePDFFigures(t *testing.T) {
	b := newBuilt(t)
	sample := b.do(b.token, "POST", "/v1/planner/projects/sample", nil)
	if sample.status != 200 && sample.status != 201 {
		t.Fatalf("sample: status %d: %v", sample.status, sample.body)
	}
	id, _ := sample.at("data", "id").(string)
	if id == "" {
		id, _ = sample.at("data", "project", "id").(string)
	}
	d := b.mustStatus(b.get("/v1/planner/projects/"+id+"/dashboard"), 200)

	for _, c := range []struct {
		widget, field string
		want          int
	}{
		{"finance", "total_estimate", 74150000}, {"finance", "total_paid", 33550000}, {"finance", "remaining", 40600000},
		{"timeline", "total", 50}, {"timeline", "completed", 15}, {"timeline", "in_progress", 29},
		{"admin_docs", "total", 15}, {"admin_docs", "completed", 9},
		{"vendors", "fixed_vendors", 7}, {"vendors", "categories_total", 10},
		{"guests", "invitations", 20}, {"guests", "total_pax", 43},
		{"rundown", "agendas", 33}, {"rundown", "songs", 30},
	} {
		if got := num(d.at("data", c.widget, c.field)); got != c.want {
			t.Errorf("sample %s.%s = %d, want %d", c.widget, c.field, got, c.want)
		}
	}

	// A second call returns the same project (idempotent).
	again := b.do(b.token, "POST", "/v1/planner/projects/sample", nil)
	if got, _ := again.at("data", "id").(string); got == "" {
		got, _ = again.at("data", "project", "id").(string)
		if got != id {
			t.Errorf("second sample call returned project %q, want %q", got, id)
		}
	} else if got != id {
		t.Errorf("second sample call returned project %q, want %q", got, id)
	}
}

// The seven routes that send a spreadsheet read ?format and set the download headers in one way (ginutil.XLSXFormat and
// SendXLSX): the parameter may be left out, xlsx is the default and the only format, and a workbook is never cached.
// openapi.yaml has to say the same, so a client generated from it does not send what the server does not need.
func TestBuildEverySpreadsheetRouteTakesFormatTheSameWay(t *testing.T) {
	routes := []string{
		"/tasks/export-spreadsheet", "/budget/export", "/vendors/export-contacts", "/shopping-items/export-shopping-list",
		"/guests/export", "/guests/export-attendance", "/guests/import-template",
	}
	b := newBuilt(t)
	sample := b.do(b.token, "POST", "/v1/planner/projects/sample", nil)
	id, _ := sample.at("data", "project", "id").(string)
	if id == "" {
		t.Fatalf("sample project: %d %v", sample.status, sample.body)
	}
	base := "/v1/planner/projects/" + id

	for _, route := range routes {
		for _, query := range []string{"", "?format=xlsx"} {
			req := httptest.NewRequest(http.MethodGet, base+route+query, nil)
			req.Header.Set("Authorization", "Bearer "+b.token)
			rec := httptest.NewRecorder()
			b.h.ServeHTTP(rec, req)
			if rec.Code != 200 {
				t.Errorf("GET %s%s: status %d, want 200: %s", route, query, rec.Code, rec.Body.String())
				continue
			}
			if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet") {
				t.Errorf("GET %s%s: Content-Type %q", route, query, got)
			}
			if got := rec.Header().Get("Content-Disposition"); !strings.HasPrefix(got, "attachment; filename=\"") || !strings.HasSuffix(got, ".xlsx\"") {
				t.Errorf("GET %s%s: Content-Disposition %q", route, query, got)
			}
			if got := rec.Header().Get("Cache-Control"); got != "no-store" {
				t.Errorf("GET %s%s: Cache-Control %q, want no-store", route, query, got)
			}
			if !bytes.HasPrefix(rec.Body.Bytes(), []byte("PK")) {
				t.Errorf("GET %s%s: the body is not a workbook", route, query)
			}
		}
		// One refusal per route: the three guest downloads share a limit of 10 a minute, and the spelling of the value
		// (XLSX, csv, ...) is covered by the unit test of ginutil.XLSXFormat.
		for _, query := range []string{"?format=pdf"} {
			got := b.get(base + route + query)
			if got.status != 400 || got.at("error", "code") != "invalid_input" || got.at("error", "message") != "format: must be xlsx" {
				t.Errorf("GET %s%s: %d %v, want 400 invalid_input \"format: must be xlsx\"", route, query, got.status, got.body)
			}
		}
	}

	raw, err := os.ReadFile(openapiPath)
	if err != nil {
		t.Skipf("openapi.yaml not available: %v", err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	for _, route := range routes {
		op, _ := doc["paths"].(map[string]any)["/v1/planner/projects/{projectId}"+route].(map[string]any)["get"].(map[string]any)
		found := false
		params, _ := op["parameters"].([]any)
		for _, p := range params {
			param, _ := p.(map[string]any)
			if param["name"] != "format" {
				continue
			}
			found = true
			schema, _ := param["schema"].(map[string]any)
			if param["required"] == true {
				t.Errorf("openapi %s: the format parameter is still required", route)
			}
			if enum, _ := schema["enum"].([]any); len(enum) != 1 || enum[0] != "xlsx" || schema["default"] != "xlsx" {
				t.Errorf("openapi %s: format must be enum [xlsx] with default xlsx, got %v", route, schema)
			}
		}
		if !found {
			t.Errorf("openapi %s: no format parameter", route)
		}
	}
}
