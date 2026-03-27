package dashboard

import (
	"database/sql"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"codeberg.org/urutau-ltd/gavia/internal/database"
	_ "modernc.org/sqlite"
)

func TestDashboardShowsDueSoonLimitAndRecentExpenses(t *testing.T) {
	db := openDashboardTestDB(t)
	if err := database.RunMigrations(db, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		t.Fatalf("RunMigrations returned error: %v", err)
	}

	if err := database.SeedReferenceData(db); err != nil {
		t.Fatalf("SeedReferenceData returned error: %v", err)
	}

	if _, err := db.Exec(`
		UPDATE app_settings
		SET dashboard_due_soon_amount = 2, dashboard_currency = 'MXN'
		WHERE id = 'app'
	`); err != nil {
		t.Fatalf("could not update app settings fixture: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO domains (id, domain, due_date, price) VALUES
			('domain-1', 'example.com', '2026-03-26', 10.00),
			('domain-2', 'example.net', '2026-03-27', 20.00);

		INSERT INTO subscriptions (id, name, type, due_date, price) VALUES
			('subscription-1', 'Email SaaS', 'saas', '2026-03-28', 30.00);

		INSERT INTO expense_entries (
			id,
			title,
			category,
			amount,
			currency,
			occurred_on,
			notes,
			created_at,
			updated_at
		) VALUES
			('expense-1', 'Hetzner invoice', 'hosting', 42.50, 'MXN', '2026-03-10', 'first', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
			('expense-2', 'Registrar renewal', 'domain', 13.00, 'MXN', '2026-03-11', 'second', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

		INSERT INTO exchange_rate_samples (
			id,
			base_currency,
			quote_currency,
			rate,
			source,
			observed_at
		) VALUES
			('rate-1', 'MXN', 'USD', 0.0500, 'fixture', '2026-03-10T00:00:00Z'),
			('rate-2', 'XMR', 'USD', 210.0000, 'fixture', '2026-03-10T00:00:00Z');

		INSERT INTO runtime_samples (
			id,
			observed_at,
			goroutines,
			heap_alloc_bytes,
			heap_inuse_bytes,
			heap_sys_bytes,
			total_alloc_bytes,
			sys_bytes,
			next_gc_bytes,
			db_open_connections,
			db_in_use,
			db_idle,
			db_wait_count,
			db_wait_duration_ms,
			cpu_count
		) VALUES
			('runtime-1', '2026-03-10T12:00:00Z', 8, 2097152, 3145728, 4194304, 8388608, 16777216, 4194304, 1, 0, 1, 0, 0, 4);

		INSERT INTO uptime_monitors (
			id,
			name,
			target_url,
			kind,
			expected_status,
			check_interval_seconds,
			timeout_ms,
			enabled
		) VALUES
			('monitor-1', 'Example API', 'https://example.com/health', 'http', 200, 300, 5000, 1);

		INSERT INTO uptime_monitor_results (
			id,
			monitor_id,
			checked_at,
			ok,
			status_code,
			latency_ms
		) VALUES
			('result-1', 'monitor-1', '2026-03-10T12:00:00Z', 1, 200, 123);
	`); err != nil {
		t.Fatalf("could not prepare dashboard fixtures: %v", err)
	}

	uiRoot := os.DirFS(filepath.Join("..", ".."))
	handler := NewHandler(slog.New(slog.NewTextHandler(io.Discard, nil)), uiRoot, db)

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rec := httptest.NewRecorder()
	handler.Index(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	if strings.Count(body, `class="dashboard-list-item-text"`) != 2 {
		t.Fatalf("expected dashboard due-soon list to honor the limit with exactly two visible rows, got %q", body)
	}

	if !strings.Contains(body, "Hetzner invoice") {
		t.Fatalf("expected dashboard to show recent expense entries, got %q", body)
	}

	if !strings.Contains(body, "expense-history-chart") || !strings.Contains(body, "runtime-history-chart") {
		t.Fatalf("expected dashboard to render chart canvases, got %q", body)
	}

	if !strings.Contains(body, "Inventory overview") || !strings.Contains(body, "inventory-distribution-chart") {
		t.Fatalf("expected dashboard to render inventory summary widgets, got %q", body)
	}

	if strings.Contains(body, "Started modules") || strings.Contains(body, "Pending modules") {
		t.Fatalf("expected dashboard to stop rendering module progress placeholders, got %q", body)
	}

	if !strings.Contains(body, "Runtime diagnostics") || !strings.Contains(body, "Goroutines") {
		t.Fatalf("expected dashboard to render runtime diagnostics, got %q", body)
	}

	if !strings.Contains(body, "Uptime snapshot") || !strings.Contains(body, "Example API") {
		t.Fatalf("expected dashboard to render uptime summary, got %q", body)
	}
}

func openDashboardTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "dashboard-test.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open returned error: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}
