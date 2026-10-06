package migrate

import (
	"database/sql"
	"embed"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed sql/*.sql
var files embed.FS

// Up applies every migration that has not run yet. A database still holding
// the pre-goose schema is first archived (see archiveLegacySchema) so that a
// deploy onto it neither fails nor loses data.
func Up(databaseURL string) error {
	db, closeDB, err := open(databaseURL)
	if err != nil {
		return err
	}
	defer closeDB()

	if err := archiveLegacySchema(db); err != nil {
		return fmt.Errorf("archive legacy schema: %w", err)
	}

	return goose.Up(db, "sql")
}

// archiveLegacySchema moves every table in public into a new legacy_<time>
// schema when public.users still has the display_name column that only the
// pre-goose backend created. Without it, CREATE TABLE users in migration 002
// fails with "already exists" and the API cannot start. Nothing is deleted:
// the data stays queryable in the archive schema until someone drops it.
//
// The move also covers a half-migrated database (m_role and goose_db_version
// created by an earlier failed start) because it takes every table in public.
func archiveLegacySchema(db *sql.DB) error {
	var legacy bool
	// pg_catalog rather than information_schema, which hides tables the
	// connected role does not own.
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM pg_attribute a
			JOIN pg_class c ON c.oid = a.attrelid
			JOIN pg_namespace n ON n.oid = c.relnamespace
			WHERE n.nspname = 'public' AND c.relname = 'users'
			  AND a.attname = 'display_name' AND NOT a.attisdropped)`).Scan(&legacy)
	if err != nil {
		return err
	}
	if !legacy {
		return nil
	}

	rows, err := db.Query(`SELECT tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename`)
	if err != nil {
		return err
	}
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return err
		}
		tables = append(tables, name)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	archive := "legacy_" + time.Now().UTC().Format("20060102_150405")
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("CREATE SCHEMA " + pgx.Identifier{archive}.Sanitize()); err != nil {
		return err
	}
	for _, table := range tables {
		stmt := "ALTER TABLE " + pgx.Identifier{"public", table}.Sanitize() + " SET SCHEMA " + pgx.Identifier{archive}.Sanitize()
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("move %s: %w", table, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	log.Printf("migrate: found the pre-goose schema; moved %d tables to schema %q (data kept, run DROP SCHEMA %s CASCADE when no longer needed)", len(tables), archive, archive)

	return nil
}

// Fresh drops the whole public schema and applies every migration from
// scratch, destroying all data in it. Unlike Up it does not archive anything,
// so it is for resetting a database that is not worth keeping.
func Fresh(databaseURL string) error {
	db, closeDB, err := open(databaseURL)
	if err != nil {
		return err
	}
	defer closeDB()

	if _, err := db.Exec(`DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;`); err != nil {
		return fmt.Errorf("reset public schema: %w", err)
	}

	return goose.Up(db, "sql")
}

func open(databaseURL string) (*sql.DB, func(), error) {
	goose.SetBaseFS(files)
	if err := goose.SetDialect("postgres"); err != nil {
		return nil, nil, err
	}

	connCfg, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		return nil, nil, fmt.Errorf("parse database url for migrate: %w", err)
	}
	connCfg.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	dsn := stdlib.RegisterConnConfig(connCfg)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		stdlib.UnregisterConnConfig(dsn)
		return nil, nil, fmt.Errorf("open db for migrate: %w", err)
	}

	return db, func() {
		db.Close()
		stdlib.UnregisterConnConfig(dsn)
	}, nil
}
