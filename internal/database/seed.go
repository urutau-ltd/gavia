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

	// Usamos una transacción para que sea ultra rápido (SQLite sufre con muchos inserts individuales)
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO providers (id, name, website, notes) VALUES (?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range providers {
		id, _ := uuid.NewV7()
		if _, err := stmt.Exec(id.String(), p.Name, p.Website, p.Notes); err != nil {
			tx.Rollback()
			return fmt.Errorf("could not insert %s: %w", p.Name, err)
		}
	}

	return tx.Commit()
}
