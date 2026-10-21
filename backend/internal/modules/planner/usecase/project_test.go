package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

func ptr[T any](v T) *T { return &v }

func TestCreateProjectSeedsSessionsAndStoresTrimmedTitle(t *testing.T) {
	svc, repo := newTestService(t, Options{})
	user := uuid.New()

	req := validWedding()
	req.Title = "  Pernikahan Sinta & Arif  "
	res, err := svc.CreateProject(context.Background(), user, req)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Created || res.Project.Title != "Pernikahan Sinta & Arif" {
		t.Fatalf("created=%v title=%q", res.Created, res.Project.Title)
	}
	if res.Project.Timezone != "Asia/Jakarta" || res.Project.Status != domain.StatusActive {
		t.Fatalf("defaults wrong: %+v", res.Project)
	}
	// prep_start_date defaults to *today in the project zone*: 21 Oct, not the UTC date 20 Oct.
	if got := res.Project.PrepStartDate.Format(domain.DateLayout); got != "2026-10-21" {
		t.Fatalf("prep_start_date = %s, want 2026-10-21", got)
	}

	sessions, _ := repo.ListSessions(context.Background(), res.Project.ID)
	var keys []string
	for _, s := range sessions {
		keys = append(keys, s.ScopeKey)
	}
	if strings.Join(keys, ",") != "all,lamaran,akad,resepsi" {
		t.Fatalf("sessions = %v, want all,lamaran,akad,resepsi", keys)
	}
	if !sessions[0].IsSystem || sessions[0].SortOrder != 0 {
		t.Errorf("'all' must be the system session at sort_order 0, got %+v", sessions[0])
	}
	if sessions[1].SessionDate != nil {
		t.Error("lamaran must be born without a date")
	}
	for _, s := range sessions[2:] {
		if s.SessionDate == nil || s.SessionDate.Format(domain.DateLayout) != "2027-03-28" {
			t.Errorf("%s must be born on Hari-H, got %v", s.ScopeKey, s.SessionDate)
		}
	}
	if len(res.Applied) != 0 {
		t.Errorf("no template requested, applied = %v", res.Applied)
	}
}

func TestCreateProjectValidation(t *testing.T) {
	svc, _ := newTestService(t, Options{})
	user := uuid.New()

	cases := []struct {
		name    string
		mutate  func(*domain.InitProjectRequest)
		code    string
		message string
	}{
		{"unknown event type", func(r *domain.InitProjectRequest) { r.EventType = "party" }, "invalid_input", "event_type: unknown event type"},
		{"empty event type", func(r *domain.InitProjectRequest) { r.EventType = "" }, "invalid_input", "event_type: unknown event type"},
		{"blank title", func(r *domain.InitProjectRequest) { r.Title = "   " }, "invalid_input", "title: must not be empty"},
		{"newline in title", func(r *domain.InitProjectRequest) { r.Title = "Nikah\nSinta" }, "invalid_input", "title: must not contain control characters"},
		{"title of 256 characters", func(r *domain.InitProjectRequest) { r.Title = strings.Repeat("é", 256) }, "invalid_input", "title: must be at most 255 characters"},
		{"lowercase zone", func(r *domain.InitProjectRequest) { r.Timezone = ptr("asia/jakarta") }, "invalid_timezone", ""},
		{"offset zone", func(r *domain.InitProjectRequest) { r.Timezone = ptr("UTC+7") }, "invalid_timezone", ""},
		{"Local zone", func(r *domain.InitProjectRequest) { r.Timezone = ptr("Local") }, "invalid_timezone", ""},
		{"abbreviation", func(r *domain.InitProjectRequest) { r.Timezone = ptr("WIB") }, "invalid_timezone", ""},
		{"well-formed but unknown zone", func(r *domain.InitProjectRequest) { r.Timezone = ptr("Asia/Nowhere") }, "invalid_timezone", "timezone: unknown time zone"},
		{"unknown region", func(r *domain.InitProjectRequest) { r.Timezone = ptr("Foo/Bar") }, "invalid_timezone", "timezone: must be a canonical IANA name such as Asia/Jakarta"},
		{"bad date format", func(r *domain.InitProjectRequest) { r.EventDate = "28-03-2027" }, "invalid_date", "event_date: must be a valid date in YYYY-MM-DD format"},
		{"impossible date", func(r *domain.InitProjectRequest) { r.EventDate = "2027-02-30" }, "invalid_date", "event_date: must be a valid date in YYYY-MM-DD format"},
		{"event in the past", func(r *domain.InitProjectRequest) { r.EventDate = "2026-10-20" }, "invalid_date", "event_date: must not be before today"},
		{"more than 5 years ahead", func(r *domain.InitProjectRequest) { r.EventDate = "2031-10-22" }, "invalid_date", "event_date: must be within 5 years from today"},
		{"prep after event", func(r *domain.InitProjectRequest) { r.PrepStartDate = ptr("2027-03-29") }, "invalid_date", "prep_start_date: must not be after event_date"},
		{"end before event", func(r *domain.InitProjectRequest) { r.EventEndDate = ptr("2027-03-27") }, "invalid_date", "event_end_date: must not be before event_date"},
		{"end more than 30 days after", func(r *domain.InitProjectRequest) { r.EventEndDate = ptr("2027-04-28") }, "invalid_date", "event_end_date: must be within 30 days of event_date"},
		{"negative budget", func(r *domain.InitProjectRequest) { r.TargetBudget = -1 }, "invalid_input", ""},
		{"budget above 10^13", func(r *domain.InitProjectRequest) { r.TargetBudget = 10_000_000_000_001 }, "invalid_input", ""},
		{"missing required metadata", func(r *domain.InitProjectRequest) { r.Metadata = map[string]any{"bride_name": "Sinta"} }, "invalid_metadata", "groom_name: required"},
		{"unknown metadata key", func(r *domain.InitProjectRequest) { r.Metadata["child_name"] = "x" }, "invalid_metadata", "child_name: unknown key for this event type"},
		{"short client token", func(r *domain.InitProjectRequest) { r.ClientToken = ptr("short") }, "invalid_input", "client_token: must be 8 to 64 characters"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := validWedding()
			tc.mutate(&req)
			_, err := svc.CreateProject(context.Background(), user, req)
			if tc.message != "" {
				wantMessage(t, err, tc.code, tc.message)
			} else {
				wantCode(t, err, tc.code)
			}
		})
	}
}

