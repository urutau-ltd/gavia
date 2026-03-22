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
	"codeberg.org/urutau-ltd/gavia/internal/ui/features/dashboard"
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

	//app.GET("/static/*", http.FileServer(http.FS(staticFS)).ServeHTTP)
	app.GET("/static/{path...}", http.StripPrefix("/static/", http.FileServer(http.FS(staticRoot))).ServeHTTP)

	app.GET("/dashboard", dashboard.NewHandler(logger, uiRoot).Index)

	// Hooks
	app.OnStart(func(ctx context.Context, st *aile.State) error {
		logger.Info("Server started",
			"Address: ", app.Addr())
		return nil
	})

	// Si esto no imprime "css/missing.css", el archivo NO ESTÁ en el binario.
	fs.WalkDir(staticRoot, ".", func(p string, d fs.DirEntry, err error) error {
		slog.Info("FS Check", "path", p)
		return nil
	})

	if err := app.Run(context.Background()); err != nil {
		logger.Error("Fatal error in server ", "err", err)
	}

}
