package http_test

import (
	"bytes"
	"context"
	"encoding/json"
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

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/config"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/db"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/ginserver"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/migrate"
	authrepo "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/auth/repository"
	authusecase "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/auth/usecase"
	plannerhttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/delivery/http"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	plannerrepo "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/repository"
	plannerusecase "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/usecase"
)

// End-to-end through the real router, BearerAuth, usecase and PostgreSQL. Like
// the repository tests it WIPES the database named by TEST_DATABASE_URL (name
// must end in _test) and is skipped without it; run with `go test -p 1`.

type env struct {
	t      *testing.T
	srv    *httptest.Server
	auth   *authusecase.Service
	svc    *plannerusecase.Service
	tokens map[string]string
}

func newEnv(t *testing.T, opts plannerusecase.Options) *env {
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
	svc := plannerusecase.New(plannerrepo.NewPostgresRepository(pool), plannerrepo.NewTxManager(pool), domain.DefaultRegistry(), opts)
	plannerhttp.RegisterRoutes(engine, auth, svc)

	srv := httptest.NewServer(engine)
	t.Cleanup(srv.Close)

	return &env{t: t, srv: srv, auth: auth, svc: svc, tokens: map[string]string{}}
}

// login registers the user once and returns a bearer token.
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

// inDays is the calendar date n days from today in Jakarta: creation refuses
// a Hari-H in the past or more than five years ahead, so fixtures are relative.
func inDays(n int) string {
	wib, _ := time.LoadLocation("Asia/Jakarta")

	return time.Now().In(wib).AddDate(0, 0, n).Format(domain.DateLayout)
}

func wedding(extra map[string]any) map[string]any {
	m := map[string]any{
		"event_type": "wedding", "title": "Pernikahan Sinta & Arif", "event_date": inDays(150), "target_budget": 75000000,
		"metadata": map[string]any{"bride_name": "Sinta", "groom_name": "Arif"},
	}
	for k, v := range extra {
		m[k] = v
	}

	return m
}

func mustStatus(t *testing.T, r resp, want int) resp {
	t.Helper()
	if r.status != want {
		t.Fatalf("status = %d, want %d; body: %s", r.status, want, r.body)
	}

	return r
}

func createProject(e *env, tok string, extra map[string]any) string {
	e.t.Helper()
	r := mustStatus(e.t, e.do(tok, "POST", "/v1/planner/projects", wedding(extra)), 201)

	return r.data()["project"].(map[string]any)["id"].(string)
}

func TestEveryRouteNeedsABearerToken(t *testing.T) {
	e := newEnv(t, plannerusecase.Options{})
	id := uuid.NewString()

	for _, rt := range [][2]string{
		{"GET", "/v1/planner/event-types"}, {"GET", "/v1/planner/projects"}, {"POST", "/v1/planner/projects"},
		{"GET", "/v1/planner/projects/summary"}, {"POST", "/v1/planner/projects/sample"},
		{"GET", "/v1/planner/projects/" + id}, {"PATCH", "/v1/planner/projects/" + id}, {"DELETE", "/v1/planner/projects/" + id},
		{"PATCH", "/v1/planner/projects/" + id + "/status"}, {"GET", "/v1/planner/projects/" + id + "/dashboard"},
		{"GET", "/v1/planner/projects/" + id + "/sessions"}, {"POST", "/v1/planner/projects/" + id + "/sessions"},
		{"POST", "/v1/planner/projects/" + id + "/sessions/reorder"}, {"PUT", "/v1/planner/projects/" + id + "/sessions/" + id},
		{"DELETE", "/v1/planner/projects/" + id + "/sessions/" + id},
	} {
		r := e.do("", rt[0], rt[1], nil)
		if r.status != 401 || r.code() != "unauthorized" {
			t.Errorf("%s %s without a token: %d %s", rt[0], rt[1], r.status, r.body)
		}
	}
}

