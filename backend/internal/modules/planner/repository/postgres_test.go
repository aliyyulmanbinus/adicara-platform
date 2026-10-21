package repository

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/db"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/migrate"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

// These tests need a real PostgreSQL and WIPE the public schema of the
// database they run against (migrate.Fresh), so they only run when
// TEST_DATABASE_URL points at a database whose name ends in "_test". Run them
// with `go test -p 1`: the migrate and repository packages share that database.
// The pool is opened with db.Connect, the same way as production (pgx simple
// protocol), because result decoding differs between protocols.
func testEnv(t *testing.T) (*PostgresRepository, *TxManager, *pgxpool.Pool) {
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

	pool, err := db.Connect(context.Background(), dsn, 5)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	return NewPostgresRepository(pool), NewTxManager(pool), pool
}

func exec(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("%s: %v", strings.Join(strings.Fields(sql), " "), err)
	}
}

func scalar[T any](t *testing.T, pool *pgxpool.Pool, sql string, args ...any) T {
	t.Helper()
	var v T
	if err := pool.QueryRow(context.Background(), sql, args...).Scan(&v); err != nil {
		t.Fatalf("%s: %v", strings.Join(strings.Fields(sql), " "), err)
	}

	return v
}

func newUser(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	n := strings.ReplaceAll(id.String(), "-", "")[:12]
	exec(t, pool, `INSERT INTO users (id, email, password_hash, username) VALUES ($1, $2, 'x', $3)`, id, n+"@example.com", "u"+n)

	return id
}

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func dayp(y int, m time.Month, d int) *time.Time {
	t := day(y, m, d)

	return &t
}

func baseProject(userID uuid.UUID) domain.Project {
	return domain.Project{
		ID: uuid.New(), UserID: userID, EventType: domain.EventWedding, Title: "Pernikahan Sinta & Arif",
		EventDate: day(2027, 3, 28), PrepStartDate: day(2026, 10, 21), Timezone: "Asia/Jakarta", TargetBudget: 75_000_000,
		Status: domain.StatusActive, SkippedVendorCategories: []string{}, Metadata: map[string]any{"bride_name": "Sinta"},
	}
}

func mustInsert(t *testing.T, r *PostgresRepository, p *domain.Project) {
	t.Helper()
	ok, err := r.InsertProject(context.Background(), p, nil)
	if err != nil || !ok {
		t.Fatalf("InsertProject: inserted=%v err=%v", ok, err)
	}
}

func seedSessions(t *testing.T, r *PostgresRepository, p domain.Project) map[string]domain.Session {
	t.Helper()
	out := map[string]domain.Session{}
	add := func(s domain.Session) {
		t.Helper()
		if err := r.InsertSession(context.Background(), &s); err != nil {
			t.Fatalf("InsertSession %s: %v", s.ScopeKey, err)
		}
		out[s.ScopeKey] = s
	}
	add(domain.Session{ID: uuid.New(), ProjectID: p.ID, ScopeKey: "all", Name: "Umum / Semua Sesi", Icon: "📋", IsSystem: true})
	add(domain.Session{ID: uuid.New(), ProjectID: p.ID, ScopeKey: "lamaran", Name: "Lamaran", Icon: "💍", SortOrder: 1})
	add(domain.Session{ID: uuid.New(), ProjectID: p.ID, ScopeKey: "akad", Name: "Akad Nikah", Icon: "📜", SortOrder: 2, SessionDate: &p.EventDate})
	add(domain.Session{ID: uuid.New(), ProjectID: p.ID, ScopeKey: "resepsi", Name: "Resepsi", Icon: "🏛️", SortOrder: 3, SessionDate: &p.EventDate})

	return out
}

// ---------- projects ----------

