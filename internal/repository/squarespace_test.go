package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/glebarez/go-sqlite"
	"github.com/google/uuid"
	"github.com/philjestin/daedalus/internal/model"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	// Create tables
	schema := `
		CREATE TABLE IF NOT EXISTS squarespace_integration (
			id TEXT PRIMARY KEY,
			site_id TEXT NOT NULL,
			site_title TEXT,
			api_key TEXT NOT NULL,
			is_active INTEGER DEFAULT 1,
			last_order_sync_at TEXT,
			last_product_sync_at TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS squarespace_orders (
			id TEXT PRIMARY KEY,
			squarespace_order_id TEXT UNIQUE NOT NULL,
			order_number TEXT,
			customer_email TEXT,
			customer_name TEXT,
			channel TEXT,
			subtotal_cents INTEGER,
			shipping_cents INTEGER,
			tax_cents INTEGER,
			discount_cents INTEGER,
			refunded_cents INTEGER,
			grand_total_cents INTEGER,
			currency TEXT DEFAULT 'USD',
			fulfillment_status TEXT,
			billing_address_json TEXT,
			shipping_address_json TEXT,
			created_on TEXT,
			modified_on TEXT,
			is_processed INTEGER DEFAULT 0,
			project_id TEXT,
			synced_at TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS squarespace_order_items (
			id TEXT PRIMARY KEY,
			order_id TEXT NOT NULL,
			squarespace_item_id TEXT UNIQUE NOT NULL,
			product_id TEXT,
			variant_id TEXT,
			product_name TEXT,
			sku TEXT,
			quantity INTEGER,
			unit_price_cents INTEGER,
			currency TEXT DEFAULT 'USD',
			image_url TEXT,
			variant_options_json TEXT,
			template_id TEXT,
			created_at TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS squarespace_products (
			id TEXT PRIMARY KEY,
			squarespace_product_id TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			description TEXT,
			url TEXT,
			type TEXT,
			is_visible INTEGER DEFAULT 1,
			tags_json TEXT,
			created_on TEXT,
			modified_on TEXT,
			synced_at TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS squarespace_product_variants (
			id TEXT PRIMARY KEY,
			product_id TEXT NOT NULL,
			squarespace_variant_id TEXT UNIQUE NOT NULL,
			sku TEXT,
			price_cents INTEGER,
			sale_price_cents INTEGER,
			on_sale INTEGER DEFAULT 0,
			stock_quantity INTEGER,
			stock_unlimited INTEGER DEFAULT 0,
			attributes_json TEXT,
			created_at TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS squarespace_product_templates (
			id TEXT PRIMARY KEY,
			squarespace_product_id TEXT NOT NULL,
			template_id TEXT NOT NULL,
			sku TEXT,
			created_at TEXT NOT NULL,
			UNIQUE(squarespace_product_id, template_id)
		);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	return db
}

func TestSquarespaceRepository_Integration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSquarespaceRepository(db)
	ctx := context.Background()

	t.Run("SaveIntegration", func(t *testing.T) {
		integration := &model.SquarespaceIntegration{
			SiteID:    "site-123",
			SiteTitle: "Test Store",
			APIKey:    "test-api-key",
			IsActive:  true,
		}

		err := repo.SaveIntegration(ctx, integration)
		if err != nil {
			t.Fatalf("SaveIntegration failed: %v", err)
		}

		if integration.ID == uuid.Nil {
			t.Error("expected ID to be set")
		}
	})

	t.Run("GetIntegration", func(t *testing.T) {
		integration, err := repo.GetIntegration(ctx)
		if err != nil {
			t.Fatalf("GetIntegration failed: %v", err)
		}

		if integration == nil {
			t.Fatal("expected integration, got nil")
		}
		if integration.SiteID != "site-123" {
			t.Errorf("expected SiteID 'site-123', got '%s'", integration.SiteID)
		}
		if integration.SiteTitle != "Test Store" {
			t.Errorf("expected SiteTitle 'Test Store', got '%s'", integration.SiteTitle)
		}
	})

	t.Run("UpdateLastSync", func(t *testing.T) {
		now := time.Now()
		err := repo.UpdateLastSync(ctx, &now, nil)
		if err != nil {
			t.Fatalf("UpdateLastSync failed: %v", err)
		}

		integration, _ := repo.GetIntegration(ctx)
		if integration.LastOrderSyncAt == nil {
			t.Error("expected LastOrderSyncAt to be set")
		}
	})

	t.Run("DeleteIntegration", func(t *testing.T) {
		err := repo.DeleteIntegration(ctx)
		if err != nil {
			t.Fatalf("DeleteIntegration failed: %v", err)
		}

		integration, _ := repo.GetIntegration(ctx)
		if integration != nil {
			t.Error("expected nil integration after delete")
		}
	})
}

func TestSquarespaceRepository_Orders(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSquarespaceRepository(db)
	ctx := context.Background()

	var orderID uuid.UUID

	t.Run("SaveOrder", func(t *testing.T) {
		order := &model.SquarespaceOrder{
			SquarespaceOrderID: "sq-order-123",
			OrderNumber:        "1001",
			CustomerEmail:      "test@example.com",
			CustomerName:       "John Doe",
			GrandTotalCents:    2999,
			Currency:           "USD",
			FulfillmentStatus:  "PENDING",
			BillingAddress: &model.SquarespaceAddress{
				FirstName: "John",
				LastName:  "Doe",
				City:      "New York",
			},
		}

		err := repo.SaveOrder(ctx, order)
		if err != nil {
			t.Fatalf("SaveOrder failed: %v", err)
		}

		if order.ID == uuid.Nil {
			t.Error("expected ID to be set")
		}
		orderID = order.ID
	})

	t.Run("GetOrderBySquarespaceID", func(t *testing.T) {
		order, err := repo.GetOrderBySquarespaceID(ctx, "sq-order-123")
		if err != nil {
			t.Fatalf("GetOrderBySquarespaceID failed: %v", err)
		}

		if order == nil {
			t.Fatal("expected order, got nil")
		}
		if order.OrderNumber != "1001" {
			t.Errorf("expected OrderNumber '1001', got '%s'", order.OrderNumber)
		}
		if order.BillingAddress == nil {
			t.Error("expected BillingAddress to be set")
		} else if order.BillingAddress.City != "New York" {
			t.Errorf("expected City 'New York', got '%s'", order.BillingAddress.City)
		}
	})

	t.Run("GetOrderByID", func(t *testing.T) {
		order, err := repo.GetOrderByID(ctx, orderID)
		if err != nil {
			t.Fatalf("GetOrderByID failed: %v", err)
		}

		if order == nil {
			t.Fatal("expected order, got nil")
		}
		if order.SquarespaceOrderID != "sq-order-123" {
			t.Errorf("expected SquarespaceOrderID 'sq-order-123', got '%s'", order.SquarespaceOrderID)
		}
	})

	t.Run("ListOrders", func(t *testing.T) {
		orders, err := repo.ListOrders(ctx, nil, 10, 0)
		if err != nil {
			t.Fatalf("ListOrders failed: %v", err)
		}

		if len(orders) != 1 {
			t.Errorf("expected 1 order, got %d", len(orders))
		}
	})

	t.Run("ListOrders_FilterProcessed", func(t *testing.T) {
		processed := false
		orders, err := repo.ListOrders(ctx, &processed, 10, 0)
		if err != nil {
			t.Fatalf("ListOrders failed: %v", err)
		}

		if len(orders) != 1 {
			t.Errorf("expected 1 unprocessed order, got %d", len(orders))
		}

		processed = true
		orders, err = repo.ListOrders(ctx, &processed, 10, 0)
		if err != nil {
			t.Fatalf("ListOrders failed: %v", err)
		}

		if len(orders) != 0 {
			t.Errorf("expected 0 processed orders, got %d", len(orders))
		}
	})

	t.Run("UpdateOrderProcessed", func(t *testing.T) {
		projectID := uuid.New()
		err := repo.UpdateOrderProcessed(ctx, orderID, &projectID)
		if err != nil {
			t.Fatalf("UpdateOrderProcessed failed: %v", err)
		}

		order, _ := repo.GetOrderByID(ctx, orderID)
		if !order.IsProcessed {
			t.Error("expected IsProcessed to be true")
		}
		if order.ProjectID == nil || *order.ProjectID != projectID {
			t.Error("expected ProjectID to be set")
		}
	})
}

func TestSquarespaceRepository_OrderItems(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSquarespaceRepository(db)
	ctx := context.Background()

	// Create an order first
	order := &model.SquarespaceOrder{
		SquarespaceOrderID: "sq-order-456",
		OrderNumber:        "1002",
	}
	repo.SaveOrder(ctx, order)

	t.Run("SaveOrderItem", func(t *testing.T) {
		item := &model.SquarespaceOrderItem{
			OrderID:           order.ID,
			SquarespaceItemID: "sq-item-1",
			ProductID:         "prod-1",
			ProductName:       "Test Product",
			SKU:               "TEST-001",
			Quantity:          2,
			UnitPriceCents:    1499,
			Currency:          "USD",
		}

		err := repo.SaveOrderItem(ctx, item)
		if err != nil {
			t.Fatalf("SaveOrderItem failed: %v", err)
		}

		if item.ID == uuid.Nil {
			t.Error("expected ID to be set")
		}
	})

	t.Run("GetOrderItems", func(t *testing.T) {
		items, err := repo.GetOrderItems(ctx, order.ID)
		if err != nil {
			t.Fatalf("GetOrderItems failed: %v", err)
		}

		if len(items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(items))
		}
		if items[0].ProductName != "Test Product" {
			t.Errorf("expected ProductName 'Test Product', got '%s'", items[0].ProductName)
		}
		if items[0].SKU != "TEST-001" {
			t.Errorf("expected SKU 'TEST-001', got '%s'", items[0].SKU)
		}
	})
}

func TestSquarespaceRepository_Products(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSquarespaceRepository(db)
	ctx := context.Background()

	var productID uuid.UUID

	t.Run("SaveProduct", func(t *testing.T) {
		product := &model.SquarespaceProduct{
			SquarespaceProductID: "sq-prod-123",
			Name:                 "Test Product",
			Description:          "A test product",
			Type:                 "PHYSICAL",
			IsVisible:            true,
		}

		err := repo.SaveProduct(ctx, product)
		if err != nil {
			t.Fatalf("SaveProduct failed: %v", err)
		}

		if product.ID == uuid.Nil {
			t.Error("expected ID to be set")
		}
		productID = product.ID
	})

	t.Run("GetProductBySquarespaceID", func(t *testing.T) {
		product, err := repo.GetProductBySquarespaceID(ctx, "sq-prod-123")
		if err != nil {
			t.Fatalf("GetProductBySquarespaceID failed: %v", err)
		}

		if product == nil {
			t.Fatal("expected product, got nil")
		}
		if product.Name != "Test Product" {
			t.Errorf("expected Name 'Test Product', got '%s'", product.Name)
		}
	})

	t.Run("GetProductByID", func(t *testing.T) {
		product, err := repo.GetProductByID(ctx, productID)
		if err != nil {
			t.Fatalf("GetProductByID failed: %v", err)
		}

		if product == nil {
			t.Fatal("expected product, got nil")
		}
	})

	t.Run("ListProducts", func(t *testing.T) {
		products, err := repo.ListProducts(ctx, 10, 0)
		if err != nil {
			t.Fatalf("ListProducts failed: %v", err)
		}

		if len(products) != 1 {
			t.Errorf("expected 1 product, got %d", len(products))
		}
	})
}

func TestSquarespaceRepository_ProductVariants(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSquarespaceRepository(db)
	ctx := context.Background()

	// Create a product first
	product := &model.SquarespaceProduct{
		SquarespaceProductID: "sq-prod-789",
		Name:                 "Product with Variants",
	}
	repo.SaveProduct(ctx, product)

	t.Run("SaveProductVariant", func(t *testing.T) {
		variant := &model.SquarespaceProductVariant{
			ProductID:            product.ID,
			SquarespaceVariantID: "sq-var-1",
			SKU:                  "VAR-001",
			PriceCents:           1999,
			StockQuantity:        10,
		}

		err := repo.SaveProductVariant(ctx, variant)
		if err != nil {
			t.Fatalf("SaveProductVariant failed: %v", err)
		}

		if variant.ID == uuid.Nil {
			t.Error("expected ID to be set")
		}
	})

	t.Run("GetProductVariants", func(t *testing.T) {
		variants, err := repo.GetProductVariants(ctx, product.ID)
		if err != nil {
			t.Fatalf("GetProductVariants failed: %v", err)
		}

		if len(variants) != 1 {
			t.Fatalf("expected 1 variant, got %d", len(variants))
		}
		if variants[0].SKU != "VAR-001" {
			t.Errorf("expected SKU 'VAR-001', got '%s'", variants[0].SKU)
		}
	})
}

func TestSquarespaceRepository_ProductTemplates(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSquarespaceRepository(db)
	ctx := context.Background()

	templateID := uuid.New()

	t.Run("SaveProductTemplate", func(t *testing.T) {
		link := &model.SquarespaceProductTemplate{
			SquarespaceProductID: "sq-prod-for-link",
			TemplateID:           templateID,
			SKU:                  "LINK-SKU",
		}

		err := repo.SaveProductTemplate(ctx, link)
		if err != nil {
			t.Fatalf("SaveProductTemplate failed: %v", err)
		}

		if link.ID == uuid.Nil {
			t.Error("expected ID to be set")
		}
	})

	t.Run("GetProductTemplate", func(t *testing.T) {
		link, err := repo.GetProductTemplate(ctx, "sq-prod-for-link", templateID)
		if err != nil {
			t.Fatalf("GetProductTemplate failed: %v", err)
		}

		if link == nil {
			t.Fatal("expected link, got nil")
		}
		if link.SKU != "LINK-SKU" {
			t.Errorf("expected SKU 'LINK-SKU', got '%s'", link.SKU)
		}
	})

	t.Run("GetTemplatesForProduct", func(t *testing.T) {
		links, err := repo.GetTemplatesForProduct(ctx, "sq-prod-for-link")
		if err != nil {
			t.Fatalf("GetTemplatesForProduct failed: %v", err)
		}

		if len(links) != 1 {
			t.Errorf("expected 1 link, got %d", len(links))
		}
	})

	t.Run("GetProductTemplatesBySKU", func(t *testing.T) {
		links, err := repo.GetProductTemplatesBySKU(ctx, "LINK-SKU")
		if err != nil {
			t.Fatalf("GetProductTemplatesBySKU failed: %v", err)
		}

		if len(links) != 1 {
			t.Errorf("expected 1 link, got %d", len(links))
		}
	})

	t.Run("DeleteProductTemplate", func(t *testing.T) {
		err := repo.DeleteProductTemplate(ctx, "sq-prod-for-link", templateID)
		if err != nil {
			t.Fatalf("DeleteProductTemplate failed: %v", err)
		}

		link, _ := repo.GetProductTemplate(ctx, "sq-prod-for-link", templateID)
		if link != nil {
			t.Error("expected link to be deleted")
		}
	})
}