func TestInvitationLinkRouteIsNotRegisteredYet(t *testing.T) {
	e := newEnv(t, plannerusecase.Options{})
	tok := e.login("owner")
	id := createProject(e, tok, nil)

	if r := e.do(tok, "PUT", "/v1/planner/projects/"+id+"/invitation", map[string]any{"invitation_id": nil}); r.status != 404 {
		t.Fatalf("PUT .../invitation = %d, want 404 until the Undangan Digital module exists", r.status)
	}
}

func TestEventTypesRegistryAndCaching(t *testing.T) {
	e := newEnv(t, plannerusecase.Options{})
	r := mustStatus(t, e.do(e.login("owner"), "GET", "/v1/planner/event-types", nil), 200)

	if cc := r.header.Get("Cache-Control"); cc != "public, max-age=3600" {
		t.Errorf("Cache-Control = %q", cc)
	}
	list := r.json()["data"].([]any)
	if len(list) != 9 {
		t.Fatalf("%d event types, want 9", len(list))
	}
	first := list[0].(map[string]any)
	if first["key"] != "wedding" || first["icon"] != "💍" || first["lead_time_days"] != float64(180) {
		t.Fatalf("first = %v", first)
	}
	if ta := first["templates_available"].(map[string]any); len(ta["timeline_presets"].([]any)) != 0 || ta["rundown"] != false {
		t.Fatalf("templates_available = %v; no module has registered an applier yet", ta)
	}
	if labels := list[7].(map[string]any)["labels"].(map[string]any); labels["guest"] != "Peserta" || labels["guest_menu"] != "Peserta & RSVP" {
		t.Fatalf("seminar labels = %v", labels)
	}
	sessions := first["sessions"].([]any)
	if len(sessions) != 3 || sessions[1].(map[string]any)["scope_key"] != "akad" {
		t.Fatalf("wedding sessions = %v", sessions)
	}
}

func TestProjectLifecycleOverHTTP(t *testing.T) {
	e := newEnv(t, plannerusecase.Options{})
	tok := e.login("owner")

	r := mustStatus(t, e.do(tok, "POST", "/v1/planner/projects", wedding(map[string]any{"title": "  Nikah  ", "client_token": "wizard-abc-123"})), 201)
	d := r.data()
	p := d["project"].(map[string]any)
	id := p["id"].(string)
	if d["created"] != true || p["title"] != "Nikah" || p["phase"] != "upcoming" || p["timezone"] != "Asia/Jakarta" || p["status"] != "active" {
		t.Fatalf("created project = %v", d)
	}
	if _, ok := d["applied"].(map[string]any); !ok {
		t.Fatalf("applied must be an object, got %v", d["applied"])
	}

	// A retry with the same client_token: 200 and the same project.
	again := mustStatus(t, e.do(tok, "POST", "/v1/planner/projects", wedding(map[string]any{"title": "ignored", "client_token": "wizard-abc-123"})), 200)
	if again.data()["created"] != false || again.data()["project"].(map[string]any)["id"] != id {
		t.Fatalf("retry = %s", again.body)
	}

	detail := mustStatus(t, e.do(tok, "GET", "/v1/planner/projects/"+id, nil), 200).data()
	if detail["shiftable_tasks"] != float64(0) || detail["project"].(map[string]any)["id"] != id {
		t.Fatalf("detail = %v", detail)
	}

	patch := mustStatus(t, e.do(tok, "PATCH", "/v1/planner/projects/"+id, map[string]any{
		"title": "Pernikahan Sinta & Arif", "event_date": inDays(157), "metadata": map[string]any{"city": "Bandung"},
	}), 200).data()
	pp := patch["project"].(map[string]any)
	if pp["event_date"] != inDays(157) || pp["metadata"].(map[string]any)["city"] != "Bandung" || pp["metadata"].(map[string]any)["bride_name"] != "Sinta" {
		t.Fatalf("patched = %v", pp)
	}
	if patch["warnings"] == nil || patch["hints"] == nil {
		t.Fatalf("warnings and hints must be arrays: %v", patch)
	}

	// Sessions born on Hari-H followed the new date.
	sessions := mustStatus(t, e.do(tok, "GET", "/v1/planner/projects/"+id+"/sessions", nil), 200).json()["data"].([]any)
	dates := map[string]any{}
	for _, s := range sessions {
		m := s.(map[string]any)
		dates[m["scope_key"].(string)] = m["session_date"]
		if m["usage"] == nil {
			t.Fatalf("GET /sessions rows carry usage: %v", m)
		}
	}
	if dates["akad"] != inDays(157) || dates["resepsi"] != inDays(157) || dates["lamaran"] != nil {
		t.Fatalf("session dates = %v", dates)
	}

	mustStatus(t, e.do(tok, "DELETE", "/v1/planner/projects/"+id+"?confirm=wrong", nil), 400)
	del := e.do(tok, "DELETE", "/v1/planner/projects/"+id+"?confirm="+url.QueryEscape("Pernikahan Sinta & Arif"), nil)
	mustStatus(t, del, 204)
	mustStatus(t, e.do(tok, "GET", "/v1/planner/projects/"+id, nil), 404)
}