func TestProjectRoundTripThroughSimpleProtocol(t *testing.T) {
	r, _, pool := testEnv(t)
	ctx := context.Background()
	owner := newUser(t, pool)

	p := baseProject(owner)
	p.EventEndDate = dayp(2027, 3, 30)
	p.SavingsTarget = new(int64)
	*p.SavingsTarget = 60_000_000
	p.SkippedVendorCategories = []string{"entertainment", "mc"}
	p.Metadata = map[string]any{"bride_name": "Sinta", "groom_name": "Arif", "venue_name": "Sasana \"Kriya\" ✨", "age": int64(7)}
	token := "client-token-1"
	ok, err := r.InsertProject(ctx, &p, &token)
	if err != nil || !ok {
		t.Fatalf("insert: %v %v", ok, err)
	}
	if p.CreatedAt.IsZero() || p.UpdatedAt.IsZero() {
		t.Fatal("InsertProject must fill created_at and updated_at")
	}

	got, err := r.ProjectForUser(ctx, p.ID, owner)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != p.Title || !got.EventDate.Equal(p.EventDate) || got.EventEndDate == nil || !got.EventEndDate.Equal(*p.EventEndDate) ||
		!got.PrepStartDate.Equal(p.PrepStartDate) || got.Timezone != "Asia/Jakarta" || got.TargetBudget != 75_000_000 ||
		got.SavingsTarget == nil || *got.SavingsTarget != 60_000_000 || got.Status != domain.StatusActive || got.IsSample ||
		strings.Join(got.SkippedVendorCategories, ",") != "entertainment,mc" || got.InvitationID != nil {
		t.Fatalf("round trip lost data: %+v", got)
	}
	if got.Metadata["venue_name"] != `Sasana "Kriya" ✨` || got.Metadata["age"] != float64(7) || got.Metadata["groom_name"] != "Arif" {
		t.Fatalf("metadata round trip: %v", got.Metadata)
	}

	// Somebody else's project is indistinguishable from a missing one.
	if _, err := r.ProjectForUser(ctx, p.ID, newUser(t, pool)); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("foreign project: %v, want ErrNotFound", err)
	}
	if _, err := r.ProjectForUser(ctx, uuid.New(), owner); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("unknown project: %v, want ErrNotFound", err)
	}

	// client_token: the second insert is a no-op that reports inserted=false.
	dup := baseProject(owner)
	ok, err = r.InsertProject(ctx, &dup, &token)
	if err != nil || ok {
		t.Fatalf("same client_token: inserted=%v err=%v, want false/nil", ok, err)
	}
	byToken, err := r.ProjectByClientToken(ctx, owner, token)
	if err != nil || byToken.ID != p.ID {
		t.Fatalf("ProjectByClientToken: %v %v", byToken.ID, err)
	}
	if _, err := r.ProjectByClientToken(ctx, newUser(t, pool), token); !errors.Is(err, domain.ErrNotFound) {
		t.Fatal("tokens are per user")
	}

	// Update writes every editable column and can clear the nullable ones.
	got.Title, got.EventEndDate, got.SavingsTarget = "Akad Sinta & Arif", nil, nil
	got.SkippedVendorCategories, got.Metadata = []string{}, map[string]any{"bride_name": "Sinta", "groom_name": "Arif"}
	got.EventDate, got.Timezone = day(2027, 4, 4), "Pacific/Auckland"
	if err := r.UpdateProject(ctx, &got); err != nil {
		t.Fatal(err)
	}
	after, _ := r.ProjectForUser(ctx, p.ID, owner)
	if after.Title != "Akad Sinta & Arif" || after.EventEndDate != nil || after.SavingsTarget != nil || len(after.SkippedVendorCategories) != 0 ||
		!after.EventDate.Equal(day(2027, 4, 4)) || after.Timezone != "Pacific/Auckland" || len(after.Metadata) != 2 {
		t.Fatalf("update result: %+v", after)
	}
	if !after.UpdatedAt.After(p.UpdatedAt) {
		t.Error("updated_at must move forward")
	}

	if err := r.SetProjectStatus(ctx, p.ID, domain.StatusArchived); err != nil {
		t.Fatal(err)
	}
	if after, _ = r.ProjectForUser(ctx, p.ID, owner); after.Status != domain.StatusArchived {
		t.Fatalf("status = %s", after.Status)
	}
	if err := r.SetProjectStatus(ctx, uuid.New(), domain.StatusActive); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("status of unknown project: %v", err)
	}

	if err := r.DeleteProject(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	if err := r.DeleteProject(ctx, p.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("second delete: %v", err)
	}
}

func TestSampleIsUniquePerUserAndNotCounted(t *testing.T) {
	r, _, pool := testEnv(t)
	ctx := context.Background()
	owner := newUser(t, pool)

	mk := func(sample bool) domain.Project {
		p := baseProject(owner)
		p.IsSample = sample

		return p
	}
	s1, s2 := mk(true), mk(true)
	if ok, err := r.InsertProject(ctx, &s1, nil); err != nil || !ok {
		t.Fatalf("first sample: %v %v", ok, err)
	}
	if ok, err := r.InsertProject(ctx, &s2, nil); err != nil || ok {
		t.Fatalf("second sample: inserted=%v err=%v, want a silent no-op", ok, err)
	}
	real := mk(false)
	mustInsert(t, r, &real)

	if n, _ := r.CountProjects(ctx, owner); n != 1 {
		t.Fatalf("CountProjects = %d, want 1 (the sample is not counted)", n)
	}
	got, err := r.SampleProject(ctx, owner)
	if err != nil || got.ID != s1.ID {
		t.Fatalf("SampleProject: %v %v", got.ID, err)
	}
	if _, err := r.SampleProject(ctx, newUser(t, pool)); !errors.Is(err, domain.ErrNotFound) {
		t.Fatal("another user has no sample")
	}
}

func TestProjectConstraintsRejectBadRows(t *testing.T) {
	r, _, pool := testEnv(t)
	owner := newUser(t, pool)

	cases := map[string]func(*domain.Project){
		"empty title":                func(p *domain.Project) { p.Title = "" },
		"title with spaces at end":   func(p *domain.Project) { p.Title = "Nikah " },
		"title with newline":         func(p *domain.Project) { p.Title = "Ni\nkah" },
		"title of 256 characters":    func(p *domain.Project) { p.Title = strings.Repeat("é", 256) },
		"lowercase zone":             func(p *domain.Project) { p.Timezone = "asia/jakarta" },
		"offset zone":                func(p *domain.Project) { p.Timezone = "UTC+7" },
		"Local zone":                 func(p *domain.Project) { p.Timezone = "Local" },
		"Etc zone":                   func(p *domain.Project) { p.Timezone = "Etc/GMT-7" },
		"negative budget":            func(p *domain.Project) { p.TargetBudget = -1 },
		"budget above 10^13":         func(p *domain.Project) { p.TargetBudget = 10_000_000_000_001 },
		"unknown event type":         func(p *domain.Project) { p.EventType = "party" },
		"end before event":           func(p *domain.Project) { p.EventEndDate = dayp(2027, 3, 27) },
		"prep after event":           func(p *domain.Project) { p.PrepStartDate = day(2027, 3, 29) },
		"unknown status":             func(p *domain.Project) { p.Status = "paused" },
		"savings target above 10^13": func(p *domain.Project) { v := int64(10_000_000_000_001); p.SavingsTarget = &v },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			p := baseProject(owner)
			mutate(&p)
			if _, err := r.InsertProject(context.Background(), &p, nil); err == nil {
				t.Fatal("the row must be rejected by a CHECK")
			}
		})
	}

	p := baseProject(owner)
	p.Title = strings.Repeat("é", 255)
	p.TargetBudget = 10_000_000_000_000
	mustInsert(t, r, &p) // the boundaries themselves are fine
}

