package storage

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewLocalStorage(t *testing.T) {
	tmpDir := t.TempDir()
	basePath := filepath.Join(tmpDir, "storage")

	// Directory shouldn't exist yet
	if _, err := os.Stat(basePath); !os.IsNotExist(err) {
		t.Fatal("base path should not exist before NewLocalStorage")
	}

	s := NewLocalStorage(basePath)

	// Directory should be created
	info, err := os.Stat(basePath)
	if err != nil {
		t.Fatalf("NewLocalStorage should create directory: %v", err)
	}
	if !info.IsDir() {
		t.Error("base path should be a directory")
	}

	if s.basePath != basePath {
		t.Errorf("basePath = %q, want %q", s.basePath, basePath)
	}
}

func TestLocalStorage_Save(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewLocalStorage(tmpDir)

	content := "hello world"
	reader := strings.NewReader(content)

	storagePath, hash, size, err := s.Save("test.txt", reader)
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	// Verify size
	if size != int64(len(content)) {
		t.Errorf("size = %d, want %d", size, len(content))
	}

	// Verify hash is correct SHA-256
	expectedHash := sha256.Sum256([]byte(content))
	expectedHashStr := hex.EncodeToString(expectedHash[:])
	if hash != expectedHashStr {
		t.Errorf("hash = %q, want %q", hash, expectedHashStr)
	}

	// Verify storage path format: {hash[0:2]}/{hash[2:4]}/{hash}/filename
	expectedPath := filepath.Join(hash[0:2], hash[2:4], hash, "test.txt")
	if storagePath != expectedPath {
		t.Errorf("storagePath = %q, want %q", storagePath, expectedPath)
	}

	// Verify file was actually saved
	fullPath := filepath.Join(tmpDir, storagePath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}
	if string(data) != content {
		t.Errorf("saved content = %q, want %q", string(data), content)
	}
}

func TestLocalStorage_Save_LargeFile(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewLocalStorage(tmpDir)

	// Create 1MB of data
	size := 1024 * 1024
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}
	reader := bytes.NewReader(data)

	storagePath, hash, savedSize, err := s.Save("large.bin", reader)
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	if savedSize != int64(size) {
		t.Errorf("savedSize = %d, want %d", savedSize, size)
	}

	// Verify hash
	expectedHash := sha256.Sum256(data)
	expectedHashStr := hex.EncodeToString(expectedHash[:])
	if hash != expectedHashStr {
		t.Errorf("hash mismatch for large file")
	}

	// Verify content
	fullPath := filepath.Join(tmpDir, storagePath)
	savedData, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}
	if !bytes.Equal(savedData, data) {
		t.Error("saved data doesn't match original")
	}
}

func TestLocalStorage_Save_ContentAddressed(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewLocalStorage(tmpDir)

	content := "identical content"

	// Save same content twice with different filenames
	path1, hash1, _, err := s.Save("file1.txt", strings.NewReader(content))
	if err != nil {
		t.Fatalf("Save 1 returned error: %v", err)
	}

	path2, hash2, _, err := s.Save("file2.txt", strings.NewReader(content))
	if err != nil {
		t.Fatalf("Save 2 returned error: %v", err)
	}

	// Hashes should be identical (same content)
	if hash1 != hash2 {
		t.Errorf("hashes should be identical for same content: %q != %q", hash1, hash2)
	}

	// Paths should be different (different filenames)
	if path1 == path2 {
		t.Errorf("paths should be different: both are %q", path1)
	}

	// But the directory portion should be the same
	dir1 := filepath.Dir(path1)
	dir2 := filepath.Dir(path2)
	if dir1 != dir2 {
		t.Errorf("directory portions should be the same: %q != %q", dir1, dir2)
	}
}

func TestLocalStorage_Get(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewLocalStorage(tmpDir)

	content := "test content for get"
	storagePath, _, _, err := s.Save("test.txt", strings.NewReader(content))
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	reader, err := s.Get(storagePath)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to read from Get: %v", err)
	}

	if string(data) != content {
		t.Errorf("Get content = %q, want %q", string(data), content)
	}
}

func TestLocalStorage_Get_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewLocalStorage(tmpDir)

	_, err := s.Get("nonexistent/path/file.txt")
	if err == nil {
		t.Error("Get should return error for non-existent file")
	}
}

