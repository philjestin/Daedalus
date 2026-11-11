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
	"sync"

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

// defaultProxyCIDRs are the private, CGNAT, and link-local networks trusted
// when TRUSTED_PROXIES is unset. Loopback is matched separately.
var defaultProxyCIDRs = []string{
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"100.64.0.0/10",
	"fc00::/7",
	"fe80::/10",
}

var defaultProxyNets = parseDefaultProxyNets()

func parseDefaultProxyNets() []*net.IPNet {
	nets := make([]*net.IPNet, 0, len(defaultProxyCIDRs))
	for _, cidr := range defaultProxyCIDRs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			panic("default trusted proxy CIDR: " + err.Error())
		}
		nets = append(nets, network)
	}
	return nets
}

type trustMode int

const (
	trustDefault trustMode = iota
	trustExplicit
	trustNone
)

// proxyTrust is the set of TCP peers allowed to supply forwarded client headers.
type proxyTrust struct {
	mode     trustMode
	loopback bool
	nets     []*net.IPNet
}

type cookieSecureMode int

const (
	cookieSecureAuto cookieSecureMode = iota
	cookieSecureOn
	cookieSecureOff
)

func proxyTrustFromEnv() proxyTrust {
	return parseProxyTrust(os.Getenv("TRUSTED_PROXIES"))
}

