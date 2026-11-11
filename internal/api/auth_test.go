package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cadimartech/rankos/internal/database"
	"github.com/cadimartech/rankos/internal/printer"
	"github.com/cadimartech/rankos/internal/realtime"
	"github.com/cadimartech/rankos/internal/repository"
	"github.com/cadimartech/rankos/internal/service"
	"github.com/cadimartech/rankos/internal/storage"
	"golang.org/x/crypto/bcrypt"
	"nhooyr.io/websocket"
)

func TestAuthMiddleware_allowAndDeny(t *testing.T) {
	auth, handler, _ := newAuthAPI(t)
	ctx := context.Background()
	_, token, err := auth.Setup(ctx, "owner", "password12", "", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}

	if code := do(t, handler, http.MethodGet, "/health", "", nil).Code; code != http.StatusOK {
		t.Fatalf("health = %d", code)
	}
	if code := do(t, handler, http.MethodGet, "/api/auth/status", "", nil).Code; code != http.StatusOK {
		t.Fatalf("status = %d", code)
	}
	if code := do(t, handler, http.MethodGet, "/api/printers", "", nil).Code; code != http.StatusUnauthorized {
		t.Fatalf("printers without session = %d", code)
	}
	if code := do(t, handler, http.MethodGet, "/api/printers/00000000-0000-0000-0000-000000000001/camera", "", nil).Code; code != http.StatusUnauthorized {
		t.Fatalf("future camera route = %d", code)
	}
	if code := do(t, handler, http.MethodGet, "/ws", "", nil).Code; code != http.StatusUnauthorized {
		t.Fatalf("websocket = %d", code)
	}
	if code := do(t, handler, http.MethodGet, "/api/integrations/etsy/callback", "", nil).Code; code != http.StatusUnauthorized {
		t.Fatalf("etsy callback = %d", code)
	}
	if code := do(t, handler, http.MethodGet, "/api/public/quotes/not-a-token", "", nil).Code; code == http.StatusUnauthorized {
		t.Fatal("public quote should not require a session")
	}
	if code := do(t, handler, http.MethodPost, "/api/integrations/etsy/webhook", "", []byte(`{}`)).Code; code == http.StatusUnauthorized {
		t.Fatal("etsy webhook should not require a session")
	}
	if code := do(t, handler, http.MethodGet, "/index.html", "", nil).Code; code == http.StatusUnauthorized {
		t.Fatal("static path should not require a session")
	}

	printers := do(t, handler, http.MethodGet, "/api/printers", token, nil)
	if printers.Code == http.StatusUnauthorized {
		t.Fatalf("printers with session = %d", printers.Code)
	}
	camera := do(t, handler, http.MethodGet, "/api/printers/00000000-0000-0000-0000-000000000001/camera", token, nil)
	if camera.Code != http.StatusNotFound {
		t.Fatalf("authenticated unknown camera route = %d, want 404", camera.Code)
	}
	callback := do(t, handler, http.MethodGet, "/api/integrations/etsy/callback", token, nil)
	if callback.Code == http.StatusUnauthorized {
		t.Fatal("etsy callback should accept the session cookie")
	}
}

func TestPrinterCommands_requireSessionAndSameOrigin(t *testing.T) {
	auth, handler, _ := newAuthAPI(t)
	_, token, err := auth.Setup(context.Background(), "owner", "password12", "", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	path := "/api/printers/00000000-0000-0000-0000-000000000001/commands"
	body := []byte(`{"kind":"pause"}`)

	if code := do(t, handler, http.MethodPost, path, "", body).Code; code != http.StatusUnauthorized {
		t.Fatalf("commands without session = %d", code)
	}
	if code := do(t, handler, http.MethodGet, "/api/printers/states", "", nil).Code; code != http.StatusUnauthorized {
		t.Fatalf("states without session = %d", code)
	}
	evil := doFrom(t, handler, http.MethodPost, path, token, body, "198.51.100.9:9", map[string]string{
		"Origin": "https://evil.example",
	})
	if evil.Code != http.StatusForbidden {
		t.Fatalf("commands foreign origin = %d", evil.Code)
	}

	same := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	same.Header.Set("Content-Type", "application/json")
	same.Header.Set("X-Requested-With", "XMLHttpRequest")
	same.Header.Set("Origin", "http://"+same.Host)
	same.AddCookie(&http.Cookie{Name: SessionCookieName, Value: token})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, same)
	if rr.Code == http.StatusUnauthorized || rr.Code == http.StatusForbidden {
		t.Fatalf("commands same-origin = %d body %s", rr.Code, rr.Body.String())
	}
}

