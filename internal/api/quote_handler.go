package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/philjestin/daedalus/internal/model"
	"github.com/philjestin/daedalus/internal/service"
)

// QuoteHandler handles quote-related HTTP requests.
type QuoteHandler struct {
	service *service.QuoteService
}

// NewQuoteHandler creates a new QuoteHandler.
func NewQuoteHandler(svc *service.QuoteService) *QuoteHandler {
	return &QuoteHandler{service: svc}
}

// List returns quotes with optional filtering.
func (h *QuoteHandler) List(w http.ResponseWriter, r *http.Request) {
	filters := model.QuoteFilters{}

	if status := r.URL.Query().Get("status"); status != "" {
		s := model.QuoteStatus(status)
		filters.Status = &s
	}
	if customerID := r.URL.Query().Get("customer_id"); customerID != "" {
		if id, err := uuid.Parse(customerID); err == nil {
			filters.CustomerID = &id
		}
	}

	quotes, err := h.service.List(r.Context(), filters)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if quotes == nil {
		quotes = []model.Quote{}
	}
	respondJSON(w, http.StatusOK, quotes)
}

// Get retrieves a single quote by ID.
func (h *QuoteHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid quote ID")
		return
	}

	quote, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if quote == nil {
		respondError(w, http.StatusNotFound, "quote not found")
		return
	}

	respondJSON(w, http.StatusOK, quote)
}

// CreateQuoteRequest represents a request to create a new quote.
type CreateQuoteRequest struct {
	CustomerID string     `json:"customer_id"`
	Title      string     `json:"title"`
	Notes      string     `json:"notes"`
	ValidUntil *time.Time `json:"valid_until"`
}

// Create creates a new quote.
func (h *QuoteHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateQuoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	customerID, err := uuid.Parse(req.CustomerID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid customer ID")
		return
	}

	quote := &model.Quote{
		CustomerID: customerID,
		Title:      req.Title,
		Notes:      req.Notes,
		ValidUntil: req.ValidUntil,
	}

	if err := h.service.Create(r.Context(), quote); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, quote)
}

// UpdateQuoteRequest represents a request to update a quote.
type UpdateQuoteRequest struct {
	Title      *string    `json:"title,omitempty"`
	Notes      *string    `json:"notes,omitempty"`
	ValidUntil *time.Time `json:"valid_until,omitempty"`
	CustomerID *string    `json:"customer_id,omitempty"`
}

// Update updates a quote.
func (h *QuoteHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid quote ID")
		return
	}

	quote, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if quote == nil {
		respondError(w, http.StatusNotFound, "quote not found")
		return
	}

	var req UpdateQuoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Title != nil {
		quote.Title = *req.Title
	}
	if req.Notes != nil {
		quote.Notes = *req.Notes
	}
	if req.ValidUntil != nil {
		quote.ValidUntil = req.ValidUntil
	}
	if req.CustomerID != nil {
		cID, err := uuid.Parse(*req.CustomerID)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid customer ID")
			return
		}
		quote.CustomerID = cID
	}

	if err := h.service.Update(r.Context(), quote); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, quote)
}

// Delete removes a quote.
func (h *QuoteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid quote ID")
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// Send transitions a quote from draft to sent.
func (h *QuoteHandler) Send(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid quote ID")
		return
	}

	quote, err := h.service.Send(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, quote)
}

// AcceptQuoteRequest represents a request to accept a quote with a chosen option.
type AcceptQuoteRequest struct {
	OptionID string `json:"option_id"`
}

// Accept transitions a quote to accepted and creates an order.
func (h *QuoteHandler) Accept(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid quote ID")
		return
	}

	var req AcceptQuoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	optionID, err := uuid.Parse(req.OptionID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid option ID")
		return
	}

	quote, err := h.service.Accept(r.Context(), id, optionID)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, quote)
}

// Reject transitions a quote to rejected.
func (h *QuoteHandler) Reject(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid quote ID")
		return
	}

	quote, err := h.service.Reject(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, quote)
}

// CreateOptionRequest represents a request to add an option to a quote.
type CreateOptionRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
}

// CreateOption adds an option to a quote.
func (h *QuoteHandler) CreateOption(w http.ResponseWriter, r *http.Request) {
	quoteID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid quote ID")
		return
	}

	var req CreateOptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	option := &model.QuoteOption{
		QuoteID:     quoteID,
		Name:        req.Name,
		Description: req.Description,
		SortOrder:   req.SortOrder,
	}

	if err := h.service.CreateOption(r.Context(), option); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, option)
}

// UpdateOptionRequest represents a request to update a quote option.
type UpdateOptionRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	SortOrder   *int    `json:"sort_order,omitempty"`
}

