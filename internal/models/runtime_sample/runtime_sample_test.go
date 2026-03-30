package runtimesample

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestPruneOlderThanDeletesExpiredSamples(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "runtime-sample-test.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open returned error: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`
		CREATE TABLE runtime_samples (
			id TEXT PRIMARY KEY NOT NULL,
			observed_at DATETIME NOT NULL
		)
	`); err != nil {
		t.Fatalf("creating runtime_samples failed: %v", err)
	}

	cutoff := time.Now().UTC().Add(-30 * 24 * time.Hour)
	if _, err := db.Exec(`
		INSERT INTO runtime_samples (id, observed_at) VALUES (?, ?), (?, ?)
	`,
		"old-sample", cutoff.Add(-time.Hour),
		"fresh-sample", cutoff.Add(time.Hour),
	); err != nil {
		t.Fatalf("inserting samples failed: %v", err)
	}

	repo := NewRepository(db)
	if err := repo.PruneOlderThan(context.Background(), cutoff); err != nil {
		t.Fatalf("PruneOlderThan returned error: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM runtime_samples`).Scan(&count); err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 remaining sample, got %d", count)
	}
}
