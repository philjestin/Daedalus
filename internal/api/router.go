package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/hyperion/printfarm/internal/realtime"
	"github.com/hyperion/printfarm/internal/service"
)

// NewRouter creates the HTTP router with all routes.
func NewRouter(services *service.Services, hub *realtime.Hub) http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:*", "http://127.0.0.1:*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	

	// WebSocket endpoint
	r.Get("/ws", hub.HandleWebSocket)

	// API routes
	r.Route("/api", func(r chi.Router) {
		// Projects
		projectHandler := &ProjectHandler{service: services.Projects}
		r.Route("/projects", func(r chi.Router) {
			r.Get("/", projectHandler.List)
			r.Post("/", projectHandler.Create)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", projectHandler.Get)
				r.Patch("/", projectHandler.Update)
				r.Delete("/", projectHandler.Delete)

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
				r.Post("/start", printJobHandler.Start)
				r.Post("/pause", printJobHandler.Pause)
				r.Post("/resume", printJobHandler.Resume)
				r.Post("/cancel", printJobHandler.Cancel)
				r.Post("/outcome", printJobHandler.RecordOutcome)
			})
		})

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
	})

	return r
}

