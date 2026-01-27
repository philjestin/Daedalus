package api

import (
	"encoding/json"
	"net/http"

	"github.com/hyperion/printfarm/internal/model"
	"github.com/hyperion/printfarm/internal/service"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	service *service.AuthService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(service *service.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

// RequestMagicLink handles POST /api/auth/request-link
func (h *AuthHandler) RequestMagicLink(w http.ResponseWriter, r *http.Request) {
	var req model.MagicLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" {
		respondError(w, http.StatusBadRequest, "email is required")
		return
	}

	if err := h.service.RequestMagicLink(r.Context(), req.Email); err != nil {
		// Don't reveal whether the email exists or not for security
		// Always return success to prevent email enumeration
		if err.Error() == "too many login attempts, please try again later" {
			respondError(w, http.StatusTooManyRequests, err.Error())
			return
		}
		// For other errors, still return success to prevent email enumeration
		// but log the error
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "If your email is valid, you will receive a magic link shortly.",
	})
}

// VerifyMagicLink handles GET /api/auth/verify?token=xxx
func (h *AuthHandler) VerifyMagicLink(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		respondError(w, http.StatusBadRequest, "token is required")
		return
	}

	authResponse, err := h.service.VerifyMagicLink(r.Context(), token)
	if err != nil {
		respondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, authResponse)
}

// GetCurrentUser handles GET /api/auth/me
func (h *AuthHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	respondJSON(w, http.StatusOK, user)
}

// Logout handles POST /api/auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	token := GetTokenFromContext(r.Context())
	if token == "" {
		respondError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	if err := h.service.Logout(r.Context(), token); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to logout")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "logged out successfully",
	})
}
