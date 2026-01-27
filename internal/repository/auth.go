package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/hyperion/printfarm/internal/model"
)

// AuthRepository handles authentication database operations.
type AuthRepository struct {
	db *sql.DB
}

// NewAuthRepository creates a new AuthRepository.
func NewAuthRepository(db *sql.DB) *AuthRepository {
	return &AuthRepository{db: db}
}

// User operations

// CreateUser creates a new user.
func (r *AuthRepository) CreateUser(ctx context.Context, u *model.User) error {
	u.ID = uuid.New()
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()
	if u.Role == "" {
		u.Role = model.UserRoleUser
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO users (id, email, email_verified, name, role, is_active, last_login_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, u.ID, u.Email, u.EmailVerified, u.Name, u.Role, u.IsActive, u.LastLoginAt, u.CreatedAt, u.UpdatedAt)
	return err
}

// GetUserByID retrieves a user by ID.
func (r *AuthRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var u model.User
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, email, email_verified, name, role, is_active, last_login_at, created_at, updated_at
		FROM users WHERE id = ?
	`, id), &u.ID, &u.Email, &u.EmailVerified, &u.Name, &u.Role, &u.IsActive, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &u, err
}

// GetUserByEmail retrieves a user by email.
func (r *AuthRepository) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	var u model.User
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, email, email_verified, name, role, is_active, last_login_at, created_at, updated_at
		FROM users WHERE email = ?
	`, email), &u.ID, &u.Email, &u.EmailVerified, &u.Name, &u.Role, &u.IsActive, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &u, err
}

// UpdateUser updates a user.
func (r *AuthRepository) UpdateUser(ctx context.Context, u *model.User) error {
	u.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE users SET email = ?, email_verified = ?, name = ?, role = ?, is_active = ?, last_login_at = ?, updated_at = ?
		WHERE id = ?
	`, u.Email, u.EmailVerified, u.Name, u.Role, u.IsActive, u.LastLoginAt, u.UpdatedAt, u.ID)
	return err
}

// Auth token operations

// CreateAuthToken creates a new auth token.
func (r *AuthRepository) CreateAuthToken(ctx context.Context, t *model.AuthToken) error {
	t.ID = uuid.New()
	t.CreatedAt = time.Now()
	if t.TokenType == "" {
		t.TokenType = model.TokenTypeMagicLink
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO auth_tokens (id, user_id, email, token, token_type, expires_at, used_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, t.ID, t.UserID, t.Email, t.Token, t.TokenType, t.ExpiresAt, t.UsedAt, t.CreatedAt)
	return err
}

// GetAuthTokenByToken retrieves an auth token by its token string.
func (r *AuthRepository) GetAuthTokenByToken(ctx context.Context, token string) (*model.AuthToken, error) {
	var t model.AuthToken
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, user_id, email, token, token_type, expires_at, used_at, created_at
		FROM auth_tokens WHERE token = ?
	`, token), &t.ID, &t.UserID, &t.Email, &t.Token, &t.TokenType, &t.ExpiresAt, &t.UsedAt, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &t, err
}

// MarkAuthTokenUsed marks an auth token as used.
func (r *AuthRepository) MarkAuthTokenUsed(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, `UPDATE auth_tokens SET used_at = ? WHERE id = ?`, now, id)
	return err
}

// CountRecentTokensByEmail counts tokens created for an email in a time window (for rate limiting).
func (r *AuthRepository) CountRecentTokensByEmail(ctx context.Context, email string, since time.Time) (int, error) {
	var count int
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM auth_tokens WHERE email = ? AND created_at > ?
	`, email, since), &count)
	return count, err
}

// DeleteExpiredTokens removes expired auth tokens.
func (r *AuthRepository) DeleteExpiredTokens(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM auth_tokens WHERE expires_at < ?`, time.Now())
	return err
}

// Session operations

// CreateSession creates a new session.
func (r *AuthRepository) CreateSession(ctx context.Context, s *model.Session) error {
	s.ID = uuid.New()
	s.CreatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO sessions (id, user_id, token_hash, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, s.ID, s.UserID, s.TokenHash, s.ExpiresAt, s.CreatedAt)
	return err
}

// GetSessionByTokenHash retrieves a session by token hash.
func (r *AuthRepository) GetSessionByTokenHash(ctx context.Context, tokenHash string) (*model.Session, error) {
	var s model.Session
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, user_id, token_hash, expires_at, created_at
		FROM sessions WHERE token_hash = ?
	`, tokenHash), &s.ID, &s.UserID, &s.TokenHash, &s.ExpiresAt, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &s, err
}

// GetSessionByID retrieves a session by ID.
func (r *AuthRepository) GetSessionByID(ctx context.Context, id uuid.UUID) (*model.Session, error) {
	var s model.Session
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, user_id, token_hash, expires_at, created_at
		FROM sessions WHERE id = ?
	`, id), &s.ID, &s.UserID, &s.TokenHash, &s.ExpiresAt, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &s, err
}

// DeleteSession removes a session.
func (r *AuthRepository) DeleteSession(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
	return err
}

// DeleteSessionByTokenHash removes a session by token hash.
func (r *AuthRepository) DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, tokenHash)
	return err
}

// DeleteUserSessions removes all sessions for a user.
func (r *AuthRepository) DeleteUserSessions(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ?`, userID)
	return err
}

// DeleteExpiredSessions removes expired sessions.
func (r *AuthRepository) DeleteExpiredSessions(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at < ?`, time.Now())
	return err
}