func TestUnknownTimezoneIsCaughtByTheTrigger(t *testing.T) {
	r, _, pool := testEnv(t)
	owner := newUser(t, pool)

	// Passes the CHECK (canonical shape) but PostgreSQL has no such zone: one
	// such row would break every Hub query of the user.
	for _, tz := range []string{"Asia/Nowhere", "Foo/Bar"} {
		p := baseProject(owner)
		p.Timezone = tz
		_, err := r.InsertProject(context.Background(), &p, nil)
		if tz == "Foo/Bar" {
			if err == nil {
				t.Errorf("%s must be rejected", tz) // not even a known region prefix: the CHECK says no
			}
			continue
		}
		if !errors.Is(err, domain.ErrInvalidTimezone) {
			t.Errorf("%s: err = %v, want ErrInvalidTimezone", tz, err)
		}
	}

	ok := baseProject(owner)
	mustInsert(t, r, &ok)
	ok.Timezone = "Asia/Nowhere"
	if err := r.UpdateProject(context.Background(), &ok); !errors.Is(err, domain.ErrInvalidTimezone) {
		t.Fatalf("UPDATE to an unknown zone: %v, want ErrInvalidTimezone", err)
	}
}

func TestEventTypeCheckMatchesTheRegistry(t *testing.T) {
	_, _, pool := testEnv(t)

	def := scalar[string](t, pool, `
		SELECT pg_get_constraintdef(c.oid) FROM pg_constraint c
		WHERE c.conrelid = 'planner_projects'::regclass AND pg_get_constraintdef(c.oid) LIKE '%event_type%'`)
	var inDB []string
	for _, m := range regexp.MustCompile(`'([a-z]+)'::text`).FindAllStringSubmatch(def, -1) {
		inDB = append(inDB, m[1])
	}
	inRegistry := domain.DefaultRegistry().Keys()
	sort.Strings(inDB)
	sort.Strings(inRegistry)
	if strings.Join(inDB, ",") != strings.Join(inRegistry, ",") {
		t.Fatalf("CHECK(event_type) lists %v but the registry has %v: add a migration when adding an event type", inDB, inRegistry)
	}
}

// ---------- hub ----------

func TestHubPhaseMatchesComputePhaseAtEveryHalfHour(t *testing.T) {
	r, _, pool := testEnv(t)
	ctx := context.Background()
	owner := newUser(t, pool)

	type fixture struct {
		p   domain.Project
		loc *time.Location
	}
	var fixtures []fixture
	for _, tz := range []string{"Asia/Jakarta", "UTC", "America/New_York", "Pacific/Auckland", "Asia/Kolkata"} {
		for _, multiDay := range []bool{false, true} {
			p := baseProject(owner)
			p.Timezone = tz
			if multiDay {
				p.EventEndDate = dayp(2027, 3, 30)
			}
			mustInsert(t, r, &p)
			loc, err := domain.LoadLocation(tz)
			if err != nil {
				t.Fatal(err)
			}
			fixtures = append(fixtures, fixture{p, loc})
		}
	}

	checked := 0
	for now := time.Date(2027, 3, 26, 0, 0, 0, 0, time.UTC); now.Before(time.Date(2027, 4, 1, 0, 0, 0, 0, time.UTC)); now = now.Add(30 * time.Minute) {
		cards, err := r.ListCards(ctx, owner, "all", now)
		if err != nil {
			t.Fatal(err)
		}
		lite, err := r.ListLite(ctx, owner, "all", now)
		if err != nil {
			t.Fatal(err)
		}
		byID := map[uuid.UUID]domain.ProjectCardResponse{}
		for _, c := range cards {
			byID[c.ID] = c
		}
		liteByID := map[uuid.UUID]domain.ProjectLiteResponse{}
		for _, l := range lite {
			liteByID[l.ID] = l
		}

		for _, f := range fixtures {
			want := domain.ComputePhase(now, f.loc, f.p.EventDate, f.p.EventEndDate)
			got := byID[f.p.ID]
			if got.PhaseInfo != want {
				t.Fatalf("%s at %s: SQL phase %+v != Go phase %+v", f.p.Timezone, now.Format(time.RFC3339), got.PhaseInfo, want)
			}
			if l := liteByID[f.p.ID]; l.Phase != want.Phase || l.DaysRemaining != want.DaysRemaining {
				t.Fatalf("%s at %s: lite %v/%d != %v/%d", f.p.Timezone, now.Format(time.RFC3339), l.Phase, l.DaysRemaining, want.Phase, want.DaysRemaining)
			}
			checked++
		}
	}
	if checked < 1000 {
		t.Fatalf("only %d comparisons ran", checked)
	}
}

func TestHubAtMidnightBoundaryInJakarta(t *testing.T) {
	r, _, pool := testEnv(t)
	owner := newUser(t, pool)
	p := baseProject(owner)
	mustInsert(t, r, &p)
	wib, _ := time.LoadLocation("Asia/Jakarta")

	// 6 Oct 2026 -> 28 Mar 2027 is 173 days at every hour of the Jakarta day.
	for _, hm := range [][2]int{{0, 0}, {6, 59}, {7, 0}, {23, 59}} {
		now := time.Date(2026, 10, 6, hm[0], hm[1], 0, 0, wib)
		cards, err := r.ListCards(context.Background(), owner, "all", now)
		if err != nil {
			t.Fatal(err)
		}
		if cards[0].DaysRemaining != 173 {
			t.Errorf("%02d:%02d WIB: days_remaining = %d, want 173", hm[0], hm[1], cards[0].DaysRemaining)
		}
	}
}

