package main

import (
	"context"
	"database/sql"
	"io/fs"
	"log/slog"
	"time"

	"codeberg.org/urutau-ltd/aile/v2"
	"codeberg.org/urutau-ltd/aile/v2/x/combine"
	xlogger "codeberg.org/urutau-ltd/aile/v2/x/logger"
	requestid "codeberg.org/urutau-ltd/aile/v2/x/request_id"
	"codeberg.org/urutau-ltd/aile/v2/x/resource"
	backupapi "codeberg.org/urutau-ltd/gavia/internal/api/backup"
	dashboardapi "codeberg.org/urutau-ltd/gavia/internal/api/dashboard"
	"codeberg.org/urutau-ltd/gavia/internal/auth"
	"codeberg.org/urutau-ltd/gavia/internal/backup"
	"codeberg.org/urutau-ltd/gavia/internal/compression"
	"codeberg.org/urutau-ltd/gavia/internal/csrf"
	"codeberg.org/urutau-ltd/gavia/internal/finance"
	accountsetting "codeberg.org/urutau-ltd/gavia/internal/models/account_setting"
	appsetting "codeberg.org/urutau-ltd/gavia/internal/models/app_setting"
	exchangerate "codeberg.org/urutau-ltd/gavia/internal/models/exchange_rate"
	expenseentry "codeberg.org/urutau-ltd/gavia/internal/models/expense_entry"
	operatingsystem "codeberg.org/urutau-ltd/gavia/internal/models/operating_system"
	runtimesample "codeberg.org/urutau-ltd/gavia/internal/models/runtime_sample"
	"codeberg.org/urutau-ltd/gavia/internal/models/session"
	uptimemonitor "codeberg.org/urutau-ltd/gavia/internal/models/uptime_monitor"
	"codeberg.org/urutau-ltd/gavia/internal/observability"
	"codeberg.org/urutau-ltd/gavia/internal/ui"
	accountsettings "codeberg.org/urutau-ltd/gavia/internal/ui/features/account_settings"
	appsettings "codeberg.org/urutau-ltd/gavia/internal/ui/features/app_settings"
	"codeberg.org/urutau-ltd/gavia/internal/ui/features/dashboard"
	"codeberg.org/urutau-ltd/gavia/internal/ui/features/dns"
	"codeberg.org/urutau-ltd/gavia/internal/ui/features/domains"
	"codeberg.org/urutau-ltd/gavia/internal/ui/features/hostings"
	"codeberg.org/urutau-ltd/gavia/internal/ui/features/ips"
	jslicenseinfo "codeberg.org/urutau-ltd/gavia/internal/ui/features/javascript_license_info"
	"codeberg.org/urutau-ltd/gavia/internal/ui/features/labels"
	licensespage "codeberg.org/urutau-ltd/gavia/internal/ui/features/licenses"
	"codeberg.org/urutau-ltd/gavia/internal/ui/features/locations"
	"codeberg.org/urutau-ltd/gavia/internal/ui/features/login"
	"codeberg.org/urutau-ltd/gavia/internal/ui/features/logout"
	operatingsystems "codeberg.org/urutau-ltd/gavia/internal/ui/features/operating_systems"
	"codeberg.org/urutau-ltd/gavia/internal/ui/features/providers"
	"codeberg.org/urutau-ltd/gavia/internal/ui/features/servers"
	"codeberg.org/urutau-ltd/gavia/internal/ui/features/subscriptions"
	uptimepage "codeberg.org/urutau-ltd/gavia/internal/ui/features/uptime"
	"codeberg.org/urutau-ltd/gavia/internal/uptime"
)

type appRepositories struct {
	account       *accountsetting.AccountSettingsRepository
	appSettings   *appsetting.AppSettingsRepository
	exchangeRate  *exchangerate.Repository
	expense       *expenseentry.ExpenseEntryRepository
	os            *operatingsystem.OperatingSystemRepository
	runtimeSample *runtimesample.Repository
	session       *session.SessionRepository
	uptimeMonitor *uptimemonitor.Repository
}

type appServices struct {
	auth          *auth.Service
	backup        *backup.Service
	compression   *compression.Service
	csrf          *csrf.Service
	finance       *finance.Service
	observability *observability.Service
	uptime        *uptime.Service
}

type appHandlers struct {
	dashboard       dashboardRoutes
	provider        resource.Collection
	location        resource.Collection
	os              resource.Collection
	ip              resource.Collection
	dns             resource.Collection
	label           resource.Collection
	domain          resource.Collection
	hosting         resource.Collection
	server          resource.Collection
	subscription    resource.Collection
	accountSettings singletonSettingsRoutes
	appSettings     appSettingsRoutes
	login           loginRoutes
	logout          logoutRoutes
	backupAPI       backupAPIRoutes
	dashboardAPI    dashboardAPIRoutes
	licenses        licensesRoutes
	jsLicenseInfo   javascriptLicenseInfoRoutes
	uptime          uptimeRoutes
}

