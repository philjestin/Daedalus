package api

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/hyperion/printfarm/internal/realtime"
	"github.com/hyperion/printfarm/internal/service"
)

// RouterConfig holds configuration for the router.
type RouterConfig struct {
	RequireAuth bool // If true, protect all API routes with authentication
}

// NewRouter creates the HTTP router with all routes.
func NewRouter(services *service.Services, hub *realtime.Hub) http.Handler {
	return NewRouterWithConfig(services, hub, RouterConfig{RequireAuth: false})
}

// NewRouterWithConfig creates the HTTP router with configuration.
func NewRouterWithConfig(services *service.Services, hub *realtime.Hub, config RouterConfig) http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	// Configure CORS - allow all origins for desktop app (local API only)
	r.Use(cors.Handler(cors.Options{
		AllowOriginFunc: func(r *http.Request, origin string) bool {
			return true // Desktop app - API is local only
		},
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders: []string{"Link"},
		MaxAge:         300,
	}))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// WebSocket endpoint
	r.Get("/ws", hub.HandleWebSocket)

	// Auth middleware (only if auth service is configured)
	var authMiddleware *AuthMiddleware
	if services.Auth != nil {
		authMiddleware = NewAuthMiddleware(services.Auth)
	}

	// API routes
	r.Route("/api", func(r chi.Router) {
		// Auth routes (always public - defined in separate group)
		if services.Auth != nil {
			authHandler := NewAuthHandler(services.Auth)
			// Public auth endpoints
			r.Post("/auth/request-link", authHandler.RequestMagicLink)
			r.Get("/auth/verify", authHandler.VerifyMagicLink)

			// Protected auth endpoints (need their own group with middleware)
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.RequireAuth)
				r.Get("/auth/me", authHandler.GetCurrentUser)
				r.Post("/auth/logout", authHandler.Logout)
			})
		}

		// Protected routes group - all routes below require auth if enabled
		r.Group(func(r chi.Router) {
			// Apply auth middleware if required
			if config.RequireAuth && authMiddleware != nil {
				r.Use(authMiddleware.RequireAuth)
			}

			// Projects
		projectHandler := &ProjectHandler{service: services.Projects}
		r.Route("/projects", func(r chi.Router) {
			r.Get("/", projectHandler.List)
			r.Post("/", projectHandler.Create)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", projectHandler.Get)
				r.Patch("/", projectHandler.Update)
				r.Delete("/", projectHandler.Delete)

				// Project pipeline endpoints
				r.Get("/jobs", projectHandler.ListJobs)
				r.Get("/job-stats", projectHandler.GetJobStats)
				r.Post("/start-production", projectHandler.StartProduction)
				r.Post("/ready-to-ship", projectHandler.MarkReadyToShip)
				r.Post("/ship", projectHandler.Ship)

				// Parts nested under project
				partHandler := &PartHandler{service: services.Parts}
				r.Get("/parts", partHandler.ListByProject)
				r.Post("/parts", partHandler.Create)
			})
		})

		// Parts
		partHandler := &PartHandler{service: services.Parts}
		r.Route("/parts/{id}", func(r chi.Router) {
			r.Get("/", partHandler.Get)
			r.Patch("/", partHandler.Update)
			r.Delete("/", partHandler.Delete)

			// Designs nested under part
			designHandler := &DesignHandler{service: services.Designs}
			r.Get("/designs", designHandler.ListByPart)
			r.Post("/designs", designHandler.Create)
		})

		// Designs
		designHandler := &DesignHandler{service: services.Designs}
		printJobByDesignHandler := &PrintJobHandler{service: services.PrintJobs}
		r.Route("/designs/{id}", func(r chi.Router) {
			r.Get("/", designHandler.Get)
			r.Get("/download", designHandler.Download)
			r.Get("/print-jobs", printJobByDesignHandler.ListByDesign)
		})

		// Printers
		printerHandler := &PrinterHandler{service: services.Printers}
		r.Route("/printers", func(r chi.Router) {
			r.Get("/", printerHandler.List)
			r.Post("/", printerHandler.Create)
			r.Get("/states", printerHandler.GetAllStates)
			r.Post("/discover", printerHandler.Discover) // Network discovery
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", printerHandler.Get)
				r.Patch("/", printerHandler.Update)
				r.Delete("/", printerHandler.Delete)
				r.Get("/state", printerHandler.GetState)
			})
		})

		// Materials
		materialHandler := &MaterialHandler{service: services.Materials}
		r.Route("/materials", func(r chi.Router) {
			r.Get("/", materialHandler.List)
			r.Post("/", materialHandler.Create)
			r.Get("/{id}", materialHandler.Get)
		})

		// Spools
		spoolHandler := &SpoolHandler{service: services.Spools}
		r.Route("/spools", func(r chi.Router) {
			r.Get("/", spoolHandler.List)
			r.Post("/", spoolHandler.Create)
			r.Get("/{id}", spoolHandler.Get)
		})

		// Print Jobs
		printJobHandler := &PrintJobHandler{service: services.PrintJobs}
		r.Route("/print-jobs", func(r chi.Router) {
			r.Get("/", printJobHandler.List)
			r.Post("/", printJobHandler.Create)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", printJobHandler.Get)
				r.Patch("/", printJobHandler.Update)
				r.Get("/preflight", printJobHandler.PreflightCheck)
				r.Post("/start", printJobHandler.Start)
				r.Post("/pause", printJobHandler.Pause)
				r.Post("/resume", printJobHandler.Resume)
				r.Post("/cancel", printJobHandler.Cancel)
				r.Post("/outcome", printJobHandler.RecordOutcome)

				// Job history endpoints
				r.Get("/events", printJobHandler.GetEvents)
				r.Get("/with-events", printJobHandler.GetWithEvents)
				r.Get("/retry-chain", printJobHandler.GetRetryChain)
				r.Post("/retry", printJobHandler.Retry)
				r.Post("/failure", printJobHandler.RecordFailure)
				r.Post("/scrap", printJobHandler.MarkAsScrap)
			})
		})

		// Jobs by recipe (for recipe detail page)
		r.Get("/templates/{id}/jobs", printJobHandler.ListByRecipe)

		// Files
		fileHandler := &FileHandler{service: services.Files}
		r.Get("/files/{id}", fileHandler.Get)

		// Expenses
		expenseHandler := &ExpenseHandler{service: services.Expenses}
		r.Route("/expenses", func(r chi.Router) {
			r.Get("/", expenseHandler.List)
			r.Post("/receipt", expenseHandler.UploadReceipt)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", expenseHandler.Get)
				r.Post("/confirm", expenseHandler.Confirm)
				r.Delete("/", expenseHandler.Delete)
			})
		})

		// Sales
		saleHandler := &SaleHandler{service: services.Sales}
		r.Route("/sales", func(r chi.Router) {
			r.Get("/", saleHandler.List)
			r.Post("/", saleHandler.Create)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", saleHandler.Get)
				r.Patch("/", saleHandler.Update)
				r.Delete("/", saleHandler.Delete)
			})
		})

		// Stats
		statsHandler := &StatsHandler{service: services.Stats}
		r.Get("/stats/financial", statsHandler.GetFinancialSummary)

		// Templates (Recipes)
		templateHandler := &TemplateHandler{service: services.Templates}
		r.Route("/templates", func(r chi.Router) {
			r.Get("/", templateHandler.List)
			r.Post("/", templateHandler.Create)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", templateHandler.Get)
				r.Patch("/", templateHandler.Update)
				r.Delete("/", templateHandler.Delete)
				r.Post("/designs", templateHandler.AddDesign)
				r.Delete("/designs/{designId}", templateHandler.RemoveDesign)
				r.Post("/instantiate", templateHandler.Instantiate)

				// Recipe material endpoints
				r.Get("/materials", templateHandler.ListMaterials)
				r.Post("/materials", templateHandler.AddMaterial)
				r.Patch("/materials/{materialId}", templateHandler.UpdateMaterial)
				r.Delete("/materials/{materialId}", templateHandler.RemoveMaterial)

				// Recipe compatibility endpoints
				r.Get("/compatible-printers", templateHandler.GetCompatiblePrinters)
				r.Get("/compatible-spools", templateHandler.GetCompatibleSpools)
				r.Get("/cost-estimate", templateHandler.GetCostEstimate)
				r.Post("/validate-printer/{printerId}", templateHandler.ValidatePrinter)
			})
		})

			// Bambu Cloud Integration
			if services.BambuCloud != nil {
				bambuCloudHandler := &BambuCloudHandler{service: services.BambuCloud}
				r.Route("/bambu-cloud", func(r chi.Router) {
					r.Post("/login", bambuCloudHandler.Login)
					r.Post("/verify", bambuCloudHandler.Verify)
					r.Get("/status", bambuCloudHandler.Status)
					r.Get("/devices", bambuCloudHandler.Devices)
					r.Post("/devices/add", bambuCloudHandler.AddDevice)
					r.Delete("/logout", bambuCloudHandler.Logout)
				})
			}

			// Etsy Integration
			if services.Etsy != nil {
				etsyHandler := NewEtsyHandler(services.Etsy)
				etsyHandler.SetTemplateSvc(services.Templates)
				r.Route("/integrations/etsy", func(r chi.Router) {
					// Auth
					r.Get("/auth", etsyHandler.StartAuth)
					r.Get("/callback", etsyHandler.Callback)
					r.Get("/status", etsyHandler.GetStatus)
					r.Post("/disconnect", etsyHandler.Disconnect)

					// Receipts/Orders
					r.Post("/receipts/sync", etsyHandler.SyncReceipts)
					r.Get("/receipts", etsyHandler.ListReceipts)
					r.Get("/receipts/{id}", etsyHandler.GetReceipt)
					r.Post("/receipts/{id}/process", etsyHandler.ProcessReceipt)

					// Listings
					r.Post("/listings/sync", etsyHandler.SyncListings)
					r.Get("/listings", etsyHandler.ListListings)
					r.Get("/listings/{id}", etsyHandler.GetListing)
					r.Post("/listings/{id}/link", etsyHandler.LinkListing)
					r.Delete("/listings/{id}/link", etsyHandler.UnlinkListing)
					r.Post("/listings/{id}/sync-inventory", etsyHandler.SyncInventory)

					// Webhooks
					r.Post("/webhook", etsyHandler.HandleWebhook)
					r.Get("/webhook/events", etsyHandler.ListWebhookEvents)
					r.Post("/webhook/events/{id}/reprocess", etsyHandler.ReprocessWebhookEvent)
				})
			}
		}) // End protected routes group
	})

	// Serve static frontend files in production
	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "./web/dist"
	}
	if info, err := os.Stat(staticDir); err == nil && info.IsDir() {
		// Serve static files with SPA fallback
		r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
			// Try to serve the exact file
			path := filepath.Join(staticDir, req.URL.Path)
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				http.ServeFile(w, req, path)
				return
			}
			// Fallback to index.html for SPA routing
			http.ServeFile(w, req, filepath.Join(staticDir, "index.html"))
		})
	}

	return r
}