func TestIsolationBetweenUsersOnEveryEndpoint(t *testing.T) {
	e := newEnv(t, plannerusecase.Options{})
	alice, bob := e.login("alice"), e.login("bob")
	id := createProject(e, alice, nil)
	sessions := mustStatus(t, e.do(alice, "GET", "/v1/planner/projects/"+id+"/sessions", nil), 200).json()["data"].([]any)
	sid := sessions[1].(map[string]any)["id"].(string)
	base := "/v1/planner/projects/" + id

	for _, rt := range []struct {
		method, path string
		body         any
	}{
		{"GET", base, nil}, {"PATCH", base, map[string]any{"title": "x"}}, {"DELETE", base + "?confirm=" + url.QueryEscape("Pernikahan Sinta & Arif"), nil},
		{"PATCH", base + "/status", map[string]any{"status": "archived"}}, {"GET", base + "/dashboard", nil},
		{"GET", base + "/sessions", nil}, {"POST", base + "/sessions", map[string]any{"name": "x"}},
		{"POST", base + "/sessions/reorder", map[string]any{"session_ids": []string{sid}}},
		{"PUT", base + "/sessions/" + sid, map[string]any{"name": "x"}}, {"DELETE", base + "/sessions/" + sid, nil},
	} {
		r := e.do(bob, rt.method, rt.path, rt.body)
		if r.status != 404 || r.code() != "not_found" {
			t.Errorf("bob %s %s: %d %s, want 404 not_found", rt.method, rt.path, r.status, r.body)
		}
	}

	// A non-UUID id never reaches a query and is the same 404; so is the
	// static /summary segment on a verb it does not serve (checked below).
	for _, bad := range []string{"not-a-uuid", "sample", "00000000-0000-0000-0000-000000000000", "urn:uuid:" + id, "{" + id + "}", strings.ReplaceAll(id, "-", "")} {
		if r := e.do(alice, "GET", "/v1/planner/projects/"+bad, nil); r.status != 404 {
			t.Errorf("GET /projects/%s = %d, want 404", bad, r.status)
		}
	}
	mustStatus(t, e.do(alice, "PATCH", "/v1/planner/projects/summary", map[string]any{}), 404)

	// Bob's views contain nothing of Alice's.
	mustStatus(t, e.do(bob, "GET", "/v1/planner/projects/summary", nil), 200)
	if list := mustStatus(t, e.do(bob, "GET", "/v1/planner/projects", nil), 200).json()["data"].([]any); len(list) != 0 {
		t.Fatalf("bob sees %d projects", len(list))
	}
	if r := e.do(alice, "GET", base, nil); r.status != 200 {
		t.Fatal("alice lost access to her own project")
	}

	// Ids of a session belonging to another project do not work either.
	other := createProject(e, alice, map[string]any{"title": "Kedua"})
	mustStatus(t, e.do(alice, "PUT", "/v1/planner/projects/"+other+"/sessions/"+sid, map[string]any{"name": "x"}), 404)
	mustStatus(t, e.do(alice, "DELETE", "/v1/planner/projects/"+other+"/sessions/"+sid, nil), 404)
}

