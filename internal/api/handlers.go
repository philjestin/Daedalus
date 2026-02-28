package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hyperion/printfarm/internal/model"
	"github.com/hyperion/printfarm/internal/printer"
	"github.com/hyperion/printfarm/internal/service"
	"github.com/hyperion/printfarm/internal/validation"
)

// respondJSON sends a JSON response.
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			slog.Error("failed to encode JSON response", "error", err)
		}
	}
	// Flush if the ResponseWriter supports it
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

// respondError sends an error response.
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// respondValidationError sends a validation error response.
func respondValidationError(w http.ResponseWriter, err error) {
	if ve, ok := err.(*validation.ValidationError); ok {
		respondJSON(w, http.StatusBadRequest, ve)
		return
	}
	respondError(w, http.StatusBadRequest, err.Error())
}

// parseUUID parses a UUID from URL parameter.
func parseUUID(r *http.Request, param string) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, param))
}

// ProjectHandler handles project endpoints.
type ProjectHandler struct {
	service *service.ProjectService
}

// List returns all projects.
func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	projects, err := h.service.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if projects == nil {
		projects = []model.Project{}
	}

	respondJSON(w, http.StatusOK, projects)
}

// Create creates a new project.
func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	var project model.Project
	if err := json.NewDecoder(r.Body).Decode(&project); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate input
	v := validation.New()
	v.Required("name", project.Name)
	v.MaxLength("name", project.Name, 255)
	v.MaxLength("description", project.Description, 5000)
	v.NoControlChars("name", project.Name)
	if err := v.Error(); err != nil {
		respondValidationError(w, err)
		return
	}

	if err := h.service.Create(r.Context(), &project); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, project)
}

// Get returns a project by ID.
func (h *ProjectHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	project, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if project == nil {
		respondError(w, http.StatusNotFound, "project not found")
		return
	}

	respondJSON(w, http.StatusOK, project)
}

// Update updates a project.
func (h *ProjectHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	project, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if project == nil {
		respondError(w, http.StatusNotFound, "project not found")
		return
	}

	if err := json.NewDecoder(r.Body).Decode(project); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	project.ID = id

	if err := h.service.Update(r.Context(), project); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, project)
}

// Delete removes a project.
func (h *ProjectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListJobs returns all print jobs for a project.
func (h *ProjectHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	jobs, err := h.service.ListJobs(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if jobs == nil {
		jobs = []model.PrintJob{}
	}

	respondJSON(w, http.StatusOK, jobs)
}

// GetJobStats returns job statistics for a project.
func (h *ProjectHandler) GetJobStats(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	stats, err := h.service.GetJobStats(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, stats)
}

// GetProjectSummary returns derived analytics for a project.
func (h *ProjectHandler) GetProjectSummary(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	summary, err := h.service.GetProjectSummary(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, summary)
}

// StartProduction auto-assigns resources and starts all queued jobs for a project.
func (h *ProjectHandler) StartProduction(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	result, err := h.service.StartProduction(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// PartHandler handles part endpoints.
type PartHandler struct {
	service       *service.PartService
	designService *service.DesignService
}

// ListByProject returns all parts for a project.
func (h *PartHandler) ListByProject(w http.ResponseWriter, r *http.Request) {
	projectID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	parts, err := h.service.ListByProject(r.Context(), projectID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if parts == nil {
		parts = []model.Part{}
	}

	respondJSON(w, http.StatusOK, parts)
}

// Create creates a new part. Supports JSON body or multipart/form-data with an optional file.
func (h *PartHandler) Create(w http.ResponseWriter, r *http.Request) {
	projectID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/form-data") {
		h.createWithFile(w, r, projectID)
		return
	}

	var part model.Part
	if err := json.NewDecoder(r.Body).Decode(&part); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	part.ProjectID = projectID

	if err := h.service.Create(r.Context(), &part); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, part)
}

// createWithFile handles multipart part creation with an optional file attachment.
func (h *PartHandler) createWithFile(w http.ResponseWriter, r *http.Request, projectID uuid.UUID) {
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		respondError(w, http.StatusBadRequest, "failed to parse form")
		return
	}

	quantity := 1
	if q := r.FormValue("quantity"); q != "" {
		if parsed, err := strconv.Atoi(q); err == nil {
			quantity = parsed
		}
	}

	part := model.Part{
		ProjectID:   projectID,
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
		Quantity:    quantity,
	}

	if err := h.service.Create(r.Context(), &part); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Check for optional file
	file, header, err := r.FormFile("file")
	if err != nil {
		// No file provided — return just the part
		respondJSON(w, http.StatusCreated, part)
		return
	}
	defer file.Close()

	if h.designService == nil {
		respondJSON(w, http.StatusCreated, part)
		return
	}

	notes := r.FormValue("notes")
	design, err := h.designService.Create(r.Context(), part.ID, header.Filename, file, notes)
	if err != nil {
		// Part was created but design failed — still return the part
		slog.Error("failed to create design for new part", "error", err, "part_id", part.ID)
		respondJSON(w, http.StatusCreated, map[string]interface{}{
			"part":         part,
			"design_error": err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"part":   part,
		"design": design,
	})
}

// Get returns a part by ID.
func (h *PartHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid part ID")
		return
	}

	part, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if part == nil {
		respondError(w, http.StatusNotFound, "part not found")
		return
	}

	respondJSON(w, http.StatusOK, part)
}

// Update updates a part.
func (h *PartHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid part ID")
		return
	}

	part, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if part == nil {
		respondError(w, http.StatusNotFound, "part not found")
		return
	}

	if err := json.NewDecoder(r.Body).Decode(part); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	part.ID = id

	if err := h.service.Update(r.Context(), part); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, part)
}

