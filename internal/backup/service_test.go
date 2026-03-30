package backup

import (
	"database/sql"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"codeberg.org/urutau-ltd/gavia/internal/database"
	accountsetting "codeberg.org/urutau-ltd/gavia/internal/models/account_setting"
	"codeberg.org/urutau-ltd/gavia/internal/security"
	_ "modernc.org/sqlite"
)

func TestExportImportEncryptedJSONRoundTrip(t *testing.T) {
	db := openBackupTestDB(t)
	runBackupMigrations(t, db)
	recoveryKey := createBackupFixtures(t, db)

	service := NewService(db)
	payload, err := service.ExportEncryptedJSON(t.Context(), mustAccountPublicKey(t, db))
	if err != nil {
		t.Fatalf("ExportEncryptedJSON returned error: %v", err)
	}

	snapshot, err := service.ParseImport(payload, recoveryKey)
	if err != nil {
		t.Fatalf("ParseImport returned error: %v", err)
	}

	if _, err := db.Exec(`DELETE FROM providers`); err != nil {
		t.Fatalf("could not clear providers: %v", err)
	}

	if _, err := db.Exec(`UPDATE app_settings SET default_currency = 'USD' WHERE id = 'app'`); err != nil {
		t.Fatalf("could not mutate app settings before import: %v", err)
	}

	if err := service.Import(t.Context(), snapshot); err != nil {
		t.Fatalf("Import returned error: %v", err)
	}

	var providerExists bool
	if err := db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM providers WHERE id = 'provider-1' AND name = 'Provider 1')
	`).Scan(&providerExists); err != nil {
		t.Fatalf("could not verify imported provider fixture: %v", err)
	}

	if !providerExists {
		t.Fatal("expected backup import to restore the provider fixture")
	}

	var defaultCurrency string
	if err := db.QueryRow(`SELECT default_currency FROM app_settings WHERE id = 'app'`).Scan(&defaultCurrency); err != nil {
		t.Fatalf("could not read app settings after import: %v", err)
	}

	if defaultCurrency != "MXN" {
		t.Fatalf("expected imported default currency MXN, got %q", defaultCurrency)
	}

	var expenseExists bool
	if err := db.QueryRow(`
		SELECT EXISTS(
			SELECT 1
			FROM expense_entries
			WHERE title = 'Hetzner invoice'
				AND amount = 42.50
				AND account_name = 'banregio checking'
				AND counterparty = 'Hetzner'
				AND payment_method = 'wire'
		)
	`).Scan(&expenseExists); err != nil {
		t.Fatalf("could not verify imported expense entry fixture: %v", err)
	}

	if !expenseExists {
		t.Fatal("expected backup import to restore the expense entry fixture")
	}

	var locationMapped bool
	if err := db.QueryRow(`
		SELECT EXISTS(
			SELECT 1
			FROM locations
			WHERE id = 'location-1'
				AND latitude = 25.686613
				AND longitude = -100.316116
		)
	`).Scan(&locationMapped); err != nil {
		t.Fatalf("could not verify imported location coordinates: %v", err)
	}

	if !locationMapped {
		t.Fatal("expected backup import to restore location coordinates")
	}

	var domainExists bool
	if err := db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM domains WHERE id = 'domain-1' AND domain = 'example.com')
	`).Scan(&domainExists); err != nil {
		t.Fatalf("could not verify imported domain fixture: %v", err)
	}

	if !domainExists {
		t.Fatal("expected backup import to restore the domain fixture")
	}

	var hostingExists bool
	if err := db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM hostings WHERE id = 'hosting-1' AND name = 'Shared Hosting Starter')
	`).Scan(&hostingExists); err != nil {
		t.Fatalf("could not verify imported hosting fixture: %v", err)
	}

	if !hostingExists {
		t.Fatal("expected backup import to restore the hosting fixture")
	}

	var serverExists bool
	if err := db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM servers WHERE id = 'server-1' AND hostname = 'srv-01.example.com')
	`).Scan(&serverExists); err != nil {
		t.Fatalf("could not verify imported server fixture: %v", err)
	}

	if !serverExists {
		t.Fatal("expected backup import to restore the server fixture")
	}

	var subscriptionExists bool
	if err := db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM subscriptions WHERE id = 'subscription-1' AND name = 'Status page')
	`).Scan(&subscriptionExists); err != nil {
		t.Fatalf("could not verify imported subscription fixture: %v", err)
	}

	if !subscriptionExists {
		t.Fatal("expected backup import to restore the subscription fixture")
	}

	var ipExists bool
	if err := db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM ips WHERE id = 'ip-1' AND address = '203.0.113.10')
	`).Scan(&ipExists); err != nil {
		t.Fatalf("could not verify imported IP fixture: %v", err)
	}

	if !ipExists {
		t.Fatal("expected backup import to restore the IP fixture")
	}

	var dnsExists bool
	if err := db.QueryRow(`
		SELECT EXISTS(
			SELECT 1
			FROM dns_records
			WHERE id = 'dns-1'
				AND hostname = 'app.example.com'
				AND domain_id = 'domain-1'
		)
	`).Scan(&dnsExists); err != nil {
		t.Fatalf("could not verify imported DNS fixture: %v", err)
	}

	if !dnsExists {
		t.Fatal("expected backup import to restore the DNS fixture")
	}

	var labelExists bool
	if err := db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM labels WHERE id = 'label-1' AND name = 'production')
	`).Scan(&labelExists); err != nil {
		t.Fatalf("could not verify imported label fixture: %v", err)
	}

	if !labelExists {
		t.Fatal("expected backup import to restore the label fixture")
	}

	var serverIPExists bool
	if err := db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM server_ips WHERE id = 'server-ip-1' AND server_id = 'server-1' AND ip_id = 'ip-1')
	`).Scan(&serverIPExists); err != nil {
		t.Fatalf("could not verify imported server IP assignment fixture: %v", err)
	}

	if !serverIPExists {
		t.Fatal("expected backup import to restore the server IP assignment fixture")
	}

	var serverLabelExists bool
	if err := db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM server_labels WHERE id = 'server-label-1' AND server_id = 'server-1' AND label_id = 'label-1')
	`).Scan(&serverLabelExists); err != nil {
		t.Fatalf("could not verify imported server label assignment fixture: %v", err)
	}

	if !serverLabelExists {
		t.Fatal("expected backup import to restore the server label assignment fixture")
	}
}

func openBackupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "backup-test.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open returned error: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func runBackupMigrations(t *testing.T, db *sql.DB) {
	t.Helper()

	if err := database.RunMigrations(db, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		t.Fatalf("RunMigrations returned error: %v", err)
	}

	if err := database.SeedReferenceData(db); err != nil {
		t.Fatalf("SeedReferenceData returned error: %v", err)
	}
}

func createBackupFixtures(t *testing.T, db *sql.DB) string {
	t.Helper()

	passwordHash, err := security.HashPassword("backup-test-password")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	publicKey, recoveryKey, err := security.GenerateRecoveryKeyPair()
	if err != nil {
		t.Fatalf("GenerateRecoveryKeyPair returned error: %v", err)
	}

	repo := accountsetting.NewAccountSettingsRepository(db)
	if err := repo.Create(t.Context(), &accountsetting.AccountSettings{
		Username:          "admin",
		PasswordHash:      passwordHash,
		APITokenHash:      "api-token-hash",
		APITokenHint:      "token-hint",
		AvatarPath:        "/static/img/avatar-1.svg",
		RecoveryPublicKey: publicKey,
	}); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO providers (id, name, website, notes, created_at, updated_at)
		VALUES ('provider-1', 'Provider 1', 'https://example.com', 'backup fixture', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`); err != nil {
		t.Fatalf("could not insert provider fixture: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO locations (id, name, city, country, latitude, longitude, notes, created_at, updated_at)
		VALUES ('location-1', 'HQ', 'Monterrey', 'Mexico', 25.686613, -100.316116, 'backup fixture', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`); err != nil {
		t.Fatalf("could not insert location fixture: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO operating_systems (id, name, notes, created_at, updated_at)
		VALUES ('os-1', 'Ubuntu 24.04', 'backup fixture', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`); err != nil {
		t.Fatalf("could not insert operating system fixture: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO ips (id, address, type, city, country, org, asn, isp, notes, created_at, updated_at)
		VALUES ('ip-1', '203.0.113.10', 'ipv4', 'Monterrey', 'Mexico', 'Example Org', 'AS64500', 'Example ISP', 'backup fixture', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`); err != nil {
		t.Fatalf("could not insert IP fixture: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO labels (id, name, notes, created_at, updated_at)
		VALUES ('label-1', 'production', 'backup fixture', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`); err != nil {
		t.Fatalf("could not insert label fixture: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO domains (id, domain, provider_id, due_date, currency, price, notes, created_at, updated_at)
		VALUES ('domain-1', 'example.com', 'provider-1', '2026-12-01', 'MXN', 120.00, 'backup fixture', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`); err != nil {
		t.Fatalf("could not insert domain fixture: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO dns_records (id, type, hostname, domain_id, address, notes, created_at, updated_at)
		VALUES ('dns-1', 'A', 'app.example.com', 'domain-1', '203.0.113.10', 'backup fixture', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`); err != nil {
		t.Fatalf("could not insert DNS fixture: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO hostings (id, name, type, location_id, provider_id, domain_id, disk_gb, price, currency, due_date, since_date, notes, created_at, updated_at)
		VALUES ('hosting-1', 'Shared Hosting Starter', 'shared', 'location-1', 'provider-1', 'domain-1', 50, 299.00, 'MXN', '2026-12-05', '2026-01-05', 'backup fixture', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`); err != nil {
		t.Fatalf("could not insert hosting fixture: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO servers (id, hostname, type, os_id, cpu_cores, memory_gb, disk_gb, location_id, provider_id, due_date, price, currency, since_date, notes, created_at, updated_at)
		VALUES ('server-1', 'srv-01.example.com', 'vps', 'os-1', 4, 8, 160, 'location-1', 'provider-1', '2026-12-10', 499.00, 'MXN', '2026-02-10', 'backup fixture', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`); err != nil {
		t.Fatalf("could not insert server fixture: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO subscriptions (id, name, type, price, currency, due_date, since_date, renewal_period, notes, created_at, updated_at)
		VALUES ('subscription-1', 'Status page', 'saas', 19.00, 'USD', '2026-12-15', '2026-01-15', 'monthly', 'backup fixture', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`); err != nil {
		t.Fatalf("could not insert subscription fixture: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO server_ips (id, server_id, ip_id)
		VALUES ('server-ip-1', 'server-1', 'ip-1')
	`); err != nil {
		t.Fatalf("could not insert server IP assignment fixture: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO server_labels (id, server_id, label_id)
		VALUES ('server-label-1', 'server-1', 'label-1')
	`); err != nil {
		t.Fatalf("could not insert server label assignment fixture: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO expense_entries (
			id,
			title,
			entry_type,
			account_name,
			category,
			counterparty,
			scope,
			amount,
			currency,
			occurred_on,
			due_on,
			paid_on,
			payment_method,
			notes,
			created_at,
			updated_at
		) VALUES (
			'expense-1',
			'Hetzner invoice',
			'expense',
			'banregio checking',
			'hosting',
			'Hetzner',
			'infrastructure',
			42.50,
			'MXN',
			'2026-03-10',
			'2026-03-10',
			'2026-03-10',
			'wire',
			'backup fixture',
			CURRENT_TIMESTAMP,
			CURRENT_TIMESTAMP
		)
	`); err != nil {
		t.Fatalf("could not insert expense entry fixture: %v", err)
	}

	return recoveryKey
}

func mustAccountPublicKey(t *testing.T, db *sql.DB) string {
	t.Helper()

	var publicKey string
	if err := db.QueryRow(`SELECT recovery_public_key FROM account_settings WHERE id = 'account'`).Scan(&publicKey); err != nil {
		t.Fatalf("could not read account recovery public key: %v", err)
	}

	return publicKey
}
