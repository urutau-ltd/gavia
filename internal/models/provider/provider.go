package provider

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Provider struct {
	Id        int       `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	Website   string    `db:"website" json:"website"`
	Notes     string    `db:"notes" json:"notes"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type ProviderRepository struct {
	db *sql.DB
}

func NewProviderRepository(db *sql.DB) *ProviderRepository {
	return &ProviderRepository{db: db}
}

func (r *ProviderRepository) Create(ctx context.Context, p *Provider) error {
	result, err := r.db.ExecContext(ctx,
		"INSERT INTO providers (name, website, notes) VALUES (?, ?, ?)",
		p.Name, p.Website, p.Notes)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	p.Id = int(id)
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	return nil
}

func (r *ProviderRepository) GetByID(ctx context.Context, id int) (*Provider, error) {
	p := &Provider{}
	err := r.db.QueryRowContext(ctx,
		"SELECT id, name, notes, website, created_at, updated_at FROM providers WHERE id = ?",
		id).Scan(
		&p.Id,
		&p.Name,
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

func (r *ProviderRepository) GetAll(ctx context.Context) ([]*Provider, error) {
	query := `SELECT id, name, notes, website, created_at, updated_at FROM providers ORDER BY name`

	rows, err := r.db.QueryContext(ctx, query)
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

func (r *ProviderRepository) Delete(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM providers WHERE id = ?", id)
	return err
}
