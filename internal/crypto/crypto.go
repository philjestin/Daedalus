// Package crypto provides encryption and decryption for sensitive data.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
)

const (
	// EncryptedPrefix is prepended to encrypted values to identify them.
	EncryptedPrefix = "enc:v1:"
)

var (
	// ErrInvalidCiphertext indicates the ciphertext is malformed.
	ErrInvalidCiphertext = errors.New("invalid ciphertext")

	// ErrNoEncryptionKey indicates no encryption key is available.
	ErrNoEncryptionKey = errors.New("no encryption key configured")
)

// Encryptor handles encryption and decryption of sensitive data.
type Encryptor struct {
	key    []byte
	cipher cipher.AEAD
	mu     sync.RWMutex
}

// instance is the default encryptor instance.
var instance *Encryptor
var once sync.Once

// Default returns the default Encryptor instance, initializing it if necessary.
// The encryption key is derived from the ENCRYPTION_KEY environment variable,
// or falls back to a machine-specific key if not set.
func Default() *Encryptor {
	once.Do(func() {
		key := deriveKey()
		instance, _ = New(key)
	})
	return instance
}

// deriveKey derives an encryption key from environment or machine identity.
func deriveKey() []byte {
	// First, check for explicit encryption key
	if key := os.Getenv("ENCRYPTION_KEY"); key != "" {
		// Hash the key to ensure it's exactly 32 bytes
		hash := sha256.Sum256([]byte(key))
		return hash[:]
	}

	// Fall back to a machine-specific key based on hostname and home directory
	// This provides some protection but isn't as secure as an explicit key
	hostname, _ := os.Hostname()
	homeDir, _ := os.UserHomeDir()
	machineID := hostname + ":" + homeDir

	hash := sha256.Sum256([]byte(machineID))
	return hash[:]
}

// New creates a new Encryptor with the given key.
// Key must be exactly 32 bytes for AES-256.
func New(key []byte) (*Encryptor, error) {
	if len(key) != 32 {
		// Hash the key to get exactly 32 bytes
		hash := sha256.Sum256(key)
		key = hash[:]
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &Encryptor{
		key:    key,
		cipher: gcm,
	}, nil
}

// Encrypt encrypts plaintext and returns a base64-encoded ciphertext with prefix.
func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	if e == nil || e.cipher == nil {
		return "", ErrNoEncryptionKey
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	// Create a unique nonce for this encryption
	nonce := make([]byte, e.cipher.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Encrypt the plaintext
	ciphertext := e.cipher.Seal(nonce, nonce, []byte(plaintext), nil)

	// Encode to base64 and add prefix
	encoded := base64.StdEncoding.EncodeToString(ciphertext)
	return EncryptedPrefix + encoded, nil
}

// Decrypt decrypts a ciphertext that was encrypted with Encrypt.
// If the value doesn't have the encrypted prefix, it's returned as-is (legacy support).
func (e *Encryptor) Decrypt(ciphertext string) (string, error) {
	// Check if this is actually encrypted
	if !strings.HasPrefix(ciphertext, EncryptedPrefix) {
		// Not encrypted, return as-is (legacy unencrypted value)
		return ciphertext, nil
	}

	if e == nil || e.cipher == nil {
		return "", ErrNoEncryptionKey
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	// Remove prefix and decode base64
	encoded := strings.TrimPrefix(ciphertext, EncryptedPrefix)
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", ErrInvalidCiphertext
	}

	// Extract nonce and ciphertext
	nonceSize := e.cipher.NonceSize()
	if len(data) < nonceSize {
		return "", ErrInvalidCiphertext
	}

	nonce := data[:nonceSize]
	encrypted := data[nonceSize:]

	// Decrypt
	plaintext, err := e.cipher.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// IsEncrypted checks if a value is encrypted (has the encrypted prefix).
func IsEncrypted(value string) bool {
	return strings.HasPrefix(value, EncryptedPrefix)
}

// Convenience functions using the default encryptor

// Encrypt encrypts plaintext using the default encryptor.
func Encrypt(plaintext string) (string, error) {
	return Default().Encrypt(plaintext)
}

// Decrypt decrypts ciphertext using the default encryptor.
func Decrypt(ciphertext string) (string, error) {
	return Default().Decrypt(ciphertext)
}
