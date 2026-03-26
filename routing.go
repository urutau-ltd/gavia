package main

import (
	"net/http"

	"codeberg.org/urutau-ltd/aile/v2"
	"codeberg.org/urutau-ltd/aile/v2/x/resource"
)

type dashboardRoutes interface {
	Index(http.ResponseWriter, *http.Request)
}

type singletonSettingsRoutes interface {
	Show(http.ResponseWriter, *http.Request)
	Edit(http.ResponseWriter, *http.Request)
	Update(http.ResponseWriter, *http.Request)
}

type loginRoutes interface {
	Show(http.ResponseWriter, *http.Request)
	Submit(http.ResponseWriter, *http.Request)
}

type logoutRoutes interface {
	Perform(http.ResponseWriter, *http.Request)
}

type uptimeRoutes interface {
	Index(http.ResponseWriter, *http.Request)
	Show(http.ResponseWriter, *http.Request)
	Create(http.ResponseWriter, *http.Request)
	Update(http.ResponseWriter, *http.Request)
	Delete(http.ResponseWriter, *http.Request)
}

type appSettingsRoutes interface {
	singletonSettingsRoutes
	Export(http.ResponseWriter, *http.Request)
	Import(http.ResponseWriter, *http.Request)
	CreateExpense(http.ResponseWriter, *http.Request)
	DeleteExpense(http.ResponseWriter, *http.Request)
}

type backupAPIRoutes interface {
	Export(http.ResponseWriter, *http.Request)
	Import(http.ResponseWriter, *http.Request)
}

type dashboardAPIRoutes interface {
	Summary(http.ResponseWriter, *http.Request)
}

type licensesRoutes interface {
	Index(http.ResponseWriter, *http.Request)
}

func mountRoutes(app *aile.App, handlers appHandlers) error {
	app.GET("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	})
	app.GET("/dashboard", handlers.dashboard.Index)
	app.GET("/login", handlers.login.Show)
	app.POST("/login", handlers.login.Submit)
	app.GET("/logout", handlers.logout.Perform)
	app.POST("/logout", handlers.logout.Perform)

	if err := resource.MountCollection(app, "/providers", handlers.provider); err != nil {
		return err
	}

	if err := resource.MountCollection(app, "/locations", handlers.location); err != nil {
		return err
	}

	if err := resource.MountCollection(app, "/os", handlers.os); err != nil {
		return err
	}

	if err := resource.MountCollection(app, "/ips", handlers.ip); err != nil {
		return err
	}

	if err := resource.MountCollection(app, "/dns", handlers.dns); err != nil {
		return err
	}

	if err := resource.MountCollection(app, "/labels", handlers.label); err != nil {
		return err
	}

	if err := resource.MountCollection(app, "/domains", handlers.domain); err != nil {
		return err
	}

	if err := resource.MountCollection(app, "/hostings", handlers.hosting); err != nil {
		return err
	}

	if err := resource.MountCollection(app, "/servers", handlers.server); err != nil {
		return err
	}

	if err := resource.MountCollection(app, "/subscriptions", handlers.subscription); err != nil {
		return err
	}

	if err := resource.MountSingleton(app, "/account-settings", handlers.accountSettings); err != nil {
		return err
	}

	if err := resource.MountSingleton(app, "/app-settings", handlers.appSettings); err != nil {
		return err
	}

	app.GET("/app-settings/export", handlers.appSettings.Export)
	app.POST("/app-settings/import", handlers.appSettings.Import)
	app.POST("/app-settings/expenses", handlers.appSettings.CreateExpense)
	app.POST("/app-settings/expenses/{id}/delete", handlers.appSettings.DeleteExpense)

	app.GET("/api/v1/backup/export", handlers.backupAPI.Export)
	app.POST("/api/v1/backup/import", handlers.backupAPI.Import)
	app.GET("/api/v1/dashboard/summary", handlers.dashboardAPI.Summary)
	app.GET("/licenses", handlers.licenses.Index)

	app.GET("/uptime", handlers.uptime.Index)
	app.GET("/uptime/{id}", handlers.uptime.Show)
	app.POST("/uptime", handlers.uptime.Create)
	app.POST("/uptime/{id}/edit", handlers.uptime.Update)
	app.POST("/uptime/{id}/delete", handlers.uptime.Delete)

	return nil
}