// Delete removes a part.
func (h *PartHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid part ID")
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DesignHandler handles design endpoints.
type DesignHandler struct {
	service *service.DesignService
}

// ListByPart returns all designs for a part.
func (h *DesignHandler) ListByPart(w http.ResponseWriter, r *http.Request) {
	partID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid part ID")
		return
	}

	designs, err := h.service.ListByPart(r.Context(), partID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if designs == nil {
		designs = []model.Design{}
	}

	respondJSON(w, http.StatusOK, designs)
}

// Create uploads a new design version.
func (h *DesignHandler) Create(w http.ResponseWriter, r *http.Request) {
	partID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid part ID")
		return
	}

	// Parse multipart form (max 100MB)
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		respondError(w, http.StatusBadRequest, "failed to parse form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	notes := r.FormValue("notes")

	design, err := h.service.Create(r.Context(), partID, header.Filename, file, notes)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, design)
}

// Get returns a design by ID.
func (h *DesignHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid design ID")
		return
	}

	design, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if design == nil {
		respondError(w, http.StatusNotFound, "design not found")
		return
	}

	respondJSON(w, http.StatusOK, design)
}

// Download returns the design file.
func (h *DesignHandler) Download(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid design ID")
		return
	}

	reader, design, err := h.service.GetFile(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Disposition", "attachment; filename="+design.FileName)
	w.Header().Set("Content-Type", "application/octet-stream")
	io.Copy(w, reader)
}

// OpenExternal opens a design file in an external application.
func (h *DesignHandler) OpenExternal(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid design ID")
		return
	}

	var req struct {
		App string `json:"app"`
	}
	if r.Body != nil && r.ContentLength > 0 {
		json.NewDecoder(r.Body).Decode(&req)
	}

	if err := h.service.OpenInExternalApp(r.Context(), id, req.App); err != nil {
		if err.Error() == "design not found" || err.Error() == "file not found" {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// PrinterHandler handles printer endpoints.
type PrinterHandler struct {
	service *service.PrinterService
}

// List returns all printers.
func (h *PrinterHandler) List(w http.ResponseWriter, r *http.Request) {
	printers, err := h.service.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if printers == nil {
		printers = []model.Printer{}
	}

	respondJSON(w, http.StatusOK, printers)
}

// Create creates a new printer.
func (h *PrinterHandler) Create(w http.ResponseWriter, r *http.Request) {
	var printer model.Printer
	if err := json.NewDecoder(r.Body).Decode(&printer); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate input
	v := validation.New()
	v.Required("name", printer.Name)
	v.MaxLength("name", printer.Name, 255)
	v.MaxLength("model", printer.Model, 255)
	v.MaxLength("manufacturer", printer.Manufacturer, 255)
	v.NoControlChars("name", printer.Name)
	v.NonNegative("cost_per_hour_cents", printer.CostPerHourCents)
	if err := v.Error(); err != nil {
		respondValidationError(w, err)
		return
	}

	if err := h.service.Create(r.Context(), &printer); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, printer)
}

// Get returns a printer by ID.
func (h *PrinterHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid printer ID")
		return
	}

	printer, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if printer == nil {
		respondError(w, http.StatusNotFound, "printer not found")
		return
	}

	respondJSON(w, http.StatusOK, printer)
}

// Update updates a printer.
func (h *PrinterHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid printer ID")
		return
	}

	printer, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if printer == nil {
		respondError(w, http.StatusNotFound, "printer not found")
		return
	}

	if err := json.NewDecoder(r.Body).Decode(printer); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	printer.ID = id

	if err := h.service.Update(r.Context(), printer); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, printer)
}

// Delete removes a printer.
func (h *PrinterHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid printer ID")
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetState returns the real-time state of a printer.
func (h *PrinterHandler) GetState(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid printer ID")
		return
	}

	state, err := h.service.GetState(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, state)
}

// GetAllStates returns the real-time state of all printers.
func (h *PrinterHandler) GetAllStates(w http.ResponseWriter, r *http.Request) {
	states := h.service.GetAllStates(r.Context())
	respondJSON(w, http.StatusOK, states)
}

// Discover scans the network for printers.
func (h *PrinterHandler) Discover(w http.ResponseWriter, r *http.Request) {
	slog.Info("starting printer discovery")
	
	ctx := context.Background()
	printers, err := h.service.DiscoverPrinters(ctx)
	if err != nil {
		slog.Error("discovery failed", "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	slog.Info("discovery complete", "found", len(printers))
	
	if printers == nil {
		printers = []printer.DiscoveredPrinter{}
	}
	
	respondJSON(w, http.StatusOK, printers)
}

// ListJobs returns all print jobs for a printer.
func (h *PrinterHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid printer ID")
		return
	}

	jobs, err := h.service.ListJobs(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if jobs == nil {
		jobs = []model.PrintJob{}
	}

	respondJSON(w, http.StatusOK, jobs)
}

// GetJobStats returns job statistics for a printer.
func (h *PrinterHandler) GetJobStats(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid printer ID")
		return
	}

	stats, err := h.service.GetJobStats(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, stats)
}

// GetPrinterAnalytics returns comprehensive analytics for a printer.
func (h *PrinterHandler) GetPrinterAnalytics(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid printer ID")
		return
	}

	analytics, err := h.service.GetPrinterAnalytics(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, analytics)
}

