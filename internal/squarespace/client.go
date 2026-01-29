// Package squarespace provides a client for the Squarespace Commerce API.
package squarespace

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	// BaseURL is the base URL for the Squarespace API.
	BaseURL = "https://api.squarespace.com/1.0"
)

// Client is a Squarespace API client.
type Client struct {
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new Squarespace API client.
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// do performs an HTTP request with authentication.
func (c *Client) do(ctx context.Context, method, path string, query url.Values) (*http.Response, error) {
	u := BaseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, u, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Daedalus/1.0")

	return c.httpClient.Do(req)
}

// parseResponse reads and parses a JSON response.
func parseResponse[T any](resp *http.Response) (T, error) {
	var result T
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return result, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return result, fmt.Errorf("parsing response: %w", err)
	}

	return result, nil
}

// Pagination represents pagination info in API responses.
type Pagination struct {
	HasNextPage    bool   `json:"hasNextPage"`
	NextPageCursor string `json:"nextPageCursor"`
	NextPageURL    string `json:"nextPageUrl"`
}

// Website represents site info from the Squarespace API.
type Website struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Domain      string `json:"domain"`
	SiteType    string `json:"siteType"`
	Language    string `json:"language"`
	TimeZone    string `json:"timeZone"`
	CreatedOn   string `json:"createdOn"`
	ModifiedOn  string `json:"modifiedOn"`
}

// WebsiteResponse wraps the website response.
type WebsiteResponse struct {
	Website Website `json:"website"`
}

// GetWebsite retrieves site information. Used to validate the API key.
func (c *Client) GetWebsite(ctx context.Context) (*Website, error) {
	resp, err := c.do(ctx, http.MethodGet, "/commerce/website", nil)
	if err != nil {
		return nil, err
	}

	result, err := parseResponse[WebsiteResponse](resp)
	if err != nil {
		return nil, err
	}

	return &result.Website, nil
}

// Money represents a monetary value from Squarespace.
type Money struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

// MoneyToCents converts a Squarespace money value to cents.
func MoneyToCents(m Money) int {
	if m.Value == "" {
		return 0
	}
	// Parse the value as a float and convert to cents
	val, err := strconv.ParseFloat(m.Value, 64)
	if err != nil {
		return 0
	}
	return int(val * 100)
}

// Address represents an address in the Squarespace API.
type Address struct {
	FirstName   string `json:"firstName"`
	LastName    string `json:"lastName"`
	Address1    string `json:"address1"`
	Address2    string `json:"address2"`
	City        string `json:"city"`
	State       string `json:"state"`
	PostalCode  string `json:"postalCode"`
	CountryCode string `json:"countryCode"`
	Phone       string `json:"phone"`
}

// LineItem represents an order line item.
type LineItem struct {
	ID             string   `json:"id"`
	ProductID      string   `json:"productId"`
	ProductName    string   `json:"productName"`
	VariantID      string   `json:"variantId"`
	SKU            string   `json:"sku"`
	Quantity       int      `json:"quantity"`
	UnitPricePaid  Money    `json:"unitPricePaid"`
	ImageURL       string   `json:"imageUrl"`
	VariantOptions []string `json:"variantOptions"`
}

// Order represents an order from the Squarespace API.
type Order struct {
	ID                      string     `json:"id"`
	OrderNumber             string     `json:"orderNumber"`
	Channel                 string     `json:"channel"`
	CustomerEmail           string     `json:"customerEmail"`
	FulfillmentStatus       string     `json:"fulfillmentStatus"`
	BillingAddress          Address    `json:"billingAddress"`
	ShippingAddress         Address    `json:"shippingAddress"`
	LineItems               []LineItem `json:"lineItems"`
	SubtotalPrice           Money      `json:"subtotalPrice"`
	ShippingTotal           Money      `json:"shippingTotal"`
	TaxTotal                Money      `json:"taxTotal"`
	DiscountTotal           Money      `json:"discountTotal"`
	RefundedTotal           Money      `json:"refundedTotal"`
	GrandTotal              Money      `json:"grandTotal"`
	CreatedOn               string     `json:"createdOn"`
	ModifiedOn              string     `json:"modifiedOn"`
	TestMode                bool       `json:"testmode"`
}

// OrdersOptions represents options for fetching orders.
type OrdersOptions struct {
	ModifiedAfter     *time.Time
	ModifiedBefore    *time.Time
	Cursor            string
	FulfillmentStatus string
}