func TestAuthWebSocket_requiresSession(t *testing.T) {
	auth, handler, _ := newAuthAPI(t)
	_, token, err := auth.Setup(context.Background(), "owner", "password12", "", "")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, denied, err := websocket.Dial(ctx, wsURL, nil)
	closeBody(denied)
	if err == nil {
		t.Fatal("unauthenticated websocket dial should fail")
	}

	conn, accepted, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPHeader: http.Header{"Cookie": []string{SessionCookieName + "=" + token}},
	})
	closeBody(accepted)
	if err != nil {
		t.Fatalf("authenticated websocket dial: %v", err)
	}
	conn.Close(websocket.StatusNormalClosure, "")
}

func TestAuthSetup_firstRunThenLocked(t *testing.T) {
	_, handler, _ := newAuthAPI(t)

	open := decodeStatus(t, do(t, handler, http.MethodGet, "/api/auth/status", "", nil))
	if !open.SetupRequired || open.Authenticated {
		t.Fatalf("status = %+v", open)
	}

	short := do(t, handler, http.MethodPost, "/api/auth/setup", "", map[string]string{
		"username": "owner",
		"password": "short",
	})
	if short.Code != http.StatusBadRequest {
		t.Fatalf("short password = %d body %s", short.Code, short.Body.String())
	}

	created := do(t, handler, http.MethodPost, "/api/auth/setup", "", map[string]string{
		"username": "Owner",
		"password": "password12",
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("setup = %d body %s", created.Code, created.Body.String())
	}
	cookie := sessionCookie(t, created)
	if !cookie.HttpOnly || cookie.Path != "/" || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("cookie = %#v", cookie)
	}
	raw := created.Header().Get("Set-Cookie")
	if !strings.Contains(raw, "SameSite=Lax") || strings.Contains(raw, "Secure") {
		t.Fatalf("set-cookie = %s", raw)
	}
	if cookie.MaxAge < 29*24*60*60 {
		t.Fatalf("max-age = %d", cookie.MaxAge)
	}
	body := created.Body.String()
	if strings.Contains(body, "password12") || strings.Contains(body, "$2") {
		t.Fatalf("setup response leaked a secret: %s", body)
	}

	again := do(t, handler, http.MethodPost, "/api/auth/setup", "", map[string]string{
		"username": "other",
		"password": "password12",
	})
	if again.Code != http.StatusForbidden {
		t.Fatalf("second setup = %d", again.Code)
	}

	signedIn := decodeStatus(t, do(t, handler, http.MethodGet, "/api/auth/status", cookie.Value, nil))
	if !signedIn.Authenticated || signedIn.SetupRequired || signedIn.User == nil || !signedIn.User.IsAdmin {
		t.Fatalf("status after setup = %+v", signedIn)
	}

	loggedOut := do(t, handler, http.MethodPost, "/api/auth/logout", cookie.Value, nil)
	if loggedOut.Code != http.StatusNoContent {
		t.Fatalf("logout = %d", loggedOut.Code)
	}
	after := decodeStatus(t, do(t, handler, http.MethodGet, "/api/auth/status", cookie.Value, nil))
	if after.Authenticated || after.SetupRequired {
		t.Fatalf("status after logout = %+v", after)
	}
}

