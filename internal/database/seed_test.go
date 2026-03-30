package database

import (
	"database/sql"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
)

func TestSeedReferenceDataIsIdempotentAndSettingsAreSingleton(t *testing.T) {
	db := openTestDB(t)
	logger := testLogger()

	if err := RunMigrations(db, logger); err != nil {
		t.Fatalf("RunMigrations returned error: %v", err)
	}

	if err := SeedReferenceData(db); err != nil {
		t.Fatalf("SeedReferenceData returned error: %v", err)
	}

	var providerCountBefore int
	if err := db.QueryRow(`SELECT COUNT(*) FROM providers`).Scan(&providerCountBefore); err != nil {
		t.Fatalf("could not count providers after first seed: %v", err)
	}

	if providerCountBefore == 0 {
		t.Fatal("expected provider seed data to be inserted")
	}

	var operatingSystemCountBefore int
	if err := db.QueryRow(`SELECT COUNT(*) FROM operating_systems`).Scan(&operatingSystemCountBefore); err != nil {
		t.Fatalf("could not count operating systems after first seed: %v", err)
	}

	if operatingSystemCountBefore == 0 {
		t.Fatal("expected operating system seed data to be inserted")
	}

	var labelCountBefore int
	if err := db.QueryRow(`SELECT COUNT(*) FROM labels`).Scan(&labelCountBefore); err != nil {
		t.Fatalf("could not count labels after first seed: %v", err)
	}

	if labelCountBefore == 0 {
		t.Fatal("expected label seed data to be inserted")
	}

	var locationCountBefore int
	if err := db.QueryRow(`SELECT COUNT(*) FROM locations`).Scan(&locationCountBefore); err != nil {
		t.Fatalf("could not count locations after first seed: %v", err)
	}

	if locationCountBefore == 0 {
		t.Fatal("expected location seed data to be inserted")
	}

	if err := SeedReferenceData(db); err != nil {
		t.Fatalf("SeedReferenceData second run returned error: %v", err)
	}

	var providerCountAfter int
	if err := db.QueryRow(`SELECT COUNT(*) FROM providers`).Scan(&providerCountAfter); err != nil {
		t.Fatalf("could not count providers after second seed: %v", err)
	}

	if providerCountAfter != providerCountBefore {
		t.Fatalf("expected provider seed count to stay at %d, got %d", providerCountBefore, providerCountAfter)
	}

	var operatingSystemCountAfter int
	if err := db.QueryRow(`SELECT COUNT(*) FROM operating_systems`).Scan(&operatingSystemCountAfter); err != nil {
		t.Fatalf("could not count operating systems after second seed: %v", err)
	}

	if operatingSystemCountAfter != operatingSystemCountBefore {
		t.Fatalf("expected operating system seed count to stay at %d, got %d", operatingSystemCountBefore, operatingSystemCountAfter)
	}

	var labelCountAfter int
	if err := db.QueryRow(`SELECT COUNT(*) FROM labels`).Scan(&labelCountAfter); err != nil {
		t.Fatalf("could not count labels after second seed: %v", err)
	}

	if labelCountAfter != labelCountBefore {
		t.Fatalf("expected label seed count to stay at %d, got %d", labelCountBefore, labelCountAfter)
	}

	var locationCountAfter int
	if err := db.QueryRow(`SELECT COUNT(*) FROM locations`).Scan(&locationCountAfter); err != nil {
		t.Fatalf("could not count locations after second seed: %v", err)
	}

	if locationCountAfter != locationCountBefore {
		t.Fatalf("expected location seed count to stay at %d, got %d", locationCountBefore, locationCountAfter)
	}

	var appSettingsCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM app_settings`).Scan(&appSettingsCount); err != nil {
		t.Fatalf("could not count app settings: %v", err)
	}

	if appSettingsCount != 1 {
		t.Fatalf("expected exactly one app settings row, got %d", appSettingsCount)
	}

	var appID, defaultServerOS, defaultCurrency, dashboardCurrency string
	if err := db.QueryRow(`
		SELECT
			id,
			default_server_os,
			default_currency,
			dashboard_currency
		FROM app_settings
		LIMIT 1
	`).Scan(
		&appID,
		&defaultServerOS,
		&defaultCurrency,
		&dashboardCurrency,
	); err != nil {
		t.Fatalf("could not query app settings row: %v", err)
	}

	if appID != "app" {
		t.Fatalf("expected app settings id %q, got %q", "app", appID)
	}

	if defaultServerOS != "Linux" {
		t.Fatalf("expected default server os %q, got %q", "Linux", defaultServerOS)
	}

	if defaultCurrency != "MXN" {
		t.Fatalf("expected default currency %q, got %q", "MXN", defaultCurrency)
	}

	if dashboardCurrency != "MXN" {
		t.Fatalf("expected dashboard currency %q, got %q", "MXN", dashboardCurrency)
	}

	var linuxCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM operating_systems WHERE name = 'Linux'`).Scan(&linuxCount); err != nil {
		t.Fatalf("could not verify Linux operating system seed: %v", err)
	}

	if linuxCount != 1 {
		t.Fatalf("expected exactly one generic Linux operating system seed, got %d", linuxCount)
	}

	var productionLabelCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM labels WHERE name = 'production'`).Scan(&productionLabelCount); err != nil {
		t.Fatalf("could not verify production label seed: %v", err)
	}

	if productionLabelCount != 1 {
		t.Fatalf("expected exactly one production label seed, got %d", productionLabelCount)
	}

	var mexicoCityCoordinates int
	if err := db.QueryRow(`
		SELECT COUNT(*)
		FROM locations
		WHERE name = 'Mexico City'
		  AND latitude IS NOT NULL
		  AND longitude IS NOT NULL
	`).Scan(&mexicoCityCoordinates); err != nil {
		t.Fatalf("could not verify Mexico City location seed: %v", err)
	}

	if mexicoCityCoordinates != 1 {
		t.Fatalf("expected exactly one mapped Mexico City location seed, got %d", mexicoCityCoordinates)
	}

	if _, err := db.Exec(`
		INSERT INTO account_settings (
			id,
			username,
			password_hash,
			api_token_hash,
			api_token_hint,
			avatar_path,
			recovery_public_key
		) VALUES (
			'account',
			'admin',
			'hash-1',
			'token-hash-1',
			'hint-1',
			'/static/img/avatar-1.svg',
			'public-key-1'
		)
	`); err != nil {
		t.Fatalf("expected first account_settings insert to succeed: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO account_settings (
			id,
			username,
			password_hash,
			api_token_hash,
			api_token_hint,
			avatar_path,
			recovery_public_key
		) VALUES (
			'account-2',
			'operator',
			'hash-2',
			'token-hash-2',
			'hint-2',
			'/static/img/avatar-2.svg',
			'public-key-2'
		)
	`); err == nil {
		t.Fatal("expected second account_settings insert to fail because the table is singleton")
	}
}

