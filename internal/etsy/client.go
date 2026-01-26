package etsy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// OAuth URLs
const (
	EtsyAuthURL  = "https://www.etsy.com/oauth/connect"
	EtsyTokenURL = "https://api.etsy.com/v3/public/oauth/token"
	EtsyAPIBase  = "https://api.etsy.com/v3"
)

// DefaultScopes are the required scopes for order management.
var DefaultScopes = []string{
	"listings_r",     // Read listings
	"transactions_r", // Read orders/receipts
	"profile_r",      // Read shop info
}

// Client handles Etsy API communication.
type Client struct {
	clientID    string
	redirectURI string
	httpClient  *http.Client
}

// NewClient creates a new Etsy API client.
func NewClient(clientID, redirectURI string) *Client {
	return &Client{
		clientID:    clientID,
		redirectURI: redirectURI,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// TokenResponse represents the OAuth token response from Etsy.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

// Shop represents an Etsy shop.
type Shop struct {
	ShopID   int64  `json:"shop_id"`
	ShopName string `json:"shop_name"`
	UserID   int64  `json:"user_id"`
}

// GenerateAuthURL creates the Etsy authorization URL.
func (c *Client) GenerateAuthURL(state, codeChallenge string) string {
	params := url.Values{
		"response_type":         {"code"},
		"client_id":             {c.clientID},
		"redirect_uri":          {c.redirectURI},
		"scope":                 {strings.Join(DefaultScopes, " ")},
		"state":                 {state},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
	}
	return EtsyAuthURL + "?" + params.Encode()
}

// ExchangeCode exchanges an authorization code for tokens.
func (c *Client) ExchangeCode(ctx context.Context, code, codeVerifier string) (*TokenResponse, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {c.clientID},
		"redirect_uri":  {c.redirectURI},
		"code":          {code},
		"code_verifier": {codeVerifier},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", EtsyTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed: %s - %s", resp.Status, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &tokenResp, nil
}

// RefreshToken refreshes an expired access token.
func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {c.clientID},
		"refresh_token": {refreshToken},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", EtsyTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed: %s - %s", resp.Status, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &tokenResp, nil
}

// GetShop retrieves the shop information for the authenticated user.
// The access token has the format {user_id}.{token}, so we extract the user_id.
func (c *Client) GetShop(ctx context.Context, accessToken string) (*Shop, error) {
	// Extract user_id from token (format: {user_id}.{token})
	parts := strings.SplitN(accessToken, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid access token format")
	}
	userID := parts[0]

	// Get the user's shop
	url := fmt.Sprintf("%s/application/users/%s/shops", EtsyAPIBase, userID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("x-api-key", c.clientID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get shop failed: %s - %s", resp.Status, string(body))
	}

	// Response contains array of shops, user typically has one
	var shopsResp struct {
		Results []struct {
			ShopID   int64  `json:"shop_id"`
			ShopName string `json:"shop_name"`
			UserID   int64  `json:"user_id"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &shopsResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	if len(shopsResp.Results) == 0 {
		return nil, fmt.Errorf("no shop found for user")
	}

	shop := shopsResp.Results[0]
	return &Shop{
		ShopID:   shop.ShopID,
		ShopName: shop.ShopName,
		UserID:   shop.UserID,
	}, nil
}

// EtsyMoney represents Etsy's money format.
type EtsyMoney struct {
	Amount       int    `json:"amount"`
	Divisor      int    `json:"divisor"`
	CurrencyCode string `json:"currency_code"`
}

// MoneyToCents converts Etsy money to cents.
func MoneyToCents(m EtsyMoney) int {
	if m.Divisor == 0 {
		return m.Amount
	}
	// Convert to cents: amount / divisor * 100
	return m.Amount * 100 / m.Divisor
}

// ReceiptQueryOptions contains options for querying receipts.
type ReceiptQueryOptions struct {
	MinCreated int64  // Unix timestamp for minimum creation date
	MaxCreated int64  // Unix timestamp for maximum creation date
	WasPaid    *bool  // Filter by payment status
	WasShipped *bool  // Filter by shipping status
	Limit      int    // Max results per page (default 25, max 100)
	Offset     int    // Offset for pagination
}

// ToURLParams converts options to URL parameters.
func (o ReceiptQueryOptions) ToURLParams() url.Values {
	params := url.Values{}
	if o.MinCreated > 0 {
		params.Set("min_created", fmt.Sprintf("%d", o.MinCreated))
	}
	if o.MaxCreated > 0 {
		params.Set("max_created", fmt.Sprintf("%d", o.MaxCreated))
	}
	if o.WasPaid != nil {
		params.Set("was_paid", fmt.Sprintf("%t", *o.WasPaid))
	}
	if o.WasShipped != nil {
		params.Set("was_shipped", fmt.Sprintf("%t", *o.WasShipped))
	}
	if o.Limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", o.Limit))
	}
	if o.Offset > 0 {
		params.Set("offset", fmt.Sprintf("%d", o.Offset))
	}
	return params
}

// APIReceipt represents a receipt from the Etsy API.
type APIReceipt struct {
	ReceiptID            int64       `json:"receipt_id"`
	ReceiptType          int         `json:"receipt_type"`
	SellerUserID         int64       `json:"seller_user_id"`
	SellerEmail          string      `json:"seller_email"`
	BuyerUserID          int64       `json:"buyer_user_id"`
	BuyerEmail           string      `json:"buyer_email"`
	Name                 string      `json:"name"`
	FirstLine            string      `json:"first_line"`
	SecondLine           string      `json:"second_line"`
	City                 string      `json:"city"`
	State                string      `json:"state"`
	Zip                  string      `json:"zip"`
	Status               string      `json:"status"`
	FormattedAddress     string      `json:"formatted_address"`
	CountryISO           string      `json:"country_iso"`
	PaymentMethod        string      `json:"payment_method"`
	PaymentEmail         string      `json:"payment_email"`
	MessageFromSeller    string      `json:"message_from_seller"`
	MessageFromBuyer     string      `json:"message_from_buyer"`
	MessageFromPayment   string      `json:"message_from_payment"`
	IsPaid               bool        `json:"is_paid"`
	IsShipped            bool        `json:"is_shipped"`
	CreateTimestamp      int64       `json:"create_timestamp"`
	CreatedTimestamp     int64       `json:"created_timestamp"`
	UpdateTimestamp      int64       `json:"update_timestamp"`
	UpdatedTimestamp     int64       `json:"updated_timestamp"`
	IsGift               bool        `json:"is_gift"`
	GiftMessage          string      `json:"gift_message"`
	Grandtotal           EtsyMoney   `json:"grandtotal"`
	Subtotal             EtsyMoney   `json:"subtotal"`
	TotalPrice           EtsyMoney   `json:"total_price"`
	TotalShippingCost    EtsyMoney   `json:"total_shipping_cost"`
	TotalTaxCost         EtsyMoney   `json:"total_tax_cost"`
	TotalVatCost         EtsyMoney   `json:"total_vat_cost"`
	DiscountAmt          EtsyMoney   `json:"discount_amt"`
	GiftWrapPrice        EtsyMoney   `json:"gift_wrap_price"`
	Transactions         []APITransaction `json:"transactions"`
}

// APITransaction represents a transaction/line item from the Etsy API.
type APITransaction struct {
	TransactionID     int64     `json:"transaction_id"`
	Title             string    `json:"title"`
	Description       string    `json:"description"`
	SellerUserID      int64     `json:"seller_user_id"`
	BuyerUserID       int64     `json:"buyer_user_id"`
	CreateTimestamp   int64     `json:"create_timestamp"`
	CreatedTimestamp  int64     `json:"created_timestamp"`
	PaidTimestamp     int64     `json:"paid_timestamp"`
	ShippedTimestamp  int64     `json:"shipped_timestamp"`
	Quantity          int       `json:"quantity"`
	ListingImageID    int64     `json:"listing_image_id"`
	ReceiptID         int64     `json:"receipt_id"`
	IsDigital         bool      `json:"is_digital"`
	FileData          string    `json:"file_data"`
	ListingID         int64     `json:"listing_id"`
	SKU               string    `json:"sku"`
	ProductID         int64     `json:"product_id"`
	TransactionType   string    `json:"transaction_type"`
	Price             EtsyMoney `json:"price"`
	ShippingCost      EtsyMoney `json:"shipping_cost"`
	Variations        []APIVariation `json:"variations"`
}

// APIVariation represents a variation selection.
type APIVariation struct {
	PropertyID   int64  `json:"property_id"`
	ValueID      int64  `json:"value_id,omitempty"`
	FormattedName  string `json:"formatted_name"`
	FormattedValue string `json:"formatted_value"`
}

// GetReceipts fetches shop receipts from Etsy.
func (c *Client) GetReceipts(ctx context.Context, accessToken string, shopID int64, opts ReceiptQueryOptions) ([]APIReceipt, error) {
	params := opts.ToURLParams()
	params.Set("includes", "Transactions")

	reqURL := fmt.Sprintf("%s/application/shops/%d/receipts?%s", EtsyAPIBase, shopID, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("x-api-key", c.clientID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get receipts failed: %s - %s", resp.Status, string(body))
	}

	var result struct {
		Count   int          `json:"count"`
		Results []APIReceipt `json:"results"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return result.Results, nil
}

// GetReceipt fetches a single receipt by ID.
func (c *Client) GetReceipt(ctx context.Context, accessToken string, shopID, receiptID int64) (*APIReceipt, error) {
	reqURL := fmt.Sprintf("%s/application/shops/%d/receipts/%d?includes=Transactions", EtsyAPIBase, shopID, receiptID)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("x-api-key", c.clientID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get receipt failed: %s - %s", resp.Status, string(body))
	}

	var receipt APIReceipt
	if err := json.Unmarshal(body, &receipt); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &receipt, nil
}

// ListingQueryOptions contains options for querying listings.
type ListingQueryOptions struct {
	State  string // active, inactive, draft, expired, sold_out
	Limit  int
	Offset int
}

// ToURLParams converts options to URL parameters.
func (o ListingQueryOptions) ToURLParams() url.Values {
	params := url.Values{}
	if o.State != "" {
		params.Set("state", o.State)
	}
	if o.Limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", o.Limit))
	}
	if o.Offset > 0 {
		params.Set("offset", fmt.Sprintf("%d", o.Offset))
	}
	return params
}

// APIListing represents a listing from the Etsy API.
type APIListing struct {
	ListingID        int64     `json:"listing_id"`
	UserID           int64     `json:"user_id"`
	ShopID           int64     `json:"shop_id"`
	Title            string    `json:"title"`
	Description      string    `json:"description"`
	State            string    `json:"state"`
	CreationTimestamp int64    `json:"creation_timestamp"`
	CreatedTimestamp int64     `json:"created_timestamp"`
	EndingTimestamp  int64     `json:"ending_timestamp"`
	OriginalCreationTimestamp int64 `json:"original_creation_timestamp"`
	LastModifiedTimestamp int64 `json:"last_modified_timestamp"`
	UpdatedTimestamp int64     `json:"updated_timestamp"`
	StateTimestamp   int64     `json:"state_timestamp"`
	Quantity         int       `json:"quantity"`
	ShopSectionID    int64     `json:"shop_section_id"`
	FeaturedRank     int       `json:"featured_rank"`
	URL              string    `json:"url"`
	NumFavorers      int       `json:"num_favorers"`
	NonTaxable       bool      `json:"non_taxable"`
	IsTaxable        bool      `json:"is_taxable"`
	IsCustomizable   bool      `json:"is_customizable"`
	IsPersonalizable bool      `json:"is_personalizable"`
	PersonalizationIsRequired bool `json:"personalization_is_required"`
	PersonalizationCharCountMax int `json:"personalization_char_count_max"`
	PersonalizationInstructions string `json:"personalization_instructions"`
	ListingType      string    `json:"listing_type"`
	Tags             []string  `json:"tags"`
	Materials        []string  `json:"materials"`
	ShippingProfileID int64    `json:"shipping_profile_id"`
	ReturnPolicyID   int64     `json:"return_policy_id"`
	ProcessingMin    int       `json:"processing_min"`
	ProcessingMax    int       `json:"processing_max"`
	WhoMade          string    `json:"who_made"`
	WhenMade         string    `json:"when_made"`
	IsSupply         bool      `json:"is_supply"`
	ItemWeight       float64   `json:"item_weight"`
	ItemWeightUnit   string    `json:"item_weight_unit"`
	ItemLength       float64   `json:"item_length"`
	ItemWidth        float64   `json:"item_width"`
	ItemHeight       float64   `json:"item_height"`
	ItemDimensionsUnit string  `json:"item_dimensions_unit"`
	IsPrivate        bool      `json:"is_private"`
	Style            []string  `json:"style"`
	FileData         string    `json:"file_data"`
	HasVariations    bool      `json:"has_variations"`
	ShouldAutoRenew  bool      `json:"should_auto_renew"`
	Language         string    `json:"language"`
	Price            EtsyMoney `json:"price"`
	TaxonomyID       int64     `json:"taxonomy_id"`
	SKUs             []string  `json:"skus"`
	Views            int       `json:"views"`
}

// GetActiveListings fetches active listings for a shop.
func (c *Client) GetActiveListings(ctx context.Context, accessToken string, shopID int64, opts ListingQueryOptions) ([]APIListing, error) {
	if opts.State == "" {
		opts.State = "active"
	}
	params := opts.ToURLParams()

	reqURL := fmt.Sprintf("%s/application/shops/%d/listings/%s?%s", EtsyAPIBase, shopID, opts.State, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("x-api-key", c.clientID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get listings failed: %s - %s", resp.Status, string(body))
	}

	var result struct {
		Count   int          `json:"count"`
		Results []APIListing `json:"results"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return result.Results, nil
}

// GetListing fetches a single listing by ID.
func (c *Client) GetListing(ctx context.Context, accessToken string, listingID int64) (*APIListing, error) {
	reqURL := fmt.Sprintf("%s/application/listings/%d", EtsyAPIBase, listingID)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("x-api-key", c.clientID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get listing failed: %s - %s", resp.Status, string(body))
	}

	var listing APIListing
	if err := json.Unmarshal(body, &listing); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &listing, nil
}

// APIInventoryProduct represents a product variant in inventory.
type APIInventoryProduct struct {
	ProductID        int64                `json:"product_id"`
	SKU              string               `json:"sku"`
	IsDeleted        bool                 `json:"is_deleted"`
	Offerings        []APIInventoryOffering `json:"offerings"`
	PropertyValues   []APIPropertyValue   `json:"property_values"`
}

// APIInventoryOffering represents an offering for a product.
type APIInventoryOffering struct {
	OfferingID int64     `json:"offering_id"`
	Quantity   int       `json:"quantity"`
	IsEnabled  bool      `json:"is_enabled"`
	IsDeleted  bool      `json:"is_deleted"`
	Price      EtsyMoney `json:"price"`
}

// APIPropertyValue represents a property value for a product.
type APIPropertyValue struct {
	PropertyID   int64    `json:"property_id"`
	PropertyName string   `json:"property_name"`
	ScaleID      int64    `json:"scale_id"`
	ScaleName    string   `json:"scale_name"`
	ValueIDs     []int64  `json:"value_ids"`
	Values       []string `json:"values"`
}

// APIInventory represents the full inventory for a listing.
type APIInventory struct {
	Products    []APIInventoryProduct `json:"products"`
	PriceOnProperty []int64           `json:"price_on_property"`
	QuantityOnProperty []int64        `json:"quantity_on_property"`
	SKUOnProperty []int64             `json:"sku_on_property"`
}

// GetListingInventory fetches inventory for a listing.
func (c *Client) GetListingInventory(ctx context.Context, accessToken string, listingID int64) (*APIInventory, error) {
	reqURL := fmt.Sprintf("%s/application/listings/%d/inventory", EtsyAPIBase, listingID)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("x-api-key", c.clientID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get inventory failed: %s - %s", resp.Status, string(body))
	}

	var inventory APIInventory
	if err := json.Unmarshal(body, &inventory); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &inventory, nil
}

// UpdateInventoryProduct represents a product to update in inventory.
type UpdateInventoryProduct struct {
	SKU            string                    `json:"sku"`
	PropertyValues []APIPropertyValue        `json:"property_values"`
	Offerings      []UpdateInventoryOffering `json:"offerings"`
}

// UpdateInventoryOffering represents an offering to update.
type UpdateInventoryOffering struct {
	OfferingID int64 `json:"offering_id,omitempty"`
	Price      int   `json:"price"` // In currency's smallest unit (cents)
	Quantity   int   `json:"quantity"`
	IsEnabled  bool  `json:"is_enabled"`
}

// UpdateListingInventory updates inventory for a listing.
func (c *Client) UpdateListingInventory(ctx context.Context, accessToken string, listingID int64, products []UpdateInventoryProduct) error {
	reqURL := fmt.Sprintf("%s/application/listings/%d/inventory", EtsyAPIBase, listingID)

	payload := map[string]interface{}{
		"products": products,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", reqURL, strings.NewReader(string(payloadBytes)))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("x-api-key", c.clientID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("update inventory failed: %s - %s", resp.Status, string(body))
	}

	return nil
}
