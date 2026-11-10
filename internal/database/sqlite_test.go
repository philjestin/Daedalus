package database

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultDBPath_freshHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := DefaultDBPath()
	if err != nil {
		t.Fatalf("DefaultDBPath: %v", err)
	}
	want := filepath.Join(home, dataDirName, dbFileName)
	if got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
	if _, err := os.Stat(filepath.Join(home, legacyDataDirName)); !os.IsNotExist(err) {
		t.Fatalf("legacy directory should not be created, stat err = %v", err)
	}
}

func TestDefaultDBPath_migratesLegacyDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	legacyDir := filepath.Join(home, legacyDataDirName)
	if err := os.MkdirAll(filepath.Join(legacyDir, "backups"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(legacyDir, legacyDBFileName), "legacy-db")
	writeFile(t, filepath.Join(legacyDir, legacyDBFileName+"-wal"), "legacy-wal")
	writeFile(t, filepath.Join(legacyDir, legacyDBFileName+"-shm"), "legacy-shm")
	writeFile(t, filepath.Join(legacyDir, "backups", "auto_startup_old.db"), "backup")
	writeFile(t, filepath.Join(legacyDir, "notes.txt"), "keep-me")

	got, err := DefaultDBPath()
	if err != nil {
		t.Fatalf("DefaultDBPath: %v", err)
	}
	newDir := filepath.Join(home, dataDirName)
	want := filepath.Join(newDir, dbFileName)
	if got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
	if _, err := os.Stat(legacyDir); !os.IsNotExist(err) {
		t.Fatalf("legacy directory still present: %v", err)
	}
	assertFile(t, want, "legacy-db")
	assertFile(t, filepath.Join(newDir, dbFileName+"-wal"), "legacy-wal")
	assertFile(t, filepath.Join(newDir, dbFileName+"-shm"), "legacy-shm")
	assertFile(t, filepath.Join(newDir, "backups", "auto_startup_old.db"), "backup")
	assertFile(t, filepath.Join(newDir, "notes.txt"), "keep-me")
}

func TestDefaultDBPath_leavesExistingDatabase(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	newDir := filepath.Join(home, dataDirName)
	legacyDir := filepath.Join(home, legacyDataDirName)
	if err := os.MkdirAll(newDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(newDir, dbFileName), "current")
	writeFile(t, filepath.Join(legacyDir, legacyDBFileName), "legacy")

	got, err := DefaultDBPath()
	if err != nil {
		t.Fatalf("DefaultDBPath: %v", err)
	}
	want := filepath.Join(newDir, dbFileName)
	if got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
	assertFile(t, want, "current")
	assertFile(t, filepath.Join(legacyDir, legacyDBFileName), "legacy")
}

func TestDefaultDBPath_movesDatabaseIntoExistingDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	newDir := filepath.Join(home, dataDirName)
	legacyDir := filepath.Join(home, legacyDataDirName)
	if err := os.MkdirAll(newDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(legacyDir, "backups"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(newDir, "other.txt"), "stay")
	writeFile(t, filepath.Join(legacyDir, legacyDBFileName), "legacy-db")
	writeFile(t, filepath.Join(legacyDir, legacyDBFileName+"-wal"), "legacy-wal")
	writeFile(t, filepath.Join(legacyDir, "backups", "auto.db"), "backup")

	got, err := DefaultDBPath()
	if err != nil {
		t.Fatalf("DefaultDBPath: %v", err)
	}
	want := filepath.Join(newDir, dbFileName)
	if got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
	assertFile(t, want, "legacy-db")
	assertFile(t, filepath.Join(newDir, dbFileName+"-wal"), "legacy-wal")
	assertFile(t, filepath.Join(newDir, "other.txt"), "stay")
	assertFile(t, filepath.Join(newDir, "backups", "auto.db"), "backup")
	if _, err := os.Stat(filepath.Join(legacyDir, legacyDBFileName)); !os.IsNotExist(err) {
		t.Fatalf("legacy database still present: %v", err)
	}
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertFile(t *testing.T, path, contents string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(data) != contents {
		t.Fatalf("%s = %q, want %q", path, data, contents)
	}
}