// parseProxyTrust interprets TRUSTED_PROXIES.
// An empty value trusts loopback plus the default private ranges.
// "none" trusts nobody. Any other value replaces the private defaults;
// loopback stays trusted.
func parseProxyTrust(raw string) proxyTrust {
	trimmed := strings.TrimSpace(raw)
	if strings.EqualFold(trimmed, "none") {
		return proxyTrust{mode: trustNone}
	}
	if trimmed == "" {
		return proxyTrust{mode: trustDefault, loopback: true, nets: defaultProxyNets}
	}
	var nets []*net.IPNet
	for _, part := range strings.Split(trimmed, ",") {
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
	return proxyTrust{mode: trustExplicit, loopback: true, nets: nets}
}

func logProxyTrust(trust proxyTrust) {
	switch trust.mode {
	case trustNone:
		slog.Warn("TRUSTED_PROXIES=none; forwarded client headers are ignored from every peer, including loopback")
	case trustExplicit:
		slog.Info("trusted proxies loaded from TRUSTED_PROXIES", "entries", len(trust.nets), "loopback", trust.loopback)
	default:
		slog.Info("trusted proxies default to loopback and private, CGNAT, and link-local ranges; set TRUSTED_PROXIES to replace them or to none to disable", "ranges", defaultProxyCIDRs)
	}
}

func (t proxyTrust) contains(ip net.IP) bool {
	ip = canonicalIP(ip)
	if ip == nil {
		return false
	}
	if t.loopback && ip.IsLoopback() {
		return true
	}
	for _, network := range t.nets {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func cookieSecureFromEnv() cookieSecureMode {
	return parseCookieSecure(os.Getenv("COOKIE_SECURE"))
}

func parseCookieSecure(value string) cookieSecureMode {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "auto":
		return cookieSecureAuto
	case "true":
		return cookieSecureOn
	case "false":
		return cookieSecureOff
	default:
		slog.Warn("invalid COOKIE_SECURE value, using auto", "value", value)
		return cookieSecureAuto
	}
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

func canonicalIP(ip net.IP) net.IP {
	if ip == nil {
		return nil
	}
	if v4 := ip.To4(); v4 != nil {
		return v4
	}
	return ip
}

var untrustedForwardWarn sync.Once

func warnUntrustedForwarded(peer string) {
	untrustedForwardWarn.Do(func() {
		slog.Warn("ignoring forwarded headers from an untrusted peer; set TRUSTED_PROXIES to the reverse proxy address if it is not on loopback or a private network", "peer", peer)
	})
}

func resetUntrustedForwardWarn() {
	untrustedForwardWarn = sync.Once{}
}

// clientIP returns the address to attribute the request to.
// distinct is true when a forwarded header supplied an address other than the TCP peer.
func clientIP(r *http.Request, trust proxyTrust) (ip string, distinct bool) {
	peer := remoteIP(r.RemoteAddr)
	peerStr := r.RemoteAddr
	if peer != nil {
		peerStr = canonicalIP(peer).String()
	}
	if peer == nil || !trust.contains(peer) {
		if hasForwardedHeaders(r) {
			warnUntrustedForwarded(peerStr)
		}
		return peerStr, false
	}
	if parsed, ok := forwardedClient(r, trust); ok && !parsed.Equal(canonicalIP(peer)) {
		return parsed.String(), true
	}
	return peerStr, false
}

// loginAddresses splits the address stored on a session from the per-IP
// backoff key. The key is empty when the peer is a trusted proxy and no
// distinct client could be resolved, so one proxy address cannot lock every user.
func loginAddresses(r *http.Request, trust proxyTrust) (sessionIP, limitIP string) {
	ip, distinct := clientIP(r, trust)
	if !distinct && trust.contains(remoteIP(r.RemoteAddr)) && hasForwardedHeaders(r) {
		return ip, ""
	}
	return ip, ip
}

func hasForwardedHeaders(r *http.Request) bool {
	return strings.TrimSpace(r.Header.Get("CF-Connecting-IP")) != "" ||
		strings.TrimSpace(r.Header.Get("X-Forwarded-For")) != "" ||
		strings.TrimSpace(r.Header.Get("X-Real-IP")) != "" ||
		strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")) != ""
}

// forwardedClient reads the client address from a trusted peer.
// CF-Connecting-IP wins, then X-Forwarded-For walked right to left, then X-Real-IP.
func forwardedClient(r *http.Request, trust proxyTrust) (net.IP, bool) {
	if ip := parseHeaderIP(r.Header.Get("CF-Connecting-IP")); ip != nil {
		return ip, true
	}
	if ip, ok := xForwardedForClient(r.Header.Get("X-Forwarded-For"), trust); ok {
		return ip, true
	}
	if ip := parseHeaderIP(r.Header.Get("X-Real-IP")); ip != nil {
		return ip, true
	}
	return nil, false
}

// xForwardedForClient walks X-Forwarded-For from the right, skipping trusted
// proxy hops. When every hop is trusted, the leftmost address is the client.
func xForwardedForClient(header string, trust proxyTrust) (net.IP, bool) {
	if strings.TrimSpace(header) == "" {
		return nil, false
	}
	var ips []net.IP
	for _, part := range strings.Split(header, ",") {
		if ip := parseHeaderIP(part); ip != nil {
			ips = append(ips, ip)
		}
	}
	if len(ips) == 0 {
		return nil, false
	}
	for i := len(ips) - 1; i >= 0; i-- {
		if !trust.contains(ips[i]) {
			return ips[i], true
		}
	}
	return ips[0], true
}

func parseHeaderIP(value string) net.IP {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"`)
	if value == "" || strings.EqualFold(value, "unknown") {
		return nil
	}
	if strings.HasPrefix(value, "[") {
		if end := strings.IndexByte(value, ']'); end > 1 {
			value = value[1:end]
		}
	}
	ip := net.ParseIP(value)
	if ip == nil {
		if host, _, err := net.SplitHostPort(value); err == nil {
			ip = net.ParseIP(host)
		}
	}
	return canonicalIP(ip)
}

func requestIsHTTPS(r *http.Request, trust proxyTrust) bool {
	if r.TLS != nil {
		return true
	}
	if !trust.contains(remoteIP(r.RemoteAddr)) {
		return false
	}
	proto := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0])
	return strings.EqualFold(proto, "https")
}

func sessionCookieSecure(r *http.Request, trust proxyTrust, mode cookieSecureMode) bool {
	switch mode {
	case cookieSecureOn:
		return true
	case cookieSecureOff:
		return false
	default:
		return requestIsHTTPS(r, trust)
	}
}