func TestCreateProjectBoundariesThatAreAllowed(t *testing.T) {
	svc, _ := newTestService(t, Options{})
	user := uuid.New()

	for name, mutate := range map[string]func(*domain.InitProjectRequest){
		"today at 06:47 WIB (UTC date is still yesterday)": func(r *domain.InitProjectRequest) { r.EventDate = "2026-10-21" },
		"exactly 1826 days ahead":                          func(r *domain.InitProjectRequest) { r.EventDate = "2031-10-21" },
		"event spanning exactly 30 days":                   func(r *domain.InitProjectRequest) { r.EventEndDate = ptr("2027-04-27") },
		"prep on Hari-H":                                   func(r *domain.InitProjectRequest) { r.PrepStartDate = ptr("2027-03-28") },
		"UTC zone":                                         func(r *domain.InitProjectRequest) { r.Timezone = ptr("UTC") },
		"budget exactly 10^13":                             func(r *domain.InitProjectRequest) { r.TargetBudget = 10_000_000_000_000 },
		"title of exactly 255 characters":                  func(r *domain.InitProjectRequest) { r.Title = strings.Repeat("é", 255) },
	} {
		t.Run(name, func(t *testing.T) {
			req := validWedding()
			mutate(&req)
			if _, err := svc.CreateProject(context.Background(), user, req); err != nil {
				t.Fatalf("must be accepted: %v", err)
			}
		})
	}
}

func TestCreateProjectInTheProjectZoneNotTheServerZone(t *testing.T) {
	// 23:47 UTC on 20 Oct is already 21 Oct in Jakarta but still 20 Oct in UTC.
	svc, _ := newTestService(t, Options{})
	req := validWedding()
	req.EventDate = "2026-10-20"

	if _, err := svc.CreateProject(context.Background(), uuid.New(), req); err == nil {
		t.Fatal("20 Oct is already the past for a Jakarta project")
	}
	req.Timezone = ptr("UTC")
	if _, err := svc.CreateProject(context.Background(), uuid.New(), req); err != nil {
		t.Fatalf("20 Oct is today for a UTC project: %v", err)
	}
}

