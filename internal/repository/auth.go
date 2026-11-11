package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cadimartech/rankos/internal/model"
	"github.com/google/uuid"
)

// ErrAlreadySetup is returned when the first-user transaction finds an account.
var ErrAlreadySetup = errors.New("setup is already complete")

// ErrUsernameTaken is returned when username uniqueness fails.
var ErrUsernameTaken = errors.New("username is already taken")

// UserRepository stores local users and sessions.
type UserRepository struct {
	db *sql.DB
}

const userColumns = `id, username, password_hash, is_admin, created_at, last_login_at`

// Count returns the number of users.
func (r *UserRepository) Count(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return n, nil
}

// CountAdmins returns the number of admin users.
func (r *UserRepository) CountAdmins(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE is_admin != 0`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count admins: %w", err)
	}
	return n, nil
}

// CreateFirstUser inserts the admin account only when the table is empty.
func (r *UserRepository) CreateFirstUser(ctx context.Context, user *model.User) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin setup: %w", err)
	}
	defer tx.Rollback()

	var n int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n); err != nil {
		return fmt.Errorf("count users: %w", err)
	}
	if n > 0 {
		return ErrAlreadySetup
	}

	prepareUser(user)
	user.IsAdmin = true
	if err := insertUser(ctx, tx, user); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit setup: %w", err)
	}
	return nil
}

// Create inserts a user. The caller decides whether the user is an admin.
func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	prepareUser(user)
	if err := insertUser(ctx, r.db, user); err != nil {
		return err
	}
	return nil
}

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func prepareUser(user *model.User) {
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now().UTC()
	}
}

func insertUser(ctx context.Context, db execer, user *model.User) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO users (id, username, password_hash, is_admin, created_at, last_login_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		user.ID.String(), user.Username, user.PasswordHash, boolToInt(user.IsAdmin), user.CreatedAt.UTC(), nullTime(user.LastLoginAt),
	)
	if isUniqueViolation(err) {
		return ErrUsernameTaken
	}
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

// GetByUsername loads a user. The username match is case-insensitive.
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users WHERE username = ? COLLATE NOCASE`, username)
	return scanUser(row)
}

// GetByID loads a user by id.
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users WHERE id = ?`, id.String())
	return scanUser(row)
}

// List returns every user ordered by username.
func (r *UserRepository) List(ctx context.Context) ([]model.User, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+userColumns+` FROM users ORDER BY username COLLATE NOCASE`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	users := []model.User{}
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return users, nil
}

// UpdatePassword replaces the stored password hash.
func (r *UserRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	res, err := r.db.ExecContext(ctx, `UPDATE users SET password_hash = ? WHERE id = ?`, passwordHash, id.String())
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	return mustAffect(res)
}

// UpdateLastLogin records a successful sign-in.
func (r *UserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID, at time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET last_login_at = ? WHERE id = ?`, at.UTC(), id.String())
	if err != nil {
		return fmt.Errorf("update last login: %w", err)
	}
	return nil
}

// Delete removes a user. Sessions cascade.
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id.String())
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return mustAffect(res)
}

// CreateSession stores a hashed session token.
func (r *UserRepository) CreateSession(ctx context.Context, session *model.Session) error {
	if session.ID == uuid.Nil {
		session.ID = uuid.New()
	}
	now := time.Now().UTC()
	if session.CreatedAt.IsZero() {
		session.CreatedAt = now
	}
	if session.LastSeenAt.IsZero() {
		session.LastSeenAt = session.CreatedAt
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, token_hash, expires_at, created_at, last_seen_at, user_agent, ip)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		session.ID.String(), session.UserID.String(), session.TokenHash,
		session.ExpiresAt.UTC(), session.CreatedAt.UTC(), session.LastSeenAt.UTC(),
		nullString(session.UserAgent), nullString(session.IP),
	)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

// GetSession loads a session and its user by token hash.
func (r *UserRepository) GetSession(ctx context.Context, tokenHash string) (*model.Session, *model.User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT s.id, s.user_id, s.token_hash, s.expires_at, s.created_at, s.last_seen_at, s.user_agent, s.ip,
		        u.id, u.username, u.password_hash, u.is_admin, u.created_at, u.last_login_at
		 FROM sessions s
		 JOIN users u ON u.id = s.user_id
		 WHERE s.token_hash = ?`, tokenHash)

	var session model.Session
	var user model.User
	var sessionID, userID, joinedUserID string
	var userAgent, ip sql.NullString
	var isAdmin int
	err := scanRow(row,
		&sessionID, &userID, &session.TokenHash, &session.ExpiresAt, &session.CreatedAt, &session.LastSeenAt, &userAgent, &ip,
		&joinedUserID, &user.Username, &user.PasswordHash, &isAdmin, &user.CreatedAt, &user.LastLoginAt,
	)
	if err != nil {
		return nil, nil, err
	}
	session.ID, err = uuid.Parse(sessionID)
	if err != nil {
		return nil, nil, fmt.Errorf("parse session id: %w", err)
	}
	session.UserID, err = uuid.Parse(userID)
	if err != nil {
		return nil, nil, fmt.Errorf("parse session user id: %w", err)
	}
	user.ID, err = uuid.Parse(joinedUserID)
	if err != nil {
		return nil, nil, fmt.Errorf("parse user id: %w", err)
	}
	user.IsAdmin = isAdmin != 0
	if userAgent.Valid {
		session.UserAgent = userAgent.String
	}
	if ip.Valid {
		session.IP = ip.String
	}
	return &session, &user, nil
}

// TouchSession extends a session and records the latest use.
func (r *UserRepository) TouchSession(ctx context.Context, id uuid.UUID, expiresAt, lastSeen time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE sessions SET expires_at = ?, last_seen_at = ? WHERE id = ?`,
		expiresAt.UTC(), lastSeen.UTC(), id.String())
	if err != nil {
		return fmt.Errorf("touch session: %w", err)
	}
	return nil
}

// DeleteSessionByHash removes one session.
func (r *UserRepository) DeleteSessionByHash(ctx context.Context, tokenHash string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, tokenHash)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// DeleteOtherSessions removes every session for a user except the one to keep.
func (r *UserRepository) DeleteOtherSessions(ctx context.Context, userID uuid.UUID, keepHash string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM sessions WHERE user_id = ? AND token_hash != ?`,
		userID.String(), keepHash)
	if err != nil {
		return fmt.Errorf("delete other sessions: %w", err)
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanUser(row scanner) (*model.User, error) {
	var user model.User
	var id string
	var isAdmin int
	if err := scanRow(row, &id, &user.Username, &user.PasswordHash, &isAdmin, &user.CreatedAt, &user.LastLoginAt); err != nil {
		return nil, err
	}
	parsed, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("parse user id: %w", err)
	}
	user.ID = parsed
	user.IsAdmin = isAdmin != 0
	return &user, nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func nullTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC()
}

func nullString(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func mustAffect(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}