// MaterialHandler handles material endpoints.
type MaterialHandler struct {
	service *service.MaterialService
}

// List returns all materials.
func (h *MaterialHandler) List(w http.ResponseWriter, r *http.Request) {
	var materials []model.Material
	var err error

	if typeFilter := r.URL.Query().Get("type"); typeFilter != "" {
		materials, err = h.service.ListByType(r.Context(), model.MaterialType(typeFilter))
	} else {
		materials, err = h.service.List(r.Context())
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if materials == nil {
		materials = []model.Material{}
	}

	respondJSON(w, http.StatusOK, materials)
}

// Create creates a new material.
func (h *MaterialHandler) Create(w http.ResponseWriter, r *http.Request) {
	var material model.Material
	if err := json.NewDecoder(r.Body).Decode(&material); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate input
	v := validation.New()
	v.Required("name", material.Name)
	v.MaxLength("name", material.Name, 255)
	v.Required("type", string(material.Type))
	v.MaxLength("type", string(material.Type), 50)
	v.MaxLength("manufacturer", material.Manufacturer, 255)
	v.MaxLength("color", material.Color, 100)
	v.NonNegativeFloat("density", material.Density)
	v.NonNegativeFloat("cost_per_kg", material.CostPerKg)
	if err := v.Error(); err != nil {
		respondValidationError(w, err)
		return
	}

	if err := h.service.Create(r.Context(), &material); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, material)
}

// Get returns a material by ID.
func (h *MaterialHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid material ID")
		return
	}

	material, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if material == nil {
		respondError(w, http.StatusNotFound, "material not found")
		return
	}

	respondJSON(w, http.StatusOK, material)
}

// Delete removes a material by ID.
func (h *MaterialHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid material ID")
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SpoolHandler handles spool endpoints.
type SpoolHandler struct {
	service *service.SpoolService
}

// List returns all spools.
func (h *SpoolHandler) List(w http.ResponseWriter, r *http.Request) {
	spools, err := h.service.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if spools == nil {
		spools = []model.MaterialSpool{}
	}

	respondJSON(w, http.StatusOK, spools)
}

// Create creates a new spool.
func (h *SpoolHandler) Create(w http.ResponseWriter, r *http.Request) {
	var spool model.MaterialSpool
	if err := json.NewDecoder(r.Body).Decode(&spool); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.Create(r.Context(), &spool); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, spool)
}

// Get returns a spool by ID.
func (h *SpoolHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid spool ID")
		return
	}

	spool, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if spool == nil {
		respondError(w, http.StatusNotFound, "spool not found")
		return
	}

	respondJSON(w, http.StatusOK, spool)
}

// PrintJobHandler handles print job endpoints.
type PrintJobHandler struct {
	service *service.PrintJobService
}

// List returns all print jobs.
func (h *PrintJobHandler) List(w http.ResponseWriter, r *http.Request) {
	var printerID *uuid.UUID
	if pidStr := r.URL.Query().Get("printer_id"); pidStr != "" {
		if pid, err := uuid.Parse(pidStr); err == nil {
			printerID = &pid
		}
	}

	var status *model.PrintJobStatus
	if s := r.URL.Query().Get("status"); s != "" {
		ps := model.PrintJobStatus(s)
		status = &ps
	}

	jobs, err := h.service.List(r.Context(), printerID, status)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if jobs == nil {
		jobs = []model.PrintJob{}
	}

	respondJSON(w, http.StatusOK, jobs)
}

// ListByDesign returns all print jobs for a design.
func (h *PrintJobHandler) ListByDesign(w http.ResponseWriter, r *http.Request) {
	designID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid design ID")
		return
	}

	jobs, err := h.service.ListByDesign(r.Context(), designID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if jobs == nil {
		jobs = []model.PrintJob{}
	}

	respondJSON(w, http.StatusOK, jobs)
}

// Create creates a new print job.
func (h *PrintJobHandler) Create(w http.ResponseWriter, r *http.Request) {
	var job model.PrintJob
	if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.Create(r.Context(), &job); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, job)
}

// Get returns a print job by ID.
func (h *PrintJobHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid job ID")
		return
	}

	job, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if job == nil {
		respondError(w, http.StatusNotFound, "job not found")
		return
	}

	respondJSON(w, http.StatusOK, job)
}

// Update updates a print job.
func (h *PrintJobHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid job ID")
		return
	}

	job, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if job == nil {
		respondError(w, http.StatusNotFound, "job not found")
		return
	}

	if err := json.NewDecoder(r.Body).Decode(job); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	job.ID = id

	if err := h.service.Update(r.Context(), job); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, job)
}