func TestCreateProjectOtherEventTypes(t *testing.T) {
	svc, _ := newTestService(t, Options{})
	user := uuid.New()

	seminar := domain.InitProjectRequest{
		EventType: domain.EventSeminar, Title: "Tech Summit 2027", EventDate: "2027-05-15", EventEndDate: ptr("2027-05-16"),
		Metadata: map[string]any{"organizer_name": "Adicara", "expected_participants": float64(300)},
	}
	if _, err := svc.CreateProject(context.Background(), user, seminar); err != nil {
		t.Fatal(err)
	}

	seminar.Metadata = map[string]any{"organizer_name": "Adicara", "expected_participants": 1.5}
	_, err := svc.CreateProject(context.Background(), user, seminar)
	wantMessage(t, err, "invalid_metadata", "expected_participants: must be an integer")

	other := domain.InitProjectRequest{EventType: domain.EventOther, Title: "Reuni", EventDate: "2027-05-15"}
	if _, err := svc.CreateProject(context.Background(), user, other); err != nil {
		t.Fatalf("'other' has no required metadata: %v", err)
	}
}

func TestCreateProjectTemplatesNeedARegisteredApplier(t *testing.T) {
	svc, repo := newTestService(t, Options{})
	user := uuid.New()

	for name, tpl := range map[string]domain.InitTemplates{
		"timeline preset":       {TimelinePreset: ptr("wedding_pdf_44")},
		"adicara tasks":         {IncludeAdicaraTasks: true},
		"admin docs":            {AdminDocs: true},
		"budget lines":          {BudgetLines: true},
		"shopping list":         {ShoppingList: true},
		"rundown":               {Rundown: true},
		"playlist":              {Playlist: true},
		"preset of other types": {TimelinePreset: ptr("general")},
	} {
		t.Run(name, func(t *testing.T) {
			req := validWedding()
			req.Templates = tpl
			_, err := svc.CreateProject(context.Background(), user, req)
			wantCode(t, err, "invalid_template")
		})
	}
	if n, _ := repo.CountProjects(context.Background(), user); n != 0 {
		t.Fatalf("a rejected request must not create anything, found %d projects", n)
	}

	// With the timeline applier registered its templates are offered; others still are not.
	svc.RegisterTemplateAppliers(&stubApplier{key: domain.TemplateTimeline})
	req := validWedding()
	req.Templates = domain.InitTemplates{TimelinePreset: ptr("wedding_pdf_44"), IncludeAdicaraTasks: true}
	if _, err := svc.CreateProject(context.Background(), user, req); err != nil {
		t.Fatalf("timeline templates must be accepted now: %v", err)
	}
	req.Templates = domain.InitTemplates{TimelinePreset: ptr("general")}
	_, err := svc.CreateProject(context.Background(), user, req)
	wantCode(t, err, "invalid_template") // `general` belongs to the non-wedding types
	req.Templates = domain.InitTemplates{Rundown: true}
	_, err = svc.CreateProject(context.Background(), user, req)
	wantCode(t, err, "invalid_template")
}

func TestCreateProjectRunsAppliersInOrderAndReportsResults(t *testing.T) {
	svc, repo := newTestService(t, Options{})
	var order []string
	mk := func(key string) *orderedApplier {
		return &orderedApplier{stubApplier{key: key, res: domain.ApplyResult{Created: 3}}, &order}
	}
	svc.RegisterTemplateAppliers(mk("playlist"), mk("budget"), mk("timeline"), mk("shopping"), mk("rundown"), mk("admin_docs"))

	req := validWedding()
	req.Templates = domain.InitTemplates{
		TimelinePreset: ptr("wedding_pdf_44"), IncludeAdicaraTasks: true, AdminDocs: true, BudgetLines: true,
		ShoppingList: true, Rundown: true, Playlist: true,
	}
	res, err := svc.CreateProject(context.Background(), uuid.New(), req)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(order, ","); got != "timeline,admin_docs,budget,shopping,rundown,playlist" {
		t.Fatalf("apply order = %s; budget lines must precede the shopping list that rolls up into them", got)
	}
	if len(res.Applied) != 6 || res.Applied["timeline"].Created != 3 || res.Applied["timeline"].Warnings == nil {
		t.Fatalf("applied = %+v; every result needs created and a non-nil warnings list", res.Applied)
	}
	if _, err := repo.ProjectForUser(context.Background(), res.Project.ID, res.Project.UserID); err != nil {
		t.Fatal(err)
	}
}

type orderedApplier struct {
	stubApplier
	order *[]string
}

func (a *orderedApplier) Apply(ctx context.Context, id uuid.UUID, req domain.ApplyRequest) (domain.ApplyResult, error) {
	*a.order = append(*a.order, a.key)

	return a.stubApplier.Apply(ctx, id, req)
}

