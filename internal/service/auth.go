package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/cadimartech/rankos/internal/model"
	"github.com/cadimartech/rankos/internal/repository"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	// SessionTTL is the idle lifetime of a login session. Each authenticated
	// request slides the deadline forward.
	SessionTTL = 30 * 24 * time.Hour

	minPasswordRunes = 8
	maxPasswordBytes = 72
	sessionTouchGap  = time.Minute
)

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

// Sentinel auth errors. Their text is safe to show to the client.
var (
	ErrSetupClosed        = errors.New("setup is already complete")
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUsernameTaken      = errors.New("username is already taken")
	ErrUnauthorized       = errors.New("authentication required")
	ErrForbidden          = errors.New("admin access required")
	ErrLastAdmin          = errors.New("cannot delete the last admin")
	ErrDeleteSelf         = errors.New("cannot delete your own account")
	ErrNotFound           = errors.New("user not found")
	ErrValidation         = errors.New("validation failed")
)

// ValidationError is a client-safe input error.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

func (e *ValidationError) Unwrap() error { return ErrValidation }

// RateLimitError is returned when login backoff is in effect.
type RateLimitError struct {
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string { return "too many login attempts" }

// Status is the public authentication snapshot.
type Status struct {
	Authenticated bool
	SetupRequired bool
	User          *model.User
}

// AuthService is the local account and session layer.
type AuthService struct {
	users    *repository.UserRepository
	limiter  *LoginLimiter
	now      func() time.Time
	hashCost int
}

// NewAuthService builds an auth service with production password cost and
// the default login limiter.
func NewAuthService(users *repository.UserRepository) *AuthService {
	return &AuthService{
		users:    users,
		limiter:  NewLoginLimiter(DefaultLimitConfig()),
		now:      time.Now,
		hashCost: bcrypt.DefaultCost,
	}
}

// HashCost returns the bcrypt cost used for new passwords.
func (s *AuthService) HashCost() int { return s.hashCost }

// SetHashCost overrides the bcrypt cost. Tests use this to keep hashing fast.
// Costs outside the bcrypt range are ignored.
func (s *AuthService) SetHashCost(cost int) {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		return
	}
	s.hashCost = cost
}

// ConfigureLoginLimit replaces the login limiter.
func (s *AuthService) ConfigureLoginLimit(cfg LimitConfig) {
	s.limiter = NewLoginLimiter(cfg)
}

// HashPassword returns a bcrypt hash. The cost must be a valid bcrypt cost.
func HashPassword(password string, cost int) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