func TestAuthLogin_rateLimitAndCookieSecure(t *testing.T) {
	auth, handler, _ := newAuthAPI(t)
	auth.ConfigureLoginLimit(service.LimitConfig{
		MaxAttempts:   2,
		IPMaxAttempts: 2,
		Window:        time.Minute,
		BaseBackoff:   time.Hour,
		MaxBackoff:    time.Hour,
	})
	if _, _, err := auth.Setup(context.Background(), "owner", "password12", "", ""); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 2; i++ {
		rr := doFrom(t, handler, http.MethodPost, "/api/auth/login", "", map[string]string{
			"username": "nobody",
			"password": "wrong-password",
		}, "198.51.100.20:4000", nil)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d = %d", i+1, rr.Code)
		}
	}
	blocked := doFrom(t, handler, http.MethodPost, "/api/auth/login", "", map[string]string{
		"username": "nobody",
		"password": "wrong-password",
	}, "198.51.100.20:4000", nil)
	if blocked.Code != http.StatusTooManyRequests {
		t.Fatalf("rate limit = %d body %s", blocked.Code, blocked.Body.String())
	}
	if blocked.Header().Get("Retry-After") == "" {
		t.Fatal("missing Retry-After")
	}
	// Forwarded headers from an untrusted peer must not pick a different bucket.
	spoofed := doFrom(t, handler, http.MethodPost, "/api/auth/login", "", map[string]string{
		"username": "someone",
		"password": "wrong-password",
	}, "198.51.100.20:4000", map[string]string{
		"CF-Connecting-IP": "203.0.113.50",
		"X-Forwarded-For":  "203.0.113.51",
	})
	if spoofed.Code != http.StatusTooManyRequests {
		t.Fatalf("spoofed ip = %d, forwarded headers should be ignored", spoofed.Code)
	}

	// A trusted proxy uses CF-Connecting-IP, so a different client is not blocked.
	other := doFrom(t, handler, http.MethodPost, "/api/auth/login", "", map[string]string{
		"username": "owner",
		"password": "password12",
	}, "127.0.0.1:9", map[string]string{
		"CF-Connecting-IP":  "203.0.113.60",
		"X-Forwarded-Proto": "https",
	})
	if other.Code != http.StatusOK {
		t.Fatalf("trusted client login = %d body %s", other.Code, other.Body.String())
	}
	secure := sessionCookie(t, other)
	if !secure.Secure || !strings.Contains(other.Header().Get("Set-Cookie"), "Secure") {
		t.Fatalf("expected Secure cookie behind trusted https proxy, header %s", other.Header().Get("Set-Cookie"))
	}

	plain := doFrom(t, handler, http.MethodPost, "/api/auth/logout", secure.Value, nil, "198.51.100.20:9", map[string]string{
		"X-Forwarded-Proto": "https",
	})
	if plain.Code != http.StatusNoContent {
		t.Fatalf("logout = %d", plain.Code)
	}
	cleared := plain.Header().Get("Set-Cookie")
	if strings.Contains(cleared, "Secure") {
		t.Fatalf("untrusted peer must not honor X-Forwarded-Proto, header %s", cleared)
	}
}

