package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hyperion/printfarm/internal/api"
	"github.com/hyperion/printfarm/internal/printer"
	"github.com/hyperion/printfarm/internal/realtime"
	"github.com/hyperion/printfarm/internal/repository"
	"github.com/hyperion/printfarm/internal/service"
	"github.com/hyperion/printfarm/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if present
	_ = godotenv.Load()

	// Configure structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Get configuration from environment
	dbURL := getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/printfarm?sslmode=disable")
	port := getEnv("PORT", "8080")
	uploadDir := getEnv("UPLOAD_DIR", "./uploads")

	// Connect to database
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Verify database connection
	if err := pool.Ping(ctx); err != nil {
		slog.Error("failed to ping database", "error", err)
		os.Exit(1)
	}
	slog.Info("connected to database")

	// Initialize storage
	fileStorage := storage.NewLocalStorage(uploadDir)

	// Initialize repositories
	repos := repository.NewRepositories(pool)

	// Initialize printer manager
	printerManager := printer.NewManager()

	// Initialize WebSocket hub for real-time updates
	hub := realtime.NewHub()
	go hub.Run()

	// Initialize services
	services := service.NewServices(repos, fileStorage, printerManager, hub)

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
		slog.Info("starting server", "port", port)
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

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	slog.Info("server stopped")
}

// getEnv returns environment variable or default value.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

