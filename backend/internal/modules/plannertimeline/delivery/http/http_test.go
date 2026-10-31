package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/config"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/db"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/ginserver"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/migrate"
	authrepo "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/auth/repository"
	authusecase "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/auth/usecase"
	plannerhttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/delivery/http"
	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	plannerrepo "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/repository"
	plannerusecase "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/usecase"
	timelinehttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannertimeline/delivery/http"
	tld "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannertimeline/domain"
	timelinerepo "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannertimeline/repository"
	timelineusecase "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannertimeline/usecase"
)

// End-to-end through the real routers, BearerAuth, both usecases and
// PostgreSQL, wired like internal/app/app.go. Like the planner tests it WIPES
// the database named by TEST_DATABASE_URL (the name must end in _test) and is
// skipped without it; run with `go test -p 1`.
//
// The clock is fixed at 4 Oct 2026, 12:00 in Jakarta: the day the sample project
// of PLANNER_SPEC was written down, so the golden dates in the tests are those of the spec.

const today = "2026-10-04"

var fixedNow = time.Date(2026, 10, 4, 5, 0, 0, 0, time.UTC)

type env struct {
	t      *testing.T
	srv    *httptest.Server
	auth   *authusecase.Service
	pool   *pgxpool.Pool
	tokens map[string]string
}

func newEnv(t *testing.T) *env {
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
	pool, err := db.Connect(context.Background(), dsn, 10)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	gin.SetMode(gin.TestMode)
	engine, err := ginserver.New(config.Config{CORSOrigins: []string{"http://localhost"}, TrustedProxies: []string{"127.0.0.1"}})
	if err != nil {
		t.Fatal(err)
	}
	auth := authusecase.New(authrepo.NewPostgresRepository(pool), "test-secret-0123456789-0123456789", time.Hour, time.Hour)

	registry := pd.DefaultRegistry()
	tx := plannerrepo.NewTxManager(pool)
	now := func() time.Time { return fixedNow }
	planner := plannerusecase.New(plannerrepo.NewPostgresRepository(pool), tx, registry, plannerusecase.Options{Now: now})
	timeline := timelineusecase.New(timelinerepo.NewPostgresRepository(pool), tx, registry, timelineusecase.Options{Now: now})
	planner.RegisterSummary(timeline.TimelineProvider(), timeline.AdminDocsProvider())
	planner.RegisterTemplateAppliers(timeline.TimelineApplier(), timeline.AdminDocsApplier())
	planner.RegisterSampleOverlays(timeline.TimelineOverlay(), timeline.AdminDocsOverlay())
	plannerhttp.RegisterRoutes(engine, auth, planner)
	timelinehttp.RegisterRoutes(engine, auth, planner, timeline)

	srv := httptest.NewServer(engine)
	t.Cleanup(srv.Close)

	return &env{t: t, srv: srv, auth: auth, pool: pool, tokens: map[string]string{}}
}

func (e *env) login(name string) string {
	e.t.Helper()
	if tok, ok := e.tokens[name]; ok {
		return tok
	}
	if _, err := e.auth.Register(context.Background(), name+"@example.com", "Password123", name); err != nil {
		e.t.Fatal(err)
	}
	_, pair, err := e.auth.Login(context.Background(), name, "Password123")
	if err != nil {
		e.t.Fatal(err)
	}
	e.tokens[name] = pair.AccessToken

	return pair.AccessToken
}

type resp struct {
	status int
	body   []byte
	header http.Header
}

func (r resp) json() map[string]any {
	var m map[string]any
	_ = json.Unmarshal(r.body, &m)

	return m
}

func (r resp) data() map[string]any {
	d, _ := r.json()["data"].(map[string]any)

	return d
}

func (r resp) code() string {
	e, _ := r.json()["error"].(map[string]any)
	c, _ := e["code"].(string)

	return c
}

func (r resp) message() string {
	e, _ := r.json()["error"].(map[string]any)
	m, _ := e["message"].(string)

	return m
}

func (e *env) do(token, method, path string, body any) resp {
	e.t.Helper()
	var rd io.Reader
	if body != nil {
		switch b := body.(type) {
		case string:
			rd = strings.NewReader(b)
		default:
			raw, _ := json.Marshal(b)
			rd = bytes.NewReader(raw)
		}
	}
	req, _ := http.NewRequest(method, e.srv.URL+path, rd)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)

	return resp{status: res.StatusCode, body: raw, header: res.Header}
}

func mustStatus(t *testing.T, r resp, want int) resp {
	t.Helper()
	if r.status != want {
		t.Fatalf("status = %d, want %d; body: %s", r.status, want, r.body)
	}

	return r
}

// wantError asserts a 4xx answer with a code and (when given) a message.
func wantError(t *testing.T, r resp, status int, code, message string) {
	t.Helper()
	if r.status != status || r.code() != code || (message != "" && r.message() != message) {
		t.Fatalf("got %d %s %q, want %d %s %q; body: %s", r.status, r.code(), r.message(), status, code, message, r.body)
	}
}

// inDays is the calendar date n days after the fixed "today".
func inDays(n int) string {
	d, _ := time.Parse(pd.DateLayout, today)

	return d.AddDate(0, 0, n).Format(pd.DateLayout)
}

func project(extra map[string]any) map[string]any {
	m := map[string]any{
		"event_type": "wedding", "title": "Pernikahan Sinta & Arif", "event_date": inDays(175), "target_budget": 75000000,
		"metadata": map[string]any{"bride_name": "Sinta", "groom_name": "Arif"},
	}
	for k, v := range extra {
		m[k] = v
	}

	return m
}

func createProject(e *env, tok string, extra map[string]any) string {
	e.t.Helper()
	r := mustStatus(e.t, e.do(tok, "POST", "/v1/planner/projects", project(extra)), 201)

	return r.data()["project"].(map[string]any)["id"].(string)
}

func base(id string) string { return "/v1/planner/projects/" + id }

func list(v any) []map[string]any {
	raw, _ := v.([]any)
	out := make([]map[string]any, len(raw))
	for i, x := range raw {
		out[i], _ = x.(map[string]any)
	}

	return out
}

func tasksOf(e *env, tok, id string) []map[string]any {
	e.t.Helper()

	return list(mustStatus(e.t, e.do(tok, "GET", base(id)+"/tasks", nil), 200).data()["tasks"])
}

func docsOf(e *env, tok, id string) []map[string]any {
	e.t.Helper()

	return list(mustStatus(e.t, e.do(tok, "GET", base(id)+"/admin-docs", nil), 200).data()["docs"])
}

func count(e *env, query string, args ...any) int {
	e.t.Helper()
	var n int
	if err := e.pool.QueryRow(context.Background(), query, args...).Scan(&n); err != nil {
		e.t.Fatal(err)
	}

	return n
}

func exec(e *env, query string, args ...any) {
	e.t.Helper()
	if _, err := e.pool.Exec(context.Background(), query, args...); err != nil {
		e.t.Fatal(err)
	}
}

func mustUUID(t *testing.T, v any) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(fmt.Sprint(v))
	if err != nil {
		t.Fatal(err)
	}

	return id
}

// ---------- access ----------

func TestEveryRouteNeedsABearerToken(t *testing.T) {
	e := newEnv(t)
	id, other := uuid.NewString(), uuid.NewString()

	for _, rt := range [][2]string{
		{"GET", "/tasks"}, {"POST", "/tasks"}, {"POST", "/tasks/reorder"}, {"POST", "/tasks/shift"}, {"POST", "/tasks/generate-defaults"},
		{"GET", "/tasks/export-spreadsheet"}, {"PATCH", "/tasks/" + other}, {"PATCH", "/tasks/" + other + "/status"}, {"DELETE", "/tasks/" + other},
		{"GET", "/admin-docs"}, {"POST", "/admin-docs"}, {"POST", "/admin-docs/reorder"}, {"POST", "/admin-docs/apply-template"},
		{"PATCH", "/admin-docs/" + other}, {"PATCH", "/admin-docs/" + other + "/status"}, {"DELETE", "/admin-docs/" + other},
		{"POST", "/admin-docs/" + other + "/sync-budget"},
	} {
		r := e.do("", rt[0], base(id)+rt[1], nil)
		if r.status != 401 || r.code() != "unauthorized" {
			t.Errorf("%s %s without a token: %d %s", rt[0], rt[1], r.status, r.body)
		}
	}
}

func TestUserBGetsNotFoundOnEveryEndpointOfUserAsProject(t *testing.T) {
	e := newEnv(t)
	a, b := e.login("owner"), e.login("intruder")
	id := createProject(e, a, map[string]any{"templates": map[string]any{"timeline_preset": "wedding_pdf_44", "admin_docs": true}})
	task, doc := tasksOf(e, a, id)[0]["id"].(string), docsOf(e, a, id)[0]["id"].(string)

	for _, rt := range []struct {
		method, path string
		body         any
	}{
		{"GET", "/tasks", nil}, {"POST", "/tasks", map[string]any{"title": "x", "start_date": inDays(5), "end_date": inDays(6)}},
		{"POST", "/tasks/reorder", map[string]any{"ids": []string{}}}, {"POST", "/tasks/shift", map[string]any{"days": 1}},
		{"POST", "/tasks/generate-defaults", map[string]any{"preset": "wedding_pdf_44"}}, {"GET", "/tasks/export-spreadsheet", nil},
		{"PATCH", "/tasks/" + task, map[string]any{"title": "x"}}, {"PATCH", "/tasks/" + task + "/status", map[string]any{"status": "selesai"}},
		{"DELETE", "/tasks/" + task, nil},
		{"GET", "/admin-docs", nil}, {"POST", "/admin-docs", map[string]any{"document_name": "x"}}, {"POST", "/admin-docs/reorder", map[string]any{"ids": []string{}}},
		{"POST", "/admin-docs/apply-template", map[string]any{}}, {"PATCH", "/admin-docs/" + doc, map[string]any{"notes": "x"}},
		{"PATCH", "/admin-docs/" + doc + "/status", map[string]any{"status": "selesai"}}, {"DELETE", "/admin-docs/" + doc, nil},
		{"POST", "/admin-docs/" + doc + "/sync-budget", nil},
	} {
		wantError(t, e.do(b, rt.method, base(id)+rt.path, rt.body), 404, "not_found", "project not found")
	}

	// B's own project: A's task and document ids are unknown there.
	own := createProject(e, b, nil)
	wantError(t, e.do(b, "PATCH", base(own)+"/tasks/"+task, map[string]any{"title": "x"}), 404, "not_found", "task not found")
	wantError(t, e.do(b, "DELETE", base(own)+"/tasks/"+task, nil), 404, "not_found", "task not found")
	wantError(t, e.do(b, "PATCH", base(own)+"/admin-docs/"+doc+"/status", map[string]any{"status": "selesai"}), 404, "not_found", "document not found")
	wantError(t, e.do(b, "POST", base(own)+"/admin-docs/"+doc+"/sync-budget", nil), 404, "not_found", "document not found")

	// A's data is untouched, and ids that are not canonical UUIDs are 404 as well.
	if tasksOf(e, a, id)[0]["title"] == "x" {
		t.Fatal("the intruder changed a task")
	}
	wantError(t, e.do(a, "PATCH", base(id)+"/tasks/not-a-uuid", map[string]any{"title": "x"}), 404, "not_found", "task not found")
	wantError(t, e.do(a, "DELETE", base(id)+"/admin-docs/urn:uuid:"+doc, nil), 404, "not_found", "document not found")
}

