package sqlite

import (
	"database/sql"
	_ "embed"
	"fmt"

	_ "modernc.org/sqlite" // Register sqlite driver
)

//go:embed migrations/001_initial.sql
var migration string

type DB struct {
	*sql.DB
}

func Open(dsn string) (*DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// In-memory databases are per-connection; multiple connections each get a
	// separate empty database. Limit to one connection so migrations and
	// queries all see the same data.
	if dsn == ":memory:" {
		db.SetMaxOpenConns(1)
	}

	// Enable WAL mode and foreign keys for better concurrency.
	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
	} {
		if _, err := db.Exec(pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("exec %s: %w", pragma, err)
		}
	}

	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return &DB{db}, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(migration)
	return err
}