func TestRunMigrationsFailsForLegacyMultipleAppSettingsRows(t *testing.T) {
	db := openTestDB(t)
	logger := testLogger()

	_, err := db.Exec(`
		CREATE TABLE migrations (id INTEGER PRIMARY KEY, name TEXT UNIQUE);
		INSERT INTO migrations (name) VALUES ('001_create_tables.sql');

		CREATE TABLE account_settings (
			id TEXT PRIMARY KEY NOT NULL,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			email TEXT NOT NULL UNIQUE,
			settings TEXT DEFAULT '{}',
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE app_settings (
			id TEXT PRIMARY KEY NOT NULL,
			show_version_footer BOOLEAN DEFAULT true,
			default_server_os TEXT NOT NULL,
			default_curency TEXT NOT NULL,
			due_soon_amount INT NOT NULL DEFAULT 5,
			recent_add_amount INT NOT NULL DEFAULT 5,
			description TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		INSERT INTO app_settings (
			id,
			show_version_footer,
			default_server_os,
			default_curency,
			due_soon_amount,
			recent_add_amount,
			description
		) VALUES
			('app-a', 1, 'Linux', 'USD', 5, 5, 'first'),
			('app-b', 1, 'BSD', 'EUR', 3, 3, 'second');
	`)
	if err != nil {
		t.Fatalf("could not prepare legacy schema: %v", err)
	}

	err = RunMigrations(db, logger)
	if err == nil {
		t.Fatal("expected migration to fail when legacy app_settings contains multiple rows")
	}

	if !strings.Contains(err.Error(), "002_enforce_singleton_settings.sql") {
		t.Fatalf("expected error to mention 002 migration, got %v", err)
	}
}

