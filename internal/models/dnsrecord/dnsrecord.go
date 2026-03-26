package dnsrecord

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type DNSRecord struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	Hostname   string    `json:"hostname"`
	DomainID   *string   `json:"domain_id"`
	DomainName *string   `json:"domain_name"`
	Address    string    `json:"address"`
	Notes      *string   `json:"notes"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (d *DNSRecord) NotesValue() string      { return stringValue(d.Notes) }
func (d *DNSRecord) DomainIDValue() string   { return stringValue(d.DomainID) }
func (d *DNSRecord) DomainNameValue() string { return stringValue(d.DomainName) }

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, item *DNSRecord) error {
	newID, err := uuid.NewV7()
	if err != nil {
		return err
	}

	now := time.Now()
	item.ID = newID.String()
	item.Type = normalizeType(item.Type)
	item.CreatedAt = now
	item.UpdatedAt = now

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO dns_records (
			id,
			type,
			hostname,
			domain_id,
			address,
			notes,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		item.ID,
		item.Type,
		item.Hostname,
		dbString(item.DomainID),
		item.Address,
		dbString(item.Notes),
		item.CreatedAt,
		item.UpdatedAt,
	)
	return err
}

func (r *Repository) GetByID(ctx context.Context, id string) (*DNSRecord, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, fmt.Errorf("invalid uuid format: %w", err)
	}

	item := &DNSRecord{}
	var domainID sql.NullString
	var domainName sql.NullString
	var notes sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT
			dr.id,
			dr.type,
			dr.hostname,
			dr.domain_id,
			d.domain,
			dr.address,
			dr.notes,
			dr.created_at,
			dr.updated_at
		FROM dns_records dr
		LEFT JOIN domains d ON d.id = dr.domain_id
		WHERE dr.id = ?
	`, id).Scan(
		&item.ID,
		&item.Type,
		&item.Hostname,
		&domainID,
		&domainName,
		&item.Address,
		&notes,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	item.DomainID = nullString(domainID)
	item.DomainName = nullString(domainName)
	item.Notes = nullString(notes)
	return item, nil
}

func (r *Repository) GetAll(ctx context.Context, searchTerm string, limit int) ([]*DNSRecord, error) {
	query := `
		SELECT
			dr.id,
			dr.type,
			dr.hostname,
			dr.domain_id,
			d.domain,
			dr.address,
			dr.notes,
			dr.created_at,
			dr.updated_at
		FROM dns_records dr
		LEFT JOIN domains d ON d.id = dr.domain_id
		WHERE 1 = 1
	`
	var args []any

	if searchTerm != "" {
		search := "%" + searchTerm + "%"
		query += ` AND (dr.type LIKE ? OR dr.hostname LIKE ? OR dr.address LIKE ? OR COALESCE(d.domain, '') LIKE ?)`
		args = append(args, search, search, search, search)
	}

	query += ` ORDER BY COALESCE(d.domain, ''), dr.hostname, dr.type`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*DNSRecord
	for rows.Next() {
		item, scanErr := scanDNSRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *Repository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dns_records`).Scan(&count)
	return count, err
}

func (r *Repository) Update(ctx context.Context, item *DNSRecord) error {
	item.Type = normalizeType(item.Type)
	item.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, `
		UPDATE dns_records
		SET
			type = ?,
			hostname = ?,
			domain_id = ?,
			address = ?,
			notes = ?,
			updated_at = ?
		WHERE id = ?
	`,
		item.Type,
		item.Hostname,
		dbString(item.DomainID),
		item.Address,
		dbString(item.Notes),
		item.UpdatedAt,
		item.ID,
	)
	return err
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("invalid uuid format: %w", err)
	}

	_, err := r.db.ExecContext(ctx, `DELETE FROM dns_records WHERE id = ?`, id)
	return err
}

func scanDNSRecord(scanner interface{ Scan(dest ...any) error }) (*DNSRecord, error) {
	item := &DNSRecord{}
	var domainID sql.NullString
	var domainName sql.NullString
	var notes sql.NullString
	if err := scanner.Scan(
		&item.ID,
		&item.Type,
		&item.Hostname,
		&domainID,
		&domainName,
		&item.Address,
		&notes,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}

	item.DomainID = nullString(domainID)
	item.DomainName = nullString(domainName)
	item.Notes = nullString(notes)
	return item, nil
}

func nullString(value sql.NullString) *string {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return nil
	}

	return &value.String
}

func dbString(value *string) any {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}

	return strings.TrimSpace(*value)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func normalizeType(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	switch value {
	case "AAAA", "CNAME", "MX", "TXT", "NS", "SOA", "SRV":
		return value
	default:
		return "A"
	}
}
