package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/hyperion/printfarm/internal/crypto"
	"github.com/hyperion/printfarm/internal/model"
)

// ShopifyRepository handles Shopify integration database operations.
type ShopifyRepository struct {
	db *sql.DB
}

// ---- Credentials ----

// SaveCredentials saves or updates Shopify credentials.
func (r *ShopifyRepository) SaveCredentials(ctx context.Context, creds *model.ShopifyCredentials) error {
	creds.UpdatedAt = time.Now()
	if creds.ID == uuid.Nil {
		creds.ID = uuid.New()
		creds.CreatedAt = time.Now()
	}

	encrypted, err := crypto.Encrypt(creds.AccessToken)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO shopify_credentials (id, shop_domain, access_token, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (shop_domain) DO UPDATE SET access_token = ?, updated_at = ?
	`, creds.ID, creds.ShopDomain, encrypted, creds.CreatedAt, creds.UpdatedAt, encrypted, creds.UpdatedAt)
	return err
}

// GetCredentials retrieves the Shopify credentials.
func (r *ShopifyRepository) GetCredentials(ctx context.Context) (*model.ShopifyCredentials, error) {
	var creds model.ShopifyCredentials
	var encryptedToken string
	err := r.db.QueryRowContext(ctx, `
		SELECT id, shop_domain, access_token, created_at, updated_at
		FROM shopify_credentials LIMIT 1
	`).Scan(&creds.ID, &creds.ShopDomain, &encryptedToken, &creds.CreatedAt, &creds.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	decrypted, err := crypto.Decrypt(encryptedToken)
	if err != nil {
		return nil, err
	}
	creds.AccessToken = decrypted

	return &creds, nil
}

// DeleteCredentials removes Shopify credentials.
func (r *ShopifyRepository) DeleteCredentials(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM shopify_credentials`)
	return err
}

// ---- Orders ----

// SaveOrder saves or updates a Shopify order.
func (r *ShopifyRepository) SaveOrder(ctx context.Context, order *model.ShopifyOrder) error {
	order.UpdatedAt = time.Now()
	if order.ID == uuid.Nil {
		order.ID = uuid.New()
		order.CreatedAt = time.Now()
	}
	order.SyncedAt = time.Now()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO shopify_orders (id, shopify_order_id, order_id, shop_domain, order_number, customer_name, customer_email, total_cents, status, synced_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (shopify_order_id) DO UPDATE SET
			order_id = ?, order_number = ?, customer_name = ?, customer_email = ?, total_cents = ?, status = ?, synced_at = ?, updated_at = ?
	`, order.ID, order.ShopifyOrderID, order.OrderID, order.ShopDomain, order.OrderNumber, order.CustomerName, order.CustomerEmail, order.TotalCents, order.Status, order.SyncedAt, order.CreatedAt, order.UpdatedAt,
		order.OrderID, order.OrderNumber, order.CustomerName, order.CustomerEmail, order.TotalCents, order.Status, order.SyncedAt, order.UpdatedAt)
	return err
}

// GetOrderByShopifyID retrieves a Shopify order by its Shopify ID.
func (r *ShopifyRepository) GetOrderByShopifyID(ctx context.Context, shopifyOrderID string) (*model.ShopifyOrder, error) {
	var order model.ShopifyOrder
	err := r.db.QueryRowContext(ctx, `
		SELECT id, shopify_order_id, order_id, shop_domain, order_number, customer_name, customer_email, total_cents, status, synced_at, created_at, updated_at
		FROM shopify_orders WHERE shopify_order_id = ?
	`, shopifyOrderID).Scan(&order.ID, &order.ShopifyOrderID, &order.OrderID, &order.ShopDomain, &order.OrderNumber, &order.CustomerName, &order.CustomerEmail, &order.TotalCents, &order.Status, &order.SyncedAt, &order.CreatedAt, &order.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &order, err
}

// GetOrderByID retrieves a Shopify order by internal ID.
func (r *ShopifyRepository) GetOrderByID(ctx context.Context, id uuid.UUID) (*model.ShopifyOrder, error) {
	var order model.ShopifyOrder
	err := r.db.QueryRowContext(ctx, `
		SELECT id, shopify_order_id, order_id, shop_domain, order_number, customer_name, customer_email, total_cents, status, synced_at, created_at, updated_at
		FROM shopify_orders WHERE id = ?
	`, id).Scan(&order.ID, &order.ShopifyOrderID, &order.OrderID, &order.ShopDomain, &order.OrderNumber, &order.CustomerName, &order.CustomerEmail, &order.TotalCents, &order.Status, &order.SyncedAt, &order.CreatedAt, &order.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &order, err
}

// ListOrders retrieves Shopify orders with optional filtering.
func (r *ShopifyRepository) ListOrders(ctx context.Context, processed *bool, limit, offset int) ([]model.ShopifyOrder, error) {
	query := `
		SELECT id, shopify_order_id, order_id, shop_domain, order_number, customer_name, customer_email, total_cents, status, synced_at, created_at, updated_at
		FROM shopify_orders WHERE 1=1
	`
	args := []interface{}{}

	if processed != nil {
		if *processed {
			query += " AND order_id IS NOT NULL"
		} else {
			query += " AND order_id IS NULL"
		}
	}

	query += " ORDER BY created_at DESC"

	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}
	if offset > 0 {
		query += " OFFSET ?"
		args = append(args, offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []model.ShopifyOrder
	for rows.Next() {
		var order model.ShopifyOrder
		if err := rows.Scan(&order.ID, &order.ShopifyOrderID, &order.OrderID, &order.ShopDomain, &order.OrderNumber, &order.CustomerName, &order.CustomerEmail, &order.TotalCents, &order.Status, &order.SyncedAt, &order.CreatedAt, &order.UpdatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

// UpdateOrderProcessed links a Shopify order to a unified order.
func (r *ShopifyRepository) UpdateOrderProcessed(ctx context.Context, shopifyOrderID uuid.UUID, orderID *uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE shopify_orders SET order_id = ?, updated_at = ?
		WHERE id = ?
	`, orderID, time.Now(), shopifyOrderID)
	return err
}

