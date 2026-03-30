package database

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type seedProvider struct {
	Name    string
	Website string
	Notes   string
}

type seedAppSettings struct {
	ID                string
	ShowVersionFooter bool
	DefaultServerOS   string
	DefaultCurrency   string
	DashboardCurrency string
	DueSoonAmount     int
	ExpenseHistory    string
}

type seedOperatingSystem struct {
	Name  string
	Notes string
}

type seedLabel struct {
	Name  string
	Notes string
}

type seedLocation struct {
	Name      string
	City      string
	Country   string
	Latitude  *float64
	Longitude *float64
	Notes     string
}

// SeedReferenceData loads deterministic bootstrap rows that are safe to create
// automatically on a fresh install. Account credentials remain manual.
func SeedReferenceData(db *sql.DB) error {
	if err := SeedProviders(db); err != nil {
		return err
	}
	if err := SeedOperatingSystems(db); err != nil {
		return err
	}
	if err := SeedLabels(db); err != nil {
		return err
	}
	if err := SeedLocations(db); err != nil {
		return err
	}

	return SeedAppSettings(db)
}

func SeedProviders(db *sql.DB) error {
	shouldSeed, err := shouldSeedReferenceTable(db, "providers")
	if err != nil {
		return err
	}
	if !shouldSeed {
		return nil
	}

	providers := []seedProvider{
		{
			"GoDaddy",
			"https://www.godaddy.com/",
			"Large company, frequent upselling",
		},
		{
			"Google Domains",
			"https://domains.google/",
			"Integration with Google services",
		},
		{
			"Cloudflare Registrar",
			"https://www.cloudflare.com/registrar/",
			"Cost-price pricing, no markup",
		},
		{
			"Dynadot",
			"https://www.dynadot.com/",
			"Competitive pricing and marketplace",
		},
		{
			"OVHcloud",
			"https://www.ovh.com/",
			"European provider with additional services",
		},
		{
			"Name.com",
			"https://www.name.com/",
			"Good option for domains and SSL"},
		{
			"Porkbun",
			"https://porkbun.com/",
			"Competitive pricing and simple interface",
		},
		{
			"EuroRegister",
			"https://www.euroregister.com/",
			"Focus on the European market",
		},
		{
			"Domain.com",
			"https://www.domain.com/",
			"Commercial and marketing-focused",
		},
		{
			"Alibaba Cloud Domains",
			"https://www.alibabacloud.com/",
			"Strong in Asia, part of Alibaba Cloud",
		},
		{
			"Hetzner",
			"https://www.hetzner.com/",
			"German provider with web services",
		},
		{
			"OVH (regional)",
			"https://www.ovhcloud.com/",
			"Regional variants of OVH",
		},
		{
			"Dyn (Oracle Dyn)",
			"https://dyn.com/",
			"DNS and services, historically domain management",
		},
		{
			"Alibaba Domains (international)",
			"https://name.alibaba.com/",
			"Registration and related services",
		},
		{
			"Kavi",
			"https://www.kavi.com/",
			"Registrar and reseller",
		},
		{
			"Nic.at (Austrian resellers)",
			"https://www.nic.at/",
			"Resellers for .at",
		},
		{
			"NIC.ch (Switzerland resellers)",
			"https://www.nic.ch/",
			"Resellers for .ch",
		},
		{
			"MarkMonitor (corporate protection)",
			"https://www.markmonitor.com/",
			"Brand protection",
		},
		{
			"CSC Digital Brand Services",
			"https://www.cscglobal.com/",
			"Corporate domain portfolio management",
		},
	}

	return seedMany(
		db,
		`INSERT OR IGNORE INTO providers (id, name, website, notes) VALUES (?, ?, ?, ?)`,
		providers,
		func(stmt *sql.Stmt, p seedProvider) error {
			id, err := uuid.NewV7()
			if err != nil {
				return fmt.Errorf("could not generate provider id for %s: %w", p.Name, err)
			}

			if _, err := stmt.Exec(id.String(), p.Name, p.Website, p.Notes); err != nil {
				return fmt.Errorf("could not insert %s: %w", p.Name, err)
			}

			return nil
		},
	)
}

