package migrate

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/pressly/goose/v3"
)

// These tests need a real PostgreSQL and WIPE the public schema of the database
// they run against, so they only run when TEST_DATABASE_URL points at a
// database whose name ends in "_test". CI has no database and skips them.
func testDB(t *testing.T) (string, *sql.DB) {
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
	db, closeDB, err := open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(closeDB)
	wipe(t, db)

	return dsn, db
}

func wipe(t *testing.T, db *sql.DB) {
	t.Helper()
	rows, err := db.Query(`SELECT nspname FROM pg_namespace WHERE nspname LIKE 'legacy\_%'`)
	if err != nil {
		t.Fatal(err)
	}
	var schemas []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		schemas = append(schemas, s)
	}
	rows.Close()
	for _, s := range schemas {
		mustExec(t, db, `DROP SCHEMA "`+s+`" CASCADE`)
	}
	mustExec(t, db, `DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;`)
}

func mustExec(t *testing.T, db *sql.DB, q string) {
	t.Helper()
	if _, err := db.Exec(q); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
}

func count(t *testing.T, db *sql.DB, q string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(q).Scan(&n); err != nil {
		t.Fatalf("%s: %v", q, err)
	}

	return n
}

// legacyFixture is the shape the pre-goose backend left behind: a users table
// with display_name, tables that reference it, and an index whose name the new
// schema also uses (so it only works if indexes move with their tables).
func legacyFixture(t *testing.T, db *sql.DB) {
	t.Helper()
	mustExec(t, db, `
		CREATE TABLE users (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), email text NOT NULL, display_name text NOT NULL, password_hash text NOT NULL);
		CREATE UNIQUE INDEX users_email_lower_idx ON users (lower(email));
		CREATE TABLE invitations (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), user_id uuid REFERENCES users(id), slug text NOT NULL UNIQUE, title text NOT NULL);
		CREATE TABLE sessions (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE);
		INSERT INTO users (email, display_name, password_hash) VALUES ('ani@example.com', 'Ani', 'hash');
		INSERT INTO invitations (user_id, slug, title) SELECT id, 'ani-dan-budi', 'Ani dan Budi' FROM users;`)
}

func legacySchema(t *testing.T, db *sql.DB) string {
	t.Helper()
	if n := count(t, db, `SELECT count(*) FROM pg_namespace WHERE nspname LIKE 'legacy\_%'`); n != 1 {
		t.Fatalf("want exactly 1 legacy schema, got %d", n)
	}
	var name string
	if err := db.QueryRow(`SELECT nspname FROM pg_namespace WHERE nspname LIKE 'legacy\_%'`).Scan(&name); err != nil {
		t.Fatal(err)
	}

	return name
}

// latestVersion is the newest migration in sql/; bump it with each new file.
const latestVersion = 12

func assertCurrentSchema(t *testing.T, db *sql.DB) {
	t.Helper()
	if v := count(t, db, `SELECT max(version_id) FROM goose_db_version`); v != latestVersion {
		t.Fatalf("goose version = %d, want %d", v, latestVersion)
	}
	if n := count(t, db, `SELECT count(*) FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'users' AND column_name = 'username'`); n != 1 {
		t.Fatal("public.users is not the new schema")
	}
}

func TestUpArchivesLegacySchemaInsteadOfFailing(t *testing.T) {
	dsn, db := testDB(t)
	legacyFixture(t, db)

	if err := Up(dsn); err != nil {
		t.Fatalf("Up on a legacy database must succeed, got: %v", err)
	}

	assertCurrentSchema(t, db)
	if n := count(t, db, `SELECT count(*) FROM users`); n != 0 {
		t.Fatalf("new users table should start empty, has %d rows", n)
	}
	archive := legacySchema(t, db)
	if n := count(t, db, `SELECT count(*) FROM "`+archive+`".users`); n != 1 {
		t.Fatalf("legacy users must be preserved, got %d rows", n)
	}
	if n := count(t, db, `SELECT count(*) FROM "`+archive+`".invitations`); n != 1 {
		t.Fatalf("legacy invitations must be preserved, got %d rows", n)
	}
	if n := count(t, db, `SELECT count(*) FROM pg_tables WHERE schemaname = 'public' AND tablename IN ('invitations', 'sessions')`); n != 0 {
		t.Fatal("legacy tables must no longer be in public")
	}

	// A second start must not archive again.
	if err := Up(dsn); err != nil {
		t.Fatal(err)
	}
	legacySchema(t, db)
}