// OrdersResponse represents the response from the orders endpoint.
type OrdersResponse struct {
	Result     []Order    `json:"result"`
	Pagination Pagination `json:"pagination"`
}

// GetOrders retrieves orders with optional filtering.
func (c *Client) GetOrders(ctx context.Context, opts *OrdersOptions) (*OrdersResponse, error) {
	query := url.Values{}

	if opts != nil {
		if opts.ModifiedAfter != nil {
			query.Set("modifiedAfter", opts.ModifiedAfter.Format(time.RFC3339))
		}
		if opts.ModifiedBefore != nil {
			query.Set("modifiedBefore", opts.ModifiedBefore.Format(time.RFC3339))
		}
		if opts.Cursor != "" {
			query.Set("cursor", opts.Cursor)
		}
		if opts.FulfillmentStatus != "" {
			query.Set("fulfillmentStatus", opts.FulfillmentStatus)
		}
	}

	resp, err := c.do(ctx, http.MethodGet, "/commerce/orders", query)
	if err != nil {
		return nil, err
	}

	return parseResponse[*OrdersResponse](resp)
}

// GetOrder retrieves a single order by ID.
func (c *Client) GetOrder(ctx context.Context, orderID string) (*Order, error) {
	resp, err := c.do(ctx, http.MethodGet, "/commerce/orders/"+orderID, nil)
	if err != nil {
		return nil, err
	}

	return parseResponse[*Order](resp)
}

// ProductImage represents a product image.
type ProductImage struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	AltText  string `json:"altText"`
	OrderNum int    `json:"orderNum"`
}

// ProductVariant represents a variant of a product.
type ProductVariant struct {
	ID             string            `json:"id"`
	SKU            string            `json:"sku"`
	Pricing        ProductPricing    `json:"pricing"`
	Stock          ProductStock      `json:"stock"`
	Attributes     map[string]string `json:"attributes"`
	ShippingWeight ProductWeight     `json:"shippingWeight"`
}

// ProductPricing represents variant pricing.
type ProductPricing struct {
	BasePrice Money `json:"basePrice"`
	SalePrice Money `json:"salePrice"`
	OnSale    bool  `json:"onSale"`
}

// ProductStock represents stock info.
type ProductStock struct {
	Quantity  int  `json:"quantity"`
	Unlimited bool `json:"unlimited"`
}

// ProductWeight represents weight info.
type ProductWeight struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

// Product represents a product from the Squarespace API.
type Product struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	URL         string           `json:"url"`
	Type        string           `json:"type"`
	IsVisible   bool             `json:"isVisible"`
	Tags        []string         `json:"tags"`
	Images      []ProductImage   `json:"images"`
	Variants    []ProductVariant `json:"variants"`
	CreatedOn   string           `json:"createdOn"`
	ModifiedOn  string           `json:"modifiedOn"`
}

// ProductsOptions represents options for fetching products.
type ProductsOptions struct {
	ModifiedAfter  *time.Time
	ModifiedBefore *time.Time
	Cursor         string
	Type           string
}

// ProductsResponse represents the response from the products endpoint.
type ProductsResponse struct {
	Result     []Product  `json:"result"`
	Pagination Pagination `json:"pagination"`
}

// GetProducts retrieves products with optional filtering.
func (c *Client) GetProducts(ctx context.Context, opts *ProductsOptions) (*ProductsResponse, error) {
	query := url.Values{}

	if opts != nil {
		if opts.ModifiedAfter != nil {
			query.Set("modifiedAfter", opts.ModifiedAfter.Format(time.RFC3339))
		}
		if opts.ModifiedBefore != nil {
			query.Set("modifiedBefore", opts.ModifiedBefore.Format(time.RFC3339))
		}
		if opts.Cursor != "" {
			query.Set("cursor", opts.Cursor)
		}
		if opts.Type != "" {
			query.Set("type", opts.Type)
		}
	}

	resp, err := c.do(ctx, http.MethodGet, "/commerce/products", query)
	if err != nil {
		return nil, err
	}

	return parseResponse[*ProductsResponse](resp)
}

// GetProduct retrieves a single product by ID.
func (c *Client) GetProduct(ctx context.Context, productID string) (*Product, error) {
	resp, err := c.do(ctx, http.MethodGet, "/commerce/products/"+productID, nil)
	if err != nil {
		return nil, err
	}

	return parseResponse[*Product](resp)
}

// CustomerName builds a full customer name from an address.
func CustomerName(addr Address) string {
	name := strings.TrimSpace(addr.FirstName + " " + addr.LastName)
	return name
}