func TestHubOrderFilterAndAggregates(t *testing.T) {
	r, tx, pool := testEnv(t)
	ctx := context.Background()
	owner := newUser(t, pool)
	now := time.Date(2026, 10, 21, 6, 47, 0, 0, time.FixedZone("WIB", 7*3600))

	mk := func(title, date string, status domain.ProjectStatus) domain.Project {
		p := baseProject(owner)
		p.Title = title
		d, _ := time.Parse(domain.DateLayout, date)
		p.EventDate, p.Status = d, status
		p.PrepStartDate = day(2026, 1, 1) // the CHECK needs prep_start_date <= event_date
		mustInsert(t, r, &p)
		if status != domain.StatusActive {
			if err := r.SetProjectStatus(ctx, p.ID, status); err != nil {
				t.Fatal(err)
			}
		}

		return p
	}
	far := mk("far", "2027-08-01", domain.StatusActive)
	near := mk("near", "2027-01-01", domain.StatusActive)
	mk("done-new", "2026-12-01", domain.StatusCompleted)
	mk("done-old", "2026-11-01", domain.StatusCompleted)
	mk("arch", "2026-10-01", domain.StatusArchived)
	_ = far

	titles := func(status string) string {
		cards, err := r.ListCards(ctx, owner, status, now)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, c := range cards {
			out = append(out, c.Title)
		}

		return strings.Join(out, ",")
	}
	if got := titles("all"); got != "near,far,done-new,done-old,arch" {
		t.Fatalf("order = %s, want active by nearest date, then completed and archived latest first", got)
	}
	if got := titles("completed"); got != "done-new,done-old" {
		t.Fatalf("completed filter = %s", got)
	}
	if got := titles("active"); got != "near,far" {
		t.Fatalf("active filter = %s", got)
	}

	// Aggregates from the other modules' tables.
	sessions := seedSessions(t, r, near)
	_ = sessions
	exec(t, pool, `INSERT INTO planner_budget_items (id, project_id, sub_event_scope, category, item_name, initial_estimate, dp_amount, term_1_amount)
		VALUES ($1,$2,'resepsi','Katering','Catering',38200000,10000000,2000000), ($3,$2,'akad','Legalitas','Biaya KUA',600000,600000,0)`,
		uuid.New(), near.ID, uuid.New())
	for i, st := range []string{"selesai", "proses", "belum"} {
		exec(t, pool, `INSERT INTO planner_tasks (id, project_id, title, start_date, end_date, status, sort_order) VALUES ($1,$2,$3,'2026-11-01','2026-11-02',$4,$5)`,
			uuid.New(), near.ID, fmt.Sprintf("Tugas %d", i), st, i)
	}
	for _, pax := range []int{2, 3} {
		exec(t, pool, `INSERT INTO planner_guests (id, project_id, name, pax_count, qr_code_token) VALUES ($1,$2,'Tamu',$3,$4)`,
			uuid.New(), near.ID, pax, uuid.NewString())
	}
	_ = tx

	cards, err := r.ListCards(ctx, owner, "active", now)
	if err != nil {
		t.Fatal(err)
	}
	c := cards[0]
	if c.Title != "near" || c.TotalEstimate != 38_800_000 || c.TotalPaid != 12_600_000 || c.TotalTasks != 3 || c.CompletedTasks != 1 || c.GuestPaxCount != 5 {
		t.Fatalf("aggregates = est %d paid %d tasks %d/%d pax %d", c.TotalEstimate, c.TotalPaid, c.CompletedTasks, c.TotalTasks, c.GuestPaxCount)
	}
	if empty := cards[1]; empty.TotalEstimate != 0 || empty.TotalTasks != 0 || empty.GuestPaxCount != 0 {
		t.Fatalf("a project without rows must show zeros, got %+v", empty)
	}
}

// ---------- sessions ----------