func TestAuthLogin_privateProxyCookieAndClients(t *testing.T) {
	auth, handler, _ := newAuthAPI(t)
	auth.ConfigureLoginLimit(service.LimitConfig{
		MaxAttempts: 3,
		Window:      time.Minute,
		BaseBackoff: time.Hour,
		MaxBackoff:  time.Hour,
	})
	if _, _, err := auth.Setup(context.Background(), "owner", "password12", "", ""); err != nil {
		t.Fatal(err)
	}

	// Three different usernames from one real client, seen through a Docker-bridge proxy.
	// The shared proxy address must not be the bucket that locks the next client.
	for _, user := range []string{"a", "b", "c"} {
		rr := doFrom(t, handler, http.MethodPost, "/api/auth/login", "", map[string]string{
			"username": user,
			"password": "wrong-password",
		}, "172.17.0.1:9", map[string]string{
			"X-Forwarded-For": "198.51.100.1, 203.0.113.10",
		})
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("%s = %d body %s", user, rr.Code, rr.Body.String())
		}
	}
	other := doFrom(t, handler, http.MethodPost, "/api/auth/login", "", map[string]string{
		"username": "d",
		"password": "wrong-password",
	}, "172.17.0.1:9", map[string]string{
		"X-Forwarded-For": "203.0.113.11",
	})
	if other.Code != http.StatusUnauthorized {
		t.Fatalf("other client = %d body %s", other.Code, other.Body.String())
	}

	secure := doFrom(t, handler, http.MethodPost, "/api/auth/login", "", map[string]string{
		"username": "owner",
		"password": "password12",
	}, "172.17.0.1:9", map[string]string{
		"X-Forwarded-For":   "203.0.113.11",
		"X-Forwarded-Proto": "https",
	})
	if secure.Code != http.StatusOK {
		t.Fatalf("login = %d body %s", secure.Code, secure.Body.String())
	}
	if cookie := sessionCookie(t, secure); !cookie.Secure {
		t.Fatalf("expected Secure cookie behind a private HTTPS proxy, header %s", secure.Header().Get("Set-Cookie"))
	}
}

func TestAuthLogin_unresolvedProxyDoesNotLockEveryone(t *testing.T) {
	auth, handler, _ := newAuthAPI(t)
	auth.ConfigureLoginLimit(service.LimitConfig{
		MaxAttempts:   2,
		IPMaxAttempts: 2,
		Window:        time.Minute,
		BaseBackoff:   time.Hour,
		MaxBackoff:    time.Hour,
	})
	if _, _, err := auth.Setup(context.Background(), "owner", "password12", "", ""); err != nil {
		t.Fatal(err)
	}

	// Headers are present but name only the proxy, so the client IP never resolves.
	proxyHeaders := map[string]string{"X-Forwarded-For": "172.17.0.1", "X-Forwarded-Proto": "http"}
	for i := 0; i < 2; i++ {
		rr := doFrom(t, handler, http.MethodPost, "/api/auth/login", "", map[string]string{
			"username": "alice",
			"password": "wrong-password",
		}, "172.17.0.1:9", proxyHeaders)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("alice %d = %d body %s", i+1, rr.Code, rr.Body.String())
		}
	}
	locked := doFrom(t, handler, http.MethodPost, "/api/auth/login", "", map[string]string{
		"username": "alice",
		"password": "wrong-password",
	}, "172.17.0.1:9", proxyHeaders)
	if locked.Code != http.StatusTooManyRequests {
		t.Fatalf("alice lock = %d", locked.Code)
	}
	bob := doFrom(t, handler, http.MethodPost, "/api/auth/login", "", map[string]string{
		"username": "bob",
		"password": "wrong-password",
	}, "172.17.0.1:9", proxyHeaders)
	if bob.Code != http.StatusUnauthorized {
		t.Fatalf("bob should not inherit the proxy lockout, got %d body %s", bob.Code, bob.Body.String())
	}
}

func TestAuthLogin_cookieSecureEnv(t *testing.T) {
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("TRUSTED_PROXIES", "")
	auth, handler, _ := newAuthAPI(t)
	if _, _, err := auth.Setup(context.Background(), "owner", "password12", "", ""); err != nil {
		t.Fatal(err)
	}
	off := doFrom(t, handler, http.MethodPost, "/api/auth/login", "", map[string]string{
		"username": "owner",
		"password": "password12",
	}, "127.0.0.1:9", map[string]string{"X-Forwarded-Proto": "https"})
	if off.Code != http.StatusOK {
		t.Fatalf("login = %d %s", off.Code, off.Body.String())
	}
	if strings.Contains(off.Header().Get("Set-Cookie"), "Secure") {
		t.Fatalf("COOKIE_SECURE=false should clear Secure, header %s", off.Header().Get("Set-Cookie"))
	}

	t.Setenv("COOKIE_SECURE", "true")
	_, forced, _ := newAuthAPI(t)
	created := doFrom(t, forced, http.MethodPost, "/api/auth/setup", "", map[string]string{
		"username": "owner",
		"password": "password12",
	}, "198.51.100.20:9", nil)
	if created.Code != http.StatusCreated {
		t.Fatalf("setup = %d %s", created.Code, created.Body.String())
	}
	if !strings.Contains(created.Header().Get("Set-Cookie"), "Secure") {
		t.Fatalf("COOKIE_SECURE=true should set Secure on plain HTTP, header %s", created.Header().Get("Set-Cookie"))
	}
}