// PreflightCheck validates a job is ready to start.
func (h *PrintJobHandler) PreflightCheck(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid job ID")
		return
	}

	result, err := h.service.PreflightCheck(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// Start sends the job to the printer.
func (h *PrintJobHandler) Start(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid job ID")
		return
	}

	if err := h.service.Start(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Pause pauses the print job.
func (h *PrintJobHandler) Pause(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid job ID")
		return
	}

	if err := h.service.Pause(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Resume resumes the print job.
func (h *PrintJobHandler) Resume(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid job ID")
		return
	}

	if err := h.service.Resume(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Cancel cancels the print job.
func (h *PrintJobHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid job ID")
		return
	}

	if err := h.service.Cancel(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
}

// RecordOutcome records the outcome of a completed print job.
func (h *PrintJobHandler) RecordOutcome(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid job ID")
		return
	}

	var outcome model.PrintOutcome
	if err := json.NewDecoder(r.Body).Decode(&outcome); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.RecordOutcome(r.Context(), id, &outcome); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Get updated job to return
	job, _ := h.service.GetByID(r.Context(), id)
	respondJSON(w, http.StatusOK, job)
}

// GetWithEvents returns a print job with its full event timeline.
func (h *PrintJobHandler) GetWithEvents(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid job ID")
		return
	}

	job, err := h.service.GetByIDWithEvents(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if job == nil {
		respondError(w, http.StatusNotFound, "job not found")
		return
	}

	respondJSON(w, http.StatusOK, job)
}

// GetEvents returns all events for a print job.
func (h *PrintJobHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid job ID")
		return
	}

	events, err := h.service.GetEvents(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if events == nil {
		events = []model.JobEvent{}
	}

	respondJSON(w, http.StatusOK, events)
}

// GetRetryChain returns all jobs in a retry chain.
func (h *PrintJobHandler) GetRetryChain(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid job ID")
		return
	}

	chain, err := h.service.GetRetryChain(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if chain == nil {
		chain = []model.PrintJob{}
	}

	respondJSON(w, http.StatusOK, chain)
}

// RetryRequest represents the request body for retrying a job.
type RetryRequest struct {
	PrinterID       string `json:"printer_id,omitempty"`
	MaterialSpoolID string `json:"material_spool_id,omitempty"`
	FailureCategory string `json:"failure_category,omitempty"`
	Notes           string `json:"notes,omitempty"`
}

// Retry creates a new job from a failed job.
func (h *PrintJobHandler) Retry(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid job ID")
		return
	}

	var req RetryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Body is optional
		req = RetryRequest{}
	}

	retryReq := &service.RetryRequest{
		Notes: req.Notes,
	}

	if req.PrinterID != "" {
		printerID, err := uuid.Parse(req.PrinterID)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid printer ID")
			return
		}
		retryReq.PrinterID = &printerID
	}

	if req.MaterialSpoolID != "" {
		spoolID, err := uuid.Parse(req.MaterialSpoolID)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid material spool ID")
			return
		}
		retryReq.MaterialSpoolID = &spoolID
	}

	if req.FailureCategory != "" {
		category := model.FailureCategory(req.FailureCategory)
		retryReq.FailureCategory = &category
	}

	newJob, err := h.service.Retry(r.Context(), id, retryReq)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, newJob)
}

// RecordFailureRequest represents the request body for recording a failure.
type RecordFailureRequest struct {
	FailureCategory string `json:"failure_category"`
	ErrorCode       string `json:"error_code"`
	ErrorMessage    string `json:"error_message"`
}

// RecordFailure records a failure for a job.
func (h *PrintJobHandler) RecordFailure(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid job ID")
		return
	}

	var req RecordFailureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	category := model.FailureCategory(req.FailureCategory)
	if category == "" {
		category = model.FailureUnknown
	}

	if err := h.service.RecordFailure(r.Context(), id, category, req.ErrorCode, req.ErrorMessage); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Return updated job
	job, _ := h.service.GetByID(r.Context(), id)
	respondJSON(w, http.StatusOK, job)
}

// MarkAsScrap marks a failed job as scrap (no retry intended).
func (h *PrintJobHandler) MarkAsScrap(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid job ID")
		return
	}

	var req service.ScrapRequest
	if r.Body != nil && r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}

	if err := h.service.MarkAsScrap(r.Context(), id, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Return updated job
	job, _ := h.service.GetByID(r.Context(), id)
	respondJSON(w, http.StatusOK, job)
}

// ListByRecipe returns all print jobs for a recipe.
func (h *PrintJobHandler) ListByRecipe(w http.ResponseWriter, r *http.Request) {
	recipeID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid recipe ID")
		return
	}

	jobs, err := h.service.ListByRecipe(r.Context(), recipeID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if jobs == nil {
		jobs = []model.PrintJob{}
	}

	respondJSON(w, http.StatusOK, jobs)
}

