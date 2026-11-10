package service

import (
	"context"
	"database/sql"
	"net/url"
	"testing"

	"github.com/cadimartech/rankos/internal/database"
	"github.com/cadimartech/rankos/internal/repository"
)

func openEtsyTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestEtsyService_Configure_valid(t *testing.T) {
	db := openEtsyTestDB(t)
	// Use exported field via repository.NewRepositories to get properly wired repos
	repos := repository.NewRepositories(db)
	etsyRepo := repos.Etsy
	settingsRepo := repos.Settings

	settingsSvc := &SettingsService{repo: settingsRepo}
	svc := NewEtsyService(etsyRepo, "", "", settingsSvc)

	if svc.IsConfigured() {
		t.Fatal("expected service to not be configured initially")
	}

	err := svc.Configure(context.Background(), "test-client-id", "http://localhost:8080/callback")
	if err != nil {
		t.Fatalf("Configure returned error: %v", err)
	}

	if !svc.IsConfigured() {
		t.Fatal("expected service to be configured after Configure()")
	}

	// Verify settings were persisted
	setting, err := settingsSvc.Get(context.Background(), "etsy_client_id")
	if err != nil {
		t.Fatalf("Get etsy_client_id: %v", err)
	}
	if setting == nil || setting.Value != "test-client-id" {
		t.Fatalf("expected etsy_client_id to be 'test-client-id', got %v", setting)
	}

	uriSetting, err := settingsSvc.Get(context.Background(), "etsy_redirect_uri")
	if err != nil {
		t.Fatalf("Get etsy_redirect_uri: %v", err)
	}
	if uriSetting == nil || uriSetting.Value != "http://localhost:8080/callback" {
		t.Fatalf("expected etsy_redirect_uri to be 'http://localhost:8080/callback', got %v", uriSetting)
	}
}

func TestEtsyService_Configure_emptyClientID(t *testing.T) {
	db := openEtsyTestDB(t)
	repos := repository.NewRepositories(db)

	settingsSvc := &SettingsService{repo: repos.Settings}
	svc := NewEtsyService(repos.Etsy, "", "", settingsSvc)

	err := svc.Configure(context.Background(), "", "http://localhost:8080/callback")
	if err == nil {
		t.Fatal("expected error for empty client_id")
	}

	if svc.IsConfigured() {
		t.Fatal("expected service to remain unconfigured after empty client_id")
	}
}

func TestEtsyService_Configure_defaultRedirectURI(t *testing.T) {
	db := openEtsyTestDB(t)
	repos := repository.NewRepositories(db)

	settingsSvc := &SettingsService{repo: repos.Settings}
	svc := NewEtsyService(repos.Etsy, "", "", settingsSvc)

	err := svc.Configure(context.Background(), "test-client-id", "")
	if err != nil {
		t.Fatalf("Configure returned error: %v", err)
	}

	if !svc.IsConfigured() {
		t.Fatal("expected service to be configured")
	}

	// Verify default redirect URI was used
	uriSetting, err := settingsSvc.Get(context.Background(), "etsy_redirect_uri")
	if err != nil {
		t.Fatalf("Get etsy_redirect_uri: %v", err)
	}
	if uriSetting == nil || uriSetting.Value != DefaultEtsyRedirectURI {
		t.Fatalf("expected default redirect URI, got %v", uriSetting)
	}
	if svc.RedirectURI() != DefaultEtsyRedirectURI {
		t.Fatalf("RedirectURI() = %q, want default", svc.RedirectURI())
	}
	if got := redirectQueryFromAuthURL(t, svc); got != DefaultEtsyRedirectURI {
		t.Fatalf("oauth redirect_uri = %q, want default", got)
	}
}

func TestResolveStartupRedirectURI(t *testing.T) {
	const envURI = "https://prints.example/api/integrations/etsy/callback"
	const storedURI = "https://shop.example/etsy/callback"

	tests := []struct {
		name        string
		envRedirect string
		stored      string
		want        string
	}{
		{name: "env wins over stored", envRedirect: envURI, stored: storedURI, want: envURI},
		{name: "env wins over stored localhost", envRedirect: envURI, stored: DefaultEtsyRedirectURI, want: envURI},
		{name: "stored used when env unset", envRedirect: "", stored: storedURI, want: storedURI},
		{name: "default when both empty", envRedirect: "", stored: "", want: DefaultEtsyRedirectURI},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveStartupRedirectURI(tt.envRedirect, tt.stored)
			if got != tt.want {
				t.Fatalf("ResolveStartupRedirectURI(%q, %q) = %q, want %q", tt.envRedirect, tt.stored, got, tt.want)
			}
		})
	}
}

func TestEtsyService_Configure_emptyKeepsEnvDefault(t *testing.T) {
	const envURI = "https://prints.example/api/integrations/etsy/callback"
	db := openEtsyTestDB(t)
	repos := repository.NewRepositories(db)
	settingsSvc := &SettingsService{repo: repos.Settings}
	svc := NewEtsyService(repos.Etsy, "", envURI, settingsSvc)

	if err := svc.Configure(context.Background(), "test-client-id", ""); err != nil {
		t.Fatalf("Configure returned error: %v", err)
	}

	uriSetting, err := settingsSvc.Get(context.Background(), "etsy_redirect_uri")
	if err != nil {
		t.Fatalf("Get etsy_redirect_uri: %v", err)
	}
	if uriSetting == nil || uriSetting.Value != envURI {
		t.Fatalf("empty redirect stored %v, want env default %q", uriSetting, envURI)
	}
	if svc.RedirectURI() != envURI {
		t.Fatalf("RedirectURI() = %q, want %q", svc.RedirectURI(), envURI)
	}
	if got := redirectQueryFromAuthURL(t, svc); got != envURI {
		t.Fatalf("oauth redirect_uri = %q, want %q", got, envURI)
	}
}

