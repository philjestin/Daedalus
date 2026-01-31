package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hyperion/printfarm/internal/model"
	"github.com/hyperion/printfarm/internal/service"
)

// AlertsHandler handles alert-related HTTP requests.
type AlertsHandler struct {
	service *service.AlertService
}

// NewAlertsHandler creates a new AlertsHandler.
func NewAlertsHandler(svc *service.AlertService) *AlertsHandler {
	return &AlertsHandler{service: svc}
}

// List returns all active alerts.
func (h *AlertsHandler) List(w http.ResponseWriter, r *http.Request) {
	alerts, err := h.service.GetActiveAlerts(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if alerts == nil {
		alerts = []model.Alert{}
	}
	respondJSON(w, http.StatusOK, alerts)
}

// GetCounts returns alert counts by severity.
func (h *AlertsHandler) GetCounts(w http.ResponseWriter, r *http.Request) {
	counts, err := h.service.GetAlertCounts(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, counts)
}

// DismissRequest represents a request to dismiss an alert.
type DismissRequest struct {
	Duration string `json:"duration"` // "1h", "4h", "24h", "permanent"
}

// Dismiss dismisses an alert.
func (h *AlertsHandler) Dismiss(w http.ResponseWriter, r *http.Request) {
	alertType := model.AlertType(chi.URLParam(r, "type"))
	entityID := chi.URLParam(r, "entityId")

	var req DismissRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Default to 1 hour if no body
		req.Duration = "1h"
	}

	var until *time.Time
	if req.Duration != "permanent" {
		until = service.GetSnoozeUntil(req.Duration)
	}

	if err := h.service.DismissAlert(r.Context(), alertType, entityID, until); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "dismissed"})
}

// Undismiss removes a dismissal.
func (h *AlertsHandler) Undismiss(w http.ResponseWriter, r *http.Request) {
	alertType := model.AlertType(chi.URLParam(r, "type"))
	entityID := chi.URLParam(r, "entityId")

	if err := h.service.UndismissAlert(r.Context(), alertType, entityID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "undismissed"})
}

// UpdateThresholdRequest represents a request to update a material's low threshold.
type UpdateThresholdRequest struct {
	ThresholdGrams int `json:"threshold_grams"`
}

// UpdateMaterialThreshold updates the low threshold for a material.
func (h *AlertsHandler) UpdateMaterialThreshold(w http.ResponseWriter, r *http.Request) {
	materialID, err := parseUUID(r, "materialId")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid material ID")
		return
	}

	var req UpdateThresholdRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.UpdateMaterialThreshold(r.Context(), materialID, req.ThresholdGrams); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}