func TestLocalStorage_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewLocalStorage(tmpDir)

	content := "content to delete"
	storagePath, _, _, err := s.Save("delete-me.txt", strings.NewReader(content))
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	// Verify file exists
	fullPath := s.GetFullPath(storagePath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Fatal("file should exist before delete")
	}

	// Delete
	if err := s.Delete(storagePath); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	// Verify file no longer exists
	if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
		t.Error("file should not exist after delete")
	}
}

func TestLocalStorage_Delete_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewLocalStorage(tmpDir)

	err := s.Delete("nonexistent/path/file.txt")
	if err == nil {
		t.Error("Delete should return error for non-existent file")
	}
}

func TestLocalStorage_GetFullPath(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewLocalStorage(tmpDir)

	testCases := []struct {
		storagePath string
		want        string
	}{
		{"ab/cd/abcdef/file.txt", filepath.Join(tmpDir, "ab/cd/abcdef/file.txt")},
		{"file.txt", filepath.Join(tmpDir, "file.txt")},
		{"a/b/c/d/e/f.bin", filepath.Join(tmpDir, "a/b/c/d/e/f.bin")},
	}

	for _, tc := range testCases {
		t.Run(tc.storagePath, func(t *testing.T) {
			got := s.GetFullPath(tc.storagePath)
			if got != tc.want {
				t.Errorf("GetFullPath(%q) = %q, want %q", tc.storagePath, got, tc.want)
			}
		})
	}
}

func TestLocalStorage_Save_SpecialFilenames(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewLocalStorage(tmpDir)

	testCases := []struct {
		name     string
		filename string
	}{
		{"spaces", "file with spaces.txt"},
		{"unicode", "文件名.txt"},
		{"special chars", "file-name_v2.0.1.txt"},
		{"dots", "file.name.with.many.dots.txt"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			content := "test content for " + tc.filename
			storagePath, _, _, err := s.Save(tc.filename, strings.NewReader(content))
			if err != nil {
				t.Fatalf("Save(%q) returned error: %v", tc.filename, err)
			}

			// Verify we can retrieve it
			reader, err := s.Get(storagePath)
			if err != nil {
				t.Fatalf("Get returned error: %v", err)
			}
			defer reader.Close()

			data, _ := io.ReadAll(reader)
			if string(data) != content {
				t.Errorf("content mismatch for filename %q", tc.filename)
			}
		})
	}
}

func TestLocalStorage_Save_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewLocalStorage(tmpDir)

	storagePath, hash, size, err := s.Save("empty.txt", strings.NewReader(""))
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	if size != 0 {
		t.Errorf("size = %d, want 0", size)
	}

	// Verify hash of empty content
	expectedHash := sha256.Sum256([]byte{})
	expectedHashStr := hex.EncodeToString(expectedHash[:])
	if hash != expectedHashStr {
		t.Errorf("hash = %q, want %q", hash, expectedHashStr)
	}

	// Verify file exists and is empty
	fullPath := filepath.Join(tmpDir, storagePath)
	info, err := os.Stat(fullPath)
	if err != nil {
		t.Fatalf("failed to stat empty file: %v", err)
	}
	if info.Size() != 0 {
		t.Errorf("file size = %d, want 0", info.Size())
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()

	content := "content to copy"
	srcPath := filepath.Join(tmpDir, "source.txt")
	dstPath := filepath.Join(tmpDir, "dest.txt")

	// Create source file
	if err := os.WriteFile(srcPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	// Copy
	if err := copyFile(srcPath, dstPath); err != nil {
		t.Fatalf("copyFile returned error: %v", err)
	}

	// Verify destination content
	data, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("failed to read destination: %v", err)
	}
	if string(data) != content {
		t.Errorf("copied content = %q, want %q", string(data), content)
	}
}

func TestCopyFile_NonExistentSource(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "nonexistent.txt")
	dstPath := filepath.Join(tmpDir, "dest.txt")

	err := copyFile(srcPath, dstPath)
	if err == nil {
		t.Error("copyFile should return error for non-existent source")
	}
}

// Benchmark for Save operation
func BenchmarkLocalStorage_Save(b *testing.B) {
	tmpDir := b.TempDir()
	s := NewLocalStorage(tmpDir)

	// 10KB content
	content := make([]byte, 10*1024)
	for i := range content {
		content[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Save("benchmark.bin", bytes.NewReader(content))
	}
}