func TestArchivedProjectIsReadOnlyButStillReadableAndExportable(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, map[string]any{"templates": map[string]any{"timeline_preset": "wedding_pdf_44", "admin_docs": true}})
	task := tasksOf(e, tok, id)[0]["id"].(string)
	mustStatus(t, e.do(tok, "PATCH", base(id)+"/status", map[string]any{"status": "archived"}), 200)

	for _, rt := range []struct {
		method, path string
		body         any
	}{
		{"POST", "/tasks", map[string]any{"title": "x", "start_date": inDays(5), "end_date": inDays(6)}},
		{"PATCH", "/tasks/" + task + "/status", map[string]any{"status": "selesai"}}, {"DELETE", "/tasks/" + task, nil},
		{"POST", "/tasks/shift", map[string]any{"days": 1}}, {"POST", "/tasks/generate-defaults", map[string]any{"preset": "wedding_pdf_44"}},
		{"POST", "/admin-docs", map[string]any{"document_name": "x"}}, {"POST", "/admin-docs/apply-template", map[string]any{}},
	} {
		wantError(t, e.do(tok, rt.method, base(id)+rt.path, rt.body), 409, "project_read_only", "")
	}
	mustStatus(t, e.do(tok, "GET", base(id)+"/tasks", nil), 200)
	mustStatus(t, e.do(tok, "GET", base(id)+"/admin-docs", nil), 200)
	if r := mustStatus(t, e.do(tok, "GET", base(id)+"/tasks/export-spreadsheet", nil), 200); r.header.Get("Content-Type") == "" {
		t.Fatal("export without content type")
	}
}

// ---------- the sample project: golden values of Wedding_Planner.pdf ----------

func TestSampleProjectGolden(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")

	r := mustStatus(t, e.do(tok, "POST", "/v1/planner/projects/sample", nil), 201)
	applied := r.data()["applied"].(map[string]any)
	if applied["timeline"].(map[string]any)["created"] != float64(50) || applied["admin_docs"].(map[string]any)["created"] != float64(15) {
		t.Fatalf("applied = %v", applied)
	}
	id := r.data()["project"].(map[string]any)["id"].(string)

	tasks := tasksOf(e, tok, id)
	if len(tasks) != 50 {
		t.Fatalf("%d tasks, want 44 PDF + 6 Adicara", len(tasks))
	}
	done, doing, todo, twoDays := 0, 0, 0, 0
	for i, tk := range tasks {
		if tk["sort_order"] != float64(i+1) {
			t.Fatalf("task %d has sort_order %v", i+1, tk["sort_order"])
		}
		switch tk["status"] {
		case "selesai":
			done++
			if i+1 > 15 {
				t.Errorf("task %d is done", i+1)
			}
		case "proses":
			doing++
		case "belum":
			todo++
			if tk["origin"] != "adicara" {
				t.Errorf("task %d (%v) is belum but comes from the PDF", i+1, tk["title"])
			}
		}
		if tk["duration_days"] == float64(2) {
			twoDays++
		}
	}
	if done != 15 || doing != 29 || todo != 6 || twoDays != 3 {
		t.Fatalf("statuses %d/%d/%d, two-day tasks %d; PDF p. 3 has 15 selesai, 29 proses, 3 two-day tasks", done, doing, todo, twoDays)
	}

	// Dates by the scaling 175/180 (4 Oct 2026 + 175 days = 28 Mar 2027).
	for _, c := range []struct {
		idx          int
		title, s, e_ string
	}{
		{0, "Pertemuan keluarga", "2026-10-04", "2026-10-05"},
		{2, "Tetapkan anggaran lamaran", "2026-10-08", "2026-10-10"},
		{6, "Lamaran", "2026-10-15", "2026-10-16"},
		{8, "Siapkan pendaftaran pernikahan", "2026-10-19", "2026-10-20"},
		{15, "Finalisasi Seserahan", "2026-11-02", "2026-11-03"},
		{44 + 0, "Finalisasi daftar tamu dan kuota pax", "2027-02-12", "2027-02-13"},
		{49, "Hari-H acara", "2027-03-28", "2027-03-28"},
	} {
		tk := tasks[c.idx]
		if tk["title"] != c.title || tk["start_date"] != c.s || tk["end_date"] != c.e_ {
			t.Errorf("task %d = %v %v..%v, want %s %s..%s", c.idx+1, tk["title"], tk["start_date"], tk["end_date"], c.title, c.s, c.e_)
		}
	}
	if tasks[24]["sub_event_scope"] != "resepsi" || tasks[8]["sub_event_scope"] != "akad" || tasks[0]["sub_event_scope"] != "lamaran" {
		t.Errorf("scopes: %v %v %v", tasks[0]["sub_event_scope"], tasks[8]["sub_event_scope"], tasks[24]["sub_event_scope"])
	}

	// The Lamaran session follows its task: 15 Oct 2026, not the 10 Oct once written in the FE spec.
	sessions := list(mustStatus(t, e.do(tok, "GET", base(id)+"/sessions", nil), 200).json()["data"])
	for _, s := range sessions {
		if s["scope_key"] == "lamaran" && s["session_date"] != "2026-10-15" {
			t.Errorf("Lamaran is dated %v", s["session_date"])
		}
	}

	docs := docsOf(e, tok, id)
	finished := 0
	for _, d := range docs {
		if d["status"] == "selesai" {
			finished++
		}
	}
	if len(docs) != 15 || finished != 9 || docs[0]["document_name"] != "Pendaftaran KUA" || docs[0]["nominal_cost"] != float64(600000) {
		t.Fatalf("%d documents, %d done; first %v", len(docs), finished, docs[0])
	}

	// Dashboard widgets: the two this module fills.
	dash := mustStatus(t, e.do(tok, "GET", base(id)+"/dashboard", nil), 200).data()
	tl := dash["timeline"].(map[string]any)
	if tl["total"] != float64(50) || tl["completed"] != float64(15) || tl["in_progress"] != float64(29) || tl["overdue"] != float64(0) || len(tl["urgent"].([]any)) != 0 {
		t.Errorf("timeline widget = %v", tl)
	}
	ad := dash["admin_docs"].(map[string]any)
	pending := ad["pending"].([]any)
	if ad["label"] != "Kelengkapan Berkas KUA" || ad["completed"] != float64(9) || ad["total"] != float64(15) || len(pending) != 5 || pending[0] != "FC KTP Ortu CMP" {
		t.Errorf("admin_docs widget = %v", ad)
	}

	// Creating the sample again is a no-op.
	again := mustStatus(t, e.do(tok, "POST", "/v1/planner/projects/sample", nil), 200)
	if again.data()["created"] != false || len(tasksOf(e, tok, id)) != 50 {
		t.Fatal("the second sample request must not add tasks")
	}
}

func TestEventTypesOffersTheTemplatesOnceTheModuleIsRegistered(t *testing.T) {
	e := newEnv(t)
	list := mustStatus(t, e.do(e.login("owner"), "GET", "/v1/planner/event-types", nil), 200).json()["data"].([]any)

	byKey := map[string]map[string]any{}
	for _, x := range list {
		m := x.(map[string]any)
		byKey[m["key"].(string)] = m
	}
	wedding := byKey["wedding"]["templates_available"].(map[string]any)
	if presets := wedding["timeline_presets"].([]any); len(presets) != 2 || presets[0] != "wedding_pdf_44" || wedding["include_adicara_tasks"] != true || wedding["admin_docs"] != true {
		t.Errorf("wedding templates_available = %v", wedding)
	}
	if other := byKey["other"]["templates_available"].(map[string]any); other["admin_docs"] != false || len(other["timeline_presets"].([]any)) != 1 {
		t.Errorf("other templates_available = %v", other)
	}

	w := byKey["wedding"]
	subjects := w["admin_subjects"].([]any)
	guide := w["admin_guide"].(map[string]any)
	if len(subjects) != 5 || subjects[3].(map[string]any)["key"] != "ortu_cpw" || guide["title"] != "Panduan Pengurusan Berkas Nikah" || len(guide["steps"].([]any)) != 4 {
		t.Errorf("wedding admin_subjects/admin_guide = %v %v", subjects, guide)
	}
	if _, leaked := w["admin_label"]; leaked {
		t.Error("admin_label must not be sent to clients")
	}
	seminar := byKey["seminar"]
	if s := seminar["admin_subjects"].([]any); len(s) != 3 || seminar["admin_guide"].(map[string]any)["title"] != "Panduan Perizinan Acara" {
		t.Errorf("seminar admin_subjects/admin_guide = %v", seminar)
	}
}

// ---------- the project wizard ----------

