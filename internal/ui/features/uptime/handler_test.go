package uptime

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
	"time"

	"codeberg.org/urutau-ltd/gavia/internal/database"
	uptimemonitor "codeberg.org/urutau-ltd/gavia/internal/models/uptime_monitor"
	uptimeservice "codeberg.org/urutau-ltd/gavia/internal/uptime"
	_ "modernc.org/sqlite"
)

func TestShowRendersMonitorCharts(t *testing.T) {
	db := openUptimeTestDB(t)
	if err := database.RunMigrations(db, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		t.Fatalf("RunMigrations returned error: %v", err)
	}

	repo := uptimemonitor.NewRepository(db)
	monitor := &uptimemonitor.Monitor{
		Name:                 "Example API",
		TargetURL:            "https://example.com/health",
		Kind:                 "http",
		ExpectedStatus:       200,
		ExpectedStatusMin:    200,
		ExpectedStatusMax:    299,
		HTTPMethod:           "GET",
		TLSMode:              "skip",
		CheckIntervalSeconds: 300,
		TimeoutMS:            5000,
		Enabled:              true,
	}
	if err := repo.Create(t.Context(), monitor); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	statusCode := 200
	latencyMS := 120
	if err := repo.CreateResult(t.Context(), &uptimemonitor.Result{
		MonitorID:  monitor.ID,
		OK:         true,
		StatusCode: &statusCode,
		LatencyMS:  &latencyMS,
	}); err != nil {
		t.Fatalf("CreateResult returned error: %v", err)
	}

	uiRoot := os.DirFS(filepath.Join("..", ".."))
	service := uptimeservice.NewService(slog.New(slog.NewTextHandler(io.Discard, nil)), repo, nil, time.Second)
	handler := NewHandler(slog.New(slog.NewTextHandler(io.Discard, nil)), uiRoot, repo, service)

	req := httptest.NewRequest(http.MethodGet, "/uptime/"+monitor.ID, nil)
	req.SetPathValue("id", monitor.ID)
	rec := httptest.NewRecorder()
	handler.Show(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "uptime-results-chart") ||
		!strings.Contains(body, "uptime-latency-chart") ||
		!strings.Contains(body, "uptime-distribution-chart") ||
		!strings.Contains(body, "uptime-status-code-chart") {
		t.Fatalf("expected uptime page to render charts, got %q", body)
	}

	if !strings.Contains(body, "Availability") || !strings.Contains(body, "Example API") || !strings.Contains(body, "Accepted status range") {
		t.Fatalf("expected uptime page to render summary cards, got %q", body)
	}

	if !strings.Contains(body, `aria-label="Uptime sections"`) ||
		!strings.Contains(body, "uptime-panel-monitors") ||
		!strings.Contains(body, "uptime-panel-results") ||
		!strings.Contains(body, "uptime-panel-charts") ||
		!strings.Contains(body, "Selected monitor") {
		t.Fatalf("expected uptime page to render tabbed sections and selected monitor details, got %q", body)
	}
}

func openUptimeTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "uptime-test.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open returned error: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}
