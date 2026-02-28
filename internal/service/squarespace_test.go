package service

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/philjestin/daedalus/internal/model"
	"github.com/philjestin/daedalus/internal/squarespace"
)

func TestConvertOrder(t *testing.T) {
	svc := &SquarespaceService{}

	apiOrder := squarespace.Order{
		ID:                "sq-order-123",
		OrderNumber:       "1001",
		CustomerEmail:     "test@example.com",
		FulfillmentStatus: "PENDING",
		Channel:           "web",
		BillingAddress: squarespace.Address{
			FirstName:   "John",
			LastName:    "Doe",
			Address1:    "123 Main St",
			City:        "New York",
			State:       "NY",
			PostalCode:  "10001",
			CountryCode: "US",
		},
		ShippingAddress: squarespace.Address{
			FirstName:   "John",
			LastName:    "Doe",
			Address1:    "456 Oak Ave",
			City:        "Boston",
			State:       "MA",
			PostalCode:  "02101",
			CountryCode: "US",
		},
		SubtotalPrice: squarespace.Money{Value: "19.99", Currency: "USD"},
		ShippingTotal: squarespace.Money{Value: "5.00", Currency: "USD"},
		TaxTotal:      squarespace.Money{Value: "2.00", Currency: "USD"},
		DiscountTotal: squarespace.Money{Value: "0", Currency: "USD"},
		GrandTotal:    squarespace.Money{Value: "26.99", Currency: "USD"},
		CreatedOn:     "2024-01-15T10:30:00Z",
		ModifiedOn:    "2024-01-15T11:00:00Z",
	}

	order := svc.convertOrder(apiOrder)

	if order.SquarespaceOrderID != "sq-order-123" {
		t.Errorf("expected SquarespaceOrderID 'sq-order-123', got '%s'", order.SquarespaceOrderID)
	}
	if order.OrderNumber != "1001" {
		t.Errorf("expected OrderNumber '1001', got '%s'", order.OrderNumber)
	}
	if order.CustomerEmail != "test@example.com" {
		t.Errorf("expected CustomerEmail 'test@example.com', got '%s'", order.CustomerEmail)
	}
	if order.CustomerName != "John Doe" {
		t.Errorf("expected CustomerName 'John Doe', got '%s'", order.CustomerName)
	}
	if order.SubtotalCents != 1999 {
		t.Errorf("expected SubtotalCents 1999, got %d", order.SubtotalCents)
	}
	if order.ShippingCents != 500 {
		t.Errorf("expected ShippingCents 500, got %d", order.ShippingCents)
	}
	if order.GrandTotalCents != 2699 {
		t.Errorf("expected GrandTotalCents 2699, got %d", order.GrandTotalCents)
	}
	if order.Currency != "USD" {
		t.Errorf("expected Currency 'USD', got '%s'", order.Currency)
	}
	if order.BillingAddress == nil {
		t.Error("expected BillingAddress to be set")
	} else if order.BillingAddress.City != "New York" {
		t.Errorf("expected BillingAddress.City 'New York', got '%s'", order.BillingAddress.City)
	}
	if order.ShippingAddress == nil {
		t.Error("expected ShippingAddress to be set")
	} else if order.ShippingAddress.City != "Boston" {
		t.Errorf("expected ShippingAddress.City 'Boston', got '%s'", order.ShippingAddress.City)
	}
}

