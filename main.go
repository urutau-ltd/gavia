package main

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"codeberg.org/urutau-ltd/aile/v2"
	"codeberg.org/urutau-ltd/gavia/internal/database"
	"codeberg.org/urutau-ltd/gavia/internal/ui"
)

//go:embed static
var staticFS embed.FS

//go:embed internal/ui
var UIFS embed.FS

var (
	buildVersion   string = "dev"
	buildTag       string = "dev"
	buildCommit    string = "unknown"
	buildDate      string = "unknown"
	upstreamRepo   string = "urutau-ltd/gavia"
	upstreamVendor string = "Urutau Limited"
)

type appConfig struct {
	Addr      string
	DBPath    string
	LogFormat string
	LogColor  string
	LogLevel  slog.Level
}

func loadConfig() appConfig {
	cfg := appConfig{
		Addr:      envOrDefault("GAVIA_ADDR", ":9091"),
		DBPath:    envOrDefault("GAVIA_DB_PATH", "./db/app.sqlite"),
		LogFormat: strings.ToLower(envOrDefault("GAVIA_LOG_FORMAT", "text")),
		LogColor:  strings.ToLower(envOrDefault("GAVIA_LOG_COLOR", "auto")),
		LogLevel:  parseLogLevel(os.Getenv("GAVIA_LOG_LEVEL")),
	}

	if cfg.LogFormat != "json" {
		cfg.LogFormat = "text"
	}
	if cfg.LogColor != "always" && cfg.LogColor != "never" {
		cfg.LogColor = "auto"
	}

	return cfg
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}

func parseLogLevel(raw string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "info":
		return slog.LevelInfo
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func newLogger(cfg appConfig) *slog.Logger {
	opts := &slog.HandlerOptions{
		AddSource: true,
		Level:     cfg.LogLevel,
	}

	if cfg.LogFormat == "json" {
		opts.ReplaceAttr = replaceLogAttr()
		return slog.New(slog.NewJSONHandler(os.Stderr, opts)).With("app", "gavia")
	}

	textWriter := io.Writer(os.Stderr)
	if shouldColorize(cfg.LogColor, os.Stderr) {
		textWriter = colorLevelWriter{next: os.Stderr}
	}

	handler := slog.NewTextHandler(textWriter, &slog.HandlerOptions{
		AddSource:   true,
		Level:       cfg.LogLevel,
		ReplaceAttr: replaceLogAttr(),
	})

	return slog.New(handler).With("app", "gavia")
}

func replaceLogAttr() func([]string, slog.Attr) slog.Attr {
	return func(_ []string, a slog.Attr) slog.Attr {
		switch a.Key {
		case slog.TimeKey:
			if t, ok := a.Value.Any().(time.Time); ok {
				return slog.String("ts", t.Format("2006-01-02 15:04:05"))
			}
			return slog.Attr{}
		case slog.LevelKey:
			if lvl, ok := a.Value.Any().(slog.Level); ok {
				return slog.String("lvl", formatLogLevel(lvl))
			}
			return slog.String("lvl", a.Value.String())
		case slog.MessageKey:
			return slog.Attr{Key: "msg", Value: a.Value}
		case slog.SourceKey:
			if src, ok := a.Value.Any().(*slog.Source); ok && src != nil {
				return slog.String("src", fmt.Sprintf("%s:%d", filepath.Base(src.File), src.Line))
			}
			return slog.Attr{}
		case "request_id":
			return slog.Attr{Key: "rid", Value: a.Value}
		}

		return a
	}
}

type colorLevelWriter struct {
	next io.Writer
}

func (w colorLevelWriter) Write(p []byte) (int, error) {
	colored := bytes.ReplaceAll(p, []byte("lvl=DBG"), []byte("lvl=\033[36mDBG\033[0m"))
	colored = bytes.ReplaceAll(colored, []byte("lvl=INF"), []byte("lvl=\033[32mINF\033[0m"))
	colored = bytes.ReplaceAll(colored, []byte("lvl=WRN"), []byte("lvl=\033[33mWRN\033[0m"))
	colored = bytes.ReplaceAll(colored, []byte("lvl=ERR"), []byte("lvl=\033[31mERR\033[0m"))

	n, err := w.next.Write(colored)
	if err != nil {
		return 0, err
	}

	if n != len(colored) {
		return 0, io.ErrShortWrite
	}

	return len(p), nil
}

func formatLogLevel(level slog.Level) string {
	switch {
	case level <= slog.LevelDebug:
		return "DBG"
	case level <= slog.LevelInfo:
		return "INF"
	case level <= slog.LevelWarn:
		return "WRN"
	default:
		return "ERR"
	}
}

func shouldColorize(mode string, stream *os.File) bool {
	switch mode {
	case "always":
		return true
	case "never":
		return false
	}

	if stream == nil || os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}

	info, err := stream.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeCharDevice != 0
}

