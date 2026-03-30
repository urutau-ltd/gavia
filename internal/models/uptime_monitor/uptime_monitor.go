package uptimemonitor

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Monitor struct {
	ID                    string    `json:"id"`
	Name                  string    `json:"name"`
	TargetURL             string    `json:"target_url"`
	Kind                  string    `json:"kind"`
	ExpectedStatus        int       `json:"expected_status"`
	ExpectedStatusMin     int       `json:"expected_status_min"`
	ExpectedStatusMax     int       `json:"expected_status_max"`
	HTTPMethod            string    `json:"http_method"`
	TLSMode               string    `json:"tls_mode"`
	RequestHeaders        *string   `json:"request_headers"`
	ExpectedBodySubstring *string   `json:"expected_body_substring"`
	CheckIntervalSeconds  int       `json:"check_interval_seconds"`
	TimeoutMS             int       `json:"timeout_ms"`
	Enabled               bool      `json:"enabled"`
	Notes                 *string   `json:"notes"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type Result struct {
	ID         string    `json:"id"`
	MonitorID  string    `json:"monitor_id"`
	CheckedAt  time.Time `json:"checked_at"`
	OK         bool      `json:"ok"`
	StatusCode *int      `json:"status_code"`
	LatencyMS  *int      `json:"latency_ms"`
	ErrorText  *string   `json:"error_text"`
	CreatedAt  time.Time `json:"created_at"`
}

type MonitorStatus struct {
	Monitor
	LastCheckedAt  *time.Time `json:"last_checked_at"`
	LastOK         *bool      `json:"last_ok"`
	LastStatusCode *int       `json:"last_status_code"`
	LastLatencyMS  *int       `json:"last_latency_ms"`
	LastErrorText  *string    `json:"last_error_text"`
}

type Summary struct {
	Total    int `json:"total"`
	Up       int `json:"up"`
	Down     int `json:"down"`
	Unknown  int `json:"unknown"`
	Enabled  int `json:"enabled"`
	Disabled int `json:"disabled"`
}

type Repository struct {
	db *sql.DB
}

func (m *Monitor) HTTPMethodValue() string {
	if m == nil {
		return "GET"
	}

	return normalizeHTTPMethod(m.HTTPMethod)
}

func (m *Monitor) TLSModeValue() string {
	if m == nil {
		return "skip"
	}

	return normalizeTLSMode(m.TLSMode)
}

func (m *Monitor) StatusRangeDisplay() string {
	if m == nil {
		return "200"
	}

	minimum, maximum := normalizeStatusRange(m.ExpectedStatusMin, m.ExpectedStatusMax, m.ExpectedStatus)
	if minimum == maximum {
		return strconv.Itoa(minimum)
	}

	return strconv.Itoa(minimum) + " to " + strconv.Itoa(maximum)
}

func (m *Monitor) RequestHeadersValue() string {
	if m == nil || m.RequestHeaders == nil {
		return ""
	}

	return *m.RequestHeaders
}

func (m *Monitor) ExpectedBodySubstringValue() string {
	if m == nil || m.ExpectedBodySubstring == nil {
		return ""
	}

	return *m.ExpectedBodySubstring
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetAll(ctx context.Context, limit int) ([]*MonitorStatus, error) {
	query := `
		SELECT
			m.id,
			m.name,
			m.target_url,
			m.kind,
			m.expected_status,
			m.expected_status_min,
			m.expected_status_max,
			m.http_method,
			m.tls_mode,
			m.request_headers,
			m.expected_body_substring,
			m.check_interval_seconds,
			m.timeout_ms,
			m.enabled,
			m.notes,
			m.created_at,
			m.updated_at,
			l.checked_at,
			l.ok,
			l.status_code,
			l.latency_ms,
			l.error_text
		FROM uptime_monitors m
		LEFT JOIN uptime_monitor_results l
			ON l.id = (
				SELECT id
				FROM uptime_monitor_results
				WHERE monitor_id = m.id
				ORDER BY checked_at DESC
				LIMIT 1
			)
		ORDER BY m.name ASC
	`

	var (
		rows *sql.Rows
		err  error
	)
	if limit > 0 {
		rows, err = r.db.QueryContext(ctx, query+` LIMIT ?`, limit)
	} else {
		rows, err = r.db.QueryContext(ctx, query)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*MonitorStatus
	for rows.Next() {
		item, err := scanMonitorStatus(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id string) (*Monitor, error) {
	item := &Monitor{}
	var (
		requestHeaders        sql.NullString
		expectedBodySubstring sql.NullString
		notes                 sql.NullString
	)
	err := r.db.QueryRowContext(ctx, `
		SELECT
			id,
			name,
			target_url,
			kind,
			expected_status,
			expected_status_min,
			expected_status_max,
			http_method,
			tls_mode,
			request_headers,
			expected_body_substring,
			check_interval_seconds,
			timeout_ms,
			enabled,
			notes,
			created_at,
			updated_at
		FROM uptime_monitors
		WHERE id = ?
	`, strings.TrimSpace(id)).Scan(
		&item.ID,
		&item.Name,
		&item.TargetURL,
		&item.Kind,
		&item.ExpectedStatus,
		&item.ExpectedStatusMin,
		&item.ExpectedStatusMax,
		&item.HTTPMethod,
		&item.TLSMode,
		&requestHeaders,
		&expectedBodySubstring,
		&item.CheckIntervalSeconds,
		&item.TimeoutMS,
		&item.Enabled,
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

	item.ExpectedStatusMin, item.ExpectedStatusMax = normalizeStatusRange(item.ExpectedStatusMin, item.ExpectedStatusMax, item.ExpectedStatus)
	item.HTTPMethod = normalizeHTTPMethod(item.HTTPMethod)
	item.TLSMode = normalizeTLSMode(item.TLSMode)
	item.RequestHeaders = nullableString(requestHeaders)
	item.ExpectedBodySubstring = nullableString(expectedBodySubstring)
	item.Notes = nullableString(notes)
	return item, nil
}

func (r *Repository) GetEnabled(ctx context.Context) ([]*Monitor, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id,
			name,
			target_url,
			kind,
			expected_status,
			expected_status_min,
			expected_status_max,
			http_method,
			tls_mode,
			request_headers,
			expected_body_substring,
			check_interval_seconds,
			timeout_ms,
			enabled,
			notes,
			created_at,
			updated_at
		FROM uptime_monitors
		WHERE enabled = 1
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*Monitor
	for rows.Next() {
		item, err := scanMonitor(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *Repository) Create(ctx context.Context, item *Monitor) error {
	newID, err := uuid.NewV7()
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	item.ID = newID.String()
	item.Kind = normalizeKind(item.Kind)
	item.Name = strings.TrimSpace(item.Name)
	item.TargetURL = strings.TrimSpace(item.TargetURL)
	item.ExpectedStatusMin, item.ExpectedStatusMax = normalizeStatusRange(item.ExpectedStatusMin, item.ExpectedStatusMax, item.ExpectedStatus)
	item.ExpectedStatus = item.ExpectedStatusMin
	item.HTTPMethod = normalizeHTTPMethod(item.HTTPMethod)
	item.TLSMode = normalizeTLSMode(item.TLSMode)
	item.CheckIntervalSeconds = normalizeInt(item.CheckIntervalSeconds, 300)
	item.TimeoutMS = normalizeInt(item.TimeoutMS, 5000)
	item.CreatedAt = now
	item.UpdatedAt = now

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO uptime_monitors (
			id,
			name,
			target_url,
			kind,
			expected_status,
			expected_status_min,
			expected_status_max,
			http_method,
			tls_mode,
			request_headers,
			expected_body_substring,
			check_interval_seconds,
			timeout_ms,
			enabled,
			notes,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		item.ID,
		item.Name,
		item.TargetURL,
		item.Kind,
		item.ExpectedStatus,
		item.ExpectedStatusMin,
		item.ExpectedStatusMax,
		item.HTTPMethod,
		item.TLSMode,
		stringPointerValue(item.RequestHeaders),
		stringPointerValue(item.ExpectedBodySubstring),
		item.CheckIntervalSeconds,
		item.TimeoutMS,
		item.Enabled,
		stringPointerValue(item.Notes),
		item.CreatedAt,
		item.UpdatedAt,
	)
	return err
}

func (r *Repository) Update(ctx context.Context, item *Monitor) error {
	item.Kind = normalizeKind(item.Kind)
	item.Name = strings.TrimSpace(item.Name)
	item.TargetURL = strings.TrimSpace(item.TargetURL)
	item.ExpectedStatusMin, item.ExpectedStatusMax = normalizeStatusRange(item.ExpectedStatusMin, item.ExpectedStatusMax, item.ExpectedStatus)
	item.ExpectedStatus = item.ExpectedStatusMin
	item.HTTPMethod = normalizeHTTPMethod(item.HTTPMethod)
	item.TLSMode = normalizeTLSMode(item.TLSMode)
	item.CheckIntervalSeconds = normalizeInt(item.CheckIntervalSeconds, 300)
	item.TimeoutMS = normalizeInt(item.TimeoutMS, 5000)
	item.UpdatedAt = time.Now().UTC()

	_, err := r.db.ExecContext(ctx, `
		UPDATE uptime_monitors
		SET
			name = ?,
			target_url = ?,
			kind = ?,
			expected_status = ?,
			expected_status_min = ?,
			expected_status_max = ?,
			http_method = ?,
			tls_mode = ?,
			request_headers = ?,
			expected_body_substring = ?,
			check_interval_seconds = ?,
			timeout_ms = ?,
			enabled = ?,
			notes = ?,
			updated_at = ?
		WHERE id = ?
	`,
		item.Name,
		item.TargetURL,
		item.Kind,
		item.ExpectedStatus,
		item.ExpectedStatusMin,
		item.ExpectedStatusMax,
		item.HTTPMethod,
		item.TLSMode,
		stringPointerValue(item.RequestHeaders),
		stringPointerValue(item.ExpectedBodySubstring),
		item.CheckIntervalSeconds,
		item.TimeoutMS,
		item.Enabled,
		stringPointerValue(item.Notes),
		item.UpdatedAt,
		item.ID,
	)
	return err
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM uptime_monitors WHERE id = ?`, strings.TrimSpace(id))
	return err
}

