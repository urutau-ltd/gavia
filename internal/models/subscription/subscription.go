package subscription

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Subscription struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Type          string    `json:"type"`
	Price         *float64  `json:"price"`
	Currency      string    `json:"currency"`
	DueDate       *string   `json:"due_date"`
	SinceDate     *string   `json:"since_date"`
	RenewalPeriod *string   `json:"renewal_period"`
	Notes         *string   `json:"notes"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (s *Subscription) PriceDisplay() string {
	if s == nil || s.Price == nil {
		return ""
	}

	return fmt.Sprintf("%.2f", *s.Price)
}

func (s *Subscription) CurrencyValue(defaultCurrency string) string {
	value := strings.ToUpper(strings.TrimSpace(s.Currency))
	if value != "" {
		return value
	}

	value = strings.ToUpper(strings.TrimSpace(defaultCurrency))
	if value != "" {
		return value
	}

	return "MXN"
}

func (s *Subscription) DueDateValue() string {
	return stringValue(s.DueDate)
}

func (s *Subscription) SinceDateValue() string {
	return stringValue(s.SinceDate)
}

func (s *Subscription) RenewalPeriodValue() string {
	return stringValue(s.RenewalPeriod)
}

func (s *Subscription) NotesValue() string {
	return stringValue(s.Notes)
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, item *Subscription) error {
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
		INSERT INTO subscriptions (
			id,
			name,
			type,
			price,
			currency,
			due_date,
			since_date,
			renewal_period,
			notes,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		item.ID,
		item.Name,
		item.Type,
		dbFloat(item.Price),
		item.Currency,
		dbString(item.DueDate),
		dbString(item.SinceDate),
		dbString(item.RenewalPeriod),
		dbString(item.Notes),
		item.CreatedAt,
		item.UpdatedAt,
	)
	return err
}

func (r *Repository) GetByID(ctx context.Context, id string) (*Subscription, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, fmt.Errorf("invalid uuid format: %w", err)
	}

	item := &Subscription{}
	var price sql.NullFloat64
	var dueDate sql.NullString
	var sinceDate sql.NullString
	var renewalPeriod sql.NullString
	var notes sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT
			id,
			name,
			type,
			price,
			currency,
			due_date,
			since_date,
			renewal_period,
			notes,
			created_at,
			updated_at
		FROM subscriptions
		WHERE id = ?
	`, id).Scan(
		&item.ID,
		&item.Name,
		&item.Type,
		&price,
		&item.Currency,
		&dueDate,
		&sinceDate,
		&renewalPeriod,
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

	item.Price = nullFloat(price)
	item.DueDate = nullString(dueDate)
	item.SinceDate = nullString(sinceDate)
	item.RenewalPeriod = nullString(renewalPeriod)
	item.Notes = nullString(notes)
	return item, nil
}

func (r *Repository) GetAll(ctx context.Context, searchTerm string, limit int) ([]*Subscription, error) {
	query := `
		SELECT
			id,
			name,
			type,
			price,
			currency,
			due_date,
			since_date,
			renewal_period,
			notes,
			created_at,
			updated_at
		FROM subscriptions
		WHERE 1 = 1
	`
	var args []any

	if searchTerm != "" {
		search := "%" + searchTerm + "%"
		query += ` AND (
			name LIKE ?
			OR type LIKE ?
			OR COALESCE(renewal_period, '') LIKE ?
			OR COALESCE(currency, '') LIKE ?
		)`
		args = append(args, search, search, search, search)
	}

	query += ` ORDER BY name`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*Subscription
	for rows.Next() {
		item, scanErr := scanSubscription(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *Repository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM subscriptions`).Scan(&count)
	return count, err
}

func (r *Repository) Update(ctx context.Context, item *Subscription) error {
	item.Currency = normalizeCurrency(item.Currency)
	item.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, `
		UPDATE subscriptions
		SET
			name = ?,
			type = ?,
			price = ?,
			currency = ?,
			due_date = ?,
			since_date = ?,
			renewal_period = ?,
			notes = ?,
			updated_at = ?
		WHERE id = ?
	`,
		item.Name,
		item.Type,
		dbFloat(item.Price),
		item.Currency,
		dbString(item.DueDate),
		dbString(item.SinceDate),
		dbString(item.RenewalPeriod),
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

	_, err := r.db.ExecContext(ctx, `DELETE FROM subscriptions WHERE id = ?`, id)
	return err
}

func scanSubscription(scanner interface {
	Scan(dest ...any) error
}) (*Subscription, error) {
	item := &Subscription{}
	var price sql.NullFloat64
	var dueDate sql.NullString
	var sinceDate sql.NullString
	var renewalPeriod sql.NullString
	var notes sql.NullString
	if err := scanner.Scan(
		&item.ID,
		&item.Name,
		&item.Type,
		&price,
		&item.Currency,
		&dueDate,
		&sinceDate,
		&renewalPeriod,
		&notes,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}

	item.Price = nullFloat(price)
	item.DueDate = nullString(dueDate)
	item.SinceDate = nullString(sinceDate)
	item.RenewalPeriod = nullString(renewalPeriod)
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
