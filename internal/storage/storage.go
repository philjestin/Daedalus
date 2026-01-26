package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Storage defines the interface for file storage operations.
type Storage interface {
	// Save stores file data and returns the storage path and hash.
	Save(filename string, reader io.Reader) (storagePath string, hash string, size int64, err error)
	// Get retrieves a file by its storage path.
	Get(storagePath string) (io.ReadCloser, error)
	// Delete removes a file by its storage path.
	Delete(storagePath string) error
	// GetFullPath returns the full filesystem path for a storage path.
	GetFullPath(storagePath string) string
}

// LocalStorage implements Storage using the local filesystem.
type LocalStorage struct {
	basePath string
}

// NewLocalStorage creates a new local storage instance.
func NewLocalStorage(basePath string) *LocalStorage {
	// Ensure base directory exists
	os.MkdirAll(basePath, 0755)
	return &LocalStorage{basePath: basePath}
}

// Save stores a file and returns its storage path and SHA-256 hash.
func (s *LocalStorage) Save(filename string, reader io.Reader) (string, string, int64, error) {
	// Create a temporary file to calculate hash while writing
	tempFile, err := os.CreateTemp(s.basePath, "upload-*")
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath) // Clean up temp file

	// Create hash writer
	hasher := sha256.New()
	multiWriter := io.MultiWriter(tempFile, hasher)

	// Copy data to temp file while calculating hash
	size, err := io.Copy(multiWriter, reader)
	if err != nil {
		tempFile.Close()
		return "", "", 0, fmt.Errorf("failed to write file: %w", err)
	}
	tempFile.Close()

	// Calculate final hash
	hash := hex.EncodeToString(hasher.Sum(nil))

	// Create storage path using hash for content-addressed storage
	// Format: {hash[0:2]}/{hash[2:4]}/{hash}/{filename}
	storagePath := filepath.Join(hash[0:2], hash[2:4], hash, filename)
	fullPath := filepath.Join(s.basePath, storagePath)

	// Create directory structure
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", "", 0, fmt.Errorf("failed to create directory: %w", err)
	}

	// Move temp file to final location
	if err := os.Rename(tempPath, fullPath); err != nil {
		// If rename fails (cross-device), copy instead
		if err := copyFile(tempPath, fullPath); err != nil {
			return "", "", 0, fmt.Errorf("failed to move file: %w", err)
		}
	}

	return storagePath, hash, size, nil
}

// Get retrieves a file by storage path.
func (s *LocalStorage) Get(storagePath string) (io.ReadCloser, error) {
	fullPath := filepath.Join(s.basePath, storagePath)
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	return file, nil
}

// Delete removes a file by storage path.
func (s *LocalStorage) Delete(storagePath string) error {
	fullPath := filepath.Join(s.basePath, storagePath)
	return os.Remove(fullPath)
}

// GetFullPath returns the full filesystem path.
func (s *LocalStorage) GetFullPath(storagePath string) string {
	return filepath.Join(s.basePath, storagePath)
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
	return err
}