func TestSessionsInsertListUpdateReorder(t *testing.T) {
	r, _, pool := testEnv(t)
	ctx := context.Background()
	p := baseProject(newUser(t, pool))
	mustInsert(t, r, &p)
	s := seedSessions(t, r, p)

	// Unique violations map to distinct sentinels.
	dupName := domain.Session{ID: uuid.New(), ProjectID: p.ID, ScopeKey: "x1", Name: "RESEPSI", Icon: "📋", SortOrder: 9}
	if err := r.InsertSession(ctx, &dupName); !errors.Is(err, domain.ErrSessionNameTaken) {
		t.Fatalf("duplicate name (case-insensitive): %v", err)
	}
	dupKey := domain.Session{ID: uuid.New(), ProjectID: p.ID, ScopeKey: "akad", Name: "Akad lain", Icon: "📋", SortOrder: 9}
	if err := r.InsertSession(ctx, &dupKey); !errors.Is(err, domain.ErrScopeKeyTaken) {
		t.Fatalf("duplicate scope_key: %v", err)
	}
	notSystem := domain.Session{ID: uuid.New(), ProjectID: p.ID, ScopeKey: "all", Name: "Palsu", Icon: "📋"}
	if err := r.InsertSession(ctx, &notSystem); err == nil {
		t.Fatal("scope_key 'all' must be the system row (CHECK)")
	}
	// The same name in another project is fine.
	other := baseProject(p.UserID)
	mustInsert(t, r, &other)
	if err := r.InsertSession(ctx, &domain.Session{ID: uuid.New(), ProjectID: other.ID, ScopeKey: "resepsi", Name: "Resepsi", Icon: "📋"}); err != nil {
		t.Fatalf("names are unique per project only: %v", err)
	}

	// Times come back as HH:MM, not 07:00:00.
	lam := s["lamaran"]
	lam.Name, lam.Icon, lam.SessionDate = "Lamaran Keluarga", "🤝", dayp(2027, 2, 14)
	lam.StartTime, lam.EndTime, lam.LocationName = ptr("07:05"), ptr("09:00"), ptr("Rumah Mempelai")
	if err := r.UpdateSession(ctx, &lam); err != nil {
		t.Fatal(err)
	}
	got, err := r.SessionByID(ctx, p.ID, lam.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Lamaran Keluarga" || got.StartTime == nil || *got.StartTime != "07:05" || *got.EndTime != "09:00" || got.SessionDate == nil ||
		!got.SessionDate.Equal(day(2027, 2, 14)) || *got.LocationName != "Rumah Mempelai" || got.ScopeKey != "lamaran" {
		t.Fatalf("session after update: %+v", got)
	}
	lam.StartTime, lam.EndTime, lam.LocationName, lam.SessionDate = nil, nil, nil, nil
	if err := r.UpdateSession(ctx, &lam); err != nil {
		t.Fatal(err)
	}
	got, _ = r.SessionByID(ctx, p.ID, lam.ID)
	if got.StartTime != nil || got.EndTime != nil || got.LocationName != nil || got.SessionDate != nil {
		t.Fatalf("clearing optional fields failed: %+v", got)
	}

	// The system session cannot be updated, another project's session cannot be read.
	allRow := s["all"]
	allRow.Name = "Semua"
	if err := r.UpdateSession(ctx, &allRow); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("updating the system session: %v", err)
	}
	if _, err := r.SessionByID(ctx, other.ID, lam.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("a session of another project: %v", err)
	}

	if err := r.ReorderSessions(ctx, p.ID, []uuid.UUID{s["resepsi"].ID, s["lamaran"].ID, s["akad"].ID}); err != nil {
		t.Fatal(err)
	}
	list, _ := r.ListSessions(ctx, p.ID)
	var order []string
	for _, x := range list {
		order = append(order, fmt.Sprintf("%s=%d", x.ScopeKey, x.SortOrder))
	}
	if got := strings.Join(order, ","); got != "all=0,resepsi=1,lamaran=2,akad=3" {
		t.Fatalf("order after reorder = %s", got)
	}
}

func ptr[T any](v T) *T { return &v }

// goldenSessionFixture builds the case the move algorithm exists for: Lamaran
// and Resepsi both have a 'catering' budget item and scenario item, plus one
// row of every other kind that points at a session.
type golden struct {
	project  domain.Project
	sessions map[string]domain.Session
}

func newGolden(t *testing.T, r *PostgresRepository, pool *pgxpool.Pool) golden {
	t.Helper()
	p := baseProject(newUser(t, pool))
	mustInsert(t, r, &p)
	s := seedSessions(t, r, p)

	budget := func(scope, key, name string, estimate, dp int64) {
		var k any
		if key != "" {
			k = key
		}
		exec(t, pool, `INSERT INTO planner_budget_items (id, project_id, sub_event_scope, category, item_name, item_key, initial_estimate, dp_amount)
			VALUES ($1,$2,$3,'Kategori',$4,$5,$6,$7)`, uuid.New(), p.ID, scope, name, k, estimate, dp)
	}
	budget("lamaran", "catering", "Catering Lamaran", 2_400_000, 1_000_000)
	budget("lamaran", "dekor", "Dekor Lamaran", 4_750_000, 4_750_000)
	budget("lamaran", "", "Pos kustom", 100_000, 0)
	budget("resepsi", "catering", "Catering Resepsi", 38_200_000, 10_000_000)
	budget("akad", "biaya_kua", "Biaya KUA", 600_000, 600_000)

	sc1, sc2 := uuid.New(), uuid.New()
	exec(t, pool, `INSERT INTO planner_budget_scenarios (id, project_id, name, total_target, is_active, active_sub_events) VALUES ($1,$2,'75 jt',75000000,true,'{lamaran,resepsi}')`, sc1, p.ID)
	exec(t, pool, `INSERT INTO planner_budget_scenarios (id, project_id, name, total_target, active_sub_events) VALUES ($1,$2,'50 jt',50000000,'{lamaran}')`, sc2, p.ID)
	exec(t, pool, `INSERT INTO planner_budget_scenarios (id, project_id, name, total_target) VALUES ($1,$2,'30 jt',30000000)`, uuid.New(), p.ID)
	for _, sc := range []uuid.UUID{sc1, sc2} {
		for _, scope := range []string{"lamaran", "resepsi"} {
			exec(t, pool, `INSERT INTO planner_budget_scenario_items (id, scenario_id, project_id, sub_event_scope, item_key, category, item_name, cost_amount)
				VALUES ($1,$2,$3,$4,'catering','Katering','Catering',1000000)`, uuid.New(), sc, p.ID, scope)
		}
	}

	exec(t, pool, `INSERT INTO planner_tasks (id, project_id, title, sub_event_scope, start_date, end_date) VALUES ($1,$2,'Tugas lamaran','lamaran','2026-11-01','2026-11-02')`, uuid.New(), p.ID)
	exec(t, pool, `INSERT INTO planner_vendors (id, project_id, sub_event_scope, category_key, category_label, vendor_name) VALUES ($1,$2,'lamaran','catering','Catering','Vendor X')`, uuid.New(), p.ID)
	exec(t, pool, `INSERT INTO planner_shopping_items (id, project_id, sub_event_scope, category, item_name) VALUES ($1,$2,'lamaran','Seserahan','Mukena')`, uuid.New(), p.ID)
	exec(t, pool, `INSERT INTO planner_guests (id, project_id, name, sub_event_scope, qr_code_token) VALUES ($1,$2,'Tamu Lamaran','lamaran',$3)`, uuid.New(), p.ID, uuid.NewString())

	return golden{p, s}
}

