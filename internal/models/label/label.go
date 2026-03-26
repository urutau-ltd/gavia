package label

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Label struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Notes     *string   `json:"notes"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (l *Label) NotesValue() string { return stringValue(l.Notes) }

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, item *Label) error {
	newID, err := uuid.NewV7()
	if err != nil {
		return err
	}

	now := time.Now()
	item.ID = newID.String()
	item.CreatedAt = now
	item.UpdatedAt = now

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO labels (id, name, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`,
		item.ID,
		item.Name,
		dbString(item.Notes),
		item.CreatedAt,
		item.UpdatedAt,
	)
	return err
}

func (r *Repository) GetByID(ctx context.Context, id string) (*Label, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, fmt.Errorf("invalid uuid format: %w", err)
	}

	item := &Label{}
	var notes sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, notes, created_at, updated_at
		FROM labels
		WHERE id = ?
	`, id).Scan(
		&item.ID,
		&item.Name,
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

	item.Notes = nullString(notes)
	return item, nil
}

func (r *Repository) GetAll(ctx context.Context, searchTerm string, limit int) ([]*Label, error) {
	query := `
		SELECT id, name, notes, created_at, updated_at
		FROM labels
		WHERE 1 = 1
	`
	var args []any

	if searchTerm != "" {
		search := "%" + searchTerm + "%"
		query += ` AND name LIKE ?`
		args = append(args, search)
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

	var items []*Label
	for rows.Next() {
		item, scanErr := scanLabel(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *Repository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM labels`).Scan(&count)
	return count, err
}

func (r *Repository) Update(ctx context.Context, item *Label) error {
	item.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, `
		UPDATE labels
		SET name = ?, notes = ?, updated_at = ?
		WHERE id = ?
	`,
		item.Name,
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

	_, err := r.db.ExecContext(ctx, `DELETE FROM labels WHERE id = ?`, id)
	return err
}

func scanLabel(scanner interface{ Scan(dest ...any) error }) (*Label, error) {
	item := &Label{}
	var notes sql.NullString
	if err := scanner.Scan(
		&item.ID,
		&item.Name,
		&notes,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}

	item.Notes = nullString(notes)
	return item, nil
}

func nullString(value sql.NullString) *string {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return nil
	}

	return &value.String
}

func dbString(value *string) any {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}

	return strings.TrimSpace(*value)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}