func TestAuthLogin_trustedProxiesNoneDisablesForwardedHeaders(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "none")
	auth, handler, _ := newAuthAPI(t)
	auth.ConfigureLoginLimit(service.LimitConfig{
		MaxAttempts:   5,
		IPMaxAttempts: 2,
		Window:        time.Minute,
		BaseBackoff:   time.Hour,
		MaxBackoff:    time.Hour,
	})
	if _, _, err := auth.Setup(context.Background(), "owner", "password12", "", ""); err != nil {
		t.Fatal(err)
	}
	plain := doFrom(t, handler, http.MethodPost, "/api/auth/login", "", map[string]string{
		"username": "owner",
		"password": "password12",
	}, "127.0.0.1:9", map[string]string{"X-Forwarded-Proto": "https"})
	if plain.Code != http.StatusOK {
		t.Fatalf("owner login = %d %s", plain.Code, plain.Body.String())
	}
	if strings.Contains(plain.Header().Get("Set-Cookie"), "Secure") {
		t.Fatalf("none must not mark the cookie Secure from X-Forwarded-Proto, header %s", plain.Header().Get("Set-Cookie"))
	}

	for _, tc := range []struct {
		user string
		xff  string
	}{
		{"ann", "203.0.113.21"},
		{"ben", "203.0.113.22"},
	} {
		rr := doFrom(t, handler, http.MethodPost, "/api/auth/login", "", map[string]string{
			"username": tc.user,
			"password": "wrong-password",
		}, "127.0.0.1:9", map[string]string{
			"X-Forwarded-For":   tc.xff,
			"X-Forwarded-Proto": "https",
		})
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("%s = %d body %s", tc.user, rr.Code, rr.Body.String())
		}
	}
	// Both attempts are the loopback peer. Forwarded headers must not split the IP bucket.
	blocked := doFrom(t, handler, http.MethodPost, "/api/auth/login", "", map[string]string{
		"username": "cara",
		"password": "wrong-password",
	}, "127.0.0.1:9", map[string]string{
		"X-Forwarded-For":   "203.0.113.23",
		"X-Forwarded-Proto": "https",
	})
	if blocked.Code != http.StatusTooManyRequests {
		t.Fatalf("none should keep the peer IP, got %d body %s", blocked.Code, blocked.Body.String())
	}
}

func TestAuthCSRF_rejectsForeignOrigin(t *testing.T) {
	auth, handler, _ := newAuthAPI(t)
	_, token, err := auth.Setup(context.Background(), "owner", "password12", "", "")
	if err != nil {
		t.Fatal(err)
	}

	evil := doFrom(t, handler, http.MethodPost, "/api/printers", token, map[string]string{"name": "x"}, "198.51.100.9:9", map[string]string{
		"Origin": "https://evil.example",
	})
	if evil.Code != http.StatusForbidden {
		t.Fatalf("foreign origin = %d", evil.Code)
	}

	login := doFrom(t, handler, http.MethodPost, "/api/auth/login", "", map[string]string{
		"username": "owner",
		"password": "password12",
	}, "198.51.100.9:9", map[string]string{"Origin": "https://evil.example"})
	if login.Code != http.StatusForbidden {
		t.Fatalf("login foreign origin = %d", login.Code)
	}

	same := httptest.NewRequest(http.MethodPost, "/api/printers", strings.NewReader(`{}`))
	same.Header.Set("Content-Type", "application/json")
	same.Header.Set("Origin", "http://"+same.Host)
	same.AddCookie(&http.Cookie{Name: SessionCookieName, Value: token})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, same)
	if rr.Code == http.StatusForbidden || rr.Code == http.StatusUnauthorized {
		t.Fatalf("same-host origin = %d body %s", rr.Code, rr.Body.String())
	}
}

