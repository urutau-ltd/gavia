package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Server struct {
	ID           string    `json:"id"`
	Hostname     string    `json:"hostname"`
	Type         string    `json:"type"`
	OSID         *string   `json:"os_id"`
	OSName       *string   `json:"os_name"`
	CPUCores     *int      `json:"cpu_cores"`
	MemoryGB     *int      `json:"memory_gb"`
	DiskGB       *int      `json:"disk_gb"`
	LocationID   *string   `json:"location_id"`
	LocationName *string   `json:"location_name"`
	ProviderID   *string   `json:"provider_id"`
	ProviderName *string   `json:"provider_name"`
	DueDate      *string   `json:"due_date"`
	Price        *float64  `json:"price"`
	Currency     string    `json:"currency"`
	SinceDate    *string   `json:"since_date"`
	Notes        *string   `json:"notes"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (s *Server) OSIDValue() string {
	return stringValue(s.OSID)
}

func (s *Server) OSNameValue() string {
	return stringValue(s.OSName)
}

func (s *Server) LocationIDValue() string {
	return stringValue(s.LocationID)
}

func (s *Server) LocationNameValue() string {
	return stringValue(s.LocationName)
}

func (s *Server) ProviderIDValue() string {
	return stringValue(s.ProviderID)
}

func (s *Server) ProviderNameValue() string {
	return stringValue(s.ProviderName)
}

func (s *Server) CPUCoresValue() string {
	if s == nil || s.CPUCores == nil {
		return ""
	}

	return strconv.Itoa(*s.CPUCores)
}

func (s *Server) MemoryGBValue() string {
	if s == nil || s.MemoryGB == nil {
		return ""
	}

	return strconv.Itoa(*s.MemoryGB)
}

func (s *Server) DiskGBValue() string {
	if s == nil || s.DiskGB == nil {
		return ""
	}

	return strconv.Itoa(*s.DiskGB)
}

func (s *Server) DueDateValue() string {
	return stringValue(s.DueDate)
}

func (s *Server) SinceDateValue() string {
	return stringValue(s.SinceDate)
}

func (s *Server) CurrencyValue(defaultCurrency string) string {
	value := strings.ToUpper(strings.TrimSpace(s.Currency))
	if value != "" {
		return value
	}

	value = strings.ToUpper(strings.TrimSpace(defaultCurrency))
	if value != "" {
		return value
	}

	return "MXN"
}

func (s *Server) PriceDisplay() string {
	if s == nil || s.Price == nil {
		return ""
	}

	return fmt.Sprintf("%.2f", *s.Price)
}

func (s *Server) NotesValue() string {
	return stringValue(s.Notes)
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, item *Server) error {
	newID, err := uuid.NewV7()
	if err != nil {
		return err
	}

	now := time.Now()
	item.ID = newID.String()
	item.Currency = normalizeCurrency(item.Currency)
	item.CreatedAt = now
	item.UpdatedAt = now

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO servers (
			id,
			hostname,
			type,
			os_id,
			cpu_cores,
			memory_gb,
			disk_gb,
			location_id,
			provider_id,
			due_date,
			price,
			currency,
			since_date,
			notes,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		item.ID,
		item.Hostname,
		item.Type,
		dbString(item.OSID),
		dbInt(item.CPUCores),
		dbInt(item.MemoryGB),
		dbInt(item.DiskGB),
		dbString(item.LocationID),
		dbString(item.ProviderID),
		dbString(item.DueDate),
		dbFloat(item.Price),
		item.Currency,
		dbString(item.SinceDate),
		dbString(item.Notes),
		item.CreatedAt,
		item.UpdatedAt,
	)
	return err
}

func (r *Repository) GetByID(ctx context.Context, id string) (*Server, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, fmt.Errorf("invalid uuid format: %w", err)
	}

	item := &Server{}
	var osID sql.NullString
	var osName sql.NullString
	var cpuCores sql.NullInt64
	var memoryGB sql.NullInt64
	var diskGB sql.NullInt64
	var locationID sql.NullString
	var locationName sql.NullString
	var providerID sql.NullString
	var providerName sql.NullString
	var dueDate sql.NullString
	var price sql.NullFloat64
	var sinceDate sql.NullString
	var notes sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT
			s.id,
			s.hostname,
			s.type,
			s.os_id,
			o.name,
			s.cpu_cores,
			s.memory_gb,
			s.disk_gb,
			s.location_id,
			l.name,
			s.provider_id,
			p.name,
			s.due_date,
			s.price,
			s.currency,
			s.since_date,
			s.notes,
			s.created_at,
			s.updated_at
		FROM servers s
		LEFT JOIN operating_systems o ON o.id = s.os_id
		LEFT JOIN locations l ON l.id = s.location_id
		LEFT JOIN providers p ON p.id = s.provider_id
		WHERE s.id = ?
	`, id).Scan(
		&item.ID,
		&item.Hostname,
		&item.Type,
		&osID,
		&osName,
		&cpuCores,
		&memoryGB,
		&diskGB,
		&locationID,
		&locationName,
		&providerID,
		&providerName,
		&dueDate,
		&price,
		&item.Currency,
		&sinceDate,
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

	item.OSID = nullString(osID)
	item.OSName = nullString(osName)
	item.CPUCores = nullInt(cpuCores)
	item.MemoryGB = nullInt(memoryGB)
	item.DiskGB = nullInt(diskGB)
	item.LocationID = nullString(locationID)
	item.LocationName = nullString(locationName)
	item.ProviderID = nullString(providerID)
	item.ProviderName = nullString(providerName)
	item.DueDate = nullString(dueDate)
	item.Price = nullFloat(price)
	item.SinceDate = nullString(sinceDate)
	item.Notes = nullString(notes)
	return item, nil
}

