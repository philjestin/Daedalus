package api

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/cadimartech/rankos/internal/model"
	"github.com/cadimartech/rankos/internal/service"
)

// SessionCookieName is the HttpOnly session cookie.
const SessionCookieName = "rankos_session"

type authContextKey struct{}

// CurrentUser returns the authenticated user set by AuthGate.
func CurrentUser(ctx context.Context) (*model.User, bool) {
	user, ok := ctx.Value(authContextKey{}).(*model.User)
	return user, ok && user != nil
}

// AuthGate requires a session for /api and /ws, with a short exemption list.
// Printer camera routes mounted under /api later are covered because the check
// is on the path prefix, before the route table is consulted.
//
// Open paths: /health, static files, /api/auth/login|logout|status|setup,
// /api/public (share-token quotes), and the Etsy webhook (signed by Etsy, not
// a browser session). The Etsy OAuth callback is a top-level browser redirect
// back to this host, so SameSite=Lax sends the cookie and the route stays
// protected.
func AuthGate(auth *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clean := cleanPath(r.URL.Path)
			if requiresSession(clean) {
				user, ok := authenticatedUser(r, auth)
				if !ok {
					respondError(w, http.StatusUnauthorized, "authentication required")
					return
				}
				r = r.WithContext(context.WithValue(r.Context(), authContextKey{}, user))
			}
			if requiresCSRF(r.Method, clean) && !originAllowed(r) {
				respondError(w, http.StatusForbidden, "cross-origin request blocked")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func authenticatedUser(r *http.Request, auth *service.AuthService) (*model.User, bool) {
	if auth == nil {
		return nil, false
	}
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil || cookie.Value == "" {
		return nil, false
	}
	user, err := auth.ValidateSession(r.Context(), cookie.Value)
	if err != nil || user == nil {
		return nil, false
	}
	return user, true
}

func requiresSession(clean string) bool {
	if clean == "/ws" {
		return true
	}
	if clean != "/api" && !strings.HasPrefix(clean, "/api/") {
		return false
	}
	return !authExempt(clean)
}

func authExempt(clean string) bool {
	switch clean {
	case "/api/auth/login", "/api/auth/logout", "/api/auth/status", "/api/auth/setup":
		return true
	case "/api/integrations/etsy/webhook":
		return true
	}
	return clean == "/api/public" || strings.HasPrefix(clean, "/api/public/")
}

func requiresCSRF(method, clean string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
	default:
		return false
	}
	if clean == "/api/integrations/etsy/webhook" {
		return false
	}
	return clean == "/api" || strings.HasPrefix(clean, "/api/")
}

func originAllowed(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	return originHostnameAllowed(r.Host, origin)
}

func originHostnameAllowed(requestHost, origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Hostname() == "" {
		return false
	}
	host := requestHost
	if name, _, err := net.SplitHostPort(host); err == nil {
		host = name
	}
	if host == "" {
		return false
	}
	return strings.EqualFold(host, parsed.Hostname())
}

func cleanPath(p string) string {
	if p == "" {
		return "/"
	}
	return path.Clean(p)
}

func trustedProxyNets() []*net.IPNet {
	var nets []*net.IPNet
	for _, part := range strings.Split(os.Getenv("TRUSTED_PROXIES"), ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		network, err := parseTrustedNet(part)
		if err != nil {
			slog.Warn("ignoring invalid TRUSTED_PROXIES entry", "value", part)
			continue
		}
		nets = append(nets, network)
	}
	return nets
}

func parseTrustedNet(part string) (*net.IPNet, error) {
	if !strings.Contains(part, "/") {
		ip := net.ParseIP(part)
		if ip == nil {
			return nil, fmt.Errorf("invalid proxy %q", part)
		}
		bits := 128
		if ip.To4() != nil {
			bits = 32
		}
		part = ip.String() + "/" + strconv.Itoa(bits)
	}
	_, network, err := net.ParseCIDR(part)
	if err != nil {
		return nil, err
	}
	return network, nil
}

func remoteIP(remoteAddr string) net.IP {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	return net.ParseIP(strings.Trim(host, "[]"))
}

func peerIsTrusted(remoteAddr string, proxies []*net.IPNet) bool {
	ip := remoteIP(remoteAddr)
	if ip == nil {
		return false
	}
	if ip.IsLoopback() {
		return true
	}
	for _, network := range proxies {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func clientIP(r *http.Request, proxies []*net.IPNet) string {
	peer := remoteIP(r.RemoteAddr)
	peerStr := r.RemoteAddr
	if peer != nil {
		peerStr = peer.String()
	}
	if !peerIsTrusted(r.RemoteAddr, proxies) {
		return peerStr
	}
	if ip := headerIP(r.Header.Get("CF-Connecting-IP")); ip != nil {
		return ip.String()
	}
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		first := strings.TrimSpace(strings.Split(forwarded, ",")[0])
		if ip := net.ParseIP(first); ip != nil {
			return ip.String()
		}
	}
	return peerStr
}

func headerIP(value string) net.IP {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return net.ParseIP(value)
}

func requestIsHTTPS(r *http.Request, proxies []*net.IPNet) bool {
	if r.TLS != nil {
		return true
	}
	if !peerIsTrusted(r.RemoteAddr, proxies) {
		return false
	}
	proto := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0])
	return strings.EqualFold(proto, "https")
}