// The state the old behaviour left after a failed start: m_role created and
// goose at version 1, with the legacy tables still in public.
func TestUpRecoversFromPartiallyMigratedLegacyDatabase(t *testing.T) {
	dsn, db := testDB(t)
	legacyFixture(t, db)
	if err := goose.UpTo(db, "sql", 1); err != nil {
		t.Fatal(err)
	}

	if err := Up(dsn); err != nil {
		t.Fatalf("Up must recover a half-migrated legacy database, got: %v", err)
	}

	assertCurrentSchema(t, db)
	archive := legacySchema(t, db)
	if n := count(t, db, `SELECT count(*) FROM "`+archive+`".users`); n != 1 {
		t.Fatalf("legacy users must be preserved, got %d rows", n)
	}
}

func TestUpOnEmptyDatabaseArchivesNothing(t *testing.T) {
	dsn, db := testDB(t)

	if err := Up(dsn); err != nil {
		t.Fatal(err)
	}

	assertCurrentSchema(t, db)
	if n := count(t, db, `SELECT count(*) FROM pg_namespace WHERE nspname LIKE 'legacy\_%'`); n != 0 {
		t.Fatalf("no legacy schema expected, got %d", n)
	}
}

// Guards against a false positive: a healthy current database, with data, must
// never be mistaken for a legacy one.
func TestUpKeepsDataInCurrentSchema(t *testing.T) {
	dsn, db := testDB(t)
	if err := Up(dsn); err != nil {
		t.Fatal(err)
	}
	mustExec(t, db, `INSERT INTO users (id, email, password_hash, username) VALUES (gen_random_uuid(), 'a@example.com', 'h', 'ani')`)

	if err := Up(dsn); err != nil {
		t.Fatal(err)
	}

	if n := count(t, db, `SELECT count(*) FROM users`); n != 1 {
		t.Fatalf("users row must survive a restart, got %d", n)
	}
	if n := count(t, db, `SELECT count(*) FROM pg_namespace WHERE nspname LIKE 'legacy\_%'`); n != 0 {
		t.Fatal("a current database must not be archived")
	}
}

// The production upgrade path: a database already at version 3 with accounts
// and sessions in it. Migration 004 must keep every row and leave the new
// columns empty, and rolling it back must not touch the data either.
func TestAuthHardeningMigrationKeepsExistingData(t *testing.T) {
	dsn, db := testDB(t)
	if err := goose.UpTo(db, "sql", 3); err != nil {
		t.Fatal(err)
	}
	mustExec(t, db, `
		INSERT INTO users (id, email, password_hash, username) VALUES ('11111111-1111-1111-1111-111111111111', 'a@example.com', 'h', 'ani');
		INSERT INTO refresh_sessions (id, user_id, expires_at, revoked_at) VALUES
			('22222222-2222-2222-2222-222222222222', '11111111-1111-1111-1111-111111111111', now() + interval '1 day', NULL),
			('33333333-3333-3333-3333-333333333333', '11111111-1111-1111-1111-111111111111', now() + interval '1 day', now());`)

	if err := Up(dsn); err != nil {
		t.Fatalf("upgrading a populated version-3 database must succeed: %v", err)
	}

	assertCurrentSchema(t, db)
	if n := count(t, db, `SELECT count(*) FROM users WHERE password_changed_at IS NULL`); n != 1 {
		t.Fatalf("existing user must keep a NULL password_changed_at, %d rows match", n)
	}
	if n := count(t, db, `SELECT count(*) FROM refresh_sessions WHERE rotated_at IS NULL`); n != 2 {
		t.Fatalf("existing sessions must keep a NULL rotated_at, %d rows match", n)
	}

	// Back to version 3: this also runs every planner Down (005-012) before 004's.
	if err := goose.DownTo(db, "sql", 3); err != nil {
		t.Fatalf("rolling back 004 (and the planner migrations above it) must work: %v", err)
	}
	if n := count(t, db, `SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND column_name IN ('password_changed_at','rotated_at')`); n != 0 {
		t.Fatal("the down migration must remove both columns")
	}
	if n := count(t, db, `SELECT count(*) FROM users`) + count(t, db, `SELECT count(*) FROM refresh_sessions`); n != 3 {
		t.Fatalf("rolling back must keep the rows, got %d", n)
	}

	if err := Up(dsn); err != nil {
		t.Fatalf("re-applying 004 after a rollback must work: %v", err)
	}
	assertCurrentSchema(t, db)
}

