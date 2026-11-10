package database

import (
	"database/sql"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/glebarez/go-sqlite"
)

//go:embed schema.sql
var schemaSQL string

const (
	dataDirName = ".rankos"
	dbFileName  = "rankos.db"

	// Legacy locations from before the RankOS rebrand. Startup moves this
	// data when the new database does not already exist. An explicit
	// DATABASE_PATH is left untouched.
	legacyDataDirName = ".daedalus"
	legacyDBFileName  = "daedalus.db"
)

// DefaultDBPath returns the default database path (~/.rankos/rankos.db).
// A previous install's data directory is migrated into that location when
// the RankOS database is not already present.
func DefaultDBPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}

	newPath := filepath.Join(home, dataDirName, dbFileName)
	legacyPath := filepath.Join(home, legacyDataDirName, legacyDBFileName)

	if err := migrateLegacyDataDir(home); err != nil {
		slog.Error("failed to migrate legacy data directory", "error", err)
		if !fileExists(newPath) && fileExists(legacyPath) {
			return legacyPath, nil
		}
		leftover := filepath.Join(home, dataDirName, legacyDBFileName)
		if !fileExists(newPath) && fileExists(leftover) {
			return leftover, nil
		}
	}
	return newPath, nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// migrateLegacyDataDir moves the previous data directory to ~/.rankos when
// rankos.db does not already exist. If ~/.rankos is already present, only
// the database files (and backups, when that directory is free) are moved.
func migrateLegacyDataDir(home string) error {
	legacyDir := filepath.Join(home, legacyDataDirName)
	newDir := filepath.Join(home, dataDirName)
	legacyDB := filepath.Join(legacyDir, legacyDBFileName)
	newDB := filepath.Join(newDir, dbFileName)

	if !fileExists(legacyDB) || fileExists(newDB) {
		return nil
	}

	if _, err := os.Stat(newDir); os.IsNotExist(err) {
		if err := os.Rename(legacyDir, newDir); err != nil {
			return fmt.Errorf("rename data directory: %w", err)
		}
		if err := relocateLegacyDBFiles(newDir, newDir); err != nil {
			return err
		}
		slog.Info("migrated legacy data directory", "from", legacyDir, "to", newDir)
		return nil
	} else if err != nil {
		return fmt.Errorf("stat data directory: %w", err)
	}

	if err := relocateLegacyDBFiles(legacyDir, newDir); err != nil {
		return err
	}
	legacyBackups := filepath.Join(legacyDir, "backups")
	newBackups := filepath.Join(newDir, "backups")
	if info, err := os.Stat(legacyBackups); err == nil && info.IsDir() {
		if _, err := os.Stat(newBackups); os.IsNotExist(err) {
			if err := os.Rename(legacyBackups, newBackups); err != nil {
				return fmt.Errorf("move backups: %w", err)
			}
		}
	}
	slog.Info("migrated legacy database into existing data directory", "to", newDir)
	return nil
}

