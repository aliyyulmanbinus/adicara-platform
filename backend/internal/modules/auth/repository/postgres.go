package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/auth/domain"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateUser(ctx context.Context, id uuid.UUID, email, username, passwordHash string) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO users (id, email, password_hash, username, role_id, status) VALUES ($1,$2,$3,$4,$5,$6)`,
		id, email, passwordHash, username, domain.RoleCustomer, domain.StatusActive)
	if err != nil {
		var pg *pgconn.PgError
		if errors.As(err, &pg) && pg.Code == "23505" {
			if pg.ConstraintName == "users_username_lower_idx" {
				return domain.ErrUsernameTaken
			}
			return domain.ErrEmailTaken
		}
		return err
	}

	return nil
}

func (r *PostgresRepository) UserByIdentifier(ctx context.Context, identifier string) (domain.User, string, error) {
	var user domain.User
	var hash string

	err := r.pool.QueryRow(ctx, `
		SELECT id, email, username, name, role_id, status, password_hash FROM users
		WHERE lower(email)=$1 OR lower(username)=$1
		LIMIT 1`, identifier).
		Scan(&user.ID, &user.Email, &user.Username, &user.Name, &user.RoleID, &user.Status, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, "", domain.ErrUserNotFound
	}

	return user, hash, err
}

func (r *PostgresRepository) UserByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	var user domain.User

	err := r.pool.QueryRow(ctx, `SELECT id, email, username, name, role_id, status FROM users WHERE id=$1`, id).
		Scan(&user.ID, &user.Email, &user.Username, &user.Name, &user.RoleID, &user.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrUserNotFound
	}

	return user, err
}

func (r *PostgresRepository) UserAuthState(ctx context.Context, id uuid.UUID) (domain.AuthState, error) {
	var state domain.AuthState

	err := r.pool.QueryRow(ctx, `SELECT status, password_changed_at FROM users WHERE id=$1`, id).
		Scan(&state.Status, &state.PasswordChangedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.AuthState{}, domain.ErrUserNotFound
	}

	return state, err
}

func (r *PostgresRepository) UpdateUserName(ctx context.Context, id uuid.UUID, name string) (domain.User, error) {
	var user domain.User

	err := r.pool.QueryRow(ctx, `
		UPDATE users SET name=$2, updated_at=now() WHERE id=$1
		RETURNING id, email, username, name, role_id, status`, id, name).
		Scan(&user.ID, &user.Email, &user.Username, &user.Name, &user.RoleID, &user.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrUserNotFound
	}

	return user, err
}

func (r *PostgresRepository) PasswordHash(ctx context.Context, id uuid.UUID) (string, error) {
	var hash string

	err := r.pool.QueryRow(ctx, `SELECT password_hash FROM users WHERE id=$1`, id).Scan(&hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrUserNotFound
	}

	return hash, err
}

func (r *PostgresRepository) ChangePassword(ctx context.Context, id uuid.UUID, newHash string, changedAt time.Time) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `UPDATE users SET password_hash=$2, password_changed_at=$3, updated_at=now() WHERE id=$1`, id, newHash, changedAt); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE refresh_sessions SET revoked_at=now()
		WHERE user_id=$1 AND revoked_at IS NULL`, id); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepository) CreateRefreshSession(ctx context.Context, sessionID, userID uuid.UUID, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO refresh_sessions (id, user_id, expires_at) VALUES ($1,$2,$3)`,
		sessionID, userID, expiresAt)
	return err
}

// ConsumeRefreshSession rotates the pair: the presented session is spent, so
// a refresh token that leaks and is replayed after the owner refreshed is
// rejected via ErrSessionInactive. rotated_at marks it as spent by rotation
// (unlike logout or a password change) so a later replay can be told apart.
func (r *PostgresRepository) ConsumeRefreshSession(ctx context.Context, sessionID, userID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE refresh_sessions SET revoked_at=now(), rotated_at=now()
		WHERE id=$1 AND user_id=$2 AND revoked_at IS NULL AND expires_at > now()`, sessionID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrSessionInactive
	}

	return nil
}

// RevokeUserSessionsOnReuse is the reuse-detection half of rotation. Both the
// grace comparison and rotated_at use the database clock, so they agree. A
// session that was never rotated (logout, password change, expiry) has a NULL
// rotated_at and so never triggers it.
func (r *PostgresRepository) RevokeUserSessionsOnReuse(ctx context.Context, sessionID, userID uuid.UUID, grace time.Duration) (int64, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE refresh_sessions SET revoked_at=now()
		WHERE user_id=$2 AND revoked_at IS NULL
		  AND EXISTS (
		    SELECT 1 FROM refresh_sessions
		    WHERE id=$1 AND user_id=$2
		      AND rotated_at < now() - make_interval(secs => $3::double precision))`,
		sessionID, userID, grace.Seconds())
	if err != nil {
		return 0, err
	}

	return tag.RowsAffected(), nil
}

// RevokeRefreshSession is used by logout: unlike ConsumeRefreshSession it is
// lenient about an already-gone session, matching prior logout behaviour.
func (r *PostgresRepository) RevokeRefreshSession(ctx context.Context, sessionID, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE refresh_sessions SET revoked_at=now()
		WHERE id=$1 AND user_id=$2 AND revoked_at IS NULL`, sessionID, userID)
	return err
}

// DeleteExpiredSessions drops rows that can no longer authorise anything.
func (r *PostgresRepository) DeleteExpiredSessions(ctx context.Context) (int64, error) {
	tag, err := r.pool.Exec(ctx, `DELETE FROM refresh_sessions WHERE expires_at < now()`)
	if err != nil {
		return 0, err
	}

	return tag.RowsAffected(), nil
}