// UpdateOption updates a quote option.
func (h *QuoteHandler) UpdateOption(w http.ResponseWriter, r *http.Request) {
	quoteID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid quote ID")
		return
	}

	optionID, err := parseUUID(r, "optionId")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid option ID")
		return
	}

	var req UpdateOptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	option := &model.QuoteOption{
		ID:      optionID,
		QuoteID: quoteID,
	}

	if req.Name != nil {
		option.Name = *req.Name
	}
	if req.Description != nil {
		option.Description = *req.Description
	}
	if req.SortOrder != nil {
		option.SortOrder = *req.SortOrder
	}

	if err := h.service.UpdateOption(r.Context(), option); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, option)
}

// DeleteOption removes a quote option.
func (h *QuoteHandler) DeleteOption(w http.ResponseWriter, r *http.Request) {
	quoteID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid quote ID")
		return
	}

	optionID, err := parseUUID(r, "optionId")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid option ID")
		return
	}

	if err := h.service.DeleteOption(r.Context(), quoteID, optionID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// CreateLineItemRequest represents a request to add a line item.
type CreateLineItemRequest struct {
	Type           string  `json:"type"`
	Description    string  `json:"description"`
	Quantity       float64 `json:"quantity"`
	Unit           string  `json:"unit"`
	UnitPriceCents int     `json:"unit_price_cents"`
	TotalCents     int     `json:"total_cents"`
	SortOrder      int     `json:"sort_order"`
	ProjectID      *string `json:"project_id,omitempty"`
}

// CreateLineItem adds a line item to a quote option.
func (h *QuoteHandler) CreateLineItem(w http.ResponseWriter, r *http.Request) {
	optionID, err := parseUUID(r, "optionId")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid option ID")
		return
	}

	var req CreateLineItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item := &model.QuoteLineItem{
		OptionID:       optionID,
		Type:           model.QuoteLineItemType(req.Type),
		Description:    req.Description,
		Quantity:       req.Quantity,
		Unit:           req.Unit,
		UnitPriceCents: req.UnitPriceCents,
		TotalCents:     req.TotalCents,
		SortOrder:      req.SortOrder,
	}

	if req.ProjectID != nil && *req.ProjectID != "" {
		pid, err := uuid.Parse(*req.ProjectID)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid project ID")
			return
		}
		item.ProjectID = &pid
	}

	if err := h.service.CreateLineItem(r.Context(), item); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, item)
}

// UpdateLineItemRequest represents a request to update a line item.
type UpdateLineItemRequest struct {
	Type           *string  `json:"type,omitempty"`
	Description    *string  `json:"description,omitempty"`
	Quantity       *float64 `json:"quantity,omitempty"`
	Unit           *string  `json:"unit,omitempty"`
	UnitPriceCents *int     `json:"unit_price_cents,omitempty"`
	TotalCents     *int     `json:"total_cents,omitempty"`
	SortOrder      *int     `json:"sort_order,omitempty"`
	ProjectID      *string  `json:"project_id,omitempty"`
}

// UpdateLineItem updates a line item.
func (h *QuoteHandler) UpdateLineItem(w http.ResponseWriter, r *http.Request) {
	optionID, err := parseUUID(r, "optionId")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid option ID")
		return
	}

	itemID, err := parseUUID(r, "itemId")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid item ID")
		return
	}

	var req UpdateLineItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item := &model.QuoteLineItem{
		ID:       itemID,
		OptionID: optionID,
	}

	if req.Type != nil {
		item.Type = model.QuoteLineItemType(*req.Type)
	}
	if req.Description != nil {
		item.Description = *req.Description
	}
	if req.Quantity != nil {
		item.Quantity = *req.Quantity
	}
	if req.Unit != nil {
		item.Unit = *req.Unit
	}
	if req.UnitPriceCents != nil {
		item.UnitPriceCents = *req.UnitPriceCents
	}
	if req.TotalCents != nil {
		item.TotalCents = *req.TotalCents
	}
	if req.SortOrder != nil {
		item.SortOrder = *req.SortOrder
	}
	if req.ProjectID != nil {
		if *req.ProjectID == "" {
			item.ProjectID = nil
		} else {
			pid, err := uuid.Parse(*req.ProjectID)
			if err != nil {
				respondError(w, http.StatusBadRequest, "invalid project ID")
				return
			}
			item.ProjectID = &pid
		}
	}

	if err := h.service.UpdateLineItem(r.Context(), item); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, item)
}

// DeleteLineItem removes a line item.
func (h *QuoteHandler) DeleteLineItem(w http.ResponseWriter, r *http.Request) {
	optionID, err := parseUUID(r, "optionId")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid option ID")
		return
	}

	itemID, err := parseUUID(r, "itemId")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid item ID")
		return
	}

	if err := h.service.DeleteLineItem(r.Context(), optionID, itemID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
