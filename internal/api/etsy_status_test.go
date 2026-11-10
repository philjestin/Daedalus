package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cadimartech/rankos/internal/database"
	"github.com/cadimartech/rankos/internal/repository"
	"github.com/cadimartech/rankos/internal/service"
)

func TestEtsyGetStatus_includesEffectiveRedirectURI(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	const redirect = "https://prints.example/api/integrations/etsy/callback"
	repos := repository.NewRepositories(db)
	svc := service.NewEtsyService(repos.Etsy, "", redirect, nil)
	handler := NewEtsyHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/integrations/etsy/status", nil)
	rr := httptest.NewRecorder()
	handler.GetStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if body["redirect_uri"] != redirect {
		t.Fatalf("redirect_uri = %v, want %q", body["redirect_uri"], redirect)
	}
	if body["configured"] != false {
		t.Fatalf("configured = %v, want false before a client ID is saved", body["configured"])
	}
	if body["connected"] != false {
		t.Fatalf("connected = %v, want false", body["connected"])
	}
}
