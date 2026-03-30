package uptimemonitor

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestPruneResultsOlderThanDeletesExpiredResults(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "uptime-monitor-test.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open returned error: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`
		CREATE TABLE uptime_monitor_results (
			id TEXT PRIMARY KEY NOT NULL,
			monitor_id TEXT NOT NULL,
			checked_at DATETIME NOT NULL
		)
	`); err != nil {
		t.Fatalf("creating uptime_monitor_results failed: %v", err)
	}

	cutoff := time.Now().UTC().Add(-30 * 24 * time.Hour)
	if _, err := db.Exec(`
		INSERT INTO uptime_monitor_results (id, monitor_id, checked_at) VALUES (?, ?, ?), (?, ?, ?)
	`,
		"old-result", "monitor-1", cutoff.Add(-time.Hour),
		"fresh-result", "monitor-1", cutoff.Add(time.Hour),
	); err != nil {
		t.Fatalf("inserting results failed: %v", err)
	}

	repo := NewRepository(db)
	if err := repo.PruneResultsOlderThan(context.Background(), cutoff); err != nil {
		t.Fatalf("PruneResultsOlderThan returned error: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM uptime_monitor_results`).Scan(&count); err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 remaining uptime result, got %d", count)
	}
}