func TestArchivedProjectIsReadOnlyExceptStatusAndDelete(t *testing.T) {
	e := newEnv(t, plannerusecase.Options{})
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	base := "/v1/planner/projects/" + id

	mustStatus(t, e.do(tok, "PATCH", base+"/status", map[string]any{"status": "archived"}), 200)

	for _, rt := range []struct {
		method, path string
		body         any
	}{
		{"PATCH", base, map[string]any{"title": "x"}}, {"POST", base + "/sessions", map[string]any{"name": "x"}},
		{"POST", base + "/sessions/reorder", map[string]any{"session_ids": []string{}}},
	} {
		if r := e.do(tok, rt.method, rt.path, rt.body); r.status != 409 || r.code() != "project_read_only" {
			t.Errorf("%s %s on an archived project: %d %s", rt.method, rt.path, r.status, r.body)
		}
	}
	// Reads still work.
	mustStatus(t, e.do(tok, "GET", base, nil), 200)
	mustStatus(t, e.do(tok, "GET", base+"/sessions", nil), 200)
	mustStatus(t, e.do(tok, "GET", base+"/dashboard", nil), 200)

	// Completed stays editable (late payments), and reopening an archive works.
	mustStatus(t, e.do(tok, "PATCH", base+"/status", map[string]any{"status": "completed"}), 200)
	mustStatus(t, e.do(tok, "PATCH", base, map[string]any{"title": "Setelah selesai"}), 200)

	// An archived project can be deleted without reopening it.
	mustStatus(t, e.do(tok, "PATCH", base+"/status", map[string]any{"status": "archived"}), 200)
	mustStatus(t, e.do(tok, "DELETE", base+"?confirm=Setelah+selesai", nil), 204)
}

func TestSessionManagementOverHTTP(t *testing.T) {
	e := newEnv(t, plannerusecase.Options{})
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	base := "/v1/planner/projects/" + id + "/sessions"

	created := mustStatus(t, e.do(tok, "POST", base, map[string]any{"name": "Ngunduh Mantu", "icon": "🎊", "start_time": "10.30", "session_date": inDays(170)}), 201).data()
	if created["scope_key"] != "ngunduh_mantu" || created["start_time"] != "10:30" || created["sort_order"] != float64(4) {
		t.Fatalf("created = %v", created)
	}
	mustStatus(t, e.do(tok, "POST", base, map[string]any{"name": "ngunduh MANTU"}), 409)
	if c := e.do(tok, "POST", base, map[string]any{"name": "ngunduh MANTU"}).code(); c != "session_name_taken" {
		t.Fatalf("code = %s", c)
	}
	mustStatus(t, e.do(tok, "POST", base, map[string]any{"name": ""}), 400)
	mustStatus(t, e.do(tok, "POST", base, "{not json"), 400)

	sid := created["id"].(string)
	put := mustStatus(t, e.do(tok, "PUT", base+"/"+sid, map[string]any{"name": "Ngunduh Mantu Bandung", "location_name": "Gedung A"}), 200).data()
	if put["location_name"] != "Gedung A" || put["scope_key"] != "ngunduh_mantu" || put["start_time"] != nil {
		t.Fatalf("PUT must replace everything but the key: %v", put)
	}

	list := mustStatus(t, e.do(tok, "GET", base, nil), 200).json()["data"].([]any)
	var ids []string
	for i := len(list) - 1; i >= 1; i-- { // reverse the non-system sessions
		ids = append(ids, list[i].(map[string]any)["id"].(string))
	}
	reordered := mustStatus(t, e.do(tok, "POST", base+"/reorder", map[string]any{"session_ids": ids}), 200).json()["data"].([]any)
	if reordered[1].(map[string]any)["scope_key"] != "ngunduh_mantu" {
		t.Fatalf("reordered = %v", reordered)
	}
	mustStatus(t, e.do(tok, "POST", base+"/reorder", map[string]any{"session_ids": ids[:2]}), 400)

	systemID := list[0].(map[string]any)["id"].(string)
	mustStatus(t, e.do(tok, "PUT", base+"/"+systemID, map[string]any{"name": "x"}), 400)
	mustStatus(t, e.do(tok, "DELETE", base+"/"+systemID, nil), 400)
	if c := e.do(tok, "DELETE", base+"/"+sid+"?move_to=nope", nil).code(); c != "invalid_scope" {
		t.Fatalf("code = %s", c)
	}
	mustStatus(t, e.do(tok, "DELETE", base+"/"+sid+"?delete_agendas=maybe", nil), 400)
	mustStatus(t, e.do(tok, "DELETE", base+"/"+sid+"?move_to=resepsi", nil), 204)
	mustStatus(t, e.do(tok, "DELETE", base+"/"+sid, nil), 404)
	mustStatus(t, e.do(tok, "DELETE", base+"/not-a-uuid", nil), 404)
}

