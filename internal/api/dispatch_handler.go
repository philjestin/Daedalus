package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/philjestin/daedalus/internal/model"
	"github.com/philjestin/daedalus/internal/service"
)

// DispatchHandler handles dispatch-related endpoints.
type DispatchHandler struct {
	service *service.DispatcherService
}

// NewDispatchHandler creates a new dispatch handler.
func NewDispatchHandler(svc *service.DispatcherService) *DispatchHandler {
	return &DispatchHandler{service: svc}
}

// ListPending returns all pending dispatch requests.
// GET /api/dispatch/requests
func (h *DispatchHandler) ListPending(w http.ResponseWriter, r *http.Request) {
	requests, err := h.service.ListPending(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if requests == nil {
		requests = []model.DispatchRequest{}
	}

	respondJSON(w, http.StatusOK, requests)
}

// Confirm confirms a dispatch request.
// POST /api/dispatch/requests/{id}/confirm
func (h *DispatchHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid request ID")
		return
	}

	if err := h.service.ConfirmDispatch(r.Context(), id); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "confirmed"})
}

// Reject rejects a dispatch request.
// POST /api/dispatch/requests/{id}/reject
func (h *DispatchHandler) Reject(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid request ID")
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	if err := h.service.RejectDispatch(r.Context(), id, req.Reason); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}

// Skip skips the current job and tries the next compatible one.
// POST /api/dispatch/requests/{id}/skip
func (h *DispatchHandler) Skip(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid request ID")
		return
	}

	if err := h.service.SkipJob(r.Context(), id); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "skipped"})
}

// GetGlobalSettings returns global auto-dispatch settings.
// GET /api/dispatch/settings
func (h *DispatchHandler) GetGlobalSettings(w http.ResponseWriter, r *http.Request) {
	enabled := h.service.IsGloballyEnabled(r.Context())
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"enabled": enabled,
	})
}

// UpdateGlobalSettings updates global auto-dispatch settings.
// PUT /api/dispatch/settings
func (h *DispatchHandler) UpdateGlobalSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.SetGlobalEnabled(r.Context(), req.Enabled); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// GetPrinterSettings returns auto-dispatch settings for a printer.
// GET /api/printers/{id}/dispatch-settings
func (h *DispatchHandler) GetPrinterSettings(w http.ResponseWriter, r *http.Request) {
	printerID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid printer ID")
		return
	}

	settings, err := h.service.GetSettings(r.Context(), printerID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, settings)
}

// UpdatePrinterSettings updates auto-dispatch settings for a printer.
// PUT /api/printers/{id}/dispatch-settings
func (h *DispatchHandler) UpdatePrinterSettings(w http.ResponseWriter, r *http.Request) {
	printerID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid printer ID")
		return
	}

	var settings model.AutoDispatchSettings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	settings.PrinterID = printerID

	if err := h.service.UpdateSettings(r.Context(), &settings); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, settings)
}

// PrintJobHandler extension for priority update
type printJobPriorityRequest struct {
	Priority int `json:"priority"`
}
