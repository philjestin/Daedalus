package model

import (
	"time"

	"github.com/google/uuid"
)

// User is a local RankOS account. PasswordHash is never serialized.
type User struct {
	ID           uuid.UUID  `json:"id"`
	Username     string     `json:"username"`
	PasswordHash string     `json:"-"`
	IsAdmin      bool       `json:"is_admin"`
	CreatedAt    time.Time  `json:"created_at"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
}

// Session is a server-side login session. TokenHash is the SHA-256 of the
// cookie value; the raw token is never stored.
type Session struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	TokenHash  string
	ExpiresAt  time.Time
	CreatedAt  time.Time
	LastSeenAt time.Time
	UserAgent  string
	IP         string
}
