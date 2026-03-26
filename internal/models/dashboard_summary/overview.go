package dashboardsummary

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

type DueItem struct {
	SourceType string   `json:"source_type"`
	Label      string   `json:"label"`
	DueDate    string   `json:"due_date"`
	Currency   string   `json:"currency"`
	Price      *float64 `json:"price"`
}

func (d DueItem) PriceDisplay() string {
	if d.Price == nil {
		return ""
	}

	return fmt.Sprintf("%.2f", *d.Price)
}

type Repository struct {
	db *sql.DB
}

type AmountBucket struct {
	Label  string  `json:"label"`
	Amount float64 `json:"amount"`
	Count  int     `json:"count"`
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) UpcomingDue(ctx context.Context, limit int) ([]*DueItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			source_type,
			label,
			due_date,
			currency,
			price
		FROM (
			SELECT 'Domain' AS source_type, domain AS label, due_date, currency, price FROM domains
			UNION ALL
			SELECT 'Hosting' AS source_type, name AS label, due_date, currency, price FROM hostings
			UNION ALL
			SELECT 'Server' AS source_type, hostname AS label, due_date, currency, price FROM servers
			UNION ALL
			SELECT 'Subscription' AS source_type, name AS label, due_date, currency, price FROM subscriptions
		)
		WHERE due_date IS NOT NULL
		  AND date(due_date) >= date('now')
		ORDER BY date(due_date) ASC, label ASC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*DueItem
	for rows.Next() {
		var price sql.NullFloat64
		item := &DueItem{}
		if err := rows.Scan(&item.SourceType, &item.Label, &item.DueDate, &item.Currency, &price); err != nil {
			return nil, err
		}

		if price.Valid {
			item.Price = &price.Float64
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func AggregateByLabel[T any](
	items []T,
	labelOf func(T) string,
	amountOf func(T) float64,
) []AmountBucket {
	grouped := make(map[string]*AmountBucket)
	for _, item := range items {
		label := strings.TrimSpace(labelOf(item))
		if label == "" {
			label = "Uncategorized"
		}

		bucket, ok := grouped[label]
		if !ok {
			bucket = &AmountBucket{Label: label}
			grouped[label] = bucket
		}

		bucket.Count++
		bucket.Amount += amountOf(item)
	}

	buckets := make([]AmountBucket, 0, len(grouped))
	for _, bucket := range grouped {
		buckets = append(buckets, *bucket)
	}

	sort.Slice(buckets, func(i, j int) bool {
		if buckets[i].Amount == buckets[j].Amount {
			return buckets[i].Label < buckets[j].Label
		}
		return buckets[i].Amount > buckets[j].Amount
	})

	return buckets
}
