package backup

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	accountsetting "codeberg.org/urutau-ltd/gavia/internal/models/account_setting"
	appsetting "codeberg.org/urutau-ltd/gavia/internal/models/app_setting"
	"codeberg.org/urutau-ltd/gavia/internal/models/dnsrecord"
	"codeberg.org/urutau-ltd/gavia/internal/models/domain"
	expenserentry "codeberg.org/urutau-ltd/gavia/internal/models/expense_entry"
	"codeberg.org/urutau-ltd/gavia/internal/models/hosting"
	ipmodel "codeberg.org/urutau-ltd/gavia/internal/models/ip"
	labelmodel "codeberg.org/urutau-ltd/gavia/internal/models/label"
	"codeberg.org/urutau-ltd/gavia/internal/models/location"
	operatingsystem "codeberg.org/urutau-ltd/gavia/internal/models/operating_system"
	"codeberg.org/urutau-ltd/gavia/internal/models/provider"
	servermodel "codeberg.org/urutau-ltd/gavia/internal/models/server"
	"codeberg.org/urutau-ltd/gavia/internal/models/serverlink"
	"codeberg.org/urutau-ltd/gavia/internal/models/subscription"
	"codeberg.org/urutau-ltd/gavia/internal/security"
)

const SnapshotFormat = "gavia.backup.v1"

type Snapshot struct {
	Format           string                              `json:"format"`
	ExportedAt       time.Time                           `json:"exported_at"`
	AccountSettings  *accountsetting.AccountSettings     `json:"account_settings"`
	AppSettings      *appsetting.AppSettings             `json:"app_settings"`
	ExpenseEntries   []*expenserentry.ExpenseEntry       `json:"expense_entries"`
	Providers        []*provider.Provider                `json:"providers"`
	Locations        []*location.Location                `json:"locations"`
	OperatingSystems []*operatingsystem.OperatingSystem  `json:"operating_systems"`
	IPs              []*ipmodel.IP                       `json:"ips"`
	DNSRecords       []*dnsrecord.DNSRecord              `json:"dns_records"`
	Labels           []*labelmodel.Label                 `json:"labels"`
	Domains          []*domain.Domain                    `json:"domains"`
	Hostings         []*hosting.Hosting                  `json:"hostings"`
	Servers          []*servermodel.Server               `json:"servers"`
	Subscriptions    []*subscription.Subscription        `json:"subscriptions"`
	ServerIPs        []*serverlink.ServerIPAssignment    `json:"server_ips"`
	ServerLabels     []*serverlink.ServerLabelAssignment `json:"server_labels"`
}

type Service struct {
	db            *sql.DB
	account       *accountsetting.AccountSettingsRepository
	app           *appsetting.AppSettingsRepository
	expenses      *expenserentry.ExpenseEntryRepository
	providers     *provider.ProviderRepository
	locations     *location.LocationRepository
	os            *operatingsystem.OperatingSystemRepository
	ips           *ipmodel.Repository
	dnsRecords    *dnsrecord.Repository
	labels        *labelmodel.Repository
	domains       *domain.Repository
	hostings      *hosting.Repository
	servers       *servermodel.Repository
	subscriptions *subscription.Repository
}

func NewService(db *sql.DB) *Service {
	return &Service{
		db:            db,
		account:       accountsetting.NewAccountSettingsRepository(db),
		app:           appsetting.NewAppSettingsRepository(db),
		expenses:      expenserentry.NewExpenseEntryRepository(db),
		providers:     provider.NewProviderRepository(db),
		locations:     location.NewLocationRepository(db),
		os:            operatingsystem.NewOperatingSystemRepository(db),
		ips:           ipmodel.NewRepository(db),
		dnsRecords:    dnsrecord.NewRepository(db),
		labels:        labelmodel.NewRepository(db),
		domains:       domain.NewRepository(db),
		hostings:      hosting.NewRepository(db),
		servers:       servermodel.NewRepository(db),
		subscriptions: subscription.NewRepository(db),
	}
}