func TestAuthUsers_adminOnly(t *testing.T) {
	_, handler, _ := newAuthAPI(t)
	created := do(t, handler, http.MethodPost, "/api/auth/setup", "", map[string]string{
		"username": "owner",
		"password": "password12",
	})
	if created.Code != http.StatusCreated {
		t.Fatal(created.Body.String())
	}
	adminCookie := sessionCookie(t, created).Value
	admin := decodeStatus(t, created)

	helperRR := do(t, handler, http.MethodPost, "/api/auth/users", adminCookie, map[string]any{
		"username": "helper",
		"password": "password12",
	})
	if helperRR.Code != http.StatusCreated {
		t.Fatalf("create user = %d %s", helperRR.Code, helperRR.Body.String())
	}
	var helper struct {
		ID      string `json:"id"`
		IsAdmin bool   `json:"is_admin"`
	}
	if err := json.Unmarshal(helperRR.Body.Bytes(), &helper); err != nil {
		t.Fatal(err)
	}
	if helper.IsAdmin {
		t.Fatal("new user should not be admin")
	}

	login := do(t, handler, http.MethodPost, "/api/auth/login", "", map[string]string{
		"username": "helper",
		"password": "password12",
	})
	if login.Code != http.StatusOK {
		t.Fatalf("helper login = %d %s", login.Code, login.Body.String())
	}
	helperCookie := sessionCookie(t, login).Value
	if code := do(t, handler, http.MethodGet, "/api/auth/users", helperCookie, nil).Code; code != http.StatusForbidden {
		t.Fatalf("non-admin list = %d", code)
	}

	list := do(t, handler, http.MethodGet, "/api/auth/users", adminCookie, nil)
	if list.Code != http.StatusOK {
		t.Fatalf("list = %d", list.Code)
	}
	if strings.Contains(list.Body.String(), "password12") || strings.Contains(list.Body.String(), "$2") {
		t.Fatalf("user list leaked a secret: %s", list.Body.String())
	}

	changed := do(t, handler, http.MethodPost, "/api/auth/password", helperCookie, map[string]string{
		"current_password": "password12",
		"new_password":     "better-password",
	})
	if changed.Code != http.StatusNoContent {
		t.Fatalf("change password = %d %s", changed.Code, changed.Body.String())
	}
	if code := do(t, handler, http.MethodPost, "/api/auth/login", "", map[string]string{
		"username": "helper",
		"password": "password12",
	}).Code; code != http.StatusUnauthorized {
		t.Fatalf("old password login = %d", code)
	}

	if code := do(t, handler, http.MethodDelete, "/api/auth/users/"+admin.User.ID, adminCookie, nil).Code; code != http.StatusForbidden {
		t.Fatalf("delete last admin = %d", code)
	}
	if code := do(t, handler, http.MethodDelete, "/api/auth/users/"+helper.ID, adminCookie, nil).Code; code != http.StatusNoContent {
		t.Fatalf("delete helper = %d", code)
	}
}

