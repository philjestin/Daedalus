package api

import (
	"bytes"
	"crypto/tls"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseProxyTrust_defaultsOverrideAndNone(t *testing.T) {
	def := parseProxyTrust("")
	trusted := []string{
		"127.0.0.1", "::1", "10.1.2.3", "172.16.0.1", "172.31.255.255",
		"192.168.1.20", "100.64.0.1", "100.127.255.255", "fd00::1", "fe80::1",
		"::ffff:172.17.0.1",
	}
	for _, raw := range trusted {
		if !def.contains(net.ParseIP(raw)) {
			t.Fatalf("%s should be trusted by default", raw)
		}
	}
	untrusted := []string{
		"203.0.113.5", "198.51.100.8", "8.8.8.8", "169.254.1.1",
		"172.15.255.255", "172.32.0.1", "100.128.0.1", "192.169.0.1", "fec0::1", "::2",
	}
	for _, raw := range untrusted {
		if def.contains(net.ParseIP(raw)) {
			t.Fatalf("%s should not be trusted by default", raw)
		}
	}

	explicit := parseProxyTrust(" 203.0.113.10, 2001:db8::/32 ")
	if !explicit.contains(net.ParseIP("203.0.113.10")) || !explicit.contains(net.ParseIP("2001:db8::5")) {
		t.Fatal("explicit entries should be trusted")
	}
	if explicit.contains(net.ParseIP("10.1.2.3")) || explicit.contains(net.ParseIP("172.17.0.1")) {
		t.Fatal("explicit TRUSTED_PROXIES replaces the private defaults")
	}
	if !explicit.contains(net.ParseIP("127.0.0.1")) || !explicit.contains(net.ParseIP("::1")) {
		t.Fatal("loopback stays trusted unless TRUSTED_PROXIES is none")
	}

	none := parseProxyTrust(" NoNe ")
	for _, raw := range []string{"127.0.0.1", "::1", "10.0.0.1", "203.0.113.10"} {
		if none.contains(net.ParseIP(raw)) {
			t.Fatalf("none trusted %s", raw)
		}
	}

	mixed := parseProxyTrust("nope, 192.0.2.10")
	if !mixed.contains(net.ParseIP("192.0.2.10")) || mixed.contains(net.ParseIP("10.0.0.1")) {
		t.Fatalf("invalid entry should be skipped, got contains 192.0.2.10=%v 10.0.0.1=%v",
			mixed.contains(net.ParseIP("192.0.2.10")), mixed.contains(net.ParseIP("10.0.0.1")))
	}
	if !mixed.loopback {
		t.Fatal("a list that is not none keeps loopback")
	}
}

func TestLoginAddresses_proxyFailureSkipsSharedLock(t *testing.T) {
	trust := parseProxyTrust("")

	proxy := httptest.NewRequest(http.MethodGet, "/health", nil)
	proxy.RemoteAddr = "172.17.0.1:9"
	proxy.Header.Set("X-Forwarded-For", "172.17.0.1")
	session, limit := loginAddresses(proxy, trust)
	if session != "172.17.0.1" || limit != "" {
		t.Fatalf("unresolved proxy session=%s limit=%q", session, limit)
	}

	proxy.Header.Del("X-Forwarded-For")
	proxy.Header.Set("X-Forwarded-Proto", "https")
	if _, limit = loginAddresses(proxy, trust); limit != "" {
		t.Fatalf("proto-only proxy should not share an IP lock, limit=%q", limit)
	}

	proxy.Header.Set("X-Forwarded-For", "203.0.113.40, 10.0.0.4")
	session, limit = loginAddresses(proxy, trust)
	if session != "203.0.113.40" || limit != "203.0.113.40" {
		t.Fatalf("resolved client session=%s limit=%s", session, limit)
	}

	direct := httptest.NewRequest(http.MethodGet, "/health", nil)
	direct.RemoteAddr = "192.168.1.20:9"
	session, limit = loginAddresses(direct, trust)
	if session != "192.168.1.20" || limit != "192.168.1.20" {
		t.Fatalf("direct lan client session=%s limit=%s", session, limit)
	}

	spoof := httptest.NewRequest(http.MethodGet, "/health", nil)
	spoof.RemoteAddr = "203.0.113.4:9"
	spoof.Header.Set("X-Forwarded-For", "198.51.100.9")
	spoof.Header.Set("X-Forwarded-Proto", "https")
	session, limit = loginAddresses(spoof, trust)
	if session != "203.0.113.4" || limit != "203.0.113.4" {
		t.Fatalf("untrusted peer session=%s limit=%s", session, limit)
	}
}

func TestClientIP_explicitProxyReplacesPrivateDefaults(t *testing.T) {
	trust := parseProxyTrust("203.0.113.10")
	private := httptest.NewRequest(http.MethodGet, "/health", nil)
	private.RemoteAddr = "10.1.2.3:9"
	private.Header.Set("X-Forwarded-For", "198.51.100.5")
	private.Header.Set("X-Real-IP", "198.51.100.6")
	if ip, distinct := clientIP(private, trust); ip != "10.1.2.3" || distinct {
		t.Fatalf("private peer after override ip=%s distinct=%v", ip, distinct)
	}

	listed := httptest.NewRequest(http.MethodGet, "/health", nil)
	listed.RemoteAddr = "203.0.113.10:9"
	listed.Header.Set("X-Forwarded-For", "198.51.100.5, 203.0.113.10")
	if ip, distinct := clientIP(listed, trust); ip != "198.51.100.5" || !distinct {
		t.Fatalf("listed proxy ip=%s distinct=%v", ip, distinct)
	}

	none := parseProxyTrust("none")
	loop := httptest.NewRequest(http.MethodGet, "/health", nil)
	loop.RemoteAddr = "127.0.0.1:9"
	loop.Header.Set("X-Forwarded-For", "203.0.113.5")
	loop.Header.Set("X-Forwarded-Proto", "https")
	if ip, distinct := clientIP(loop, none); ip != "127.0.0.1" || distinct {
		t.Fatalf("none loopback ip=%s distinct=%v", ip, distinct)
	}
	if requestIsHTTPS(loop, none) {
		t.Fatal("none must ignore X-Forwarded-Proto")
	}
	loop.Header.Del("X-Forwarded-Proto")
	loop.TLS = &tls.ConnectionState{}
	if !requestIsHTTPS(loop, none) {
		t.Fatal("real TLS stays https when proxy trust is disabled")
	}
}

func TestClientIP_warnsOnceWhenForwardedHeadersAreUntrusted(t *testing.T) {
	resetUntrustedForwardWarn()
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})))
	t.Cleanup(func() { slog.SetDefault(previous) })

	trust := parseProxyTrust("")
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.RemoteAddr = "203.0.113.10:9"
	req.Header.Set("X-Forwarded-For", "198.51.100.9")
	if ip, distinct := clientIP(req, trust); ip != "203.0.113.10" || distinct {
		t.Fatalf("ip=%s distinct=%v", ip, distinct)
	}
	req.Header.Set("X-Real-IP", "198.51.100.8")
	clientIP(req, trust)
	if strings.Count(buf.String(), "TRUSTED_PROXIES") != 1 {
		t.Fatalf("logs:\n%s", buf.String())
	}

	req.RemoteAddr = "127.0.0.1:9"
	clientIP(req, trust)
	if strings.Count(buf.String(), "TRUSTED_PROXIES") != 1 {
		t.Fatalf("trusted peer logged again:\n%s", buf.String())
	}
}
