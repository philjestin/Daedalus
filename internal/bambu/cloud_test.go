package bambu

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// cloudflareBlockPage is the shape of the HTML page Bambu's Cloudflare
// returns to a blocked server (status 403). The Ray ID is the one reported
// with this bug; it is not a credential.
const cloudflareBlockPage = `<!DOCTYPE html>
<html lang="en">
<head><title>Attention Required! | Cloudflare</title></head>
<body>
<h1>Sorry, you have been blocked</h1>
<h2>You are unable to access bambulab.com</h2>
<p>Why have I been blocked?</p>
<p>Cloudflare Ray ID: <span>a588c18a6b56efa6</span></p>
</body>
</html>`

const wantBlockedMessage = "Bambu Cloud rejected the request (blocked by Bambu's Cloudflare). Use LAN mode with IP, serial and Access Code."

func TestLogin_cloudflareBlockPage(t *testing.T) {
	var gotMethod, gotPath, gotContentType, gotUserAgent, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		gotUserAgent = r.Header.Get("User-Agent")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, cloudflareBlockPage)
	}))
	t.Cleanup(srv.Close)

	c := NewCloudClient()
	c.baseURL = srv.URL
	c.httpClient = srv.Client()

	_, err := c.Login("user@example.com", "secret")
	if err == nil {
		t.Fatal("Login returned nil error for a Cloudflare HTML block")
	}
	if err.Error() != wantBlockedMessage {
		t.Fatalf("error = %q, want %q", err.Error(), wantBlockedMessage)
	}
	if strings.Contains(err.Error(), "<") || strings.Contains(err.Error(), "a588c18a6b56efa6") || strings.Contains(strings.ToLower(err.Error()), "<html") {
		t.Fatalf("error leaked the block page: %s", err.Error())
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/v1/user-service/user/login" {
		t.Errorf("path = %s, want /v1/user-service/user/login", gotPath)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %s, want application/json", gotContentType)
	}
	if gotUserAgent == "" {
		t.Error("User-Agent was empty")
	}
	if strings.Contains(strings.ToLower(gotUserAgent), "mozilla") {
		t.Errorf("User-Agent looks like a browser impersonation: %s", gotUserAgent)
	}
	if !strings.Contains(gotBody, `"account":"user@example.com"`) || !strings.Contains(gotBody, `"password":"secret"`) {
		t.Errorf("body = %s, want account and password JSON", gotBody)
	}
}

func TestLogin_htmlChallengeOn200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, cloudflareBlockPage)
	}))
	t.Cleanup(srv.Close)

	c := NewCloudClient()
	c.baseURL = srv.URL
	c.httpClient = srv.Client()

	_, err := c.Login("user@example.com", "secret")
	if err == nil {
		t.Fatal("Login returned nil error for an HTML body")
	}
	if err.Error() != wantBlockedMessage {
		t.Fatalf("error = %q, want %q", err.Error(), wantBlockedMessage)
	}
	if strings.Contains(err.Error(), "DOCTYPE") || strings.Contains(err.Error(), "a588c18a6b56efa6") {
		t.Fatalf("error leaked the block page: %s", err.Error())
	}
}

func TestGetDevices_cloudflareBlockPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, cloudflareBlockPage)
	}))
	t.Cleanup(srv.Close)

	c := NewCloudClient()
	c.baseURL = srv.URL
	c.httpClient = srv.Client()

	_, err := c.GetDevices("token")
	if err == nil {
		t.Fatal("GetDevices returned nil error for a Cloudflare HTML block")
	}
	if err.Error() != wantBlockedMessage {
		t.Fatalf("error = %q, want %q", err.Error(), wantBlockedMessage)
	}
}

func TestLogin_successParsesToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/user-service/user/login" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"accessToken":"tok","refreshToken":"ref","expiresIn":7776000,"loginType":"verifyCode"}`)
	}))
	t.Cleanup(srv.Close)

	c := NewCloudClient()
	c.baseURL = srv.URL
	c.httpClient = srv.Client()

	resp, err := c.Login("user@example.com", "secret")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if resp.AccessToken != "tok" || resp.RefreshToken != "ref" || resp.ExpiresIn != 7776000 || resp.LoginType != "verifyCode" {
		t.Fatalf("parsed response = %+v", resp)
	}
}

func TestLogin_jsonAPIErrorStaysJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"code":2,"message":"Invalid credentials"}`)
	}))
	t.Cleanup(srv.Close)

	c := NewCloudClient()
	c.baseURL = srv.URL
	c.httpClient = srv.Client()

	_, err := c.Login("user@example.com", "wrong")
	if err == nil {
		t.Fatal("expected an error for a JSON API failure")
	}
	if err.Error() == wantBlockedMessage {
		t.Fatalf("JSON API error was reported as a Cloudflare block: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "Invalid credentials") {
		t.Fatalf("error = %q, want it to include the API message", err.Error())
	}
	if strings.Contains(err.Error(), "<") {
		t.Fatalf("error contained markup: %s", err.Error())
	}
}
