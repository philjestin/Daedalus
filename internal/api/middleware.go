package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/hyperion/printfarm/internal/model"
	"github.com/hyperion/printfarm/internal/service"
)

// Context keys for auth data
type contextKey string

const (
	ContextKeyUser   contextKey = "user"
	ContextKeyClaims contextKey = "claims"
	ContextKeyToken  contextKey = "token"
)

// AuthMiddleware provides JWT authentication middleware.
type AuthMiddleware struct {
	authService *service.AuthService
}

// NewAuthMiddleware creates a new AuthMiddleware.
func NewAuthMiddleware(authService *service.AuthService) *AuthMiddleware {
	return &AuthMiddleware{authService: authService}
}

// RequireAuth returns middleware that requires a valid JWT token.
func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			respondError(w, http.StatusUnauthorized, "authorization header required")
			return
		}

		// Parse Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			respondError(w, http.StatusUnauthorized, "invalid authorization header format")
			return
		}
		token := parts[1]

		// Validate JWT
		claims, err := m.authService.ValidateJWT(r.Context(), token)
		if err != nil {
			respondError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		// Get user from claims
		user, err := m.authService.GetCurrentUser(r.Context(), claims)
		if err != nil {
			respondError(w, http.StatusUnauthorized, "user not found or disabled")
			return
		}

		// Add user and claims to context
		ctx := context.WithValue(r.Context(), ContextKeyUser, user)
		ctx = context.WithValue(ctx, ContextKeyClaims, claims)
		ctx = context.WithValue(ctx, ContextKeyToken, token)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuth returns middleware that validates JWT if present, but doesn't require it.
func (m *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			// Parse Bearer token
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
				token := parts[1]

				// Validate JWT
				claims, err := m.authService.ValidateJWT(r.Context(), token)
				if err == nil {
					// Get user from claims
					user, err := m.authService.GetCurrentUser(r.Context(), claims)
					if err == nil {
						// Add user and claims to context
						ctx := context.WithValue(r.Context(), ContextKeyUser, user)
						ctx = context.WithValue(ctx, ContextKeyClaims, claims)
						ctx = context.WithValue(ctx, ContextKeyToken, token)
						r = r.WithContext(ctx)
					}
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

// RequireAdmin returns middleware that requires an admin user.
func (m *AuthMiddleware) RequireAdmin(next http.Handler) http.Handler {
	return m.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUserFromContext(r.Context())
		if user == nil || user.Role != model.UserRoleAdmin {
			respondError(w, http.StatusForbidden, "admin access required")
			return
		}
		next.ServeHTTP(w, r)
	}))
}

// GetUserFromContext returns the authenticated user from the context.
func GetUserFromContext(ctx context.Context) *model.User {
	user, _ := ctx.Value(ContextKeyUser).(*model.User)
	return user
}

// GetClaimsFromContext returns the JWT claims from the context.
func GetClaimsFromContext(ctx context.Context) *model.JWTClaims {
	claims, _ := ctx.Value(ContextKeyClaims).(*model.JWTClaims)
	return claims
}

// GetTokenFromContext returns the JWT token string from the context.
func GetTokenFromContext(ctx context.Context) string {
	token, _ := ctx.Value(ContextKeyToken).(string)
	return token
}