func TestWizardAppliesTimelineAndDocumentsAtomically(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")

	// Wedding, essential preset, with the Adicara tasks and the documents.
	r := mustStatus(t, e.do(tok, "POST", "/v1/planner/projects", project(map[string]any{"templates": map[string]any{
		"timeline_preset": "wedding_essential_22", "include_adicara_tasks": true, "admin_docs": true,
	}})), 201)
	applied := r.data()["applied"].(map[string]any)
	tl, docs := applied["timeline"].(map[string]any), applied["admin_docs"].(map[string]any)
	if tl["created"] != float64(28) || tl["skipped"] != float64(0) || docs["created"] != float64(15) {
		t.Fatalf("applied = %v", applied)
	}
	if w := tl["warnings"].([]any); len(w) != 1 || w[0].(map[string]any)["code"] != "lead_time_compressed" {
		t.Errorf("175 days is less than the 180-day standard: warnings = %v", tl["warnings"])
	}
	id := r.data()["project"].(map[string]any)["id"].(string)
	if n := len(tasksOf(e, tok, id)); n != 28 {
		t.Fatalf("%d tasks", n)
	}

	// Adicara tasks alone (no base preset) are allowed for the wedding.
	r = mustStatus(t, e.do(tok, "POST", "/v1/planner/projects", project(map[string]any{"templates": map[string]any{"include_adicara_tasks": true}})), 201)
	if n := r.data()["applied"].(map[string]any)["timeline"].(map[string]any)["created"]; n != float64(6) {
		t.Fatalf("Adicara tasks alone = %v", n)
	}

	// Seminar: the general preset plus its three tasks, and four permit documents.
	r = mustStatus(t, e.do(tok, "POST", "/v1/planner/projects", map[string]any{
		"event_type": "seminar", "title": "Seminar", "event_date": inDays(90), "metadata": map[string]any{"organizer_name": "BEM"},
		"templates": map[string]any{"timeline_preset": "general", "admin_docs": true},
	}), 201)
	applied = r.data()["applied"].(map[string]any)
	if applied["timeline"].(map[string]any)["created"] != float64(18) || applied["admin_docs"].(map[string]any)["created"] != float64(4) {
		t.Fatalf("seminar applied = %v", applied)
	}
	if w := applied["timeline"].(map[string]any)["warnings"].([]any); len(w) != 0 {
		t.Errorf("90 days is more than the 75-day lead time of a seminar: warnings = %v", w)
	}
	sid := r.data()["project"].(map[string]any)["id"].(string)
	st := tasksOf(e, tok, sid)
	if st[2]["title"] != "Undang dan konfirmasi pembicara" || st[2]["phase_name"] != "Perencanaan" || st[2]["category"] != "Umum" || st[2]["origin"] != "adicara" {
		t.Errorf("third seminar task = %v", st[2])
	}
	if d := docsOf(e, tok, sid); d[0]["document_name"] != "Kontrak venue" || d[0]["subject_role"] != "lembaga" || d[0]["origin"] != "adicara" {
		t.Errorf("first seminar document = %v", d[0])
	}

	// "other" has no document template; a preset of the other family is refused; nothing is left behind.
	before := count(e, `SELECT count(*) FROM planner_projects`)
	wantError(t, e.do(tok, "POST", "/v1/planner/projects", map[string]any{
		"event_type": "other", "title": "Acara", "event_date": inDays(30), "templates": map[string]any{"admin_docs": true},
	}), 400, "invalid_template", "admin_docs: not available for this event type")
	wantError(t, e.do(tok, "POST", "/v1/planner/projects", project(map[string]any{"templates": map[string]any{"timeline_preset": "general"}})),
		400, "invalid_template", "timeline_preset: not available for this event type")
	if count(e, `SELECT count(*) FROM planner_projects`) != before {
		t.Fatal("a rejected wizard request must not leave a project behind")
	}

	// A Hari-H that is today: every task between today and Hari-H, with the "too short" warning.
	r = mustStatus(t, e.do(tok, "POST", "/v1/planner/projects", project(map[string]any{"event_date": today, "templates": map[string]any{"timeline_preset": "wedding_pdf_44"}})), 201)
	tl = r.data()["applied"].(map[string]any)["timeline"].(map[string]any)
	if w := tl["warnings"].([]any); len(w) != 1 || w[0].(map[string]any)["code"] != "lead_time_too_short" || w[0].(map[string]any)["count"] != float64(44) {
		t.Errorf("warnings = %v", tl["warnings"])
	}
	for _, tk := range tasksOf(e, tok, r.data()["project"].(map[string]any)["id"].(string)) {
		if tk["start_date"] != today || tk["end_date"] != today {
			t.Fatalf("task %v is %v..%v, want today", tk["title"], tk["start_date"], tk["end_date"])
		}
	}
}

// ---------- tasks ----------

func newTask(title string, extra map[string]any) map[string]any {
	m := map[string]any{"title": title, "start_date": inDays(20), "end_date": inDays(22)}
	for k, v := range extra {
		m[k] = v
	}

	return m
}

func TestTaskLifecycle(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)

	// An empty project: fase 1 without a name, defaults for scope and category.
	r := mustStatus(t, e.do(tok, "POST", base(id)+"/tasks", newTask("  Survei venue  ", map[string]any{"pic": "Sinta", "notes": "baris 1\r\nbaris 2"})), 201)
	task := r.data()
	if task["title"] != "Survei venue" || task["sub_event_scope"] != "all" || task["category"] != "Umum" || task["phase_order"] != float64(1) ||
		task["phase_name"] != "" || task["status"] != "belum" || task["origin"] != "custom" || task["sort_order"] != float64(1) ||
		task["duration_days"] != float64(2) || task["overdue"] != false || task["notes"] != "baris 1\nbaris 2" {
		t.Fatalf("created task = %v", task)
	}
	if _, has := task["vendor_id"]; has {
		t.Error("vendor_id must be omitted when empty")
	}
	if _, has := task["template_key"]; has {
		t.Error("template_key must be omitted when empty")
	}
	tid := task["id"].(string)

	// The next task follows the phase of the last one; an explicit phase takes the known name.
	mustStatus(t, e.do(tok, "PATCH", base(id)+"/tasks/"+tid, map[string]any{"phase_order": 2, "phase_name": "Persiapan"}), 200)
	second := mustStatus(t, e.do(tok, "POST", base(id)+"/tasks", newTask("Pilih katering", map[string]any{"sub_event_scope": "resepsi", "category": "Vendor"})), 201).data()
	if second["phase_order"] != float64(2) || second["phase_name"] != "Persiapan" || second["sort_order"] != float64(2) || second["sub_event_scope"] != "resepsi" {
		t.Fatalf("second task = %v", second)
	}
	third := mustStatus(t, e.do(tok, "POST", base(id)+"/tasks", newTask("Cetak undangan", map[string]any{"phase_order": 2})), 201).data()
	if third["phase_name"] != "Persiapan" {
		t.Fatalf("an explicit phase must take over the known name: %v", third)
	}

	// PATCH changes only what is sent; clear_vendor and status.
	patched := mustStatus(t, e.do(tok, "PATCH", base(id)+"/tasks/"+tid, map[string]any{"status": "proses", "pic": "Arif"}), 200).data()
	if patched["status"] != "proses" || patched["pic"] != "Arif" || patched["title"] != "Survei venue" || patched["start_date"] != inDays(20) {
		t.Fatalf("patched = %v", patched)
	}
	toggled := mustStatus(t, e.do(tok, "PATCH", base(id)+"/tasks/"+tid+"/status", map[string]any{"status": "selesai"}), 200).data()
	back := mustStatus(t, e.do(tok, "PATCH", base(id)+"/tasks/"+tid+"/status", map[string]any{"status": "belum"}), 200).data()
	if toggled["status"] != "selesai" || back["status"] != "belum" {
		t.Fatal("a finished task may go back to belum")
	}

	// The list carries the summary of the whole project, not of the filter.
	filtered := mustStatus(t, e.do(tok, "GET", base(id)+"/tasks?scope=resepsi", nil), 200).data()
	if len(filtered["tasks"].([]any)) != 1 || filtered["summary"].(map[string]any)["total"] != float64(3) {
		t.Fatalf("filtered = %v", filtered)
	}
	for q, want := range map[string]int{"?scope=all": 2, "?category=Vendor": 1, "?phase=2": 3, "?status=belum": 3, "?status=selesai": 0} {
		if n := len(mustStatus(t, e.do(tok, "GET", base(id)+"/tasks"+q, nil), 200).data()["tasks"].([]any)); n != want {
			t.Errorf("GET /tasks%s returned %d, want %d", q, n, want)
		}
	}
	if arr := mustStatus(t, e.do(tok, "GET", base(id)+"/tasks?scope=nothing", nil), 200).data()["tasks"]; arr == nil || len(arr.([]any)) != 0 {
		t.Errorf("a filter without matches must return an empty array, got %v", arr)
	}

	mustStatus(t, e.do(tok, "DELETE", base(id)+"/tasks/"+tid, nil), 204)
	wantError(t, e.do(tok, "DELETE", base(id)+"/tasks/"+tid, nil), 404, "not_found", "task not found")
	if n := len(tasksOf(e, tok, id)); n != 2 {
		t.Fatalf("%d tasks after delete", n)
	}
}