func TestClientIP_forwardedChain(t *testing.T) {
	trust := proxyTrust{
		mode:     trustExplicit,
		loopback: true,
		nets: []*net.IPNet{
			mustProxyNet(t, "10.0.0.0/8"),
			mustProxyNet(t, "172.16.0.0/12"),
			mustProxyNet(t, "192.168.0.0/16"),
			mustProxyNet(t, "100.64.0.0/10"),
			mustProxyNet(t, "fc00::/7"),
			mustProxyNet(t, "fe80::/10"),
		},
	}

	// Rightmost untrusted address wins. A forged address to the left is ignored,
	// and trusted hops (the proxy itself) are skipped.
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.RemoteAddr = "172.17.0.1:443"
	req.Header.Set("X-Forwarded-For", "198.51.100.9, 203.0.113.50, 10.0.0.8")
	if got, distinct := clientIP(req, trust); got != "203.0.113.50" || !distinct {
		t.Fatalf("forwarded chain = %s distinct=%v", got, distinct)
	}

	// X-Real-IP must not override a resolved forwarding chain.
	req.Header.Set("X-Real-IP", "198.51.100.8")
	if got, _ := clientIP(req, trust); got != "203.0.113.50" {
		t.Fatalf("x-real-ip overrode x-forwarded-for: %s", got)
	}
	req.Header.Del("X-Real-IP")

	// Every hop is in a trusted range, so the original (leftmost) client remains.
	req.Header.Set("X-Forwarded-For", "192.168.1.40, 172.17.0.2")
	if got, distinct := clientIP(req, trust); got != "192.168.1.40" || !distinct {
		t.Fatalf("lan client = %s distinct=%v", got, distinct)
	}

	req.Header.Del("X-Forwarded-For")
	req.Header.Set("X-Real-IP", "203.0.113.70")
	if got := mustClientIP(t, req, trust); got != "203.0.113.70" {
		t.Fatalf("x-real-ip = %s", got)
	}

	req.Header.Del("X-Real-IP")
	req.Header.Set("X-Forwarded-For", "bogus, [2001:db8::5]")
	if got := mustClientIP(t, req, trust); got != "2001:db8::5" {
		t.Fatalf("ipv6 forwarded = %s", got)
	}

	// IPv4-mapped Docker bridge addresses are still private peers.
	mapped := httptest.NewRequest(http.MethodGet, "/health", nil)
	mapped.RemoteAddr = "[::ffff:172.17.0.1]:9"
	mapped.Header.Set("X-Forwarded-For", " 203.0.113.77 ")
	if got := mustClientIP(t, mapped, trust); got != "203.0.113.77" {
		t.Fatalf("mapped peer = %s", got)
	}

	// Default trust (no explicit list) includes Docker bridge and CGNAT peers.
	defaults := parseProxyTrust("")
	bare := httptest.NewRequest(http.MethodGet, "/health", nil)
	bare.RemoteAddr = "172.17.0.1:9"
	bare.Header.Set("X-Forwarded-For", "198.51.100.4, 203.0.113.15")
	if got := mustClientIP(t, bare, defaults); got != "203.0.113.15" {
		t.Fatalf("default private peer = %s", got)
	}
	bare.RemoteAddr = "[fd00::20]:9"
	bare.Header.Set("CF-Connecting-IP", "2001:db8::20")
	bare.Header.Set("X-Forwarded-For", "198.51.100.1")
	if got := mustClientIP(t, bare, defaults); got != "2001:db8::20" {
		t.Fatalf("cf from ula peer = %s", got)
	}

	// Spoofed headers from a public peer never replace the TCP address.
	spoof := httptest.NewRequest(http.MethodGet, "/health", nil)
	spoof.RemoteAddr = "203.0.113.10:4000"
	spoof.Header.Set("CF-Connecting-IP", "198.51.100.2")
	spoof.Header.Set("X-Forwarded-For", "198.51.100.3")
	spoof.Header.Set("X-Real-IP", "198.51.100.4")
	if got, distinct := clientIP(spoof, defaults); got != "203.0.113.10" || distinct {
		t.Fatalf("untrusted spoof = %s distinct=%v", got, distinct)
	}
}

func mustClientIP(t *testing.T, r *http.Request, trust proxyTrust) string {
	t.Helper()
	ip, distinct := clientIP(r, trust)
	if !distinct {
		t.Fatalf("expected a forwarded client, got peer %s", ip)
	}
	return ip
}

