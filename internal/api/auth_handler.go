package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/cadimartech/rankos/internal/model"
	"github.com/cadimartech/rankos/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// AuthHandler serves local login, setup, and user management.
type AuthHandler struct {
	auth   *service.AuthService
	trust  proxyTrust
	secure cookieSecureMode
}

// NewAuthHandler builds the auth HTTP handler.
func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	trust := proxyTrustFromEnv()
	logProxyTrust(trust)
	return &AuthHandler{
		auth:   auth,
		trust:  trust,
		secure: cookieSecureFromEnv(),
	}
}

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type passwordChange struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type createUserBody struct {
	Username string `json:"username"`
	Password string `json:"password"`
	IsAdmin  bool   `json:"is_admin"`
}

// Status reports whether the first account still needs to be created and
// whether the request's session is valid.
func (h *AuthHandler) Status(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	st, err := h.auth.Status(r.Context(), rawSessionToken(r))
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeStatus(w, http.StatusOK, st)
}

// Setup creates the first admin. It is rejected after any user exists.
func (h *AuthHandler) Setup(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	var body credentials
	if err := decodeAuthJSON(w, r, &body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	sessionIP, _ := loginAddresses(r, h.trust)
	user, token, err := h.auth.Setup(r.Context(), body.Username, body.Password, r.UserAgent(), sessionIP)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	h.setSessionCookie(w, r, token)
	writeStatus(w, http.StatusCreated, &service.Status{Authenticated: true, User: user})
}

// Login opens a session when the username and password match.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	client := h.loginClient(r)
	var body credentials
	if err := decodeAuthJSON(w, r, &body); err != nil {
		limitIP := ""
		if client.LimitIP {
			limitIP = client.IP
		}
		if limErr := h.auth.NoteFailedLogin(limitIP, ""); limErr != nil {
			writeAuthError(w, limErr)
			return
		}
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	user, token, err := h.auth.LoginAs(r.Context(), body.Username, body.Password, r.UserAgent(), client)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	h.setSessionCookie(w, r, token)
	writeStatus(w, http.StatusOK, &service.Status{Authenticated: true, User: user})
}

// Logout deletes the current session and clears the cookie.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	if err := h.auth.Logout(r.Context(), rawSessionToken(r)); err != nil {
		writeAuthError(w, err)
		return
	}
	h.clearSessionCookie(w, r)
	w.WriteHeader(http.StatusNoContent)
}

// ChangePassword replaces the signed-in user's password.
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	user := h.requireUser(w, r)
	if user == nil {
		return
	}
	var body passwordChange
	if err := decodeAuthJSON(w, r, &body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.auth.ChangePassword(r.Context(), user.ID, body.CurrentPassword, body.NewPassword, rawSessionToken(r)); err != nil {
		writeAuthError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListUsers returns every account. Admin only.
func (h *AuthHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	user := h.requireUser(w, r)
	if user == nil {
		return
	}
	users, err := h.auth.ListUsers(r.Context(), user.ID)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, users)
}

// CreateUser adds an account. Admin only.
func (h *AuthHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	user := h.requireUser(w, r)
	if user == nil {
		return
	}
	var body createUserBody
	if err := decodeAuthJSON(w, r, &body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	created, err := h.auth.CreateUser(r.Context(), user.ID, body.Username, body.Password, body.IsAdmin)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, created)
}

// DeleteUser removes an account. Admin only.
func (h *AuthHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	user := h.requireUser(w, r)
	if user == nil {
		return
	}
	targetID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	if err := h.auth.DeleteUser(r.Context(), user.ID, targetID); err != nil {
		writeAuthError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) ready(w http.ResponseWriter) bool {
	if h == nil || h.auth == nil {
		respondError(w, http.StatusServiceUnavailable, "authentication is not configured")
		return false
	}
	return true
}

func (h *AuthHandler) requireUser(w http.ResponseWriter, r *http.Request) *model.User {
	user, ok := CurrentUser(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return nil
	}
	return user
}

func (h *AuthHandler) loginClient(r *http.Request) service.LoginClient {
	sessionIP, limitIP := loginAddresses(r, h.trust)
	return service.LoginClient{IP: sessionIP, LimitIP: limitIP != ""}
}

func (h *AuthHandler) setSessionCookie(w http.ResponseWriter, r *http.Request, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   sessionCookieSecure(r, h.trust, h.secure),
		MaxAge:   int(service.SessionTTL.Seconds()),
		Expires:  time.Now().Add(service.SessionTTL),
	})
}

func (h *AuthHandler) clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   sessionCookieSecure(r, h.trust, h.secure),
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func rawSessionToken(r *http.Request) string {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func decodeAuthJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16))
	return dec.Decode(dst)
}

func writeStatus(w http.ResponseWriter, code int, st *service.Status) {
	if st == nil {
		st = &service.Status{}
	}
	body := map[string]any{
		"authenticated":  st.Authenticated,
		"setup_required": st.SetupRequired,
	}
	if st.User != nil {
		body["user"] = st.User
	}
	respondJSON(w, code, body)
}

func writeAuthError(w http.ResponseWriter, err error) {
	var rate *service.RateLimitError
	var validation *service.ValidationError
	switch {
	case errors.As(err, &rate):
		secs := int(rate.RetryAfter / time.Second)
		if rate.RetryAfter > 0 && secs < 1 {
			secs = 1
		}
		if secs < 0 {
			secs = 0
		}
		w.Header().Set("Retry-After", strconv.Itoa(secs))
		respondError(w, http.StatusTooManyRequests, "too many login attempts")
	case errors.As(err, &validation):
		respondError(w, http.StatusBadRequest, validation.Message)
	case errors.Is(err, service.ErrSetupClosed):
		respondError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, service.ErrInvalidCredentials):
		respondError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, service.ErrUsernameTaken):
		respondError(w, http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrNotFound):
		respondError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrUnauthorized):
		respondError(w, http.StatusUnauthorized, "authentication required")
	case errors.Is(err, service.ErrForbidden), errors.Is(err, service.ErrLastAdmin), errors.Is(err, service.ErrDeleteSelf):
		respondError(w, http.StatusForbidden, err.Error())
	default:
		slog.Error("auth request failed", "error", err)
		respondError(w, http.StatusInternalServerError, "internal error")
	}
}