func TestTaskValidation(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	long := strings.Repeat("x", 256)

	for name, c := range map[string]struct {
		body    map[string]any
		status  int
		code    string
		message string
	}{
		"blank title":         {newTask("   ", nil), 400, "invalid_input", "title: must not be empty"},
		"long title":          {newTask(long, nil), 400, "invalid_input", "title: must be at most 255 characters"},
		"control in title":    {newTask("a\tb", nil), 400, "invalid_input", "title: must not contain control characters"},
		"newline in title":    {newTask("a\nb", nil), 400, "invalid_input", "title: must not contain control characters"},
		"long category":       {newTask("x", map[string]any{"category": strings.Repeat("c", 61)}), 400, "invalid_input", "category: must be at most 60 characters"},
		"long pic":            {newTask("x", map[string]any{"pic": strings.Repeat("c", 101)}), 400, "invalid_input", "pic: must be at most 100 characters"},
		"tab in notes":        {newTask("x", map[string]any{"notes": "a\tb"}), 400, "invalid_input", "notes: must not contain control characters"},
		"long notes":          {newTask("x", map[string]any{"notes": strings.Repeat("n", 2001)}), 400, "invalid_input", "notes: must be at most 2000 characters"},
		"phase 21":            {newTask("x", map[string]any{"phase_order": 21}), 400, "invalid_input", "phase_order: must be between 1 and 20"},
		"phase negative":      {newTask("x", map[string]any{"phase_order": -1}), 400, "invalid_input", "phase_order: must be between 1 and 20"},
		"bad start":           {newTask("x", map[string]any{"start_date": "2026-13-01"}), 400, "invalid_date", "start_date: must be a valid date in YYYY-MM-DD format"},
		"missing end":         {map[string]any{"title": "x", "start_date": inDays(1)}, 400, "invalid_date", "end_date: must be a valid date in YYYY-MM-DD format"},
		"end before start":    {newTask("x", map[string]any{"start_date": inDays(5), "end_date": inDays(4)}), 400, "invalid_date", "end_date: must not be before start_date"},
		"start before window": {newTask("x", map[string]any{"start_date": "2023-01-01", "end_date": inDays(4)}), 400, "invalid_date", "start_date: must be within the project schedule window"},
		"end after window":    {newTask("x", map[string]any{"end_date": "2030-01-01"}), 400, "invalid_date", "end_date: must be within the project schedule window"},
		"unknown scope":       {newTask("x", map[string]any{"sub_event_scope": "nowhere"}), 400, "invalid_scope", "sub_event_scope: unknown session"},
		"malformed scope":     {newTask("x", map[string]any{"sub_event_scope": "Bad Scope"}), 400, "invalid_scope", "sub_event_scope: unknown session"},
		"vendor of nobody":    {newTask("x", map[string]any{"vendor_id": uuid.NewString()}), 400, "invalid_input", "vendor_id: must be a vendor of this project"},
		"vendor not a uuid":   {newTask("x", map[string]any{"vendor_id": "abc"}), 400, "invalid_json", ""},
		"title is not a text": {map[string]any{"title": 5, "start_date": inDays(1), "end_date": inDays(1)}, 400, "invalid_json", ""},
	} {
		t.Run(name, func(t *testing.T) {
			wantError(t, e.do(tok, "POST", base(id)+"/tasks", c.body), c.status, c.code, c.message)
		})
	}
	if n := len(tasksOf(e, tok, id)); n != 0 {
		t.Fatalf("rejected requests created %d tasks", n)
	}

	// A 255-character title and the edges of the window are accepted.
	mustStatus(t, e.do(tok, "POST", base(id)+"/tasks", newTask(strings.Repeat("é", 255), nil)), 201)
	mustStatus(t, e.do(tok, "POST", base(id)+"/tasks", newTask("lower edge", map[string]any{"start_date": inDays(175 - 1095), "end_date": inDays(175 - 1095)})), 201)
	mustStatus(t, e.do(tok, "POST", base(id)+"/tasks", newTask("upper edge", map[string]any{"start_date": inDays(175 + 365), "end_date": inDays(175 + 365)})), 201)

	// A vendor of this project is accepted and shows its name and WhatsApp number.
	vendor := uuid.New()
	exec(e, `INSERT INTO planner_vendors (id, project_id, category_key, category_label, vendor_name, formatted_wa_number)
	         VALUES ($1, $2, 'venue', 'Venue', 'Sasana Kriya Ballroom', '628123456789')`, vendor, id)
	withVendor := mustStatus(t, e.do(tok, "POST", base(id)+"/tasks", newTask("Visit venue", map[string]any{"vendor_id": vendor})), 201).data()
	if withVendor["vendor_id"] != vendor.String() || withVendor["vendor_name"] != "Sasana Kriya Ballroom" || withVendor["vendor_wa"] != "628123456789" {
		t.Fatalf("task with vendor = %v", withVendor)
	}
	wid := withVendor["id"].(string)

	// A vendor of another project is refused, and clear_vendor removes the link.
	other := createProject(e, tok, nil)
	foreign := uuid.New()
	exec(e, `INSERT INTO planner_vendors (id, project_id, category_key, category_label, vendor_name) VALUES ($1, $2, 'mua', 'MUA', 'Lain')`, foreign, other)
	wantError(t, e.do(tok, "PATCH", base(id)+"/tasks/"+wid, map[string]any{"vendor_id": foreign}), 400, "invalid_input", "vendor_id: must be a vendor of this project")
	wantError(t, e.do(tok, "PATCH", base(id)+"/tasks/"+wid, map[string]any{"vendor_id": vendor, "clear_vendor": true}), 400, "invalid_input", "clear_vendor: cannot be combined with vendor_id")
	cleared := mustStatus(t, e.do(tok, "PATCH", base(id)+"/tasks/"+wid, map[string]any{"clear_vendor": true}), 200).data()
	if _, has := cleared["vendor_id"]; has {
		t.Fatalf("vendor not cleared: %v", cleared)
	}

	// PATCH validates only what is sent: a task outside the window can still change status or title.
	exec(e, `UPDATE planner_tasks SET start_date = '2020-01-01', end_date = '2020-01-02' WHERE id = $1`, wid)
	mustStatus(t, e.do(tok, "PATCH", base(id)+"/tasks/"+wid, map[string]any{"status": "selesai"}), 200)
	mustStatus(t, e.do(tok, "PATCH", base(id)+"/tasks/"+wid, map[string]any{"title": "Renamed"}), 200)
	wantError(t, e.do(tok, "PATCH", base(id)+"/tasks/"+wid, map[string]any{"start_date": "2020-01-01"}), 400, "invalid_date", "start_date: must be within the project schedule window")
	for body, want := range map[string]string{
		`{"end_date":"2026-12-01","start_date":"2026-12-02"}`: "end_date: must not be before start_date",
		`{"start_date":"2027-01-01"}`:                         "start_date: must not be after end_date", // the stored end is 2020
		`{"end_date":"2019-01-01"}`:                           "end_date: must be within the project schedule window",
		`{"title":""}`:                                        "title: must not be empty",
		`{"status":"done"}`:                                   "status: must be one of belum, proses, selesai",
		`{"phase_order":0}`:                                   "phase_order: must be between 1 and 20",
		`{"sub_event_scope":"nowhere"}`:                       "sub_event_scope: unknown session",
	} {
		if r := e.do(tok, "PATCH", base(id)+"/tasks/"+wid, body); r.status != 400 || r.message() != want {
			t.Errorf("PATCH %s = %d %q, want 400 %q", body, r.status, r.message(), want)
		}
	}
	wantError(t, e.do(tok, "PATCH", base(id)+"/tasks/"+wid+"/status", map[string]any{"status": "done"}), 400, "invalid_input", "status: must be one of belum, proses, selesai")
	wantError(t, e.do(tok, "POST", base(id)+"/tasks", `{"title": `), 400, "invalid_json", "")
}

func TestOverdueIsComputedByTheServerInTheProjectZone(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	wib := createProject(e, tok, nil)
	jpr := createProject(e, tok, map[string]any{"timezone": "Asia/Jayapura"}) // UTC+9
	for _, id := range []string{wib, jpr} {
		mustStatus(t, e.do(tok, "POST", base(id)+"/tasks", newTask("Due 4 Oct", map[string]any{"start_date": today, "end_date": today})), 201)
	}

	// 12:00 WIB on 4 Oct: nobody is overdue yet, the task due today is urgent (0 days left).
	l := mustStatus(t, e.do(tok, "GET", base(wib)+"/tasks", nil), 200).data()
	if l["summary"].(map[string]any)["overdue"] != float64(0) || list(l["tasks"])[0]["overdue"] != false {
		t.Fatalf("due today is not overdue: %v", l)
	}

	repo := timelinerepo.NewPostgresRepository(e.pool)
	ctx := context.Background()
	// 17:30 UTC on 4 Oct is 00:30 on 5 Oct in Jakarta and 02:30 on 5 Oct in Jayapura; 15:30 UTC is 22:30 / 00:30.
	justAfterMidnightWIB := time.Date(2026, 10, 4, 17, 30, 0, 0, time.UTC)
	justBeforeMidnightWIB := time.Date(2026, 10, 4, 16, 30, 0, 0, time.UTC)
	for _, c := range []struct {
		name    string
		project string
		zone    string
		now     time.Time
		overdue int
	}{
		{"Jakarta 23:30 on the 4th", wib, "Asia/Jakarta", justBeforeMidnightWIB, 0},
		{"Jakarta 00:30 on the 5th", wib, "Asia/Jakarta", justAfterMidnightWIB, 1},
		{"Jayapura at the same instant as Jakarta 23:30 is already the 5th", jpr, "Asia/Jayapura", justBeforeMidnightWIB, 1},
	} {
		sum, err := repo.TimelineSummary(ctx, mustUUID(t, c.project), tld.At{Zone: c.zone, Now: c.now})
		if err != nil {
			t.Fatal(err)
		}
		if sum.Overdue != c.overdue {
			t.Errorf("%s: overdue = %d, want %d", c.name, sum.Overdue, c.overdue)
		}
	}
}

