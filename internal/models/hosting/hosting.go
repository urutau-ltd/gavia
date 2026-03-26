package hosting

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Hosting struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	LocationID   *string   `json:"location_id"`
	LocationName *string   `json:"location_name"`
	ProviderID   *string   `json:"provider_id"`
	ProviderName *string   `json:"provider_name"`
	DomainID     *string   `json:"domain_id"`
	DomainName   *string   `json:"domain_name"`
	DiskGB       *int      `json:"disk_gb"`
	Price        *float64  `json:"price"`
	Currency     string    `json:"currency"`
	DueDate      *string   `json:"due_date"`
	SinceDate    *string   `json:"since_date"`
	Notes        *string   `json:"notes"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (h *Hosting) LocationIDValue() string {
	return stringValue(h.LocationID)
}

func (h *Hosting) LocationNameValue() string {
	return stringValue(h.LocationName)
}

func (h *Hosting) ProviderIDValue() string {
	return stringValue(h.ProviderID)
}

func (h *Hosting) ProviderNameValue() string {
	return stringValue(h.ProviderName)
}

func (h *Hosting) DomainIDValue() string {
	return stringValue(h.DomainID)
}

func (h *Hosting) DomainNameValue() string {
	return stringValue(h.DomainName)
}

func (h *Hosting) DiskGBValue() string {
	if h == nil || h.DiskGB == nil {
		return ""
	}

	return strconv.Itoa(*h.DiskGB)
}

func (h *Hosting) PriceDisplay() string {
	if h == nil || h.Price == nil {
		return ""
	}

	return fmt.Sprintf("%.2f", *h.Price)
}

func (h *Hosting) DueDateValue() string {
	return stringValue(h.DueDate)
}

func (h *Hosting) SinceDateValue() string {
	return stringValue(h.SinceDate)
}

func (h *Hosting) CurrencyValue(defaultCurrency string) string {
	value := strings.ToUpper(strings.TrimSpace(h.Currency))
	if value != "" {
		return value
	}

	value = strings.ToUpper(strings.TrimSpace(defaultCurrency))
	if value != "" {
		return value
	}

	return "MXN"
}

func (h *Hosting) NotesValue() string {
	return stringValue(h.Notes)
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, item *Hosting) error {
	newID, err := uuid.NewV7()
	if err != nil {
		return err
	}

	now := time.Now()
	item.ID = newID.String()
	item.Currency = normalizeCurrency(item.Currency)
	item.CreatedAt = now
	item.UpdatedAt = now

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO hostings (
			id,
			name,
			type,
			location_id,
			provider_id,
			domain_id,
			disk_gb,
			price,
			currency,
			due_date,
			since_date,
			notes,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		item.ID,
		item.Name,
		item.Type,
		dbString(item.LocationID),
		dbString(item.ProviderID),
		dbString(item.DomainID),
		dbInt(item.DiskGB),
		dbFloat(item.Price),
		item.Currency,
		dbString(item.DueDate),
		dbString(item.SinceDate),
		dbString(item.Notes),
		item.CreatedAt,
		item.UpdatedAt,
	)
	return err
}

func (r *Repository) GetByID(ctx context.Context, id string) (*Hosting, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, fmt.Errorf("invalid uuid format: %w", err)
	}

	item := &Hosting{}
	var locationID sql.NullString
	var locationName sql.NullString
	var providerID sql.NullString
	var providerName sql.NullString
	var domainID sql.NullString
	var domainName sql.NullString
	var diskGB sql.NullInt64
	var price sql.NullFloat64
	var dueDate sql.NullString
	var sinceDate sql.NullString
	var notes sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT
			h.id,
			h.name,
			h.type,
			h.location_id,
			l.name,
			h.provider_id,
			p.name,
			h.domain_id,
			d.domain,
			h.disk_gb,
			h.price,
			h.currency,
			h.due_date,
			h.since_date,
			h.notes,
			h.created_at,
			h.updated_at
		FROM hostings h
		LEFT JOIN locations l ON l.id = h.location_id
		LEFT JOIN providers p ON p.id = h.provider_id
		LEFT JOIN domains d ON d.id = h.domain_id
		WHERE h.id = ?
	`, id).Scan(
		&item.ID,
		&item.Name,
		&item.Type,
		&locationID,
		&locationName,
		&providerID,
		&providerName,
		&domainID,
		&domainName,
		&diskGB,
		&price,
		&item.Currency,
		&dueDate,
		&sinceDate,
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

	item.LocationID = nullString(locationID)
	item.LocationName = nullString(locationName)
	item.ProviderID = nullString(providerID)
	item.ProviderName = nullString(providerName)
	item.DomainID = nullString(domainID)
	item.DomainName = nullString(domainName)
	item.DiskGB = nullInt(diskGB)
	item.Price = nullFloat(price)
	item.DueDate = nullString(dueDate)
	item.SinceDate = nullString(sinceDate)
	item.Notes = nullString(notes)
	return item, nil
}

