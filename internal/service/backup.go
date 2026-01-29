package service

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// BackupInfo represents metadata about a backup file.
type BackupInfo struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
}

// BackupService handles database backup operations.
type BackupService struct {
	db        *sql.DB
	dbPath    string
	backupDir string
}

// NewBackupService creates a new backup service.
func NewBackupService(db *sql.DB, dbPath string) *BackupService {
	// Backup directory is adjacent to the database
	backupDir := filepath.Join(filepath.Dir(dbPath), "backups")
	return &BackupService{
		db:        db,
		dbPath:    dbPath,
		backupDir: backupDir,
	}
}

// ensureBackupDir creates the backup directory if it doesn't exist.
func (s *BackupService) ensureBackupDir() error {
	return os.MkdirAll(s.backupDir, 0755)
}

// CreateBackup creates a new backup of the database.
// Uses SQLite's VACUUM INTO for a consistent, compact backup.
func (s *BackupService) CreateBackup(ctx context.Context) (*BackupInfo, error) {
	if err := s.ensureBackupDir(); err != nil {
		return nil, fmt.Errorf("create backup directory: %w", err)
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	backupName := fmt.Sprintf("backup_%s.db", timestamp)
	backupPath := filepath.Join(s.backupDir, backupName)

	slog.Info("creating database backup", "path", backupPath)

	// Use VACUUM INTO for a clean, consistent backup
	_, err := s.db.ExecContext(ctx, fmt.Sprintf("VACUUM INTO '%s'", backupPath))
	if err != nil {
		return nil, fmt.Errorf("vacuum into backup: %w", err)
	}

	// Get file info for the backup
	info, err := os.Stat(backupPath)
	if err != nil {
		return nil, fmt.Errorf("stat backup file: %w", err)
	}

	slog.Info("backup created successfully", "path", backupPath, "size", info.Size())

	return &BackupInfo{
		Name:      backupName,
		Path:      backupPath,
		Size:      info.Size(),
		CreatedAt: info.ModTime(),
	}, nil
}

// ListBackups returns all available backups, sorted by creation time (newest first).
func (s *BackupService) ListBackups(ctx context.Context) ([]BackupInfo, error) {
	if err := s.ensureBackupDir(); err != nil {
		return nil, fmt.Errorf("ensure backup directory: %w", err)
	}

	entries, err := os.ReadDir(s.backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []BackupInfo{}, nil
		}
		return nil, fmt.Errorf("read backup directory: %w", err)
	}

	var backups []BackupInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".db") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		backups = append(backups, BackupInfo{
			Name:      entry.Name(),
			Path:      filepath.Join(s.backupDir, entry.Name()),
			Size:      info.Size(),
			CreatedAt: info.ModTime(),
		})
	}

	// Sort by creation time, newest first
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	return backups, nil
}

// DeleteBackup deletes a specific backup file.
func (s *BackupService) DeleteBackup(ctx context.Context, name string) error {
	// Validate name to prevent path traversal
	if strings.Contains(name, "/") || strings.Contains(name, "\\") || strings.Contains(name, "..") {
		return fmt.Errorf("invalid backup name")
	}

	backupPath := filepath.Join(s.backupDir, name)

	// Check if file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup not found: %s", name)
	}

	if err := os.Remove(backupPath); err != nil {
		return fmt.Errorf("delete backup: %w", err)
	}

	slog.Info("backup deleted", "name", name)
	return nil
}

// RestoreBackup restores the database from a backup.
// WARNING: This will replace the current database!
// The application should be restarted after a restore.
func (s *BackupService) RestoreBackup(ctx context.Context, name string) error {
	// Validate name to prevent path traversal
	if strings.Contains(name, "/") || strings.Contains(name, "\\") || strings.Contains(name, "..") {
		return fmt.Errorf("invalid backup name")
	}

	backupPath := filepath.Join(s.backupDir, name)

	// Check if backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup not found: %s", name)
	}

	slog.Info("restoring database from backup", "backup", backupPath, "target", s.dbPath)

	// Create a pre-restore backup just in case
	preRestoreBackup := s.dbPath + ".pre-restore"
	if err := copyFile(s.dbPath, preRestoreBackup); err != nil {
		slog.Warn("failed to create pre-restore backup", "error", err)
	}

	// Close the database connection before restore
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("close database: %w", err)
	}

	// Copy backup to database path
	if err := copyFile(backupPath, s.dbPath); err != nil {
		return fmt.Errorf("copy backup to database: %w", err)
	}

	// Remove WAL and SHM files if they exist (they're invalid after restore)
	os.Remove(s.dbPath + "-wal")
	os.Remove(s.dbPath + "-shm")

	slog.Info("database restored from backup - application restart required")

	return nil
}

// GetBackupDir returns the backup directory path.
func (s *BackupService) GetBackupDir() string {
	return s.backupDir
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return destFile.Sync()
}
