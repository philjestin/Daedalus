package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/philjestin/daedalus/internal/model"
	"github.com/philjestin/daedalus/internal/service"
)

// EtsyHandler handles Etsy integration endpoints.
type EtsyHandler struct {
	service       *service.EtsyService
	templateSvc   *service.TemplateService
	orderSvc      *service.OrderService
	webhookSecret string
}

// NewEtsyHandler creates a new EtsyHandler.
func NewEtsyHandler(svc *service.EtsyService, orderSvc *service.OrderService) *EtsyHandler {
	return &EtsyHandler{service: svc, orderSvc: orderSvc}
}

// SetTemplateSvc sets the template service for receipt processing.
func (h *EtsyHandler) SetTemplateSvc(svc *service.TemplateService) {
	h.templateSvc = svc
}

// SetWebhookSecret sets the webhook secret for signature verification.
func (h *EtsyHandler) SetWebhookSecret(secret string) {
	h.webhookSecret = secret
}

// ConfigureEtsyRequest represents the request body for configuring the Etsy integration.
type ConfigureEtsyRequest struct {
	ClientID    string `json:"client_id"`
	RedirectURI string `json:"redirect_uri,omitempty"`
}

// Configure saves the Etsy Client ID and activates the integration.
// PUT /api/integrations/etsy/configure
func (h *EtsyHandler) Configure(w http.ResponseWriter, r *http.Request) {
	var req ConfigureEtsyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ClientID == "" {
		respondError(w, http.StatusBadRequest, "client_id is required")
		return
	}
	if err := h.service.Configure(r.Context(), req.ClientID, req.RedirectURI); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "configured"})
}

