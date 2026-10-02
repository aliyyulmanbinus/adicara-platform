package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidAuthInput = errors.New("invalid authentication input")
	ErrInvalidCSRF      = errors.New("invalid csrf token")
)

type AuthRepository interface {
	CreateUserWithSession(context.Context, string, string, string, []byte, []byte, time.Time) (domain.User, error)
	FindCredentialByEmail(context.Context, string) (domain.UserCredential, error)
	CreateSession(context.Context, string, []byte, []byte, time.Time) error
	FindSessionByTokenHash(context.Context, []byte) (domain.SessionIdentity, error)
	DeleteSessionByTokenHash(context.Context, []byte) error
}

type RegisterInput struct {
	Email       string
	DisplayName string
	Password    string
}

type LoginInput struct {
	Email    string
	Password string
}

type AuthResult struct {
	User      domain.User
	Session   string
	CSRFToken string
	ExpiresAt time.Time
}

type AuthService struct {
	repository AuthRepository
	sessionTTL time.Duration
	now        func() time.Time
}

func NewAuthService(repository AuthRepository, sessionTTL time.Duration) *AuthService {
	return &AuthService{repository: repository, sessionTTL: sessionTTL, now: time.Now}
}

func (s *AuthService) Register(ctx context.Context, input RegisterInput) (AuthResult, error) {
	email, displayName, err := validateRegistration(input)
	if err != nil {
		return AuthResult{}, err
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return AuthResult{}, fmt.Errorf("hash password: %w", err)
	}
	result, tokenHash, csrfHash, err := s.newAuthResult()
	if err != nil {
		return AuthResult{}, err
	}
	user, err := s.repository.CreateUserWithSession(
		ctx, email, displayName, string(passwordHash), tokenHash, csrfHash, result.ExpiresAt,
	)
	if err != nil {
		return AuthResult{}, err
	}
	result.User = user
	return result, nil
}

func (s *AuthService) Login(ctx context.Context, input LoginInput) (AuthResult, error) {
	email := normalizeEmail(input.Email)
	if !validEmail(email) || len(input.Password) == 0 || len(input.Password) > 128 {
		return AuthResult{}, domain.ErrInvalidCredentials
	}
	credential, err := s.repository.FindCredentialByEmail(ctx, email)
	if err != nil {
		return AuthResult{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(credential.PasswordHash), []byte(input.Password)) != nil {
		return AuthResult{}, domain.ErrInvalidCredentials
	}
	result, tokenHash, csrfHash, err := s.newAuthResult()
	if err != nil {
		return AuthResult{}, err
	}
	if err := s.repository.CreateSession(ctx, credential.ID, tokenHash, csrfHash, result.ExpiresAt); err != nil {
		return AuthResult{}, err
	}
	result.User = credential.User
	return result, nil
}

func (s *AuthService) Authenticate(ctx context.Context, rawSessionToken string) (domain.SessionIdentity, error) {
	if rawSessionToken == "" {
		return domain.SessionIdentity{}, domain.ErrInvalidSession
	}
	return s.repository.FindSessionByTokenHash(ctx, hashToken(rawSessionToken))
}

func (s *AuthService) ValidateCSRF(identity domain.SessionIdentity, rawCSRFToken string) error {
	if rawCSRFToken == "" {
		return ErrInvalidCSRF
	}
	actual := hashToken(rawCSRFToken)
	if subtle.ConstantTimeCompare(actual, identity.CSRFTokenHash) != 1 {
		return ErrInvalidCSRF
	}
	return nil
}

func (s *AuthService) Logout(ctx context.Context, identity domain.SessionIdentity) error {
	return s.repository.DeleteSessionByTokenHash(ctx, identity.TokenHash)
}

func (s *AuthService) newAuthResult() (AuthResult, []byte, []byte, error) {
	sessionToken, err := randomToken()
	if err != nil {
		return AuthResult{}, nil, nil, fmt.Errorf("create session token: %w", err)
	}
	csrfToken, err := randomToken()
	if err != nil {
		return AuthResult{}, nil, nil, fmt.Errorf("create csrf token: %w", err)
	}
	return AuthResult{
		Session:   sessionToken,
		CSRFToken: csrfToken,
		ExpiresAt: s.now().Add(s.sessionTTL),
	}, hashToken(sessionToken), hashToken(csrfToken), nil
}

func validateRegistration(input RegisterInput) (string, string, error) {
	email := normalizeEmail(input.Email)
	displayName := strings.TrimSpace(input.DisplayName)
	if !validEmail(email) || len(email) > 254 {
		return "", "", ErrInvalidAuthInput
	}
	if len(displayName) < 2 || len(displayName) > 100 {
		return "", "", ErrInvalidAuthInput
	}
	if len(input.Password) < 12 || len(input.Password) > 128 {
		return "", "", ErrInvalidAuthInput
	}
	return email, displayName, nil
}

func normalizeEmail(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func validEmail(value string) bool {
	address, err := mail.ParseAddress(value)
	return err == nil && address.Address == value && strings.Contains(value, "@")
}

func randomToken() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func hashToken(raw string) []byte {
	sum := sha256.Sum256([]byte(raw))
	return sum[:]
}