func mustProxyNet(t *testing.T, cidr string) *net.IPNet {
	t.Helper()
	network, err := parseTrustedNet(cidr)
	if err != nil {
		t.Fatal(err)
	}
	return network
}

func TestClientIP_trustBoundary(t *testing.T) {
	trust := parseProxyTrust("")
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.RemoteAddr = "198.51.100.20:4000"
	req.Header.Set("CF-Connecting-IP", "203.0.113.9")
	req.Header.Set("X-Forwarded-For", "203.0.113.8")
	if got, distinct := clientIP(req, trust); got != "198.51.100.20" || distinct {
		t.Fatalf("untrusted client ip = %s distinct=%v", got, distinct)
	}

	req.RemoteAddr = "127.0.0.1:9"
	if got := mustClientIP(t, req, trust); got != "203.0.113.9" {
		t.Fatalf("cf connecting ip = %s", got)
	}
	req.Header.Del("CF-Connecting-IP")
	if got := mustClientIP(t, req, trust); got != "203.0.113.8" {
		t.Fatalf("x-forwarded-for = %s", got)
	}

	if requestIsHTTPS(req, trust) {
		t.Fatal("http loopback without forwarded proto is not https")
	}
	req.Header.Set("X-Forwarded-Proto", "https")
	if !requestIsHTTPS(req, trust) {
		t.Fatal("trusted proxy https should set Secure")
	}
	req.RemoteAddr = "198.51.100.20:9"
	if requestIsHTTPS(req, trust) {
		t.Fatal("untrusted peer must ignore X-Forwarded-Proto")
	}
	if sessionCookieSecure(req, trust, cookieSecureOn) != true {
		t.Fatal("COOKIE_SECURE=true forces Secure")
	}
	if sessionCookieSecure(req, trust, cookieSecureOff) {
		t.Fatal("COOKIE_SECURE=false clears Secure")
	}
}

type statusBody struct {
	Authenticated bool `json:"authenticated"`
	SetupRequired bool `json:"setup_required"`
	User          *struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		IsAdmin  bool   `json:"is_admin"`
	} `json:"user"`
}

func newAuthAPI(t *testing.T) (*service.AuthService, http.Handler, *realtime.Hub) {
	t.Helper()
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	store := storage.NewLocalStorage(t.TempDir())
	repos := repository.NewRepositories(db)
	hub := realtime.NewHub()
	go hub.Run()
	t.Cleanup(hub.Stop)
	mgr := printer.NewManager()
	services := service.NewServices(repos, store, mgr, hub)
	services.Auth.SetHashCost(bcrypt.MinCost)
	services.Etsy = service.NewEtsyService(repos.Etsy, "", "", services.Settings)
	return services.Auth, NewRouter(services, hub), hub
}

func do(t *testing.T, handler http.Handler, method, path, cookie string, body any) *httptest.ResponseRecorder {
	t.Helper()
	return doFrom(t, handler, method, path, cookie, body, "", nil)
}

func doFrom(t *testing.T, handler http.Handler, method, path, cookie string, body any, remote string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	switch v := body.(type) {
	case nil:
	case []byte:
		reader = bytes.NewReader(v)
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != "" {
		req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: cookie})
	}
	if remote != "" {
		req.RemoteAddr = remote
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func closeBody(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
}

func sessionCookie(t *testing.T, rr *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	res := rr.Result()
	defer res.Body.Close()
	for _, cookie := range res.Cookies() {
		if cookie.Name == SessionCookieName {
			return cookie
		}
	}
	t.Fatalf("missing session cookie, headers %v", rr.Header())
	return nil
}

func decodeStatus(t *testing.T, rr *httptest.ResponseRecorder) statusBody {
	t.Helper()
	var body statusBody
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode status %d: %v body %s", rr.Code, err, rr.Body.String())
	}
	return body
}