func main() {
	cfg := loadConfig()
	logger := newLogger(cfg)
	slog.SetDefault(logger)
	ui.SetAppVersion(buildVersion)

	logger.Info("Bootstrapping runtime",
		"addr", cfg.Addr,
		"db_path", cfg.DBPath,
		"log_format", cfg.LogFormat,
		"log_color", cfg.LogColor,
		"log_level", cfg.LogLevel.String(),
		"version", buildVersion,
		"tag", buildTag,
		"commit", buildCommit,
		"build_date", buildDate,
		"repo", upstreamRepo,
		"vendor", upstreamVendor,
	)

	app, err := aile.New(aile.WithAddr(cfg.Addr))
	if err != nil {
		log.Fatal(err)
	}

	// Database
	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0o755); err != nil {
		logger.Error("Failed to prepare database directory", "path", filepath.Dir(cfg.DBPath), "err", err)
		os.Exit(1)
	}

	dbConn, err := database.Client(cfg.DBPath)
	if err != nil {
		logger.Error("Failed to initialize database", "path", cfg.DBPath, "err", err)
		os.Exit(1)
	}

	if err = dbConn.Ping(); err != nil {
		logger.Error("Failed to ping database", "path", cfg.DBPath, "err", err)
		os.Exit(1)
	}

	err = database.SetPragmas(dbConn)
	if err != nil {
		logger.Error("Unable to set database pragmas", "err", err)
		os.Exit(1)
	}
	logger.Info("Database pragmas loaded")

	if err := database.RunMigrations(dbConn, logger); err != nil {
		logger.Error("Migrations failed", "err", err)
		os.Exit(1)
	}

	logger.Info("Migrations completed")

	if err := database.SeedReferenceData(dbConn); err != nil {
		logger.Error("Reference data seeding failed", "err", err)
	} else {
		logger.Info("Reference data seed completed")
	}

	repositories := newRepositories(dbConn)
	services := newServices(logger, dbConn, repositories)
	applyUISettings(context.Background(), logger, repositories.appSettings)
	configureMiddleware(app, logger, services.compression, services.csrf, services.auth)

	uiRoot, err := fs.Sub(UIFS, "internal/ui")
	if err != nil {
		logger.Error("Error in fs.Sub UI", "err", err)
		os.Exit(1)
	}

	staticRoot, err := fs.Sub(staticFS, "static")
	if err != nil {
		logger.Error("Error in fs.Sub static", "err", err)
		os.Exit(1)
	}

	if err := app.Static("/static", staticRoot); err != nil {
		logger.Error("Error mounting static files", "err", err)
		os.Exit(1)
	}

	handlers := newHandlers(logger, uiRoot, dbConn, repositories, services)
	if err := mountRoutes(app, handlers); err != nil {
		logger.Error("Error mounting routes", "err", err)
		os.Exit(1)
	}

	logger.Info("Routes mounted")
	registerLifecycle(app, logger, dbConn, cfg, services)

	if err := app.Run(context.Background()); err != nil {
		logger.Error("Fatal server error", "err", err)
	}

}
