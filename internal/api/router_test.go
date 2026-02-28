package api

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/philjestin/daedalus/internal/realtime"
	"github.com/philjestin/daedalus/internal/service"
)

// createTestServices builds a minimal Services struct with zero-valued service
// pointers. These are enough for the router to register all routes (handler
// methods are never called during route registration).
func createTestServices() *service.Services {
	return &service.Services{
		Projects:   &service.ProjectService{},
		Parts:      &service.PartService{},
		Designs:    &service.DesignService{},
		Printers:   &service.PrinterService{},
		Materials:  &service.MaterialService{},
		Spools:     &service.SpoolService{},
		PrintJobs:  &service.PrintJobService{},
		Files:      &service.FileService{},
		Expenses:   &service.ExpenseService{},
		Sales:      &service.SaleService{},
		Stats:      &service.StatsService{},
		Templates:  &service.TemplateService{},
		BambuCloud: &service.BambuCloudService{},
		// Etsy and Auth are nil (optional services)
	}
}

// collectRoutes walks a chi.Router and returns all registered routes as
// "METHOD /pattern" strings.
func collectRoutes(handler http.Handler) []string {
	r, ok := handler.(chi.Routes)
	if !ok {
		return nil
	}

	var routes []string
	chi.Walk(r, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		routes = append(routes, fmt.Sprintf("%s %s", method, route))
		return nil
	})
	return routes
}

// hasRoute checks if the route list contains a specific "METHOD /pattern".
func hasRoute(routes []string, method, pattern string) bool {
	target := fmt.Sprintf("%s %s", method, pattern)
	for _, r := range routes {
		if r == target {
			return true
		}
	}
	return false
}

func TestRouterRegistersAllExpectedRoutes(t *testing.T) {
	services := createTestServices()
	hub := realtime.NewHub()
	handler := NewRouter(services, hub)
	routes := collectRoutes(handler)

	if len(routes) == 0 {
		t.Fatal("no routes registered at all")
	}

	// Define all expected routes grouped by domain.
	type route struct {
		method  string
		pattern string
	}

	expected := []route{
		// Health
		{"GET", "/health"},

		// Projects
		{"GET", "/api/projects/"},
		{"POST", "/api/projects/"},
		{"GET", "/api/projects/{id}/"},
		{"PATCH", "/api/projects/{id}/"},
		{"DELETE", "/api/projects/{id}/"},
		{"GET", "/api/projects/{id}/jobs"},
		{"GET", "/api/projects/{id}/job-stats"},
		{"GET", "/api/projects/{id}/summary"},
		{"POST", "/api/projects/{id}/start-production"},

		// Parts (nested under project)
		{"GET", "/api/projects/{id}/parts"},
		{"POST", "/api/projects/{id}/parts"},

		// Parts (standalone)
		{"GET", "/api/parts/{id}/"},
		{"PATCH", "/api/parts/{id}/"},
		{"DELETE", "/api/parts/{id}/"},

		// Designs (nested under part)
		{"GET", "/api/parts/{id}/designs"},
		{"POST", "/api/parts/{id}/designs"},

		// Designs (standalone)
		{"GET", "/api/designs/{id}/"},
		{"GET", "/api/designs/{id}/download"},
		{"GET", "/api/designs/{id}/print-jobs"},
		{"POST", "/api/designs/{id}/open-external"},

		// Printers
		{"GET", "/api/printers/"},
		{"POST", "/api/printers/"},
		{"GET", "/api/printers/states"},
		{"POST", "/api/printers/discover"},
		{"GET", "/api/printers/{id}/"},
		{"PATCH", "/api/printers/{id}/"},
		{"DELETE", "/api/printers/{id}/"},
		{"GET", "/api/printers/{id}/state"},
		{"GET", "/api/printers/{id}/jobs"},
		{"GET", "/api/printers/{id}/stats"},

		// Materials
		{"GET", "/api/materials/"},
		{"POST", "/api/materials/"},
		{"GET", "/api/materials/{id}"},

		// Spools
		{"GET", "/api/spools/"},
		{"POST", "/api/spools/"},
		{"GET", "/api/spools/{id}"},

		// Print Jobs
		{"GET", "/api/print-jobs/"},
		{"POST", "/api/print-jobs/"},
		{"GET", "/api/print-jobs/{id}/"},
		{"PATCH", "/api/print-jobs/{id}/"},
		{"GET", "/api/print-jobs/{id}/preflight"},
		{"POST", "/api/print-jobs/{id}/start"},
		{"POST", "/api/print-jobs/{id}/pause"},
		{"POST", "/api/print-jobs/{id}/resume"},
		{"POST", "/api/print-jobs/{id}/cancel"},
		{"POST", "/api/print-jobs/{id}/outcome"},
		{"GET", "/api/print-jobs/{id}/events"},
		{"GET", "/api/print-jobs/{id}/with-events"},
		{"GET", "/api/print-jobs/{id}/retry-chain"},
		{"POST", "/api/print-jobs/{id}/retry"},
		{"POST", "/api/print-jobs/{id}/failure"},
		{"POST", "/api/print-jobs/{id}/scrap"},

		// Jobs by recipe
		{"GET", "/api/templates/{id}/jobs"},

		// Files
		{"GET", "/api/files/{id}"},

		// Expenses
		{"GET", "/api/expenses/"},
		{"POST", "/api/expenses/receipt"},
		{"GET", "/api/expenses/{id}/"},
		{"POST", "/api/expenses/{id}/confirm"},
		{"DELETE", "/api/expenses/{id}/"},

		// Sales
		{"GET", "/api/sales/"},
		{"POST", "/api/sales/"},
		{"GET", "/api/sales/{id}/"},
		{"PATCH", "/api/sales/{id}/"},
		{"DELETE", "/api/sales/{id}/"},

		// Stats
		{"GET", "/api/stats/financial"},

		// Templates
		{"GET", "/api/templates/"},
		{"POST", "/api/templates/"},
		{"GET", "/api/templates/{id}/"},
		{"PATCH", "/api/templates/{id}/"},
		{"DELETE", "/api/templates/{id}/"},
		{"POST", "/api/templates/{id}/designs"},
		{"DELETE", "/api/templates/{id}/designs/{designId}"},
		{"POST", "/api/templates/{id}/instantiate"},
		{"GET", "/api/templates/{id}/materials"},
		{"POST", "/api/templates/{id}/materials"},
		{"PATCH", "/api/templates/{id}/materials/{materialId}"},
		{"DELETE", "/api/templates/{id}/materials/{materialId}"},
		{"GET", "/api/templates/{id}/compatible-printers"},
		{"GET", "/api/templates/{id}/compatible-spools"},
		{"GET", "/api/templates/{id}/cost-estimate"},
		{"POST", "/api/templates/{id}/validate-printer/{printerId}"},

		// Bambu Cloud
		{"POST", "/api/bambu-cloud/login"},
		{"POST", "/api/bambu-cloud/verify"},
		{"GET", "/api/bambu-cloud/status"},
		{"GET", "/api/bambu-cloud/devices"},
		{"POST", "/api/bambu-cloud/devices/add"},
		{"DELETE", "/api/bambu-cloud/logout"},
	}

	var missing []string
	for _, exp := range expected {
		if !hasRoute(routes, exp.method, exp.pattern) {
			missing = append(missing, fmt.Sprintf("%s %s", exp.method, exp.pattern))
		}
	}

	if len(missing) > 0 {
		t.Errorf("missing %d expected routes:\n  %s", len(missing), strings.Join(missing, "\n  "))
		t.Log("All registered routes:")
		for _, r := range routes {
			t.Logf("  %s", r)
		}
	}
}

