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
	City      *string   `db:"city" json:"city"`
	Country   *string   `db:"country" json:"country"`
	Latitude  *float64  `db:"latitude" json:"latitude"`
	Longitude *float64  `db:"longitude" json:"longitude"`
	Notes     *string   `db:"notes" json:"notes"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

func (l *Location) CityValue() string {
	return stringValue(l.City)
}

func (l *Location) CountryValue() string {
	return stringValue(l.Country)
}

func (l *Location) NotesValue() string {
	return stringValue(l.Notes)
}

func (l *Location) LatitudeValue() string {
	if l == nil || l.Latitude == nil {
		return ""
	}

	return fmt.Sprintf("%.6f", *l.Latitude)
}

func (l *Location) LongitudeValue() string {
	if l == nil || l.Longitude == nil {
		return ""
	}

	return fmt.Sprintf("%.6f", *l.Longitude)
}

func (l *Location) HasCoordinates() bool {
	return l != nil && l.Latitude != nil && l.Longitude != nil
}

func (l *Location) MapLinkURL() string {
	if !l.HasCoordinates() {
		return ""
	}

	return fmt.Sprintf(
		"https://www.openstreetmap.org/?mlat=%.6f&mlon=%.6f#map=13/%.6f/%.6f",
		*l.Latitude,
		*l.Longitude,
		*l.Latitude,
		*l.Longitude,
	)
}

func (l *Location) MapEmbedURL() string {
	if !l.HasCoordinates() {
		return ""
	}

	const delta = 0.02
	return fmt.Sprintf(
		"https://www.openstreetmap.org/export/embed.html?bbox=%.6f,%.6f,%.6f,%.6f&layer=mapnik&marker=%.6f,%.6f",
		*l.Longitude-delta,
		*l.Latitude-delta,
		*l.Longitude+delta,
		*l.Latitude+delta,
		*l.Latitude,
		*l.Longitude,
	)
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
		"INSERT INTO locations (id, name, city, country, latitude, longitude, notes, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		l.Id,
		l.Name,
		dbString(l.City),
		dbString(l.Country),
		dbFloat(l.Latitude),
		dbFloat(l.Longitude),
		dbString(l.Notes),
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

	var city sql.NullString
	var country sql.NullString
	var latitude sql.NullFloat64
	var longitude sql.NullFloat64
	var notes sql.NullString
	l := &Location{}
	err := r.db.QueryRowContext(ctx,
		"SELECT id, name, city, country, latitude, longitude, notes, created_at, updated_at FROM locations WHERE id = ?",
		id).Scan(
		&l.Id,
		&l.Name,
		&city,
		&country,
		&latitude,
		&longitude,
		&notes,
		&l.CreatedAt,
		&l.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	l.City = nullableString(city)
	l.Country = nullableString(country)
	l.Latitude = nullableFloat(latitude)
	l.Longitude = nullableFloat(longitude)
	l.Notes = nullableString(notes)
	return l, nil
}

// GetAll returns locations filtered by search and capped by limit.
// Search spans name/city/country so a single text box can drive the table.
func (r *LocationRepository) GetAll(ctx context.Context, searchTerm string, limit int) ([]*Location, error) {
	query := `SELECT id, name, city, country, latitude, longitude, notes, created_at, updated_at FROM locations WHERE 1=1`
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
		var city sql.NullString
		var country sql.NullString
		var latitude sql.NullFloat64
		var longitude sql.NullFloat64
		var notes sql.NullString
		l := &Location{}
		err := rows.Scan(
			&l.Id,
			&l.Name,
			&city,
			&country,
			&latitude,
			&longitude,
			&notes,
			&l.CreatedAt,
			&l.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning location: %w", err)
		}

		l.City = nullableString(city)
		l.Country = nullableString(country)
		l.Latitude = nullableFloat(latitude)
		l.Longitude = nullableFloat(longitude)
		l.Notes = nullableString(notes)
		locations = append(locations, l)
	}

	return locations, rows.Err()
}

// Count returns the total number of stored locations.
// Overview pages use it to summarize inventory without fetching full records.
func (r *LocationRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM locations").Scan(&count)
	return count, err
}

// Update writes mutable fields and refreshes UpdatedAt.
func (r *LocationRepository) Update(ctx context.Context, l *Location) error {
	l.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx,
		"UPDATE locations SET name = ?, city = ?, country = ?, latitude = ?, longitude = ?, notes = ?, updated_at = ? WHERE id = ?",
		l.Name,
		dbString(l.City),
		dbString(l.Country),
		dbFloat(l.Latitude),
		dbFloat(l.Longitude),
		dbString(l.Notes),
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

func nullableFloat(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}

	result := value.Float64
	return &result
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