func (g golden) totals(t *testing.T, pool *pgxpool.Pool) (estimate, paid int64) {
	t.Helper()
	err := pool.QueryRow(context.Background(), `SELECT sum(estimate)::bigint, sum(paid)::bigint FROM planner_budget_lines WHERE project_id=$1`, g.project.ID).Scan(&estimate, &paid)
	if err != nil {
		t.Fatal(err)
	}

	return
}

func TestSessionUsageCountsEveryKindOfRow(t *testing.T) {
	r, _, pool := testEnv(t)
	g := newGolden(t, r, pool)
	exec(t, pool, `INSERT INTO planner_rundowns (id, project_id, session_id, start_time, activity_name) VALUES ($1,$2,$3,'09:00','Pembukaan'), ($4,$2,$3,'09:30','Doa')`,
		uuid.New(), g.project.ID, g.sessions["resepsi"].ID, uuid.New())

	list, err := r.ListSessionsWithUsage(context.Background(), g.project.ID)
	if err != nil {
		t.Fatal(err)
	}
	usage := map[string]domain.SessionUsage{}
	for _, s := range list {
		usage[s.ScopeKey] = s.Usage
	}
	want := domain.SessionUsage{Tasks: 1, Vendors: 1, BudgetItems: 3, ScenarioItems: 2, ShoppingItems: 1, Guests: 1}
	if usage["lamaran"] != want {
		t.Fatalf("lamaran usage = %+v, want %+v (scenario items count too)", usage["lamaran"], want)
	}
	if u := usage["resepsi"]; u.Agendas != 2 || u.BudgetItems != 1 || u.ScenarioItems != 2 {
		t.Fatalf("resepsi usage = %+v", u)
	}

	one, err := r.SessionUsage(context.Background(), g.project.ID, g.sessions["lamaran"])
	if err != nil || one != want {
		t.Fatalf("SessionUsage = %+v %v, want %+v", one, err, want)
	}
}

func TestMoveSessionDataKeepsTotalsAndNeutralisesKeyCollisions(t *testing.T) {
	r, tx, pool := testEnv(t)
	ctx := context.Background()
	g := newGolden(t, r, pool)
	estimateBefore, paidBefore := g.totals(t, pool)

	// A plain UPDATE of the scope would fail on planner_budget_items_key_idx
	// (23505): both sessions have a 'catering' item.
	err := tx.InTx(ctx, func(ctx context.Context) error {
		if _, err := r.LockProject(ctx, g.project.ID); err != nil {
			return err
		}

		return r.MoveSessionData(ctx, g.project.ID, "lamaran", "resepsi")
	})
	if err != nil {
		t.Fatalf("move: %v", err)
	}

	pid := g.project.ID
	if n := scalar[int](t, pool, `SELECT count(*) FROM planner_event_sessions WHERE project_id=$1 AND scope_key='lamaran'`, pid); n != 0 {
		t.Error("the session must be deleted")
	}
	for _, table := range []string{"planner_tasks", "planner_vendors", "planner_shopping_items", "planner_guests", "planner_budget_items", "planner_budget_scenario_items"} {
		if n := scalar[int](t, pool, `SELECT count(*) FROM `+table+` WHERE project_id=$1 AND sub_event_scope='lamaran'`, pid); n != 0 {
			t.Errorf("%s still points at the deleted session (%d rows)", table, n)
		}
	}
	if n := scalar[int](t, pool, `SELECT count(*) FROM planner_budget_items WHERE project_id=$1 AND sub_event_scope='resepsi'`, pid); n != 4 {
		t.Errorf("resepsi has %d budget items, want its own 1 + 3 moved", n)
	}
	// Exactly the colliding item lost its key; the others kept theirs.
	if n := scalar[int](t, pool, `SELECT count(*) FROM planner_budget_items WHERE project_id=$1 AND sub_event_scope='resepsi' AND item_key='catering'`, pid); n != 1 {
		t.Errorf("resepsi has %d items keyed 'catering', want 1 (the pre-existing one)", n)
	}
	if k := scalar[*string](t, pool, `SELECT item_key FROM planner_budget_items WHERE item_name='Catering Lamaran'`); k != nil {
		t.Errorf("the moved catering item kept key %q, want NULL", *k)
	}
	if k := scalar[*string](t, pool, `SELECT item_key FROM planner_budget_items WHERE item_name='Dekor Lamaran'`); k == nil || *k != "dekor" {
		t.Errorf("a non-colliding item must keep its key, got %v", k)
	}
	if n := scalar[int](t, pool, `SELECT count(*) FROM planner_budget_scenario_items WHERE project_id=$1 AND sub_event_scope='resepsi' AND item_key IS NULL`, pid); n != 2 {
		t.Errorf("%d scenario items lost their key, want 2 (one per scenario)", n)
	}

	estimateAfter, paidAfter := g.totals(t, pool)
	if estimateAfter != estimateBefore || paidAfter != paidBefore || estimateAfter != 46_050_000 {
		t.Fatalf("totals changed: estimate %d -> %d, paid %d -> %d", estimateBefore, estimateAfter, paidBefore, paidAfter)
	}

	// active_sub_events had no FK; the removed session must be gone from it.
	got := map[string]string{}
	rows, _ := pool.Query(ctx, `SELECT name, active_sub_events::text FROM planner_budget_scenarios WHERE project_id=$1`, pid)
	for rows.Next() {
		var name, arr string
		_ = rows.Scan(&name, &arr)
		got[name] = arr
	}
	rows.Close()
	if got["75 jt"] != "{resepsi}" || got["50 jt"] != "{resepsi}" || got["30 jt"] != "{}" {
		t.Fatalf("active_sub_events = %v, want {resepsi}, {resepsi}, {}", got)
	}
}