func TestReorderTasks(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, map[string]any{"templates": map[string]any{"timeline_preset": "wedding_essential_22"}})
	tasks := tasksOf(e, tok, id)
	ids := make([]string, len(tasks))
	for i, tk := range tasks {
		ids[len(tasks)-1-i] = tk["id"].(string)
	}

	// Reversed: the response is the whole list in the new order, sort_order 1..n.
	r := mustStatus(t, e.do(tok, "POST", base(id)+"/tasks/reorder", map[string]any{"ids": ids}), 200).data()
	got := list(r["tasks"])
	for i, tk := range got {
		if tk["id"] != ids[i] || tk["sort_order"] != float64(i+1) {
			t.Fatalf("position %d = %v (sort_order %v)", i, tk["id"], tk["sort_order"])
		}
	}
	if r["summary"].(map[string]any)["total"] != float64(22) {
		t.Fatalf("summary = %v", r["summary"])
	}

	// Anything but the full list is refused, and nothing changes.
	other := createProject(e, tok, map[string]any{"templates": map[string]any{"timeline_preset": "wedding_essential_22"}})
	foreign := tasksOf(e, tok, other)[0]["id"].(string)
	for name, bad := range map[string][]string{
		"one missing":    ids[1:],
		"one repeated":   append(append([]string{}, ids[1:]...), ids[1]),
		"a foreign task": append(append([]string{}, ids[1:]...), foreign),
		"empty":          {},
		"one too many":   append(append([]string{}, ids...), ids[0]),
	} {
		wantError(t, e.do(tok, "POST", base(id)+"/tasks/reorder", map[string]any{"ids": bad}), 400, "invalid_input", "ids: must list every id exactly once")
		_ = name
	}
	wantError(t, e.do(tok, "POST", base(id)+"/tasks/reorder", map[string]any{"ids": []string{"nope"}}), 400, "invalid_json", "")
	again := tasksOf(e, tok, id)
	for i, tk := range again {
		if tk["id"] != ids[i] {
			t.Fatalf("a refused reorder changed the order at %d", i)
		}
	}
	if sort := tasksOf(e, tok, other)[0]["sort_order"]; sort != float64(1) {
		t.Fatalf("the other project's task was touched: sort_order %v", sort)
	}
}

func TestManualShiftMovesOnlyUnfinishedTemplateTasks(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	r := mustStatus(t, e.do(tok, "POST", "/v1/planner/projects/sample", nil), 201)
	id := r.data()["project"].(map[string]any)["id"].(string)
	custom := mustStatus(t, e.do(tok, "POST", base(id)+"/tasks", newTask("Custom", map[string]any{"start_date": "2026-12-01", "end_date": "2026-12-02"})), 201).data()["id"].(string)
	before := tasksOf(e, tok, id)

	// 51 tasks, 15 done, 1 custom: 35 template tasks are shiftable.
	res := mustStatus(t, e.do(tok, "POST", base(id)+"/tasks/shift", map[string]any{"days": 7}), 200).data()
	if res["shifted"] != float64(35) || len(res["warnings"].([]any)) != 0 {
		t.Fatalf("shift = %v", res)
	}
	after := tasksOf(e, tok, id)
	for i := range after {
		moved := after[i]["start_date"] != before[i]["start_date"]
		shouldMove := before[i]["status"] != "selesai" && before[i]["template_key"] != nil
		if moved != shouldMove {
			t.Errorf("task %v: moved=%v, want %v", before[i]["title"], moved, shouldMove)
		}
		if after[i]["id"] == custom && moved {
			t.Error("a custom task must not move")
		}
	}

	// Not idempotent: another call moves again. Going back past today warns about overdue tasks.
	mustStatus(t, e.do(tok, "POST", base(id)+"/tasks/shift", map[string]any{"days": -7}), 200)
	back := mustStatus(t, e.do(tok, "POST", base(id)+"/tasks/shift", map[string]any{"days": -40}), 200).data()
	w := back["warnings"].([]any)
	if len(w) != 1 || w[0].(map[string]any)["code"] != "tasks_overdue_after_shift" || w[0].(map[string]any)["count"].(float64) < 1 {
		t.Fatalf("warnings = %v", back["warnings"])
	}

	wantError(t, e.do(tok, "POST", base(id)+"/tasks/shift", map[string]any{"days": 0}), 400, "invalid_input", "days: must not be zero")
	wantError(t, e.do(tok, "POST", base(id)+"/tasks/shift", map[string]any{"days": 1827}), 400, "invalid_input", "days: must be at most 1826 in absolute value")
	wantError(t, e.do(tok, "POST", base(id)+"/tasks/shift", map[string]any{"days": -1826}), 400, "invalid_date", "days: must keep every shifted task within the project schedule window")

	// With nothing to move the answer is an empty success.
	empty := createProject(e, tok, nil)
	if res := mustStatus(t, e.do(tok, "POST", base(empty)+"/tasks/shift", map[string]any{"days": 3}), 200).data(); res["shifted"] != float64(0) {
		t.Fatalf("empty shift = %v", res)
	}
}

func TestGenerateModes(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	gen := func(body map[string]any) resp { return e.do(tok, "POST", base(id)+"/tasks/generate-defaults", body) }

	// Append twice: the second call adds nothing.
	first := mustStatus(t, gen(map[string]any{"preset": "wedding_pdf_44", "include_adicara_tasks": true}), 200).data()
	if first["created"] != float64(50) || first["skipped"] != float64(0) || first["deleted"] != float64(0) {
		t.Fatalf("first = %v", first)
	}
	second := mustStatus(t, gen(map[string]any{"preset": "wedding_pdf_44", "include_adicara_tasks": true, "mode": "append"}), 200).data()
	if second["created"] != float64(0) || second["skipped"] != float64(50) || len(second["warnings"].([]any)) != 0 {
		t.Fatalf("second = %v", second)
	}
	if n := len(tasksOf(e, tok, id)); n != 50 {
		t.Fatalf("%d tasks after appending twice", n)
	}

	// A custom task and some progress.
	custom := mustStatus(t, e.do(tok, "POST", base(id)+"/tasks", newTask("Custom", nil)), 201).data()["id"].(string)
	tasks := tasksOf(e, tok, id)
	mustStatus(t, e.do(tok, "PATCH", base(id)+"/tasks/"+tasks[0]["id"].(string)+"/status", map[string]any{"status": "selesai"}), 200)

	// replace_template needs confirm when progress would be lost; the refusal changes nothing.
	wantError(t, gen(map[string]any{"preset": "wedding_pdf_44", "mode": "replace_template"}), 400, "invalid_input", "confirm: required")
	if n := len(tasksOf(e, tok, id)); n != 51 {
		t.Fatalf("a refused replace changed the tasks: %d", n)
	}
	rt := mustStatus(t, gen(map[string]any{"preset": "wedding_essential_22", "mode": "replace_template", "confirm": true}), 200).data()
	if rt["deleted"] != float64(50) || rt["created"] != float64(22) {
		t.Fatalf("replace_template = %v", rt)
	}
	after := tasksOf(e, tok, id)
	// The custom task was number 51; the new tasks continue after it (max + 1).
	if len(after) != 23 || after[0]["id"] != custom || after[0]["sort_order"] != float64(51) || after[1]["sort_order"] != float64(52) || after[22]["sort_order"] != float64(73) {
		t.Fatalf("custom task must stay in front, new tasks continue the numbering: %d tasks, first %v, last %v", len(after), after[0], after[len(after)-1])
	}
	for _, tk := range after[1:] {
		if tk["status"] != "belum" {
			t.Fatalf("regenerated tasks start as belum: %v", tk)
		}
	}

	// All tasks are belum now: replace_template needs no confirm; replace_all always does.
	mustStatus(t, gen(map[string]any{"preset": "wedding_pdf_44", "mode": "replace_template"}), 200)
	wantError(t, gen(map[string]any{"preset": "wedding_pdf_44", "mode": "replace_all"}), 400, "invalid_input", "confirm: required")
	all := mustStatus(t, gen(map[string]any{"preset": "wedding_pdf_44", "mode": "replace_all", "confirm": true}), 200).data()
	if all["deleted"] != float64(45) || all["created"] != float64(44) || len(tasksOf(e, tok, id)) != 44 {
		t.Fatalf("replace_all = %v", all)
	}

	// Validation.
	wantError(t, gen(map[string]any{}), 400, "invalid_input", "preset: required")
	wantError(t, gen(map[string]any{"preset": "general"}), 400, "invalid_template", "preset: not available for this event type")
	wantError(t, gen(map[string]any{"preset": "wedding_pdf_44", "mode": "merge"}), 400, "invalid_input", "mode: must be one of append, replace_template, replace_all")

	// A deleted session sends its tasks to Umum, with a warning.
	exec(e, `UPDATE planner_tasks SET sub_event_scope = 'all' WHERE project_id = $1`, id)
	exec(e, `DELETE FROM planner_tasks WHERE project_id = $1`, id)
	mustStatus(t, e.do(tok, "DELETE", base(id)+"/sessions/"+sessionID(t, e, tok, id, "lamaran"), nil), 204)
	fb := mustStatus(t, gen(map[string]any{"preset": "wedding_pdf_44"}), 200).data()
	w := fb["warnings"].([]any)
	if len(w) != 2 || w[1].(map[string]any)["code"] != "scope_fallback" || w[1].(map[string]any)["count"] != float64(7) {
		t.Fatalf("warnings = %v (tasks 1-7 belong to the deleted Lamaran session)", fb["warnings"])
	}
	if tasksOf(e, tok, id)[0]["sub_event_scope"] != "all" {
		t.Fatal("the fallback scope is all")
	}

	// A seminar takes its own preset only.
	sem := createProject(e, tok, map[string]any{"event_type": "seminar", "title": "Seminar", "metadata": map[string]any{"organizer_name": "BEM"}, "event_date": inDays(90)})
	wantError(t, e.do(tok, "POST", base(sem)+"/tasks/generate-defaults", map[string]any{"preset": "wedding_pdf_44"}), 400, "invalid_template", "preset: not available for this event type")
	if g := mustStatus(t, e.do(tok, "POST", base(sem)+"/tasks/generate-defaults", map[string]any{"preset": "general", "include_adicara_tasks": true}), 200).data(); g["created"] != float64(18) {
		t.Fatalf("seminar general = %v", g)
	}

	// An event in the past has no schedule to make.
	past := createProject(e, tok, nil)
	exec(e, `UPDATE planner_projects SET event_date = '2026-09-01', prep_start_date = '2026-08-01' WHERE id = $1`, past)
	wantError(t, e.do(tok, "POST", base(past)+"/tasks/generate-defaults", map[string]any{"preset": "wedding_pdf_44"}), 400, "invalid_date", "event_date: must not be before today")
}

func sessionID(t *testing.T, e *env, tok, project, scope string) string {
	t.Helper()
	for _, s := range list(mustStatus(t, e.do(tok, "GET", base(project)+"/sessions", nil), 200).json()["data"]) {
		if s["scope_key"] == scope {
			return s["id"].(string)
		}
	}
	t.Fatalf("no session %q", scope)

	return ""
}

// ---------- limits and races ----------