func (r *Repository) GetAll(ctx context.Context, searchTerm string, limit int) ([]*Server, error) {
	query := `
		SELECT
			s.id,
			s.hostname,
			s.type,
			s.os_id,
			o.name,
			s.cpu_cores,
			s.memory_gb,
			s.disk_gb,
			s.location_id,
			l.name,
			s.provider_id,
			p.name,
			s.due_date,
			s.price,
			s.currency,
			s.since_date,
			s.notes,
			s.created_at,
			s.updated_at
		FROM servers s
		LEFT JOIN operating_systems o ON o.id = s.os_id
		LEFT JOIN locations l ON l.id = s.location_id
		LEFT JOIN providers p ON p.id = s.provider_id
		WHERE 1 = 1
	`
	var args []any

	if searchTerm != "" {
		search := "%" + searchTerm + "%"
		query += ` AND (
			s.hostname LIKE ?
			OR s.type LIKE ?
			OR COALESCE(o.name, '') LIKE ?
			OR COALESCE(l.name, '') LIKE ?
			OR COALESCE(p.name, '') LIKE ?
			OR COALESCE(s.currency, '') LIKE ?
		)`
		args = append(args, search, search, search, search, search, search)
	}

	query += ` ORDER BY s.hostname`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*Server
	for rows.Next() {
		item, scanErr := scanServer(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *Repository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM servers`).Scan(&count)
	return count, err
}

func (r *Repository) Update(ctx context.Context, item *Server) error {
	item.Currency = normalizeCurrency(item.Currency)
	item.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, `
		UPDATE servers
		SET
			hostname = ?,
			type = ?,
			os_id = ?,
			cpu_cores = ?,
			memory_gb = ?,
			disk_gb = ?,
			location_id = ?,
			provider_id = ?,
			due_date = ?,
			price = ?,
			currency = ?,
			since_date = ?,
			notes = ?,
			updated_at = ?
		WHERE id = ?
	`,
		item.Hostname,
		item.Type,
		dbString(item.OSID),
		dbInt(item.CPUCores),
		dbInt(item.MemoryGB),
		dbInt(item.DiskGB),
		dbString(item.LocationID),
		dbString(item.ProviderID),
		dbString(item.DueDate),
		dbFloat(item.Price),
		item.Currency,
		dbString(item.SinceDate),
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

	_, err := r.db.ExecContext(ctx, `DELETE FROM servers WHERE id = ?`, id)
	return err
}

func scanServer(scanner interface {
	Scan(dest ...any) error
}) (*Server, error) {
	item := &Server{}
	var osID sql.NullString
	var osName sql.NullString
	var cpuCores sql.NullInt64
	var memoryGB sql.NullInt64
	var diskGB sql.NullInt64
	var locationID sql.NullString
	var locationName sql.NullString
	var providerID sql.NullString
	var providerName sql.NullString
	var dueDate sql.NullString
	var price sql.NullFloat64
	var sinceDate sql.NullString
	var notes sql.NullString
	if err := scanner.Scan(
		&item.ID,
		&item.Hostname,
		&item.Type,
		&osID,
		&osName,
		&cpuCores,
		&memoryGB,
		&diskGB,
		&locationID,
		&locationName,
		&providerID,
		&providerName,
		&dueDate,
		&price,
		&item.Currency,
		&sinceDate,
		&notes,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}

	item.OSID = nullString(osID)
	item.OSName = nullString(osName)
	item.CPUCores = nullInt(cpuCores)
	item.MemoryGB = nullInt(memoryGB)
	item.DiskGB = nullInt(diskGB)
	item.LocationID = nullString(locationID)
	item.LocationName = nullString(locationName)
	item.ProviderID = nullString(providerID)
	item.ProviderName = nullString(providerName)
	item.DueDate = nullString(dueDate)
	item.Price = nullFloat(price)
	item.SinceDate = nullString(sinceDate)
	item.Notes = nullString(notes)
	return item, nil
}

func nullString(value sql.NullString) *string {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return nil
	}

	return &value.String
}

func nullInt(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}

	intValue := int(value.Int64)
	return &intValue
}

func nullFloat(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}

	return &value.Float64
}

func dbString(value *string) any {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}

	return strings.TrimSpace(*value)
}

func dbInt(value *int) any {
	if value == nil {
		return nil
	}

	return *value
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

func normalizeCurrency(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" {
		return "MXN"
	}

	return value
}
