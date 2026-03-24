package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"codeberg.org/urutau-ltd/aile"
	"codeberg.org/urutau-ltd/aile/x/combine"
	xlogger "codeberg.org/urutau-ltd/aile/x/logger"
	requestid "codeberg.org/urutau-ltd/aile/x/request_id"
	"codeberg.org/urutau-ltd/gavia/internal/database"
	"codeberg.org/urutau-ltd/gavia/internal/ui/features/dashboard"
	"codeberg.org/urutau-ltd/gavia/internal/ui/features/locations"
	"codeberg.org/urutau-ltd/gavia/internal/ui/features/providers"
)

//go:embed static
var staticFS embed.FS

//go:embed internal/ui
var UIFS embed.FS

func newLogger() *slog.Logger {
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: true,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			switch a.Key {
			case slog.TimeKey:
				if t, ok := a.Value.Any().(time.Time); ok {
					return slog.String("t", t.Format("2006-01-02 15:04:05"))
				}
				return slog.Attr{}
			case slog.LevelKey:
				if lvl, ok := a.Value.Any().(slog.Level); ok {
					switch {
					case lvl <= slog.LevelDebug:
						return slog.String("lvl", "DBG")
					case lvl <= slog.LevelInfo:
						return slog.String("lvl", "INF")
					case lvl <= slog.LevelWarn:
						return slog.String("lvl", "WRN")
					default:
						return slog.String("lvl", "ERR")
					}
				}
				return slog.String("lvl", a.Value.String())
			case slog.MessageKey:
				return slog.Attr{Key: "msg", Value: a.Value}
			case slog.SourceKey:
				if src, ok := a.Value.Any().(*slog.Source); ok {
					return slog.String("src", fmt.Sprintf("%s:%d", filepath.Base(src.File), src.Line))
				}
			}
			return a
		},
	})

	return slog.New(handler).With("app", "gavia")
}

func main() {
	logger := newLogger()
	slog.SetDefault(logger)

	app, err := aile.New(aile.WithAddr(":9091"))
	if err != nil {
		log.Fatal(err)
	}

	// Database
	dbConn, err := database.Client("./db/app.sqlite")
	if err != nil {
		logger.Error("Failed to initialize database", "err", err)
		os.Exit(1)
	}

	if err = dbConn.Ping(); err != nil {
		logger.Error("Failed to ping database. Is the file in your disk?",
			"err",
			err,
		)
		os.Exit(1)
	}

	err = database.SetPragmas(dbConn)
	if err != nil {
		logger.Error("Unable to set database pragmas")
		os.Exit(1)
	}
	logger.Info("Load database pragmas")

	if err := database.RunMigrations(dbConn, logger); err != nil {
		logger.Error("Migrations failed", "err", err)
		os.Exit(1)
	}

	logger.Info("Ran migrations")

	if err := database.SeedProviders(dbConn); err != nil {
		logger.Error("Seeding providers failed", "err", err)
	}

	logger.Info("Database initialized and seeded")

	// Middleware
	app.Use(combine.Middleware(
		aile.Recovery(),
		requestid.Middleware(requestid.Config{
			Header: "X-Request-ID",
		}),
		xlogger.Middleware(logger),
	))

	uiRoot, err := fs.Sub(UIFS, "internal/ui")
	if err != nil {
		logger.Error("Error in fs.Sub UI: ", err)
		os.Exit(1)
	}

	staticRoot, err := fs.Sub(staticFS, "static")
	if err != nil {
		logger.Error("Error in fs.Sub Statics: ", err)
		os.Exit(1)
	}

	app.GET("/static/{path...}", http.StripPrefix("/static/", http.FileServer(http.FS(staticRoot))).ServeHTTP)

	dashboardHandler := dashboard.NewHandler(logger, uiRoot, dbConn)
	providerHandler := providers.NewHandler(logger, uiRoot, dbConn)
	locationHandler := locations.NewHandler(logger, uiRoot, dbConn)

	app.GET("/dashboard", dashboardHandler.Index)

	// providers/ routes
	app.GET("/providers", providerHandler.Index)
	app.GET("/providers/new", providerHandler.New)
	app.POST("/providers", providerHandler.Create)
	app.GET("/providers/{id}", providerHandler.Show)
	app.GET("/providers/{id}/edit", providerHandler.Edit)
	app.POST("/providers/{id}/edit", providerHandler.Update)
	app.DELETE("/providers/{id}", providerHandler.Delete)

	// locations/ routes
	app.GET("/locations", locationHandler.Index)
	app.GET("/locations/new", locationHandler.New)
	app.POST("/locations", locationHandler.Create)
	app.GET("/locations/{id}", locationHandler.Show)
	app.GET("/locations/{id}/edit", locationHandler.Edit)
	app.POST("/locations/{id}/edit", locationHandler.Update)
	app.DELETE("/locations/{id}", locationHandler.Delete)

	logger.Info("Mount routes")

	// Hooks
	app.OnStart(func(ctx context.Context, st *aile.State) error {
		logger.Info("Server started",
			"Address: ", app.Addr())
		return nil
	})

	app.OnShutdown(func(ctx context.Context, st *aile.State) error {
		logger.Info("Stopping server...")
		err := dbConn.Close()
		if err != nil {
			logger.Error("Failed to close database!", "err", err)
			return err
		}
		logger.Info("Database shutdown successful")

		return nil
	})

	if err := app.Run(context.Background()); err != nil {
		logger.Error("Fatal error in server ", "err", err)
	}

}
