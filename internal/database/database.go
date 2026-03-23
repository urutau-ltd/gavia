package database

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
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