func TestConvertOrderItem(t *testing.T) {
	svc := &SquarespaceService{}
	orderID := uuid.New()

	apiItem := squarespace.LineItem{
		ID:          "sq-item-1",
		ProductID:   "sq-prod-1",
		VariantID:   "sq-var-1",
		ProductName: "Test Product",
		SKU:         "TEST-001",
		Quantity:    2,
		UnitPricePaid: squarespace.Money{Value: "14.99", Currency: "USD"},
		ImageURL:    "https://example.com/image.jpg",
		VariantOptions: []string{"Size: Large", "Color: Blue"},
	}

	item := svc.convertOrderItem(orderID, apiItem)

	if item.OrderID != orderID {
		t.Errorf("expected OrderID %s, got %s", orderID, item.OrderID)
	}
	if item.SquarespaceItemID != "sq-item-1" {
		t.Errorf("expected SquarespaceItemID 'sq-item-1', got '%s'", item.SquarespaceItemID)
	}
	if item.ProductID != "sq-prod-1" {
		t.Errorf("expected ProductID 'sq-prod-1', got '%s'", item.ProductID)
	}
	if item.ProductName != "Test Product" {
		t.Errorf("expected ProductName 'Test Product', got '%s'", item.ProductName)
	}
	if item.SKU != "TEST-001" {
		t.Errorf("expected SKU 'TEST-001', got '%s'", item.SKU)
	}
	if item.Quantity != 2 {
		t.Errorf("expected Quantity 2, got %d", item.Quantity)
	}
	if item.UnitPriceCents != 1499 {
		t.Errorf("expected UnitPriceCents 1499, got %d", item.UnitPriceCents)
	}

	// Check variant options were stored
	if item.VariantOptions == nil {
		t.Error("expected VariantOptions to be set")
	} else {
		var opts []string
		json.Unmarshal(item.VariantOptions, &opts)
		if len(opts) != 2 {
			t.Errorf("expected 2 variant options, got %d", len(opts))
		}
	}
}

func TestConvertProduct(t *testing.T) {
	svc := &SquarespaceService{}

	apiProduct := squarespace.Product{
		ID:          "sq-prod-123",
		Name:        "Test Product",
		Description: "A great product",
		URL:         "/store/test-product",
		Type:        "PHYSICAL",
		IsVisible:   true,
		Tags:        []string{"sale", "featured"},
		CreatedOn:   "2024-01-01T00:00:00Z",
		ModifiedOn:  "2024-01-10T00:00:00Z",
	}

	product := svc.convertProduct(apiProduct)

	if product.SquarespaceProductID != "sq-prod-123" {
		t.Errorf("expected SquarespaceProductID 'sq-prod-123', got '%s'", product.SquarespaceProductID)
	}
	if product.Name != "Test Product" {
		t.Errorf("expected Name 'Test Product', got '%s'", product.Name)
	}
	if product.Description != "A great product" {
		t.Errorf("expected Description 'A great product', got '%s'", product.Description)
	}
	if product.Type != "PHYSICAL" {
		t.Errorf("expected Type 'PHYSICAL', got '%s'", product.Type)
	}
	if !product.IsVisible {
		t.Error("expected IsVisible to be true")
	}

	// Check tags were stored
	if product.Tags == nil {
		t.Error("expected Tags to be set")
	} else {
		var tags []string
		json.Unmarshal(product.Tags, &tags)
		if len(tags) != 2 {
			t.Errorf("expected 2 tags, got %d", len(tags))
		}
	}
}

func TestConvertProductVariant(t *testing.T) {
	svc := &SquarespaceService{}
	productID := uuid.New()

	apiVariant := squarespace.ProductVariant{
		ID:  "sq-var-1",
		SKU: "VAR-001",
		Pricing: squarespace.ProductPricing{
			BasePrice: squarespace.Money{Value: "29.99", Currency: "USD"},
			SalePrice: squarespace.Money{Value: "24.99", Currency: "USD"},
			OnSale:    true,
		},
		Stock: squarespace.ProductStock{
			Quantity:  50,
			Unlimited: false,
		},
		Attributes: map[string]string{
			"Size":  "Large",
			"Color": "Red",
		},
	}

	variant := svc.convertProductVariant(productID, apiVariant)

	if variant.ProductID != productID {
		t.Errorf("expected ProductID %s, got %s", productID, variant.ProductID)
	}
	if variant.SquarespaceVariantID != "sq-var-1" {
		t.Errorf("expected SquarespaceVariantID 'sq-var-1', got '%s'", variant.SquarespaceVariantID)
	}
	if variant.SKU != "VAR-001" {
		t.Errorf("expected SKU 'VAR-001', got '%s'", variant.SKU)
	}
	if variant.PriceCents != 2999 {
		t.Errorf("expected PriceCents 2999, got %d", variant.PriceCents)
	}
	if variant.SalePriceCents != 2499 {
		t.Errorf("expected SalePriceCents 2499, got %d", variant.SalePriceCents)
	}
	if !variant.OnSale {
		t.Error("expected OnSale to be true")
	}
	if variant.StockQuantity != 50 {
		t.Errorf("expected StockQuantity 50, got %d", variant.StockQuantity)
	}
	if variant.StockUnlimited {
		t.Error("expected StockUnlimited to be false")
	}

	// Check attributes were stored
	if variant.Attributes == nil {
		t.Error("expected Attributes to be set")
	}
}

