package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/auth/domain"
)

// stateRepo fakes just what Refresh, Authenticate, and Login touch.
type stateRepo struct {
	domain.Repository

	state      domain.AuthState
	stateErr   error
	consumeErr error

	reuseCalls int
	reuseGrace time.Duration
	reuseErr   error
	created    int
}

func (r *stateRepo) UserAuthState(ctx context.Context, id uuid.UUID) (domain.AuthState, error) {
	return r.state, r.stateErr
}

func (r *stateRepo) ConsumeRefreshSession(ctx context.Context, sessionID, userID uuid.UUID) error {
	return r.consumeErr
}

func (r *stateRepo) RevokeUserSessionsOnReuse(ctx context.Context, sessionID, userID uuid.UUID, grace time.Duration) (int64, error) {
	r.reuseCalls++
	r.reuseGrace = grace

	return 0, r.reuseErr
}

func (r *stateRepo) CreateRefreshSession(ctx context.Context, sessionID, userID uuid.UUID, expiresAt time.Time) error {
	r.created++
	return nil
}

func (r *stateRepo) UserByIdentifier(ctx context.Context, identifier string) (domain.User, string, error) {
	return domain.User{}, "", domain.ErrUserNotFound
}

func activeState() domain.AuthState { return domain.AuthState{Status: domain.StatusActive} }

func newStateService(repo *stateRepo) *Service {
	return New(repo, "test-secret", time.Minute, time.Hour)
}

func accessFor(t *testing.T, s *Service, id uuid.UUID) string {
	t.Helper()
	token, err := s.sign(id, "access", time.Minute, uuid.Nil)
	if err != nil {
		t.Fatal(err)
	}

	return token
}

func refreshFor(t *testing.T, s *Service, id uuid.UUID) string {
	t.Helper()
	token, err := s.sign(id, "refresh", time.Hour, uuid.New())
	if err != nil {
		t.Fatal(err)
	}

	return token
}

// --- login timing ---------------------------------------------------------

// Without the dummy comparison an unknown identifier returns in microseconds,
// while a known one with a wrong password spends a full bcrypt, which lets an
// attacker enumerate accounts. The floor is far below any bcrypt cost-12 time.
func TestLoginSpendsBcryptTimeForUnknownIdentifier(t *testing.T) {
	s := newStateService(&stateRepo{})

	start := time.Now()
	_, _, err := s.Login(context.Background(), "nobody@example.com", "Password123")
	elapsed := time.Since(start)

	requireAPIError(t, err, 401, "unauthorized")
	if elapsed < 20*time.Millisecond {
		t.Fatalf("unknown identifier returned in %v; it must cost about as much as a wrong password", elapsed)
	}
}

// --- refresh: reuse detection and account status --------------------------

func TestRefreshReplayRevokesSessionsAfterGrace(t *testing.T) {
	repo := &stateRepo{consumeErr: domain.ErrSessionInactive}
	s := newStateService(repo)

	_, err := s.Refresh(context.Background(), refreshFor(t, s, uuid.New()))

	requireAPIError(t, err, 401, "unauthorized")
	if repo.reuseCalls != 1 {
		t.Fatalf("a rejected refresh must run reuse detection once, ran %d times", repo.reuseCalls)
	}
	if repo.reuseGrace != refreshReuseGrace {
		t.Fatalf("grace = %v, want %v", repo.reuseGrace, refreshReuseGrace)
	}
	if repo.created != 0 {
		t.Fatal("no new session may be issued for a spent token")
	}
}

func TestRefreshReplayStillFailsWhenReuseDetectionErrors(t *testing.T) {
	repo := &stateRepo{consumeErr: domain.ErrSessionInactive, reuseErr: context.DeadlineExceeded}
	s := newStateService(repo)

	if _, err := s.Refresh(context.Background(), refreshFor(t, s, uuid.New())); err == nil {
		t.Fatal("Refresh must not succeed for a spent token")
	}
}

func TestRefreshRejectsInactiveAccount(t *testing.T) {
	repo := &stateRepo{state: domain.AuthState{Status: "inactive"}}
	s := newStateService(repo)

	_, err := s.Refresh(context.Background(), refreshFor(t, s, uuid.New()))

	requireAPIError(t, err, 401, "unauthorized")
	if repo.created != 0 {
		t.Fatal("an inactive account must not receive a new session")
	}
}

func TestRefreshRejectsVanishedAccount(t *testing.T) {
	repo := &stateRepo{stateErr: domain.ErrUserNotFound}
	s := newStateService(repo)

	_, err := s.Refresh(context.Background(), refreshFor(t, s, uuid.New()))

	requireAPIError(t, err, 401, "unauthorized")
}

func TestRefreshIssuesNewPairForActiveAccount(t *testing.T) {
	repo := &stateRepo{state: activeState()}
	s := newStateService(repo)

	tokens, err := s.Refresh(context.Background(), refreshFor(t, s, uuid.New()))
	if err != nil {
		t.Fatal(err)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" || repo.created != 1 {
		t.Fatalf("expected a fresh pair and one new session, got %+v (%d sessions)", tokens, repo.created)
	}
	if repo.reuseCalls != 0 {
		t.Fatal("reuse detection must only run when the token was rejected")
	}
}