func TestMoveSessionDataToAllAndRemovesAgendas(t *testing.T) {
	r, tx, pool := testEnv(t)
	ctx := context.Background()
	g := newGolden(t, r, pool)
	exec(t, pool, `INSERT INTO planner_rundowns (id, project_id, session_id, start_time, activity_name) VALUES ($1,$2,$3,'09:00','Resepsi dibuka')`,
		uuid.New(), g.project.ID, g.sessions["resepsi"].ID)

	err := tx.InTx(ctx, func(ctx context.Context) error {
		return r.MoveSessionData(ctx, g.project.ID, "lamaran", domain.AllScope)
	})
	if err != nil {
		t.Fatal(err)
	}
	if n := scalar[int](t, pool, `SELECT count(*) FROM planner_budget_items WHERE project_id=$1 AND sub_event_scope='all'`, g.project.ID); n != 3 {
		t.Errorf("%d budget items moved to 'all', want 3", n)
	}
	// 'all' never enters active_sub_events: empty already means every session.
	if n := scalar[int](t, pool, `SELECT count(*) FROM planner_budget_scenarios WHERE project_id=$1 AND 'all' = ANY(active_sub_events)`, g.project.ID); n != 0 {
		t.Error("'all' must not be written into active_sub_events")
	}

	// Deleting a session cascades its agendas.
	err = tx.InTx(ctx, func(ctx context.Context) error {
		return r.MoveSessionData(ctx, g.project.ID, "resepsi", "akad")
	})
	if err != nil {
		t.Fatal(err)
	}
	if n := scalar[int](t, pool, `SELECT count(*) FROM planner_rundowns WHERE project_id=$1`, g.project.ID); n != 0 {
		t.Errorf("%d agendas survived the deletion of their session", n)
	}
}

func TestDeletingASessionThatIsStillUsedFailsAtCommit(t *testing.T) {
	r, tx, pool := testEnv(t)
	g := newGolden(t, r, pool)

	err := tx.InTx(context.Background(), func(ctx context.Context) error {
		_, err := db.Querier(ctx, pool).Exec(ctx, `DELETE FROM planner_event_sessions WHERE project_id=$1 AND scope_key='lamaran'`, g.project.ID)
		return err // the statement itself succeeds: the foreign keys are deferred
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("err = %v, want ErrConflict from the deferred foreign key at COMMIT", err)
	}
	if n := scalar[int](t, pool, `SELECT count(*) FROM planner_event_sessions WHERE project_id=$1 AND scope_key='lamaran'`, g.project.ID); n != 1 {
		t.Fatal("the session must still exist")
	}
}

func TestDeleteProjectWithEverythingAttachedSucceeds(t *testing.T) {
	r, _, pool := testEnv(t)
	g := newGolden(t, r, pool)
	exec(t, pool, `INSERT INTO planner_rundowns (id, project_id, session_id, start_time, activity_name) VALUES ($1,$2,$3,'09:00','x')`,
		uuid.New(), g.project.ID, g.sessions["resepsi"].ID)
	contributor := uuid.New()
	exec(t, pool, `INSERT INTO planner_contributors (id, project_id, name) VALUES ($1,$2,'Sinta')`, contributor, g.project.ID)
	exec(t, pool, `INSERT INTO planner_savings_entries (id, project_id, contributor_id, entry_date, amount) VALUES ($1,$2,$3,'2026-02-01',1000000)`, uuid.New(), g.project.ID, contributor)

	// With a RESTRICT/NO ACTION foreign key into the sessions this fails whenever a scenario exists.
	if err := r.DeleteProject(context.Background(), g.project.ID); err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}
	for _, table := range []string{"planner_event_sessions", "planner_tasks", "planner_vendors", "planner_budget_items", "planner_budget_scenarios",
		"planner_budget_scenario_items", "planner_shopping_items", "planner_guests", "planner_rundowns", "planner_contributors", "planner_savings_entries"} {
		if n := scalar[int](t, pool, `SELECT count(*) FROM `+table); n != 0 {
			t.Errorf("%s keeps %d orphan rows", table, n)
		}
	}
}

func TestDeletingTheUserDeletesTheirProjects(t *testing.T) {
	r, _, pool := testEnv(t)
	g := newGolden(t, r, pool)

	exec(t, pool, `DELETE FROM users WHERE id=$1`, g.project.UserID)
	if n := scalar[int](t, pool, `SELECT count(*) FROM planner_projects`); n != 0 {
		t.Fatalf("%d projects survived their user", n)
	}
}

// ---------- tasks and session dates (date change) ----------