// fillTasks inserts n custom tasks straight into the table.
func fillTasks(e *env, project string, n int) {
	exec(e, `INSERT INTO planner_tasks (id, project_id, title, start_date, end_date, sort_order, origin)
	         SELECT gen_random_uuid(), $1, 'Filler ' || g, DATE '2026-12-01', DATE '2026-12-02', g, 'custom' FROM generate_series(1, $2) g`, project, n)
}

func TestTaskLimit(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)

	fillTasks(e, id, 499)
	mustStatus(t, e.do(tok, "POST", base(id)+"/tasks", newTask("The 500th", nil)), 201)
	wantError(t, e.do(tok, "POST", base(id)+"/tasks", newTask("The 501st", nil)), 409, "task_limit_reached", "at most 500 tasks per project")

	// A generator that would pass the limit writes nothing at all (also not the deletes of replace_*).
	if r := e.do(tok, "POST", base(id)+"/tasks/generate-defaults", map[string]any{"preset": "wedding_pdf_44"}); r.status != 409 || r.code() != "task_limit_reached" {
		t.Fatalf("append past the limit: %d %s", r.status, r.body)
	}
	if n := count(e, `SELECT count(*) FROM planner_tasks WHERE project_id = $1`, id); n != 500 {
		t.Fatalf("%d tasks after a refused generate", n)
	}
	// replace_all frees the room first, so it fits.
	mustStatus(t, e.do(tok, "POST", base(id)+"/tasks/generate-defaults", map[string]any{"preset": "wedding_pdf_44", "mode": "replace_all", "confirm": true}), 200)
	if n := count(e, `SELECT count(*) FROM planner_tasks WHERE project_id = $1`, id); n != 44 {
		t.Fatalf("%d tasks after replace_all", n)
	}
}

func TestTwoSimultaneousCreatesCannotPassTheLimit(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	fillTasks(e, id, 499)

	statuses := make([]int, 6)
	var wg sync.WaitGroup
	for i := range statuses {
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuses[i] = e.do(tok, "POST", base(id)+"/tasks", newTask(fmt.Sprintf("Racer %d", i), nil)).status
		}()
	}
	wg.Wait()

	created, refused := 0, 0
	for _, s := range statuses {
		switch s {
		case 201:
			created++
		case 409:
			refused++
		}
	}
	if created != 1 || refused != len(statuses)-1 {
		t.Fatalf("statuses %v: want exactly one 201", statuses)
	}
	if n := count(e, `SELECT count(*) FROM planner_tasks WHERE project_id = $1`, id); n != 500 {
		t.Fatalf("%d tasks", n)
	}
	// Distinct sort orders even though the creates raced.
	if n := count(e, `SELECT count(DISTINCT sort_order) FROM planner_tasks WHERE project_id = $1`, id); n != 500 {
		t.Fatalf("%d distinct sort_order values", n)
	}
}

// ---------- documents ----------

func TestDocumentLifecycleAndValidation(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)

	// Blank subject: "bersama" for the wedding.
	doc := mustStatus(t, e.do(tok, "POST", base(id)+"/admin-docs", map[string]any{"document_name": "  Surat izin  ", "nominal_cost": 25000, "notes": "a\nb"}), 201).data()
	if doc["document_name"] != "Surat izin" || doc["subject_role"] != "bersama" || doc["status"] != "belum" || doc["origin"] != "custom" ||
		doc["sort_order"] != float64(1) || doc["quantity_detail"] != "" || doc["nominal_cost"] != float64(25000) {
		t.Fatalf("doc = %v", doc)
	}
	if _, has := doc["budget_item_id"]; has {
		t.Error("budget_item_id must be omitted until the cost is synced")
	}
	did := doc["id"].(string)

	for name, c := range map[string]struct {
		body    map[string]any
		message string
	}{
		"blank name":      {map[string]any{"document_name": " "}, "document_name: must not be empty"},
		"long name":       {map[string]any{"document_name": strings.Repeat("d", 256)}, "document_name: must be at most 255 characters"},
		"unknown subject": {map[string]any{"document_name": "x", "subject_role": "panitia"}, "subject_role: must be one of the event type subjects"},
		"long quantity":   {map[string]any{"document_name": "x", "quantity_detail": strings.Repeat("q", 101)}, "quantity_detail: must be at most 100 characters"},
		"negative cost":   {map[string]any{"document_name": "x", "nominal_cost": -1}, "nominal_cost: must be between 0 and 10000000000000"},
		"huge cost":       {map[string]any{"document_name": "x", "nominal_cost": 10_000_000_000_001}, "nominal_cost: must be between 0 and 10000000000000"},
		"tab in notes":    {map[string]any{"document_name": "x", "notes": "a\tb"}, "notes: must not contain control characters"},
	} {
		t.Run(name, func(t *testing.T) {
			wantError(t, e.do(tok, "POST", base(id)+"/admin-docs", c.body), 400, "invalid_input", c.message)
		})
	}
	mustStatus(t, e.do(tok, "POST", base(id)+"/admin-docs", map[string]any{"document_name": strings.Repeat("d", 255), "subject_role": "ortu_cpw", "nominal_cost": 10_000_000_000_000}), 201)

	patched := mustStatus(t, e.do(tok, "PATCH", base(id)+"/admin-docs/"+did, map[string]any{"subject_role": "cpp", "quantity_detail": "2 lembar"}), 200).data()
	if patched["subject_role"] != "cpp" || patched["quantity_detail"] != "2 lembar" || patched["document_name"] != "Surat izin" || patched["nominal_cost"] != float64(25000) {
		t.Fatalf("patched = %v", patched)
	}
	wantError(t, e.do(tok, "PATCH", base(id)+"/admin-docs/"+did, map[string]any{"subject_role": "umum"}), 400, "invalid_input", "subject_role: must be one of the event type subjects")
	done := mustStatus(t, e.do(tok, "PATCH", base(id)+"/admin-docs/"+did+"/status", map[string]any{"status": "selesai"}), 200).data()
	if done["status"] != "selesai" {
		t.Fatal("status not changed")
	}
	wantError(t, e.do(tok, "PATCH", base(id)+"/admin-docs/"+did+"/status", map[string]any{"status": "proses"}), 400, "invalid_input", "status: must be one of belum, selesai")

	// The list filters by subject and status; the summary counts everything.
	l := mustStatus(t, e.do(tok, "GET", base(id)+"/admin-docs?subject=cpp&status=selesai", nil), 200).data()
	if len(l["docs"].([]any)) != 1 || l["summary"].(map[string]any)["total"] != float64(2) || l["summary"].(map[string]any)["completed"] != float64(1) {
		t.Fatalf("list = %v", l)
	}
	wantError(t, e.do(tok, "GET", base(id)+"/admin-docs?subject=panitia", nil), 400, "invalid_input", "subject: must be one of the event type subjects")
	wantError(t, e.do(tok, "GET", base(id)+"/admin-docs?status=proses", nil), 400, "invalid_input", "status: must be one of belum, selesai")

	// Non-wedding types default to "umum".
	sem := createProject(e, tok, map[string]any{"event_type": "seminar", "title": "Seminar", "metadata": map[string]any{"organizer_name": "BEM"}, "event_date": inDays(90)})
	if d := mustStatus(t, e.do(tok, "POST", base(sem)+"/admin-docs", map[string]any{"document_name": "Proposal"}), 201).data(); d["subject_role"] != "umum" {
		t.Fatalf("seminar default subject = %v", d["subject_role"])
	}

	// Reorder: the full list, once each.
	ids := []string{}
	for _, d := range docsOf(e, tok, id) {
		ids = append([]string{d["id"].(string)}, ids...)
	}
	re := mustStatus(t, e.do(tok, "POST", base(id)+"/admin-docs/reorder", map[string]any{"ids": ids}), 200).data()
	if list(re["docs"])[0]["id"] != ids[0] || list(re["docs"])[0]["sort_order"] != float64(1) {
		t.Fatalf("reorder = %v", re)
	}
	wantError(t, e.do(tok, "POST", base(id)+"/admin-docs/reorder", map[string]any{"ids": ids[1:]}), 400, "invalid_input", "ids: must list every id exactly once")

	mustStatus(t, e.do(tok, "DELETE", base(id)+"/admin-docs/"+did, nil), 204)
	wantError(t, e.do(tok, "DELETE", base(id)+"/admin-docs/"+did, nil), 404, "not_found", "document not found")
}

