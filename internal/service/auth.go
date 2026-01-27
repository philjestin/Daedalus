package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/hyperion/printfarm/internal/model"
	"github.com/hyperion/printfarm/internal/repository"
)

const (
	// MagicLinkExpiry is how long magic links are valid.
	MagicLinkExpiry = 15 * time.Minute
	// JWTExpiry is how long JWT tokens are valid.
	JWTExpiry = 7 * 24 * time.Hour
	// RateLimitWindow is the window for rate limiting magic link requests.
	RateLimitWindow = 15 * time.Minute
	// RateLimitMax is the maximum number of magic link requests per email per window.
	RateLimitMax = 5
)

// AuthConfig holds configuration for the auth service.
type AuthConfig struct {
	JWTSecret   string
	FrontendURL string
	AppName     string
}

// AuthService handles authentication business logic.
type AuthService struct {
	repo   *repository.AuthRepository
	config AuthConfig
}

// NewAuthService creates a new AuthService.
func NewAuthService(repo *repository.AuthRepository, config AuthConfig) *AuthService {
	return &AuthService{
		repo:   repo,
		config: config,
	}
}

// RequestMagicLink creates a magic link and sends it via email.
func (s *AuthService) RequestMagicLink(ctx context.Context, emailAddr string) error {
	// Normalize email
	emailAddr = strings.ToLower(strings.TrimSpace(emailAddr))
	if emailAddr == "" {
		return fmt.Errorf("email is required")
	}

	// Rate limiting: check recent token count
	since := time.Now().Add(-RateLimitWindow)
	count, err := s.repo.CountRecentTokensByEmail(ctx, emailAddr, since)
	if err != nil {
		return fmt.Errorf("failed to check rate limit: %w", err)
	}
	if count >= RateLimitMax {
		return fmt.Errorf("too many login attempts, please try again later")
	}

	// Generate secure random token (32 bytes = 64 hex chars)
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	// Check if user already exists
	user, err := s.repo.GetUserByEmail(ctx, emailAddr)
	if err != nil {
		return fmt.Errorf("failed to check user: %w", err)
	}

	// Create auth token
	authToken := &model.AuthToken{
		Email:     emailAddr,
		Token:     token,
		TokenType: model.TokenTypeMagicLink,
		ExpiresAt: time.Now().Add(MagicLinkExpiry),
	}
	if user != nil {
		authToken.UserID = &user.ID
	}

	if err := s.repo.CreateAuthToken(ctx, authToken); err != nil {
		return fmt.Errorf("failed to create auth token: %w", err)
	}

	// Build magic link URL
	magicLinkURL := fmt.Sprintf("%s/verify?token=%s", s.config.FrontendURL, token)

	// Log the magic link (desktop app - no email sending)
	slog.Info("magic link generated", "url", magicLinkURL)

	return nil
}

// VerifyMagicLink verifies a magic link token and returns a JWT.
func (s *AuthService) VerifyMagicLink(ctx context.Context, token string) (*model.AuthResponse, error) {
	// Get the auth token
	authToken, err := s.repo.GetAuthTokenByToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}
	if authToken == nil {
		return nil, fmt.Errorf("invalid or expired token")
	}

	// Check if token is expired
	if authToken.IsExpired() {
		return nil, fmt.Errorf("token has expired")
	}

	// Check if token has been used
	if authToken.IsUsed() {
		return nil, fmt.Errorf("token has already been used")
	}

	// Mark token as used
	if err := s.repo.MarkAuthTokenUsed(ctx, authToken.ID); err != nil {
		return nil, fmt.Errorf("failed to mark token as used: %w", err)
	}

	// Get or create user
	user, err := s.repo.GetUserByEmail(ctx, authToken.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		// Create new user
		user = &model.User{
			Email:         authToken.Email,
			EmailVerified: true, // Verified by clicking the magic link
			Role:          model.UserRoleUser,
			IsActive:      true,
		}
		if err := s.repo.CreateUser(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
		slog.Info("new user created", "user_id", user.ID, "email", user.Email)
	} else {
		// Update existing user
		user.EmailVerified = true
		now := time.Now()
		user.LastLoginAt = &now
		if err := s.repo.UpdateUser(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to update user: %w", err)
		}
	}

	// Check if user is active
	if !user.IsActive {
		return nil, fmt.Errorf("account is disabled")
	}

	// Generate JWT
	jwtToken, expiresAt, err := s.generateJWT(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate JWT: %w", err)
	}

	return &model.AuthResponse{
		Token:     jwtToken,
		ExpiresAt: expiresAt.Unix(),
		User:      user,
	}, nil
}

// generateJWT creates a new JWT for the user and creates a session.
func (s *AuthService) generateJWT(ctx context.Context, user *model.User) (string, time.Time, error) {
	expiresAt := time.Now().Add(JWTExpiry)
	sessionID := uuid.New()

	claims := &model.JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    s.config.AppName,
			Subject:   user.ID.String(),
		},
		UserID:    user.ID,
		Email:     user.Email,
		Role:      user.Role,
		SessionID: sessionID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign token: %w", err)
	}

	// Create session for token invalidation
	tokenHash := hashToken(tokenString)
	session := &model.Session{
		ID:        sessionID,
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	}
	if err := s.repo.CreateSession(ctx, session); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to create session: %w", err)
	}

	return tokenString, expiresAt, nil
}

// ValidateJWT validates a JWT token and returns the claims.
func (s *AuthService) ValidateJWT(ctx context.Context, tokenString string) (*model.JWTClaims, error) {
	// Parse and validate the token
	token, err := jwt.ParseWithClaims(tokenString, &model.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*model.JWTClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Check if session still exists (for logout support)
	tokenHash := hashToken(tokenString)
	session, err := s.repo.GetSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("failed to check session: %w", err)
	}
	if session == nil {
		return nil, fmt.Errorf("session has been invalidated")
	}

	return claims, nil
}

// GetCurrentUser returns the user for the given claims.
func (s *AuthService) GetCurrentUser(ctx context.Context, claims *model.JWTClaims) (*model.User, error) {
	user, err := s.repo.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}
	if !user.IsActive {
		return nil, fmt.Errorf("account is disabled")
	}
	return user, nil
}

// Logout invalidates the current session.
func (s *AuthService) Logout(ctx context.Context, tokenString string) error {
	tokenHash := hashToken(tokenString)
	return s.repo.DeleteSessionByTokenHash(ctx, tokenHash)
}

// LogoutAll invalidates all sessions for a user.
func (s *AuthService) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	return s.repo.DeleteUserSessions(ctx, userID)
}

// CleanupExpired removes expired tokens and sessions.
func (s *AuthService) CleanupExpired(ctx context.Context) error {
	if err := s.repo.DeleteExpiredTokens(ctx); err != nil {
		slog.Error("failed to delete expired tokens", "error", err)
	}
	if err := s.repo.DeleteExpiredSessions(ctx); err != nil {
		slog.Error("failed to delete expired sessions", "error", err)
	}
	return nil
}

// hashToken creates a SHA-256 hash of a token for storage.
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
