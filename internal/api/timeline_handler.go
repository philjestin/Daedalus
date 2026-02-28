package api

import (
	"net/http"
	"time"

	"github.com/philjestin/daedalus/internal/model"
	"github.com/philjestin/daedalus/internal/service"
)

// TimelineHandler handles timeline-related HTTP requests.
type TimelineHandler struct {
	service *service.TimelineService
}

// NewTimelineHandler creates a new TimelineHandler.
func NewTimelineHandler(svc *service.TimelineService) *TimelineHandler {
	return &TimelineHandler{service: svc}
}

// GetTimeline returns timeline items for the Gantt view.
func (h *TimelineHandler) GetTimeline(w http.ResponseWriter, r *http.Request) {
	var startDate, endDate *time.Time

	if start := r.URL.Query().Get("start"); start != "" {
		if t, err := time.Parse(time.RFC3339, start); err == nil {
			startDate = &t
		}
	}

	if end := r.URL.Query().Get("end"); end != "" {
		if t, err := time.Parse(time.RFC3339, end); err == nil {
			endDate = &t
		}
	}

	items, err := h.service.GetTimeline(r.Context(), startDate, endDate)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if items == nil {
		items = []model.TimelineItem{}
	}

	respondJSON(w, http.StatusOK, items)
}

// GetOrderTimeline returns a detailed timeline for a single order.
func (h *TimelineHandler) GetOrderTimeline(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	item, err := h.service.GetOrderTimeline(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if item == nil {
		respondError(w, http.StatusNotFound, "order not found")
		return
	}

	respondJSON(w, http.StatusOK, item)
}

// GetProjectTimeline returns a detailed timeline for a single project.
func (h *TimelineHandler) GetProjectTimeline(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	item, err := h.service.GetProjectTimeline(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if item == nil {
		respondError(w, http.StatusNotFound, "project not found")
		return
	}

	respondJSON(w, http.StatusOK, item)
}
