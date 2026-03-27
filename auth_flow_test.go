package main

import (
	"context"
	"database/sql"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"codeberg.org/urutau-ltd/aile/v2"
	"codeberg.org/urutau-ltd/aile/v2/x/combine"
	backupapi "codeberg.org/urutau-ltd/gavia/internal/api/backup"
	dashboardapi "codeberg.org/urutau-ltd/gavia/internal/api/dashboard"
	"codeberg.org/urutau-ltd/gavia/internal/auth"
	"codeberg.org/urutau-ltd/gavia/internal/backup"
	"codeberg.org/urutau-ltd/gavia/internal/csrf"
	"codeberg.org/urutau-ltd/gavia/internal/database"
	accountsetting "codeberg.org/urutau-ltd/gavia/internal/models/account_setting"
	appsetting "codeberg.org/urutau-ltd/gavia/internal/models/app_setting"
	expenseentry "codeberg.org/urutau-ltd/gavia/internal/models/expense_entry"
	operatingsystem "codeberg.org/urutau-ltd/gavia/internal/models/operating_system"
	"codeberg.org/urutau-ltd/gavia/internal/models/session"
	uptimemonitor "codeberg.org/urutau-ltd/gavia/internal/models/uptime_monitor"
	"codeberg.org/urutau-ltd/gavia/internal/ui"
	accountsettings "codeberg.org/urutau-ltd/gavia/internal/ui/features/account_settings"
	appsettings "codeberg.org/urutau-ltd/gavia/internal/ui/features/app_settings"
	"codeberg.org/urutau-ltd/gavia/internal/ui/features/dashboard"
	"codeberg.org/urutau-ltd/gavia/internal/ui/features/dns"
	"codeberg.org/urutau-ltd/gavia/internal/ui/features/domains"
	"codeberg.org/urutau-ltd/gavia/internal/ui/features/hostings"
	"codeberg.org/urutau-ltd/gavia/internal/ui/features/ips"
	"codeberg.org/urutau-ltd/gavia/internal/ui/features/labels"
	licensespage "codeberg.org/urutau-ltd/gavia/internal/ui/features/licenses"
	"codeberg.org/urutau-ltd/gavia/internal/ui/features/locations"
	"codeberg.org/urutau-ltd/gavia/internal/ui/features/login"
	"codeberg.org/urutau-ltd/gavia/internal/ui/features/logout"
	operatingsystems "codeberg.org/urutau-ltd/gavia/internal/ui/features/operating_systems"
	"codeberg.org/urutau-ltd/gavia/internal/ui/features/providers"
	"codeberg.org/urutau-ltd/gavia/internal/ui/features/servers"
	"codeberg.org/urutau-ltd/gavia/internal/ui/features/subscriptions"
	uptimepage "codeberg.org/urutau-ltd/gavia/internal/ui/features/uptime"
	_ "modernc.org/sqlite"
)

func TestSetupLoginLogoutFlow(t *testing.T) {
	handler := buildAppHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected setup redirect status %d, got %d", http.StatusSeeOther, rec.Code)
	}

	if got := rec.Header().Get("Location"); got != auth.SetupPath {
		t.Fatalf("expected setup redirect to %q, got %q", auth.SetupPath, got)
	}

	req = httptest.NewRequest(http.MethodGet, auth.SetupPath, nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected setup page status %d, got %d", http.StatusOK, rec.Code)
	}

	setupPage := rec.Body.String()
	if !strings.Contains(setupPage, `hx-boost="false"`) {
		t.Fatalf("expected setup form to disable htmx boost, got %q", setupPage)
	}
	setupCSRFCookie := csrfCookieFromResponse(t, rec.Result().Cookies())
	setupCSRFToken := csrfTokenFromHTML(t, setupPage)

	setupForm := url.Values{
		"_csrf":            {setupCSRFToken},
		"username":         {"admin"},
		"password":         {"correct horse battery staple"},
		"confirm_password": {"correct horse battery staple"},
		"avatar_path":      {"/static/img/avatar-3.svg"},
	}
	req = httptest.NewRequest(http.MethodPost, "/account-settings/edit", strings.NewReader(setupForm.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(setupCSRFCookie)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected setup status %d, got %d", http.StatusOK, rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "Administrator account created successfully.") {
		t.Fatalf("expected setup response body to confirm account creation, got %q", rec.Body.String())
	}

	if strings.Contains(rec.Body.String(), "Initial Account Setup") {
		t.Fatalf("expected setup response to stop rendering setup-only navigation, got %q", rec.Body.String())
	}

	if !strings.Contains(rec.Body.String(), "Dashboard") {
		t.Fatalf("expected setup response to render authenticated navigation, got %q", rec.Body.String())
	}

	setupCookie := sessionCookieFromResponse(t, rec.Result().Cookies())

	req = httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.AddCookie(setupCookie)
	req.AddCookie(setupCSRFCookie)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected authenticated dashboard status %d, got %d", http.StatusOK, rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "Free Software Licenses") {
		t.Fatalf("expected authenticated dashboard to use the classic navigation, got %q", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(setupCookie)
	req.AddCookie(setupCSRFCookie)
	req.Header.Set(csrf.HeaderName, setupCSRFToken)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected logout redirect status %d, got %d", http.StatusSeeOther, rec.Code)
	}

	if got := rec.Header().Get("Location"); got != "/login?notice=logged-out" {
		t.Fatalf("expected logout redirect to %q, got %q", "/login?notice=logged-out", got)
	}

	loginForm := url.Values{
		"_csrf":    {setupCSRFToken},
		"username": {"admin"},
		"password": {"correct horse battery staple"},
	}
	req = httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(loginForm.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(setupCSRFCookie)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected login redirect status %d, got %d", http.StatusSeeOther, rec.Code)
	}

	if got := rec.Header().Get("Location"); got != auth.DashboardPath {
		t.Fatalf("expected login redirect to %q, got %q", auth.DashboardPath, got)
	}

	_ = sessionCookieFromResponse(t, rec.Result().Cookies())
}

