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

	"codeberg.org/urutau-ltd/gavia/internal/database"
	uptimemonitor "codeberg.org/urutau-ltd/gavia/internal/models/uptime_monitor"
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
	handler := NewHandler(slog.New(slog.NewTextHandler(io.Discard, nil)), uiRoot, repo)

	req := httptest.NewRequest(http.MethodGet, "/uptime/"+monitor.ID, nil)
	req.SetPathValue("id", monitor.ID)
	rec := httptest.NewRecorder()
	handler.Show(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "uptime-results-chart") || !strings.Contains(body, "uptime-distribution-chart") {
		t.Fatalf("expected uptime page to render charts, got %q", body)
	}

	if !strings.Contains(body, "Availability") || !strings.Contains(body, "Example API") {
		t.Fatalf("expected uptime page to render summary cards, got %q", body)
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
