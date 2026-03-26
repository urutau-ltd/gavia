package ip

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type IP struct {
	ID        string    `json:"id"`
	Address   string    `json:"address"`
	Type      string    `json:"type"`
	City      *string   `json:"city"`
	Country   *string   `json:"country"`
	Org       *string   `json:"org"`
	ASN       *string   `json:"asn"`
	ISP       *string   `json:"isp"`
	Notes     *string   `json:"notes"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (i *IP) CityValue() string    { return stringValue(i.City) }
func (i *IP) CountryValue() string { return stringValue(i.Country) }
func (i *IP) OrgValue() string     { return stringValue(i.Org) }
func (i *IP) ASNValue() string     { return stringValue(i.ASN) }
func (i *IP) ISPValue() string     { return stringValue(i.ISP) }
func (i *IP) NotesValue() string   { return stringValue(i.Notes) }

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, item *IP) error {
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
		INSERT INTO ips (
			id,
			address,
			type,
			city,
			country,
			org,
			asn,
			isp,
			notes,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		item.ID,
		item.Address,
		item.Type,
		dbString(item.City),
		dbString(item.Country),
		dbString(item.Org),
		dbString(item.ASN),
		dbString(item.ISP),
		dbString(item.Notes),
		item.CreatedAt,
		item.UpdatedAt,
	)
	return err
}

func (r *Repository) GetByID(ctx context.Context, id string) (*IP, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, fmt.Errorf("invalid uuid format: %w", err)
	}

	item := &IP{}
	var city sql.NullString
	var country sql.NullString
	var org sql.NullString
	var asn sql.NullString
	var isp sql.NullString
	var notes sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT id, address, type, city, country, org, asn, isp, notes, created_at, updated_at
		FROM ips
		WHERE id = ?
	`, id).Scan(
		&item.ID,
		&item.Address,
		&item.Type,
		&city,
		&country,
		&org,
		&asn,
		&isp,
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

	item.City = nullString(city)
	item.Country = nullString(country)
	item.Org = nullString(org)
	item.ASN = nullString(asn)
	item.ISP = nullString(isp)
	item.Notes = nullString(notes)
	return item, nil
}

func (r *Repository) GetAll(ctx context.Context, searchTerm string, limit int) ([]*IP, error) {
	query := `
		SELECT id, address, type, city, country, org, asn, isp, notes, created_at, updated_at
		FROM ips
		WHERE 1 = 1
	`
	var args []any

	if searchTerm != "" {
		search := "%" + searchTerm + "%"
		query += ` AND (
			address LIKE ?
			OR type LIKE ?
			OR COALESCE(city, '') LIKE ?
			OR COALESCE(country, '') LIKE ?
			OR COALESCE(org, '') LIKE ?
			OR COALESCE(asn, '') LIKE ?
			OR COALESCE(isp, '') LIKE ?
		)`
		args = append(args, search, search, search, search, search, search, search)
	}

	query += ` ORDER BY address`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*IP
	for rows.Next() {
		item, scanErr := scanIP(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *Repository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ips`).Scan(&count)
	return count, err
}

func (r *Repository) Update(ctx context.Context, item *IP) error {
	item.Type = normalizeType(item.Type)
	item.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, `
		UPDATE ips
		SET
			address = ?,
			type = ?,
			city = ?,
			country = ?,
			org = ?,
			asn = ?,
			isp = ?,
			notes = ?,
			updated_at = ?
		WHERE id = ?
	`,
		item.Address,
		item.Type,
		dbString(item.City),
		dbString(item.Country),
		dbString(item.Org),
		dbString(item.ASN),
		dbString(item.ISP),
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

	_, err := r.db.ExecContext(ctx, `DELETE FROM ips WHERE id = ?`, id)
	return err
}

func scanIP(scanner interface{ Scan(dest ...any) error }) (*IP, error) {
	item := &IP{}
	var city sql.NullString
	var country sql.NullString
	var org sql.NullString
	var asn sql.NullString
	var isp sql.NullString
	var notes sql.NullString
	if err := scanner.Scan(
		&item.ID,
		&item.Address,
		&item.Type,
		&city,
		&country,
		&org,
		&asn,
		&isp,
		&notes,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}

	item.City = nullString(city)
	item.Country = nullString(country)
	item.Org = nullString(org)
	item.ASN = nullString(asn)
	item.ISP = nullString(isp)
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
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "ipv6":
		return "ipv6"
	default:
		return "ipv4"
	}
}