func TestApplyDocumentTemplate(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	apply := func(body map[string]any) resp { return e.do(tok, "POST", base(id)+"/admin-docs/apply-template", body) }

	first := mustStatus(t, apply(map[string]any{}), 200).data()
	if first["created"] != float64(15) || first["skipped"] != float64(0) || first["deleted"] != float64(0) {
		t.Fatalf("first = %v", first)
	}
	if again := mustStatus(t, apply(map[string]any{"mode": "append"}), 200).data(); again["created"] != float64(0) || again["skipped"] != float64(15) {
		t.Fatalf("append twice = %v", again)
	}
	docs := docsOf(e, tok, id)
	if docs[1]["document_name"] != "FC CPP ( KTP, Ijazah, Akte, KK)" || docs[1]["notes"] != "Fotokopi legalitas calon pengantin pria" || docs[8]["nominal_cost"] != float64(15000) || docs[0]["origin"] != "pdf" {
		t.Fatalf("documents = %v / %v", docs[0], docs[1])
	}

	// Progress and a custom document; the first template document is linked to a budget line.
	custom := mustStatus(t, e.do(tok, "POST", base(id)+"/admin-docs", map[string]any{"document_name": "Custom"}), 201).data()["id"].(string)
	mustStatus(t, e.do(tok, "PATCH", base(id)+"/admin-docs/"+docs[2]["id"].(string)+"/status", map[string]any{"status": "selesai"}), 200)
	kua := docs[0]["id"].(string)
	sync := mustStatus(t, e.do(tok, "POST", base(id)+"/admin-docs/"+kua+"/sync-budget", nil), 201).data()
	if sync["created"] != true {
		t.Fatalf("sync = %v", sync)
	}

	wantError(t, apply(map[string]any{"mode": "replace_template"}), 400, "invalid_input", "confirm: required")
	rep := mustStatus(t, apply(map[string]any{"mode": "replace_template", "confirm": true}), 200).data()
	w := rep["warnings"].([]any)
	if rep["deleted"] != float64(15) || rep["created"] != float64(15) || len(w) != 1 || w[0].(map[string]any)["code"] != "budget_link_lost" || w[0].(map[string]any)["count"] != float64(1) {
		t.Fatalf("replace_template = %v", rep)
	}
	after := docsOf(e, tok, id)
	if len(after) != 16 || after[0]["id"] != custom || after[0]["sort_order"] != float64(16) || after[1]["sort_order"] != float64(17) {
		t.Fatalf("the custom document stays in front: %d documents, first %v", len(after), after[0])
	}
	for _, d := range after[1:] {
		if d["status"] != "belum" {
			t.Fatalf("regenerated documents start as belum: %v", d)
		}
	}
	// The budget line keeps its amount and lost the link.
	if n := count(e, `SELECT count(*) FROM planner_budget_items WHERE project_id = $1 AND admin_doc_id IS NULL AND initial_estimate = 600000 AND source = 'admin_doc'`, id); n != 1 {
		t.Fatalf("the unlinked budget line is gone (%d)", n)
	}

	// Nothing to lose: no confirm needed. replace_all does not exist for documents.
	mustStatus(t, apply(map[string]any{"mode": "replace_template"}), 200)
	wantError(t, apply(map[string]any{"mode": "replace_all"}), 400, "invalid_input", "mode: must be one of append, replace_template")

	// "other" has no document template.
	other := createProject(e, tok, map[string]any{"event_type": "other", "title": "Acara", "metadata": map[string]any{}, "event_date": inDays(30)})
	wantError(t, e.do(tok, "POST", base(other)+"/admin-docs/apply-template", map[string]any{}), 400, "invalid_template", "admin_docs: not available for this event type")
}

func TestDocumentLimit(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	exec(e, `INSERT INTO planner_admin_docs (id, project_id, document_name, sort_order)
	         SELECT gen_random_uuid(), $1, 'Filler ' || g, g FROM generate_series(1, 199) g`, id)

	mustStatus(t, e.do(tok, "POST", base(id)+"/admin-docs", map[string]any{"document_name": "The 200th"}), 201)
	wantError(t, e.do(tok, "POST", base(id)+"/admin-docs", map[string]any{"document_name": "The 201st"}), 409, "admin_doc_limit_reached", "at most 200 documents per project")
	if r := e.do(tok, "POST", base(id)+"/admin-docs/apply-template", map[string]any{}); r.status != 409 || r.code() != "admin_doc_limit_reached" {
		t.Fatalf("apply past the limit: %d %s", r.status, r.body)
	}
	if n := count(e, `SELECT count(*) FROM planner_admin_docs WHERE project_id = $1`, id); n != 200 {
		t.Fatalf("%d documents", n)
	}
}

// ---------- sync with the budget ----------

func budgetItems(e *env, project string) []map[string]any {
	rows, err := e.pool.Query(context.Background(), `
		SELECT id::text, sub_event_scope, category, item_name, COALESCE(item_key, ''), source, COALESCE(admin_doc_id::text, ''), initial_estimate
		FROM planner_budget_items WHERE project_id = $1 ORDER BY sort_order, created_at`, project)
	if err != nil {
		e.t.Fatal(err)
	}
	defer rows.Close()

	var out []map[string]any
	for rows.Next() {
		var id, scope, category, name, key, source, doc string
		var estimate int64
		if err := rows.Scan(&id, &scope, &category, &name, &key, &source, &doc, &estimate); err != nil {
			e.t.Fatal(err)
		}
		out = append(out, map[string]any{"id": id, "scope": scope, "category": category, "name": name, "key": key, "source": source, "doc": doc, "estimate": estimate})
	}

	return out
}

func TestSyncBudget(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, map[string]any{"templates": map[string]any{"admin_docs": true}})
	docs := docsOf(e, tok, id)
	kua, materai, photo := docs[0]["id"].(string), docs[12]["id"].(string), docs[8]["id"].(string)
	sync := func(doc string, body any) resp {
		return e.do(tok, "POST", base(id)+"/admin-docs/"+doc+"/sync-budget", body)
	}

	// Menu 03 does not exist yet; its table does. A line `biaya_kua` waits in the Akad session.
	exec(e, `INSERT INTO planner_budget_items (id, project_id, sub_event_scope, category, item_name, item_key, source, initial_estimate, sort_order)
	         VALUES ($1, $2, 'akad', 'Legalitas', 'Biaya KUA', 'biaya_kua', 'template', 0, 1)`, uuid.New(), id)
	line := budgetItems(e, id)[0]["id"].(string)

	// (A) the waiting line is linked and its zero estimate takes the cost.
	a := mustStatus(t, sync(kua, nil), 200).data()
	if a["budget_item_id"] != line || a["created"] != false || a["linked"] != true || len(a["warnings"].([]any)) != 0 {
		t.Fatalf("A = %v", a)
	}
	if it := budgetItems(e, id); len(it) != 1 || it[0]["doc"] != kua || it[0]["estimate"] != int64(600000) {
		t.Fatalf("A items = %v", it)
	}
	if docsOf(e, tok, id)[0]["budget_item_id"] != line {
		t.Fatal("the document must show the line it is linked to")
	}

	// (B) again: nothing changes.
	b := mustStatus(t, sync(kua, nil), 200).data()
	if b["budget_item_id"] != line || b["created"] != false || b["linked"] != false || len(budgetItems(e, id)) != 1 {
		t.Fatalf("B = %v", b)
	}

	// (C) a document without a mapping gets a new line under "Legalitas", session "all".
	exec(e, `UPDATE planner_admin_docs SET nominal_cost = 20000 WHERE id = $1`, materai)
	c := mustStatus(t, sync(materai, ""), 201).data()
	items := budgetItems(e, id)
	if c["created"] != true || c["linked"] != false || len(items) != 2 || items[1]["category"] != "Legalitas" || items[1]["source"] != "admin_doc" ||
		items[1]["scope"] != "all" || items[1]["key"] != "" || items[1]["doc"] != materai || items[1]["estimate"] != int64(20000) || items[1]["name"] != "Materai 10.000" {
		t.Fatalf("C = %v / %v", c, items)
	}

	// (D) another KUA line with a different estimate: linked, kept as it is, and the response warns.
	other := createProject(e, tok, map[string]any{"templates": map[string]any{"admin_docs": true}})
	exec(e, `INSERT INTO planner_budget_items (id, project_id, sub_event_scope, category, item_name, item_key, source, initial_estimate, sort_order)
	         VALUES ($1, $2, 'akad', 'Legalitas', 'Biaya KUA', 'biaya_kua', 'template', 500000, 1)`, uuid.New(), other)
	kua2 := docsOf(e, tok, other)[0]["id"].(string)
	d := mustStatus(t, e.do(tok, "POST", base(other)+"/admin-docs/"+kua2+"/sync-budget", nil), 200).data()
	w := d["warnings"].([]any)
	if d["linked"] != true || len(w) != 1 || w[0].(map[string]any)["code"] != "estimate_differs" || budgetItems(e, other)[0]["estimate"] != int64(500000) {
		t.Fatalf("D = %v / %v", d, budgetItems(e, other))
	}

	// (E) a 255-character name is cut to the 150 characters the budget allows.
	exec(e, `UPDATE planner_admin_docs SET document_name = $2, nominal_cost = 1000 WHERE id = $1`, photo, strings.Repeat("é", 255))
	mustStatus(t, sync(photo, nil), 201)
	if name := budgetItems(e, id)[2]["name"].(string); len([]rune(name)) != 150 {
		t.Fatalf("name has %d characters", len([]rune(name)))
	}

	// (F) deleting a document keeps its line and amount, without the link.
	mustStatus(t, e.do(tok, "DELETE", base(id)+"/admin-docs/"+materai, nil), 204)
	if it := budgetItems(e, id)[1]; it["doc"] != "" || it["estimate"] != int64(20000) {
		t.Fatalf("F = %v", it)
	}

	// (G) a line that already belongs to another document is not taken over: the KUA document gets a new line.
	third := createProject(e, tok, map[string]any{"templates": map[string]any{"admin_docs": true}})
	thirdDocs := docsOf(e, tok, third)
	exec(e, `INSERT INTO planner_budget_items (id, project_id, sub_event_scope, category, item_name, item_key, source, admin_doc_id, initial_estimate, sort_order)
	         VALUES ($1, $2, 'akad', 'Legalitas', 'Biaya KUA', 'biaya_kua', 'admin_doc', $3, 100000, 1)`, uuid.New(), third, thirdDocs[1]["id"])
	g := mustStatus(t, e.do(tok, "POST", base(third)+"/admin-docs/"+thirdDocs[0]["id"].(string)+"/sync-budget", nil), 201).data()
	gi := budgetItems(e, third)
	if g["created"] != true || len(gi) != 2 || gi[0]["doc"] != thirdDocs[1]["id"] || gi[0]["estimate"] != int64(100000) || gi[1]["doc"] != thirdDocs[0]["id"] || gi[1]["scope"] != "akad" {
		t.Fatalf("G = %v / %v", g, gi)
	}

	// (H) a document without a cost is refused.
	wantError(t, sync(docs[1]["id"].(string), nil), 400, "invalid_input", "nominal_cost: must be greater than 0")

	// The requested session must exist; an explicit one overrides the default.
	exec(e, `UPDATE planner_admin_docs SET nominal_cost = 5000 WHERE id = $1`, docs[3]["id"])
	wantError(t, sync(docs[3]["id"].(string), map[string]any{"sub_event_scope": "nowhere"}), 400, "invalid_scope", "sub_event_scope: unknown session")
	wantError(t, sync(docs[3]["id"].(string), "{bad"), 400, "invalid_json", "")
	r := mustStatus(t, sync(docs[3]["id"].(string), map[string]any{"sub_event_scope": "resepsi"}), 201).data()
	last := budgetItems(e, id)
	if r["created"] != true || last[len(last)-1]["scope"] != "resepsi" {
		t.Fatalf("explicit scope: %v / %v", r, last[len(last)-1])
	}
}

