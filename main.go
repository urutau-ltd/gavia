package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"

	"codeberg.org/urutau-ltd/aile"
	"codeberg.org/urutau-ltd/aile/x/combine"
	xlogger "codeberg.org/urutau-ltd/aile/x/logger"
	requestid "codeberg.org/urutau-ltd/aile/x/request_id"
	"codeberg.org/urutau-ltd/gavia/internal/database"
	"codeberg.org/urutau-ltd/gavia/internal/ui/features/dashboard"
	"codeberg.org/urutau-ltd/gavia/internal/ui/features/providers"
)

//go:embed static
var staticFS embed.FS

//go:embed internal/ui
var UIFS embed.FS

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	app, err := aile.New(aile.WithAddr(":9091"))
	if err != nil {
		log.Fatal(err)
	}

	// Database
	dbConn, err := database.Client("./app.sqlite")
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

	app.GET("/dashboard", dashboard.NewHandler(logger, uiRoot, dbConn).Index)
	app.GET("/providers", providers.NewHandler(logger, uiRoot, dbConn).Index)

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
