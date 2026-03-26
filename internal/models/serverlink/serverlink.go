package serverlink

import (
	"context"
	"database/sql"
	"strings"

	ipmodel "codeberg.org/urutau-ltd/gavia/internal/models/ip"
	labelmodel "codeberg.org/urutau-ltd/gavia/internal/models/label"
	"github.com/google/uuid"
)

type ServerIPAssignment struct {
	ID       string `json:"id"`
	ServerID string `json:"server_id"`
	IPID     string `json:"ip_id"`
}

type ServerLabelAssignment struct {
	ID       string `json:"id"`
	ServerID string `json:"server_id"`
	LabelID  string `json:"label_id"`
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetServerIPs(ctx context.Context, serverID string) ([]*ipmodel.IP, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			i.id,
			i.address,
			i.type,
			i.city,
			i.country,
			i.org,
			i.asn,
			i.isp,
			i.notes,
			i.created_at,
			i.updated_at
		FROM server_ips si
		JOIN ips i ON i.id = si.ip_id
		WHERE si.server_id = ?
		ORDER BY i.address
	`, strings.TrimSpace(serverID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*ipmodel.IP
	for rows.Next() {
		item, scanErr := scanIP(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *Repository) GetServerLabels(ctx context.Context, serverID string) ([]*labelmodel.Label, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			l.id,
			l.name,
			l.notes,
			l.created_at,
			l.updated_at
		FROM server_labels sl
		JOIN labels l ON l.id = sl.label_id
		WHERE sl.server_id = ?
		ORDER BY l.name
	`, strings.TrimSpace(serverID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*labelmodel.Label
	for rows.Next() {
		item, scanErr := scanLabel(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *Repository) GetAssignedIPIDs(ctx context.Context, serverID string) (map[string]struct{}, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT ip_id FROM server_ips WHERE server_id = ?`, strings.TrimSpace(serverID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	selected := make(map[string]struct{})
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		selected[id] = struct{}{}
	}

	return selected, rows.Err()
}

func (r *Repository) GetAssignedLabelIDs(ctx context.Context, serverID string) (map[string]struct{}, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT label_id FROM server_labels WHERE server_id = ?`, strings.TrimSpace(serverID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	selected := make(map[string]struct{})
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		selected[id] = struct{}{}
	}

	return selected, rows.Err()
}

func (r *Repository) ReplaceServerIPs(ctx context.Context, serverID string, ipIDs []string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM server_ips WHERE server_id = ?`, strings.TrimSpace(serverID)); err != nil {
		return err
	}

	for _, ipID := range uniqueStrings(ipIDs) {
		newID, err := uuid.NewV7()
		if err != nil {
			return err
		}

		if _, err := r.db.ExecContext(ctx, `
			INSERT INTO server_ips (id, server_id, ip_id)
			VALUES (?, ?, ?)
		`, newID.String(), strings.TrimSpace(serverID), ipID); err != nil {
			return err
		}
	}

	return nil
}

func (r *Repository) ReplaceServerLabels(ctx context.Context, serverID string, labelIDs []string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM server_labels WHERE server_id = ?`, strings.TrimSpace(serverID)); err != nil {
		return err
	}

	for _, labelID := range uniqueStrings(labelIDs) {
		newID, err := uuid.NewV7()
		if err != nil {
			return err
		}

		if _, err := r.db.ExecContext(ctx, `
			INSERT INTO server_labels (id, server_id, label_id)
			VALUES (?, ?, ?)
		`, newID.String(), strings.TrimSpace(serverID), labelID); err != nil {
			return err
		}
	}

	return nil
}

func ListServerIPAssignments(ctx context.Context, db *sql.DB) ([]*ServerIPAssignment, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, server_id, ip_id FROM server_ips ORDER BY server_id, ip_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*ServerIPAssignment
	for rows.Next() {
		item := &ServerIPAssignment{}
		if err := rows.Scan(&item.ID, &item.ServerID, &item.IPID); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func ListServerLabelAssignments(ctx context.Context, db *sql.DB) ([]*ServerLabelAssignment, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, server_id, label_id FROM server_labels ORDER BY server_id, label_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*ServerLabelAssignment
	for rows.Next() {
		item := &ServerLabelAssignment{}
		if err := rows.Scan(&item.ID, &item.ServerID, &item.LabelID); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func scanIP(scanner interface{ Scan(dest ...any) error }) (*ipmodel.IP, error) {
	item := &ipmodel.IP{}
	var city sql.NullString
	var country sql.NullString
	var org sql.NullString
	var asn sql.NullString
	var isp sql.NullString
	var notes sql.NullString
	if err := scanner.Scan(
		&item.ID,
		&item.Address,
		&item.Type,
		&city,
		&country,
		&org,
		&asn,
		&isp,
		&notes,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}

	if city.Valid && strings.TrimSpace(city.String) != "" {
		item.City = &city.String
	}
	if country.Valid && strings.TrimSpace(country.String) != "" {
		item.Country = &country.String
	}
	if org.Valid && strings.TrimSpace(org.String) != "" {
		item.Org = &org.String
	}
	if asn.Valid && strings.TrimSpace(asn.String) != "" {
		item.ASN = &asn.String
	}
	if isp.Valid && strings.TrimSpace(isp.String) != "" {
		item.ISP = &isp.String
	}
	if notes.Valid && strings.TrimSpace(notes.String) != "" {
		item.Notes = &notes.String
	}
	return item, nil
}

func scanLabel(scanner interface{ Scan(dest ...any) error }) (*labelmodel.Label, error) {
	item := &labelmodel.Label{}
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

	if notes.Valid && strings.TrimSpace(notes.String) != "" {
		item.Notes = &notes.String
	}
	return item, nil
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}

	return out
}
