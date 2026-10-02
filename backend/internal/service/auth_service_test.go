package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type stubAuthRepository struct {
	credential    domain.UserCredential
	identity      domain.SessionIdentity
	createdUser   domain.User
	passwordHash  string
	tokenHash     []byte
	csrfTokenHash []byte
}

func (s *stubAuthRepository) CreateUserWithSession(_ context.Context, email, displayName, passwordHash string, tokenHash, csrfHash []byte, _ time.Time) (domain.User, error) {
	s.passwordHash = passwordHash
	s.tokenHash = tokenHash
	s.csrfTokenHash = csrfHash
	if s.createdUser.ID == "" {
		s.createdUser = domain.User{ID: "00000000-0000-4000-8000-000000000001", Email: email, DisplayName: displayName}
	}
	return s.createdUser, nil
}

func (s *stubAuthRepository) FindCredentialByEmail(context.Context, string) (domain.UserCredential, error) {
	if s.credential.ID == "" {
		return domain.UserCredential{}, domain.ErrInvalidCredentials
	}
	return s.credential, nil
}

func (s *stubAuthRepository) CreateSession(_ context.Context, _ string, tokenHash, csrfHash []byte, _ time.Time) error {
	s.tokenHash = tokenHash
	s.csrfTokenHash = csrfHash
	return nil
}

func (s *stubAuthRepository) FindSessionByTokenHash(context.Context, []byte) (domain.SessionIdentity, error) {
	return s.identity, nil
}

func (s *stubAuthRepository) DeleteSessionByTokenHash(context.Context, []byte) error { return nil }

func TestRegisterNormalizesEmailAndHashesPassword(t *testing.T) {
	repository := &stubAuthRepository{}
	auth := NewAuthService(repository, time.Hour)
	auth.now = func() time.Time { return time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC) }

	result, err := auth.Register(context.Background(), RegisterInput{
		Email: "  NARA@Example.COM ", DisplayName: "Nara", Password: "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.User.Email != "nara@example.com" {
		t.Fatalf("email was not normalized: %q", result.User.Email)
	}
	if repository.passwordHash == "correct horse battery staple" {
		t.Fatal("password was stored in plaintext")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(repository.passwordHash), []byte("correct horse battery staple")); err != nil {
		t.Fatalf("stored password hash is invalid: %v", err)
	}
	if result.Session == "" || result.CSRFToken == "" || string(repository.tokenHash) == result.Session {
		t.Fatal("session and csrf tokens must be random and stored only as hashes")
	}
}

func TestRegisterRejectsShortPassword(t *testing.T) {
	auth := NewAuthService(&stubAuthRepository{}, time.Hour)
	_, err := auth.Register(context.Background(), RegisterInput{Email: "nara@example.com", DisplayName: "Nara", Password: "short"})
	if !errors.Is(err, ErrInvalidAuthInput) {
		t.Fatalf("expected ErrInvalidAuthInput, got %v", err)
	}
}

func TestValidateCSRFUsesStoredHash(t *testing.T) {
	auth := NewAuthService(&stubAuthRepository{}, time.Hour)
	identity := domain.SessionIdentity{CSRFTokenHash: hashToken("expected-token")}
	if err := auth.ValidateCSRF(identity, "expected-token"); err != nil {
		t.Fatalf("expected valid csrf token: %v", err)
	}
	if !errors.Is(auth.ValidateCSRF(identity, "different-token"), ErrInvalidCSRF) {
		t.Fatal("expected different csrf token to fail")
	}
}
