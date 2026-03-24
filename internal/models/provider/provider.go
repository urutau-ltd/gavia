package provider

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Provider models one provider record from the providers table.
// This struct is shared by repository methods and UI handlers to keep a single
// source of truth for provider fields.
type Provider struct {
	Id        string    `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	Website   string    `db:"website" json:"website"`
	Notes     string    `db:"notes" json:"notes"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// ProviderRepository contains all SQL operations for providers.
// Keeping persistence here prevents HTTP handlers from depending on SQL details.
type ProviderRepository struct {
	db *sql.DB
}

// NewProviderRepository wires a database handle into a provider repository.
// Handlers call this once during bootstrap and reuse it per request.
func NewProviderRepository(db *sql.DB) *ProviderRepository {
	return &ProviderRepository{db: db}
}

// Create inserts a new provider and assigns identity/timestamps in one place.
// This guarantees all callers follow the same ID and time conventions.
func (r *ProviderRepository) Create(ctx context.Context, p *Provider) error {
	newID, err := uuid.NewV7()
	if err != nil {
		return err
	}

	p.Id = newID.String()
	p.CreatedAt = time.Now()
	p.UpdatedAt = p.CreatedAt

	_, err = r.db.ExecContext(ctx,
		"INSERT INTO providers (id, name, website, notes) VALUES (?, ?, ?, ?)",
		p.Id,
		p.Name,
		p.Website,
		p.Notes,
	)

	return err
}

// GetByID returns one provider by UUID.
// It returns nil,nil when the record does not exist so handlers can map that
// case to 404 responses without treating it as a storage failure.
func (r *ProviderRepository) GetByID(ctx context.Context, id string) (*Provider, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, fmt.Errorf("invalid uuid format: %w", err)
	}

	p := &Provider{}
	err := r.db.QueryRowContext(ctx,
		"SELECT id, name, notes, website, created_at, updated_at FROM providers WHERE id = ?",
		id).Scan(
		&p.Id,
		&p.Name,
		&p.Notes,
		&p.Website,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}

// GetAll returns providers filtered by optional search term and limit.
// The method owns query composition so pagination/filter logic is not duplicated
// across handlers.
func (r *ProviderRepository) GetAll(ctx context.Context, searchTerm string, limit int) ([]*Provider, error) {
	query := `SELECT id, name, notes, website, created_at, updated_at FROM providers WHERE 1=1`
	var args []any

	if searchTerm != "" {
		query += " AND name LIKE ?"
		args = append(args, "%"+searchTerm+"%")
	}

	query += " ORDER BY name"

	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var providers []*Provider
	for rows.Next() {
		p := &Provider{}
		err := rows.Scan(
			&p.Id,
			&p.Name,
			&p.Notes,
			&p.Website,
			&p.CreatedAt,
			&p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning provider: %w", err)
		}

		providers = append(providers, p)
	}
	return providers, rows.Err()
}

// Update persists mutable provider fields and refreshes UpdatedAt.
// Timestamp assignment lives here so every update path behaves consistently.
func (r *ProviderRepository) Update(ctx context.Context, p *Provider) error {
	p.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx,
		"UPDATE providers SET name = ?, website = ?, notes = ?, updated_at = ? WHERE id = ?",
		p.Name,
		p.Website,
		p.Notes,
		p.UpdatedAt,
		p.Id)
	return err
}

// Delete removes one provider by UUID.
// UUID validation is done here to fail fast before issuing SQL.
func (r *ProviderRepository) Delete(ctx context.Context, id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("invalid uuid format: %w", err)
	}

	_, err := r.db.ExecContext(ctx, "DELETE FROM providers WHERE id = ?", id)
	return err
}
