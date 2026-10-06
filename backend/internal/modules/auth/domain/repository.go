package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrEmailTaken      = errors.New("email already registered")
	ErrUsernameTaken   = errors.New("username already registered")
	ErrUserNotFound    = errors.New("user not found")
	ErrSessionInactive = errors.New("refresh session is not active")
)

type Repository interface {
	CreateUser(ctx context.Context, id uuid.UUID, email, username, passwordHash string) error
	UserByIdentifier(ctx context.Context, identifier string) (User, string, error)
	UserByID(ctx context.Context, id uuid.UUID) (User, error)
	UserAuthState(ctx context.Context, id uuid.UUID) (AuthState, error)
	UpdateUserName(ctx context.Context, id uuid.UUID, name string) (User, error)
	PasswordHash(ctx context.Context, id uuid.UUID) (string, error)
	// ChangePassword stores the new hash, stamps password_changed_at with
	// changedAt (the caller's clock, the same one that stamps token iat), and
	// revokes every refresh session, all in one transaction.
	ChangePassword(ctx context.Context, id uuid.UUID, newHash string, changedAt time.Time) error

	CreateRefreshSession(ctx context.Context, sessionID, userID uuid.UUID, expiresAt time.Time) error
	// ConsumeRefreshSession spends an active session by rotation, recording
	// rotated_at. It returns ErrSessionInactive when there is nothing to spend.
	ConsumeRefreshSession(ctx context.Context, sessionID, userID uuid.UUID) error
	// RevokeUserSessionsOnReuse revokes every active session of userID, but only
	// if sessionID was spent by rotation more than grace ago. It returns how
	// many sessions it revoked.
	RevokeUserSessionsOnReuse(ctx context.Context, sessionID, userID uuid.UUID, grace time.Duration) (int64, error)
	RevokeRefreshSession(ctx context.Context, sessionID, userID uuid.UUID) error
	DeleteExpiredSessions(ctx context.Context) (int64, error)
}
