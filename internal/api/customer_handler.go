package api

import (
	"encoding/json"
	"net/http"

	"github.com/philjestin/daedalus/internal/model"
	"github.com/philjestin/daedalus/internal/service"
)

// CustomerHandler handles customer-related HTTP requests.
type CustomerHandler struct {
	service *service.CustomerService
}

// NewCustomerHandler creates a new CustomerHandler.
func NewCustomerHandler(svc *service.CustomerService) *CustomerHandler {
	return &CustomerHandler{service: svc}
}

// List returns customers with optional search filtering.
func (h *CustomerHandler) List(w http.ResponseWriter, r *http.Request) {
	filters := model.CustomerFilters{
		Search: r.URL.Query().Get("search"),
	}

	customers, err := h.service.List(r.Context(), filters)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if customers == nil {
		customers = []model.Customer{}
	}
	respondJSON(w, http.StatusOK, customers)
}

// Get retrieves a single customer by ID.
func (h *CustomerHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid customer ID")
		return
	}

	customer, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if customer == nil {
		respondError(w, http.StatusNotFound, "customer not found")
		return
	}

	respondJSON(w, http.StatusOK, customer)
}

// CreateCustomerRequest represents a request to create a customer.
type CreateCustomerRequest struct {
	Name            string         `json:"name"`
	Email           string         `json:"email"`
	Company         string         `json:"company"`
	Phone           string         `json:"phone"`
	Notes           string         `json:"notes"`
	BillingAddress  *model.Address `json:"billing_address"`
	ShippingAddress *model.Address `json:"shipping_address"`
}

// Create creates a new customer.
func (h *CustomerHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateCustomerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	customer := &model.Customer{
		Name:            req.Name,
		Email:           req.Email,
		Company:         req.Company,
		Phone:           req.Phone,
		Notes:           req.Notes,
		BillingAddress:  req.BillingAddress,
		ShippingAddress: req.ShippingAddress,
	}

	if err := h.service.Create(r.Context(), customer); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, customer)
}

// UpdateCustomerRequest represents a request to update a customer.
type UpdateCustomerRequest struct {
	Name            *string        `json:"name,omitempty"`
	Email           *string        `json:"email,omitempty"`
	Company         *string        `json:"company,omitempty"`
	Phone           *string        `json:"phone,omitempty"`
	Notes           *string        `json:"notes,omitempty"`
	BillingAddress  *model.Address `json:"billing_address,omitempty"`
	ShippingAddress *model.Address `json:"shipping_address,omitempty"`
}

// Update updates a customer.
func (h *CustomerHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid customer ID")
		return
	}

	customer, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if customer == nil {
		respondError(w, http.StatusNotFound, "customer not found")
		return
	}

	var req UpdateCustomerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name != nil {
		customer.Name = *req.Name
	}
	if req.Email != nil {
		customer.Email = *req.Email
	}
	if req.Company != nil {
		customer.Company = *req.Company
	}
	if req.Phone != nil {
		customer.Phone = *req.Phone
	}
	if req.Notes != nil {
		customer.Notes = *req.Notes
	}
	if req.BillingAddress != nil {
		customer.BillingAddress = req.BillingAddress
	}
	if req.ShippingAddress != nil {
		customer.ShippingAddress = req.ShippingAddress
	}

	if err := h.service.Update(r.Context(), customer); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, customer)
}

// Delete removes a customer.
func (h *CustomerHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid customer ID")
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