// StartAuth initiates the OAuth flow and returns the authorization URL.
// GET /api/integrations/etsy/auth
func (h *EtsyHandler) StartAuth(w http.ResponseWriter, r *http.Request) {
	if !h.service.IsConfigured() {
		respondError(w, http.StatusServiceUnavailable, "Etsy integration not configured. Set ETSY_CLIENT_ID environment variable.")
		return
	}

	authURL, err := h.service.StartOAuth(r.Context())
	if err != nil {
		slog.Error("failed to start Etsy OAuth", "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"url": authURL})
}

// Callback handles the OAuth callback from Etsy.
// GET /api/integrations/etsy/callback
func (h *EtsyHandler) Callback(w http.ResponseWriter, r *http.Request) {
	if !h.service.IsConfigured() {
		http.Redirect(w, r, "/settings?etsy=error&message=not_configured", http.StatusTemporaryRedirect)
		return
	}

	// Check for error from Etsy
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		errDesc := r.URL.Query().Get("error_description")
		slog.Warn("Etsy OAuth error", "error", errParam, "description", errDesc)
		http.Redirect(w, r, "/settings?etsy=error&message="+errParam, http.StatusTemporaryRedirect)
		return
	}

	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")

	if state == "" || code == "" {
		slog.Warn("Etsy OAuth callback missing state or code")
		http.Redirect(w, r, "/settings?etsy=error&message=missing_params", http.StatusTemporaryRedirect)
		return
	}

	_, err := h.service.HandleCallback(r.Context(), state, code)
	if err != nil {
		slog.Error("failed to handle Etsy OAuth callback", "error", err)
		http.Redirect(w, r, "/settings?etsy=error&message=callback_failed", http.StatusTemporaryRedirect)
		return
	}

	// Redirect to settings page with success indicator
	http.Redirect(w, r, "/settings?etsy=connected", http.StatusTemporaryRedirect)
}

// GetStatus returns the current Etsy integration status.
// GET /api/integrations/etsy/status
func (h *EtsyHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	integration, err := h.service.GetStatus(r.Context())
	if err != nil {
		slog.Error("failed to get Etsy status", "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if integration == nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"connected":  false,
			"configured": h.service.IsConfigured(),
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"connected":        true,
		"configured":       h.service.IsConfigured(),
		"shop_id":          integration.ShopID,
		"shop_name":        integration.ShopName,
		"token_expires_at": integration.TokenExpiresAt,
		"scopes":           integration.Scopes,
		"is_active":        integration.IsActive,
		"last_sync_at":     integration.LastSyncAt,
		"created_at":       integration.CreatedAt,
		"updated_at":       integration.UpdatedAt,
	})
}

// Disconnect removes the Etsy integration.
// POST /api/integrations/etsy/disconnect
func (h *EtsyHandler) Disconnect(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Disconnect(r.Context()); err != nil {
		slog.Error("failed to disconnect Etsy", "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "disconnected"})
}

// ---- Receipt Handlers ----

// SyncReceipts fetches new receipts from Etsy.
// POST /api/integrations/etsy/receipts/sync
func (h *EtsyHandler) SyncReceipts(w http.ResponseWriter, r *http.Request) {
	if !h.service.IsConfigured() {
		respondError(w, http.StatusServiceUnavailable, "Etsy integration not configured")
		return
	}

	result, err := h.service.SyncReceipts(r.Context())
	if err != nil {
		slog.Error("failed to sync receipts", "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// ListReceipts returns stored Etsy receipts.
// GET /api/integrations/etsy/receipts
func (h *EtsyHandler) ListReceipts(w http.ResponseWriter, r *http.Request) {
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

	receipts, err := h.service.ListReceipts(r.Context(), processed, limit, offset)
	if err != nil {
		slog.Error("failed to list receipts", "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if receipts == nil {
		receipts = []model.EtsyReceipt{}
	}

	respondJSON(w, http.StatusOK, receipts)
}

// GetReceipt returns a single receipt by ID.
// GET /api/integrations/etsy/receipts/{id}
func (h *EtsyHandler) GetReceipt(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid receipt ID")
		return
	}

	receipt, err := h.service.GetReceipt(r.Context(), id)
	if err != nil {
		slog.Error("failed to get receipt", "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if receipt == nil {
		respondError(w, http.StatusNotFound, "receipt not found")
		return
	}

	respondJSON(w, http.StatusOK, receipt)
}

// ProcessReceipt creates a project from a receipt.
// POST /api/integrations/etsy/receipts/{id}/process
func (h *EtsyHandler) ProcessReceipt(w http.ResponseWriter, r *http.Request) {
	if h.templateSvc == nil {
		respondError(w, http.StatusInternalServerError, "template service not configured")
		return
	}

	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid receipt ID")
		return
	}

	order, err := h.service.ProcessReceipt(r.Context(), id, h.templateSvc, h.orderSvc)
	if err != nil {
		slog.Error("failed to process receipt", "error", err)
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"order": order,
	})
}

// ---- Listing Handlers ----

// SyncListings fetches active listings from Etsy.
// POST /api/integrations/etsy/listings/sync
func (h *EtsyHandler) SyncListings(w http.ResponseWriter, r *http.Request) {
	if !h.service.IsConfigured() {
		respondError(w, http.StatusServiceUnavailable, "Etsy integration not configured")
		return
	}

	result, err := h.service.SyncListings(r.Context())
	if err != nil {
		slog.Error("failed to sync listings", "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// ListListings returns stored Etsy listings.
// GET /api/integrations/etsy/listings
func (h *EtsyHandler) ListListings(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")

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

	listings, err := h.service.ListListings(r.Context(), state, limit, offset)
	if err != nil {
		slog.Error("failed to list listings", "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if listings == nil {
		listings = []model.EtsyListing{}
	}

	respondJSON(w, http.StatusOK, listings)
}

// GetListing returns a single listing by ID.
// GET /api/integrations/etsy/listings/{id}
func (h *EtsyHandler) GetListing(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid listing ID")
		return
	}

	listing, err := h.service.GetListing(r.Context(), id)
	if err != nil {
		slog.Error("failed to get listing", "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if listing == nil {
		respondError(w, http.StatusNotFound, "listing not found")
		return
	}

	respondJSON(w, http.StatusOK, listing)
}

// LinkListingRequest represents the request body for linking a listing to a template.
type LinkListingRequest struct {
	TemplateID    string `json:"template_id"`
	SKU           string `json:"sku"`
	SyncInventory bool   `json:"sync_inventory"`
}

// LinkListing links a listing to a template.
// POST /api/integrations/etsy/listings/{id}/link
func (h *EtsyHandler) LinkListing(w http.ResponseWriter, r *http.Request) {
	listingID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid listing ID")
		return
	}

	var req LinkListingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	templateID, err := parseUUIDString(req.TemplateID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid template ID")
		return
	}

	if err := h.service.LinkListingToTemplate(r.Context(), listingID, templateID, req.SKU, req.SyncInventory); err != nil {
		slog.Error("failed to link listing", "error", err)
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "linked"})
}

// UnlinkListing removes a link between a listing and a template.
// DELETE /api/integrations/etsy/listings/{id}/link
func (h *EtsyHandler) UnlinkListing(w http.ResponseWriter, r *http.Request) {
	listingID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid listing ID")
		return
	}

	templateIDStr := r.URL.Query().Get("template_id")
	if templateIDStr == "" {
		respondError(w, http.StatusBadRequest, "template_id is required")
		return
	}

	templateID, err := parseUUIDString(templateIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid template ID")
		return
	}

	if err := h.service.UnlinkListingFromTemplate(r.Context(), listingID, templateID); err != nil {
		slog.Error("failed to unlink listing", "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SyncInventory pushes local inventory to Etsy.
// POST /api/integrations/etsy/listings/{id}/sync-inventory
func (h *EtsyHandler) SyncInventory(w http.ResponseWriter, r *http.Request) {
	listingID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid listing ID")
		return
	}

	if err := h.service.SyncInventoryToEtsy(r.Context(), listingID); err != nil {
		slog.Error("failed to sync inventory", "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "synced"})
}

// ---- Webhook Handlers ----

// HandleWebhook processes incoming Etsy webhook events.
// POST /api/integrations/etsy/webhook
func (h *EtsyHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("failed to read webhook body", "error", err)
		respondError(w, http.StatusBadRequest, "failed to read body")
		return
	}

	// Verify signature if secret is configured
	signature := r.Header.Get("X-Etsy-Signature")
	if h.webhookSecret != "" && signature != "" {
		if !verifyWebhookSignature(body, signature, h.webhookSecret) {
			slog.Warn("webhook signature verification failed")
			respondError(w, http.StatusUnauthorized, "invalid signature")
			return
		}
	}

	// Parse event
	var payload struct {
		Type       string          `json:"type"`
		ResourceType string        `json:"resource_type"`
		ResourceID int64           `json:"resource_id"`
		ShopID     int64           `json:"shop_id"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		slog.Error("failed to parse webhook payload", "error", err)
		respondError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	// Create event record
	event := &model.EtsyWebhookEvent{
		EventType:    payload.Type,
		ResourceType: payload.ResourceType,
		ResourceID:   payload.ResourceID,
		ShopID:       payload.ShopID,
		Payload:      body,
		Signature:    signature,
	}

	// Save event immediately for audit
	if err := h.service.SaveWebhookEvent(r.Context(), event); err != nil {
		slog.Error("failed to save webhook event", "error", err)
	}

	// Respond 200 OK immediately (Etsy expects response within 3 seconds)
	w.WriteHeader(http.StatusOK)

	// Process event asynchronously
	go func() {
		ctx := r.Context()
		var processErr error

		switch event.EventType {
		case model.EtsyEventReceiptCreated, model.EtsyEventReceiptUpdated:
			processErr = h.service.HandleReceiptCreated(ctx, event)
		case model.EtsyEventListingUpdated:
			processErr = h.service.HandleListingUpdated(ctx, event)
		case model.EtsyEventListingInventoryUpdated:
			processErr = h.service.HandleInventoryUpdated(ctx, event)
		default:
			slog.Warn("unknown webhook event type", "type", event.EventType)
		}

		if processErr != nil {
			slog.Error("failed to process webhook event", "event_id", event.ID, "error", processErr)
		}
	}()
}

// ListWebhookEvents returns webhook event history.
// GET /api/integrations/etsy/webhook/events
func (h *EtsyHandler) ListWebhookEvents(w http.ResponseWriter, r *http.Request) {
	eventType := r.URL.Query().Get("type")

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

	events, err := h.service.ListWebhookEvents(r.Context(), eventType, limit, offset)
	if err != nil {
		slog.Error("failed to list webhook events", "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if events == nil {
		events = []model.EtsyWebhookEvent{}
	}

	respondJSON(w, http.StatusOK, events)
}

// ReprocessWebhookEvent retries processing a failed webhook event.
// POST /api/integrations/etsy/webhook/events/{id}/reprocess
func (h *EtsyHandler) ReprocessWebhookEvent(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid event ID")
		return
	}

	if err := h.service.ReprocessWebhookEvent(r.Context(), id); err != nil {
		slog.Error("failed to reprocess webhook event", "error", err)
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "reprocessed"})
}

// verifyWebhookSignature verifies an HMAC-SHA256 signature.
func verifyWebhookSignature(body []byte, signature, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// parseUUIDString parses a UUID from a string.
func parseUUIDString(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}
