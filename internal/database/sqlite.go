package database

import (
	"database/sql"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	_ "github.com/glebarez/go-sqlite"
)

//go:embed schema.sql
var schemaSQL string

// DefaultDBPath returns the default database path (~/.daedalus/daedalus.db).
func DefaultDBPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ".daedalus", "daedalus.db"), nil
}

// Open opens or creates a SQLite database at the given path.
// It configures WAL mode, foreign keys, and busy timeout.
func Open(path string) (*sql.DB, error) {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Single connection for SQLite to ensure PRAGMAs persist
	db.SetMaxOpenConns(1)

	// Configure connection
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set busy timeout: %w", err)
	}

	// Run schema
	if err := RunMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	slog.Info("database opened", "path", path)
	return db, nil
}

// RunMigrations applies the embedded schema to the database.
func RunMigrations(db *sql.DB) error {
	_, err := db.Exec(schemaSQL)
	if err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}

	// Add columns that may not exist in older databases.
	// SQLite doesn't support ADD COLUMN IF NOT EXISTS, so we ignore errors.
	alterStatements := []string{
		`ALTER TABLE printers ADD COLUMN serial_number TEXT DEFAULT ''`,
		`ALTER TABLE printers ADD COLUMN cost_per_hour_cents INTEGER DEFAULT 0`,
		`ALTER TABLE printers ADD COLUMN purchase_price_cents INTEGER DEFAULT 0`,
		`ALTER TABLE print_jobs ADD COLUMN printer_time_cost_cents INTEGER`,
		`ALTER TABLE print_jobs ADD COLUMN material_cost_cents INTEGER`,
		`ALTER TABLE project_supplies ADD COLUMN material_id TEXT REFERENCES materials(id)`,
		// Auto-dispatch feature columns
		`ALTER TABLE print_jobs ADD COLUMN priority INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE print_jobs ADD COLUMN auto_dispatch_enabled INTEGER NOT NULL DEFAULT 1`,
		// Low-spool threshold (Phase 1)
		`ALTER TABLE materials ADD COLUMN low_threshold_grams INTEGER NOT NULL DEFAULT 100`,
		// Unified orders (Phase 2) - link projects to orders
		`ALTER TABLE projects ADD COLUMN order_id TEXT REFERENCES orders(id)`,
		`ALTER TABLE projects ADD COLUMN order_item_id TEXT REFERENCES order_items(id)`,
		// Link Etsy receipts to unified orders
		`ALTER TABLE etsy_receipts ADD COLUMN order_id TEXT REFERENCES orders(id)`,
		// Link Squarespace orders to unified orders
		`ALTER TABLE squarespace_orders ADD COLUMN order_id TEXT REFERENCES orders(id)`,
	}
	for _, stmt := range alterStatements {
		db.Exec(stmt) // Ignore error if column already exists
	}

	return nil
}
