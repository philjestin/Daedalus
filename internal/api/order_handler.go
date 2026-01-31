package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/hyperion/printfarm/internal/model"
	"github.com/hyperion/printfarm/internal/service"
)

// OrderHandler handles order-related HTTP requests.
type OrderHandler struct {
	service *service.OrderService
}

// NewOrderHandler creates a new OrderHandler.
func NewOrderHandler(svc *service.OrderService) *OrderHandler {
	return &OrderHandler{service: svc}
}

// List returns orders with optional filtering.
func (h *OrderHandler) List(w http.ResponseWriter, r *http.Request) {
	filters := model.OrderFilters{}

	// Parse query parameters
	if status := r.URL.Query().Get("status"); status != "" {
		s := model.OrderStatus(status)
		filters.Status = &s
	}
	if source := r.URL.Query().Get("source"); source != "" {
		s := model.OrderSource(source)
		filters.Source = &s
	}
	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			filters.Limit = l
		}
	}
	if offset := r.URL.Query().Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			filters.Offset = o
		}
	}

	orders, err := h.service.List(r.Context(), filters)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if orders == nil {
		orders = []model.Order{}
	}
	respondJSON(w, http.StatusOK, orders)
}

// Get retrieves a single order by ID.
func (h *OrderHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	order, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if order == nil {
		respondError(w, http.StatusNotFound, "order not found")
		return
	}

	respondJSON(w, http.StatusOK, order)
}

// CreateOrderRequest represents a request to create a new order.
type CreateOrderRequest struct {
	CustomerName  string     `json:"customer_name"`
	CustomerEmail string     `json:"customer_email"`
	DueDate       *time.Time `json:"due_date"`
	Priority      int        `json:"priority"`
	Notes         string     `json:"notes"`
}

// Create creates a new manual order.
func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	order := &model.Order{
		Source:        model.OrderSourceManual,
		CustomerName:  req.CustomerName,
		CustomerEmail: req.CustomerEmail,
		DueDate:       req.DueDate,
		Priority:      req.Priority,
		Notes:         req.Notes,
		Status:        model.OrderStatusPending,
	}

	if err := h.service.Create(r.Context(), order); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, order)
}

// UpdateOrderRequest represents a request to update an order.
type UpdateOrderRequest struct {
	CustomerName  *string            `json:"customer_name,omitempty"`
	CustomerEmail *string            `json:"customer_email,omitempty"`
	DueDate       *time.Time         `json:"due_date,omitempty"`
	Priority      *int               `json:"priority,omitempty"`
	Notes         *string            `json:"notes,omitempty"`
	Status        *model.OrderStatus `json:"status,omitempty"`
}

// Update updates an order.
func (h *OrderHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	order, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if order == nil {
		respondError(w, http.StatusNotFound, "order not found")
		return
	}

	var req UpdateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.CustomerName != nil {
		order.CustomerName = *req.CustomerName
	}
	if req.CustomerEmail != nil {
		order.CustomerEmail = *req.CustomerEmail
	}
	if req.DueDate != nil {
		order.DueDate = req.DueDate
	}
	if req.Priority != nil {
		order.Priority = *req.Priority
	}
	if req.Notes != nil {
		order.Notes = *req.Notes
	}

	if err := h.service.Update(r.Context(), order); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Handle status update separately
	if req.Status != nil && *req.Status != order.Status {
		if err := h.service.UpdateStatus(r.Context(), id, *req.Status); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	// Reload order
	order, _ = h.service.GetByID(r.Context(), id)
	respondJSON(w, http.StatusOK, order)
}

// Delete removes an order.
func (h *OrderHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpdateStatus updates the status of an order.
func (h *OrderHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	var req struct {
		Status model.OrderStatus `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.UpdateStatus(r.Context(), id, req.Status); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	order, _ := h.service.GetByID(r.Context(), id)
	respondJSON(w, http.StatusOK, order)
}

// AddItemRequest represents a request to add an item to an order.
type AddItemRequest struct {
	TemplateID *uuid.UUID `json:"template_id,omitempty"`
	SKU        string     `json:"sku"`
	Quantity   int        `json:"quantity"`
	Notes      string     `json:"notes"`
}

// AddItem adds an item to an order.
func (h *OrderHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	orderID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	var req AddItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item := &model.OrderItem{
		OrderID:    orderID,
		TemplateID: req.TemplateID,
		SKU:        req.SKU,
		Quantity:   req.Quantity,
		Notes:      req.Notes,
	}

	if err := h.service.AddItem(r.Context(), orderID, item); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, item)
}

// RemoveItem removes an item from an order.
func (h *OrderHandler) RemoveItem(w http.ResponseWriter, r *http.Request) {
	orderID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid order ID")
		return
	}
	itemID, err := parseUUID(r, "itemId")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid item ID")
		return
	}

	if err := h.service.RemoveItem(r.Context(), orderID, itemID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ProcessItem creates a project from an order item's template.
func (h *OrderHandler) ProcessItem(w http.ResponseWriter, r *http.Request) {
	orderID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid order ID")
		return
	}
	itemID, err := parseUUID(r, "itemId")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid item ID")
		return
	}

	project, err := h.service.ProcessItem(r.Context(), orderID, itemID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, project)
}

// GetProgress returns the progress of an order.
func (h *OrderHandler) GetProgress(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	progress, err := h.service.GetProgress(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if progress == nil {
		respondError(w, http.StatusNotFound, "order not found")
		return
	}

	respondJSON(w, http.StatusOK, progress)
}

// GetCounts returns order counts by status.
func (h *OrderHandler) GetCounts(w http.ResponseWriter, r *http.Request) {
	counts, err := h.service.GetCounts(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, counts)
}

// MarkShippedRequest represents a request to mark an order as shipped.
type MarkShippedRequest struct {
	TrackingNumber string `json:"tracking_number"`
}

// MarkShipped marks an order as shipped.
func (h *OrderHandler) MarkShipped(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	var req MarkShippedRequest
	json.NewDecoder(r.Body).Decode(&req)

	if err := h.service.MarkShipped(r.Context(), id, req.TrackingNumber); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	order, _ := h.service.GetByID(r.Context(), id)
	respondJSON(w, http.StatusOK, order)
}
