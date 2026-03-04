package service

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/philjestin/daedalus/internal/database"
	"github.com/philjestin/daedalus/internal/repository"
)

// openFileTestDB opens a file-based SQLite database in a temp directory.
// VACUUM INTO doesn't work with :memory: databases.
func openFileTestDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db, dbPath
}

func TestCreateBackupWithPrefix(t *testing.T) {
	db, dbPath := openFileTestDB(t)
	svc := NewBackupService(db, dbPath)
	ctx := context.Background()

	// Create a backup with custom prefix
	info, err := svc.CreateBackupWithPrefix(ctx, "auto_startup")
	if err != nil {
		t.Fatalf("CreateBackupWithPrefix failed: %v", err)
	}

	if info == nil {
		t.Fatal("expected backup info, got nil")
	}

	if info.Size == 0 {
		t.Error("backup should have non-zero size")
	}

	// Verify the file exists
	if _, err := os.Stat(info.Path); os.IsNotExist(err) {
		t.Errorf("backup file does not exist: %s", info.Path)
	}

	// Verify prefix is in the name
	if got := info.Name; got[:len("auto_startup_")] != "auto_startup_" {
		t.Errorf("expected name to start with 'auto_startup_', got %q", got)
	}
}

func TestCreateBackupWithPrefix_scheduled(t *testing.T) {
	db, dbPath := openFileTestDB(t)
	svc := NewBackupService(db, dbPath)
	ctx := context.Background()

	info, err := svc.CreateBackupWithPrefix(ctx, "scheduled")
	if err != nil {
		t.Fatalf("CreateBackupWithPrefix failed: %v", err)
	}

	if got := info.Name; got[:len("scheduled_")] != "scheduled_" {
		t.Errorf("expected name to start with 'scheduled_', got %q", got)
	}
}

