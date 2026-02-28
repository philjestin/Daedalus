package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/philjestin/daedalus/internal/api"
	"github.com/philjestin/daedalus/internal/database"
	"github.com/philjestin/daedalus/internal/printer"
	"github.com/philjestin/daedalus/internal/realtime"
	"github.com/philjestin/daedalus/internal/repository"
	"github.com/philjestin/daedalus/internal/service"
	"github.com/philjestin/daedalus/internal/storage"
	"github.com/philjestin/daedalus/internal/version"
	"github.com/joho/godotenv"
)

// App struct holds the application state
type App struct {
	ctx         context.Context
	server      *http.Server
	db          *sql.DB
	dbPath      string
	hub         *realtime.Hub
	printerMgr  *printer.Manager
	services    *service.Services
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Load .env file if present
	_ = godotenv.Load()

	// Initialize Sentry for crash reporting
	if dsn := os.Getenv("SENTRY_DSN"); dsn != "" {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn:              dsn,
			Release:          version.Version,
			Environment:      getEnv("ENVIRONMENT", "development"),
			TracesSampleRate: 0.2,
		}); err != nil {
			slog.Error("failed to initialize Sentry", "error", err)
		} else {
			slog.Info("Sentry initialized", "environment", getEnv("ENVIRONMENT", "development"))
		}
	}

	// Configure structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Get configuration from environment
	port := getEnv("PORT", "8080")
	uploadDir := getEnv("UPLOAD_DIR", "./uploads")

	// Etsy OAuth configuration (optional)
	etsyClientID := os.Getenv("ETSY_CLIENT_ID")
	etsyRedirectURI := getEnv("ETSY_REDIRECT_URI", "http://localhost:8080/api/integrations/etsy/callback")

	// Open SQLite database
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		var err error
		dbPath, err = database.DefaultDBPath()
		if err != nil {
			slog.Error("failed to get default database path", "error", err)
			return
		}
	}

	db, err := database.Open(dbPath)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		return
	}
	a.db = db
	a.dbPath = dbPath

	// Initialize storage
	fileStorage := storage.NewLocalStorage(uploadDir)

	// Initialize repositories
	repos := repository.NewRepositories(db)

	// Initialize WebSocket hub for real-time updates
	a.hub = realtime.NewHub()
	go a.hub.Run()

	// Initialize printer manager with hub for broadcasting state changes
	printerManager := printer.NewManager()
	printerManager.SetBroadcaster(a.hub)
	a.printerMgr = printerManager

	// Initialize services
	servicesConfig := service.ServicesConfig{
		Etsy: service.EtsyConfig{
			ClientID:    etsyClientID,
			RedirectURI: etsyRedirectURI,
		},
	}
	a.services = service.NewServicesWithConfig(repos, fileStorage, printerManager, a.hub, servicesConfig)

	// Initialize backup service with database access
	backupService := service.NewBackupService(db, a.dbPath)
	a.services.SetBackupService(backupService)
	backupService.SetSettingsService(a.services.Settings)
	backupService.StartScheduler()

	// Initialize PrintJobService
	a.services.PrintJobs.Init()

	// Reconnect all saved printers at startup
	a.services.Printers.ConnectAllPrinters(context.Background())

	if etsyClientID != "" {
		slog.Info("Etsy integration enabled", "redirect_uri", etsyRedirectURI)
	}

	// Initialize HTTP router
	router := api.NewRouter(a.services, a.hub)

	// Create HTTP server
	a.server = &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		slog.Info("starting API server", "version", version.String(), "port", port)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
		}
	}()
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
	slog.Info("shutting down...")

	// Shutdown HTTP server first (stop accepting new requests)
	if a.server != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if err := a.server.Shutdown(shutdownCtx); err != nil {
			slog.Error("server shutdown error", "error", err)
		}
		slog.Info("HTTP server stopped")
	}

	// Stop backup scheduler
	if a.services != nil && a.services.Backup != nil {
		a.services.Backup.StopScheduler()
	}

	// Disconnect all printers (closes MQTT connections)
	if a.printerMgr != nil {
		a.printerMgr.DisconnectAll()
	}

	// Stop WebSocket hub (closes all client connections)
	if a.hub != nil {
		a.hub.Stop()
	}

	// Close database last (after all operations complete)
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			slog.Error("database close error", "error", err)
		}
		slog.Info("database closed")
	}

	// Flush Sentry events
	sentry.Flush(2 * time.Second)

	slog.Info("shutdown complete")
}

// GetVersion returns the application version string for the frontend.
func (a *App) GetVersion() string {
	return version.String()
}

// GetAPIURL returns the API URL for the frontend
func (a *App) GetAPIURL() string {
	port := getEnv("PORT", "8080")
	return fmt.Sprintf("http://localhost:%s", port)
}

// getEnv returns environment variable or default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
