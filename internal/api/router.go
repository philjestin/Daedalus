package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/philjestin/daedalus/internal/realtime"
	"github.com/philjestin/daedalus/internal/service"
	"github.com/philjestin/daedalus/internal/version"
)

// NewRouter creates the HTTP router with all routes.
func NewRouter(services *service.Services, hub *realtime.Hub) http.Handler {
	r := chi.NewRouter()

	// Middleware
	sentryMiddleware := sentryhttp.New(sentryhttp.Options{Repanic: true})
	r.Use(sentryMiddleware.Handle)
	r.Use(RequestLogger) // Custom structured logging with request IDs and timing
	r.Use(middleware.Recoverer)
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
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"version": version.Version,
			"commit":  version.Commit,
		})
	})

	// WebSocket endpoint
	r.Get("/ws", hub.HandleWebSocket)

	// Public API routes (no auth required)
	r.Route("/api/public", func(r chi.Router) {
		// Public quote by share token
		if services.Quotes != nil {
			quoteHandler := NewQuoteHandler(services.Quotes)
			r.Get("/quotes/{token}", quoteHandler.GetByShareToken)
		}
		// Public business info
		if services.Settings != nil {
			r.Get("/business-info", func(w http.ResponseWriter, req *http.Request) {
				ctx := req.Context()
				keys := []string{"business_name", "business_address_json", "business_phone", "business_email", "business_website"}
				result := map[string]interface{}{}
				for _, key := range keys {
					setting, err := services.Settings.Get(ctx, key)
					if err == nil && setting != nil {
						result[key] = setting.Value
					}
				}
				respondJSON(w, http.StatusOK, result)
			})
		}
	})

	// API routes
	r.Route("/api", func(r chi.Router) {
		// Projects (Product Catalog)
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
				r.Get("/summary", projectHandler.GetProjectSummary)
				r.Post("/start-production", projectHandler.StartProduction)

				// Tasks for this project
				if services.Tasks != nil {
					taskHandler := NewTaskHandler(services.Tasks)
					r.Get("/tasks", taskHandler.ListByProject)
				}

				// Parts nested under project
				partHandler := &PartHandler{service: services.Parts, designService: services.Designs}
				r.Get("/parts", partHandler.ListByProject)
				r.Post("/parts", partHandler.Create)

				// Supplies nested under project
				supplyHandler := &ProjectSupplyHandler{service: services.ProjectSupplies}
				r.Get("/supplies", supplyHandler.List)
				r.Post("/supplies", supplyHandler.Create)
			})
		})

		// Tasks (Work Instances)
		if services.Tasks != nil {
			taskHandler := NewTaskHandler(services.Tasks)
			r.Route("/tasks", func(r chi.Router) {
				r.Get("/", taskHandler.List)
				r.Post("/", taskHandler.Create)
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", taskHandler.Get)
					r.Patch("/", taskHandler.Update)
					r.Delete("/", taskHandler.Delete)
					r.Patch("/status", taskHandler.UpdateStatus)
					r.Get("/progress", taskHandler.GetProgress)
					r.Post("/start", taskHandler.StartTask)
					r.Post("/complete", taskHandler.CompleteTask)
					r.Post("/cancel", taskHandler.CancelTask)
					r.Get("/checklist", taskHandler.GetChecklist)
					r.Post("/checklist/regenerate", taskHandler.RegenerateChecklist)
					r.Post("/checklist/{itemId}/print", taskHandler.PrintFromChecklist)
				r.Patch("/checklist/{itemId}", taskHandler.ToggleChecklistItem)
				})
			})
		}

		// Supplies (standalone delete)
		supplyHandler := &ProjectSupplyHandler{service: services.ProjectSupplies}
		r.Delete("/supplies/{id}", supplyHandler.Delete)

		// Parts
		partHandler := &PartHandler{service: services.Parts, designService: services.Designs}
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
			r.Post("/open-external", designHandler.OpenExternal)
		})

		// Printers
		printerHandler := &PrinterHandler{service: services.Printers}
		dispatchHandler := NewDispatchHandler(services.Dispatcher)
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
				r.Get("/jobs", printerHandler.ListJobs)
				r.Get("/stats", printerHandler.GetJobStats)
				r.Get("/analytics", printerHandler.GetPrinterAnalytics)
				// Auto-dispatch settings
				r.Get("/dispatch-settings", dispatchHandler.GetPrinterSettings)
				r.Put("/dispatch-settings", dispatchHandler.UpdatePrinterSettings)
			})
		})

		// Materials
		materialHandler := &MaterialHandler{service: services.Materials}
		r.Route("/materials", func(r chi.Router) {
			r.Get("/", materialHandler.List)
			r.Post("/", materialHandler.Create)
			r.Get("/{id}", materialHandler.Get)
			r.Delete("/{id}", materialHandler.Delete)
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
				// Priority for auto-dispatch queue
				r.Patch("/priority", printJobHandler.UpdatePriority)
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
				r.Post("/retry", expenseHandler.Retry)
				r.Delete("/", expenseHandler.Delete)
			})
		})

		// Sales
		saleHandler := &SaleHandler{service: services.Sales}
		r.Route("/sales", func(r chi.Router) {
			r.Get("/", saleHandler.List)
			r.Post("/", saleHandler.Create)
			r.Get("/weekly-insights", saleHandler.GetWeeklyInsights)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", saleHandler.Get)
				r.Patch("/", saleHandler.Update)
				r.Delete("/", saleHandler.Delete)
			})
		})

		// Settings
		settingsHandler := &SettingsHandler{service: services.Settings}
		r.Route("/settings", func(r chi.Router) {
			r.Get("/", settingsHandler.List)
			r.Route("/{key}", func(r chi.Router) {
				r.Get("/", settingsHandler.Get)
				r.Put("/", settingsHandler.Set)
				r.Delete("/", settingsHandler.Delete)
			})
		})

		// Dispatch (auto-dispatch queue management)
		r.Route("/dispatch", func(r chi.Router) {
			r.Get("/requests", dispatchHandler.ListPending)
			r.Post("/requests/{id}/confirm", dispatchHandler.Confirm)
			r.Post("/requests/{id}/reject", dispatchHandler.Reject)
			r.Post("/requests/{id}/skip", dispatchHandler.Skip)
			r.Get("/settings", dispatchHandler.GetGlobalSettings)
			r.Put("/settings", dispatchHandler.UpdateGlobalSettings)
		})

		// Backups
		if services.Backup != nil {
			backupHandler := &BackupHandler{service: services.Backup}
			r.Route("/backups", func(r chi.Router) {
				r.Get("/", backupHandler.List)
				r.Post("/", backupHandler.Create)
				r.Get("/config", backupHandler.GetConfig)
				r.Put("/config", backupHandler.UpdateConfig)
				r.Delete("/{name}", backupHandler.Delete)
				r.Post("/{name}/restore", backupHandler.Restore)
			})
		}

		// Feedback
		feedbackHandler := &FeedbackHandler{service: services.Feedback}
		r.Route("/feedback", func(r chi.Router) {
			r.Post("/", feedbackHandler.Submit)
			r.Get("/", feedbackHandler.List)
			r.Delete("/{id}", feedbackHandler.Delete)
		})

		// Stats
		statsHandler := &StatsHandler{service: services.Stats}
		r.Get("/stats/financial", statsHandler.GetFinancialSummary)
		r.Get("/stats/time-series", statsHandler.GetTimeSeries)
		r.Get("/stats/expenses-by-category", statsHandler.GetExpensesByCategory)
		r.Get("/stats/sales-by-channel", statsHandler.GetSalesByChannel)
		r.Get("/stats/sales-by-project", statsHandler.GetSalesByProject)

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

				// Recipe supply endpoints
				r.Get("/supplies", templateHandler.ListSupplies)
				r.Post("/supplies", templateHandler.AddSupply)
				r.Patch("/supplies/{supplyId}", templateHandler.UpdateSupply)
				r.Delete("/supplies/{supplyId}", templateHandler.RemoveSupply)

				// Recipe compatibility endpoints
				r.Get("/compatible-printers", templateHandler.GetCompatiblePrinters)
				r.Get("/compatible-spools", templateHandler.GetCompatibleSpools)
				r.Get("/cost-estimate", templateHandler.GetCostEstimate)
				r.Post("/validate-printer/{printerId}", templateHandler.ValidatePrinter)

				// Analytics
				r.Get("/analytics", templateHandler.GetAnalytics)
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

		// Etsy Integration (always registered — can be configured at runtime)
		etsyHandler := NewEtsyHandler(services.Etsy, services.Orders)
		etsyHandler.SetTemplateSvc(services.Templates)
		r.Route("/integrations/etsy", func(r chi.Router) {
			// Configuration
			r.Put("/configure", etsyHandler.Configure)

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

		// Squarespace Integration
		if services.Squarespace != nil {
			squarespaceHandler := NewSquarespaceHandler(services.Squarespace, services.Orders)
			r.Route("/integrations/squarespace", func(r chi.Router) {
				// Connection
				r.Post("/connect", squarespaceHandler.Connect)
				r.Get("/status", squarespaceHandler.GetStatus)
				r.Post("/disconnect", squarespaceHandler.Disconnect)

				// Orders
				r.Post("/orders/sync", squarespaceHandler.SyncOrders)
				r.Get("/orders", squarespaceHandler.ListOrders)
				r.Get("/orders/{id}", squarespaceHandler.GetOrder)
				r.Post("/orders/{id}/process", squarespaceHandler.ProcessOrder)

				// Products
				r.Post("/products/sync", squarespaceHandler.SyncProducts)
				r.Get("/products", squarespaceHandler.ListProducts)
				r.Get("/products/{id}", squarespaceHandler.GetProduct)
				r.Post("/products/{id}/link", squarespaceHandler.LinkProduct)
				r.Delete("/products/{id}/link", squarespaceHandler.UnlinkProduct)
			})
		}

		// ============================================
		// New Feature Gap Endpoints
		// ============================================

		// Alerts
		if services.Alerts != nil {
			alertsHandler := NewAlertsHandler(services.Alerts)
			r.Route("/alerts", func(r chi.Router) {
				r.Get("/", alertsHandler.List)
				r.Get("/counts", alertsHandler.GetCounts)
				r.Post("/{type}/{entityId}/dismiss", alertsHandler.Dismiss)
				r.Delete("/{type}/{entityId}/dismiss", alertsHandler.Undismiss)
			})
			r.Patch("/materials/{materialId}/threshold", alertsHandler.UpdateMaterialThreshold)
		}

		// Orders (Unified)
		if services.Orders != nil {
			orderHandler := NewOrderHandler(services.Orders)
			r.Route("/orders", func(r chi.Router) {
				r.Get("/", orderHandler.List)
				r.Post("/", orderHandler.Create)
				r.Get("/counts", orderHandler.GetCounts)
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", orderHandler.Get)
					r.Patch("/", orderHandler.Update)
					r.Delete("/", orderHandler.Delete)
					r.Patch("/status", orderHandler.UpdateStatus)
					r.Get("/progress", orderHandler.GetProgress)
					r.Post("/ship", orderHandler.MarkShipped)
					// Order items
					r.Post("/items", orderHandler.AddItem)
					r.Delete("/items/{itemId}", orderHandler.RemoveItem)
					r.Post("/items/{itemId}/process", orderHandler.ProcessItem)
				})
			})
		}

		// Customers
		if services.Customers != nil {
			customerHandler := NewCustomerHandler(services.Customers)
			r.Route("/customers", func(r chi.Router) {
				r.Get("/", customerHandler.List)
				r.Post("/", customerHandler.Create)
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", customerHandler.Get)
					r.Patch("/", customerHandler.Update)
					r.Delete("/", customerHandler.Delete)
				})
			})
		}

		// Quotes
		if services.Quotes != nil {
			quoteHandler := NewQuoteHandler(services.Quotes)
			r.Route("/quotes", func(r chi.Router) {
				r.Get("/", quoteHandler.List)
				r.Post("/", quoteHandler.Create)
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", quoteHandler.Get)
					r.Patch("/", quoteHandler.Update)
					r.Delete("/", quoteHandler.Delete)
					r.Post("/send", quoteHandler.Send)
					r.Post("/accept", quoteHandler.Accept)
					r.Post("/reject", quoteHandler.Reject)
					// Options
					r.Post("/options", quoteHandler.CreateOption)
					r.Route("/options/{optionId}", func(r chi.Router) {
						r.Patch("/", quoteHandler.UpdateOption)
						r.Delete("/", quoteHandler.DeleteOption)
						// Line items
						r.Post("/items", quoteHandler.CreateLineItem)
						r.Route("/items/{itemId}", func(r chi.Router) {
							r.Patch("/", quoteHandler.UpdateLineItem)
							r.Delete("/", quoteHandler.DeleteLineItem)
						})
					})
				})
			})
		}

		// Tags
		if services.Tags != nil {
			tagsHandler := NewTagsHandler(services.Tags)
			r.Route("/tags", func(r chi.Router) {
				r.Get("/", tagsHandler.List)
				r.Post("/", tagsHandler.Create)
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", tagsHandler.Get)
					r.Patch("/", tagsHandler.Update)
					r.Delete("/", tagsHandler.Delete)
					r.Get("/parts", tagsHandler.ListPartsByTag)
					r.Get("/designs", tagsHandler.ListDesignsByTag)
				})
			})
			// Part tags
			r.Get("/parts/{id}/tags", tagsHandler.GetPartTags)
			r.Post("/parts/{id}/tags/{tagId}", tagsHandler.AddTagToPart)
			r.Delete("/parts/{id}/tags/{tagId}", tagsHandler.RemoveTagFromPart)
			// Design tags
			r.Get("/designs/{id}/tags", tagsHandler.GetDesignTags)
			r.Post("/designs/{id}/tags/{tagId}", tagsHandler.AddTagToDesign)
			r.Delete("/designs/{id}/tags/{tagId}", tagsHandler.RemoveTagFromDesign)
		}

		// Shopify Integration
		if services.Shopify != nil {
			shopifyHandler := NewShopifyHandler(services.Shopify, services.Orders, service.ShopifyConfig{})
			r.Route("/integrations/shopify", func(r chi.Router) {
				r.Get("/auth-url", shopifyHandler.GetAuthURL)
				r.Get("/callback", shopifyHandler.Callback)
				r.Get("/status", shopifyHandler.GetStatus)
				r.Delete("/", shopifyHandler.Disconnect)
				r.Post("/sync", shopifyHandler.SyncOrders)
				r.Get("/orders", shopifyHandler.ListOrders)
				r.Get("/orders/{id}", shopifyHandler.GetOrder)
				r.Post("/orders/{id}/process", shopifyHandler.ProcessOrder)
				r.Post("/products/{productId}/link", shopifyHandler.LinkProduct)
				r.Delete("/products/{productId}/link", shopifyHandler.UnlinkProduct)
			})
		}

		// Timeline (Gantt View)
		if services.Timeline != nil {
			timelineHandler := NewTimelineHandler(services.Timeline)
			r.Route("/timeline", func(r chi.Router) {
				r.Get("/", timelineHandler.GetTimeline)
				r.Get("/orders/{id}", timelineHandler.GetOrderTimeline)
				r.Get("/projects/{id}", timelineHandler.GetProjectTimeline)
			})
		}
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