func TestEnforceRetention_deletesOld(t *testing.T) {
	db, dbPath := openFileTestDB(t)
	svc := NewBackupService(db, dbPath)

	// Set up settings service with retention count = 2
	settingsRepo := newSettingsRepo(t, db)
	settingsSvc := &SettingsService{repo: settingsRepo}
	svc.SetSettingsService(settingsSvc)

	ctx := context.Background()

	// Set retention to 2
	if err := settingsSvc.Set(ctx, "backup_retention_count", "2"); err != nil {
		t.Fatalf("set retention count: %v", err)
	}

	// Create 4 auto_startup backups with different timestamps
	for i := 0; i < 4; i++ {
		name := fmt.Sprintf("auto_startup_2025-01-0%d_12-00-00.db", i+1)
		path := filepath.Join(svc.backupDir, name)
		if err := os.MkdirAll(svc.backupDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
		// Set modification time so they sort correctly (newest first)
		modTime := time.Date(2025, 1, i+1, 12, 0, 0, 0, time.UTC)
		os.Chtimes(path, modTime, modTime)
	}

	// Create a manual backup — should NOT be deleted
	manualPath := filepath.Join(svc.backupDir, "backup_2025-01-01_12-00-00.db")
	if err := os.WriteFile(manualPath, []byte("manual"), 0644); err != nil {
		t.Fatal(err)
	}

	// Enforce retention
	if err := svc.EnforceRetention(ctx); err != nil {
		t.Fatalf("EnforceRetention failed: %v", err)
	}

	// Check: should have 2 auto_startup backups left (newest ones)
	entries, err := os.ReadDir(svc.backupDir)
	if err != nil {
		t.Fatalf("read backup dir: %v", err)
	}

	autoCount := 0
	manualExists := false
	for _, e := range entries {
		if e.Name() == "backup_2025-01-01_12-00-00.db" {
			manualExists = true
		}
		if len(e.Name()) > 13 && e.Name()[:13] == "auto_startup_" {
			autoCount++
		}
	}

	if autoCount != 2 {
		t.Errorf("expected 2 auto_startup backups after retention, got %d", autoCount)
	}
	if !manualExists {
		t.Error("manual backup should not be deleted by retention policy")
	}
}

func TestEnforceRetention_unlimitedKeepsAll(t *testing.T) {
	db, dbPath := openFileTestDB(t)
	svc := NewBackupService(db, dbPath)

	settingsRepo := newSettingsRepo(t, db)
	settingsSvc := &SettingsService{repo: settingsRepo}
	svc.SetSettingsService(settingsSvc)

	ctx := context.Background()

	// Set retention to 0 (unlimited)
	if err := settingsSvc.Set(ctx, "backup_retention_count", "0"); err != nil {
		t.Fatalf("set retention count: %v", err)
	}

	// Create 5 auto_startup backups
	if err := os.MkdirAll(svc.backupDir, 0755); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		name := fmt.Sprintf("auto_startup_2025-01-0%d_12-00-00.db", i+1)
		path := filepath.Join(svc.backupDir, name)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Enforce retention — should keep all
	if err := svc.EnforceRetention(ctx); err != nil {
		t.Fatalf("EnforceRetention failed: %v", err)
	}

	entries, err := os.ReadDir(svc.backupDir)
	if err != nil {
		t.Fatalf("read backup dir: %v", err)
	}

	count := 0
	for _, e := range entries {
		if len(e.Name()) > 13 && e.Name()[:13] == "auto_startup_" {
			count++
		}
	}
	if count != 5 {
		t.Errorf("expected 5 auto_startup backups with unlimited retention, got %d", count)
	}
}

func TestGetConfig_defaults(t *testing.T) {
	db, dbPath := openFileTestDB(t)
	svc := NewBackupService(db, dbPath)

	settingsRepo := newSettingsRepo(t, db)
	settingsSvc := &SettingsService{repo: settingsRepo}
	svc.SetSettingsService(settingsSvc)

	ctx := context.Background()
	config := svc.GetConfig(ctx)

	if !config.AutoOnStartup {
		t.Error("default AutoOnStartup should be true")
	}
	if config.ScheduleEnabled {
		t.Error("default ScheduleEnabled should be false")
	}
	if config.ScheduleInterval != "daily" {
		t.Errorf("default ScheduleInterval should be 'daily', got %q", config.ScheduleInterval)
	}
	if config.RetentionCount != 10 {
		t.Errorf("default RetentionCount should be 10, got %d", config.RetentionCount)
	}
}

func TestUpdateConfig(t *testing.T) {
	db, dbPath := openFileTestDB(t)
	svc := NewBackupService(db, dbPath)

	settingsRepo := newSettingsRepo(t, db)
	settingsSvc := &SettingsService{repo: settingsRepo}
	svc.SetSettingsService(settingsSvc)

	ctx := context.Background()

	newConfig := BackupConfig{
		AutoOnStartup:    false,
		ScheduleEnabled:  true,
		ScheduleInterval: "weekly",
		RetentionCount:   5,
	}

	if err := svc.UpdateConfig(ctx, newConfig); err != nil {
		t.Fatalf("UpdateConfig failed: %v", err)
	}

	// Read back
	got := svc.GetConfig(ctx)

	if got.AutoOnStartup != false {
		t.Error("expected AutoOnStartup to be false")
	}
	if got.ScheduleEnabled != true {
		t.Error("expected ScheduleEnabled to be true")
	}
	if got.ScheduleInterval != "weekly" {
		t.Errorf("expected ScheduleInterval 'weekly', got %q", got.ScheduleInterval)
	}
	if got.RetentionCount != 5 {
		t.Errorf("expected RetentionCount 5, got %d", got.RetentionCount)
	}
}

func TestUpdateConfig_invalidInterval(t *testing.T) {
	db, dbPath := openFileTestDB(t)
	svc := NewBackupService(db, dbPath)

	settingsRepo := newSettingsRepo(t, db)
	settingsSvc := &SettingsService{repo: settingsRepo}
	svc.SetSettingsService(settingsSvc)

	ctx := context.Background()

	err := svc.UpdateConfig(ctx, BackupConfig{
		ScheduleInterval: "monthly",
		RetentionCount:   5,
	})

	if err == nil {
		t.Error("expected error for invalid interval")
	}
}

func TestEnforceRetention_scheduledSeparateFromStartup(t *testing.T) {
	db, dbPath := openFileTestDB(t)
	svc := NewBackupService(db, dbPath)

	settingsRepo := newSettingsRepo(t, db)
	settingsSvc := &SettingsService{repo: settingsRepo}
	svc.SetSettingsService(settingsSvc)

	ctx := context.Background()

	// Set retention to 1
	if err := settingsSvc.Set(ctx, "backup_retention_count", "1"); err != nil {
		t.Fatalf("set retention count: %v", err)
	}

	if err := os.MkdirAll(svc.backupDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create 3 auto_startup and 3 scheduled backups
	for i := 0; i < 3; i++ {
		for _, prefix := range []string{"auto_startup", "scheduled"} {
			name := fmt.Sprintf("%s_2025-01-0%d_12-00-00.db", prefix, i+1)
			path := filepath.Join(svc.backupDir, name)
			if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
				t.Fatal(err)
			}
			modTime := time.Date(2025, 1, i+1, 12, 0, 0, 0, time.UTC)
			os.Chtimes(path, modTime, modTime)
		}
	}

	if err := svc.EnforceRetention(ctx); err != nil {
		t.Fatalf("EnforceRetention failed: %v", err)
	}

	entries, err := os.ReadDir(svc.backupDir)
	if err != nil {
		t.Fatalf("read backup dir: %v", err)
	}

	startupCount, scheduledCount := 0, 0
	for _, e := range entries {
		if len(e.Name()) > 13 && e.Name()[:13] == "auto_startup_" {
			startupCount++
		}
		if len(e.Name()) > 10 && e.Name()[:10] == "scheduled_" {
			scheduledCount++
		}
	}

	if startupCount != 1 {
		t.Errorf("expected 1 auto_startup backup, got %d", startupCount)
	}
	if scheduledCount != 1 {
		t.Errorf("expected 1 scheduled backup, got %d", scheduledCount)
	}
}

// newSettingsRepo creates a SettingsRepository backed by the given db.
func newSettingsRepo(t *testing.T, db *sql.DB) *repository.SettingsRepository {
	t.Helper()
	repos := repository.NewRepositories(db)
	return repos.Settings
}