func TestSquarespaceIntegrationModel(t *testing.T) {
	now := time.Now()
	integration := &model.SquarespaceIntegration{
		ID:                uuid.New(),
		SiteID:            "site-123",
		SiteTitle:         "Test Store",
		APIKey:            "secret-key",
		IsActive:          true,
		LastOrderSyncAt:   &now,
		LastProductSyncAt: nil,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	// Verify JSON serialization doesn't expose API key
	jsonData, err := json.Marshal(integration)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal(jsonData, &parsed)

	if _, exists := parsed["api_key"]; exists {
		t.Error("API key should not be exposed in JSON")
	}
	if parsed["site_id"] != "site-123" {
		t.Errorf("expected site_id 'site-123', got '%v'", parsed["site_id"])
	}
}

func TestSquarespaceOrderModel(t *testing.T) {
	orderID := uuid.New()
	projectID := uuid.New()
	now := time.Now()

	order := &model.SquarespaceOrder{
		ID:                 orderID,
		SquarespaceOrderID: "sq-123",
		OrderNumber:        "1001",
		CustomerEmail:      "test@example.com",
		CustomerName:       "Test User",
		GrandTotalCents:    2999,
		Currency:           "USD",
		FulfillmentStatus:  "FULFILLED",
		IsProcessed:        true,
		ProjectID:          &projectID,
		BillingAddress: &model.SquarespaceAddress{
			FirstName: "Test",
			LastName:  "User",
			City:      "Boston",
		},
		CreatedOn:  &now,
		ModifiedOn: &now,
		SyncedAt:   now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	jsonData, err := json.Marshal(order)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed model.SquarespaceOrder
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.ID != orderID {
		t.Errorf("expected ID %s, got %s", orderID, parsed.ID)
	}
	if parsed.OrderNumber != "1001" {
		t.Errorf("expected OrderNumber '1001', got '%s'", parsed.OrderNumber)
	}
	if !parsed.IsProcessed {
		t.Error("expected IsProcessed to be true")
	}
	if parsed.ProjectID == nil || *parsed.ProjectID != projectID {
		t.Error("expected ProjectID to be set correctly")
	}
}

func TestSquarespaceProductModel(t *testing.T) {
	productID := uuid.New()
	now := time.Now()

	product := &model.SquarespaceProduct{
		ID:                   productID,
		SquarespaceProductID: "sq-prod-1",
		Name:                 "Test Product",
		Description:          "Description",
		Type:                 "PHYSICAL",
		IsVisible:            true,
		SyncedAt:             now,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	jsonData, err := json.Marshal(product)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed model.SquarespaceProduct
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.ID != productID {
		t.Errorf("expected ID %s, got %s", productID, parsed.ID)
	}
	if parsed.Name != "Test Product" {
		t.Errorf("expected Name 'Test Product', got '%s'", parsed.Name)
	}
}

func TestSquarespaceProductTemplateModel(t *testing.T) {
	linkID := uuid.New()
	templateID := uuid.New()
	now := time.Now()

	link := &model.SquarespaceProductTemplate{
		ID:                   linkID,
		SquarespaceProductID: "sq-prod-1",
		TemplateID:           templateID,
		SKU:                  "LINK-SKU",
		CreatedAt:            now,
	}

	jsonData, err := json.Marshal(link)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed model.SquarespaceProductTemplate
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.TemplateID != templateID {
		t.Errorf("expected TemplateID %s, got %s", templateID, parsed.TemplateID)
	}
	if parsed.SKU != "LINK-SKU" {
		t.Errorf("expected SKU 'LINK-SKU', got '%s'", parsed.SKU)
	}
}

func TestCreateFromTemplateOptions(t *testing.T) {
	t.Run("squarespace source", func(t *testing.T) {
		opts := CreateFromTemplateOptions{
			OrderQuantity:   1,
			Source:          "squarespace",
			ExternalOrderID: "squarespace-sq-123",
		}

		if opts.Source != "squarespace" {
			t.Errorf("expected source 'squarespace', got '%s'", opts.Source)
		}
		if opts.ExternalOrderID != "squarespace-sq-123" {
			t.Errorf("expected ExternalOrderID 'squarespace-sq-123', got '%s'", opts.ExternalOrderID)
		}
	})
}