// VerifyPassword reports whether password matches a bcrypt hash.
func VerifyPassword(hash, password string) bool {
	if hash == "" || password == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

var (
	dummyOnce sync.Once
	dummyHash []byte
)

func compareDummy(password string) {
	dummyOnce.Do(func() {
		hash, err := bcrypt.GenerateFromPassword([]byte("rankos-dummy-password"), bcrypt.DefaultCost)
		if err != nil {
			dummyHash = []byte("$2a$10$7EqJtq98hPqEX7fNZaFWoOHiC.a.a.a.a.a.a.a.a.a.a.a.a")
			return
		}
		dummyHash = hash
	})
	_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
}

// Setup creates the first admin and a session. It fails once any user exists.
func (s *AuthService) Setup(ctx context.Context, username, password, userAgent, ip string) (*model.User, string, error) {
	username = strings.TrimSpace(username)
	if err := validateUsername(username); err != nil {
		return nil, "", err
	}
	if err := validatePassword(password); err != nil {
		return nil, "", err
	}
	hash, err := HashPassword(password, s.hashCost)
	if err != nil {
		return nil, "", err
	}
	user := &model.User{Username: username, PasswordHash: hash, IsAdmin: true}
	if err := s.users.CreateFirstUser(ctx, user); err != nil {
		if errors.Is(err, repository.ErrAlreadySetup) {
			return nil, "", ErrSetupClosed
		}
		if errors.Is(err, repository.ErrUsernameTaken) {
			return nil, "", ErrUsernameTaken
		}
		return nil, "", err
	}
	return s.finishLogin(ctx, user, userAgent, ip)
}

// Login checks the password and opens a session.
func (s *AuthService) Login(ctx context.Context, username, password, userAgent, ip string) (*model.User, string, error) {
	username = strings.TrimSpace(username)
	key := strings.ToLower(username)
	now := s.now()
	if wait, ok := s.limiter.Allow(ip, key, now); !ok {
		return nil, "", &RateLimitError{RetryAfter: wait}
	}

	reject := func() (*model.User, string, error) {
		s.limiter.Fail(ip, key, s.now())
		compareDummy(password)
		return nil, "", ErrInvalidCredentials
	}

	if err := validateUsername(username); err != nil {
		return reject()
	}
	if err := validatePassword(password); err != nil {
		return reject()
	}
	user, err := s.users.GetByUsername(ctx, username)
	if errors.Is(err, sql.ErrNoRows) {
		return reject()
	}
	if err != nil {
		return nil, "", err
	}
	if !VerifyPassword(user.PasswordHash, password) {
		s.limiter.Fail(ip, key, s.now())
		return nil, "", ErrInvalidCredentials
	}
	s.limiter.Success(ip, key)
	return s.finishLogin(ctx, user, userAgent, ip)
}

// NoteFailedLogin counts a login attempt that never reached password checking,
// such as a body that could not be parsed. It returns Err rate limit when the
// client is already locked.
func (s *AuthService) NoteFailedLogin(ip, username string) error {
	key := strings.ToLower(strings.TrimSpace(username))
	now := s.now()
	if wait, ok := s.limiter.Allow(ip, key, now); !ok {
		return &RateLimitError{RetryAfter: wait}
	}
	s.limiter.Fail(ip, key, now)
	return nil
}

func (s *AuthService) finishLogin(ctx context.Context, user *model.User, userAgent, ip string) (*model.User, string, error) {
	now := s.now().UTC()
	if err := s.users.UpdateLastLogin(ctx, user.ID, now); err != nil {
		return nil, "", err
	}
	user.LastLoginAt = &now
	token, err := s.startSession(ctx, user.ID, userAgent, ip)
	if err != nil {
		return nil, "", err
	}
	return publicUser(user), token, nil
}

// Logout deletes the session for a raw token. A missing token is a no-op.
func (s *AuthService) Logout(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return nil
	}
	return s.users.DeleteSessionByHash(ctx, hashToken(rawToken))
}

// Status reports whether setup is still open and whether the token is valid.
// An invalid token is signed-out, not an error.
func (s *AuthService) Status(ctx context.Context, rawToken string) (*Status, error) {
	n, err := s.users.Count(ctx)
	if err != nil {
		return nil, err
	}
	st := &Status{SetupRequired: n == 0}
	if rawToken == "" {
		return st, nil
	}
	user, err := s.ValidateSession(ctx, rawToken)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			return st, nil
		}
		return nil, err
	}
	st.Authenticated = true
	st.User = user
	return st, nil
}

// ValidateSession loads the user for a raw token and slides the expiry.
func (s *AuthService) ValidateSession(ctx context.Context, rawToken string) (*model.User, error) {
	if rawToken == "" {
		return nil, ErrUnauthorized
	}
	session, user, err := s.users.GetSession(ctx, hashToken(rawToken))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUnauthorized
	}
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	if !session.ExpiresAt.After(now) {
		if delErr := s.users.DeleteSessionByHash(ctx, session.TokenHash); delErr != nil {
			slog.Error("failed to delete expired session", "error", delErr)
		}
		return nil, ErrUnauthorized
	}
	if now.Sub(session.LastSeenAt) >= sessionTouchGap || session.ExpiresAt.Before(now.Add(SessionTTL-time.Hour)) {
		if touchErr := s.users.TouchSession(ctx, session.ID, now.Add(SessionTTL), now); touchErr != nil {
			slog.Error("failed to refresh session", "error", touchErr)
		}
	}
	return publicUser(user), nil
}

