package location

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Location models one location record from the locations table.
// This struct is the contract shared between repository and UI layers.
type Location struct {
	Id        string    `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	City      string    `db:"city" json:"city"`
	Country   string    `db:"country" json:"country"`
	Notes     string    `db:"notes" json:"notes"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// LocationRepository centralizes SQL access for location CRUD operations.
// Keeping queries here lets handlers focus on HTTP/HTMX concerns only.
type LocationRepository struct {
	db *sql.DB
}

// NewLocationRepository creates a repository bound to the app database handle.
func NewLocationRepository(db *sql.DB) *LocationRepository {
	return &LocationRepository{db: db}
}

// Create inserts a new location and assigns UUID/timestamps.
func (r *LocationRepository) Create(ctx context.Context, l *Location) error {
	newID, err := uuid.NewV7()
	if err != nil {
		return err
	}

	l.Id = newID.String()
	l.CreatedAt = time.Now()
	l.UpdatedAt = l.CreatedAt

	_, err = r.db.ExecContext(ctx,
		"INSERT INTO locations (id, name, city, country, notes, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		l.Id,
		l.Name,
		l.City,
		l.Country,
		l.Notes,
		l.CreatedAt,
		l.UpdatedAt,
	)

	return err
}

// GetByID returns one location by UUID.
// It returns nil,nil when the row is absent so callers can render 404 cleanly.
func (r *LocationRepository) GetByID(ctx context.Context, id string) (*Location, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, fmt.Errorf("invalid uuid format: %w", err)
	}

	l := &Location{}
	err := r.db.QueryRowContext(ctx,
		"SELECT id, name, city, country, notes, created_at, updated_at FROM locations WHERE id = ?",
		id).Scan(
		&l.Id,
		&l.Name,
		&l.City,
		&l.Country,
		&l.Notes,
		&l.CreatedAt,
		&l.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return l, nil
}

// GetAll returns locations filtered by search and capped by limit.
// Search spans name/city/country so a single text box can drive the table.
func (r *LocationRepository) GetAll(ctx context.Context, searchTerm string, limit int) ([]*Location, error) {
	query := `SELECT id, name, city, country, notes, created_at, updated_at FROM locations WHERE 1=1`
	var args []any

	if searchTerm != "" {
		query += " AND (name LIKE ? OR city LIKE ? OR country LIKE ?)"
		search := "%" + searchTerm + "%"
		args = append(args, search, search, search)
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

	var locations []*Location
	for rows.Next() {
		l := &Location{}
		err := rows.Scan(
			&l.Id,
			&l.Name,
			&l.City,
			&l.Country,
			&l.Notes,
			&l.CreatedAt,
			&l.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning location: %w", err)
		}
		locations = append(locations, l)
	}

	return locations, rows.Err()
}

// Update writes mutable fields and refreshes UpdatedAt.
func (r *LocationRepository) Update(ctx context.Context, l *Location) error {
	l.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx,
		"UPDATE locations SET name = ?, city = ?, country = ?, notes = ?, updated_at = ? WHERE id = ?",
		l.Name,
		l.City,
		l.Country,
		l.Notes,
		l.UpdatedAt,
		l.Id,
	)
	return err
}

// Delete removes one location by UUID after validating its format.
func (r *LocationRepository) Delete(ctx context.Context, id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("invalid uuid format: %w", err)
	}

	_, err := r.db.ExecContext(ctx, "DELETE FROM locations WHERE id = ?", id)
	return err
}
