package exchangerate

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Sample struct {
	ID            string    `json:"id"`
	BaseCurrency  string    `json:"base_currency"`
	QuoteCurrency string    `json:"quote_currency"`
	Rate          float64   `json:"rate"`
	Source        string    `json:"source"`
	ObservedAt    time.Time `json:"observed_at"`
	CreatedAt     time.Time `json:"created_at"`
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, sample *Sample) error {
	newID, err := uuid.NewV7()
	if err != nil {
		return err
	}

	if sample.ObservedAt.IsZero() {
		sample.ObservedAt = time.Now().UTC()
	}

	sample.ID = newID.String()
	sample.BaseCurrency = strings.ToUpper(strings.TrimSpace(sample.BaseCurrency))
	sample.QuoteCurrency = strings.ToUpper(strings.TrimSpace(sample.QuoteCurrency))
	sample.Source = strings.TrimSpace(sample.Source)

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO exchange_rate_samples (
			id,
			base_currency,
			quote_currency,
			rate,
			source,
			observed_at
		) VALUES (?, ?, ?, ?, ?, ?)
	`,
		sample.ID,
		sample.BaseCurrency,
		sample.QuoteCurrency,
		sample.Rate,
		sample.Source,
		sample.ObservedAt.UTC(),
	)
	return err
}

func (r *Repository) GetLatest(ctx context.Context, baseCurrency, quoteCurrency string) (*Sample, error) {
	baseCurrency = strings.ToUpper(strings.TrimSpace(baseCurrency))
	quoteCurrency = strings.ToUpper(strings.TrimSpace(quoteCurrency))

	sample := &Sample{}
	err := r.db.QueryRowContext(ctx, `
		SELECT
			id,
			base_currency,
			quote_currency,
			rate,
			source,
			observed_at,
			created_at
		FROM exchange_rate_samples
		WHERE base_currency = ?
		  AND quote_currency = ?
		ORDER BY observed_at DESC
		LIMIT 1
	`, baseCurrency, quoteCurrency).Scan(
		&sample.ID,
		&sample.BaseCurrency,
		&sample.QuoteCurrency,
		&sample.Rate,
		&sample.Source,
		&sample.ObservedAt,
		&sample.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return sample, nil
}

func (r *Repository) GetRecent(ctx context.Context, baseCurrency, quoteCurrency string, limit int) ([]*Sample, error) {
	baseCurrency = strings.ToUpper(strings.TrimSpace(baseCurrency))
	quoteCurrency = strings.ToUpper(strings.TrimSpace(quoteCurrency))

	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id,
			base_currency,
			quote_currency,
			rate,
			source,
			observed_at,
			created_at
		FROM exchange_rate_samples
		WHERE base_currency = ?
		  AND quote_currency = ?
		ORDER BY observed_at DESC
		LIMIT ?
	`, baseCurrency, quoteCurrency, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var samples []*Sample
	for rows.Next() {
		item := &Sample{}
		if err := rows.Scan(
			&item.ID,
			&item.BaseCurrency,
			&item.QuoteCurrency,
			&item.Rate,
			&item.Source,
			&item.ObservedAt,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		samples = append(samples, item)
	}

	return samples, rows.Err()
}