func TestDeletingASessionWithAgendasReportsUsage(t *testing.T) {
	e := newEnv(t, plannerusecase.Options{})
	tok := e.login("owner")
	id := createProject(e, tok, nil)
	base := "/v1/planner/projects/" + id + "/sessions"
	list := mustStatus(t, e.do(tok, "GET", base, nil), 200).json()["data"].([]any)
	resepsi := list[3].(map[string]any)["id"].(string)

	pool, err := db.Connect(context.Background(), os.Getenv("TEST_DATABASE_URL"), 2)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	for i := 0; i < 2; i++ {
		if _, err := pool.Exec(context.Background(), `INSERT INTO planner_rundowns (id, project_id, session_id, start_time, activity_name) VALUES ($1,$2,$3,'09:00','Agenda')`,
			uuid.New(), id, resepsi); err != nil {
			t.Fatal(err)
		}
	}

	r := e.do(tok, "DELETE", base+"/"+resepsi, nil)
	if r.status != 409 || r.code() != "session_in_use" {
		t.Fatalf("got %d %s", r.status, r.body)
	}
	details := r.json()["error"].(map[string]any)["details"].(map[string]any)["usage"].(map[string]any)
	if details["agendas"] != float64(2) {
		t.Fatalf("details = %v, want usage.agendas = 2", details)
	}

	mustStatus(t, e.do(tok, "DELETE", base+"/"+resepsi+"?delete_agendas=true", nil), 204)
	var n int
	_ = pool.QueryRow(context.Background(), `SELECT count(*) FROM planner_rundowns WHERE project_id=$1`, id).Scan(&n)
	if n != 0 {
		t.Fatalf("%d agendas left", n)
	}
}