// UpdatePriority updates a job's priority in the queue.
func (h *PrintJobHandler) UpdatePriority(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid job ID")
		return
	}

	var req struct {
		Priority int `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.UpdatePriority(r.Context(), id, req.Priority); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// FileHandler handles file endpoints.
type FileHandler struct {
	service *service.FileService
}

// Get returns a file by ID.
func (h *FileHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid file ID")
		return
	}

	reader, file, err := h.service.GetReader(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Disposition", "attachment; filename="+file.OriginalName)
	w.Header().Set("Content-Type", file.ContentType)
	io.Copy(w, reader)
}

// ExpenseHandler handles expense endpoints.
type ExpenseHandler struct {
	service *service.ExpenseService
}

// List returns all expenses.
func (h *ExpenseHandler) List(w http.ResponseWriter, r *http.Request) {
	var status *model.ExpenseStatus
	if s := r.URL.Query().Get("status"); s != "" {
		es := model.ExpenseStatus(s)
		status = &es
	}

	expenses, err := h.service.List(r.Context(), status)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if expenses == nil {
		expenses = []model.Expense{}
	}

	respondJSON(w, http.StatusOK, expenses)
}

// Get returns an expense by ID with its items.
func (h *ExpenseHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid expense ID")
		return
	}

	expense, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if expense == nil {
		respondError(w, http.StatusNotFound, "expense not found")
		return
	}

	respondJSON(w, http.StatusOK, expense)
}

// UploadReceipt handles receipt file upload.
func (h *ExpenseHandler) UploadReceipt(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (max 32MB)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		respondError(w, http.StatusBadRequest, "failed to parse form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	// Read file data
	data, err := io.ReadAll(file)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to read file")
		return
	}

	expense, err := h.service.UploadReceipt(r.Context(), header.Filename, data)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, expense)
}

// Confirm confirms an expense and applies inventory changes.
func (h *ExpenseHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid expense ID")
		return
	}

	var req service.ConfirmExpenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.ConfirmExpense(r.Context(), id, &req); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Return updated expense
	expense, _ := h.service.GetByID(r.Context(), id)
	respondJSON(w, http.StatusOK, expense)
}

// Retry re-triggers AI parsing for a failed or stuck expense.
func (h *ExpenseHandler) Retry(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid expense ID")
		return
	}

	expense, err := h.service.RetryParse(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondError(w, http.StatusNotFound, err.Error())
		} else {
			respondError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	respondJSON(w, http.StatusOK, expense)
}

// Delete deletes an expense.
func (h *ExpenseHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid expense ID")
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SaleHandler handles sale endpoints.
type SaleHandler struct {
	service *service.SaleService
}

// List returns all sales.
func (h *SaleHandler) List(w http.ResponseWriter, r *http.Request) {
	var projectID *uuid.UUID
	if pidStr := r.URL.Query().Get("project_id"); pidStr != "" {
		if pid, err := uuid.Parse(pidStr); err == nil {
			projectID = &pid
		}
	}

	sales, err := h.service.List(r.Context(), projectID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if sales == nil {
		sales = []model.Sale{}
	}

	respondJSON(w, http.StatusOK, sales)
}

// Create creates a new sale.
func (h *SaleHandler) Create(w http.ResponseWriter, r *http.Request) {
	var sale model.Sale
	if err := json.NewDecoder(r.Body).Decode(&sale); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate input
	v := validation.New()
	v.MaxLength("channel", string(sale.Channel), 100)
	v.MaxLength("platform", sale.Platform, 100)
	v.MaxLength("customer_name", sale.CustomerName, 255)
	v.MaxLength("order_reference", sale.OrderReference, 255)
	v.MaxLength("item_description", sale.ItemDescription, 1000)
	v.NonNegative("gross_cents", sale.GrossCents)
	v.NonNegative("fees_cents", sale.FeesCents)
	if err := v.Error(); err != nil {
		respondValidationError(w, err)
		return
	}

	if err := h.service.Create(r.Context(), &sale); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, sale)
}

// Get returns a sale by ID.
func (h *SaleHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid sale ID")
		return
	}

	sale, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if sale == nil {
		respondError(w, http.StatusNotFound, "sale not found")
		return
	}

	respondJSON(w, http.StatusOK, sale)
}

// Update updates a sale.
func (h *SaleHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid sale ID")
		return
	}

	sale, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if sale == nil {
		respondError(w, http.StatusNotFound, "sale not found")
		return
	}

	if err := json.NewDecoder(r.Body).Decode(sale); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	sale.ID = id

	if err := h.service.Update(r.Context(), sale); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, sale)
}

// Delete deletes a sale.
func (h *SaleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid sale ID")
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetWeeklyInsights returns this-week vs last-week sales comparison.
func (h *SaleHandler) GetWeeklyInsights(w http.ResponseWriter, r *http.Request) {
	insights, err := h.service.GetWeeklyInsights(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, insights)
}

// parsePeriodTime converts a period string to a start time.
func parsePeriodTime(period string) time.Time {
	now := time.Now()
	switch period {
	case "60d":
		return now.AddDate(0, 0, -60)
	case "90d":
		return now.AddDate(0, 0, -90)
	case "12m":
		return now.AddDate(-1, 0, 0)
	default: // "30d"
		return now.AddDate(0, 0, -30)
	}
}

// StatsHandler handles statistics endpoints.
type StatsHandler struct {
	service *service.StatsService
}

// GetFinancialSummary returns aggregated financial statistics.
// Accepts an optional "period" query param (30d, 60d, 90d, 12m) to filter by time range.
func (h *StatsHandler) GetFinancialSummary(w http.ResponseWriter, r *http.Request) {
	var since *time.Time
	if period := r.URL.Query().Get("period"); period != "" {
		t := parsePeriodTime(period)
		since = &t
	}

	summary, err := h.service.GetFinancialSummary(r.Context(), since)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, summary)
}

// GetTimeSeries returns time-series data for revenue, expenses, and profit.
func (h *StatsHandler) GetTimeSeries(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "30d"
	}

	data, err := h.service.GetTimeSeriesData(r.Context(), period)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, data)
}

// GetExpensesByCategory returns expense totals grouped by category.
func (h *StatsHandler) GetExpensesByCategory(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "30d"
	}

	data, err := h.service.GetExpensesByCategory(r.Context(), period)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if data == nil {
		data = []service.CategoryBreakdown{}
	}

	respondJSON(w, http.StatusOK, data)
}

// GetSalesByChannel returns sales totals grouped by channel.
func (h *StatsHandler) GetSalesByChannel(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "30d"
	}

	data, err := h.service.GetSalesByChannel(r.Context(), period)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if data == nil {
		data = []service.ChannelBreakdown{}
	}

	respondJSON(w, http.StatusOK, data)
}

// GetSalesByProject returns sales aggregated by project.
func (h *StatsHandler) GetSalesByProject(w http.ResponseWriter, r *http.Request) {
	data, err := h.service.GetSalesByProject(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if data == nil {
		data = []service.ProjectSales{}
	}

	respondJSON(w, http.StatusOK, data)
}

// TemplateHandler handles template endpoints.
type TemplateHandler struct {
	service *service.TemplateService
}

// List returns all templates.
func (h *TemplateHandler) List(w http.ResponseWriter, r *http.Request) {
	activeOnly := r.URL.Query().Get("active") == "true"

	templates, err := h.service.List(r.Context(), activeOnly)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if templates == nil {
		templates = []model.Template{}
	}

	respondJSON(w, http.StatusOK, templates)
}

// Create creates a new template.
func (h *TemplateHandler) Create(w http.ResponseWriter, r *http.Request) {
	var template model.Template
	if err := json.NewDecoder(r.Body).Decode(&template); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate input
	v := validation.New()
	v.Required("name", template.Name)
	v.MaxLength("name", template.Name, 255)
	v.MaxLength("description", template.Description, 5000)
	v.NoControlChars("name", template.Name)
	v.NonNegative("estimated_print_seconds", template.EstimatedPrintSeconds)
	v.NonNegativeFloat("estimated_material_grams", template.EstimatedMaterialGrams)
	if err := v.Error(); err != nil {
		respondValidationError(w, err)
		return
	}

	if err := h.service.Create(r.Context(), &template); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, template)
}

// Get returns a template by ID with its designs.
func (h *TemplateHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid template ID")
		return
	}

	template, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if template == nil {
		respondError(w, http.StatusNotFound, "template not found")
		return
	}

	respondJSON(w, http.StatusOK, template)
}

// Update updates a template.
func (h *TemplateHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid template ID")
		return
	}

	template, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if template == nil {
		respondError(w, http.StatusNotFound, "template not found")
		return
	}

	if err := json.NewDecoder(r.Body).Decode(template); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	template.ID = id

	if err := h.service.Update(r.Context(), template); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, template)
}

// Delete removes a template.
func (h *TemplateHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid template ID")
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AddDesignRequest represents the request body for adding a design to a template.
type AddDesignRequest struct {
	DesignID  string `json:"design_id"`
	Quantity  int    `json:"quantity"`
	IsPrimary bool   `json:"is_primary"`
	Notes     string `json:"notes"`
}

// AddDesign adds a design to a template.
func (h *TemplateHandler) AddDesign(w http.ResponseWriter, r *http.Request) {
	templateID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid template ID")
		return
	}

	var req AddDesignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	designID, err := uuid.Parse(req.DesignID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid design ID")
		return
	}

	td := &model.TemplateDesign{
		TemplateID: templateID,
		DesignID:   designID,
		Quantity:   req.Quantity,
		IsPrimary:  req.IsPrimary,
		Notes:      req.Notes,
	}

	if err := h.service.AddDesign(r.Context(), td); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, td)
}

// RemoveDesign removes a design from a template.
func (h *TemplateHandler) RemoveDesign(w http.ResponseWriter, r *http.Request) {
	templateID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid template ID")
		return
	}

	designID, err := parseUUID(r, "designId")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid design ID")
		return
	}

	if err := h.service.RemoveDesign(r.Context(), templateID, designID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// InstantiateRequest represents the request body for instantiating a template.
type InstantiateRequest struct {
	OrderQuantity   int    `json:"order_quantity"`
	CustomerNotes   string `json:"customer_notes"`
	ExternalOrderID string `json:"external_order_id"`
	Source          string `json:"source"`
	MaterialSpoolID string `json:"material_spool_id"`
}

// Instantiate creates a project from a template.
func (h *TemplateHandler) Instantiate(w http.ResponseWriter, r *http.Request) {
	templateID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid template ID")
		return
	}

	var req InstantiateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	opts := service.CreateFromTemplateOptions{
		OrderQuantity:   req.OrderQuantity,
		CustomerNotes:   req.CustomerNotes,
		ExternalOrderID: req.ExternalOrderID,
		Source:          req.Source,
	}

	if req.MaterialSpoolID != "" {
		spoolID, err := uuid.Parse(req.MaterialSpoolID)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid material spool ID")
			return
		}
		opts.MaterialSpoolID = &spoolID
	}

	project, jobs, err := h.service.CreateProjectFromTemplate(r.Context(), templateID, opts)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"project": project,
		"jobs":    jobs,
	})
}

// ListMaterials returns all materials for a template/recipe.
func (h *TemplateHandler) ListMaterials(w http.ResponseWriter, r *http.Request) {
	templateID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid template ID")
		return
	}

	materials, err := h.service.ListMaterials(r.Context(), templateID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if materials == nil {
		materials = []model.RecipeMaterial{}
	}

	respondJSON(w, http.StatusOK, materials)
}

// AddMaterialRequest represents the request body for adding a material.
type AddMaterialRequest struct {
	MaterialType  string           `json:"material_type"`
	ColorSpec     *model.ColorSpec `json:"color_spec,omitempty"`
	WeightGrams   float64          `json:"weight_grams"`
	AMSPosition   *int             `json:"ams_position,omitempty"`
	SequenceOrder int              `json:"sequence_order"`
	Notes         string           `json:"notes"`
}

// AddMaterial adds a material to a template/recipe.
func (h *TemplateHandler) AddMaterial(w http.ResponseWriter, r *http.Request) {
	templateID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid template ID")
		return
	}

	var req AddMaterialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	material := &model.RecipeMaterial{
		RecipeID:      templateID,
		MaterialType:  model.MaterialType(req.MaterialType),
		ColorSpec:     req.ColorSpec,
		WeightGrams:   req.WeightGrams,
		AMSPosition:   req.AMSPosition,
		SequenceOrder: req.SequenceOrder,
		Notes:         req.Notes,
	}

	if err := h.service.AddMaterial(r.Context(), material); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, material)
}

// UpdateMaterial updates a material in a template/recipe.
func (h *TemplateHandler) UpdateMaterial(w http.ResponseWriter, r *http.Request) {
	materialID, err := parseUUID(r, "materialId")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid material ID")
		return
	}

	material, err := h.service.GetMaterial(r.Context(), materialID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if material == nil {
		respondError(w, http.StatusNotFound, "material not found")
		return
	}

	if err := json.NewDecoder(r.Body).Decode(material); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	material.ID = materialID

	if err := h.service.UpdateMaterial(r.Context(), material); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, material)
}

// RemoveMaterial removes a material from a template/recipe.
func (h *TemplateHandler) RemoveMaterial(w http.ResponseWriter, r *http.Request) {
	materialID, err := parseUUID(r, "materialId")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid material ID")
		return
	}

	if err := h.service.RemoveMaterial(r.Context(), materialID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetCompatiblePrinters returns printers that match recipe constraints.
func (h *TemplateHandler) GetCompatiblePrinters(w http.ResponseWriter, r *http.Request) {
	templateID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid template ID")
		return
	}

	printers, err := h.service.FindCompatiblePrinters(r.Context(), templateID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if printers == nil {
		printers = []model.Printer{}
	}

	respondJSON(w, http.StatusOK, printers)
}

// GetCompatibleSpools returns spools that match recipe material requirements.
func (h *TemplateHandler) GetCompatibleSpools(w http.ResponseWriter, r *http.Request) {
	templateID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid template ID")
		return
	}

	spools, err := h.service.FindCompatibleSpools(r.Context(), templateID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if spools == nil {
		spools = []service.CompatibleSpool{}
	}

	respondJSON(w, http.StatusOK, spools)
}

// GetCostEstimate returns the cost breakdown for a recipe.
func (h *TemplateHandler) GetCostEstimate(w http.ResponseWriter, r *http.Request) {
	templateID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid template ID")
		return
	}

	estimate, err := h.service.CalculateRecipeCost(r.Context(), templateID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, estimate)
}

// ValidatePrinter checks if a printer meets recipe constraints.
func (h *TemplateHandler) ValidatePrinter(w http.ResponseWriter, r *http.Request) {
	templateID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid template ID")
		return
	}

	printerID, err := parseUUID(r, "printerId")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid printer ID")
		return
	}

	result, err := h.service.ValidatePrinterForRecipe(r.Context(), templateID, printerID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// ListSupplies returns all supply items for a recipe.
func (h *TemplateHandler) ListSupplies(w http.ResponseWriter, r *http.Request) {
	templateID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid template ID")
		return
	}

	supplies, err := h.service.ListSupplies(r.Context(), templateID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if supplies == nil {
		supplies = []model.RecipeSupply{}
	}

	respondJSON(w, http.StatusOK, supplies)
}

// AddSupply adds a supply item to a recipe.
func (h *TemplateHandler) AddSupply(w http.ResponseWriter, r *http.Request) {
	templateID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid template ID")
		return
	}

	var supply model.RecipeSupply
	if err := json.NewDecoder(r.Body).Decode(&supply); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	supply.RecipeID = templateID

	if err := h.service.AddSupply(r.Context(), &supply); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, supply)
}

// UpdateSupply updates a supply item in a recipe.
func (h *TemplateHandler) UpdateSupply(w http.ResponseWriter, r *http.Request) {
	supplyID, err := parseUUID(r, "supplyId")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid supply ID")
		return
	}

	supply, err := h.service.GetSupply(r.Context(), supplyID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if supply == nil {
		respondError(w, http.StatusNotFound, "supply not found")
		return
	}

	if err := json.NewDecoder(r.Body).Decode(supply); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	supply.ID = supplyID

	if err := h.service.UpdateSupply(r.Context(), supply); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, supply)
}

// RemoveSupply removes a supply item from a recipe.
func (h *TemplateHandler) RemoveSupply(w http.ResponseWriter, r *http.Request) {
	supplyID, err := parseUUID(r, "supplyId")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid supply ID")
		return
	}

	if err := h.service.RemoveSupply(r.Context(), supplyID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetAnalytics returns aggregated performance analytics for a template.
func (h *TemplateHandler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	templateID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid template ID")
		return
	}

	analytics, err := h.service.GetTemplateAnalytics(r.Context(), templateID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, analytics)
}

// SettingsHandler handles settings endpoints.
type SettingsHandler struct {
	service *service.SettingsService
}

// List returns all settings (with sensitive values masked).
func (h *SettingsHandler) List(w http.ResponseWriter, r *http.Request) {
	settings, err := h.service.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Mask sensitive values
	type maskedSetting struct {
		Key       string `json:"key"`
		Value     string `json:"value"`
		UpdatedAt string `json:"updated_at"`
	}
	result := make([]maskedSetting, 0, len(settings))
	for _, s := range settings {
		val := s.Value
		if isSensitiveKey(s.Key) && len(val) > 8 {
			val = val[:4] + "..." + val[len(val)-4:]
		}
		result = append(result, maskedSetting{
			Key:       s.Key,
			Value:     val,
			UpdatedAt: s.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	respondJSON(w, http.StatusOK, result)
}

// Get returns a single setting (with sensitive values masked).
func (h *SettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if key == "" {
		respondError(w, http.StatusBadRequest, "key is required")
		return
	}

	setting, err := h.service.Get(r.Context(), key)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if setting == nil {
		respondError(w, http.StatusNotFound, "setting not found")
		return
	}

	val := setting.Value
	if isSensitiveKey(key) && len(val) > 8 {
		val = val[:4] + "..." + val[len(val)-4:]
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"key":        setting.Key,
		"value":      val,
		"updated_at": setting.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// Set creates or updates a setting.
func (h *SettingsHandler) Set(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if key == "" {
		respondError(w, http.StatusBadRequest, "key is required")
		return
	}

	var req struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.Set(r.Context(), key, req.Value); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Delete removes a setting.
func (h *SettingsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if key == "" {
		respondError(w, http.StatusBadRequest, "key is required")
		return
	}

	if err := h.service.Delete(r.Context(), key); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// isSensitiveKey returns true for keys that should have their values masked in GET responses.
func isSensitiveKey(key string) bool {
	return strings.Contains(key, "api_key") || strings.Contains(key, "secret") || strings.Contains(key, "token") || strings.Contains(key, "password")
}

// BambuCloudHandler handles Bambu Cloud integration endpoints.
type BambuCloudHandler struct {
	service *service.BambuCloudService
}

// Login authenticates with Bambu Cloud.
func (h *BambuCloudHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	needsCode, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		respondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	status := "ok"
	if needsCode {
		status = "verify_code_required"
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": status})
}

// Verify completes login with a verification code.
func (h *BambuCloudHandler) Verify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" || req.Code == "" {
		respondError(w, http.StatusBadRequest, "email and code are required")
		return
	}

	if err := h.service.VerifyCode(r.Context(), req.Email, req.Code); err != nil {
		respondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Status returns the current Bambu Cloud auth status.
func (h *BambuCloudHandler) Status(w http.ResponseWriter, r *http.Request) {
	auth, err := h.service.GetStoredAuth(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if auth == nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"connected": false,
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"connected": true,
		"email":     auth.Email,
	})
}

// Devices fetches the list of printers from Bambu Cloud.
func (h *BambuCloudHandler) Devices(w http.ResponseWriter, r *http.Request) {
	devices, err := h.service.GetDevices(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, devices)
}

// AddDevice creates a printer from a cloud device.
func (h *BambuCloudHandler) AddDevice(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DevID string `json:"dev_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.DevID == "" {
		respondError(w, http.StatusBadRequest, "dev_id is required")
		return
	}

	p, err := h.service.AddDevice(r.Context(), req.DevID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, p)
}

