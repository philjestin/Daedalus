package api

import (
	"context"
	"net/http"
	"testing"

	"github.com/cadimartech/rankos/internal/database"
	"github.com/cadimartech/rankos/internal/printer"
	"github.com/cadimartech/rankos/internal/realtime"
	"github.com/cadimartech/rankos/internal/repository"
	"github.com/cadimartech/rankos/internal/service"
	"github.com/cadimartech/rankos/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

// testEnv bundles everything needed for handler integration tests.
type testEnv struct {
	handler  http.Handler
	services *service.Services
	storage  *storage.LocalStorage
}

// newTestEnv creates an in-memory SQLite-backed test environment with a real
// HTTP router. Cleanup is registered via t.Cleanup.
func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	// In-memory SQLite
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Temp storage directory
	storageDir := t.TempDir()
	store := storage.NewLocalStorage(storageDir)

	repos := repository.NewRepositories(db)
	hub := realtime.NewHub()
	printerMgr := printer.NewManager()

	services := service.NewServices(repos, store, printerMgr, hub)
	// Ensure BambuCloud service is available (NewServices already sets it).
	if services.Auth != nil {
		services.Auth.SetHashCost(bcrypt.MinCost)
	}

	router := NewRouter(services, hub)

	// Existing handler tests call the router directly. Seed an admin session
	// and attach it when the request did not bring its own cookie.
	_, token, err := services.Auth.Setup(context.Background(), "test-admin", "test-password", "", "127.0.0.1")
	if err != nil {
		t.Fatalf("seed test admin: %v", err)
	}

	return &testEnv{
		handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, cookieErr := r.Cookie(SessionCookieName); cookieErr != nil {
				r.AddCookie(&http.Cookie{Name: SessionCookieName, Value: token})
			}
			router.ServeHTTP(w, r)
		}),
		services: services,
		storage:  store,
	}
}
