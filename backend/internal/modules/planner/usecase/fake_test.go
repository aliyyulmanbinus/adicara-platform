package usecase

import (
	"context"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

// memRepo is an in-memory domain.Repository. Only what the usecase reads is
// modelled; the SQL itself is covered by the integration tests.
type memRepo struct {
	mu       sync.Mutex
	projects map[uuid.UUID]domain.Project
	tokens   map[string]uuid.UUID // "user|token" -> project
	sessions map[uuid.UUID][]domain.Session

	shiftable      int
	overdue        int
	otherDates     int
	usage          domain.SessionUsage
	shiftedDays    []int
	sessionShifts  [][2]time.Time
	moves          [][2]string
	updatedProject *domain.Project
}

func newMemRepo() *memRepo {
	return &memRepo{projects: map[uuid.UUID]domain.Project{}, tokens: map[string]uuid.UUID{}, sessions: map[uuid.UUID][]domain.Session{}}
}

// snapshot/restore give fakeTx its rollback.
type memState struct {
	projects map[uuid.UUID]domain.Project
	tokens   map[string]uuid.UUID
	sessions map[uuid.UUID][]domain.Session
}

func (r *memRepo) snapshot() memState {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := memState{projects: map[uuid.UUID]domain.Project{}, tokens: map[string]uuid.UUID{}, sessions: map[uuid.UUID][]domain.Session{}}
	for k, v := range r.projects {
		s.projects[k] = v
	}
	for k, v := range r.tokens {
		s.tokens[k] = v
	}
	for k, v := range r.sessions {
		s.sessions[k] = append([]domain.Session(nil), v...)
	}

	return s
}

func (r *memRepo) restore(s memState) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.projects, r.tokens, r.sessions = s.projects, s.tokens, s.sessions
}

type fakeTx struct{ repo *memRepo }

func (f fakeTx) InTx(ctx context.Context, fn func(ctx context.Context) error) error {
	before := f.repo.snapshot()
	if err := fn(ctx); err != nil {
		f.repo.restore(before)
		return err
	}

	return nil
}

func (r *memRepo) InsertProject(_ context.Context, p *domain.Project, token *string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if token != nil {
		if _, dup := r.tokens[p.UserID.String()+"|"+*token]; dup {
			return false, nil
		}
		r.tokens[p.UserID.String()+"|"+*token] = p.ID
	}
	if p.IsSample {
		for _, e := range r.projects {
			if e.UserID == p.UserID && e.IsSample {
				return false, nil
			}
		}
	}
	p.CreatedAt, p.UpdatedAt = time.Now(), time.Now()
	r.projects[p.ID] = *p

	return true, nil
}

func (r *memRepo) ProjectForUser(_ context.Context, id, userID uuid.UUID) (domain.Project, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.projects[id]; ok && p.UserID == userID {
		return p, nil
	}

	return domain.Project{}, domain.ErrNotFound
}

func (r *memRepo) LockProject(_ context.Context, id uuid.UUID) (domain.Project, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.projects[id]; ok {
		return p, nil
	}

	return domain.Project{}, domain.ErrNotFound
}

func (r *memRepo) ProjectByClientToken(_ context.Context, userID uuid.UUID, token string) (domain.Project, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if id, ok := r.tokens[userID.String()+"|"+token]; ok {
		if p, ok := r.projects[id]; ok {
			return p, nil
		}
	}

	return domain.Project{}, domain.ErrNotFound
}

func (r *memRepo) SampleProject(_ context.Context, userID uuid.UUID) (domain.Project, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.projects {
		if p.UserID == userID && p.IsSample {
			return p, nil
		}
	}

	return domain.Project{}, domain.ErrNotFound
}

func (r *memRepo) CountProjects(_ context.Context, userID uuid.UUID) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, p := range r.projects {
		if p.UserID == userID && !p.IsSample {
			n++
		}
	}

	return n, nil
}

