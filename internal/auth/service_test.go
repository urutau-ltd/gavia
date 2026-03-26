package auth

import (
	"database/sql"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"codeberg.org/urutau-ltd/gavia/internal/database"
	accountsetting "codeberg.org/urutau-ltd/gavia/internal/models/account_setting"
	"codeberg.org/urutau-ltd/gavia/internal/models/session"
	"codeberg.org/urutau-ltd/gavia/internal/security"
	_ "modernc.org/sqlite"
)

func TestMiddlewareRedirectsSetupFlow(t *testing.T) {
	db := openAuthTestDB(t)
	runAuthMigrations(t, db)

	service := NewService(
		accountsetting.NewAccountSettingsRepository(db),
		session.NewSessionRepository(db),
	)

	handler := service.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected setup redirect status %d, got %d", http.StatusSeeOther, rec.Code)
	}

	if got := rec.Header().Get("Location"); got != SetupPath {
		t.Fatalf("expected redirect to %q, got %q", SetupPath, got)
	}
}

func TestMiddlewareRedirectsUnauthenticatedUsersToLogin(t *testing.T) {
	db := openAuthTestDB(t)
	runAuthMigrations(t, db)
	createAccount(t, db)

	service := NewService(
		accountsetting.NewAccountSettingsRepository(db),
		session.NewSessionRepository(db),
	)

	handler := service.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected login redirect status %d, got %d", http.StatusSeeOther, rec.Code)
	}

	if got := rec.Header().Get("Location"); got != LoginPath {
		t.Fatalf("expected redirect to %q, got %q", LoginPath, got)
	}
}

func TestMiddlewareAllowsAPIRequestsWithValidAPIToken(t *testing.T) {
	db := openAuthTestDB(t)
	runAuthMigrations(t, db)
	apiToken := createAccountWithToken(t, db)

	service := NewService(
		accountsetting.NewAccountSettingsRepository(db),
		session.NewSessionRepository(db),
	)

	handler := service.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/backup/export", nil)
	req.Header.Set("X-API-Token", apiToken)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected API request status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestMiddlewareRejectsAPIRequestsWithoutCredentials(t *testing.T) {
	db := openAuthTestDB(t)
	runAuthMigrations(t, db)
	createAccount(t, db)

	service := NewService(
		accountsetting.NewAccountSettingsRepository(db),
		session.NewSessionRepository(db),
	)

	handler := service.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/backup/export", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected API request status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func openAuthTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "auth-test.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open returned error: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func runAuthMigrations(t *testing.T, db *sql.DB) {
	t.Helper()

	if err := database.RunMigrations(db, nilLogger()); err != nil {
		t.Fatalf("RunMigrations returned error: %v", err)
	}

	if err := database.SeedReferenceData(db); err != nil {
		t.Fatalf("SeedReferenceData returned error: %v", err)
	}
}

func createAccount(t *testing.T, db *sql.DB) {
	t.Helper()

	passwordHash, err := security.HashPassword("super-secret-password")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	publicKey, _, err := security.GenerateRecoveryKeyPair()
	if err != nil {
		t.Fatalf("GenerateRecoveryKeyPair returned error: %v", err)
	}

	_, apiTokenHash, apiTokenHint, err := security.GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}

	repo := accountsetting.NewAccountSettingsRepository(db)
	if err := repo.Create(t.Context(), &accountsetting.AccountSettings{
		Username:          "admin",
		PasswordHash:      passwordHash,
		APITokenHash:      apiTokenHash,
		APITokenHint:      apiTokenHint,
		AvatarPath:        "/static/img/avatar-1.svg",
		RecoveryPublicKey: publicKey,
	}); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
}

func createAccountWithToken(t *testing.T, db *sql.DB) string {
	t.Helper()

	passwordHash, err := security.HashPassword("super-secret-password")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	publicKey, _, err := security.GenerateRecoveryKeyPair()
	if err != nil {
		t.Fatalf("GenerateRecoveryKeyPair returned error: %v", err)
	}

	apiToken, apiTokenHash, apiTokenHint, err := security.GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}

	repo := accountsetting.NewAccountSettingsRepository(db)
	if err := repo.Create(t.Context(), &accountsetting.AccountSettings{
		Username:          "admin",
		PasswordHash:      passwordHash,
		APITokenHash:      apiTokenHash,
		APITokenHint:      apiTokenHint,
		AvatarPath:        "/static/img/avatar-1.svg",
		RecoveryPublicKey: publicKey,
	}); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	return apiToken
}

func nilLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