// relocateLegacyDBFiles renames the legacy database file and its SQLite
// sidecars (wal, shm, journal) to the RankOS names, moving them when the
// directories differ.
func relocateLegacyDBFiles(srcDir, destDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("read data directory: %w", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if name != legacyDBFileName && !strings.HasPrefix(name, legacyDBFileName+"-") {
			continue
		}
		destName := dbFileName + name[len(legacyDBFileName):]
		if err := os.Rename(filepath.Join(srcDir, name), filepath.Join(destDir, destName)); err != nil {
			return fmt.Errorf("rename %s: %w", name, err)
		}
	}
	return nil
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

	// Auto-backup before migrations (if enabled and not a fresh database)
	CreateStartupBackup(db, path)

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
		// Unified orders (Phase 2) - link projects to orders (legacy, kept for backwards compatibility)
		`ALTER TABLE projects ADD COLUMN order_id TEXT REFERENCES orders(id)`,
		`ALTER TABLE projects ADD COLUMN order_item_id TEXT REFERENCES order_items(id)`,
		// Link Etsy receipts to unified orders
		`ALTER TABLE etsy_receipts ADD COLUMN order_id TEXT REFERENCES orders(id)`,
		// Link Squarespace orders to unified orders
		`ALTER TABLE squarespace_orders ADD COLUMN order_id TEXT REFERENCES orders(id)`,
		// Projects as Product Catalog (template-like fields)
		`ALTER TABLE projects ADD COLUMN sku TEXT`,
		`ALTER TABLE projects ADD COLUMN price_cents INTEGER`,
		`ALTER TABLE projects ADD COLUMN printer_type TEXT`,
		`ALTER TABLE projects ADD COLUMN allowed_printer_ids TEXT DEFAULT '[]'`,
		`ALTER TABLE projects ADD COLUMN default_settings TEXT DEFAULT '{}'`,
		`ALTER TABLE projects ADD COLUMN notes TEXT DEFAULT ''`,
		// Tasks (work instances) - task_id in print_jobs
		`ALTER TABLE print_jobs ADD COLUMN task_id TEXT REFERENCES tasks(id)`,
		// Order items link to projects
		`ALTER TABLE order_items ADD COLUMN project_id TEXT REFERENCES projects(id)`,
		// Task pickup/shipping date
		`ALTER TABLE tasks ADD COLUMN pickup_date TEXT`,
		// Quote line items link to projects
		`ALTER TABLE quote_line_items ADD COLUMN project_id TEXT REFERENCES projects(id) ON DELETE SET NULL`,
		// Sales link to customers
		`ALTER TABLE sales ADD COLUMN customer_id TEXT REFERENCES customers(id) ON DELETE SET NULL`,
		// Customer addresses
		`ALTER TABLE customers ADD COLUMN billing_address_json TEXT`,
		`ALTER TABLE customers ADD COLUMN shipping_address_json TEXT`,
		// Quote financial fields
		`ALTER TABLE quotes ADD COLUMN discount_type TEXT DEFAULT 'none'`,
		`ALTER TABLE quotes ADD COLUMN discount_value INTEGER DEFAULT 0`,
		`ALTER TABLE quotes ADD COLUMN rush_fee_cents INTEGER DEFAULT 0`,
		`ALTER TABLE quotes ADD COLUMN tax_rate INTEGER DEFAULT 0`,
		`ALTER TABLE quotes ADD COLUMN terms TEXT`,
		`ALTER TABLE quotes ADD COLUMN requested_due_date TEXT`,
		`ALTER TABLE quotes ADD COLUMN billing_address_json TEXT`,
		`ALTER TABLE quotes ADD COLUMN shipping_address_json TEXT`,
		`ALTER TABLE quotes ADD COLUMN share_token TEXT`,
	}
	for _, stmt := range alterStatements {
		db.Exec(stmt) // Ignore error if column already exists
	}

	// Create indexes that may not exist
	db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_quotes_share_token ON quotes(share_token)`) //nolint:errcheck // best-effort index creation

	return nil
}

// CreateStartupBackup creates an automatic backup before migrations run.
// It reads the backup_auto_on_startup setting directly via raw SQL to avoid
// service layer dependencies. Skips silently on fresh databases (no settings table).
func CreateStartupBackup(db *sql.DB, dbPath string) {
	// Skip in-memory databases (testing)
	if dbPath == ":memory:" || dbPath == "" {
		return
	}

	// Check if settings table exists (fresh databases won't have it)
	var tableName string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='settings'").Scan(&tableName)
	if err != nil {
		return // Fresh database, skip
	}

	// Check if auto-backup on startup is enabled (default: true)
	var value string
	err = db.QueryRow("SELECT value FROM settings WHERE key = 'backup_auto_on_startup'").Scan(&value)
	if err != nil {
		// Setting not found — default is enabled
		value = "true"
	}
	if value != "true" {
		slog.Info("startup backup disabled by setting")
		return
	}

	// Create backup directory
	backupDir := filepath.Join(filepath.Dir(dbPath), "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		slog.Error("failed to create backup directory for startup backup", "error", err)
		return
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	backupPath := filepath.Join(backupDir, fmt.Sprintf("auto_startup_%s.db", timestamp))

	slog.Info("creating startup backup", "path", backupPath)

	_, err = db.Exec(fmt.Sprintf("VACUUM INTO '%s'", backupPath))
	if err != nil {
		slog.Error("failed to create startup backup", "error", err)
		return
	}

	slog.Info("startup backup created", "path", backupPath)
}
