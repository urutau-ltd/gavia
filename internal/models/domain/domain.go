package domain

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Domain struct {
	ID           string    `json:"id"`
	Domain       string    `json:"domain"`
	ProviderID   *string   `json:"provider_id"`
	ProviderName *string   `json:"provider_name"`
	DueDate      *string   `json:"due_date"`
	Currency     string    `json:"currency"`
	Price        *float64  `json:"price"`
	Notes        *string   `json:"notes"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (d *Domain) ProviderIDValue() string {
	return stringValue(d.ProviderID)
}

func (d *Domain) ProviderNameValue() string {
	return stringValue(d.ProviderName)
}

func (d *Domain) DueDateValue() string {
	return stringValue(d.DueDate)
}

func (d *Domain) CurrencyValue(defaultCurrency string) string {
	value := strings.ToUpper(strings.TrimSpace(d.Currency))
	if value != "" {
		return value
	}

	value = strings.ToUpper(strings.TrimSpace(defaultCurrency))
	if value != "" {
		return value
	}

	return "MXN"
}

func (d *Domain) PriceDisplay() string {
	if d == nil || d.Price == nil {
		return ""
	}

	return fmt.Sprintf("%.2f", *d.Price)
}

func (d *Domain) NotesValue() string {
	return stringValue(d.Notes)
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, item *Domain) error {
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
		INSERT INTO domains (
			id,
			domain,
			provider_id,
			due_date,
			currency,
			price,
			notes,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		item.ID,
		item.Domain,
		dbString(item.ProviderID),
		dbString(item.DueDate),
		item.Currency,
		dbFloat(item.Price),
		dbString(item.Notes),
		item.CreatedAt,
		item.UpdatedAt,
	)
	return err
}

func (r *Repository) GetByID(ctx context.Context, id string) (*Domain, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, fmt.Errorf("invalid uuid format: %w", err)
	}

	item := &Domain{}
	var providerID sql.NullString
	var providerName sql.NullString
	var dueDate sql.NullString
	var price sql.NullFloat64
	var notes sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT
			d.id,
			d.domain,
			d.provider_id,
			p.name,
			d.due_date,
			d.currency,
			d.price,
			d.notes,
			d.created_at,
			d.updated_at
		FROM domains d
		LEFT JOIN providers p ON p.id = d.provider_id
		WHERE d.id = ?
	`, id).Scan(
		&item.ID,
		&item.Domain,
		&providerID,
		&providerName,
		&dueDate,
		&item.Currency,
		&price,
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

	item.ProviderID = nullString(providerID)
	item.ProviderName = nullString(providerName)
	item.DueDate = nullString(dueDate)
	item.Price = nullFloat(price)
	item.Notes = nullString(notes)
	return item, nil
}

func (r *Repository) GetAll(ctx context.Context, searchTerm string, limit int) ([]*Domain, error) {
	query := `
		SELECT
			d.id,
			d.domain,
			d.provider_id,
			p.name,
			d.due_date,
			d.currency,
			d.price,
			d.notes,
			d.created_at,
			d.updated_at
		FROM domains d
		LEFT JOIN providers p ON p.id = d.provider_id
		WHERE 1 = 1
	`
	var args []any

	if searchTerm != "" {
		search := "%" + searchTerm + "%"
		query += ` AND (d.domain LIKE ? OR COALESCE(p.name, '') LIKE ? OR COALESCE(d.currency, '') LIKE ?)`
		args = append(args, search, search, search)
	}

	query += ` ORDER BY d.domain`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*Domain
	for rows.Next() {
		item, scanErr := scanDomain(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *Repository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM domains`).Scan(&count)
	return count, err
}

func (r *Repository) Update(ctx context.Context, item *Domain) error {
	item.Currency = normalizeCurrency(item.Currency)
	item.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, `
		UPDATE domains
		SET
			domain = ?,
			provider_id = ?,
			due_date = ?,
			currency = ?,
			price = ?,
			notes = ?,
			updated_at = ?
		WHERE id = ?
	`,
		item.Domain,
		dbString(item.ProviderID),
		dbString(item.DueDate),
		item.Currency,
		dbFloat(item.Price),
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

	_, err := r.db.ExecContext(ctx, `DELETE FROM domains WHERE id = ?`, id)
	return err
}

func scanDomain(scanner interface {
	Scan(dest ...any) error
}) (*Domain, error) {
	item := &Domain{}
	var providerID sql.NullString
	var providerName sql.NullString
	var dueDate sql.NullString
	var price sql.NullFloat64
	var notes sql.NullString
	if err := scanner.Scan(
		&item.ID,
		&item.Domain,
		&providerID,
		&providerName,
		&dueDate,
		&item.Currency,
		&price,
		&notes,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}

	item.ProviderID = nullString(providerID)
	item.ProviderName = nullString(providerName)
	item.DueDate = nullString(dueDate)
	item.Price = nullFloat(price)
	item.Notes = nullString(notes)
	return item, nil
}

func nullString(value sql.NullString) *string {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return nil
	}

	return &value.String
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