// ChangePassword checks the current password, stores a new hash, and drops
// every other session for the user.
func (s *AuthService) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword, keepRawToken string) error {
	if err := validatePassword(newPassword); err != nil {
		return err
	}
	user, err := s.users.GetByID(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrUnauthorized
	}
	if err != nil {
		return err
	}
	if !VerifyPassword(user.PasswordHash, currentPassword) {
		return ErrInvalidCredentials
	}
	hash, err := HashPassword(newPassword, s.hashCost)
	if err != nil {
		return err
	}
	if err := s.users.UpdatePassword(ctx, userID, hash); err != nil {
		return err
	}
	keep := ""
	if keepRawToken != "" {
		keep = hashToken(keepRawToken)
	}
	return s.users.DeleteOtherSessions(ctx, userID, keep)
}

// ListUsers returns accounts for an admin.
func (s *AuthService) ListUsers(ctx context.Context, actorID uuid.UUID) ([]model.User, error) {
	if err := s.requireAdmin(ctx, actorID); err != nil {
		return nil, err
	}
	users, err := s.users.List(ctx)
	if err != nil {
		return nil, err
	}
	for i := range users {
		users[i].PasswordHash = ""
	}
	return users, nil
}

// CreateUser adds an account. Only an admin can call it.
func (s *AuthService) CreateUser(ctx context.Context, actorID uuid.UUID, username, password string, isAdmin bool) (*model.User, error) {
	if err := s.requireAdmin(ctx, actorID); err != nil {
		return nil, err
	}
	username = strings.TrimSpace(username)
	if err := validateUsername(username); err != nil {
		return nil, err
	}
	if err := validatePassword(password); err != nil {
		return nil, err
	}
	hash, err := HashPassword(password, s.hashCost)
	if err != nil {
		return nil, err
	}
	user := &model.User{Username: username, PasswordHash: hash, IsAdmin: isAdmin}
	if err := s.users.Create(ctx, user); err != nil {
		if errors.Is(err, repository.ErrUsernameTaken) {
			return nil, ErrUsernameTaken
		}
		return nil, err
	}
	return publicUser(user), nil
}

// DeleteUser removes an account. Admins cannot delete themselves or the last admin.
func (s *AuthService) DeleteUser(ctx context.Context, actorID, targetID uuid.UUID) error {
	if err := s.requireAdmin(ctx, actorID); err != nil {
		return err
	}
	target, err := s.users.GetByID(ctx, targetID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if target.IsAdmin {
		n, err := s.users.CountAdmins(ctx)
		if err != nil {
			return err
		}
		if n <= 1 {
			return ErrLastAdmin
		}
	}
	if target.ID == actorID {
		return ErrDeleteSelf
	}
	if err := s.users.Delete(ctx, targetID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *AuthService) requireAdmin(ctx context.Context, actorID uuid.UUID) error {
	actor, err := s.users.GetByID(ctx, actorID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrUnauthorized
	}
	if err != nil {
		return err
	}
	if !actor.IsAdmin {
		return ErrForbidden
	}
	return nil
}

func (s *AuthService) startSession(ctx context.Context, userID uuid.UUID, userAgent, ip string) (string, error) {
	raw, err := newSessionToken()
	if err != nil {
		return "", err
	}
	now := s.now().UTC()
	session := &model.Session{
		UserID:     userID,
		TokenHash:  hashToken(raw),
		ExpiresAt:  now.Add(SessionTTL),
		CreatedAt:  now,
		LastSeenAt: now,
		UserAgent:  trim(userAgent, 512),
		IP:         trim(ip, 64),
	}
	if err := s.users.CreateSession(ctx, session); err != nil {
		return "", err
	}
	return raw, nil
}

func publicUser(user *model.User) *model.User {
	if user == nil {
		return nil
	}
	copied := *user
	copied.PasswordHash = ""
	return &copied
}

func validateUsername(username string) error {
	if !usernamePattern.MatchString(username) {
		return &ValidationError{Message: "username must be 1-64 characters and use letters, numbers, dots, underscores, or hyphens"}
	}
	return nil
}

func validatePassword(password string) error {
	if utf8.RuneCountInString(password) < minPasswordRunes {
		return &ValidationError{Message: "password must be at least 8 characters"}
	}
	if len(password) > maxPasswordBytes {
		return &ValidationError{Message: "password must be at most 72 bytes"}
	}
	return nil
}

func newSessionToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func trim(v string, max int) string {
	if len(v) <= max {
		return v
	}
	return v[:max]
}
