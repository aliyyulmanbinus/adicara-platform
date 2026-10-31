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
	financehttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerfinance/delivery/http"
	financerepo "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerfinance/repository"
	financeusecase "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerfinance/usecase"
	guesthttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerguest/delivery/http"
	guestrepo "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerguest/repository"
	guestusecase "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerguest/usecase"
	rundownhttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerrundown/delivery/http"
	rundownrepo "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerrundown/repository"
	rundownusecase "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerrundown/usecase"
	timelinehttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannertimeline/delivery/http"
	timelinerepo "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannertimeline/repository"
	timelineusecase "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannertimeline/usecase"
	vendorhttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannervendor/delivery/http"
	vendorrepo "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannervendor/repository"
	vendorusecase "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannervendor/usecase"
)

// End-to-end through the real routers, BearerAuth, the usecases of menus 01-06 and PostgreSQL, wired like
// internal/app/app.go. Like the other planner tests it WIPES the database named by TEST_DATABASE_URL (the name must end
// in _test) and is skipped without it; run with `go test -p 1`.
//
// The clock is fixed at 4 Oct 2026, 12:00 in Jakarta: the day the sample project of PLANNER_SPEC was written down, so the
// golden figures are those of the spec (175 days to Hari-H, 28 March 2027).

const today = "2026-10-04"

var fixedNow = time.Date(2026, 10, 4, 5, 0, 0, 0, time.UTC)

type env struct {
	t      *testing.T
	engine *gin.Engine
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
	pool, err := db.Connect(context.Background(), dsn, 20)
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
	finance := financeusecase.New(financerepo.NewPostgresRepository(pool), tx, registry, financeusecase.Options{Now: now})
	vendor := vendorusecase.New(vendorrepo.NewPostgresRepository(pool), tx, registry, vendorusecase.Options{Now: now})
	guest := guestusecase.New(guestrepo.NewPostgresRepository(pool), tx, registry, guestusecase.Options{Now: now})
	rundown := rundownusecase.New(rundownrepo.NewPostgresRepository(pool), tx, registry, rundownusecase.Options{Now: now})
	planner.RegisterSummary(timeline.TimelineProvider(), timeline.AdminDocsProvider(), finance.Provider(), vendor.Provider(), guest.Provider(), rundown.Provider())
	planner.RegisterTemplateAppliers(timeline.TimelineApplier(), timeline.AdminDocsApplier(), finance.Applier(), vendor.Applier(), rundown.RundownApplier(), rundown.PlaylistApplier())
	planner.RegisterSampleOverlays(timeline.TimelineOverlay(), timeline.AdminDocsOverlay(), finance.Overlay(), vendor.VendorsOverlay(), vendor.ShoppingOverlay(), guest.GuestsOverlay())
	plannerhttp.RegisterRoutes(engine, auth, planner)
	timelinehttp.RegisterRoutes(engine, auth, planner, timeline)
	financehttp.RegisterRoutes(engine, auth, planner, finance)
	vendorhttp.RegisterRoutes(engine, auth, planner, vendor)
	guesthttp.RegisterRoutes(engine, auth, planner, guest)
	rundownhttp.RegisterRoutes(engine, auth, planner, rundown)

	srv := httptest.NewServer(engine)
	t.Cleanup(srv.Close)

	return &env{t: t, engine: engine, srv: srv, auth: auth, pool: pool, tokens: map[string]string{}}
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

func createSample(e *env, tok string) string {
	e.t.Helper()
	r := mustStatus(e.t, e.do(tok, "POST", "/v1/planner/projects/sample", nil), 201)

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

func num(v any) int64 {
	f, _ := v.(float64)

	return int64(f)
}

func get(e *env, tok, path string) map[string]any {
	e.t.Helper()

	return mustStatus(e.t, e.do(tok, "GET", path, nil), 200).data()
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

func jsonOf(v any) string {
	b, _ := json.Marshal(v)

	return string(b)
}

func mustUUID(t *testing.T, v any) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(fmt.Sprint(v))
	if err != nil {
		t.Fatal(err)
	}

	return id
}

// concurrently runs n calls at the same time and returns their statuses.
func concurrently(n int, call func(i int) int) []int {
	var wg sync.WaitGroup
	out := make([]int, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			out[i] = call(i)
		}(i)
	}
	wg.Wait()

	return out
}

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
