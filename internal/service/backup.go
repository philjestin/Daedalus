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
	"strconv"
	"strings"
	"sync"
	"time"
)

// BackupConfig holds automatic backup configuration.
type BackupConfig struct {
	AutoOnStartup    bool   `json:"auto_on_startup"`
	ScheduleEnabled  bool   `json:"schedule_enabled"`
	ScheduleInterval string `json:"schedule_interval"` // "daily" or "weekly"
	RetentionCount   int    `json:"retention_count"`    // 0 = unlimited
}

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

	settings *SettingsService
	stopCh   chan struct{}
	stopOnce sync.Once
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

// SetSettingsService sets the settings service for reading backup configuration.
func (s *BackupService) SetSettingsService(settings *SettingsService) {
	s.settings = settings
}

// CreateBackupWithPrefix creates a backup with a custom filename prefix.
func (s *BackupService) CreateBackupWithPrefix(ctx context.Context, prefix string) (*BackupInfo, error) {
	if err := s.ensureBackupDir(); err != nil {
		return nil, fmt.Errorf("create backup directory: %w", err)
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	backupName := fmt.Sprintf("%s_%s.db", prefix, timestamp)
	backupPath := filepath.Join(s.backupDir, backupName)

	slog.Info("creating database backup", "path", backupPath, "prefix", prefix)

	_, err := s.db.ExecContext(ctx, fmt.Sprintf("VACUUM INTO '%s'", backupPath))
	if err != nil {
		return nil, fmt.Errorf("vacuum into backup: %w", err)
	}

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

// GetConfig returns the current backup configuration from settings.
func (s *BackupService) GetConfig(ctx context.Context) BackupConfig {
	config := BackupConfig{
		AutoOnStartup:    true,
		ScheduleEnabled:  false,
		ScheduleInterval: "daily",
		RetentionCount:   10,
	}

	if s.settings == nil {
		return config
	}

	if v, err := s.settings.Get(ctx, "backup_auto_on_startup"); err == nil && v != nil {
		config.AutoOnStartup = v.Value == "true"
	}
	if v, err := s.settings.Get(ctx, "backup_schedule_enabled"); err == nil && v != nil {
		config.ScheduleEnabled = v.Value == "true"
	}
	if v, err := s.settings.Get(ctx, "backup_schedule_interval"); err == nil && v != nil {
		if v.Value == "daily" || v.Value == "weekly" {
			config.ScheduleInterval = v.Value
		}
	}
	if v, err := s.settings.Get(ctx, "backup_retention_count"); err == nil && v != nil {
		if n, err := strconv.Atoi(v.Value); err == nil && n >= 0 {
			config.RetentionCount = n
		}
	}

	return config
}

// UpdateConfig updates the backup configuration in settings.
func (s *BackupService) UpdateConfig(ctx context.Context, config BackupConfig) error {
	if s.settings == nil {
		return fmt.Errorf("settings service not available")
	}

	if config.ScheduleInterval != "daily" && config.ScheduleInterval != "weekly" {
		return fmt.Errorf("invalid schedule interval: %s", config.ScheduleInterval)
	}
	if config.RetentionCount < 0 {
		return fmt.Errorf("retention count must be >= 0")
	}

	if err := s.settings.Set(ctx, "backup_auto_on_startup", strconv.FormatBool(config.AutoOnStartup)); err != nil {
		return fmt.Errorf("set backup_auto_on_startup: %w", err)
	}
	if err := s.settings.Set(ctx, "backup_schedule_enabled", strconv.FormatBool(config.ScheduleEnabled)); err != nil {
		return fmt.Errorf("set backup_schedule_enabled: %w", err)
	}
	if err := s.settings.Set(ctx, "backup_schedule_interval", config.ScheduleInterval); err != nil {
		return fmt.Errorf("set backup_schedule_interval: %w", err)
	}
	if err := s.settings.Set(ctx, "backup_retention_count", strconv.Itoa(config.RetentionCount)); err != nil {
		return fmt.Errorf("set backup_retention_count: %w", err)
	}

	return nil
}

// EnforceRetention deletes the oldest automatic backups beyond the retention count.
// Manual backups (prefix "backup_") are never deleted.
func (s *BackupService) EnforceRetention(ctx context.Context) error {
	config := s.GetConfig(ctx)
	if config.RetentionCount == 0 {
		return nil // Unlimited retention
	}

	backups, err := s.ListBackups(ctx)
	if err != nil {
		return fmt.Errorf("list backups: %w", err)
	}

	// Separate auto backups by prefix type
	var autoStartup, scheduled []BackupInfo
	for _, b := range backups {
		if strings.HasPrefix(b.Name, "auto_startup_") {
			autoStartup = append(autoStartup, b)
		} else if strings.HasPrefix(b.Name, "scheduled_") {
			scheduled = append(scheduled, b)
		}
		// "backup_" prefix = manual, never auto-deleted
	}

	// Enforce retention for each prefix group independently
	if err := s.deleteOldBackups(autoStartup, config.RetentionCount); err != nil {
		return err
	}
	if err := s.deleteOldBackups(scheduled, config.RetentionCount); err != nil {
		return err
	}

	return nil
}

// deleteOldBackups deletes the oldest backups beyond the retention count.
// Assumes backups are sorted newest-first (from ListBackups).
func (s *BackupService) deleteOldBackups(backups []BackupInfo, retentionCount int) error {
	if len(backups) <= retentionCount {
		return nil
	}

	toDelete := backups[retentionCount:]
	for _, b := range toDelete {
		slog.Info("deleting old backup (retention policy)", "name", b.Name)
		if err := os.Remove(b.Path); err != nil && !os.IsNotExist(err) {
			slog.Error("failed to delete old backup", "name", b.Name, "error", err)
		}
	}

	return nil
}

// StartScheduler starts the background scheduler for periodic backups.
func (s *BackupService) StartScheduler() {
	s.stopCh = make(chan struct{})

	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		slog.Info("backup scheduler started")

		for {
			select {
			case <-ticker.C:
				s.maybeRunScheduledBackup()
			case <-s.stopCh:
				slog.Info("backup scheduler stopped")
				return
			}
		}
	}()
}

// StopScheduler stops the background scheduler.
func (s *BackupService) StopScheduler() {
	s.stopOnce.Do(func() {
		if s.stopCh != nil {
			close(s.stopCh)
		}
	})
}

// maybeRunScheduledBackup checks if a scheduled backup is due and creates one.
func (s *BackupService) maybeRunScheduledBackup() {
	ctx := context.Background()
	config := s.GetConfig(ctx)

	if !config.ScheduleEnabled {
		return
	}

	// Determine the interval duration
	var interval time.Duration
	switch config.ScheduleInterval {
	case "weekly":
		interval = 7 * 24 * time.Hour
	default: // "daily"
		interval = 24 * time.Hour
	}

	// Check last scheduled backup time
	backups, err := s.ListBackups(ctx)
	if err != nil {
		slog.Error("scheduler: failed to list backups", "error", err)
		return
	}

	for _, b := range backups {
		if strings.HasPrefix(b.Name, "scheduled_") {
			if time.Since(b.CreatedAt) < interval {
				return // Not time yet
			}
			break // Found the most recent scheduled backup and it's old enough
		}
	}

	// Create scheduled backup
	slog.Info("scheduler: creating scheduled backup")
	_, err = s.CreateBackupWithPrefix(ctx, "scheduled")
	if err != nil {
		slog.Error("scheduler: failed to create backup", "error", err)
		return
	}

	// Enforce retention after creating backup
	if err := s.EnforceRetention(ctx); err != nil {
		slog.Error("scheduler: failed to enforce retention", "error", err)
	}
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