func TestHubSummaryAndSample(t *testing.T) {
	e := newEnv(t, plannerusecase.Options{})
	tok := e.login("owner")

	empty := mustStatus(t, e.do(tok, "GET", "/v1/planner/projects/summary", nil), 200).data()
	if empty["active_count"] != float64(0) || empty["nearest_project"] != nil || len(empty["projects"].([]any)) != 0 {
		t.Fatalf("empty hub = %v", empty)
	}

	sample := mustStatus(t, e.do(tok, "POST", "/v1/planner/projects/sample", nil), 201).data()
	sp := sample["project"].(map[string]any)
	if sample["created"] != true || sp["is_sample"] != true || sp["days_remaining"] != float64(175) || sp["title"] != "Contoh: Pernikahan Sinta & Arif" {
		t.Fatalf("sample = %v", sample)
	}
	again := mustStatus(t, e.do(tok, "POST", "/v1/planner/projects/sample", nil), 200).data()
	if again["created"] != false || again["project"].(map[string]any)["id"] != sp["id"] {
		t.Fatalf("second sample = %v", again)
	}

	createProject(e, tok, map[string]any{"title": "Dekat", "event_date": inDays(60)})
	createProject(e, tok, map[string]any{"title": "Jauh", "event_date": inDays(300), "target_budget": 25000000})

	sum := mustStatus(t, e.do(tok, "GET", "/v1/planner/projects/summary", nil), 200).data()
	if sum["active_count"] != float64(2) || sum["combined_target_budget"] != float64(100_000_000) {
		t.Fatalf("summary = %v; the sample must not count", sum)
	}
	if np := sum["nearest_project"].(map[string]any); np["title"] != "Contoh: Pernikahan Sinta & Arif" && np["title"] != "Dekat" {
		t.Fatalf("nearest = %v", np)
	} else if np["title"] != "Dekat" {
		t.Fatalf("nearest = %v, want Dekat (the sample is ignored)", np["title"])
	}
	if len(sum["projects"].([]any)) != 3 {
		t.Fatalf("projects = %d, want 3 including the sample", len(sum["projects"].([]any)))
	}

	lite := mustStatus(t, e.do(tok, "GET", "/v1/planner/projects?view=lite", nil), 200).json()["data"].([]any)
	if len(lite) != 3 {
		t.Fatalf("lite = %d", len(lite))
	}
	if _, has := lite[0].(map[string]any)["total_estimate"]; has {
		t.Fatal("the switcher query must not carry aggregates")
	}
	cards := mustStatus(t, e.do(tok, "GET", "/v1/planner/projects?status=active", nil), 200).json()["data"].([]any)
	if _, has := cards[0].(map[string]any)["total_estimate"]; !has {
		t.Fatal("the Hub query carries aggregates")
	}
	mustStatus(t, e.do(tok, "GET", "/v1/planner/projects?status=bogus", nil), 400)
	mustStatus(t, e.do(tok, "GET", "/v1/planner/projects?view=wide", nil), 400)
}

func TestSampleRacesProduceOneProject(t *testing.T) {
	e := newEnv(t, plannerusecase.Options{})
	tok := e.login("owner")

	var wg sync.WaitGroup
	statuses := make(chan int, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuses <- e.do(tok, "POST", "/v1/planner/projects/sample", nil).status
		}()
	}
	wg.Wait()
	close(statuses)

	counts := map[int]int{}
	for s := range statuses {
		counts[s]++
	}
	if counts[201] != 1 || counts[200] != 7 {
		t.Fatalf("statuses = %v, want exactly one 201 and seven 200", counts)
	}
	list := mustStatus(t, e.do(tok, "GET", "/v1/planner/projects", nil), 200).json()["data"].([]any)
	if len(list) != 1 {
		t.Fatalf("%d projects, want 1", len(list))
	}
}

func TestClientTokenRacesProduceOneProject(t *testing.T) {
	e := newEnv(t, plannerusecase.Options{})
	tok := e.login("owner")

	var wg sync.WaitGroup
	statuses := make(chan int, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuses <- e.do(tok, "POST", "/v1/planner/projects", wedding(map[string]any{"client_token": "double-click-1"})).status
		}()
	}
	wg.Wait()
	close(statuses)

	counts := map[int]int{}
	for s := range statuses {
		counts[s]++
	}
	if counts[201] != 1 || counts[200] != 7 {
		t.Fatalf("statuses = %v, want exactly one 201 and seven 200", counts)
	}
}