func TestShiftTemplateTasksAndSessionDates(t *testing.T) {
	r, _, pool := testEnv(t)
	ctx := context.Background()
	p := baseProject(newUser(t, pool))
	mustInsert(t, r, &p)
	seedSessions(t, r, p)

	task := func(title, status string, tpl any, start, end string) {
		exec(t, pool, `INSERT INTO planner_tasks (id, project_id, title, start_date, end_date, status, template_key) VALUES ($1,$2,$3,$4::date,$5::date,$6,$7)`,
			uuid.New(), p.ID, title, start, end, status, tpl)
	}
	task("template, open", "belum", "wedding.a", "2027-01-10", "2027-01-12")
	task("template, running", "proses", "wedding.b", "2027-02-01", "2027-02-02")
	task("template, done", "selesai", "wedding.c", "2027-01-01", "2027-01-02")
	task("custom, open", "belum", nil, "2027-01-10", "2027-01-12")

	if n, err := r.CountShiftableTasks(ctx, p.ID); err != nil || n != 2 {
		t.Fatalf("shiftable = %d %v, want 2 (template tasks that are not done)", n, err)
	}
	shifted, err := r.ShiftTemplateTasks(ctx, p.ID, -7)
	if err != nil || shifted != 2 {
		t.Fatalf("shifted = %d %v", shifted, err)
	}
	dates := func(title string) string {
		return scalar[string](t, pool, `SELECT to_char(start_date,'YYYY-MM-DD') || '..' || to_char(end_date,'YYYY-MM-DD') FROM planner_tasks WHERE title=$1`, title)
	}
	if got := dates("template, open"); got != "2027-01-03..2027-01-05" {
		t.Errorf("template, open = %s", got)
	}
	if got := dates("template, running"); got != "2027-01-25..2027-01-26" {
		t.Errorf("template, running = %s", got)
	}
	if got := dates("template, done"); got != "2027-01-01..2027-01-02" {
		t.Errorf("a finished task must not move: %s", got)
	}
	if got := dates("custom, open"); got != "2027-01-10..2027-01-12" {
		t.Errorf("a custom task must not move: %s", got)
	}

	if n, _ := r.CountOverdueTemplateTasks(ctx, p.ID, day(2027, 1, 4)); n != 0 {
		// open ends 01-05, running ends 01-26: neither is overdue on 01-04
		t.Errorf("overdue on 4 Jan = %d, want 0", n)
	}
	if n, _ := r.CountOverdueTemplateTasks(ctx, p.ID, day(2027, 1, 6)); n != 1 {
		t.Errorf("overdue on 6 Jan = %d, want 1", n)
	}

	// Sessions dated on the old Hari-H follow it; the Lamaran on another date stays.
	exec(t, pool, `UPDATE planner_event_sessions SET session_date='2027-02-14' WHERE project_id=$1 AND scope_key='lamaran'`, p.ID)
	if n, _ := r.CountSessionsOnOtherDates(ctx, p.ID, day(2027, 3, 28)); n != 1 {
		t.Errorf("sessions on other dates = %d, want 1", n)
	}
	if err := r.ShiftSessionDates(ctx, p.ID, day(2027, 3, 28), day(2027, 4, 4)); err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	list, _ := r.ListSessions(ctx, p.ID)
	for _, s := range list {
		if s.SessionDate != nil {
			got[s.ScopeKey] = s.SessionDate.Format(domain.DateLayout)
		}
	}
	if got["akad"] != "2027-04-04" || got["resepsi"] != "2027-04-04" || got["lamaran"] != "2027-02-14" || len(got) != 3 {
		t.Fatalf("session dates = %v", got)
	}
}

// ---------- transactions ----------

func TestTxManagerRollsBackJoinsAndCommits(t *testing.T) {
	r, tx, pool := testEnv(t)
	ctx := context.Background()
	owner := newUser(t, pool)
	boom := errors.New("boom")

	// Rollback: everything written before the error disappears.
	p := baseProject(owner)
	err := tx.InTx(ctx, func(ctx context.Context) error {
		if ok, err := r.InsertProject(ctx, &p, nil); err != nil || !ok {
			return fmt.Errorf("insert: %v %v", ok, err)
		}

		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
	if n, _ := r.CountProjects(ctx, owner); n != 0 {
		t.Fatal("the project survived a rolled-back transaction")
	}

	// Nested InTx joins the outer transaction: the inner success is undone by the outer failure.
	q := baseProject(owner)
	err = tx.InTx(ctx, func(ctx context.Context) error {
		if err := tx.InTx(ctx, func(ctx context.Context) error {
			_, err := r.InsertProject(ctx, &q, nil)
			return err
		}); err != nil {
			return err
		}

		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
	if n, _ := r.CountProjects(ctx, owner); n != 0 {
		t.Fatal("a nested InTx must not commit on its own")
	}

	// Commit makes the work visible; reads inside the transaction see it before commit.
	err = tx.InTx(ctx, func(ctx context.Context) error {
		if _, err := r.InsertProject(ctx, &q, nil); err != nil {
			return err
		}
		if _, err := r.LockProject(ctx, q.ID); err != nil {
			return fmt.Errorf("the project is visible inside its own transaction: %w", err)
		}
		if _, err := r.ProjectForUser(context.Background(), q.ID, owner); !errors.Is(err, domain.ErrNotFound) {
			return fmt.Errorf("an uncommitted project must be invisible outside: %v", err)
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if n, _ := r.CountProjects(ctx, owner); n != 1 {
		t.Fatalf("CountProjects = %d, want 1 after commit", n)
	}
}

func TestLockProjectSerialisesConcurrentTransactions(t *testing.T) {
	r, tx, pool := testEnv(t)
	ctx := context.Background()
	p := baseProject(newUser(t, pool))
	mustInsert(t, r, &p)

	locked := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- tx.InTx(ctx, func(ctx context.Context) error {
			if _, err := r.LockProject(ctx, p.ID); err != nil {
				return err
			}
			close(locked)
			<-release

			return nil
		})
	}()
	<-locked

	second := make(chan time.Time, 1)
	go func() {
		_ = tx.InTx(ctx, func(ctx context.Context) error {
			_, err := r.LockProject(ctx, p.ID)
			second <- time.Now()

			return err
		})
	}()

	select {
	case <-second:
		t.Fatal("the second transaction got the lock while the first still holds it")
	case <-time.After(300 * time.Millisecond):
	}
	released := time.Now()
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	select {
	case at := <-second:
		if at.Before(released) {
			t.Fatal("the lock was granted early")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the second transaction never got the lock")
	}
}
