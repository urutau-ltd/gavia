package expenseentry

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ExpenseEntry struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Category   string    `json:"category"`
	Amount     float64   `json:"amount"`
	Currency   string    `json:"currency"`
	OccurredOn string    `json:"occurred_on"`
	Notes      *string   `json:"notes"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (e *ExpenseEntry) AmountDisplay() string {
	return fmt.Sprintf("%.2f", e.Amount)
}

func (e *ExpenseEntry) NotesValue() string {
	if e == nil || e.Notes == nil {
		return ""
	}

	return *e.Notes
}

type ExpenseEntryRepository struct {
	db *sql.DB
}

func NewExpenseEntryRepository(db *sql.DB) *ExpenseEntryRepository {
	return &ExpenseEntryRepository{db: db}
}

func (r *ExpenseEntryRepository) GetRecent(ctx context.Context, limit int) ([]*ExpenseEntry, error) {
	query := `
		SELECT
			id,
			title,
			category,
			amount,
			currency,
			occurred_on,
			notes,
			created_at,
			updated_at
		FROM expense_entries
		ORDER BY occurred_on DESC, created_at DESC
	`

	var rows *sql.Rows
	var err error
	if limit > 0 {
		rows, err = r.db.QueryContext(ctx, query+` LIMIT ?`, limit)
	} else {
		rows, err = r.db.QueryContext(ctx, query)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*ExpenseEntry
	for rows.Next() {
		entry, scanErr := scanExpenseEntry(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

func (r *ExpenseEntryRepository) GetAll(ctx context.Context) ([]*ExpenseEntry, error) {
	return r.GetRecent(ctx, 0)
}

func (r *ExpenseEntryRepository) Create(ctx context.Context, entry *ExpenseEntry) error {
	newID, err := uuid.NewV7()
	if err != nil {
		return err
	}

	now := time.Now()
	entry.ID = newID.String()
	entry.Category = defaultText(entry.Category, "manual")
	entry.Currency = strings.ToUpper(defaultText(entry.Currency, "MXN"))
	entry.OccurredOn = defaultText(entry.OccurredOn, now.Format(time.DateOnly))
	entry.CreatedAt = now
	entry.UpdatedAt = now

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO expense_entries (
			id,
			title,
			category,
			amount,
			currency,
			occurred_on,
			notes,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		entry.ID,
		entry.Title,
		entry.Category,
		entry.Amount,
		entry.Currency,
		entry.OccurredOn,
		nullableString(entry.Notes),
		entry.CreatedAt,
		entry.UpdatedAt,
	)
	return err
}

func (r *ExpenseEntryRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM expense_entries WHERE id = ?`, strings.TrimSpace(id))
	return err
}

func scanExpenseEntry(scanner interface {
	Scan(dest ...any) error
}) (*ExpenseEntry, error) {
	var notes sql.NullString
	entry := &ExpenseEntry{}
	if err := scanner.Scan(
		&entry.ID,
		&entry.Title,
		&entry.Category,
		&entry.Amount,
		&entry.Currency,
		&entry.OccurredOn,
		&notes,
		&entry.CreatedAt,
		&entry.UpdatedAt,
	); err != nil {
		return nil, err
	}

	entry.Notes = nullString(notes)
	return entry, nil
}

func nullString(value sql.NullString) *string {
	if !value.Valid || value.String == "" {
		return nil
	}

	return &value.String
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}

	return *value
}

func defaultText(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}

	return value
}
