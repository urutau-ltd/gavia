package dashboardapi

import (
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"codeberg.org/urutau-ltd/gavia/internal/database"
	_ "modernc.org/sqlite"
)

func TestSummaryReturnsDashboardPayload(t *testing.T) {
	db := openDashboardAPITestDB(t)
	if err := database.RunMigrations(db, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		t.Fatalf("RunMigrations returned error: %v", err)
	}

	if err := database.SeedReferenceData(db); err != nil {
		t.Fatalf("SeedReferenceData returned error: %v", err)
	}

	if _, err := db.Exec(`
		UPDATE app_settings
		SET dashboard_due_soon_amount = 2, dashboard_currency = 'MXN'
		WHERE id = 'app';

		INSERT INTO providers (id, name) VALUES ('provider-api-1', 'Extra provider');
		INSERT INTO locations (id, name) VALUES ('location-api-1', 'Extra location');
		INSERT INTO domains (id, domain, due_date, price) VALUES
			('domain-api-1', 'example.com', '2026-03-26', 10.00),
			('domain-api-2', 'example.net', '2026-03-27', 20.00);
		INSERT INTO expense_entries (
			id,
			title,
			category,
			amount,
			currency,
			occurred_on,
			created_at,
			updated_at
		) VALUES
			('expense-api-1', 'Hetzner invoice', 'hosting', 42.50, 'MXN', '2026-03-10', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
			('expense-api-2', 'Registrar renewal', 'domain', 13.00, 'USD', '2026-03-11', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

		INSERT INTO exchange_rate_samples (
			id,
			base_currency,
			quote_currency,
			rate,
			source,
			observed_at
		) VALUES
			('rate-api-1', 'MXN', 'USD', 0.0500, 'fixture', '2026-03-10T00:00:00Z'),
			('rate-api-2', 'XMR', 'USD', 210.0000, 'fixture', '2026-03-10T00:00:00Z');

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
			('runtime-api-1', '2026-03-10T12:00:00Z', 8, 2097152, 3145728, 4194304, 8388608, 16777216, 4194304, 1, 0, 1, 0, 0, 4);

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
			('monitor-api-1', 'Example API', 'https://example.com/health', 'http', 200, 300, 5000, 1);

		INSERT INTO uptime_monitor_results (
			id,
			monitor_id,
			checked_at,
			ok,
			status_code,
			latency_ms
		) VALUES
			('result-api-1', 'monitor-api-1', '2026-03-10T12:00:00Z', 1, 200, 123);
	`); err != nil {
		t.Fatalf("could not prepare dashboard API fixtures: %v", err)
	}

	handler := NewHandler(slog.New(slog.NewTextHandler(io.Discard, nil)), db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", nil)
	rec := httptest.NewRecorder()
	handler.Summary(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var payload struct {
		Stats struct {
			ProviderCount int `json:"provider_count"`
			LocationCount int `json:"location_count"`
		} `json:"stats"`
		DueSoon struct {
			Total    float64 `json:"total"`
			BySource []struct {
				Label string `json:"label"`
				Count int    `json:"count"`
			} `json:"by_source"`
		} `json:"due_soon"`
		Expenses struct {
			ByCategory []struct {
				Label string `json:"label"`
			} `json:"by_category"`
		} `json:"expenses"`
		Currency struct {
			Latest struct {
				MXNToUSD *float64 `json:"mxn_to_usd"`
				MXNToXMR *float64 `json:"mxn_to_xmr"`
			} `json:"latest"`
			Totals []struct {
				Currency  string  `json:"currency"`
				Value     float64 `json:"value"`
				Available bool    `json:"available"`
			} `json:"totals"`
		} `json:"currency"`
		Diagnostics struct {
			Latest *struct {
				Goroutines int `json:"goroutines"`
			} `json:"latest"`
		} `json:"diagnostics"`
		Uptime struct {
			Summary *struct {
				Total int `json:"total"`
				Up    int `json:"up"`
			} `json:"summary"`
			Monitors []struct {
				Name string `json:"name"`
			} `json:"monitors"`
		} `json:"uptime"`
	}

	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("could not decode summary payload: %v", err)
	}

	if payload.Stats.ProviderCount < 2 {
		t.Fatalf("expected provider count to include fixtures, got %d", payload.Stats.ProviderCount)
	}

	if payload.Stats.LocationCount < 1 {
		t.Fatalf("expected location count to include fixtures, got %d", payload.Stats.LocationCount)
	}

	if payload.DueSoon.Total != 30 {
		t.Fatalf("expected due-soon total to be 30, got %f", payload.DueSoon.Total)
	}

	if len(payload.DueSoon.BySource) == 0 || payload.DueSoon.BySource[0].Label != "Domain" {
		t.Fatalf("expected grouped due-soon data, got %+v", payload.DueSoon.BySource)
	}

	if len(payload.Expenses.ByCategory) < 2 {
		t.Fatalf("expected grouped expense categories, got %+v", payload.Expenses.ByCategory)
	}

	if payload.Currency.Latest.MXNToUSD == nil || *payload.Currency.Latest.MXNToUSD != 0.05 {
		t.Fatalf("expected latest MXN to USD rate in payload, got %+v", payload.Currency.Latest)
	}

	if len(payload.Currency.Totals) != 3 {
		t.Fatalf("expected converted currency totals, got %+v", payload.Currency.Totals)
	}

	if payload.Diagnostics.Latest == nil || payload.Diagnostics.Latest.Goroutines != 8 {
		t.Fatalf("expected runtime diagnostics in payload, got %+v", payload.Diagnostics.Latest)
	}

	if payload.Uptime.Summary == nil || payload.Uptime.Summary.Total != 1 || payload.Uptime.Summary.Up != 1 {
		t.Fatalf("expected uptime summary in payload, got %+v", payload.Uptime.Summary)
	}

	if len(payload.Uptime.Monitors) != 1 || payload.Uptime.Monitors[0].Name != "Example API" {
		t.Fatalf("expected uptime monitors in payload, got %+v", payload.Uptime.Monitors)
	}
}

func openDashboardAPITestDB(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "dashboard-api-test.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open returned error: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}
