package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hyperion/printfarm/internal/model"
	"github.com/hyperion/printfarm/internal/service"
)

// respondJSON sends a JSON response.
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
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
	printers, err := h.service.DiscoverPrinters(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
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