func TestCreateProjectIsAtomicWhenATemplateFails(t *testing.T) {
	svc, repo := newTestService(t, Options{})
	user := uuid.New()
	good := &stubApplier{key: "timeline"}
	bad := &stubApplier{key: "admin_docs", err: errors.New("disk full")}
	third := &stubApplier{key: "budget"}
	svc.RegisterTemplateAppliers(good, bad, third)

	req := validWedding()
	req.Templates = domain.InitTemplates{TimelinePreset: ptr("wedding_pdf_44"), AdminDocs: true, BudgetLines: true}
	_, err := svc.CreateProject(context.Background(), user, req)
	if err == nil || !strings.Contains(err.Error(), "admin_docs") {
		t.Fatalf("want the applier error, got %v", err)
	}
	if len(third.calls) != 0 {
		t.Error("appliers after the failing one must not run")
	}
	if n, _ := repo.CountProjects(context.Background(), user); n != 0 {
		t.Fatalf("project survived a failed template: %d projects", n)
	}
	if len(repo.sessions) != 0 {
		t.Fatalf("sessions survived a failed template: %v", repo.sessions)
	}
}

func TestCreateProjectClientTokenMakesRetriesNoOps(t *testing.T) {
	svc, repo := newTestService(t, Options{MaxProjects: 1})
	user := uuid.New()
	timeline := &stubApplier{key: "timeline"}
	svc.RegisterTemplateAppliers(timeline)

	req := validWedding()
	req.ClientToken = ptr("wizard-token-1")
	req.Templates = domain.InitTemplates{TimelinePreset: ptr("wedding_pdf_44")}
	first, err := svc.CreateProject(context.Background(), user, req)
	if err != nil || !first.Created {
		t.Fatalf("first: created=%v err=%v", first.Created, err)
	}

	// Same token again: the original project, no second one, templates not re-run —
	// even though the user is now at the project limit.
	again, err := svc.CreateProject(context.Background(), user, req)
	if err != nil {
		t.Fatalf("a retry must succeed even at the limit: %v", err)
	}
	if again.Created || again.Project.ID != first.Project.ID {
		t.Fatalf("retry returned created=%v id=%s, want the original %s", again.Created, again.Project.ID, first.Project.ID)
	}
	if len(timeline.calls) != 1 {
		t.Fatalf("timeline applied %d times, want 1", len(timeline.calls))
	}
	if n, _ := repo.CountProjects(context.Background(), user); n != 1 {
		t.Fatalf("%d projects, want 1", n)
	}

	// A different token at the limit is refused.
	req.ClientToken = ptr("wizard-token-2")
	_, err = svc.CreateProject(context.Background(), user, req)
	wantCode(t, err, "project_limit_reached")

	// Another user may reuse the token.
	if res, err := svc.CreateProject(context.Background(), uuid.New(), req); err != nil || !res.Created {
		t.Fatalf("tokens are per user: created=%v err=%v", res.Created, err)
	}
}

func TestProjectLimitIgnoresTheSampleAndOtherUsers(t *testing.T) {
	svc, _ := newTestService(t, Options{MaxProjects: 2})
	user := uuid.New()

	if _, err := svc.CreateSample(context.Background(), user); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err := svc.CreateProject(context.Background(), user, validWedding()); err != nil {
			t.Fatalf("project %d: %v", i+1, err)
		}
	}
	_, err := svc.CreateProject(context.Background(), user, validWedding())
	wantCode(t, err, "project_limit_reached")

	if _, err := svc.CreateProject(context.Background(), uuid.New(), validWedding()); err != nil {
		t.Fatalf("another user is unaffected: %v", err)
	}
}

// ---------- update ----------

func createdProject(t *testing.T, svc *Service, req domain.InitProjectRequest) domain.Project {
	t.Helper()
	res, err := svc.CreateProject(context.Background(), uuid.New(), req)
	if err != nil {
		t.Fatal(err)
	}

	return res.Project
}

