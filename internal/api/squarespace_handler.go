package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/philjestin/daedalus/internal/model"
	"github.com/philjestin/daedalus/internal/service"
)

// SquarespaceHandler handles Squarespace integration endpoints.
type SquarespaceHandler struct {
	service  *service.SquarespaceService
	orderSvc *service.OrderService
}

// NewSquarespaceHandler creates a new SquarespaceHandler.
func NewSquarespaceHandler(svc *service.SquarespaceService, orderSvc *service.OrderService) *SquarespaceHandler {
	return &SquarespaceHandler{service: svc, orderSvc: orderSvc}
}

// ConnectRequest represents the request body for connecting to Squarespace.
type ConnectRequest struct {
	APIKey string `json:"api_key"`
}

// Connect validates the API key and saves the integration.
// POST /api/integrations/squarespace/connect
func (h *SquarespaceHandler) Connect(w http.ResponseWriter, r *http.Request) {
	var req ConnectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.APIKey == "" {
		respondError(w, http.StatusBadRequest, "api_key is required")
		return
	}

	integration, err := h.service.Connect(r.Context(), req.APIKey)
	if err != nil {
		slog.Error("failed to connect Squarespace", "error", err)
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, integration)
}

// GetStatus returns the current Squarespace integration status.
// GET /api/integrations/squarespace/status
func (h *SquarespaceHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	integration, err := h.service.GetStatus(r.Context())
	if err != nil {
		slog.Error("failed to get Squarespace status", "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if integration == nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"connected": false,
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"connected":             true,
		"id":                    integration.ID,
		"site_id":               integration.SiteID,
		"site_title":            integration.SiteTitle,
		"is_active":             integration.IsActive,
		"last_order_sync_at":    integration.LastOrderSyncAt,
		"last_product_sync_at":  integration.LastProductSyncAt,
		"created_at":            integration.CreatedAt,
		"updated_at":            integration.UpdatedAt,
	})
}

// Disconnect removes the Squarespace integration.
// POST /api/integrations/squarespace/disconnect
func (h *SquarespaceHandler) Disconnect(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Disconnect(r.Context()); err != nil {
		slog.Error("failed to disconnect Squarespace", "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "disconnected"})
}

// ---- Order Handlers ----

// SyncOrders fetches new orders from Squarespace.
// POST /api/integrations/squarespace/orders/sync
func (h *SquarespaceHandler) SyncOrders(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.SyncOrders(r.Context())
	if err != nil {
		slog.Error("failed to sync Squarespace orders", "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// ListOrders returns stored Squarespace orders.
// GET /api/integrations/squarespace/orders
func (h *SquarespaceHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	var processed *bool
	if p := r.URL.Query().Get("processed"); p != "" {
		val := p == "true"
		processed = &val
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			offset = val
		}
	}

	orders, err := h.service.ListOrders(r.Context(), processed, limit, offset)
	if err != nil {
		slog.Error("failed to list Squarespace orders", "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if orders == nil {
		orders = []model.SquarespaceOrder{}
	}

	respondJSON(w, http.StatusOK, orders)
}

// GetOrder returns a single order by ID.
// GET /api/integrations/squarespace/orders/{id}
func (h *SquarespaceHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	order, err := h.service.GetOrder(r.Context(), id)
	if err != nil {
		slog.Error("failed to get Squarespace order", "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if order == nil {
		respondError(w, http.StatusNotFound, "order not found")
		return
	}

	respondJSON(w, http.StatusOK, order)
}

// ProcessOrder creates a unified order from a Squarespace order.
// POST /api/integrations/squarespace/orders/{id}/process
func (h *SquarespaceHandler) ProcessOrder(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	order, err := h.service.ProcessOrder(r.Context(), id, h.orderSvc)
	if err != nil {
		slog.Error("failed to process Squarespace order", "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"order_id": order.ID,
		"order":    order,
	})
}

// ---- Product Handlers ----

// SyncProducts fetches products from Squarespace.
// POST /api/integrations/squarespace/products/sync
func (h *SquarespaceHandler) SyncProducts(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.SyncProducts(r.Context())
	if err != nil {
		slog.Error("failed to sync Squarespace products", "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// ListProducts returns stored Squarespace products.
// GET /api/integrations/squarespace/products
func (h *SquarespaceHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			offset = val
		}
	}

	products, err := h.service.ListProducts(r.Context(), limit, offset)
	if err != nil {
		slog.Error("failed to list Squarespace products", "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if products == nil {
		products = []model.SquarespaceProduct{}
	}

	respondJSON(w, http.StatusOK, products)
}

// GetProduct returns a single product by ID.
// GET /api/integrations/squarespace/products/{id}
func (h *SquarespaceHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid product ID")
		return
	}

	product, err := h.service.GetProduct(r.Context(), id)
	if err != nil {
		slog.Error("failed to get Squarespace product", "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if product == nil {
		respondError(w, http.StatusNotFound, "product not found")
		return
	}

	respondJSON(w, http.StatusOK, product)
}

// LinkProductRequest represents the request body for linking a product to a template.
type LinkProductRequest struct {
	TemplateID string `json:"template_id"`
	SKU        string `json:"sku"`
}

// LinkProduct links a Squarespace product to a template.
// POST /api/integrations/squarespace/products/{id}/link
func (h *SquarespaceHandler) LinkProduct(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid product ID")
		return
	}

	var req LinkProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	templateID, err := parseUUIDString(req.TemplateID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid template_id")
		return
	}

	if err := h.service.LinkProductToTemplate(r.Context(), id, templateID, req.SKU); err != nil {
		slog.Error("failed to link Squarespace product to template", "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "linked"})
}

// UnlinkProduct removes a product-template link.
// DELETE /api/integrations/squarespace/products/{id}/link
func (h *SquarespaceHandler) UnlinkProduct(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid product ID")
		return
	}

	templateIDStr := r.URL.Query().Get("template_id")
	if templateIDStr == "" {
		respondError(w, http.StatusBadRequest, "template_id query parameter is required")
		return
	}

	templateID, err := parseUUIDString(templateIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid template_id")
		return
	}

	if err := h.service.UnlinkProductFromTemplate(r.Context(), id, templateID); err != nil {
		slog.Error("failed to unlink Squarespace product from template", "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
