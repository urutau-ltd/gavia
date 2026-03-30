package exchangerate

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestPruneOlderThanDeletesExpiredSamples(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "exchange-rate-test.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open returned error: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`
		CREATE TABLE exchange_rate_samples (
			id TEXT PRIMARY KEY NOT NULL,
			observed_at DATETIME NOT NULL
		)
	`); err != nil {
		t.Fatalf("creating exchange_rate_samples failed: %v", err)
	}

	cutoff := time.Now().UTC().Add(-365 * 24 * time.Hour)
	if _, err := db.Exec(`
		INSERT INTO exchange_rate_samples (id, observed_at) VALUES (?, ?), (?, ?)
	`,
		"old-rate", cutoff.Add(-time.Hour),
		"fresh-rate", cutoff.Add(time.Hour),
	); err != nil {
		t.Fatalf("inserting rates failed: %v", err)
	}

	repo := NewRepository(db)
	if err := repo.PruneOlderThan(context.Background(), cutoff); err != nil {
		t.Fatalf("PruneOlderThan returned error: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM exchange_rate_samples`).Scan(&count); err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 remaining rate sample, got %d", count)
	}
}