func TestEtsyService_Configure_explicitWinsAndEmptyKeepsIt(t *testing.T) {
	const envURI = "https://prints.example/api/integrations/etsy/callback"
	const explicit = "https://shop.example/etsy/callback"
	db := openEtsyTestDB(t)
	repos := repository.NewRepositories(db)
	settingsSvc := &SettingsService{repo: repos.Settings}
	svc := NewEtsyService(repos.Etsy, "", envURI, settingsSvc)

	if err := svc.Configure(context.Background(), "test-client-id", explicit); err != nil {
		t.Fatalf("Configure explicit: %v", err)
	}
	if svc.RedirectURI() != explicit {
		t.Fatalf("RedirectURI() = %q, want explicit %q", svc.RedirectURI(), explicit)
	}

	// A later save that omits redirect_uri must not fall back to localhost or the env default.
	if err := svc.Configure(context.Background(), "test-client-id", ""); err != nil {
		t.Fatalf("Configure empty: %v", err)
	}
	uriSetting, err := settingsSvc.Get(context.Background(), "etsy_redirect_uri")
	if err != nil {
		t.Fatalf("Get etsy_redirect_uri: %v", err)
	}
	if uriSetting == nil || uriSetting.Value != explicit {
		t.Fatalf("empty redirect overwrote effective URI with %v, want %q", uriSetting, explicit)
	}
	if got := redirectQueryFromAuthURL(t, svc); got != explicit {
		t.Fatalf("oauth redirect_uri = %q, want %q", got, explicit)
	}
}

func TestEtsyService_Configure_startupEnvOverridesStoredLocalhost(t *testing.T) {
	const envURI = "https://prints.example/api/integrations/etsy/callback"
	db := openEtsyTestDB(t)
	repos := repository.NewRepositories(db)
	settingsSvc := &SettingsService{repo: repos.Settings}
	svc := NewEtsyService(repos.Etsy, "", envURI, settingsSvc)

	if err := settingsSvc.Set(context.Background(), "etsy_redirect_uri", DefaultEtsyRedirectURI); err != nil {
		t.Fatalf("seed stored redirect: %v", err)
	}
	stored, err := settingsSvc.Get(context.Background(), "etsy_redirect_uri")
	if err != nil {
		t.Fatalf("Get etsy_redirect_uri: %v", err)
	}

	redirectURI := ResolveStartupRedirectURI(envURI, stored.Value)
	if err := svc.Configure(context.Background(), "test-client-id", redirectURI); err != nil {
		t.Fatalf("Configure: %v", err)
	}
	if svc.RedirectURI() != envURI {
		t.Fatalf("RedirectURI() = %q, want env %q", svc.RedirectURI(), envURI)
	}
	uriSetting, err := settingsSvc.Get(context.Background(), "etsy_redirect_uri")
	if err != nil {
		t.Fatalf("Get etsy_redirect_uri: %v", err)
	}
	if uriSetting == nil || uriSetting.Value != envURI {
		t.Fatalf("stored redirect = %v, want env %q", uriSetting, envURI)
	}
	if got := redirectQueryFromAuthURL(t, svc); got != envURI {
		t.Fatalf("oauth redirect_uri = %q, want %q", got, envURI)
	}
}

func TestEtsyService_Configure_startupUsesStoredWhenEnvUnset(t *testing.T) {
	const storedURI = "https://shop.example/etsy/callback"
	db := openEtsyTestDB(t)
	repos := repository.NewRepositories(db)
	settingsSvc := &SettingsService{repo: repos.Settings}
	svc := NewEtsyService(repos.Etsy, "", "", settingsSvc)

	redirectURI := ResolveStartupRedirectURI("", storedURI)
	if err := svc.Configure(context.Background(), "test-client-id", redirectURI); err != nil {
		t.Fatalf("Configure stored: %v", err)
	}
	if err := svc.Configure(context.Background(), "test-client-id", ""); err != nil {
		t.Fatalf("Configure empty: %v", err)
	}
	if svc.RedirectURI() != storedURI {
		t.Fatalf("RedirectURI() = %q, want stored %q", svc.RedirectURI(), storedURI)
	}
	uriSetting, err := settingsSvc.Get(context.Background(), "etsy_redirect_uri")
	if err != nil {
		t.Fatalf("Get etsy_redirect_uri: %v", err)
	}
	if uriSetting == nil || uriSetting.Value != storedURI {
		t.Fatalf("stored redirect = %v, want %q", uriSetting, storedURI)
	}
}

func redirectQueryFromAuthURL(t *testing.T, svc *EtsyService) string {
	t.Helper()
	authURL, err := svc.StartOAuth(context.Background())
	if err != nil {
		t.Fatalf("StartOAuth: %v", err)
	}
	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("parse auth URL: %v", err)
	}
	return parsed.Query().Get("redirect_uri")
}

func TestEtsyService_Configure_noSettingsService(t *testing.T) {
	svc := NewEtsyService(nil, "", "", nil)

	err := svc.Configure(context.Background(), "test-client-id", "http://localhost:8080/callback")
	if err != nil {
		t.Fatalf("Configure without settingsSvc returned error: %v", err)
	}

	if !svc.IsConfigured() {
		t.Fatal("expected service to be configured even without settingsSvc")
	}
}