func (s *Service) Export(ctx context.Context) (*Snapshot, error) {
	account, err := s.account.Get(ctx)
	if err != nil {
		return nil, err
	}

	appSettings, err := s.app.Get(ctx)
	if err != nil {
		return nil, err
	}

	expenseEntries, err := s.expenses.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	providers, err := s.providers.GetAll(ctx, "", 0)
	if err != nil {
		return nil, err
	}

	locations, err := s.locations.GetAll(ctx, "", 0)
	if err != nil {
		return nil, err
	}

	operatingSystems, err := s.os.GetAll(ctx, "", 0)
	if err != nil {
		return nil, err
	}

	ips, err := s.ips.GetAll(ctx, "", 0)
	if err != nil {
		return nil, err
	}

	dnsRecords, err := s.dnsRecords.GetAll(ctx, "", 0)
	if err != nil {
		return nil, err
	}

	labels, err := s.labels.GetAll(ctx, "", 0)
	if err != nil {
		return nil, err
	}

	domains, err := s.domains.GetAll(ctx, "", 0)
	if err != nil {
		return nil, err
	}

	hostings, err := s.hostings.GetAll(ctx, "", 0)
	if err != nil {
		return nil, err
	}

	servers, err := s.servers.GetAll(ctx, "", 0)
	if err != nil {
		return nil, err
	}

	subscriptions, err := s.subscriptions.GetAll(ctx, "", 0)
	if err != nil {
		return nil, err
	}

	serverIPs, err := serverlink.ListServerIPAssignments(ctx, s.db)
	if err != nil {
		return nil, err
	}

	serverLabels, err := serverlink.ListServerLabelAssignments(ctx, s.db)
	if err != nil {
		return nil, err
	}

	return &Snapshot{
		Format:           SnapshotFormat,
		ExportedAt:       time.Now().UTC(),
		AccountSettings:  account,
		AppSettings:      appSettings,
		ExpenseEntries:   expenseEntries,
		Providers:        providers,
		Locations:        locations,
		OperatingSystems: operatingSystems,
		IPs:              ips,
		DNSRecords:       dnsRecords,
		Labels:           labels,
		Domains:          domains,
		Hostings:         hostings,
		Servers:          servers,
		Subscriptions:    subscriptions,
		ServerIPs:        serverIPs,
		ServerLabels:     serverLabels,
	}, nil
}

func (s *Service) ExportJSON(ctx context.Context) ([]byte, error) {
	snapshot, err := s.Export(ctx)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(snapshot, "", "  ")
}

func (s *Service) ExportEncryptedJSON(ctx context.Context, publicKey string) ([]byte, error) {
	payload, err := s.ExportJSON(ctx)
	if err != nil {
		return nil, err
	}

	bundle, err := security.EncryptBackup(payload, publicKey)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(bundle, "", "  ")
}

func (s *Service) ParseImport(payload []byte, recoveryKey string) (*Snapshot, error) {
	var probe struct {
		Format string `json:"format"`
	}
	if err := json.Unmarshal(payload, &probe); err != nil {
		return nil, err
	}

	if strings.TrimSpace(probe.Format) == "" {
		return nil, errors.New("backup payload is missing its format marker")
	}

	if probe.Format != SnapshotFormat {
		var bundle security.EncryptedBackup
		if err := json.Unmarshal(payload, &bundle); err != nil {
			return nil, err
		}

		decrypted, err := security.DecryptBackup(bundle, recoveryKey)
		if err != nil {
			return nil, err
		}

		payload = decrypted
	}

	var snapshot Snapshot
	if err := json.Unmarshal(payload, &snapshot); err != nil {
		return nil, err
	}

	if err := validateSnapshot(&snapshot); err != nil {
		return nil, err
	}

	return &snapshot, nil
}