func TestRunMigrationsMovesLegacyDashboardExpenseHistoryIntoExpenseEntries(t *testing.T) {
	db := openTestDB(t)
	logger := testLogger()

	_, err := db.Exec(`
		CREATE TABLE migrations (id INTEGER PRIMARY KEY, name TEXT UNIQUE);
		INSERT INTO migrations (name) VALUES
			('001_create_tables.sql'),
			('002_enforce_singleton_settings.sql'),
			('003_auth_and_settings.sql');

		CREATE TABLE account_settings (
			id TEXT PRIMARY KEY NOT NULL DEFAULT 'account',
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			api_token_hash TEXT NOT NULL DEFAULT '',
			api_token_hint TEXT NOT NULL DEFAULT '',
			avatar_path TEXT NOT NULL DEFAULT '/static/img/avatar-1.svg',
			recovery_public_key TEXT NOT NULL DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE app_settings (
			id TEXT PRIMARY KEY NOT NULL DEFAULT 'app',
			show_version_footer BOOLEAN NOT NULL DEFAULT 1,
			default_server_os TEXT NOT NULL DEFAULT 'Linux',
			default_currency TEXT NOT NULL DEFAULT 'MXN',
			dashboard_currency TEXT NOT NULL DEFAULT 'MXN',
			dashboard_due_soon_amount INTEGER NOT NULL DEFAULT 5,
			dashboard_expense_history_json TEXT NOT NULL DEFAULT '[]',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE domains (
			id TEXT PRIMARY KEY NOT NULL,
			domain TEXT NOT NULL UNIQUE,
			provider_id TEXT,
			due_date DATE,
			price REAL,
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE hostings (
			id TEXT PRIMARY KEY NOT NULL,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			location_id INTEGER,
			provider_id TEXT,
			disk_gb INTEGER,
			domain_id INTEGER,
			price REAL,
			due_date DATE,
			since_date DATE,
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE servers (
			id TEXT PRIMARY KEY NOT NULL,
			hostname TEXT NOT NULL UNIQUE,
			type TEXT NOT NULL,
			os_id INTEGER,
			cpu_cores INTEGER,
			memory_gb INTEGER,
			disk_gb INTEGER,
			location_id INTEGER,
			provider_id TEXT,
			due_date DATE,
			price REAL,
			since_date DATE,
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE subscriptions (
			id TEXT PRIMARY KEY NOT NULL,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			price REAL,
			due_date DATE,
			since_date DATE,
			renewal_period TEXT,
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		INSERT INTO app_settings (
			id,
			show_version_footer,
			default_server_os,
			default_currency,
			dashboard_currency,
			dashboard_due_soon_amount,
			dashboard_expense_history_json
		) VALUES (
			'app',
			1,
			'Linux',
			'MXN',
			'MXN',
			5,
			'["Registrar renewal","March infra budget"]'
		);
	`)
	if err != nil {
		t.Fatalf("could not prepare legacy app settings schema: %v", err)
	}

	if err := RunMigrations(db, logger); err != nil {
		t.Fatalf("RunMigrations returned error: %v", err)
	}

	var expenseCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM expense_entries`).Scan(&expenseCount); err != nil {
		t.Fatalf("could not count migrated expense entries: %v", err)
	}

	if expenseCount != 2 {
		t.Fatalf("expected 2 migrated expense entries, got %d", expenseCount)
	}

	var historyJSON string
	if err := db.QueryRow(`SELECT dashboard_expense_history_json FROM app_settings WHERE id = 'app'`).Scan(&historyJSON); err != nil {
		t.Fatalf("could not read app settings history after migration: %v", err)
	}

	if historyJSON != "[]" {
		t.Fatalf("expected legacy dashboard expense history to be cleared, got %q", historyJSON)
	}
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open returned error: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