func TestUpdateProjectChangesOnlyWhatWasSent(t *testing.T) {
	svc, _ := newTestService(t, Options{})
	p := createdProject(t, svc, validWedding())

	res, err := svc.UpdateProject(context.Background(), p.ID, domain.UpdateProjectRequest{Title: ptr("  Akad Sinta & Arif ")})
	if err != nil {
		t.Fatal(err)
	}
	got := res.Project
	if got.Title != "Akad Sinta & Arif" || got.TargetBudget != p.TargetBudget || !got.EventDate.Equal(p.EventDate) || got.Metadata["bride_name"] != "Sinta" {
		t.Fatalf("PATCH touched more than the title: %+v", got)
	}
	if res.Warnings != nil && len(res.Warnings) != 0 || len(res.Hints) != 0 {
		t.Fatalf("no date change, no notices: %+v %+v", res.Warnings, res.Hints)
	}

	_, err = svc.UpdateProject(context.Background(), p.ID, domain.UpdateProjectRequest{Title: ptr("   ")})
	wantMessage(t, err, "invalid_input", "title: must not be empty")
}

func TestUpdateProjectMetadataIsAMergePatch(t *testing.T) {
	svc, _ := newTestService(t, Options{})
	req := validWedding()
	req.Metadata = map[string]any{"bride_name": "Sinta", "groom_name": "Arif", "hero_title": "Bismillah Nikah", "city": "Bandung"}
	p := createdProject(t, svc, req)

	res, err := svc.UpdateProject(context.Background(), p.ID, domain.UpdateProjectRequest{
		Metadata: map[string]any{"city": "Jakarta", "hero_title": nil, "venue_name": "Sasana Kriya"},
	})
	if err != nil {
		t.Fatal(err)
	}
	m := res.Project.Metadata
	if m["city"] != "Jakarta" || m["venue_name"] != "Sasana Kriya" || m["bride_name"] != "Sinta" || m["groom_name"] != "Arif" {
		t.Fatalf("merge result wrong: %v", m)
	}
	if _, has := m["hero_title"]; has {
		t.Fatalf("null must delete the key: %v", m)
	}

	_, err = svc.UpdateProject(context.Background(), p.ID, domain.UpdateProjectRequest{Metadata: map[string]any{"bride_name": nil}})
	wantMessage(t, err, "invalid_metadata", "bride_name: required")
	_, err = svc.UpdateProject(context.Background(), p.ID, domain.UpdateProjectRequest{Metadata: map[string]any{"bride_name": "  "}})
	wantMessage(t, err, "invalid_metadata", "bride_name: required")
	_, err = svc.UpdateProject(context.Background(), p.ID, domain.UpdateProjectRequest{Metadata: map[string]any{"nope": "x"}})
	wantCode(t, err, "invalid_metadata")
}

func TestUpdateProjectShiftsTheEndDateAndSessionsWithHariH(t *testing.T) {
	svc, repo := newTestService(t, Options{})
	req := validWedding()
	req.EventEndDate = ptr("2027-03-30")
	p := createdProject(t, svc, req)

	// Akad and Resepsi are dated on Hari-H; a Lamaran dated elsewhere must stay.
	repo.otherDates = 1
	res, err := svc.UpdateProject(context.Background(), p.ID, domain.UpdateProjectRequest{EventDate: ptr("2027-04-04")})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Project.EventEndDate.Format(domain.DateLayout); got != "2027-04-06" {
		t.Fatalf("event_end_date = %s; a 3-day event moved by 7 days must end 2027-04-06", got)
	}
	if len(repo.sessionShifts) != 1 || repo.sessionShifts[0][0].Format(domain.DateLayout) != "2027-03-28" || repo.sessionShifts[0][1].Format(domain.DateLayout) != "2027-04-04" {
		t.Fatalf("sessions on the old Hari-H must follow: %v", repo.sessionShifts)
	}
	if len(res.Hints) != 1 || res.Hints[0].Code != "sessions_dates_unchanged" || res.Hints[0].Count != 1 {
		t.Fatalf("hints = %+v, want sessions_dates_unchanged x1", res.Hints)
	}

	// An end date that is sent wins over the shifted one; clearing removes it.
	res, err = svc.UpdateProject(context.Background(), p.ID, domain.UpdateProjectRequest{EventDate: ptr("2027-04-05"), EventEndDate: ptr("2027-04-07")})
	if err != nil || res.Project.EventEndDate.Format(domain.DateLayout) != "2027-04-07" {
		t.Fatalf("explicit end date: %v %v", res.Project.EventEndDate, err)
	}
	res, err = svc.UpdateProject(context.Background(), p.ID, domain.UpdateProjectRequest{ClearEventEndDate: true})
	if err != nil || res.Project.EventEndDate != nil {
		t.Fatalf("clear_event_end_date: %v %v", res.Project.EventEndDate, err)
	}
	_, err = svc.UpdateProject(context.Background(), p.ID, domain.UpdateProjectRequest{ClearEventEndDate: true, EventEndDate: ptr("2027-04-07")})
	wantCode(t, err, "invalid_input")
}

