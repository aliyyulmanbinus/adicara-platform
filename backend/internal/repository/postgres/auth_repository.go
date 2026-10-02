package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthRepository struct {
	pool *pgxpool.Pool
}

func NewAuthRepository(pool *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{pool: pool}
}

func (r *AuthRepository) CreateUserWithSession(
	ctx context.Context,
	email, displayName, passwordHash string,
	tokenHash, csrfTokenHash []byte,
	expiresAt time.Time,
) (domain.User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.User{}, fmt.Errorf("begin registration: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const userQuery = `
		INSERT INTO users (email, display_name, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id::text, email, display_name, created_at, updated_at`
	var user domain.User
	err = tx.QueryRow(ctx, userQuery, email, displayName, passwordHash).Scan(
		&user.ID, &user.Email, &user.DisplayName, &user.CreatedAt, &user.UpdatedAt,
	)
	if isUniqueViolation(err) {
		return domain.User{}, domain.ErrEmailConflict
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("insert user: %w", err)
	}

	if err := insertSession(ctx, tx, user.ID, tokenHash, csrfTokenHash, expiresAt); err != nil {
		return domain.User{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.User{}, fmt.Errorf("commit registration: %w", err)
	}
	return user, nil
}

func (r *AuthRepository) FindCredentialByEmail(ctx context.Context, email string) (domain.UserCredential, error) {
	const query = `
		SELECT id::text, email, display_name, created_at, updated_at, password_hash
		FROM users
		WHERE email = $1`
	var credential domain.UserCredential
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&credential.ID,
		&credential.Email,
		&credential.DisplayName,
		&credential.CreatedAt,
		&credential.UpdatedAt,
		&credential.PasswordHash,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.UserCredential{}, domain.ErrInvalidCredentials
	}
	if err != nil {
		return domain.UserCredential{}, fmt.Errorf("query credential: %w", err)
	}
	return credential, nil
}

func (r *AuthRepository) CreateSession(
	ctx context.Context,
	userID string,
	tokenHash, csrfTokenHash []byte,
	expiresAt time.Time,
) error {
	if err := insertSession(ctx, r.pool, userID, tokenHash, csrfTokenHash, expiresAt); err != nil {
		return err
	}
	return nil
}

func (r *AuthRepository) FindSessionByTokenHash(ctx context.Context, tokenHash []byte) (domain.SessionIdentity, error) {
	const query = `
		SELECT u.id::text, u.email, u.display_name, u.created_at, u.updated_at,
		       s.token_hash, s.csrf_token_hash, s.expires_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1 AND s.expires_at > NOW()`
	var identity domain.SessionIdentity
	err := r.pool.QueryRow(ctx, query, tokenHash).Scan(
		&identity.User.ID,
		&identity.User.Email,
		&identity.User.DisplayName,
		&identity.User.CreatedAt,
		&identity.User.UpdatedAt,
		&identity.TokenHash,
		&identity.CSRFTokenHash,
		&identity.ExpiresAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.SessionIdentity{}, domain.ErrInvalidSession
	}
	if err != nil {
		return domain.SessionIdentity{}, fmt.Errorf("query session: %w", err)
	}
	return identity, nil
}

func (r *AuthRepository) DeleteSessionByTokenHash(ctx context.Context, tokenHash []byte) error {
	if _, err := r.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

type sessionExecer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func insertSession(
	ctx context.Context,
	execer sessionExecer,
	userID string,
	tokenHash, csrfTokenHash []byte,
	expiresAt time.Time,
) error {
	const query = `
		INSERT INTO sessions (user_id, token_hash, csrf_token_hash, expires_at)
		VALUES ($1, $2, $3, $4)`
	if _, err := execer.Exec(ctx, query, userID, tokenHash, csrfTokenHash, expiresAt); err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
