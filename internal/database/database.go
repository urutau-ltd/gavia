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

func SetPragmas(db *sql.DB) error {
	const q = `
PRAGMA foreign_keys = ON;
PRAGMA synchronous = NORMAL;
PRAGMA journal_mode = 'WAL';
PRAGMA cache_size = -64000;`

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
			return fmt.Errorf("error en %s: %w", entry.Name(), err)
		}

		tx.Exec("INSERT INTO migrations (name) VALUES (?)", entry.Name())
		tx.Commit()
		logger.Info("Migración aplicada: ", "entry_name", entry.Name())
	}
	return nil
}
