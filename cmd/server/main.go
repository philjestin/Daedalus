package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/hyperion/printfarm/internal/api"
	"github.com/hyperion/printfarm/internal/database"
	"github.com/hyperion/printfarm/internal/printer"
	"github.com/hyperion/printfarm/internal/realtime"
	"github.com/hyperion/printfarm/internal/repository"
	"github.com/hyperion/printfarm/internal/service"
	"github.com/hyperion/printfarm/internal/storage"
	"github.com/hyperion/printfarm/internal/version"
	"github.com/joho/godotenv"
)

func main() {
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
			defer sentry.Flush(2 * time.Second)
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
			os.Exit(1)
		}
	}

	db, err := database.Open(dbPath)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Initialize storage
	fileStorage := storage.NewLocalStorage(uploadDir)

	// Initialize repositories
	repos := repository.NewRepositories(db)

	// Initialize WebSocket hub for real-time updates
	hub := realtime.NewHub()
	go hub.Run()

	// Initialize printer manager with hub for broadcasting state changes
	printerManager := printer.NewManager()
	printerManager.SetBroadcaster(hub)

	// Initialize services
	servicesConfig := service.ServicesConfig{
		Etsy: service.EtsyConfig{
			ClientID:    etsyClientID,
			RedirectURI: etsyRedirectURI,
		},
	}
	services := service.NewServicesWithConfig(repos, fileStorage, printerManager, hub, servicesConfig)

	// Initialize backup service with database access
	backupService := service.NewBackupService(db, dbPath)
	services.SetBackupService(backupService)
	backupService.SetSettingsService(services.Settings)
	backupService.StartScheduler()

	// Initialize PrintJobService to register for printer status changes (auto failure detection)
	services.PrintJobs.Init()

	// Reconnect all saved printers at startup
	services.Printers.ConnectAllPrinters(context.Background())

	if etsyClientID != "" {
		slog.Info("Etsy integration enabled", "redirect_uri", etsyRedirectURI)
	}

	// Initialize HTTP router
	router := api.NewRouter(services, hub)

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second, // Long timeout for network scanning
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		slog.Info("starting server", "version", version.String(), "port", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server first (stop accepting new requests)
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}
	slog.Info("HTTP server stopped")

	// Stop backup scheduler
	backupService.StopScheduler()

	// Disconnect all printers (closes MQTT connections)
	printerManager.DisconnectAll()

	// Stop WebSocket hub (closes all client connections)
	hub.Stop()

	slog.Info("shutdown complete")
}

// getEnv returns environment variable or default value.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
