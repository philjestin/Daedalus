package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EtsyIntegration represents the connected Etsy shop.
type EtsyIntegration struct {
	ID             uuid.UUID  `json:"id"`
	ShopID         int64      `json:"shop_id"`
	ShopName       string     `json:"shop_name"`
	UserID         int64      `json:"user_id"`
	AccessToken    string     `json:"-"` // Never expose
	RefreshToken   string     `json:"-"` // Never expose
	TokenExpiresAt time.Time  `json:"token_expires_at"`
	Scopes         []string   `json:"scopes"`
	IsActive       bool       `json:"is_active"`
	LastSyncAt     *time.Time `json:"last_sync_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// EtsyOAuthState stores pending OAuth state for PKCE verification.
type EtsyOAuthState struct {
	State        string
	CodeVerifier string
	RedirectURI  string
	CreatedAt    time.Time
}

// EtsyReceipt represents an order/receipt from Etsy.
type EtsyReceipt struct {
	ID                      uuid.UUID          `json:"id"`
	EtsyReceiptID           int64              `json:"etsy_receipt_id"`
	EtsyShopID              int64              `json:"etsy_shop_id"`
	BuyerUserID             int64              `json:"buyer_user_id,omitempty"`
	BuyerEmail              string             `json:"buyer_email,omitempty"`
	Name                    string             `json:"name"`
	Status                  string             `json:"status"`
	MessageFromBuyer        string             `json:"message_from_buyer,omitempty"`
	IsShipped               bool               `json:"is_shipped"`
	IsPaid                  bool               `json:"is_paid"`
	IsGift                  bool               `json:"is_gift"`
	GiftMessage             string             `json:"gift_message,omitempty"`
	GrandtotalCents         int                `json:"grandtotal_cents"`
	SubtotalCents           int                `json:"subtotal_cents"`
	TotalPriceCents         int                `json:"total_price_cents"`
	TotalShippingCostCents  int                `json:"total_shipping_cost_cents"`
	TotalTaxCostCents       int                `json:"total_tax_cost_cents"`
	DiscountCents           int                `json:"discount_cents"`
	Currency                string             `json:"currency"`
	ShippingName            string             `json:"shipping_name,omitempty"`
	ShippingAddressFirstLine  string           `json:"shipping_address_first_line,omitempty"`
	ShippingAddressSecondLine string           `json:"shipping_address_second_line,omitempty"`
	ShippingCity            string             `json:"shipping_city,omitempty"`
	ShippingState           string             `json:"shipping_state,omitempty"`
	ShippingZip             string             `json:"shipping_zip,omitempty"`
	ShippingCountryCode     string             `json:"shipping_country_code,omitempty"`
	CreateTimestamp         *time.Time         `json:"create_timestamp,omitempty"`
	UpdateTimestamp         *time.Time         `json:"update_timestamp,omitempty"`
	IsProcessed             bool               `json:"is_processed"`
	ProjectID               *uuid.UUID         `json:"project_id,omitempty"`
	SyncedAt                time.Time          `json:"synced_at"`
	CreatedAt               time.Time          `json:"created_at"`
	UpdatedAt               time.Time          `json:"updated_at"`
	Items                   []EtsyReceiptItem  `json:"items,omitempty"`
}

// EtsyReceiptItem represents a line item in an Etsy receipt.
type EtsyReceiptItem struct {
	ID                 uuid.UUID       `json:"id"`
	EtsyReceiptItemID  int64           `json:"etsy_receipt_item_id"`
	ReceiptID          uuid.UUID       `json:"receipt_id"`
	EtsyListingID      int64           `json:"etsy_listing_id"`
	EtsyTransactionID  int64           `json:"etsy_transaction_id"`
	Title              string          `json:"title"`
	Description        string          `json:"description,omitempty"`
	Quantity           int             `json:"quantity"`
	PriceCents         int             `json:"price_cents"`
	ShippingCostCents  int             `json:"shipping_cost_cents"`
	SKU                string          `json:"sku,omitempty"`
	Variations         json.RawMessage `json:"variations,omitempty"`
	IsDigital          bool            `json:"is_digital"`
	TemplateID         *uuid.UUID      `json:"template_id,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
}

// EtsySyncState tracks the last sync timestamps for a shop.
type EtsySyncState struct {
	ID                uuid.UUID  `json:"id"`
	ShopID            int64      `json:"shop_id"`
	LastReceiptSyncAt *time.Time `json:"last_receipt_sync_at,omitempty"`
	LastListingSyncAt *time.Time `json:"last_listing_sync_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// EtsyListing represents a listing from Etsy.
type EtsyListing struct {
	ID               uuid.UUID       `json:"id"`
	EtsyListingID    int64           `json:"etsy_listing_id"`
	EtsyShopID       int64           `json:"etsy_shop_id"`
	Title            string          `json:"title"`
	Description      string          `json:"description,omitempty"`
	State            string          `json:"state"`
	Quantity         int             `json:"quantity"`
	URL              string          `json:"url,omitempty"`
	Views            int             `json:"views"`
	NumFavorers      int             `json:"num_favorers"`
	IsCustomizable   bool            `json:"is_customizable"`
	IsPersonalizable bool            `json:"is_personalizable"`
	Tags             json.RawMessage `json:"tags,omitempty"`
	HasVariations    bool            `json:"has_variations"`
	PriceCents       int             `json:"price_cents,omitempty"`
	Currency         string          `json:"currency"`
	SKUs             json.RawMessage `json:"skus,omitempty"`
	SyncedAt         time.Time       `json:"synced_at"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	LinkedTemplate   *Template       `json:"linked_template,omitempty"`
}

// EtsyListingTemplate links an Etsy listing to a template.
type EtsyListingTemplate struct {
	ID            uuid.UUID  `json:"id"`
	EtsyListingID int64      `json:"etsy_listing_id"`
	TemplateID    uuid.UUID  `json:"template_id"`
	SKU           string     `json:"sku,omitempty"`
	SyncInventory bool       `json:"sync_inventory"`
	CreatedAt     time.Time  `json:"created_at"`
}

// EtsyWebhookEvent represents a webhook event from Etsy.
type EtsyWebhookEvent struct {
	ID           uuid.UUID       `json:"id"`
	EventType    string          `json:"event_type"`
	ResourceType string          `json:"resource_type"`
	ResourceID   int64           `json:"resource_id,omitempty"`
	ShopID       int64           `json:"shop_id,omitempty"`
	Payload      json.RawMessage `json:"payload"`
	Signature    string          `json:"signature,omitempty"`
	Processed    bool            `json:"processed"`
	ProcessedAt  *time.Time      `json:"processed_at,omitempty"`
	Error        string          `json:"error,omitempty"`
	ReceivedAt   time.Time       `json:"received_at"`
	CreatedAt    time.Time       `json:"created_at"`
}

// EtsyWebhookEventType constants for webhook events.
const (
	EtsyEventReceiptCreated         = "receipt.created"
	EtsyEventReceiptUpdated         = "receipt.updated"
	EtsyEventListingUpdated         = "listing.updated"
	EtsyEventListingInventoryUpdated = "listing.inventory.updated"
)

// SyncResult represents the result of a sync operation.
type SyncResult struct {
	TotalFetched int `json:"total_fetched"`
	Created      int `json:"created"`
	Updated      int `json:"updated"`
	Skipped      int `json:"skipped"`
	Errors       int `json:"errors"`
}
