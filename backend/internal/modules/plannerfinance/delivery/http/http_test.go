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
	timelinehttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannertimeline/delivery/http"
	timelinerepo "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannertimeline/repository"
	timelineusecase "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannertimeline/usecase"
)

// End-to-end through the real routers, BearerAuth, the usecases of menus 01-03
// and PostgreSQL, wired like internal/app/app.go. Like the other planner tests
// it WIPES the database named by TEST_DATABASE_URL (the name must end in _test)
// and is skipped without it; run with `go test -p 1`.
//
// The clock is fixed at 4 Oct 2026, 12:00 in Jakarta: the day the sample project
// of PLANNER_SPEC was written down, so the golden figures are those of the spec
// (175 days to Hari-H, 28 March 2027).

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
	finance := financeusecase.New(financerepo.NewPostgresRepository(pool), tx, registry, financeusecase.Options{Now: now})
	planner.RegisterSummary(timeline.TimelineProvider(), timeline.AdminDocsProvider(), finance.Provider())
	planner.RegisterTemplateAppliers(timeline.TimelineApplier(), timeline.AdminDocsApplier(), finance.Applier())
	planner.RegisterSampleOverlays(timeline.TimelineOverlay(), timeline.AdminDocsOverlay(), finance.Overlay())
	plannerhttp.RegisterRoutes(engine, auth, planner)
	timelinehttp.RegisterRoutes(engine, auth, planner, timeline)
	financehttp.RegisterRoutes(engine, auth, planner, finance)

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

// items returns the budget lines of the project keyed by "scope/item_key" (or "scope/name" without a key).
func lines(e *env, tok, id string) map[string]map[string]any {
	e.t.Helper()
	out := map[string]map[string]any{}
	for _, l := range list(mustStatus(e.t, e.do(tok, "GET", base(id)+"/budget/items", nil), 200).data()["items"]) {
		k, _ := l["item_key"].(string)
		if k == "" {
			k = l["item_name"].(string)
		}
		out[l["sub_event_scope"].(string)+"/"+k] = l
	}

	return out
}

func sessionsOf(e *env, tok, id string) []map[string]any {
	e.t.Helper()

	return list(mustStatus(e.t, e.do(tok, "GET", base(id)+"/sessions", nil), 200).json()["data"])
}

