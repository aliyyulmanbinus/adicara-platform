package repository

import (
	"context"
	"errors"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/migrate"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/auth/domain"
)

// These tests need a real PostgreSQL and WIPE the public schema of the database
// they run against (migrate.Fresh), so they only run when TEST_DATABASE_URL
// points at a database whose name ends in "_test". CI has no database and
// skips them.
func testRepo(t *testing.T) (*PostgresRepository, *pgxpool.Pool) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if name := strings.TrimPrefix(u.Path, "/"); !strings.HasSuffix(name, "_test") {
		t.Fatalf("refusing to wipe database %q: its name must end in _test", name)
	}
	if err := migrate.Fresh(dsn); err != nil {
		t.Fatal(err)
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	return NewPostgresRepository(pool), pool
}

func newUser(t *testing.T, r *PostgresRepository, name string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	if err := r.CreateUser(context.Background(), id, name+"@example.com", name, "hash"); err != nil {
		t.Fatal(err)
	}

	return id
}

func newSession(t *testing.T, r *PostgresRepository, userID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	if err := r.CreateRefreshSession(context.Background(), id, userID, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	return id
}

func activeSessions(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID) int {
	t.Helper()
	var n int
	err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM refresh_sessions WHERE user_id=$1 AND revoked_at IS NULL`, userID).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}

	return n
}

func ageRotation(t *testing.T, pool *pgxpool.Pool, sessionID uuid.UUID, age time.Duration) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`UPDATE refresh_sessions SET rotated_at = now() - make_interval(secs => $2::double precision) WHERE id=$1`,
		sessionID, age.Seconds())
	if err != nil {
		t.Fatal(err)
	}
}

func TestConsumeRefreshSessionMarksRotation(t *testing.T) {
	r, pool := testRepo(t)
	ctx := context.Background()
	user := newUser(t, r, "ani")
	session := newSession(t, r, user)

	if err := r.ConsumeRefreshSession(ctx, session, user); err != nil {
		t.Fatal(err)
	}

	var revoked, rotated *time.Time
	if err := pool.QueryRow(ctx, `SELECT revoked_at, rotated_at FROM refresh_sessions WHERE id=$1`, session).Scan(&revoked, &rotated); err != nil {
		t.Fatal(err)
	}
	if revoked == nil || rotated == nil {
		t.Fatalf("rotation must set both revoked_at and rotated_at, got %v / %v", revoked, rotated)
	}

	if err := r.ConsumeRefreshSession(ctx, session, user); !errors.Is(err, domain.ErrSessionInactive) {
		t.Fatalf("a spent session must not be consumable again, got %v", err)
	}
}

func TestRevokeUserSessionsOnReuse(t *testing.T) {
	ctx := context.Background()
	grace := 10 * time.Second

	t.Run("a fresh rotation is only a race: nothing is revoked", func(t *testing.T) {
		r, pool := testRepo(t)
		user := newUser(t, r, "ani")
		spent, other := newSession(t, r, user), newSession(t, r, user)
		if err := r.ConsumeRefreshSession(ctx, spent, user); err != nil {
			t.Fatal(err)
		}

		n, err := r.RevokeUserSessionsOnReuse(ctx, spent, user, grace)
		if err != nil {
			t.Fatal(err)
		}
		if n != 0 || activeSessions(t, pool, user) != 1 {
			t.Fatalf("revoked %d, %d still active; want 0 revoked and the other session alive", n, activeSessions(t, pool, user))
		}
		_ = other
	})

	t.Run("a stale rotation is theft: every active session of that user dies", func(t *testing.T) {
		r, pool := testRepo(t)
		user, bystander := newUser(t, r, "ani"), newUser(t, r, "budi")
		spent := newSession(t, r, user)
		newSession(t, r, user)
		newSession(t, r, user)
		newSession(t, r, bystander)
		if err := r.ConsumeRefreshSession(ctx, spent, user); err != nil {
			t.Fatal(err)
		}
		ageRotation(t, pool, spent, time.Minute)

		n, err := r.RevokeUserSessionsOnReuse(ctx, spent, user, grace)
		if err != nil {
			t.Fatal(err)
		}
		if n != 2 || activeSessions(t, pool, user) != 0 {
			t.Fatalf("revoked %d, %d still active; want both live sessions revoked", n, activeSessions(t, pool, user))
		}
		if activeSessions(t, pool, bystander) != 1 {
			t.Fatal("another user's session must not be touched")
		}
	})

	t.Run("a logged-out session never triggers it", func(t *testing.T) {
		r, pool := testRepo(t)
		user := newUser(t, r, "ani")
		loggedOut := newSession(t, r, user)
		newSession(t, r, user)
		if err := r.RevokeRefreshSession(ctx, loggedOut, user); err != nil {
			t.Fatal(err)
		}

		n, err := r.RevokeUserSessionsOnReuse(ctx, loggedOut, user, grace)
		if err != nil {
			t.Fatal(err)
		}
		if n != 0 || activeSessions(t, pool, user) != 1 {
			t.Fatalf("replaying a logged-out token revoked %d sessions", n)
		}
	})

	t.Run("a session of another user cannot be used to revoke someone else", func(t *testing.T) {
		r, pool := testRepo(t)
		attacker, victim := newUser(t, r, "ani"), newUser(t, r, "budi")
		spent := newSession(t, r, attacker)
		newSession(t, r, victim)
		if err := r.ConsumeRefreshSession(ctx, spent, attacker); err != nil {
			t.Fatal(err)
		}
		ageRotation(t, pool, spent, time.Minute)

		n, err := r.RevokeUserSessionsOnReuse(ctx, spent, victim, grace)
		if err != nil {
			t.Fatal(err)
		}
		if n != 0 || activeSessions(t, pool, victim) != 1 {
			t.Fatalf("revoked %d of the victim's sessions via someone else's session id", n)
		}
	})

	t.Run("an unknown session id revokes nothing", func(t *testing.T) {
		r, pool := testRepo(t)
		user := newUser(t, r, "ani")
		newSession(t, r, user)

		n, err := r.RevokeUserSessionsOnReuse(ctx, uuid.New(), user, grace)
		if err != nil {
			t.Fatal(err)
		}
		if n != 0 || activeSessions(t, pool, user) != 1 {
			t.Fatalf("revoked %d sessions for an unknown id", n)
		}
	})
}

func TestChangePasswordStampsCutoffAndRevokesSessions(t *testing.T) {
	r, pool := testRepo(t)
	ctx := context.Background()
	user := newUser(t, r, "ani")
	newSession(t, r, user)
	newSession(t, r, user)

	state, err := r.UserAuthState(ctx, user)
	if err != nil {
		t.Fatal(err)
	}
	if state.Status != domain.StatusActive || state.PasswordChangedAt != nil {
		t.Fatalf("a new account is active with no cutoff, got %+v", state)
	}

	changedAt := time.Now().Truncate(time.Second)
	if err := r.ChangePassword(ctx, user, "new-hash", changedAt); err != nil {
		t.Fatal(err)
	}

	state, err = r.UserAuthState(ctx, user)
	if err != nil {
		t.Fatal(err)
	}
	if state.PasswordChangedAt == nil || !state.PasswordChangedAt.Equal(changedAt) {
		t.Fatalf("password_changed_at = %v, want %v (the caller's clock)", state.PasswordChangedAt, changedAt)
	}
	if activeSessions(t, pool, user) != 0 {
		t.Fatal("a password change must revoke every refresh session")
	}
	var rotated int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM refresh_sessions WHERE user_id=$1 AND rotated_at IS NOT NULL`, user).Scan(&rotated); err != nil {
		t.Fatal(err)
	}
	if rotated != 0 {
		t.Fatal("sessions revoked by a password change are not rotations, or replaying them would look like theft")
	}
}

func TestUserAuthStateUnknownUser(t *testing.T) {
	r, _ := testRepo(t)

	if _, err := r.UserAuthState(context.Background(), uuid.New()); !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("got %v, want ErrUserNotFound", err)
	}
}