func (s *Service) Import(ctx context.Context, snapshot *Snapshot) error {
	if err := validateSnapshot(snapshot); err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if err := replaceSnapshot(ctx, tx, snapshot); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

func replaceSnapshot(ctx context.Context, tx *sql.Tx, snapshot *Snapshot) error {
	deleteStatements := []string{
		`DELETE FROM user_sessions`,
		`DELETE FROM server_ips`,
		`DELETE FROM server_labels`,
		`DELETE FROM hostings`,
		`DELETE FROM servers`,
		`DELETE FROM subscriptions`,
		`DELETE FROM domains`,
		`DELETE FROM dns_records`,
		`DELETE FROM ips`,
		`DELETE FROM labels`,
		`DELETE FROM expense_entries`,
		`DELETE FROM providers`,
		`DELETE FROM locations`,
		`DELETE FROM operating_systems`,
		`DELETE FROM account_settings`,
		`DELETE FROM app_settings`,
	}
	for _, statement := range deleteStatements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}

	for _, item := range snapshot.ExpenseEntries {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO expense_entries (
				id,
				title,
				category,
				amount,
				currency,
				occurred_on,
				notes,
				created_at,
				updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			item.ID,
			item.Title,
			defaultString(item.Category, "manual"),
			item.Amount,
			defaultString(item.Currency, "MXN"),
			defaultString(item.OccurredOn, time.Now().UTC().Format(time.DateOnly)),
			stringPointerValue(item.Notes),
			normalizeTime(item.CreatedAt),
			normalizeTime(item.UpdatedAt),
		); err != nil {
			return err
		}
	}

	for _, item := range snapshot.Providers {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO providers (id, name, website, notes, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`,
			item.Id,
			item.Name,
			stringPointerValue(item.Website),
			stringPointerValue(item.Notes),
			normalizeTime(item.CreatedAt),
			normalizeTime(item.UpdatedAt),
		); err != nil {
			return err
		}
	}

	for _, item := range snapshot.Locations {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO locations (id, name, city, country, notes, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`,
			item.Id,
			item.Name,
			stringPointerValue(item.City),
			stringPointerValue(item.Country),
			stringPointerValue(item.Notes),
			normalizeTime(item.CreatedAt),
			normalizeTime(item.UpdatedAt),
		); err != nil {
			return err
		}
	}

	for _, item := range snapshot.OperatingSystems {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO operating_systems (id, name, notes, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?)
		`,
			item.Id,
			item.Name,
			stringPointerValue(item.Notes),
			normalizeTime(item.CreatedAt),
			normalizeTime(item.UpdatedAt),
		); err != nil {
			return err
		}
	}

	for _, item := range snapshot.IPs {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO ips (
				id,
				address,
				type,
				city,
				country,
				org,
				asn,
				isp,
				notes,
				created_at,
				updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			item.ID,
			item.Address,
			defaultString(item.Type, "ipv4"),
			stringPointerValue(item.City),
			stringPointerValue(item.Country),
			stringPointerValue(item.Org),
			stringPointerValue(item.ASN),
			stringPointerValue(item.ISP),
			stringPointerValue(item.Notes),
			normalizeTime(item.CreatedAt),
			normalizeTime(item.UpdatedAt),
		); err != nil {
			return err
		}
	}

	for _, item := range snapshot.DNSRecords {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO dns_records (id, type, hostname, domain_id, address, notes, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`,
			item.ID,
			defaultString(item.Type, "A"),
			item.Hostname,
			stringPointerValue(item.DomainID),
			item.Address,
			stringPointerValue(item.Notes),
			normalizeTime(item.CreatedAt),
			normalizeTime(item.UpdatedAt),
		); err != nil {
			return err
		}
	}

	for _, item := range snapshot.Labels {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO labels (id, name, notes, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?)
		`,
			item.ID,
			item.Name,
			stringPointerValue(item.Notes),
			normalizeTime(item.CreatedAt),
			normalizeTime(item.UpdatedAt),
		); err != nil {
			return err
		}
	}

	for _, item := range snapshot.Domains {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO domains (
				id,
				domain,
				provider_id,
				due_date,
				currency,
				price,
				notes,
				created_at,
				updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			item.ID,
			item.Domain,
			stringPointerValue(item.ProviderID),
			stringPointerValue(item.DueDate),
			defaultString(item.Currency, "MXN"),
			floatPointerValue(item.Price),
			stringPointerValue(item.Notes),
			normalizeTime(item.CreatedAt),
			normalizeTime(item.UpdatedAt),
		); err != nil {
			return err
		}
	}

	for _, item := range snapshot.Hostings {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO hostings (
				id,
				name,
				type,
				location_id,
				provider_id,
				domain_id,
				disk_gb,
				price,
				currency,
				due_date,
				since_date,
				notes,
				created_at,
				updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			item.ID,
			item.Name,
			item.Type,
			stringPointerValue(item.LocationID),
			stringPointerValue(item.ProviderID),
			stringPointerValue(item.DomainID),
			intPointerValue(item.DiskGB),
			floatPointerValue(item.Price),
			defaultString(item.Currency, "MXN"),
			stringPointerValue(item.DueDate),
			stringPointerValue(item.SinceDate),
			stringPointerValue(item.Notes),
			normalizeTime(item.CreatedAt),
			normalizeTime(item.UpdatedAt),
		); err != nil {
			return err
		}
	}

	for _, item := range snapshot.Servers {
		if _, err := tx.ExecContext(ctx, `
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
			stringPointerValue(item.OSID),
			intPointerValue(item.CPUCores),
			intPointerValue(item.MemoryGB),
			intPointerValue(item.DiskGB),
			stringPointerValue(item.LocationID),
			stringPointerValue(item.ProviderID),
			stringPointerValue(item.DueDate),
			floatPointerValue(item.Price),
			defaultString(item.Currency, "MXN"),
			stringPointerValue(item.SinceDate),
			stringPointerValue(item.Notes),
			normalizeTime(item.CreatedAt),
			normalizeTime(item.UpdatedAt),
		); err != nil {
			return err
		}
	}

	for _, item := range snapshot.Subscriptions {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO subscriptions (
				id,
				name,
				type,
				price,
				currency,
				due_date,
				since_date,
				renewal_period,
				notes,
				created_at,
				updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			item.ID,
			item.Name,
			item.Type,
			floatPointerValue(item.Price),
			defaultString(item.Currency, "MXN"),
			stringPointerValue(item.DueDate),
			stringPointerValue(item.SinceDate),
			stringPointerValue(item.RenewalPeriod),
			stringPointerValue(item.Notes),
			normalizeTime(item.CreatedAt),
			normalizeTime(item.UpdatedAt),
		); err != nil {
			return err
		}
	}

	for _, item := range snapshot.ServerIPs {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO server_ips (id, server_id, ip_id)
			VALUES (?, ?, ?)
		`,
			item.ID,
			item.ServerID,
			item.IPID,
		); err != nil {
			return err
		}
	}

	for _, item := range snapshot.ServerLabels {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO server_labels (id, server_id, label_id)
			VALUES (?, ?, ?)
		`,
			item.ID,
			item.ServerID,
			item.LabelID,
		); err != nil {
			return err
		}
	}

	account := snapshot.AccountSettings
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO account_settings (
			id,
			username,
			password_hash,
			api_token_hash,
			api_token_hint,
			avatar_path,
			recovery_public_key,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		normalizeSingletonID(account.ID, "account"),
		account.Username,
		account.PasswordHash,
		account.APITokenHash,
		account.APITokenHint,
		defaultAvatarPath(account.AvatarPath),
		account.RecoveryPublicKey,
		normalizeTime(account.CreatedAt),
		normalizeTime(account.UpdatedAt),
	); err != nil {
		return err
	}

	settings := snapshot.AppSettings
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO app_settings (
			id,
			show_version_footer,
			default_server_os,
			default_currency,
			dashboard_currency,
			dashboard_due_soon_amount,
			dashboard_expense_history_json,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		normalizeSingletonID(settings.ID, "app"),
		settings.ShowVersionFooter,
		defaultString(settings.DefaultServerOS, "Linux"),
		defaultString(settings.DefaultCurrency, "MXN"),
		defaultString(settings.DashboardCurrency, "MXN"),
		settings.DashboardDueSoonAmount,
		defaultString(settings.DashboardExpenseHistoryJSON, "[]"),
		normalizeTime(settings.CreatedAt),
		normalizeTime(settings.UpdatedAt),
	); err != nil {
		return err
	}

	return nil
}