func get(e *env, tok, path string) map[string]any {
	e.t.Helper()

	return mustStatus(e.t, e.do(tok, "GET", path, nil), 200).data()
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

// ---------- the golden sample project ----------

// TestSampleProjectFinanceGolden checks the figures of Wedding_Planner.pdf
// p. 4-7 through the whole stack: the sample project is created by menu 01 with
// the template of this module and its overlay on top, then read back through
// every read endpoint (03 BE 6.3, 6.4, 6.6 and the golden figures of 7.1).
func TestSampleProjectFinanceGolden(t *testing.T) {
	e := newEnv(t)
	tok := e.login("owner")
	id := createSample(e, tok)

	matrix := get(e, tok, base(id)+"/budget/matrix")
	kpi := matrix["kpi"].(map[string]any)
	for k, want := range map[string]int64{
		"total_estimate": 74_150_000, "total_paid": 33_550_000, "total_remaining": 40_600_000, "total_overpaid": 0, "line_count": 14,
	} {
		if got := num(kpi[k]); got != want {
			t.Errorf("kpi.%s = %d, want %d", k, got, want)
		}
	}
	if st := kpi["line_status"].(map[string]any); num(st["settled"]) != 7 || num(st["partial"]) != 7 || num(st["unpaid"]) != 0 {
		t.Errorf("line_status = %v, want 7 settled, 7 partial, 0 unpaid", st)
	}

	rows := list(matrix["rows"])
	want := []struct {
		scope                                                      string
		estimate, dp, t1, t2, paid, remaining, overpaid, lineCount int64
	}{
		{"lamaran", 7_150_000, 3_250_000, 1_000_000, 500_000, 4_750_000, 2_400_000, 0, 5},
		{"akad", 600_000, 600_000, 0, 0, 600_000, 0, 0, 1},
		{"resepsi", 66_400_000, 16_200_000, 11_000_000, 1_000_000, 28_200_000, 38_200_000, 0, 8},
	}
	if len(rows) != len(want) {
		t.Fatalf("matrix has %d rows, want %d: %v", len(rows), len(want), rows)
	}
	for i, w := range want {
		r := rows[i]
		got := []int64{num(r["estimate"]), num(r["dp"]), num(r["term_1"]), num(r["term_2"]), num(r["paid"]), num(r["remaining"]), num(r["overpaid"]), num(r["line_count"])}
		exp := []int64{w.estimate, w.dp, w.t1, w.t2, w.paid, w.remaining, w.overpaid, w.lineCount}
		if r["sub_event_scope"] != w.scope || fmt.Sprint(got) != fmt.Sprint(exp) {
			t.Errorf("row %d = %v %v, want %s %v", i, r["sub_event_scope"], got, w.scope, exp)
		}
	}
	total := matrix["total"].(map[string]any)
	if total["sub_event_scope"] != "" || num(total["estimate"]) != 74_150_000 || num(total["dp"]) != 20_050_000 || num(total["term_1"]) != 12_000_000 ||
		num(total["term_2"]) != 1_500_000 || num(total["paid"]) != 33_550_000 || num(total["remaining"]) != 40_600_000 {
		t.Errorf("total row = %v", total)
	}
	if as := matrix["active_scenario"].(map[string]any); as["name"] != "Opsi Ideal 75 Jt" || num(as["total_target"]) != 75_000_000 || num(as["unallocated"]) != 850_000 {
		t.Errorf("active_scenario = %v", as)
	}

	cash := get(e, tok, base(id)+"/budget/cashflow-summary")
	for k, w := range map[string]int64{
		"total_savings_collected": 47_500_000, "total_paid": 33_550_000, "available_cash": 13_950_000,
		"remaining_debt": 40_600_000, "overpaid": 0, "cash_deficit": 26_650_000,
	} {
		if num(cash[k]) != w {
			t.Errorf("cashflow.%s = %d, want %d", k, num(cash[k]), w)
		}
	}

	items := lines(e, tok, id)
	if len(items) != 14 {
		t.Fatalf("%d lines, want 14", len(items))
	}
	for key, st := range map[string]string{
		"lamaran/catering": "settled", "lamaran/gaun_kebaya": "settled", "lamaran/fotografer": "dp_paid", "lamaran/cincin": "dp_paid", "lamaran/dekorasi": "settled",
		"akad/biaya_kua": "settled", "resepsi/mahar": "settled", "resepsi/catering": "partial", "resepsi/rias_busana": "dp_paid",
		"resepsi/venue_dekor": "dp_paid", "resepsi/undangan_souvenir": "dp_paid", "resepsi/dokumentasi": "dp_paid", "resepsi/seserahan": "settled", "resepsi/hiburan": "settled",
	} {
		l := items[key]
		if l == nil || l["payment_status"] != st || l["source"] != "template" {
			t.Errorf("%s = %v, want status %s from the template", key, l, st)
		}
	}
	if c := items["resepsi/catering"]; num(c["estimate"]) != 25_000_000 || num(c["dp_amount"]) != 5_000_000 || num(c["term_1_amount"]) != 10_000_000 || num(c["remaining"]) != 10_000_000 {
		t.Errorf("resepsi/catering = %v", c)
	}

	sv := get(e, tok, base(id)+"/savings")
	k := sv["kpi"].(map[string]any)
	if num(k["target"]) != 75_000_000 || num(k["collected"]) != 47_500_000 || num(k["shortfall"]) != 27_500_000 || k["target_follows_budget"] != true {
		t.Errorf("savings kpi = %v", k)
	}
	cs := list(sv["contributors"])
	if len(cs) != 2 || cs[0]["name"] != "Arif" || num(cs[0]["total"]) != 30_000_000 || num(cs[0]["entry_count"]) != 11 ||
		cs[1]["name"] != "Sinta" || num(cs[1]["total"]) != 17_500_000 || cs[1]["role_label"] != "bride" {
		t.Errorf("contributors = %v", cs)
	}
	ms := list(sv["milestones"])
	if len(ms) != 3 || ms[0]["sub_event_scope"] != "lamaran" || ms[1]["sub_event_scope"] != "akad" || ms[2]["sub_event_scope"] != "resepsi" {
		t.Fatalf("milestones = %v", ms)
	}
	for i, w := range [][3]any{{int64(7_150_000), int64(7_150_000), true}, {int64(600_000), int64(600_000), true}, {int64(66_400_000), int64(39_750_000), false}} {
		if num(ms[i]["target_amount"]) != w[0] || num(ms[i]["secured_amount"]) != w[1] || ms[i]["is_secured"] != w[2] {
			t.Errorf("milestone %d = %v, want %v", i, ms[i], w)
		}
	}
	if num(sv["unscheduled_estimate"]) != 0 {
		t.Errorf("unscheduled_estimate = %v", sv["unscheduled_estimate"])
	}
	rec := sv["recommendation"].(map[string]any)
	if num(rec["months_left"]) != 6 || num(rec["recommended_monthly"]) != 4_584_000 || num(rec["per_contributor"]) != 2_292_000 ||
		rec["target_date"] != inDays(175) || rec["basis"] != "session" {
		t.Errorf("recommendation = %v", rec)
	}
	mx := sv["matrix"].(map[string]any)
	months := mx["months"].([]any)
	if len(months) != 11 || months[0] != "2026-02" || months[10] != "2026-12" || len(list(mx["rows"])) != 11 {
		t.Errorf("matrix months = %v", months)
	}
	if first := list(mx["rows"])[0]; num(first["total"]) != 3_000_000 || len(first["by_contributor"].(map[string]any)) != 2 {
		t.Errorf("first matrix row = %v", first)
	}
	if len(list(sv["entries"])) != 22 {
		t.Errorf("%d entries, want 22", len(list(sv["entries"])))
	}

	sc := get(e, tok, base(id)+"/scenarios")
	scenarios := list(sc["scenarios"])
	if len(scenarios) != 3 {
		t.Fatalf("%d scenarios", len(scenarios))
	}
	for i, w := range []struct {
		name   string
		total  int64
		active bool
	}{{"Opsi Hemat 50 Jt", 50_000_000, false}, {"Opsi Ideal 75 Jt", 75_000_000, true}, {"Opsi 100 Jt", 100_000_000, false}} {
		s := scenarios[i]
		if s["name"] != w.name || num(s["total_target"]) != w.total || s["is_active"] != w.active || num(s["allocated"]) != w.total || len(list(s["items"])) != 10 {
			t.Errorf("scenario %d = %v %v %v alloc %v items %d", i, s["name"], s["total_target"], s["is_active"], s["allocated"], len(list(s["items"])))
		}
		if as := s["active_sub_events"].([]any); len(as) != 2 || as[0] != "akad" || as[1] != "resepsi" {
			t.Errorf("scenario %d active_sub_events = %v", i, as)
		}
	}
	if pr := list(sc["presets"]); len(pr) != 6 || pr[0]["key"] != "wedding_10jt" || pr[0]["kind"] != "fixed" || num(pr[5]["total_target"]) != 100_000_000 {
		t.Errorf("presets = %v", pr)
	}

	// The dashboard widget of menu 01 is now filled (it was "coming soon" before this module).
	fin := get(e, tok, base(id)+"/dashboard")["finance"].(map[string]any)
	for k, w := range map[string]int64{
		"target_budget": 75_000_000, "total_estimate": 74_150_000, "total_paid": 33_550_000, "remaining": 40_600_000, "overpaid": 0,
		"savings_target": 75_000_000, "savings_collected": 47_500_000, "free_cash": 13_950_000, "cash_deficit": 26_650_000,
	} {
		if num(fin[k]) != w {
			t.Errorf("dashboard.finance.%s = %d, want %d", k, num(fin[k]), w)
		}
	}
	if st := fin["line_status"].(map[string]any); num(st["settled"]) != 7 || num(st["partial"]) != 7 || num(st["unpaid"]) != 0 {
		t.Errorf("dashboard.finance.line_status = %v", st)
	}
}
