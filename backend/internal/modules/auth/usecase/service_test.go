package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/auth/domain"
)

func testService() *Service {
	return New(nil, "test-secret", time.Minute, time.Hour)
}

type fakeRepo struct {
	domain.Repository
	user domain.User
	hash string
}

func (r *fakeRepo) UserByIdentifier(ctx context.Context, identifier string) (domain.User, string, error) {
	if identifier != r.user.Email {
		return domain.User{}, "", domain.ErrUserNotFound
	}

	return r.user, r.hash, nil
}

func (r *fakeRepo) CreateRefreshSession(ctx context.Context, sessionID, userID uuid.UUID, expiresAt time.Time) error {
	return nil
}

func TestLoginRejectsInactiveAccount(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("Password123"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}

	repo := &fakeRepo{
		user: domain.User{ID: uuid.New(), Email: "user@example.com", Status: domain.StatusActive},
		hash: string(hash),
	}
	s := New(repo, "test-secret", time.Minute, time.Hour)

	repo.user.Status = "inactive"
	if _, _, err := s.Login(context.Background(), "user@example.com", "Password123"); err == nil {
		t.Fatal("login dengan akun inactive harus ditolak")
	}

	repo.user.Status = domain.StatusActive
	if _, _, err := s.Login(context.Background(), "user@example.com", "Password123"); err != nil {
		t.Fatalf("login dengan akun active harus lolos, dapat error: %v", err)
	}
}

func TestParseRefreshRejectsTokenWithoutSession(t *testing.T) {
	s := testService()
	// Tokens issued before refresh sessions existed carry no jti.
	token, err := s.sign(uuid.New(), "refresh", time.Hour, uuid.Nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.parseRefresh(token); err == nil {
		t.Fatal("a refresh token without jti must be rejected")
	}
}

func TestParseRefreshRejectsAccessToken(t *testing.T) {
	s := testService()
	token, err := s.sign(uuid.New(), "access", time.Minute, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.parseRefresh(token); err == nil {
		t.Fatal("an access token must not be usable as a refresh token")
	}
}

func TestParseRefreshAcceptsIssuedPair(t *testing.T) {
	s := testService()
	userID, sessionID := uuid.New(), uuid.New()
	token, err := s.sign(userID, "refresh", time.Hour, sessionID)
	if err != nil {
		t.Fatal(err)
	}
	gotUser, gotSession, err := s.parseRefresh(token)
	if err != nil {
		t.Fatal(err)
	}
	if gotUser != userID || gotSession != sessionID {
		t.Fatalf("got %s/%s, want %s/%s", gotUser, gotSession, userID, sessionID)
	}
}

// createRepo accepts every CreateUser so a test only fails when validation
// lets bad input through to hashing/storage.
type createRepo struct {
	domain.Repository
	created int
}

func (r *createRepo) CreateUser(ctx context.Context, id uuid.UUID, email, username, passwordHash string) error {
	r.created++
	return nil
}

func requireAPIError(t *testing.T, err error, status int, code string) {
	t.Helper()
	var ae *apierr.Error
	if !errors.As(err, &ae) {
		t.Fatalf("want *apierr.Error %d/%s, got %T: %v", status, code, err, err)
	}
	if ae.Status != status || ae.Code != code {
		t.Fatalf("got %d/%s, want %d/%s", ae.Status, ae.Code, status, code)
	}
}

// bcrypt refuses passwords over 72 bytes; that must surface as a 400, not a 500.
func TestRegisterRejectsPasswordBcryptCannotHash(t *testing.T) {
	repo := &createRepo{}
	s := New(repo, "test-secret", time.Minute, time.Hour)

	long := strings.Repeat("a1", 37) // 74 bytes
	_, err := s.Register(context.Background(), "user@example.com", long, "valid.user")
	requireAPIError(t, err, 400, "password_too_long")
	if repo.created != 0 {
		t.Fatal("user must not be created")
	}

	// exactly 72 bytes is the largest accepted password
	if _, err := s.Register(context.Background(), "user@example.com", strings.Repeat("a1", 36), "valid.user"); err != nil {
		t.Fatalf("72-byte password must be accepted: %v", err)
	}
}

func TestChangePasswordRejectsPasswordBcryptCannotHash(t *testing.T) {
	s := New(&createRepo{}, "test-secret", time.Minute, time.Hour)
	err := s.ChangePassword(context.Background(), uuid.New(), "whatever1", strings.Repeat("a1", 37))
	requireAPIError(t, err, 400, "password_too_long")
}

func TestRegisterValidatesEmailAndUsernameShape(t *testing.T) {
	tests := []struct {
		name, email, username, code string
	}{
		{"email without at", "not-an-email", "valid.user", "invalid_email"},
		{"email without domain dot", "user@localhost", "valid.user", "invalid_email"},
		{"email with display name", "Bob <bob@example.com>", "valid.user", "invalid_email"},
		{"email too long", strings.Repeat("a", 250) + "@example.com", "valid.user", "invalid_email"},
		{"username too long", "user@example.com", strings.Repeat("a", 31), "invalid_username"},
		{"username too short", "user@example.com", "ab", "invalid_username"},
		{"username with space", "user@example.com", "nara arka", "invalid_username"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &createRepo{}
			s := New(repo, "test-secret", time.Minute, time.Hour)
			_, err := s.Register(context.Background(), tt.email, "Password123", tt.username)
			requireAPIError(t, err, 400, tt.code)
			if repo.created != 0 {
				t.Fatal("user must not be created")
			}
		})
	}
}