func TestUnsafeSetupPostWithoutCSRFTokenRejected(t *testing.T) {
	handler := buildAppHandler(t)

	form := url.Values{
		"username":         {"admin"},
		"password":         {"correct horse battery staple"},
		"confirm_password": {"correct horse battery staple"},
		"avatar_path":      {"/static/img/avatar-3.svg"},
	}
	req := httptest.NewRequest(http.MethodPost, auth.SetupPath, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected setup POST without CSRF token to be rejected with %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func buildAppHandler(t *testing.T) http.Handler {
	t.Helper()

	db := openFlowTestDB(t)
	runFlowMigrations(t, db)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := aile.MustNew()

	accountRepo := accountsetting.NewAccountSettingsRepository(db)
	appSettingsRepo := appsetting.NewAppSettingsRepository(db)
	expenseRepo := expenseentry.NewExpenseEntryRepository(db)
	osRepo := operatingsystem.NewOperatingSystemRepository(db)
	sessionRepo := session.NewSessionRepository(db)
	uptimeRepo := uptimemonitor.NewRepository(db)
	authService := auth.NewService(accountRepo, sessionRepo)
	backupService := backup.NewService(db)
	csrfService := csrf.NewService()

	ui.SetShowVersionFooter(true)

	app.Use(combine.Middleware(csrfService.Middleware(), authService.Middleware()))

	uiRoot, err := fs.Sub(UIFS, "internal/ui")
	if err != nil {
		t.Fatalf("fs.Sub returned error: %v", err)
	}

	dashboardHandler := dashboard.NewHandler(logger, uiRoot, db)
	providerHandler := providers.NewHandler(logger, uiRoot, db)
	locationHandler := locations.NewHandler(logger, uiRoot, db)
	osHandler := operatingsystems.NewHandler(logger, uiRoot, db)
	ipHandler := ips.NewHandler(logger, uiRoot, db)
	dnsHandler := dns.NewHandler(logger, uiRoot, db)
	labelHandler := labels.NewHandler(logger, uiRoot, db)
	domainHandler := domains.NewHandler(logger, uiRoot, db)
	hostingHandler := hostings.NewHandler(logger, uiRoot, db)
	serverHandler := servers.NewHandler(logger, uiRoot, db)
	subscriptionHandler := subscriptions.NewHandler(logger, uiRoot, db)
	loginHandler := login.NewHandler(logger, uiRoot, authService)
	logoutHandler := logout.NewHandler(logger, authService)
	accountSettingsHandler := accountsettings.NewHandler(logger, uiRoot, accountRepo, authService)
	appSettingsHandler := appsettings.NewHandler(
		logger,
		uiRoot,
		appSettingsRepo,
		accountRepo,
		expenseRepo,
		osRepo,
		backupService,
		authService,
	)
	backupAPIHandler := backupapi.NewHandler(logger, backupService, accountRepo)
	dashboardAPIHandler := dashboardapi.NewHandler(logger, db)
	licensesHandler := licensespage.NewHandler(logger, uiRoot)
	uptimeHandler := uptimepage.NewHandler(logger, uiRoot, uptimeRepo)

	if err := mountRoutes(app, appHandlers{
		dashboard:       dashboardHandler,
		provider:        providerHandler,
		location:        locationHandler,
		os:              osHandler,
		ip:              ipHandler,
		dns:             dnsHandler,
		label:           labelHandler,
		domain:          domainHandler,
		hosting:         hostingHandler,
		server:          serverHandler,
		subscription:    subscriptionHandler,
		accountSettings: accountSettingsHandler,
		appSettings:     appSettingsHandler,
		login:           loginHandler,
		logout:          logoutHandler,
		backupAPI:       backupAPIHandler,
		dashboardAPI:    dashboardAPIHandler,
		licenses:        licensesHandler,
		uptime:          uptimeHandler,
	}); err != nil {
		t.Fatalf("mountRoutes returned error: %v", err)
	}

	state, err := app.Build(context.Background())
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	return state.Handler
}

func openFlowTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "flow-test.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open returned error: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func runFlowMigrations(t *testing.T, db *sql.DB) {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := database.RunMigrations(db, logger); err != nil {
		t.Fatalf("RunMigrations returned error: %v", err)
	}

	if err := database.SeedReferenceData(db); err != nil {
		t.Fatalf("SeedReferenceData returned error: %v", err)
	}
}

func sessionCookieFromResponse(t *testing.T, cookies []*http.Cookie) *http.Cookie {
	t.Helper()

	for _, cookie := range cookies {
		if cookie.Name == auth.SessionCookieName {
			return cookie
		}
	}

	t.Fatal("expected response to set the session cookie")
	return nil
}

func csrfCookieFromResponse(t *testing.T, cookies []*http.Cookie) *http.Cookie {
	t.Helper()

	for _, cookie := range cookies {
		if cookie.Name == csrf.CookieName {
			return cookie
		}
	}

	t.Fatal("expected response to set the CSRF cookie")
	return nil
}

func csrfTokenFromHTML(t *testing.T, body string) string {
	t.Helper()

	matches := regexp.MustCompile(`name="_csrf"\s+value="([^"]+)"`).FindStringSubmatch(body)
	if len(matches) != 2 {
		t.Fatalf("expected response HTML to include a CSRF hidden input, got %q", body)
	}

	return matches[1]
}