func (r *Repository) GetAll(ctx context.Context, searchTerm string, limit int) ([]*Hosting, error) {
	query := `
		SELECT
			h.id,
			h.name,
			h.type,
			h.location_id,
			l.name,
			h.provider_id,
			p.name,
			h.domain_id,
			d.domain,
			h.disk_gb,
			h.price,
			h.currency,
			h.due_date,
			h.since_date,
			h.notes,
			h.created_at,
			h.updated_at
		FROM hostings h
		LEFT JOIN locations l ON l.id = h.location_id
		LEFT JOIN providers p ON p.id = h.provider_id
		LEFT JOIN domains d ON d.id = h.domain_id
		WHERE 1 = 1
	`
	var args []any

	if searchTerm != "" {
		search := "%" + searchTerm + "%"
		query += ` AND (
			h.name LIKE ?
			OR h.type LIKE ?
			OR COALESCE(l.name, '') LIKE ?
			OR COALESCE(p.name, '') LIKE ?
			OR COALESCE(d.domain, '') LIKE ?
			OR COALESCE(h.currency, '') LIKE ?
		)`
		args = append(args, search, search, search, search, search, search)
	}

	query += ` ORDER BY h.name`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*Hosting
	for rows.Next() {
		item, scanErr := scanHosting(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *Repository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM hostings`).Scan(&count)
	return count, err
}

func (r *Repository) Update(ctx context.Context, item *Hosting) error {
	item.Currency = normalizeCurrency(item.Currency)
	item.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, `
		UPDATE hostings
		SET
			name = ?,
			type = ?,
			location_id = ?,
			provider_id = ?,
			domain_id = ?,
			disk_gb = ?,
			price = ?,
			currency = ?,
			due_date = ?,
			since_date = ?,
			notes = ?,
			updated_at = ?
		WHERE id = ?
	`,
		item.Name,
		item.Type,
		dbString(item.LocationID),
		dbString(item.ProviderID),
		dbString(item.DomainID),
		dbInt(item.DiskGB),
		dbFloat(item.Price),
		item.Currency,
		dbString(item.DueDate),
		dbString(item.SinceDate),
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

	_, err := r.db.ExecContext(ctx, `DELETE FROM hostings WHERE id = ?`, id)
	return err
}

func scanHosting(scanner interface {
	Scan(dest ...any) error
}) (*Hosting, error) {
	item := &Hosting{}
	var locationID sql.NullString
	var locationName sql.NullString
	var providerID sql.NullString
	var providerName sql.NullString
	var domainID sql.NullString
	var domainName sql.NullString
	var diskGB sql.NullInt64
	var price sql.NullFloat64
	var dueDate sql.NullString
	var sinceDate sql.NullString
	var notes sql.NullString
	if err := scanner.Scan(
		&item.ID,
		&item.Name,
		&item.Type,
		&locationID,
		&locationName,
		&providerID,
		&providerName,
		&domainID,
		&domainName,
		&diskGB,
		&price,
		&item.Currency,
		&dueDate,
		&sinceDate,
		&notes,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}

	item.LocationID = nullString(locationID)
	item.LocationName = nullString(locationName)
	item.ProviderID = nullString(providerID)
	item.ProviderName = nullString(providerName)
	item.DomainID = nullString(domainID)
	item.DomainName = nullString(domainName)
	item.DiskGB = nullInt(diskGB)
	item.Price = nullFloat(price)
	item.DueDate = nullString(dueDate)
	item.SinceDate = nullString(sinceDate)
	item.Notes = nullString(notes)
	return item, nil
}

func nullString(value sql.NullString) *string {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return nil
	}

	return &value.String
}

func nullInt(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}

	intValue := int(value.Int64)
	return &intValue
}

func nullFloat(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}

	return &value.Float64
}

func dbString(value *string) any {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}

	return strings.TrimSpace(*value)
}

func dbInt(value *int) any {
	if value == nil {
		return nil
	}

	return *value
}

func dbFloat(value *float64) any {
	if value == nil {
		return nil
	}

	return *value
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func normalizeCurrency(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" {
		return "MXN"
	}

	return value
}
