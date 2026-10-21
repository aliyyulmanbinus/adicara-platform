package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

type stubProvider struct {
	key string
	fn  func(ctx context.Context, p domain.Project) (any, error)
}

func (s *stubProvider) Key() string { return s.key }

func (s *stubProvider) Summarize(ctx context.Context, p domain.Project) (any, error) {
	if s.fn == nil {
		return map[string]any{"ok": s.key}, nil
	}

	return s.fn(ctx, p)
}

func TestDashboardWithoutProvidersHasNullWidgets(t *testing.T) {
	svc, _, p := projectWithSessions(t)

	d, err := svc.Dashboard(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(d)
	var got map[string]json.RawMessage
	_ = json.Unmarshal(raw, &got)
	for _, key := range []string{"timeline", "finance", "vendors", "guests", "admin_docs", "rundown"} {
		v, present := got[key]
		if !present || string(v) != "null" {
			t.Errorf("widget %q = %s (present=%v), want null", key, v, present)
		}
	}
	if len(d.Sessions) != 3 {
		t.Fatalf("%d sessions, want the 3 wedding sessions without 'all'", len(d.Sessions))
	}
	for _, s := range d.Sessions {
		if s.IsSystem || s.ScopeKey == "all" {
			t.Errorf("the dashboard must not list the system session: %+v", s)
		}
	}

	pj := d.Project
	if pj.DaysRemaining != 158 || pj.Phase != domain.PhaseUpcoming || pj.TotalDays != 1 || pj.InvitationLinked {
		t.Fatalf("project block = %+v", pj)
	}
	if d.ServerTime.IsZero() || !d.ServerTime.Equal(jakartaMorning) {
		t.Fatalf("server_time = %v, want %v", d.ServerTime, jakartaMorning)
	}
}

func TestDashboardMergesProvidersAndKeepsGoingWhenOneFails(t *testing.T) {
	svc, _, p := projectWithSessions(t)
	svc.RegisterSummary(
		&stubProvider{key: "timeline", fn: func(context.Context, domain.Project) (any, error) {
			return &domain.TimelineSummary{Total: 44, Completed: 11, Urgent: []domain.UrgentTask{}}, nil
		}},
		&stubProvider{key: "finance", fn: func(context.Context, domain.Project) (any, error) { return nil, errors.New("boom") }},
		&stubProvider{key: "vendors", fn: func(context.Context, domain.Project) (any, error) { panic("provider bug") }},
		&stubProvider{key: "guests"},
	)

	d, err := svc.Dashboard(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(d)
	var got map[string]any
	_ = json.Unmarshal(raw, &got)

	if tl := got["timeline"].(map[string]any); tl["total"] != float64(44) || tl["in_progress"] != float64(0) {
		t.Fatalf("timeline = %v", tl)
	}
	for _, key := range []string{"finance", "vendors"} {
		if e, _ := got[key].(map[string]any); e["error"] != "internal" {
			t.Errorf("%s = %v, want {error: internal}", key, got[key])
		}
	}
	if got["guests"].(map[string]any)["ok"] != "guests" {
		t.Errorf("guests = %v", got["guests"])
	}
	if got["admin_docs"] != nil || got["rundown"] != nil {
		t.Errorf("widgets without a provider must be null: %v / %v", got["admin_docs"], got["rundown"])
	}
}

func TestDashboardSlowProviderTimesOutWithoutBlockingTheOthers(t *testing.T) {
	svc, _ := newTestService(t, Options{DashboardTimeout: 80 * time.Millisecond})
	p := createdProject(t, svc, validWedding())

	released := make(chan struct{})
	t.Cleanup(func() { close(released) })
	svc.RegisterSummary(
		&stubProvider{key: "rundown", fn: func(ctx context.Context, _ domain.Project) (any, error) {
			select { // ignores nothing: waits for the deadline
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-released:
				return "late", nil
			}
		}},
		&stubProvider{key: "guests"},
		&stubProvider{key: "vendors", fn: func(ctx context.Context, _ domain.Project) (any, error) {
			<-released // ignores ctx entirely
			return "late", nil
		}},
	)

	start := time.Now()
	d, err := svc.Dashboard(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("dashboard took %v; a provider that ignores ctx must not hold it past the deadline", elapsed)
	}
	if e, ok := d.Rundown.(domain.WidgetError); !ok || e.Error != "timeout" {
		t.Errorf("rundown = %#v, want timeout", d.Rundown)
	}
	if e, ok := d.Vendors.(domain.WidgetError); !ok || e.Error != "timeout" {
		t.Errorf("vendors = %#v, want timeout", d.Vendors)
	}
	if m, ok := d.Guests.(map[string]any); !ok || m["ok"] != "guests" {
		t.Errorf("guests = %#v, want its summary", d.Guests)
	}
}

func TestDashboardRunsAtMostThreeProvidersAtOnce(t *testing.T) {
	svc, _, p := projectWithSessions(t)

	var running, peak atomic.Int32
	slow := func(context.Context, domain.Project) (any, error) {
		n := running.Add(1)
		for {
			old := peak.Load()
			if n <= old || peak.CompareAndSwap(old, n) {
				break
			}
		}
		time.Sleep(40 * time.Millisecond)
		running.Add(-1)

		return "x", nil
	}
	for _, key := range domain.SummaryKeys {
		svc.RegisterSummary(&stubProvider{key: key, fn: slow})
	}

	if _, err := svc.Dashboard(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if got := peak.Load(); got > 3 || got < 2 {
		t.Fatalf("peak concurrency = %d, want 2..3 (the pool is shared with BearerAuth)", got)
	}
}

func TestDashboardMultiDayProjectAndInvitationLink(t *testing.T) {
	svc, repo := newTestService(t, Options{})
	// 21 Oct 2026 is day 2 of 3 in Jakarta. Created directly: the API refuses
	// to create a project whose Hari-H is already in the past.
	link := uuid.New()
	end := time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)
	p := domain.Project{
		ID: uuid.New(), UserID: uuid.New(), EventType: domain.EventWedding, Title: "Tiga hari", Timezone: "Asia/Jakarta",
		EventDate: time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC), EventEndDate: &end, Status: domain.StatusActive,
		InvitationID: &link, Metadata: map[string]any{},
	}
	repo.projects[p.ID] = p

	d, err := svc.Dashboard(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	pj := d.Project
	if pj.Phase != domain.PhaseOngoing || pj.DayNumber != 2 || pj.TotalDays != 3 || !pj.InvitationLinked || pj.EventEndDate == nil || *pj.EventEndDate != "2026-10-22" {
		t.Fatalf("project block = %+v", pj)
	}
}

func TestSummaryPayloadsUseSnakeCaseKeys(t *testing.T) {
	sd := "2027-03-28"
	cases := map[string]any{
		"finance": domain.FinanceSummary{CashDeficit: 5, LineStatus: domain.LineStatus{Settled: 1}},
		"vendors": domain.VendorSummary{CategoriesTotal: 10, CategoriesFixed: 7},
		"guests":  domain.GuestSummary{Sides: []domain.GuestSide{{Label: "Pihak Pria"}}, RSVP: domain.RSVPSummary{AttendingPax: 3}},
		"docs":    domain.AdminDocSummary{Label: "Kelengkapan Berkas KUA", Pending: []string{}},
		"rundown": domain.RundownSummary{CurrentAgenda: &domain.CurrentAgenda{Activity: "Ijab Kabul"}, NextSession: &domain.NextSession{Name: "Akad", Date: sd}},
	}
	want := map[string][]string{
		"finance": {"target_budget", "total_estimate", "total_paid", "remaining", "overpaid", "savings_target", "savings_collected", "free_cash", "cash_deficit", "line_status"},
		"vendors": {"categories_total", "categories_fixed", "skipped", "fixed_vendors", "pipeline"},
		"guests":  {"invitations", "total_pax", "sides", "sent", "rsvp"},
		"docs":    {"label", "completed", "total", "pending"},
		"rundown": {"sessions", "agendas", "completed", "in_progress", "songs", "current_agenda", "next_session"},
	}
	for name, v := range cases {
		raw, _ := json.Marshal(v)
		var m map[string]any
		_ = json.Unmarshal(raw, &m)
		for _, k := range want[name] {
			if _, ok := m[k]; !ok {
				t.Errorf("%s: missing key %q in %s", name, k, raw)
			}
		}
	}
}

// ---------- sample ----------

func TestCreateSampleIsIdempotentAndAppliesWhatIsRegistered(t *testing.T) {
	svc, repo := newTestService(t, Options{})
	user := uuid.New()
	timeline := &stubApplier{key: "timeline"}
	budget := &stubApplier{key: "budget"}
	svc.RegisterTemplateAppliers(timeline, budget)
	var overlays []string
	svc.RegisterSampleOverlays(&stubOverlay{key: "finance", seen: &overlays}, &stubOverlay{key: "guests", seen: &overlays})

	first, err := svc.CreateSample(context.Background(), user)
	if err != nil || !first.Created {
		t.Fatalf("first: created=%v err=%v", first.Created, err)
	}
	p := first.Project
	if p.Title != "Contoh: Pernikahan Sinta & Arif" || !p.IsSample || p.EventType != domain.EventWedding || p.TargetBudget != 75_000_000 {
		t.Fatalf("sample = %+v", p)
	}
	// 175 days from today in Jakarta (21 Oct 2026), like "Towards the Day: 175 Days" in the PDF.
	if got := p.EventDate.Format(domain.DateLayout); got != "2027-04-14" {
		t.Fatalf("event_date = %s, want 2027-04-14 (21 Oct 2026 + 175 days)", got)
	}
	if p.Metadata["hero_title"] != "Bismillah Nikah" || p.Metadata["bride_name"] != "Sinta Maharani" {
		t.Fatalf("metadata = %v", p.Metadata)
	}
	if len(timeline.calls) != 1 || *timeline.calls[0].TimelinePreset != "wedding_pdf_44" || !timeline.calls[0].IncludeAdicaraTasks {
		t.Fatalf("timeline applied with %+v, want wedding_pdf_44 + Adicara tasks", timeline.calls)
	}
	if len(budget.calls) != 1 {
		t.Fatalf("budget applied %d times", len(budget.calls))
	}
	if len(overlays) != 2 {
		t.Fatalf("overlays ran %v, want both", overlays)
	}

	second, err := svc.CreateSample(context.Background(), user)
	if err != nil || second.Created || second.Project.ID != p.ID {
		t.Fatalf("second call: created=%v id=%s err=%v; want the existing sample", second.Created, second.Project.ID, err)
	}
	if len(timeline.calls) != 1 || len(overlays) != 2 {
		t.Fatal("the second call must not re-apply anything")
	}
	if n := len(repo.projects); n != 1 {
		t.Fatalf("%d projects, want 1", n)
	}

	other, err := svc.CreateSample(context.Background(), uuid.New())
	if err != nil || !other.Created {
		t.Fatalf("each user gets their own sample: created=%v err=%v", other.Created, err)
	}
}

func TestCreateSampleRollsBackWhenAnOverlayFails(t *testing.T) {
	svc, repo := newTestService(t, Options{})
	svc.RegisterSampleOverlays(&stubOverlay{key: "finance", err: errors.New("bad dataset")})

	if _, err := svc.CreateSample(context.Background(), uuid.New()); err == nil {
		t.Fatal("want the overlay error")
	}
	if len(repo.projects) != 0 || len(repo.sessions) != 0 {
		t.Fatalf("a half-built sample survived: %d projects, %d session lists", len(repo.projects), len(repo.sessions))
	}
}

type stubOverlay struct {
	key  string
	err  error
	seen *[]string
}

func (o *stubOverlay) Key() string { return o.key }

func (o *stubOverlay) ApplySample(context.Context, domain.Project) error {
	if o.seen != nil {
		*o.seen = append(*o.seen, o.key)
	}

	return o.err
}