func TestUpdateProjectShiftTasksIsExplicitAndAtomicWithTheDateChange(t *testing.T) {
	svc, repo := newTestService(t, Options{})
	p := createdProject(t, svc, validWedding())
	repo.shiftable = 12

	// Not asked: only a hint, nothing shifted.
	res, err := svc.UpdateProject(context.Background(), p.ID, domain.UpdateProjectRequest{EventDate: ptr("2027-03-21")})
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.shiftedDays) != 0 {
		t.Fatalf("tasks shifted without shift_tasks: %v", repo.shiftedDays)
	}
	if len(res.Hints) != 1 || res.Hints[0].Code != "tasks_not_shifted" || res.Hints[0].Count != 12 || res.Hints[0].ShiftDays == nil || *res.Hints[0].ShiftDays != -7 {
		t.Fatalf("hints = %+v, want tasks_not_shifted count=12 shift_days=-7", res.Hints)
	}

	// Asked: shifted by the delta, with the overdue warning when some are now late.
	repo.overdue = 4
	res, err = svc.UpdateProject(context.Background(), p.ID, domain.UpdateProjectRequest{EventDate: ptr("2027-03-14"), ShiftTasks: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.shiftedDays) != 1 || repo.shiftedDays[0] != -7 {
		t.Fatalf("shifted by %v, want [-7]", repo.shiftedDays)
	}
	if len(res.Warnings) != 1 || res.Warnings[0].Code != "tasks_overdue_after_shift" || res.Warnings[0].Count != 4 {
		t.Fatalf("warnings = %+v, want tasks_overdue_after_shift x4", res.Warnings)
	}
	if len(res.Hints) != 0 {
		t.Fatalf("hints = %+v, want none when the tasks were shifted", res.Hints)
	}

	// shift_tasks with an unchanged date does nothing (a double click must not shift twice).
	repo.shiftedDays = nil
	if _, err := svc.UpdateProject(context.Background(), p.ID, domain.UpdateProjectRequest{EventDate: ptr("2027-03-14"), ShiftTasks: true}); err != nil {
		t.Fatal(err)
	}
	if len(repo.shiftedDays) != 0 {
		t.Fatalf("an unchanged date shifted tasks again: %v", repo.shiftedDays)
	}
}

func TestUpdateProjectDateRules(t *testing.T) {
	svc, _ := newTestService(t, Options{})
	p := createdProject(t, svc, validWedding()) // prep_start_date = 2026-10-21

	// Moving Hari-H to the past is allowed (correcting data) as long as prep still precedes it.
	if _, err := svc.UpdateProject(context.Background(), p.ID, domain.UpdateProjectRequest{PrepStartDate: ptr("2026-01-01"), EventDate: ptr("2026-02-01")}); err != nil {
		t.Fatalf("past dates are allowed on PATCH: %v", err)
	}
	_, err := svc.UpdateProject(context.Background(), p.ID, domain.UpdateProjectRequest{EventDate: ptr("2025-12-31")})
	wantMessage(t, err, "invalid_date", "prep_start_date: must not be after event_date")
	_, err = svc.UpdateProject(context.Background(), p.ID, domain.UpdateProjectRequest{EventEndDate: ptr("2026-01-31")})
	wantMessage(t, err, "invalid_date", "event_end_date: must not be before event_date")
	_, err = svc.UpdateProject(context.Background(), p.ID, domain.UpdateProjectRequest{EventEndDate: ptr("2026-03-15")})
	wantMessage(t, err, "invalid_date", "event_end_date: must be within 30 days of event_date")
}

