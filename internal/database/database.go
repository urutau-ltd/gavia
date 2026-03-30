package database

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

var (
	client  *sql.DB
	once    sync.Once
	initErr error
)

func Client(dbPath string) (*sql.DB, error) {
	once.Do(func() {
		db, err := sql.Open("sqlite", dbPath)

		if err != nil {
			initErr = err
			return
		}
		client = db
	})

	return client, initErr
}

func ConfigurePool(db *sql.DB) {
	if db == nil {
		return
	}

	// Keep SQLite on a single long-lived physical connection so PRAGMAs stay
	// consistent and we avoid multiplying page-cache memory across pooled conns.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	db.SetConnMaxIdleTime(0)
}

func SetPragmas(db *sql.DB) error {
	const q = `
PRAGMA foreign_keys = ON;
PRAGMA busy_timeout = 5000;
PRAGMA synchronous = NORMAL;
PRAGMA journal_mode = 'WAL';
PRAGMA wal_autocheckpoint = 1000;
PRAGMA cache_size = -16384;`

	_, err := db.Exec(q)
	if err != nil {
		return errors.New(
			fmt.Sprintf("unable to set pragmas, %v", err),
		)
	}

	return nil
}

func RunMigrations(db *sql.DB, logger *slog.Logger) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS migrations (id INTEGER PRIMARY KEY, name TEXT UNIQUE)`)
	if err != nil {
		return err
	}

	entries, _ := migrationsFS.ReadDir("migrations")
	for _, entry := range entries {
		var exists bool
		db.QueryRow("SELECT EXISTS(SELECT 1 FROM migrations WHERE name = ?)", entry.Name()).Scan(&exists)
		if exists {
			continue
		}

		content, _ := migrationsFS.ReadFile("migrations/" + entry.Name())

		tx, err := db.Begin()
		if err != nil {
			return err
		}

		if _, err := tx.Exec(string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %s failed: %w", entry.Name(), err)
		}

		tx.Exec("INSERT INTO migrations (name) VALUES (?)", entry.Name())
		tx.Commit()
		logger.Info("Migration applied", "entry_name", entry.Name())
	}
	return nil
}
