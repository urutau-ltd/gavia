package operatingsystem

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// OperatingSystem models one O.S record from the operating_systems table.
// This struct is shared by repository methods and UI handlers to keep a single
// source of truth for operating_system fields.
type OperatingSystem struct {
	Id        string    `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	Notes     *string   `db:"notes" json:"notes"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

func (os *OperatingSystem) NotesValue() string {
	return stringValue(os.Notes)
}

// OperatingSystemRepository contains all SQL operations for operating systems
// Keeping persistence here prevents HTTP handlers from depending on SQL
// details.
type OperatingSystemRepository struct {
	db *sql.DB
}

// NewOperatingSystemRepository wires a database handle into an operating
// system repository. Handlers call this once during bootstrap and reuse it
// per request.
func NewOperatingSystemRepository(db *sql.DB) *OperatingSystemRepository {
	return &OperatingSystemRepository{db: db}
}

// Create inserts a new operating system and assigns identity/timestamps in one place.
// This guarantees all callers follow the same ID and time conventions.
func (r *OperatingSystemRepository) Create(ctx context.Context, o *OperatingSystem) error {
	newID, err := uuid.NewV7()
	if err != nil {
		return err
	}

	o.Id = newID.String()
	o.CreatedAt = time.Now()
	o.UpdatedAt = o.CreatedAt

	_, err = r.db.ExecContext(ctx,
		"INSERT INTO operating_systems (id, name, notes) VALUES (?, ?, ?)",
		o.Id,
		o.Name,
		dbString(o.Notes),
	)

	return err
}

// GetByID returns one operating system by UUID.
// It returns nil,nil when the record does not exist so handlers can map that
// case to 404 responses without treating it as a storage failure.
func (r *OperatingSystemRepository) GetByID(ctx context.Context, id string) (*OperatingSystem, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, fmt.Errorf("invalid uuid format: %w", err)
	}

	var notes sql.NullString
	p := &OperatingSystem{}
	err := r.db.QueryRowContext(ctx,
		"SELECT id, name, notes, created_at, updated_at FROM operating_systems WHERE id = ?",
		id).Scan(
		&p.Id,
		&p.Name,
		&notes,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	p.Notes = nullableString(notes)
	return p, nil
}

// GetAll returns operating systems filtered by optional search term and limit.
// The method owns query composition so pagination/filter logic is not duplicated
// across handlers.
func (r *OperatingSystemRepository) GetAll(ctx context.Context, searchTerm string, limit int) ([]*OperatingSystem, error) {
	query := `SELECT id, name, notes, created_at, updated_at FROM operating_systems WHERE 1=1`
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
	var operatingSystems []*OperatingSystem
	for rows.Next() {
		var notes sql.NullString
		p := &OperatingSystem{}
		err := rows.Scan(
			&p.Id,
			&p.Name,
			&notes,
			&p.CreatedAt,
			&p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning operating system: %w", err)
		}

		p.Notes = nullableString(notes)
		operatingSystems = append(operatingSystems, p)
	}
	return operatingSystems, rows.Err()
}

// Count returns the total number of operating systems in storage.
// Dashboards and overview pages use it to show inventory size without loading rows.
func (r *OperatingSystemRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM operating_systems").Scan(&count)
	return count, err
}

// Update persists mutable operating_system fields and refreshes UpdatedAt.
// Timestamp assignment lives here so every update path behaves consistently.
func (r *OperatingSystemRepository) Update(ctx context.Context, p *OperatingSystem) error {
	p.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx,
		"UPDATE operating_systems SET name = ?, notes = ?, updated_at = ? WHERE id = ?",
		p.Name,
		dbString(p.Notes),
		p.UpdatedAt,
		p.Id)
	return err
}

// Delete removes one operating system by UUID.
// UUID validation is done here to fail fast before issuing SQL.
func (r *OperatingSystemRepository) Delete(ctx context.Context, id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("invalid uuid format: %w", err)
	}

	_, err := r.db.ExecContext(ctx, "DELETE FROM operating_systems WHERE id = ?", id)
	return err
}

func nullableString(value sql.NullString) *string {
	if !value.Valid || value.String == "" {
		return nil
	}

	return new(value.String)
}

func dbString(value *string) any {
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