func TestUpdateProjectMoneySavingsAndCategories(t *testing.T) {
	svc, _ := newTestService(t, Options{})
	p := createdProject(t, svc, validWedding())

	res, err := svc.UpdateProject(context.Background(), p.ID, domain.UpdateProjectRequest{
		TargetBudget: ptr(int64(80_000_000)), SavingsTarget: ptr(int64(60_000_000)),
		SkippedVendorCategories: &[]string{"entertainment", "mc", "mc"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Project.TargetBudget != 80_000_000 || *res.Project.SavingsTarget != 60_000_000 || strings.Join(res.Project.SkippedVendorCategories, ",") != "entertainment,mc" {
		t.Fatalf("got %+v", res.Project)
	}

	res, err = svc.UpdateProject(context.Background(), p.ID, domain.UpdateProjectRequest{ClearSavingsTarget: true, SkippedVendorCategories: &[]string{}})
	if err != nil || res.Project.SavingsTarget != nil || len(res.Project.SkippedVendorCategories) != 0 {
		t.Fatalf("clear: %+v %v", res.Project, err)
	}

	_, err = svc.UpdateProject(context.Background(), p.ID, domain.UpdateProjectRequest{TargetBudget: ptr(int64(-5))})
	wantCode(t, err, "invalid_input")
	_, err = svc.UpdateProject(context.Background(), p.ID, domain.UpdateProjectRequest{SavingsTarget: ptr(int64(10_000_000_000_001))})
	wantCode(t, err, "invalid_input")
	_, err = svc.UpdateProject(context.Background(), p.ID, domain.UpdateProjectRequest{ClearSavingsTarget: true, SavingsTarget: ptr(int64(1))})
	wantCode(t, err, "invalid_input")
	_, err = svc.UpdateProject(context.Background(), p.ID, domain.UpdateProjectRequest{SkippedVendorCategories: &[]string{"Has Space"}})
	wantCode(t, err, "invalid_input")
	_, err = svc.UpdateProject(context.Background(), p.ID, domain.UpdateProjectRequest{Timezone: ptr("Mars/Olympus")})
	wantCode(t, err, "invalid_timezone")
}

func TestStatusDeleteAndDetail(t *testing.T) {
	svc, repo := newTestService(t, Options{})
	user := uuid.New()
	res, _ := svc.CreateProject(context.Background(), user, validWedding())
	p := res.Project

	got, err := svc.ChangeStatus(context.Background(), p, domain.StatusArchived)
	if err != nil || got.Status != domain.StatusArchived {
		t.Fatalf("archive: %+v %v", got, err)
	}
	if got, err = svc.ChangeStatus(context.Background(), got, domain.StatusActive); err != nil || got.Status != domain.StatusActive {
		t.Fatalf("reopen: %+v %v", got, err)
	}
	_, err = svc.ChangeStatus(context.Background(), p, "paused")
	wantCode(t, err, "invalid_input")

	repo.shiftable = 7
	d, err := svc.ProjectDetail(context.Background(), p)
	if err != nil || d.ShiftableTasks != 7 {
		t.Fatalf("detail: %+v %v", d, err)
	}

	wantMessage(t, svc.DeleteProject(context.Background(), p, "pernikahan sinta & arif"), "invalid_input", "confirm: must match the project title exactly")
	wantCode(t, svc.DeleteProject(context.Background(), p, ""), "invalid_input")
	if err := svc.DeleteProject(context.Background(), p, p.Title); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ProjectForUser(context.Background(), p.ID, user); err == nil {
		t.Fatal("project must be gone")
	} else {
		wantCode(t, err, "not_found")
	}
}

func TestProjectForUserHidesOtherUsersProjects(t *testing.T) {
	svc, _ := newTestService(t, Options{})
	owner := uuid.New()
	res, _ := svc.CreateProject(context.Background(), owner, validWedding())

	_, err := svc.ProjectForUser(context.Background(), res.Project.ID, uuid.New())
	wantCode(t, err, "not_found")
	if _, err := svc.ProjectForUser(context.Background(), res.Project.ID, owner); err != nil {
		t.Fatal(err)
	}
}

// ---------- hub ----------

func TestSummaryCountsIgnoreTheSampleAndPickTheNearestActiveEvent(t *testing.T) {
	svc, repo := newTestService(t, Options{})
	user := uuid.New()
	mk := func(title, date string, budget int64, status domain.ProjectStatus) domain.Project {
		req := validWedding()
		req.Title, req.EventDate, req.TargetBudget = title, date, budget
		res, err := svc.CreateProject(context.Background(), user, req)
		if err != nil {
			t.Fatal(err)
		}
		if status != domain.StatusActive {
			repo.SetProjectStatus(context.Background(), res.Project.ID, status)
		}

		return res.Project
	}
	mk("Far", "2027-06-01", 10_000_000, domain.StatusActive)
	mk("Near", "2027-01-10", 20_000_000, domain.StatusActive)
	mk("Done", "2026-11-01", 5_000_000, domain.StatusCompleted)
	mk("Old", "2026-11-05", 7_000_000, domain.StatusArchived)
	if _, err := svc.CreateSample(context.Background(), user); err != nil {
		t.Fatal(err)
	}

	sum, err := svc.Summary(context.Background(), user)
	if err != nil {
		t.Fatal(err)
	}
	if sum.ActiveCount != 2 || sum.CompletedCount != 1 || sum.ArchivedCount != 1 {
		t.Fatalf("counts = %d/%d/%d, want 2/1/1 without the sample", sum.ActiveCount, sum.CompletedCount, sum.ArchivedCount)
	}
	if sum.CombinedTargetBudget != 30_000_000 {
		t.Fatalf("combined budget = %d, want 30.000.000 (active, no sample)", sum.CombinedTargetBudget)
	}
	if sum.NearestProject == nil || sum.NearestProject.Title != "Near" {
		t.Fatalf("nearest = %+v, want Near (the sample is excluded even though it is sooner)", sum.NearestProject)
	}
	if len(sum.Projects) != 5 {
		t.Fatalf("%d projects, want all 5 including the sample", len(sum.Projects))
	}
}

func TestSummaryNearestSkipsFinishedAndPrefersOngoing(t *testing.T) {
	now := time.Date(2027, 3, 29, 9, 0, 0, 0, time.FixedZone("WIB", 7*3600))
	svc, repo := newTestService(t, Options{Now: func() time.Time { return now }})
	user := uuid.New()

	add := func(title string, event time.Time, end *time.Time) {
		id := uuid.New()
		repo.projects[id] = domain.Project{ID: id, UserID: user, Title: title, EventType: domain.EventOther, Timezone: "Asia/Jakarta",
			EventDate: event, EventEndDate: end, Status: domain.StatusActive}
	}
	d := func(day int) time.Time { return time.Date(2027, 3, day, 0, 0, 0, 0, time.UTC) }
	e := d(30)
	add("Finished", d(20), nil)
	add("Ongoing three-day", d(28), &e)
	add("Upcoming", d(31), nil)

	sum, err := svc.Summary(context.Background(), user)
	if err != nil {
		t.Fatal(err)
	}
	if sum.NearestProject == nil || sum.NearestProject.Title != "Ongoing three-day" || sum.NearestProject.DayNumber != 2 {
		t.Fatalf("nearest = %+v, want the ongoing event on day 2", sum.NearestProject)
	}

	if _, err := svc.ListCards(context.Background(), user, "bogus"); err == nil {
		t.Fatal("an unknown status filter must be rejected")
	}
}

func TestEventTypesOfferOnlyTemplatesThatCanBeApplied(t *testing.T) {
	svc, _ := newTestService(t, Options{})

	views := svc.EventTypes()
	if len(views) != 9 {
		t.Fatalf("%d event types, want 9", len(views))
	}
	for _, v := range views {
		a := v.TemplatesAvailable
		if len(a.TimelinePresets) != 0 || a.AdminDocs || a.BudgetLines || a.ShoppingList || a.Rundown || a.Playlist {
			t.Fatalf("%s offers templates although no module registered an applier: %+v", v.Key, a)
		}
	}

	svc.RegisterTemplateAppliers(&stubApplier{key: "timeline"}, &stubApplier{key: "rundown"})
	for _, v := range svc.EventTypes() {
		if v.Key == domain.EventSeminar && (strings.Join(v.TemplatesAvailable.TimelinePresets, ",") != "general" || !v.TemplatesAvailable.Rundown || v.TemplatesAvailable.BudgetLines) {
			t.Fatalf("seminar after registering timeline+rundown: %+v", v.TemplatesAvailable)
		}
	}
}

func TestRegisteringAnUnknownOrDuplicateProviderPanics(t *testing.T) {
	svc, _ := newTestService(t, Options{})
	mustPanic := func(name string, fn func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Errorf("%s must panic at start-up", name)
			}
		}()
		fn()
	}

	mustPanic("unknown applier", func() { svc.RegisterTemplateAppliers(&stubApplier{key: "tasks"}) })
	svc.RegisterTemplateAppliers(&stubApplier{key: "timeline"})
	mustPanic("duplicate applier", func() { svc.RegisterTemplateAppliers(&stubApplier{key: "timeline"}) })
	mustPanic("unknown provider", func() { svc.RegisterSummary(&stubProvider{key: "weather"}) })
	svc.RegisterSummary(&stubProvider{key: "guests"})
	mustPanic("duplicate provider", func() { svc.RegisterSummary(&stubProvider{key: "guests"}) })
}