func TestCreateValidationAndLimitsOverHTTP(t *testing.T) {
	e := newEnv(t, plannerusecase.Options{MaxProjects: 2})
	tok := e.login("owner")

	for name, tc := range map[string]struct {
		body map[string]any
		code string
	}{
		"blank title":    {wedding(map[string]any{"title": "   "}), "invalid_input"},
		"bad zone":       {wedding(map[string]any{"timezone": "asia/jakarta"}), "invalid_timezone"},
		"past date":      {wedding(map[string]any{"event_date": "2000-01-01"}), "invalid_date"},
		"no metadata":    {wedding(map[string]any{"metadata": map[string]any{}}), "invalid_metadata"},
		"template":       {wedding(map[string]any{"templates": map[string]any{"rundown": true}}), "invalid_template"},
		"unknown type":   {wedding(map[string]any{"event_type": "party"}), "invalid_input"},
		"negative money": {wedding(map[string]any{"target_budget": -5}), "invalid_input"},
	} {
		r := e.do(tok, "POST", "/v1/planner/projects", tc.body)
		if r.status != 400 || r.code() != tc.code {
			t.Errorf("%s: %d %s, want 400 %s", name, r.status, r.body, tc.code)
		}
	}
	mustStatus(t, e.do(tok, "POST", "/v1/planner/projects", "[]"), 400)
	big := wedding(map[string]any{"title": strings.Repeat("x", 100_000)})
	if r := e.do(tok, "POST", "/v1/planner/projects", big); r.status != 400 || r.code() != "invalid_json" {
		t.Errorf("an oversized body: %d %s", r.status, r.code())
	}

	// A sample never takes a slot of the limit. A second user, because every
	// request above already counted against the 10-per-minute creation limit.
	tok = e.login("limited")
	mustStatus(t, e.do(tok, "POST", "/v1/planner/projects/sample", nil), 201)
	createProject(e, tok, nil)
	createProject(e, tok, nil)
	if r := e.do(tok, "POST", "/v1/planner/projects", wedding(nil)); r.status != 409 || r.code() != "project_limit_reached" {
		t.Errorf("third project: %d %s", r.status, r.body)
	}
}

func TestCreateRateLimitIsPerUser(t *testing.T) {
	e := newEnv(t, plannerusecase.Options{MaxProjects: 1000})
	alice, bob := e.login("alice"), e.login("bob")

	limited := 0
	for i := 0; i < 12; i++ {
		switch r := e.do(alice, "POST", "/v1/planner/projects", wedding(nil)); r.status {
		case 201:
		case 429:
			limited++
			if r.code() != "rate_limited" {
				t.Errorf("code = %s", r.code())
			}
		default:
			t.Fatalf("unexpected %d %s", r.status, r.body)
		}
	}
	if limited != 2 {
		t.Fatalf("%d requests were limited, want 2 of 12 (10 per minute)", limited)
	}
	mustStatus(t, e.do(bob, "POST", "/v1/planner/projects", wedding(nil)), 201) // another user is unaffected
}

func TestDashboardOverHTTPWithProviders(t *testing.T) {
	e := newEnv(t, plannerusecase.Options{})
	tok := e.login("owner")
	id := createProject(e, tok, map[string]any{"event_end_date": inDays(152)})

	d := mustStatus(t, e.do(tok, "GET", "/v1/planner/projects/"+id+"/dashboard", nil), 200).data()
	for _, k := range []string{"timeline", "finance", "vendors", "guests", "admin_docs", "rundown"} {
		if v, present := d[k]; !present || v != nil {
			t.Errorf("widget %s = %v (present=%v), want null before the module registers a provider", k, v, present)
		}
	}
	proj := d["project"].(map[string]any)
	if proj["total_days"] != float64(3) || proj["event_end_date"] != inDays(152) || proj["invitation_linked"] != false || proj["phase"] != "upcoming" {
		t.Fatalf("project = %v", proj)
	}
	if _, err := time.Parse(time.RFC3339, d["server_time"].(string)); err != nil {
		t.Fatalf("server_time = %v", d["server_time"])
	}
	if sessions := d["sessions"].([]any); len(sessions) != 3 {
		t.Fatalf("dashboard sessions = %d, want 3 (without 'all')", len(sessions))
	}

	e.svc.RegisterSummary(failingProvider{}, okProvider{})
	d = mustStatus(t, e.do(tok, "GET", "/v1/planner/projects/"+id+"/dashboard", nil), 200).data()
	if d["finance"].(map[string]any)["error"] != "internal" {
		t.Errorf("finance = %v, want {error: internal}", d["finance"])
	}
	if d["timeline"].(map[string]any)["total"] != float64(44) {
		t.Errorf("timeline = %v", d["timeline"])
	}
}

type failingProvider struct{}

