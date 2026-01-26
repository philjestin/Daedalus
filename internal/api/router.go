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
		r.Route("/designs/{id}", func(r chi.Router) {
			r.Get("/", designHandler.Get)
			r.Get("/download", designHandler.Download)
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
			})
		})

		// Files
		fileHandler := &FileHandler{service: services.Files}
		r.Get("/files/{id}", fileHandler.Get)
	})

	return r
}

