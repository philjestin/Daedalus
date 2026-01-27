package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hyperion/printfarm/internal/model"
	"github.com/hyperion/printfarm/internal/printer"
	"github.com/hyperion/printfarm/internal/service"
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
	var status *model.ProjectStatus
	if s := r.URL.Query().Get("status"); s != "" {
		ps := model.ProjectStatus(s)
		status = &ps
	}

	projects, err := h.service.List(r.Context(), status)
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

// MarkReadyToShip marks a project as ready for shipping.
func (h *ProjectHandler) MarkReadyToShip(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	if err := h.service.MarkReadyToShip(r.Context(), id); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Return updated project
	project, _ := h.service.GetByID(r.Context(), id)
	respondJSON(w, http.StatusOK, project)
}

// ShipRequest represents the request body for marking a project as shipped.
type ShipRequest struct {
	TrackingNumber string `json:"tracking_number"`
}

// Ship marks a project as shipped with tracking number.
func (h *ProjectHandler) Ship(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	var req ShipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Tracking number is optional
		req = ShipRequest{}
	}

	if err := h.service.MarkShipped(r.Context(), id, req.TrackingNumber); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Return updated project
	project, _ := h.service.GetByID(r.Context(), id)
	respondJSON(w, http.StatusOK, project)
}

// PartHandler handles part endpoints.
type PartHandler struct {
	service *service.PartService
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

// Create creates a new part.
func (h *PartHandler) Create(w http.ResponseWriter, r *http.Request) {
	projectID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid project ID")
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

// MaterialHandler handles material endpoints.
type MaterialHandler struct {
	service *service.MaterialService
}

// List returns all materials.
func (h *MaterialHandler) List(w http.ResponseWriter, r *http.Request) {
	materials, err := h.service.List(r.Context())
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

// StatsHandler handles statistics endpoints.
type StatsHandler struct {
	service *service.StatsService
}

// GetFinancialSummary returns aggregated financial statistics.
func (h *StatsHandler) GetFinancialSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := h.service.GetFinancialSummary(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, summary)
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

