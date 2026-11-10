package api

import (
	"net/http"
	"testing"

	"github.com/cadimartech/rankos/internal/database"
	"github.com/cadimartech/rankos/internal/printer"
	"github.com/cadimartech/rankos/internal/realtime"
	"github.com/cadimartech/rankos/internal/repository"
	"github.com/cadimartech/rankos/internal/service"
	"github.com/cadimartech/rankos/internal/storage"
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

	router := NewRouter(services, hub)

	return &testEnv{
		handler:  router,
		services: services,
		storage:  store,
	}
}
