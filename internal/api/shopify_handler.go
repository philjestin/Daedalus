package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/hyperion/printfarm/internal/model"
	"github.com/hyperion/printfarm/internal/service"
)

// ShopifyHandler handles Shopify integration HTTP requests.
type ShopifyHandler struct {
	service  *service.ShopifyService
	orderSvc *service.OrderService
	config   service.ShopifyConfig
}

// NewShopifyHandler creates a new ShopifyHandler.
func NewShopifyHandler(svc *service.ShopifyService, orderSvc *service.OrderService, config service.ShopifyConfig) *ShopifyHandler {
	return &ShopifyHandler{
		service:  svc,
		orderSvc: orderSvc,
		config:   config,
	}
}

// GetAuthURL returns the OAuth authorization URL.
func (h *ShopifyHandler) GetAuthURL(w http.ResponseWriter, r *http.Request) {
	shopDomain := r.URL.Query().Get("shop")
	if shopDomain == "" {
		respondError(w, http.StatusBadRequest, "shop domain is required")
		return
	}

	authURL, err := h.service.GetAuthURL(shopDomain, h.config)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"auth_url": authURL})
}

// Callback handles the OAuth callback.
func (h *ShopifyHandler) Callback(w http.ResponseWriter, r *http.Request) {
	shopDomain := r.URL.Query().Get("shop")
	code := r.URL.Query().Get("code")

	if shopDomain == "" || code == "" {
		respondError(w, http.StatusBadRequest, "missing shop or code parameter")
		return
	}

	if err := h.service.HandleOAuthCallback(r.Context(), shopDomain, code, h.config); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Redirect to integrations page
	http.Redirect(w, r, "/settings/integrations?shopify=connected", http.StatusFound)
}

// GetStatus returns the current integration status.
func (h *ShopifyHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.service.GetStatus(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, status)
}

// Disconnect removes the Shopify integration.
func (h *ShopifyHandler) Disconnect(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Disconnect(r.Context()); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "disconnected"})
}

// SyncOrders triggers an order sync.
func (h *ShopifyHandler) SyncOrders(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.SyncOrders(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, result)
}

// ListOrders returns synced Shopify orders.
func (h *ShopifyHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	var processed *bool
	if p := r.URL.Query().Get("processed"); p != "" {
		b := p == "true"
		processed = &b
	}

	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil {
			offset = parsed
		}
	}

	orders, err := h.service.ListOrders(r.Context(), processed, limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if orders == nil {
		orders = []model.ShopifyOrder{}
	}
	respondJSON(w, http.StatusOK, orders)
}

// GetOrder returns a single Shopify order.
func (h *ShopifyHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	order, err := h.service.GetOrder(r.Context(), id)
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

// ProcessOrder creates a unified Order from a Shopify order.
func (h *ShopifyHandler) ProcessOrder(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	order, err := h.service.ProcessOrder(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, order)
}

// ShopifyLinkProductRequest represents a request to link a Shopify product to a template.
type ShopifyLinkProductRequest struct {
	TemplateID string `json:"template_id"`
	SKU        string `json:"sku"`
}

// LinkProduct links a Shopify product to a template.
func (h *ShopifyHandler) LinkProduct(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "productId")
	if productID == "" {
		respondError(w, http.StatusBadRequest, "product ID is required")
		return
	}

	var req ShopifyLinkProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	templateID, err := parseUUIDString(req.TemplateID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid template ID")
		return
	}

	if err := h.service.LinkProductToTemplate(r.Context(), productID, templateID, req.SKU); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "linked"})
}

// UnlinkProduct removes a product-template link.
func (h *ShopifyHandler) UnlinkProduct(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "productId")
	if productID == "" {
		respondError(w, http.StatusBadRequest, "product ID is required")
		return
	}

	templateIDStr := r.URL.Query().Get("template_id")
	templateID, err := parseUUIDString(templateIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid template ID")
		return
	}

	if err := h.service.UnlinkProductFromTemplate(r.Context(), productID, templateID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