// Logout clears stored Bambu Cloud credentials.
func (h *BambuCloudHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Logout(r.Context()); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ProjectSupplyHandler handles project supply HTTP requests.
type ProjectSupplyHandler struct {
	service *service.ProjectSupplyService
}

// List retrieves all supplies for a project.
func (h *ProjectSupplyHandler) List(w http.ResponseWriter, r *http.Request) {
	projectID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	supplies, err := h.service.ListByProject(r.Context(), projectID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if supplies == nil {
		supplies = []model.ProjectSupply{}
	}
	respondJSON(w, http.StatusOK, supplies)
}

// Create creates a new project supply.
func (h *ProjectSupplyHandler) Create(w http.ResponseWriter, r *http.Request) {
	projectID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	var supply model.ProjectSupply
	if err := json.NewDecoder(r.Body).Decode(&supply); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	supply.ProjectID = projectID

	if err := h.service.Create(r.Context(), &supply); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, supply)
}

// Delete removes a project supply.
func (h *ProjectSupplyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	supplyID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid supply ID")
		return
	}

	if err := h.service.Delete(r.Context(), supplyID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// BackupHandler handles database backup HTTP requests.
type BackupHandler struct {
	service *service.BackupService
}

// List returns all available backups.
func (h *BackupHandler) List(w http.ResponseWriter, r *http.Request) {
	backups, err := h.service.ListBackups(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, backups)
}

// Create creates a new backup.
func (h *BackupHandler) Create(w http.ResponseWriter, r *http.Request) {
	backup, err := h.service.CreateBackup(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, backup)
}

// Delete removes a backup.
func (h *BackupHandler) Delete(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		respondError(w, http.StatusBadRequest, "backup name required")
		return
	}

	if err := h.service.DeleteBackup(r.Context(), name); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Restore restores the database from a backup.
// WARNING: This will restart the application!
func (h *BackupHandler) Restore(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		respondError(w, http.StatusBadRequest, "backup name required")
		return
	}

	if err := h.service.RestoreBackup(r.Context(), name); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Database restored. Please restart the application.",
	})
}

// GetConfig returns the backup configuration.
func (h *BackupHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	config := h.service.GetConfig(r.Context())
	respondJSON(w, http.StatusOK, config)
}

// UpdateConfig updates the backup configuration.
func (h *BackupHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var config service.BackupConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.UpdateConfig(r.Context(), config); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, config)
}