// ---- Order Items ----

// SaveOrderItem saves a Shopify order line item.
func (r *ShopifyRepository) SaveOrderItem(ctx context.Context, item *model.ShopifyOrderItem) error {
	if item.ID == uuid.Nil {
		item.ID = uuid.New()
	}
	item.CreatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO shopify_order_items (id, shopify_order_id, shopify_line_item_id, sku, title, quantity, price_cents, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (shopify_line_item_id) DO UPDATE SET
			sku = ?, title = ?, quantity = ?, price_cents = ?
	`, item.ID, item.ShopifyOrderID, item.ShopifyLineItemID, item.SKU, item.Title, item.Quantity, item.PriceCents, item.CreatedAt,
		item.SKU, item.Title, item.Quantity, item.PriceCents)
	return err
}

// GetOrderItems retrieves all items for a Shopify order.
func (r *ShopifyRepository) GetOrderItems(ctx context.Context, shopifyOrderID uuid.UUID) ([]model.ShopifyOrderItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, shopify_order_id, shopify_line_item_id, sku, title, quantity, price_cents, created_at
		FROM shopify_order_items WHERE shopify_order_id = ?
	`, shopifyOrderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.ShopifyOrderItem
	for rows.Next() {
		var item model.ShopifyOrderItem
		if err := rows.Scan(&item.ID, &item.ShopifyOrderID, &item.ShopifyLineItemID, &item.SKU, &item.Title, &item.Quantity, &item.PriceCents, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// ---- Product Templates ----

// SaveProductTemplate links a Shopify product to a template.
func (r *ShopifyRepository) SaveProductTemplate(ctx context.Context, link *model.ShopifyProductTemplate) error {
	if link.ID == uuid.Nil {
		link.ID = uuid.New()
	}
	link.CreatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO shopify_product_templates (id, shopify_product_id, template_id, sku, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (shopify_product_id, template_id) DO UPDATE SET sku = ?
	`, link.ID, link.ShopifyProductID, link.TemplateID, link.SKU, link.CreatedAt, link.SKU)
	return err
}

// DeleteProductTemplate removes a product-template link.
func (r *ShopifyRepository) DeleteProductTemplate(ctx context.Context, productID string, templateID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM shopify_product_templates WHERE shopify_product_id = ? AND template_id = ?
	`, productID, templateID)
	return err
}

// GetProductTemplatesBySKU retrieves template links by SKU.
func (r *ShopifyRepository) GetProductTemplatesBySKU(ctx context.Context, sku string) ([]model.ShopifyProductTemplate, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, shopify_product_id, template_id, sku, created_at
		FROM shopify_product_templates WHERE sku = ?
	`, sku)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []model.ShopifyProductTemplate
	for rows.Next() {
		var link model.ShopifyProductTemplate
		if err := rows.Scan(&link.ID, &link.ShopifyProductID, &link.TemplateID, &link.SKU, &link.CreatedAt); err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	return links, rows.Err()
}

// GetTemplatesForProduct retrieves all template links for a Shopify product.
func (r *ShopifyRepository) GetTemplatesForProduct(ctx context.Context, productID string) ([]model.ShopifyProductTemplate, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, shopify_product_id, template_id, sku, created_at
		FROM shopify_product_templates WHERE shopify_product_id = ?
	`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []model.ShopifyProductTemplate
	for rows.Next() {
		var link model.ShopifyProductTemplate
		if err := rows.Scan(&link.ID, &link.ShopifyProductID, &link.TemplateID, &link.SKU, &link.CreatedAt); err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	return links, rows.Err()
}
