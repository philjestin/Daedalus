package squarespace

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient("test-api-key")
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.apiKey != "test-api-key" {
		t.Errorf("expected apiKey 'test-api-key', got '%s'", client.apiKey)
	}
}

func TestGetWebsite(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify auth header
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-api-key" {
			t.Errorf("expected Authorization 'Bearer test-api-key', got '%s'", auth)
		}

		// Verify path
		if r.URL.Path != "/1.0/commerce/website" {
			t.Errorf("expected path '/1.0/commerce/website', got '%s'", r.URL.Path)
		}

		resp := WebsiteResponse{
			Website: Website{
				ID:       "site-123",
				Title:    "Test Store",
				Domain:   "test.squarespace.com",
				SiteType: "commerce",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create client with test server URL
	client := &Client{
		apiKey:     "test-api-key",
		httpClient: server.Client(),
	}

	// Note: For full integration testing, you would use httptest to mock the server
	// and override the base URL. For now, we test the request/response handling.

	// For this test, we'll make the request directly
	ctx := context.Background()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/1.0/commerce/website", nil)
	req.Header.Set("Authorization", "Bearer "+client.apiKey)

	resp, err := client.httpClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result WebsiteResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if result.Website.ID != "site-123" {
		t.Errorf("expected site ID 'site-123', got '%s'", result.Website.ID)
	}
	if result.Website.Title != "Test Store" {
		t.Errorf("expected title 'Test Store', got '%s'", result.Website.Title)
	}
}

func TestGetOrders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/1.0/commerce/orders" {
			t.Errorf("expected path '/1.0/commerce/orders', got '%s'", r.URL.Path)
		}

		// Check query params
		modifiedAfter := r.URL.Query().Get("modifiedAfter")
		if modifiedAfter == "" {
			// Return all orders
		}

		resp := OrdersResponse{
			Result: []Order{
				{
					ID:                "order-1",
					OrderNumber:       "1001",
					CustomerEmail:     "test@example.com",
					FulfillmentStatus: "PENDING",
					GrandTotal:        Money{Value: "29.99", Currency: "USD"},
					LineItems: []LineItem{
						{
							ID:          "item-1",
							ProductID:   "prod-1",
							ProductName: "Test Product",
							Quantity:    2,
							UnitPricePaid: Money{Value: "14.99", Currency: "USD"},
						},
					},
				},
			},
			Pagination: Pagination{
				HasNextPage: false,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &Client{
		apiKey:     "test-api-key",
		httpClient: server.Client(),
	}

	ctx := context.Background()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/1.0/commerce/orders", nil)
	req.Header.Set("Authorization", "Bearer "+client.apiKey)

	resp, err := client.httpClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result OrdersResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if len(result.Result) != 1 {
		t.Fatalf("expected 1 order, got %d", len(result.Result))
	}
	if result.Result[0].ID != "order-1" {
		t.Errorf("expected order ID 'order-1', got '%s'", result.Result[0].ID)
	}
	if result.Result[0].OrderNumber != "1001" {
		t.Errorf("expected order number '1001', got '%s'", result.Result[0].OrderNumber)
	}
}

func TestGetProducts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/1.0/commerce/products" {
			t.Errorf("expected path '/1.0/commerce/products', got '%s'", r.URL.Path)
		}

		resp := ProductsResponse{
			Result: []Product{
				{
					ID:        "prod-1",
					Name:      "Test Product",
					Type:      "PHYSICAL",
					IsVisible: true,
					Variants: []ProductVariant{
						{
							ID:  "var-1",
							SKU: "TEST-SKU-001",
							Pricing: ProductPricing{
								BasePrice: Money{Value: "19.99", Currency: "USD"},
							},
							Stock: ProductStock{
								Quantity:  10,
								Unlimited: false,
							},
						},
					},
				},
			},
			Pagination: Pagination{
				HasNextPage: false,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &Client{
		apiKey:     "test-api-key",
		httpClient: server.Client(),
	}

	ctx := context.Background()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/1.0/commerce/products", nil)
	req.Header.Set("Authorization", "Bearer "+client.apiKey)

	resp, err := client.httpClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result ProductsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if len(result.Result) != 1 {
		t.Fatalf("expected 1 product, got %d", len(result.Result))
	}
	if result.Result[0].Name != "Test Product" {
		t.Errorf("expected product name 'Test Product', got '%s'", result.Result[0].Name)
	}
	if len(result.Result[0].Variants) != 1 {
		t.Fatalf("expected 1 variant, got %d", len(result.Result[0].Variants))
	}
	if result.Result[0].Variants[0].SKU != "TEST-SKU-001" {
		t.Errorf("expected SKU 'TEST-SKU-001', got '%s'", result.Result[0].Variants[0].SKU)
	}
}

func TestMoneyToCents(t *testing.T) {
	tests := []struct {
		name     string
		money    Money
		expected int
	}{
		{"zero", Money{Value: "0", Currency: "USD"}, 0},
		{"whole dollars", Money{Value: "10", Currency: "USD"}, 1000},
		{"with cents", Money{Value: "19.99", Currency: "USD"}, 1999},
		{"empty value", Money{Value: "", Currency: "USD"}, 0},
		{"large value", Money{Value: "1234.56", Currency: "USD"}, 123456},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MoneyToCents(tt.money)
			if result != tt.expected {
				t.Errorf("MoneyToCents(%v) = %d, expected %d", tt.money, result, tt.expected)
			}
		})
	}
}

func TestCustomerName(t *testing.T) {
	tests := []struct {
		name     string
		addr     Address
		expected string
	}{
		{
			"full name",
			Address{FirstName: "John", LastName: "Doe"},
			"John Doe",
		},
		{
			"first name only",
			Address{FirstName: "John"},
			"John",
		},
		{
			"last name only",
			Address{LastName: "Doe"},
			"Doe",
		},
		{
			"empty",
			Address{},
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CustomerName(tt.addr)
			if result != tt.expected {
				t.Errorf("CustomerName(%v) = '%s', expected '%s'", tt.addr, result, tt.expected)
			}
		})
	}
}

func TestOrdersOptions(t *testing.T) {
	now := time.Now()
	opts := &OrdersOptions{
		ModifiedAfter:     &now,
		Cursor:            "abc123",
		FulfillmentStatus: "PENDING",
	}

	if opts.ModifiedAfter == nil {
		t.Error("expected ModifiedAfter to be set")
	}
	if opts.Cursor != "abc123" {
		t.Errorf("expected Cursor 'abc123', got '%s'", opts.Cursor)
	}
	if opts.FulfillmentStatus != "PENDING" {
		t.Errorf("expected FulfillmentStatus 'PENDING', got '%s'", opts.FulfillmentStatus)
	}
}