func newRepositories(db *sql.DB) appRepositories {
	return appRepositories{
		account:       accountsetting.NewAccountSettingsRepository(db),
		appSettings:   appsetting.NewAppSettingsRepository(db),
		exchangeRate:  exchangerate.NewRepository(db),
		expense:       expenseentry.NewExpenseEntryRepository(db),
		os:            operatingsystem.NewOperatingSystemRepository(db),
		runtimeSample: runtimesample.NewRepository(db),
		session:       session.NewSessionRepository(db),
		uptimeMonitor: uptimemonitor.NewRepository(db),
	}
}

func newServices(logger *slog.Logger, db *sql.DB, repos appRepositories) appServices {
	return appServices{
		auth:          auth.NewService(repos.account, repos.session),
		backup:        backup.NewService(db),
		compression:   compression.NewService(),
		csrf:          csrf.NewService(),
		finance:       finance.NewService(logger, repos.exchangeRate, finance.ServiceConfig{}),
		observability: observability.NewService(logger, db, repos.runtimeSample, 30*time.Second),
		uptime:        uptime.NewService(logger, repos.uptimeMonitor, nil, 30*time.Second),
	}
}

func newHandlers(
	logger *slog.Logger,
	uiRoot fs.FS,
	db *sql.DB,
	repos appRepositories,
	services appServices,
) appHandlers {
	return appHandlers{
		dashboard: dashboard.NewHandler(logger, uiRoot, db),
		provider:  providers.NewHandler(logger, uiRoot, db),
		location:  locations.NewHandler(logger, uiRoot, db),
		os:        operatingsystems.NewHandler(logger, uiRoot, db),
		ip:        ips.NewHandler(logger, uiRoot, db),
		dns:       dns.NewHandler(logger, uiRoot, db),
		label:     labels.NewHandler(logger, uiRoot, db),
		domain:    domains.NewHandler(logger, uiRoot, db),
		hosting:   hostings.NewHandler(logger, uiRoot, db),
		server:    servers.NewHandler(logger, uiRoot, db),
		subscription: subscriptions.NewHandler(
			logger,
			uiRoot,
			db,
		),
		login:  login.NewHandler(logger, uiRoot, services.auth),
		logout: logout.NewHandler(logger, services.auth),
		accountSettings: accountsettings.NewHandler(
			logger,
			uiRoot,
			repos.account,
			services.auth,
		),
		appSettings: appsettings.NewHandler(
			logger,
			uiRoot,
			repos.appSettings,
			repos.account,
			repos.expense,
			repos.os,
			services.backup,
			services.auth,
		),
		backupAPI:    backupapi.NewHandler(logger, services.backup, repos.account),
		dashboardAPI: dashboardapi.NewHandler(logger, db),
		licenses:     licensespage.NewHandler(logger, uiRoot),
		jsLicenseInfo: jslicenseinfo.NewHandler(
			logger,
			uiRoot,
		),
		uptime: uptimepage.NewHandler(logger, uiRoot, repos.uptimeMonitor),
	}
}

func applyUISettings(ctx context.Context, logger *slog.Logger, repo *appsetting.AppSettingsRepository) {
	settings, err := repo.Get(ctx)
	if err != nil {
		logger.Warn("Unable to load footer visibility from app settings", "err", err)
		return
	}

	if settings != nil {
		ui.SetShowVersionFooter(settings.ShowVersionFooter)
	}
}

func configureMiddleware(app *aile.App, logger *slog.Logger, compressionService *compression.Service, csrfService *csrf.Service, authService *auth.Service) {
	app.Use(combine.Middleware(
		aile.Recovery(),
		requestid.Middleware(requestid.Config{
			Header: "X-Request-ID",
		}),
		xlogger.Middleware(logger),
		compressionService.Middleware(),
		csrfService.Middleware(),
		authService.Middleware(),
	))
}

func registerLifecycle(app *aile.App, logger *slog.Logger, db *sql.DB, cfg appConfig, services appServices) {
	backgroundCtx, backgroundCancel := context.WithCancel(context.Background())

	app.OnStart(func(ctx context.Context, st *aile.State) error {
		go services.observability.Start(backgroundCtx)
		go services.finance.Start(backgroundCtx)
		go services.uptime.Start(backgroundCtx)

		logger.Info("Server started",
			"addr", app.Addr(),
			"db_path", cfg.DBPath,
			"version", buildVersion,
		)
		return nil
	})

	app.OnShutdown(func(ctx context.Context, st *aile.State) error {
		logger.Info("Stopping server")
		backgroundCancel()

		if err := db.Close(); err != nil {
			logger.Error("Failed to close database", "err", err)
			return err
		}

		logger.Info("Database shutdown successful")
		return nil
	})
}