func SeedAppSettings(db *sql.DB) error {
	defaultSettings := []seedAppSettings{
		{
			ID:                "app",
			ShowVersionFooter: true,
			DefaultServerOS:   "Linux",
			DefaultCurrency:   "MXN",
			DashboardCurrency: "MXN",
			DueSoonAmount:     5,
			ExpenseHistory:    "[]",
		},
	}

	return seedMany(
		db,
		`INSERT OR IGNORE INTO app_settings (
			id,
			show_version_footer,
			default_server_os,
			default_currency,
			dashboard_currency,
			dashboard_due_soon_amount,
			dashboard_expense_history_json
		) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		defaultSettings,
		func(stmt *sql.Stmt, s seedAppSettings) error {
			if _, err := stmt.Exec(
				s.ID,
				s.ShowVersionFooter,
				s.DefaultServerOS,
				s.DefaultCurrency,
				s.DashboardCurrency,
				s.DueSoonAmount,
				s.ExpenseHistory,
			); err != nil {
				return fmt.Errorf("could not insert app settings: %w", err)
			}

			return nil
		},
	)
}

func SeedOperatingSystems(db *sql.DB) error {
	shouldSeed, err := shouldSeedReferenceTable(db, "operating_systems")
	if err != nil {
		return err
	}
	if !shouldSeed {
		return nil
	}

	items := []seedOperatingSystem{
		{Name: "Linux", Notes: "Generic Linux baseline kept for the default server O.S. setting."},
		{Name: "Ubuntu Server 24.04 LTS", Notes: "Common long-term support default for VPS and bare metal."},
		{Name: "Debian 12 (Bookworm)", Notes: "Stable and conservative base system."},
		{Name: "Rocky Linux 9", Notes: "Enterprise Linux compatible distribution."},
		{Name: "AlmaLinux 9", Notes: "Another Enterprise Linux compatible option."},
		{Name: "Alpine Linux 3.20", Notes: "Minimal footprint system for containers and lightweight nodes."},
		{Name: "FreeBSD 14.1", Notes: "Solid choice for BSD-oriented infrastructure."},
		{Name: "OpenBSD 7.6", Notes: "Security-focused BSD with careful defaults."},
		{Name: "NetBSD 10", Notes: "Portable BSD for eclectic setups."},
		{Name: "Windows Server 2022", Notes: "Windows workload baseline."},
	}

	return seedMany(
		db,
		`INSERT OR IGNORE INTO operating_systems (id, name, notes) VALUES (?, ?, ?)`,
		items,
		func(stmt *sql.Stmt, item seedOperatingSystem) error {
			id, err := uuid.NewV7()
			if err != nil {
				return fmt.Errorf("could not generate operating system id for %s: %w", item.Name, err)
			}

			if _, err := stmt.Exec(id.String(), item.Name, nullableSeedText(item.Notes)); err != nil {
				return fmt.Errorf("could not insert operating system %s: %w", item.Name, err)
			}

			return nil
		},
	)
}

func SeedLabels(db *sql.DB) error {
	shouldSeed, err := shouldSeedReferenceTable(db, "labels")
	if err != nil {
		return err
	}
	if !shouldSeed {
		return nil
	}

	items := []seedLabel{
		{Name: "production", Notes: "Live systems or records that should be treated as real."},
		{Name: "staging", Notes: "Pre-release systems used for verification."},
		{Name: "backup", Notes: "Assets dedicated to backup, replication or disaster recovery."},
		{Name: "monitoring", Notes: "Monitoring, probes or observability-related assets."},
		{Name: "billing", Notes: "Records worth revisiting during invoices or renewals."},
		{Name: "personal", Notes: "Personal or low-stakes inventory outside production."},
		{Name: "lab", Notes: "Sandbox, experiments or disposable infrastructure."},
	}

	return seedMany(
		db,
		`INSERT OR IGNORE INTO labels (id, name, notes) VALUES (?, ?, ?)`,
		items,
		func(stmt *sql.Stmt, item seedLabel) error {
			id, err := uuid.NewV7()
			if err != nil {
				return fmt.Errorf("could not generate label id for %s: %w", item.Name, err)
			}

			if _, err := stmt.Exec(id.String(), item.Name, nullableSeedText(item.Notes)); err != nil {
				return fmt.Errorf("could not insert label %s: %w", item.Name, err)
			}

			return nil
		},
	)
}

func SeedLocations(db *sql.DB) error {
	shouldSeed, err := shouldSeedReferenceTable(db, "locations")
	if err != nil {
		return err
	}
	if !shouldSeed {
		return nil
	}

	items := []seedLocation{
		{
			Name:      "Mexico City",
			City:      "Mexico City",
			Country:   "Mexico",
			Latitude:  seedFloat(19.432608),
			Longitude: seedFloat(-99.133209),
			Notes:     "Starter pin for the local operator.",
		},
		{
			Name:      "Queretaro",
			City:      "Queretaro",
			Country:   "Mexico",
			Latitude:  seedFloat(20.588793),
			Longitude: seedFloat(-100.389888),
			Notes:     "One of the usual local data-center reference points.",
		},
		{
			Name:      "Ashburn",
			City:      "Ashburn",
			Country:   "United States",
			Latitude:  seedFloat(39.043757),
			Longitude: seedFloat(-77.487442),
			Notes:     "Because every infra map eventually mentions Ashburn.",
		},
		{
			Name:      "Frankfurt",
			City:      "Frankfurt",
			Country:   "Germany",
			Latitude:  seedFloat(50.110924),
			Longitude: seedFloat(8.682127),
			Notes:     "A respectful nod to European hosting gravity.",
		},
		{
			Name:      "Montevideo",
			City:      "Montevideo",
			Country:   "Uruguay",
			Latitude:  seedFloat(-34.901112),
			Longitude: seedFloat(-56.164532),
			Notes:     "A subtle southern nod in the default map pins.",
		},
	}

	return seedMany(
		db,
		`INSERT OR IGNORE INTO locations (id, name, city, country, latitude, longitude, notes) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		items,
		func(stmt *sql.Stmt, item seedLocation) error {
			id, err := uuid.NewV7()
			if err != nil {
				return fmt.Errorf("could not generate location id for %s: %w", item.Name, err)
			}

			if _, err := stmt.Exec(
				id.String(),
				item.Name,
				nullableSeedText(item.City),
				nullableSeedText(item.Country),
				nullableSeedFloat(item.Latitude),
				nullableSeedFloat(item.Longitude),
				nullableSeedText(item.Notes),
			); err != nil {
				return fmt.Errorf("could not insert location %s: %w", item.Name, err)
			}

			return nil
		},
	)
}

func seedMany[T any](db *sql.DB, query string, rows []T, exec func(*sql.Stmt, T) error) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(query)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, row := range rows {
		if err := exec(stmt, row); err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func shouldSeedReferenceTable(db *sql.DB, table string) (bool, error) {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil {
		return false, fmt.Errorf("could not count existing rows in %s: %w", table, err)
	}

	return count == 0, nil
}

func nullableSeedText(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	return value
}

func nullableSeedFloat(value *float64) any {
	if value == nil {
		return nil
	}

	return *value
}

func seedFloat(value float64) *float64 {
	return &value
}