// Two clicks at once must make one budget line: the budget table has no unique
// index on admin_doc_id, so only the lock on the document row prevents two.
func TestSyncBudgetDoubleClickMakesOneLine(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, map[string]any{"templates": map[string]any{"admin_docs": true}})
	doc := docsOf(e, tok, id)[8]["id"].(string) // 15.000

	statuses := make([]int, 8)
	var wg sync.WaitGroup
	for i := range statuses {
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuses[i] = e.do(tok, "POST", base(id)+"/admin-docs/"+doc+"/sync-budget", nil).status
		}()
	}
	wg.Wait()

	created := 0
	for _, s := range statuses {
		switch s {
		case 201:
			created++
		case 200:
		default:
			t.Fatalf("statuses %v", statuses)
		}
	}
	if created != 1 || len(budgetItems(e, id)) != 1 {
		t.Fatalf("statuses %v; %d budget lines, want exactly one", statuses, len(budgetItems(e, id)))
	}
}

// ---------- export ----------

func TestExportSpreadsheet(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	r := mustStatus(t, e.do(tok, "POST", "/v1/planner/projects/sample", nil), 201)
	id := r.data()["project"].(map[string]any)["id"].(string)
	mustStatus(t, e.do(tok, "POST", base(id)+"/tasks", newTask("=cmd|' /C calc'!A0", nil)), 201)

	res := mustStatus(t, e.do(tok, "GET", base(id)+"/tasks/export-spreadsheet", nil), 200)
	if ct := res.header.Get("Content-Type"); ct != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Fatalf("Content-Type = %q", ct)
	}
	if cd := res.header.Get("Content-Disposition"); cd != `attachment; filename="jadwal-contoh-pernikahan-sinta-arif-2026-10-04.xlsx"` {
		t.Fatalf("Content-Disposition = %q", cd)
	}
	if cc := res.header.Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("Cache-Control = %q, a workbook of the user's plans is never cached", cc)
	}
	if !bytes.HasPrefix(res.body, []byte("PK")) || len(res.body) < 5000 {
		t.Fatalf("not a zip: %d bytes", len(res.body))
	}
	// ?format is optional like on every other spreadsheet route; xlsx is the default and the only format.
	mustStatus(t, e.do(tok, "GET", base(id)+"/tasks/export-spreadsheet?format=xlsx", nil), 200)
	wantError(t, e.do(tok, "GET", base(id)+"/tasks/export-spreadsheet?format=pdf", nil), 400, "invalid_input", "format: must be xlsx")
	// The cell content is checked in export_test.go; here: all 51 tasks are in the file.
	if n := bytes.Count(res.body, []byte("PK\x03\x04")); n < 5 {
		t.Fatalf("%d parts in the workbook", n)
	}

	// A title with quotes and an emoji never reaches the header (the column forbids control characters, so no line break here).
	exec(e, `UPDATE planner_projects SET title = E'Pesta "Ulang" Tahun \U0001F382 \\ X' WHERE id = $1`, id)
	res = mustStatus(t, e.do(tok, "GET", base(id)+"/tasks/export-spreadsheet", nil), 200)
	if cd := res.header.Get("Content-Disposition"); cd != `attachment; filename="jadwal-pesta-ulang-tahun-x-2026-10-04.xlsx"` {
		t.Fatalf("Content-Disposition = %q", cd)
	}
}

func TestExportIsRateLimitedPerUser(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createProject(e, tok, nil)

	for i := 0; i < 10; i++ {
		mustStatus(t, e.do(tok, "GET", base(id)+"/tasks/export-spreadsheet", nil), 200)
	}
	wantError(t, e.do(tok, "GET", base(id)+"/tasks/export-spreadsheet", nil), 429, "rate_limited", "")
	// Another user, and every other endpoint, are not affected.
	mustStatus(t, e.do(tok, "GET", base(id)+"/tasks", nil), 200)
	own := createProject(e, e.login("second"), nil)
	mustStatus(t, e.do(e.login("second"), "GET", base(own)+"/tasks/export-spreadsheet", nil), 200)
}

// The widget figures of the audit, as the SQL gives them at different moments
// ("now" is a parameter of the query, not now()).
func TestTimelineWidgetAcrossTime(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := mustStatus(t, e.do(tok, "POST", "/v1/planner/projects/sample", nil), 201).data()["project"].(map[string]any)["id"].(string)
	repo := timelinerepo.NewPostgresRepository(e.pool)
	at := func(s string) tld.At {
		now, err := time.ParseInLocation("2006-01-02 15:04", s, time.FixedZone("WIB", 7*3600))
		if err != nil {
			t.Fatal(err)
		}

		return tld.At{Zone: "Asia/Jakarta", Now: now}
	}
	summary := func(when string) pd.TimelineSummary {
		s, err := repo.TimelineSummary(context.Background(), mustUUID(t, id), at(when))
		if err != nil {
			t.Fatal(err)
		}

		return s
	}

	// 4 Oct 2026: 15 done, 29 in progress, nothing due or late. Nothing is urgent: the first open task ends on 3 Nov.
	s := summary("2026-10-04 12:00")
	if s.Total != 50 || s.Completed != 15 || s.InProgress != 29 || s.Overdue != 0 || len(s.Urgent) != 0 {
		t.Fatalf("4 Oct: %+v", s)
	}

	// 1 Nov 2026: tasks 16, 17, 18 are due within 7 days, nearest first.
	s = summary("2026-11-01 12:00")
	if len(s.Urgent) != 3 || s.Urgent[0].Title != "Finalisasi Seserahan" || s.Urgent[0].DaysLeft != 2 || s.Urgent[1].DaysLeft != 4 || s.Urgent[2].DaysLeft != 6 || s.Urgent[0].DueDate != "2026-11-03" {
		t.Fatalf("1 Nov: %+v", s.Urgent)
	}

	// 4 Nov 2026: task 16 ended on 3 Nov and is late; it is still listed, with a negative number of days.
	s = summary("2026-11-04 09:00")
	if s.Overdue != 1 || s.Urgent[0].DaysLeft != -1 {
		t.Fatalf("4 Nov: overdue %d, urgent %+v", s.Overdue, s.Urgent)
	}
	// Midnight in Jakarta: 23:59 on the 3rd is not late yet, 00:00 on the 4th is.
	if got := summary("2026-11-03 23:59").Overdue; got != 0 {
		t.Errorf("23:59 on the due date: overdue %d", got)
	}
	if got := summary("2026-11-04 00:00").Overdue; got != 1 {
		t.Errorf("00:00 after the due date: overdue %d", got)
	}

	// The same figure arrives on the dashboard.
	mustStatus(t, e.do(tok, "PATCH", base(id)+"/tasks/"+tasksOf(e, tok, id)[15]["id"].(string)+"/status", map[string]any{"status": "selesai"}), 200)
	if s := summary("2026-11-04 09:00"); s.Overdue != 0 || s.Completed != 16 {
		t.Fatalf("after finishing task 16: %+v", s)
	}
}

// Where a foreign key fails depends on the key (02 BE 1): the vendor key on the
// statement itself, the deferred session key only at COMMIT. The usecase checks
// both first; these are the safety nets behind those checks.
func TestForeignKeyErrorsAreMappedWhereTheyAppear(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id, other := createProject(e, tok, nil), createProject(e, tok, nil)
	foreign := uuid.New()
	exec(e, `INSERT INTO planner_vendors (id, project_id, category_key, category_label, vendor_name) VALUES ($1, $2, 'mua', 'MUA', 'Lain')`, foreign, other)

	repo := timelinerepo.NewPostgresRepository(e.pool)
	tx := plannerrepo.NewTxManager(e.pool)
	ctx := context.Background()
	task := func(scope string, vendor *uuid.UUID) *tld.Task {
		return &tld.Task{
			ID: uuid.New(), ProjectID: mustUUID(t, id), Title: "t", SubEventScope: scope, Category: "Umum", PhaseOrder: 1,
			StartDate: time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2026, 12, 2, 0, 0, 0, 0, time.UTC),
			Status: tld.TaskBelum, VendorID: vendor, Origin: tld.OriginCustom,
		}
	}

	// A vendor of another project: refused on the statement.
	if err := repo.InsertTask(ctx, task("all", &foreign)); !errors.Is(err, tld.ErrInvalidVendor) {
		t.Fatalf("foreign vendor: %v", err)
	}
	// An unknown session: the statement succeeds and the COMMIT fails, which the shared TxManager maps to a conflict.
	err := tx.InTx(ctx, func(ctx context.Context) error { return repo.InsertTask(ctx, task("nowhere", nil)) })
	if !errors.Is(err, tld.ErrConflict) {
		t.Fatalf("unknown session at commit: %v", err)
	}
	if n := count(e, `SELECT count(*) FROM planner_tasks WHERE project_id = $1`, id); n != 0 {
		t.Fatalf("%d tasks were left behind", n)
	}
	// A duplicate template key outside ON CONFLICT is a conflict too.
	key := "wedding.lamaran"
	first, second := task("all", nil), task("all", nil)
	first.TemplateKey, second.TemplateKey = &key, &key
	if err := repo.InsertTask(ctx, first); err != nil {
		t.Fatal(err)
	}
	if err := repo.InsertTask(ctx, second); !errors.Is(err, tld.ErrConflict) {
		t.Fatalf("duplicate template_key: %v", err)
	}
	// Reads and writes of ids from another project find nothing.
	if _, err := repo.TaskByID(ctx, mustUUID(t, other), first.ID, tld.At{Zone: "Asia/Jakarta", Now: fixedNow}); !errors.Is(err, tld.ErrNotFound) {
		t.Fatalf("task of another project: %v", err)
	}
	if err := repo.DeleteTask(ctx, mustUUID(t, other), first.ID); !errors.Is(err, tld.ErrNotFound) {
		t.Fatalf("delete in another project: %v", err)
	}
	if err := repo.SetTaskStatus(ctx, mustUUID(t, other), first.ID, tld.TaskSelesai); !errors.Is(err, tld.ErrNotFound) {
		t.Fatalf("status in another project: %v", err)
	}
}