// The planner schema (005-012) must apply on top of a populated auth schema,
// roll back completely (including the timezone trigger function) without
// touching accounts, and apply again.
func TestPlannerMigrationsRollBackAndReapply(t *testing.T) {
	dsn, db := testDB(t)
	if err := Up(dsn); err != nil {
		t.Fatal(err)
	}

	plannerObjects := `SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public' AND c.relname LIKE 'planner\_%' AND c.relkind IN ('r', 'v')`
	if n := count(t, db, plannerObjects); n != 15 {
		t.Fatalf("%d planner tables and views, want 14 tables + the planner_budget_lines view", n)
	}

	mustExec(t, db, `
		INSERT INTO users (id, email, password_hash, username) VALUES ('11111111-1111-1111-1111-111111111111', 'a@example.com', 'h', 'ani');
		INSERT INTO planner_projects (id, user_id, event_type, title, event_date, prep_start_date)
			VALUES ('44444444-4444-4444-4444-444444444444', '11111111-1111-1111-1111-111111111111', 'wedding', 'Nikah', '2027-03-28', '2026-10-21');`)

	if err := goose.DownTo(db, "sql", 4); err != nil {
		t.Fatalf("rolling back 005-012 must work: %v", err)
	}
	if n := count(t, db, plannerObjects); n != 0 {
		t.Fatalf("%d planner objects survived the rollback", n)
	}
	if n := count(t, db, `SELECT count(*) FROM pg_proc WHERE proname = 'planner_check_timezone'`); n != 0 {
		t.Fatal("the down migration must drop the timezone trigger function")
	}
	if n := count(t, db, `SELECT count(*) FROM users`); n != 1 {
		t.Fatalf("rolling back the planner schema touched accounts: %d users", n)
	}

	if err := Up(dsn); err != nil {
		t.Fatalf("re-applying the planner migrations: %v", err)
	}
	assertCurrentSchema(t, db)
	if n := count(t, db, plannerObjects); n != 15 {
		t.Fatalf("%d planner objects after re-applying, want 15", n)
	}
}

// Menu 06 appends its template with INSERT … ON CONFLICT (project_id, template_key) WHERE template_key IS NOT NULL DO
// NOTHING, like menus 02-04. That statement needs a unique partial index on each of the two tables; without it Postgres
// refuses the statement and a repeated key is accepted twice (06 BE 1.1).
func TestRundownTemplateKeyIndexesMakeAppendIdempotent(t *testing.T) {
	dsn, db := testDB(t)
	if err := Up(dsn); err != nil {
		t.Fatal(err)
	}

	for _, idx := range []string{"planner_rundowns_template_key_idx", "planner_playlists_template_key_idx"} {
		var def string
		if err := db.QueryRow(`SELECT indexdef FROM pg_indexes WHERE schemaname = 'public' AND indexname = $1`, idx).Scan(&def); err != nil {
			t.Fatalf("%s: %v", idx, err)
		}
		if !strings.Contains(def, "UNIQUE") || !strings.Contains(def, "(project_id, template_key)") || !strings.Contains(def, "WHERE (template_key IS NOT NULL)") {
			t.Errorf("%s = %s", idx, def)
		}
	}

	mustExec(t, db, `
		INSERT INTO users (id, email, password_hash, username) VALUES ('11111111-1111-1111-1111-111111111111', 'a@example.com', 'h', 'ani');
		INSERT INTO planner_projects (id, user_id, event_type, title, event_date, prep_start_date)
			VALUES ('44444444-4444-4444-4444-444444444444', '11111111-1111-1111-1111-111111111111', 'wedding', 'Nikah', '2027-03-28', '2026-10-21');
		INSERT INTO planner_event_sessions (id, project_id, scope_key, name, icon, is_system, sort_order)
			VALUES ('55555555-5555-5555-5555-555555555555', '44444444-4444-4444-4444-444444444444', 'akad', 'Akad Nikah', 'x', false, 1);`)

	agenda := `INSERT INTO planner_rundowns (id, project_id, session_id, start_time, activity_name, template_key)
		VALUES (gen_random_uuid(), '44444444-4444-4444-4444-444444444444', '55555555-5555-5555-5555-555555555555', '07:00', 'Persiapan', %s)
		ON CONFLICT (project_id, template_key) WHERE template_key IS NOT NULL DO NOTHING`
	song := `INSERT INTO planner_playlists (id, project_id, song_title, artist, template_key)
		VALUES (gen_random_uuid(), '44444444-4444-4444-4444-444444444444', 'Akad', 'Payung Teduh', %s)
		ON CONFLICT (project_id, template_key) WHERE template_key IS NOT NULL DO NOTHING`
	for table, stmt := range map[string]string{"planner_rundowns": agenda, "planner_playlists": song} {
		mustExec(t, db, fmt.Sprintf(stmt, `'wedding.x_01'`))
		mustExec(t, db, fmt.Sprintf(stmt, `'wedding.x_01'`))
		if n := count(t, db, `SELECT count(*) FROM `+table); n != 1 {
			t.Errorf("%s: a repeated template_key left %d rows, want 1", table, n)
		}
		// rows without a key (made by the user) are never in conflict
		mustExec(t, db, fmt.Sprintf(stmt, `NULL`))
		mustExec(t, db, fmt.Sprintf(stmt, `NULL`))
		if n := count(t, db, `SELECT count(*) FROM `+table); n != 3 {
			t.Errorf("%s: two rows without a key left %d rows, want 3", table, n)
		}
	}
}
