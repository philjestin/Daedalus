package service

import (
	"context"
	"database/sql"
	"testing"

	"github.com/hyperion/printfarm/internal/database"
	"github.com/hyperion/printfarm/internal/repository"
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
	etsyRepo := &repository.EtsyRepository{}
	settingsRepo := &repository.SettingsRepository{}
	// Use exported field via repository.NewRepositories to get properly wired repos
	repos := repository.NewRepositories(db)
	etsyRepo = repos.Etsy
	settingsRepo = repos.Settings

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
	if uriSetting == nil || uriSetting.Value != "http://localhost:8080/api/integrations/etsy/callback" {
		t.Fatalf("expected default redirect URI, got %v", uriSetting)
	}
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
