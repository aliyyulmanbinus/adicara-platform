package usecase

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

func projectWithSessions(t *testing.T) (*Service, *memRepo, domain.Project) {
	t.Helper()
	svc, repo := newTestService(t, Options{})

	return svc, repo, createdProject(t, svc, validWedding())
}

func sessionByKey(t *testing.T, repo *memRepo, projectID uuid.UUID, key string) domain.Session {
	t.Helper()
	list, _ := repo.ListSessions(context.Background(), projectID)
	for _, s := range list {
		if s.ScopeKey == key {
			return s
		}
	}
	t.Fatalf("no session %q in %v", key, list)

	return domain.Session{}
}

func TestCreateSessionDerivesKeyOrderAndNormalizesTimes(t *testing.T) {
	svc, _, p := projectWithSessions(t)

	s, err := svc.CreateSession(context.Background(), p.ID, domain.CreateSessionRequest{
		Name: "  Ngunduh Mantu ", Icon: ptr("🎊"), SessionDate: ptr("2027-04-10"), StartTime: ptr("9.05"), EndTime: ptr("11.00"), LocationName: ptr(" Rumah Bandung "),
	})
	if err != nil {
		t.Fatal(err)
	}
	if s.ScopeKey != "ngunduh_mantu" || s.Name != "Ngunduh Mantu" || s.SortOrder != 4 || s.Icon != "🎊" {
		t.Fatalf("got %+v", s)
	}
	if *s.StartTime != "09:05" || *s.EndTime != "11:00" || *s.LocationName != "Rumah Bandung" || s.SessionDate.Format(domain.DateLayout) != "2027-04-10" {
		t.Fatalf("times/location not normalized: %+v", s)
	}

	defaulted, err := svc.CreateSession(context.Background(), p.ID, domain.CreateSessionRequest{Name: "Foto Keluarga"})
	if err != nil {
		t.Fatal(err)
	}
	if defaulted.Icon != "📋" || defaulted.SortOrder != 5 || defaulted.StartTime != nil || defaulted.SessionDate != nil {
		t.Fatalf("defaults wrong: %+v", defaulted)
	}
}

func TestCreateSessionKeyCollisionsAndReservedNames(t *testing.T) {
	svc, _, p := projectWithSessions(t)
	create := func(name string) domain.Session {
		t.Helper()
		s, err := svc.CreateSession(context.Background(), p.ID, domain.CreateSessionRequest{Name: name})
		if err != nil {
			t.Fatalf("%q: %v", name, err)
		}

		return s
	}

	if k := create("Acara-Utama").ScopeKey; k != "acara_utama" {
		t.Fatalf("key = %q", k)
	}
	// Different names, same derived key: suffixes.
	if k := create("Acara Utama!").ScopeKey; k != "acara_utama_2" {
		t.Fatalf("key = %q, want acara_utama_2", k)
	}
	if k := create("acara_utama?").ScopeKey; k != "acara_utama_3" {
		t.Fatalf("key = %q, want acara_utama_3", k)
	}
	if k := create("🎉").ScopeKey; k != "sesi" {
		t.Fatalf("emoji-only name key = %q, want sesi", k)
	}
	if k := create("🎊").ScopeKey; k != "sesi_2" {
		t.Fatalf("second emoji-only name key = %q, want sesi_2", k)
	}
	if s := create("All"); s.ScopeKey != "all_sesi" || s.IsSystem {
		t.Fatalf("the reserved key must never be given to a user session: %+v", s)
	}
}

func TestCreateSessionRejectsDuplicateNamesAndBadInput(t *testing.T) {
	svc, _, p := projectWithSessions(t)
	ctx := context.Background()

	_, err := svc.CreateSession(ctx, p.ID, domain.CreateSessionRequest{Name: "RESEPSI"})
	wantCode(t, err, "session_name_taken") // case-insensitive against the seeded "Resepsi"

	for name, req := range map[string]domain.CreateSessionRequest{
		"blank name":     {Name: "  "},
		"long name":      {Name: strings.Repeat("x", 101)},
		"control name":   {Name: "a\tb"},
		"long icon":      {Name: "ok", Icon: ptr("123456789")},
		"bad date":       {Name: "ok", SessionDate: ptr("2027-13-01")},
		"bad start time": {Name: "ok", StartTime: ptr("25:00")},
		"bad end time":   {Name: "ok", EndTime: ptr("7:5")},
		"long location":  {Name: "ok", LocationName: ptr(strings.Repeat("x", 151))},
	} {
		_, err := svc.CreateSession(ctx, p.ID, req)
		if err == nil {
			t.Errorf("%s: must be rejected", name)
			continue
		}
		if c := apiCode(t, err); c != "invalid_input" && c != "invalid_date" {
			t.Errorf("%s: code %s", name, c)
		}
	}

	// A blank icon, empty time strings and a blank location are "not sent".
	if _, err := svc.CreateSession(ctx, p.ID, domain.CreateSessionRequest{Name: "Kosong", Icon: ptr(" "), StartTime: ptr(""), LocationName: ptr("  ")}); err != nil {
		t.Fatalf("blank optionals: %v", err)
	}
}

func TestCreateSessionLimitIsTwentyNonSystemSessions(t *testing.T) {
	svc, _, p := projectWithSessions(t) // 3 seeded + 'all'
	for i := 0; i < 17; i++ {
		if _, err := svc.CreateSession(context.Background(), p.ID, domain.CreateSessionRequest{Name: fmt.Sprintf("Sesi tambahan %d", i)}); err != nil {
			t.Fatalf("session %d: %v", i+4, err)
		}
	}
	_, err := svc.CreateSession(context.Background(), p.ID, domain.CreateSessionRequest{Name: "Yang ke-21"})
	wantCode(t, err, "session_limit_reached")
}

