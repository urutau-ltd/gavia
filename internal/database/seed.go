package database

import (
	"database/sql"
	"fmt"

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

// SeedReferenceData loads deterministic bootstrap rows that are safe to create
// automatically on a fresh install. Account credentials remain manual.
func SeedReferenceData(db *sql.DB) error {
	if err := SeedProviders(db); err != nil {
		return err
	}

	return SeedAppSettings(db)
}

func SeedProviders(db *sql.DB) error {
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