func (r *Repository) CreateResult(ctx context.Context, result *Result) error {
	newID, err := uuid.NewV7()
	if err != nil {
		return err
	}

	if result.CheckedAt.IsZero() {
		result.CheckedAt = time.Now().UTC()
	}
	result.ID = newID.String()

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO uptime_monitor_results (
			id,
			monitor_id,
			checked_at,
			ok,
			status_code,
			latency_ms,
			error_text
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`,
		result.ID,
		result.MonitorID,
		result.CheckedAt.UTC(),
		result.OK,
		intPointerValue(result.StatusCode),
		intPointerValue(result.LatencyMS),
		stringPointerValue(result.ErrorText),
	)
	return err
}

func (r *Repository) GetRecentResults(ctx context.Context, monitorID string, limit int) ([]*Result, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id,
			monitor_id,
			checked_at,
			ok,
			status_code,
			latency_ms,
			error_text,
			created_at
		FROM uptime_monitor_results
		WHERE monitor_id = ?
		ORDER BY checked_at DESC
		LIMIT ?
	`, strings.TrimSpace(monitorID), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*Result
	for rows.Next() {
		item, err := scanResult(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *Repository) GetSummary(ctx context.Context) (*Summary, error) {
	items, err := r.GetAll(ctx, 0)
	if err != nil {
		return nil, err
	}

	summary := &Summary{}
	for _, item := range items {
		summary.Total++
		if item.Enabled {
			summary.Enabled++
		} else {
			summary.Disabled++
		}

		switch {
		case item.LastOK == nil:
			summary.Unknown++
		case *item.LastOK:
			summary.Up++
		default:
			summary.Down++
		}
	}

	return summary, nil
}

func scanMonitor(scanner interface{ Scan(dest ...any) error }) (*Monitor, error) {
	item := &Monitor{}
	var (
		requestHeaders        sql.NullString
		expectedBodySubstring sql.NullString
		notes                 sql.NullString
	)
	if err := scanner.Scan(
		&item.ID,
		&item.Name,
		&item.TargetURL,
		&item.Kind,
		&item.ExpectedStatus,
		&item.ExpectedStatusMin,
		&item.ExpectedStatusMax,
		&item.HTTPMethod,
		&item.TLSMode,
		&requestHeaders,
		&expectedBodySubstring,
		&item.CheckIntervalSeconds,
		&item.TimeoutMS,
		&item.Enabled,
		&notes,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}

	item.ExpectedStatusMin, item.ExpectedStatusMax = normalizeStatusRange(item.ExpectedStatusMin, item.ExpectedStatusMax, item.ExpectedStatus)
	item.HTTPMethod = normalizeHTTPMethod(item.HTTPMethod)
	item.TLSMode = normalizeTLSMode(item.TLSMode)
	item.RequestHeaders = nullableString(requestHeaders)
	item.ExpectedBodySubstring = nullableString(expectedBodySubstring)
	item.Notes = nullableString(notes)
	return item, nil
}

func scanMonitorStatus(scanner interface{ Scan(dest ...any) error }) (*MonitorStatus, error) {
	item := &MonitorStatus{}
	var (
		requestHeaders        sql.NullString
		expectedBodySubstring sql.NullString
		notes                 sql.NullString
		checkedAt             sql.NullTime
		lastOK                sql.NullBool
		statusCode            sql.NullInt64
		latencyMS             sql.NullInt64
		errorText             sql.NullString
	)
	if err := scanner.Scan(
		&item.ID,
		&item.Name,
		&item.TargetURL,
		&item.Kind,
		&item.ExpectedStatus,
		&item.ExpectedStatusMin,
		&item.ExpectedStatusMax,
		&item.HTTPMethod,
		&item.TLSMode,
		&requestHeaders,
		&expectedBodySubstring,
		&item.CheckIntervalSeconds,
		&item.TimeoutMS,
		&item.Enabled,
		&notes,
		&item.CreatedAt,
		&item.UpdatedAt,
		&checkedAt,
		&lastOK,
		&statusCode,
		&latencyMS,
		&errorText,
	); err != nil {
		return nil, err
	}

	item.ExpectedStatusMin, item.ExpectedStatusMax = normalizeStatusRange(item.ExpectedStatusMin, item.ExpectedStatusMax, item.ExpectedStatus)
	item.HTTPMethod = normalizeHTTPMethod(item.HTTPMethod)
	item.TLSMode = normalizeTLSMode(item.TLSMode)
	item.RequestHeaders = nullableString(requestHeaders)
	item.ExpectedBodySubstring = nullableString(expectedBodySubstring)
	item.Notes = nullableString(notes)
	if checkedAt.Valid {
		item.LastCheckedAt = &checkedAt.Time
	}
	if lastOK.Valid {
		value := lastOK.Bool
		item.LastOK = &value
	}
	item.LastStatusCode = nullableInt(statusCode)
	item.LastLatencyMS = nullableInt(latencyMS)
	item.LastErrorText = nullableString(errorText)
	return item, nil
}

func scanResult(scanner interface{ Scan(dest ...any) error }) (*Result, error) {
	item := &Result{}
	var (
		statusCode sql.NullInt64
		latencyMS  sql.NullInt64
		errorText  sql.NullString
	)
	if err := scanner.Scan(
		&item.ID,
		&item.MonitorID,
		&item.CheckedAt,
		&item.OK,
		&statusCode,
		&latencyMS,
		&errorText,
		&item.CreatedAt,
	); err != nil {
		return nil, err
	}

	item.StatusCode = nullableInt(statusCode)
	item.LatencyMS = nullableInt(latencyMS)
	item.ErrorText = nullableString(errorText)
	return item, nil
}

func normalizeKind(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "http"
	}
	return value
}

func normalizeInt(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

func normalizeHTTPMethod(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS":
		return strings.ToUpper(strings.TrimSpace(value))
	default:
		return "GET"
	}
}

func normalizeTLSMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "verify":
		return "verify"
	default:
		return "skip"
	}
}

func normalizeStatusRange(minimum, maximum, fallback int) (int, int) {
	fallback = normalizeInt(fallback, 200)
	minimum = normalizeInt(minimum, fallback)
	maximum = normalizeInt(maximum, fallback)
	if maximum < minimum {
		maximum = minimum
	}
	return minimum, maximum
}

func nullableString(value sql.NullString) *string {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return nil
	}
	result := value.String
	return &result
}

func nullableInt(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	result := int(value.Int64)
	return &result
}

func stringPointerValue(value *string) any {
	if value == nil {
		return nil
	}
	text := strings.TrimSpace(*value)
	if text == "" {
		return nil
	}
	return text
}

func intPointerValue(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}