func (failingProvider) Key() string { return "finance" }

func (failingProvider) Summarize(context.Context, domain.Project) (any, error) {
	panic("a bug in the finance module")
}

type okProvider struct{}

func (okProvider) Key() string { return "timeline" }

func (okProvider) Summarize(context.Context, domain.Project) (any, error) {
	return domain.TimelineSummary{Total: 44, Urgent: []domain.UrgentTask{}}, nil
}

func TestShiftTasksOverHTTPIsAtomicWithTheDateChange(t *testing.T) {
	e := newEnv(t, plannerusecase.Options{})
	tok := e.login("owner")
	id := createProject(e, tok, nil)

	pool, err := db.Connect(context.Background(), os.Getenv("TEST_DATABASE_URL"), 2)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	for i, st := range []string{"belum", "proses", "selesai"} {
		if _, err := pool.Exec(context.Background(), `INSERT INTO planner_tasks (id, project_id, title, start_date, end_date, status, template_key)
			VALUES ($1,$2,$3,'2099-01-10','2099-01-12',$4,$5)`, uuid.New(), id, fmt.Sprintf("T%d", i), st, fmt.Sprintf("wedding.t%d", i)); err != nil {
			t.Fatal(err)
		}
	}

	detail := mustStatus(t, e.do(tok, "GET", "/v1/planner/projects/"+id, nil), 200).data()
	if detail["shiftable_tasks"] != float64(2) {
		t.Fatalf("shiftable_tasks = %v, want 2", detail["shiftable_tasks"])
	}

	// Without shift_tasks: a hint, nothing moves.
	r := mustStatus(t, e.do(tok, "PATCH", "/v1/planner/projects/"+id, map[string]any{"event_date": inDays(143)}), 200).data()
	hints := r["hints"].([]any)
	if len(hints) != 1 || hints[0].(map[string]any)["code"] != "tasks_not_shifted" || hints[0].(map[string]any)["shift_days"] != float64(-7) || hints[0].(map[string]any)["count"] != float64(2) {
		t.Fatalf("hints = %v", hints)
	}
	var start string
	_ = pool.QueryRow(context.Background(), `SELECT to_char(min(start_date),'YYYY-MM-DD') FROM planner_tasks WHERE project_id=$1`, id).Scan(&start)
	if start != "2099-01-10" {
		t.Fatalf("tasks moved without shift_tasks: %s", start)
	}

	// With shift_tasks, and an invalid change in the same request: nothing is applied.
	bad := e.do(tok, "PATCH", "/v1/planner/projects/"+id, map[string]any{"event_date": inDays(136), "shift_tasks": true, "target_budget": -1})
	mustStatus(t, bad, 400)
	_ = pool.QueryRow(context.Background(), `SELECT to_char(min(start_date),'YYYY-MM-DD') FROM planner_tasks WHERE project_id=$1`, id).Scan(&start)
	if start != "2099-01-10" {
		t.Fatalf("a failed PATCH shifted tasks: %s", start)
	}
	if got := mustStatus(t, e.do(tok, "GET", "/v1/planner/projects/"+id, nil), 200).data()["project"].(map[string]any)["event_date"]; got != inDays(143) {
		t.Fatalf("a failed PATCH changed the date to %v", got)
	}

	mustStatus(t, e.do(tok, "PATCH", "/v1/planner/projects/"+id, map[string]any{"event_date": inDays(136), "shift_tasks": true}), 200)
	rows, _ := pool.Query(context.Background(), `SELECT title, to_char(start_date,'YYYY-MM-DD') FROM planner_tasks WHERE project_id=$1 ORDER BY title`, id)
	got := map[string]string{}
	for rows.Next() {
		var title, d string
		_ = rows.Scan(&title, &d)
		got[title] = d
	}
	rows.Close()
	if got["T0"] != "2099-01-03" || got["T1"] != "2099-01-03" || got["T2"] != "2099-01-10" {
		t.Fatalf("task dates = %v; open template tasks move 7 days earlier, the finished one stays", got)
	}
}