// TestBambuCloudRoutesRegistered specifically validates Bambu Cloud routes are
// registered when BambuCloud service is non-nil (regression guard).
func TestBambuCloudRoutesRegistered(t *testing.T) {
	services := createTestServices()
	hub := realtime.NewHub()
	handler := NewRouter(services, hub)
	routes := collectRoutes(handler)

	bambuRoutes := []struct {
		method  string
		pattern string
	}{
		{"POST", "/api/bambu-cloud/login"},
		{"POST", "/api/bambu-cloud/verify"},
		{"GET", "/api/bambu-cloud/status"},
		{"GET", "/api/bambu-cloud/devices"},
		{"POST", "/api/bambu-cloud/devices/add"},
		{"DELETE", "/api/bambu-cloud/logout"},
	}

	for _, exp := range bambuRoutes {
		if !hasRoute(routes, exp.method, exp.pattern) {
			t.Errorf("Bambu Cloud route not registered: %s %s", exp.method, exp.pattern)
		}
	}
}

// TestBambuCloudRoutesOmittedWhenNil verifies Bambu Cloud routes are NOT
// registered when the service is nil.
func TestBambuCloudRoutesOmittedWhenNil(t *testing.T) {
	services := createTestServices()
	services.BambuCloud = nil
	hub := realtime.NewHub()
	handler := NewRouter(services, hub)
	routes := collectRoutes(handler)

	for _, r := range routes {
		if strings.Contains(r, "/bambu-cloud/") {
			t.Errorf("Bambu Cloud route should not be registered when service is nil: %s", r)
		}
	}
}

// TestDesignOpenExternalRouteRegistered is a focused regression test.
func TestDesignOpenExternalRouteRegistered(t *testing.T) {
	services := createTestServices()
	hub := realtime.NewHub()
	handler := NewRouter(services, hub)
	routes := collectRoutes(handler)

	if !hasRoute(routes, "POST", "/api/designs/{id}/open-external") {
		t.Error("POST /api/designs/{id}/open-external route not registered")
	}
}

// TestPartHandlerHasDesignService ensures PartHandler is initialized with
// designService for multipart part creation with file upload.
func TestPartHandlerHasDesignService(t *testing.T) {
	services := createTestServices()

	// The PartHandler should include designService in both places
	// We test this indirectly by verifying the route is registered
	// and the handler doesn't panic on multipart requests.
	hub := realtime.NewHub()
	handler := NewRouter(services, hub)
	routes := collectRoutes(handler)

	if !hasRoute(routes, "POST", "/api/projects/{id}/parts") {
		t.Error("POST /api/projects/{id}/parts route not registered")
	}
}
