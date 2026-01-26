package etsy

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

// GenerateCodeVerifier creates a random code verifier for PKCE.
// Returns a 43-128 character URL-safe string.
func GenerateCodeVerifier() (string, error) {
	// Generate 32 random bytes (will be 43 characters when base64url encoded)
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64URLEncode(b), nil
}

// GenerateCodeChallenge creates a code challenge from the verifier using S256 method.
func GenerateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64URLEncode(hash[:])
}

// GenerateState creates a random state parameter for OAuth.
func GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64URLEncode(b), nil
}

// base64URLEncode encodes data to base64url without padding.
func base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}
