package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// SquarespaceIntegration represents the connected Squarespace site.
type SquarespaceIntegration struct {
	ID                uuid.UUID  `json:"id"`
	SiteID            string     `json:"site_id"`
	SiteTitle         string     `json:"site_title"`
	APIKey            string     `json:"-"` // Never expose
	IsActive          bool       `json:"is_active"`
	LastOrderSyncAt   *time.Time `json:"last_order_sync_at,omitempty"`
	LastProductSyncAt *time.Time `json:"last_product_sync_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// SquarespaceAddress represents a billing or shipping address.
type SquarespaceAddress struct {
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Address1    string `json:"address1"`
	Address2    string `json:"address2"`
	City        string `json:"city"`
	State       string `json:"state"`
	PostalCode  string `json:"postal_code"`
	CountryCode string `json:"country_code"`
	Phone       string `json:"phone"`
}

// SquarespaceOrder represents an order from Squarespace.
type SquarespaceOrder struct {
	ID                 uuid.UUID                 `json:"id"`
	SquarespaceOrderID string                    `json:"squarespace_order_id"`
	OrderNumber        string                    `json:"order_number"`
	CustomerEmail      string                    `json:"customer_email"`
	CustomerName       string                    `json:"customer_name"`
	Channel            string                    `json:"channel"`
	SubtotalCents      int                       `json:"subtotal_cents"`
	ShippingCents      int                       `json:"shipping_cents"`
	TaxCents           int                       `json:"tax_cents"`
	DiscountCents      int                       `json:"discount_cents"`
	RefundedCents      int                       `json:"refunded_cents"`
	GrandTotalCents    int                       `json:"grand_total_cents"`
	Currency           string                    `json:"currency"`
	FulfillmentStatus  string                    `json:"fulfillment_status"`
	BillingAddress     *SquarespaceAddress       `json:"billing_address,omitempty"`
	ShippingAddress    *SquarespaceAddress       `json:"shipping_address,omitempty"`
	CreatedOn          *time.Time                `json:"created_on,omitempty"`
	ModifiedOn         *time.Time                `json:"modified_on,omitempty"`
	IsProcessed        bool                      `json:"is_processed"`
	ProjectID          *uuid.UUID                `json:"project_id,omitempty"`
	Items              []SquarespaceOrderItem    `json:"items,omitempty"`
	SyncedAt           time.Time                 `json:"synced_at"`
	CreatedAt          time.Time                 `json:"created_at"`
	UpdatedAt          time.Time                 `json:"updated_at"`
}

// SquarespaceOrderItem represents a line item in a Squarespace order.
type SquarespaceOrderItem struct {
	ID                uuid.UUID       `json:"id"`
	OrderID           uuid.UUID       `json:"order_id"`
	SquarespaceItemID string          `json:"squarespace_item_id"`
	ProductID         string          `json:"product_id"`
	VariantID         string          `json:"variant_id"`
	ProductName       string          `json:"product_name"`
	SKU               string          `json:"sku"`
	Quantity          int             `json:"quantity"`
	UnitPriceCents    int             `json:"unit_price_cents"`
	Currency          string          `json:"currency"`
	ImageURL          string          `json:"image_url"`
	VariantOptions    json.RawMessage `json:"variant_options,omitempty"`
	TemplateID        *uuid.UUID      `json:"template_id,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
}

// SquarespaceProduct represents a product from Squarespace.
type SquarespaceProduct struct {
	ID                   uuid.UUID                    `json:"id"`
	SquarespaceProductID string                       `json:"squarespace_product_id"`
	Name                 string                       `json:"name"`
	Description          string                       `json:"description"`
	URL                  string                       `json:"url"`
	Type                 string                       `json:"type"`
	IsVisible            bool                         `json:"is_visible"`
	Tags                 json.RawMessage              `json:"tags,omitempty"`
	Variants             []SquarespaceProductVariant  `json:"variants,omitempty"`
	CreatedOn            *time.Time                   `json:"created_on,omitempty"`
	ModifiedOn           *time.Time                   `json:"modified_on,omitempty"`
	SyncedAt             time.Time                    `json:"synced_at"`
	CreatedAt            time.Time                    `json:"created_at"`
	UpdatedAt            time.Time                    `json:"updated_at"`
}

// SquarespaceProductVariant represents a variant of a Squarespace product.
type SquarespaceProductVariant struct {
	ID                   uuid.UUID       `json:"id"`
	ProductID            uuid.UUID       `json:"product_id"`
	SquarespaceVariantID string          `json:"squarespace_variant_id"`
	SKU                  string          `json:"sku"`
	PriceCents           int             `json:"price_cents"`
	SalePriceCents       int             `json:"sale_price_cents"`
	OnSale               bool            `json:"on_sale"`
	StockQuantity        int             `json:"stock_quantity"`
	StockUnlimited       bool            `json:"stock_unlimited"`
	Attributes           json.RawMessage `json:"attributes,omitempty"`
	CreatedAt            time.Time       `json:"created_at"`
}

// SquarespaceProductTemplate links a Squarespace product to a template.
type SquarespaceProductTemplate struct {
	ID                   uuid.UUID `json:"id"`
	SquarespaceProductID string    `json:"squarespace_product_id"`
	TemplateID           uuid.UUID `json:"template_id"`
	SKU                  string    `json:"sku,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
}