func validateSnapshot(snapshot *Snapshot) error {
	if snapshot == nil {
		return errors.New("backup snapshot is empty")
	}

	if strings.TrimSpace(snapshot.Format) != SnapshotFormat {
		return errors.New("unsupported backup format")
	}

	if snapshot.AccountSettings == nil {
		return errors.New("backup snapshot is missing account settings")
	}

	if strings.TrimSpace(snapshot.AccountSettings.Username) == "" {
		return errors.New("backup snapshot is missing the account username")
	}

	if strings.TrimSpace(snapshot.AccountSettings.PasswordHash) == "" {
		return errors.New("backup snapshot is missing the account password hash")
	}

	if snapshot.AppSettings == nil {
		return errors.New("backup snapshot is missing app settings")
	}

	if strings.TrimSpace(snapshot.AppSettings.DefaultServerOS) == "" {
		return errors.New("backup snapshot is missing the default server operating system")
	}

	return nil
}

func normalizeTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}

	return value
}

func stringPointerValue(value *string) any {
	if value == nil {
		return nil
	}

	return *value
}

func intPointerValue(value *int) any {
	if value == nil {
		return nil
	}

	return *value
}

func floatPointerValue(value *float64) any {
	if value == nil {
		return nil
	}

	return *value
}

func normalizeSingletonID(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}

	return value
}

func defaultString(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}

	return value
}

func defaultAvatarPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "/static/img/avatar-1.svg"
	}

	return value
}
