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
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	EntryType     string    `json:"entry_type"`
	AccountName   string    `json:"account_name"`
	Category      string    `json:"category"`
	Counterparty  *string   `json:"counterparty"`
	Scope         string    `json:"scope"`
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	OccurredOn    string    `json:"occurred_on"`
	DueOn         *string   `json:"due_on"`
	PaidOn        *string   `json:"paid_on"`
	PaymentMethod *string   `json:"payment_method"`
	Notes         *string   `json:"notes"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
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

func (e *ExpenseEntry) EntryTypeValue() string {
	if e == nil {
		return "expense"
	}

	return normalizeEntryType(e.EntryType)
}

func (e *ExpenseEntry) AccountNameValue() string {
	if e == nil {
		return "cash"
	}

	return defaultText(e.AccountName, "cash")
}

func (e *ExpenseEntry) ScopeValue() string {
	if e == nil {
		return "infrastructure"
	}

	return normalizeScope(e.Scope)
}

func (e *ExpenseEntry) CounterpartyValue() string {
	if e == nil || e.Counterparty == nil {
		return ""
	}

	return *e.Counterparty
}

func (e *ExpenseEntry) DueOnValue() string {
	if e == nil || e.DueOn == nil {
		return ""
	}

	return *e.DueOn
}

func (e *ExpenseEntry) PaidOnValue() string {
	if e == nil || e.PaidOn == nil {
		return ""
	}

	return *e.PaidOn
}

func (e *ExpenseEntry) PaymentMethodValue() string {
	if e == nil || e.PaymentMethod == nil {
		return ""
	}

	return *e.PaymentMethod
}

func (e *ExpenseEntry) IsSpendLike() bool {
	if e == nil {
		return false
	}

	switch normalizeEntryType(e.EntryType) {
	case "income", "transfer", "refund":
		return false
	default:
		return true
	}
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
	entry.EntryType = normalizeEntryType(entry.EntryType)
	entry.AccountName = defaultText(entry.AccountName, "cash")
	entry.Category = defaultText(entry.Category, "manual")
	entry.Scope = normalizeScope(entry.Scope)
	entry.Currency = strings.ToUpper(defaultText(entry.Currency, "MXN"))
	entry.OccurredOn = defaultText(entry.OccurredOn, now.Format(time.DateOnly))
	entry.CreatedAt = now
	entry.UpdatedAt = now

	_, err = r.db.ExecContext(ctx, `
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
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		entry.ID,
		entry.Title,
		entry.EntryType,
		entry.AccountName,
		entry.Category,
		nullableString(entry.Counterparty),
		entry.Scope,
		entry.Amount,
		entry.Currency,
		entry.OccurredOn,
		nullableString(entry.DueOn),
		nullableString(entry.PaidOn),
		nullableString(entry.PaymentMethod),
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
	var (
		counterparty  sql.NullString
		dueOn         sql.NullString
		paidOn        sql.NullString
		paymentMethod sql.NullString
		notes         sql.NullString
	)
	entry := &ExpenseEntry{}
	if err := scanner.Scan(
		&entry.ID,
		&entry.Title,
		&entry.EntryType,
		&entry.AccountName,
		&entry.Category,
		&counterparty,
		&entry.Scope,
		&entry.Amount,
		&entry.Currency,
		&entry.OccurredOn,
		&dueOn,
		&paidOn,
		&paymentMethod,
		&notes,
		&entry.CreatedAt,
		&entry.UpdatedAt,
	); err != nil {
		return nil, err
	}

	entry.EntryType = normalizeEntryType(entry.EntryType)
	entry.AccountName = defaultText(entry.AccountName, "cash")
	entry.Scope = normalizeScope(entry.Scope)
	entry.Counterparty = nullString(counterparty)
	entry.DueOn = nullString(dueOn)
	entry.PaidOn = nullString(paidOn)
	entry.PaymentMethod = nullString(paymentMethod)
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

func normalizeEntryType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "income", "transfer", "refund":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "expense"
	}
}

func normalizeScope(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "household", "office", "personal", "other":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "infrastructure"
	}
}