func (r *memRepo) ListCards(_ context.Context, userID uuid.UUID, status string, now time.Time) ([]domain.ProjectCardResponse, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []domain.ProjectCardResponse
	for _, p := range r.projects {
		if p.UserID != userID || (status != "all" && string(p.Status) != status) {
			continue
		}
		loc, _ := domain.LoadLocation(p.Timezone)
		out = append(out, domain.ProjectCardResponse{
			ID: p.ID, EventType: p.EventType, Title: p.Title, EventDate: p.EventDate.Format(domain.DateLayout),
			PhaseInfo: domain.ComputePhase(now, loc, p.EventDate, p.EventEndDate),
			Status:    p.Status, IsSample: p.IsSample, TargetBudget: p.TargetBudget,
		})
	}
	// Same order as the SQL: active first by date, then the rest.
	rank := map[domain.ProjectStatus]int{domain.StatusActive: 0, domain.StatusCompleted: 1, domain.StatusArchived: 2}
	sort.Slice(out, func(i, j int) bool {
		if rank[out[i].Status] != rank[out[j].Status] {
			return rank[out[i].Status] < rank[out[j].Status]
		}
		if out[i].Status == domain.StatusActive {
			return out[i].EventDate < out[j].EventDate
		}

		return out[i].EventDate > out[j].EventDate
	})

	return out, nil
}

func (r *memRepo) ListLite(context.Context, uuid.UUID, string, time.Time) ([]domain.ProjectLiteResponse, error) {
	return nil, nil
}

func (r *memRepo) UpdateProject(_ context.Context, p *domain.Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *p
	r.updatedProject = &cp
	r.projects[p.ID] = *p

	return nil
}

func (r *memRepo) SetProjectStatus(_ context.Context, id uuid.UUID, status domain.ProjectStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.projects[id]
	if !ok {
		return domain.ErrNotFound
	}
	p.Status = status
	r.projects[id] = p

	return nil
}

func (r *memRepo) DeleteProject(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.projects[id]; !ok {
		return domain.ErrNotFound
	}
	delete(r.projects, id)

	return nil
}

func (r *memRepo) CountShiftableTasks(context.Context, uuid.UUID) (int, error) {
	return r.shiftable, nil
}

func (r *memRepo) ShiftTemplateTasks(_ context.Context, _ uuid.UUID, days int) (int, error) {
	r.shiftedDays = append(r.shiftedDays, days)

	return r.shiftable, nil
}

func (r *memRepo) CountOverdueTemplateTasks(context.Context, uuid.UUID, time.Time) (int, error) {
	return r.overdue, nil
}

func (r *memRepo) ShiftSessionDates(_ context.Context, projectID uuid.UUID, from, to time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessionShifts = append(r.sessionShifts, [2]time.Time{from, to})
	list := r.sessions[projectID]
	for i := range list {
		if list[i].SessionDate != nil && list[i].SessionDate.Equal(from) {
			d := to
			list[i].SessionDate = &d
		}
	}

	return nil
}

func (r *memRepo) CountSessionsOnOtherDates(context.Context, uuid.UUID, time.Time) (int, error) {
	return r.otherDates, nil
}

func (r *memRepo) ListSessions(_ context.Context, projectID uuid.UUID) ([]domain.Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := append([]domain.Session(nil), r.sessions[projectID]...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].SortOrder < out[j].SortOrder })

	return out, nil
}

func (r *memRepo) ListSessionsWithUsage(ctx context.Context, projectID uuid.UUID) ([]domain.SessionWithUsage, error) {
	list, _ := r.ListSessions(ctx, projectID)
	out := make([]domain.SessionWithUsage, len(list))
	for i, s := range list {
		out[i] = domain.SessionWithUsage{Session: s, Usage: r.usage}
	}

	return out, nil
}

func (r *memRepo) SessionByID(_ context.Context, projectID, sessionID uuid.UUID) (domain.Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, s := range r.sessions[projectID] {
		if s.ID == sessionID {
			return s, nil
		}
	}

	return domain.Session{}, domain.ErrNotFound
}

