package appsetting

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

const singletonID = "app"

type AppSettings struct {
	ID                          string    `json:"id"`
	ShowVersionFooter           bool      `json:"show_version_footer"`
	DefaultServerOS             string    `json:"default_server_os"`
	DefaultCurrency             string    `json:"default_currency"`
	DashboardCurrency           string    `json:"dashboard_currency"`
	DashboardDueSoonAmount      int       `json:"dashboard_due_soon_amount"`
	DashboardExpenseHistoryJSON string    `json:"dashboard_expense_history_json"`
	CreatedAt                   time.Time `json:"created_at"`
	UpdatedAt                   time.Time `json:"updated_at"`
}

type AppSettingsRepository struct {
	db *sql.DB
}

func Defaults() *AppSettings {
	return &AppSettings{
		ID:                          singletonID,
		ShowVersionFooter:           true,
		DefaultServerOS:             "Linux",
		DefaultCurrency:             "MXN",
		DashboardCurrency:           "MXN",
		DashboardDueSoonAmount:      5,
		DashboardExpenseHistoryJSON: "[]",
	}
}

func NewAppSettingsRepository(db *sql.DB) *AppSettingsRepository {
	return &AppSettingsRepository{db: db}
}

func (r *AppSettingsRepository) Get(ctx context.Context) (*AppSettings, error) {
	settings := &AppSettings{}
	err := r.db.QueryRowContext(ctx, `
		SELECT
			id,
			show_version_footer,
			default_server_os,
			default_currency,
			dashboard_currency,
			dashboard_due_soon_amount,
			dashboard_expense_history_json,
			created_at,
			updated_at
		FROM app_settings
		WHERE id = ?
	`, singletonID).Scan(
		&settings.ID,
		&settings.ShowVersionFooter,
		&settings.DefaultServerOS,
		&settings.DefaultCurrency,
		&settings.DashboardCurrency,
		&settings.DashboardDueSoonAmount,
		&settings.DashboardExpenseHistoryJSON,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return settings, nil
}

func (r *AppSettingsRepository) Update(ctx context.Context, settings *AppSettings) error {
	settings.ID = singletonID
	settings.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, `
		UPDATE app_settings
		SET
			show_version_footer = ?,
			default_server_os = ?,
			default_currency = ?,
			dashboard_currency = ?,
			dashboard_due_soon_amount = ?,
			dashboard_expense_history_json = ?,
			updated_at = ?
		WHERE id = ?
	`,
		settings.ShowVersionFooter,
		settings.DefaultServerOS,
		settings.DefaultCurrency,
		settings.DashboardCurrency,
		settings.DashboardDueSoonAmount,
		settings.DashboardExpenseHistoryJSON,
		settings.UpdatedAt,
		settings.ID,
	)
	return err
}
