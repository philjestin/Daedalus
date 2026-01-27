package model

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// UserRole represents the role of a user.
type UserRole string

const (
	UserRoleUser  UserRole = "user"
	UserRoleAdmin UserRole = "admin"
)

// User represents a user account.
type User struct {
	ID            uuid.UUID  `json:"id"`
	Email         string     `json:"email"`
	EmailVerified bool       `json:"email_verified"`
	Name          string     `json:"name"`
	Role          UserRole   `json:"role"`
	IsActive      bool       `json:"is_active"`
	LastLoginAt   *time.Time `json:"last_login_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// TokenType represents the type of auth token.
type TokenType string

const (
	TokenTypeMagicLink TokenType = "magic_link"
)

// AuthToken represents a magic link token.
type AuthToken struct {
	ID        uuid.UUID  `json:"id"`
	UserID    *uuid.UUID `json:"user_id,omitempty"`
	Email     string     `json:"email"`
	Token     string     `json:"token"`
	TokenType TokenType  `json:"token_type"`
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// IsExpired returns true if the token has expired.
func (t *AuthToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// IsUsed returns true if the token has been used.
func (t *AuthToken) IsUsed() bool {
	return t.UsedAt != nil
}

// Session represents an active user session for JWT invalidation.
type Session struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	TokenHash string    `json:"token_hash"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// JWTClaims represents the claims in a JWT token.
type JWTClaims struct {
	jwt.RegisteredClaims
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email"`
	Role      UserRole  `json:"role"`
	SessionID uuid.UUID `json:"session_id"`
}

// AuthResponse is returned after successful authentication.
type AuthResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
	User      *User  `json:"user"`
}

// MagicLinkRequest is the request body for requesting a magic link.
type MagicLinkRequest struct {
	Email string `json:"email"`
}

// VerifyTokenRequest is the request body for verifying a magic link token.
type VerifyTokenRequest struct {
	Token string `json:"token"`
}