func TestUpdateSessionReplacesEverythingExceptTheKey(t *testing.T) {
	svc, repo, p := projectWithSessions(t)
	resepsi := sessionByKey(t, repo, p.ID, "resepsi")
	resepsi.StartTime, resepsi.LocationName = ptr("10:00"), ptr("Gedung A")
	repo.UpdateSession(context.Background(), &resepsi)

	// PUT without start_time/location clears them: the form always sends everything.
	s, err := svc.UpdateSession(context.Background(), p.ID, resepsi.ID, domain.UpdateSessionRequest{Name: "Resepsi Siang", Icon: ptr("🥂")})
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "Resepsi Siang" || s.Icon != "🥂" || s.StartTime != nil || s.LocationName != nil || s.SessionDate != nil || s.ScopeKey != "resepsi" {
		t.Fatalf("got %+v", s)
	}

	all := sessionByKey(t, repo, p.ID, "all")
	_, err = svc.UpdateSession(context.Background(), p.ID, all.ID, domain.UpdateSessionRequest{Name: "Semua"})
	wantCode(t, err, "invalid_input")
	_, err = svc.UpdateSession(context.Background(), p.ID, uuid.New(), domain.UpdateSessionRequest{Name: "x"})
	wantCode(t, err, "not_found")
	_, err = svc.UpdateSession(context.Background(), p.ID, resepsi.ID, domain.UpdateSessionRequest{Name: "lamaran"})
	wantCode(t, err, "session_name_taken")
}

func TestReorderSessionsNeedsEveryNonSystemSessionOnce(t *testing.T) {
	svc, repo, p := projectWithSessions(t)
	ctx := context.Background()
	lamaran, akad, resepsi := sessionByKey(t, repo, p.ID, "lamaran"), sessionByKey(t, repo, p.ID, "akad"), sessionByKey(t, repo, p.ID, "resepsi")
	all := sessionByKey(t, repo, p.ID, "all")

	out, err := svc.ReorderSessions(ctx, p.ID, domain.ReorderSessionsRequest{SessionIDs: []uuid.UUID{resepsi.ID, lamaran.ID, akad.ID}})
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	for _, s := range out {
		keys = append(keys, fmt.Sprintf("%s=%d", s.ScopeKey, s.SortOrder))
	}
	if got := strings.Join(keys, ","); got != "all=0,resepsi=1,lamaran=2,akad=3" {
		t.Fatalf("order = %s", got)
	}

	for name, ids := range map[string][]uuid.UUID{
		"missing":        {resepsi.ID, lamaran.ID},
		"duplicate":      {resepsi.ID, resepsi.ID, lamaran.ID},
		"foreign":        {resepsi.ID, lamaran.ID, uuid.New()},
		"includes 'all'": {resepsi.ID, lamaran.ID, all.ID},
		"empty":          {},
		"too many":       {resepsi.ID, lamaran.ID, akad.ID, uuid.New()},
	} {
		_, err := svc.ReorderSessions(ctx, p.ID, domain.ReorderSessionsRequest{SessionIDs: ids})
		if err == nil {
			t.Errorf("%s: must be rejected", name)
			continue
		}
		wantCode(t, err, "invalid_input")
	}
}

func TestDeleteSessionRules(t *testing.T) {
	svc, repo, p := projectWithSessions(t)
	ctx := context.Background()
	lamaran, all := sessionByKey(t, repo, p.ID, "lamaran"), sessionByKey(t, repo, p.ID, "all")

	wantCode(t, svc.DeleteSession(ctx, p.ID, all.ID, "", false), "invalid_input")
	wantCode(t, svc.DeleteSession(ctx, p.ID, uuid.New(), "", false), "not_found")
	wantCode(t, svc.DeleteSession(ctx, p.ID, lamaran.ID, "lamaran", false), "invalid_scope")
	wantCode(t, svc.DeleteSession(ctx, p.ID, lamaran.ID, "nope", false), "invalid_scope")
	if len(repo.moves) != 0 {
		t.Fatalf("a rejected delete moved data: %v", repo.moves)
	}

	// Default destination is 'all'.
	if err := svc.DeleteSession(ctx, p.ID, lamaran.ID, "", false); err != nil {
		t.Fatal(err)
	}
	if len(repo.moves) != 1 || repo.moves[0] != [2]string{"lamaran", "all"} {
		t.Fatalf("moves = %v, want lamaran -> all", repo.moves)
	}
}

func TestDeleteSessionWithAgendasNeedsConfirmationAndReportsUsage(t *testing.T) {
	svc, repo, p := projectWithSessions(t)
	ctx := context.Background()
	resepsi := sessionByKey(t, repo, p.ID, "resepsi")
	repo.usage = domain.SessionUsage{Tasks: 3, Agendas: 14, ScenarioItems: 2}

	err := svc.DeleteSession(ctx, p.ID, resepsi.ID, "akad", false)
	wantCode(t, err, "session_in_use")
	details, ok := err.(*apierr.Error).Details.(map[string]any)
	if !ok {
		t.Fatalf("session_in_use must carry details, got %#v", err.(*apierr.Error).Details)
	}
	if u := details["usage"].(domain.SessionUsage); u.Agendas != 14 || u.Tasks != 3 || u.ScenarioItems != 2 {
		t.Fatalf("usage = %+v", u)
	}
	if len(repo.moves) != 0 {
		t.Fatal("nothing may change when the delete is refused")
	}

	if err := svc.DeleteSession(ctx, p.ID, resepsi.ID, "akad", true); err != nil {
		t.Fatal(err)
	}
	if len(repo.moves) != 1 || repo.moves[0] != [2]string{"resepsi", "akad"} {
		t.Fatalf("moves = %v", repo.moves)
	}
}
