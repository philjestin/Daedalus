package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hyperion/printfarm/internal/model"
	"github.com/hyperion/printfarm/internal/service"
)

// FeedbackHandler handles feedback endpoints.
type FeedbackHandler struct {
	service *service.FeedbackService
}

// CreateFeedbackRequest is the request body for submitting feedback.
type CreateFeedbackRequest struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Contact string `json:"contact,omitempty"`
	Page    string `json:"page,omitempty"`
}

// Submit handles POST /api/feedback.
func (h *FeedbackHandler) Submit(w http.ResponseWriter, r *http.Request) {
	var req CreateFeedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	feedback := &model.Feedback{
		Type:    req.Type,
		Message: req.Message,
		Contact: req.Contact,
		Page:    req.Page,
	}

	if err := h.service.Submit(r.Context(), feedback); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, feedback)
}

// List handles GET /api/feedback.
func (h *FeedbackHandler) List(w http.ResponseWriter, r *http.Request) {
	feedback, err := h.service.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list feedback")
		return
	}

	respondJSON(w, http.StatusOK, feedback)
}

// Delete handles DELETE /api/feedback/{id}.
func (h *FeedbackHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid feedback ID")
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusNotFound, "feedback not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