// --- authenticate: the access token is checked against the database --------

func TestAuthenticateAcceptsActiveAccount(t *testing.T) {
	id := uuid.New()
	s := newStateService(&stateRepo{state: activeState()})

	got, err := s.Authenticate(context.Background(), accessFor(t, newStateService(&stateRepo{}), id))
	if err != nil {
		t.Fatal(err)
	}
	if got != id {
		t.Fatalf("got %s, want %s", got, id)
	}
}

func TestAuthenticateRejectsInactiveAndMissingAccounts(t *testing.T) {
	tests := []struct {
		name string
		repo *stateRepo
	}{
		{"inactive", &stateRepo{state: domain.AuthState{Status: "inactive"}}},
		{"deleted", &stateRepo{stateErr: domain.ErrUserNotFound}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newStateService(tt.repo)
			_, err := s.Authenticate(context.Background(), accessFor(t, s, uuid.New()))
			requireAPIError(t, err, 401, "unauthorized")
		})
	}
}

func TestAuthenticateSurfacesDatabaseFailureAsError(t *testing.T) {
	s := newStateService(&stateRepo{stateErr: context.DeadlineExceeded})

	_, err := s.Authenticate(context.Background(), accessFor(t, s, uuid.New()))
	if err == nil {
		t.Fatal("a database failure must not be reported as a valid token")
	}
	// Not an API error: it must surface as a 500, not as a 401 that would make
	// the client throw away a perfectly good session.
	var ae *apierr.Error
	if errors.As(err, &ae) {
		t.Fatalf("a database failure must not become an API error (got %d)", ae.Status)
	}
}

func TestAuthenticateRejectsTokenIssuedBeforePasswordChange(t *testing.T) {
	changed := time.Now().Add(5 * time.Second)
	repo := &stateRepo{state: domain.AuthState{Status: domain.StatusActive, PasswordChangedAt: &changed}}
	s := newStateService(repo)

	_, err := s.Authenticate(context.Background(), accessFor(t, s, uuid.New()))

	requireAPIError(t, err, 401, "unauthorized")
}

func TestAuthenticateAcceptsTokenIssuedAfterPasswordChange(t *testing.T) {
	changed := time.Now().Add(-time.Minute)
	repo := &stateRepo{state: domain.AuthState{Status: domain.StatusActive, PasswordChangedAt: &changed}}
	s := newStateService(repo)

	if _, err := s.Authenticate(context.Background(), accessFor(t, s, uuid.New())); err != nil {
		t.Fatalf("a token issued after the change must work: %v", err)
	}
}

// A user who changes the password and logs in within the same second must not
// be locked out: iat has one-second resolution.
func TestAuthenticateAcceptsTokenIssuedInSameSecondAsPasswordChange(t *testing.T) {
	s := newStateService(&stateRepo{})
	token := accessFor(t, s, uuid.New())

	claims, err := s.parse(token)
	if err != nil {
		t.Fatal(err)
	}
	iat := time.Unix(int64(claims["iat"].(float64)), 0)
	changed := iat.Add(900 * time.Millisecond) // later than the token, same second
	s.repo = &stateRepo{state: domain.AuthState{Status: domain.StatusActive, PasswordChangedAt: &changed}}

	if _, err := s.Authenticate(context.Background(), token); err != nil {
		t.Fatalf("same-second token must be accepted: %v", err)
	}
}

func TestAuthenticateRejectsRefreshTokenAndGarbage(t *testing.T) {
	s := newStateService(&stateRepo{state: activeState()})

	for name, token := range map[string]string{
		"refresh token": refreshFor(t, s, uuid.New()),
		"garbage":       "not-a-jwt",
		"empty":         "",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := s.Authenticate(context.Background(), token)
			requireAPIError(t, err, 401, "unauthorized")
		})
	}
}

// --- change password stamps the cutoff ------------------------------------

type passwordRepo struct {
	domain.Repository
	hash      string
	changedAt time.Time
}

func (r *passwordRepo) PasswordHash(ctx context.Context, id uuid.UUID) (string, error) {
	return r.hash, nil
}

func (r *passwordRepo) ChangePassword(ctx context.Context, id uuid.UUID, newHash string, changedAt time.Time) error {
	r.changedAt = changedAt
	return nil
}

func TestChangePasswordStampsCutoffWithCallerClock(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("OldPassword1"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	repo := &passwordRepo{hash: string(hash)}
	s := New(repo, "test-secret", time.Minute, time.Hour)

	before := time.Now()
	if err := s.ChangePassword(context.Background(), uuid.New(), "OldPassword1", "NewPassword2"); err != nil {
		t.Fatal(err)
	}

	if repo.changedAt.Before(before) || time.Since(repo.changedAt) > time.Minute {
		t.Fatalf("changedAt = %v, want about now", repo.changedAt)
	}
}
