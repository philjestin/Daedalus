package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hyperion/printfarm/internal/model"
	"github.com/hyperion/printfarm/internal/service"
)

// TaskHandler handles task HTTP requests.
type TaskHandler struct {
	service *service.TaskService
}

// NewTaskHandler creates a new TaskHandler.
func NewTaskHandler(service *service.TaskService) *TaskHandler {
	return &TaskHandler{service: service}
}

// List returns all tasks with optional filters.
func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	filters := model.TaskFilters{}

	// Parse optional filters
	if projectID := r.URL.Query().Get("project_id"); projectID != "" {
		id, err := uuid.Parse(projectID)
		if err == nil {
			filters.ProjectID = &id
		}
	}
	if orderID := r.URL.Query().Get("order_id"); orderID != "" {
		id, err := uuid.Parse(orderID)
		if err == nil {
			filters.OrderID = &id
		}
	}
	if status := r.URL.Query().Get("status"); status != "" {
		s := model.TaskStatus(status)
		filters.Status = &s
	}

	tasks, err := h.service.List(ctx, filters)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, tasks)
}

// Create creates a new task.
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		ProjectID   uuid.UUID  `json:"project_id"`
		OrderID     *uuid.UUID `json:"order_id,omitempty"`
		OrderItemID *uuid.UUID `json:"order_item_id,omitempty"`
		Name        string     `json:"name"`
		Quantity    int        `json:"quantity"`
		Notes       string     `json:"notes,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	task := &model.Task{
		ProjectID:   req.ProjectID,
		OrderID:     req.OrderID,
		OrderItemID: req.OrderItemID,
		Name:        req.Name,
		Quantity:    req.Quantity,
		Notes:       req.Notes,
	}

	if err := h.service.Create(ctx, task); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	respondJSON(w, http.StatusCreated, task)
}

// Get returns a task by ID.
func (h *TaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid task ID", http.StatusBadRequest)
		return
	}

	task, err := h.service.GetByID(ctx, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if task == nil {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, task)
}

// Update updates a task.
func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid task ID", http.StatusBadRequest)
		return
	}

	task, err := h.service.GetByID(ctx, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if task == nil {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	var req struct {
		Name     *string `json:"name,omitempty"`
		Quantity *int    `json:"quantity,omitempty"`
		Notes    *string `json:"notes,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name != nil {
		task.Name = *req.Name
	}
	if req.Quantity != nil {
		task.Quantity = *req.Quantity
	}
	if req.Notes != nil {
		task.Notes = *req.Notes
	}

	if err := h.service.Update(ctx, task); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, task)
}

// UpdateStatus updates the status of a task.
func (h *TaskHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid task ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Status model.TaskStatus `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.service.UpdateStatus(ctx, id, req.Status); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	task, _ := h.service.GetByID(ctx, id)
	respondJSON(w, http.StatusOK, task)
}

// Delete removes a task.
func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid task ID", http.StatusBadRequest)
		return
	}

	if err := h.service.Delete(ctx, id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListByProject returns tasks for a project.
func (h *TaskHandler) ListByProject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid project ID", http.StatusBadRequest)
		return
	}

	tasks, err := h.service.ListByProject(ctx, projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, tasks)
}

// GetProgress returns the progress of a task.
func (h *TaskHandler) GetProgress(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid task ID", http.StatusBadRequest)
		return
	}

	progress, err := h.service.GetProgress(ctx, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]float64{"progress": progress})
}

// StartTask marks a task as in_progress.
func (h *TaskHandler) StartTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid task ID", http.StatusBadRequest)
		return
	}

	if err := h.service.StartTask(ctx, id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	task, _ := h.service.GetByID(ctx, id)
	respondJSON(w, http.StatusOK, task)
}

// CompleteTask marks a task as completed.
func (h *TaskHandler) CompleteTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid task ID", http.StatusBadRequest)
		return
	}

	if err := h.service.CompleteTask(ctx, id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	task, _ := h.service.GetByID(ctx, id)
	respondJSON(w, http.StatusOK, task)
}

// CancelTask marks a task as cancelled.
func (h *TaskHandler) CancelTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid task ID", http.StatusBadRequest)
		return
	}

	if err := h.service.CancelTask(ctx, id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	task, _ := h.service.GetByID(ctx, id)
	respondJSON(w, http.StatusOK, task)
}