func (r *memRepo) InsertSession(_ context.Context, s *domain.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range r.sessions[s.ProjectID] {
		if strings.EqualFold(e.Name, s.Name) {
			return domain.ErrSessionNameTaken
		}
		if e.ScopeKey == s.ScopeKey {
			return domain.ErrScopeKeyTaken
		}
	}
	r.sessions[s.ProjectID] = append(r.sessions[s.ProjectID], *s)

	return nil
}

func (r *memRepo) UpdateSession(_ context.Context, s *domain.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	list := r.sessions[s.ProjectID]
	for _, e := range list {
		if e.ID != s.ID && strings.EqualFold(e.Name, s.Name) {
			return domain.ErrSessionNameTaken
		}
	}
	for i := range list {
		if list[i].ID == s.ID {
			list[i] = *s
			return nil
		}
	}

	return domain.ErrNotFound
}

func (r *memRepo) ReorderSessions(_ context.Context, projectID uuid.UUID, ids []uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	list := r.sessions[projectID]
	for i := range list {
		for pos, id := range ids {
			if list[i].ID == id {
				list[i].SortOrder = pos + 1
			}
		}
	}

	return nil
}

func (r *memRepo) SessionUsage(context.Context, uuid.UUID, domain.Session) (domain.SessionUsage, error) {
	return r.usage, nil
}

func (r *memRepo) MoveSessionData(_ context.Context, projectID uuid.UUID, from, to string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.moves = append(r.moves, [2]string{from, to})
	list := r.sessions[projectID]
	for i := range list {
		if list[i].ScopeKey == from {
			r.sessions[projectID] = append(list[:i:i], list[i+1:]...)
			break
		}
	}

	return nil
}

// ---------- fixtures ----------

// jakartaMorning is 21 Oct 2026 06:47 WIB: 23:47 UTC the day before, the window
// where truncating to 24h gives the wrong calendar day.
var jakartaMorning = time.Date(2026, 10, 21, 6, 47, 0, 0, time.FixedZone("WIB", 7*3600))

func newTestService(t *testing.T, opts Options) (*Service, *memRepo) {
	t.Helper()
	repo := newMemRepo()
	if opts.Now == nil {
		opts.Now = func() time.Time { return jakartaMorning }
	}

	return New(repo, fakeTx{repo}, domain.DefaultRegistry(), opts), repo
}

func validWedding() domain.InitProjectRequest {
	return domain.InitProjectRequest{
		EventType: domain.EventWedding, Title: "Pernikahan Sinta & Arif", EventDate: "2027-03-28", TargetBudget: 75_000_000,
		Metadata: map[string]any{"bride_name": "Sinta", "groom_name": "Arif"},
	}
}

func apiCode(t *testing.T, err error) string {
	t.Helper()
	ae, ok := err.(*apierr.Error)
	if !ok {
		t.Fatalf("want *apierr.Error, got %T: %v", err, err)
	}

	return ae.Code
}

func wantCode(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("want error %q, got nil", code)
	}
	if got := apiCode(t, err); got != code {
		t.Fatalf("error code = %q (%v), want %q", got, err, code)
	}
}

func wantMessage(t *testing.T, err error, code, message string) {
	t.Helper()
	wantCode(t, err, code)
	if got := err.(*apierr.Error).Message; got != message {
		t.Fatalf("message = %q, want %q", got, message)
	}
}

// stubApplier records its calls and can fail or add a warning.
type stubApplier struct {
	key   string
	err   error
	calls []domain.ApplyRequest
	res   domain.ApplyResult
}

func (a *stubApplier) Key() string { return a.key }

func (a *stubApplier) Apply(_ context.Context, _ uuid.UUID, req domain.ApplyRequest) (domain.ApplyResult, error) {
	a.calls = append(a.calls, req)

	return a.res, a.err
}
