package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/philjestin/daedalus/internal/crypto"
	"github.com/philjestin/daedalus/internal/model"
)

// SquarespaceRepository handles Squarespace integration database operations.
type SquarespaceRepository struct {
	db *sql.DB
}

// NewSquarespaceRepository creates a new SquarespaceRepository.
func NewSquarespaceRepository(db *sql.DB) *SquarespaceRepository {
	return &SquarespaceRepository{db: db}
}

// ---- Integration Methods ----

// SaveIntegration creates or updates a Squarespace integration.
// The API key is encrypted before storage.
func (r *SquarespaceRepository) SaveIntegration(ctx context.Context, i *model.SquarespaceIntegration) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	i.CreatedAt = time.Now()
	i.UpdatedAt = time.Now()

	// Encrypt API key before storing
	apiKey := i.APIKey
	if apiKey != "" {
		if encrypted, err := crypto.Encrypt(apiKey); err == nil {
			apiKey = encrypted
		} else {
			slog.Warn("failed to encrypt Squarespace API key", "error", err)
		}
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO squarespace_integration (id, site_id, site_title, api_key, is_active, last_order_sync_at, last_product_sync_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (id) DO UPDATE SET
			site_id = EXCLUDED.site_id,
			site_title = EXCLUDED.site_title,
			api_key = EXCLUDED.api_key,
			is_active = EXCLUDED.is_active,
			last_order_sync_at = EXCLUDED.last_order_sync_at,
			last_product_sync_at = EXCLUDED.last_product_sync_at,
			updated_at = EXCLUDED.updated_at
	`, i.ID, i.SiteID, i.SiteTitle, apiKey, i.IsActive, i.LastOrderSyncAt, i.LastProductSyncAt, i.CreatedAt, i.UpdatedAt)
	return err
}

// GetIntegration retrieves the current Squarespace integration (only one per install).
// The API key is decrypted before returning.
func (r *SquarespaceRepository) GetIntegration(ctx context.Context) (*model.SquarespaceIntegration, error) {
	var i model.SquarespaceIntegration
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, site_id, site_title, api_key, is_active, last_order_sync_at, last_product_sync_at, created_at, updated_at
		FROM squarespace_integration
		WHERE is_active = 1
		LIMIT 1
	`), &i.ID, &i.SiteID, &i.SiteTitle, &i.APIKey, &i.IsActive, &i.LastOrderSyncAt, &i.LastProductSyncAt, &i.CreatedAt, &i.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Decrypt API key
	if decrypted, err := crypto.Decrypt(i.APIKey); err == nil {
		i.APIKey = decrypted
	}

	return &i, nil
}

// DeleteIntegration removes the Squarespace integration.
func (r *SquarespaceRepository) DeleteIntegration(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM squarespace_integration WHERE is_active = 1`)
	return err
}

// UpdateLastSync updates the last sync timestamps.
func (r *SquarespaceRepository) UpdateLastSync(ctx context.Context, orderSync, productSync *time.Time) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE squarespace_integration
		SET last_order_sync_at = COALESCE(?, last_order_sync_at),
		    last_product_sync_at = COALESCE(?, last_product_sync_at),
		    updated_at = ?
		WHERE is_active = 1
	`, orderSync, productSync, now)
	return err
}

// ---- Order Methods ----

// SaveOrder creates or updates a Squarespace order.
func (r *SquarespaceRepository) SaveOrder(ctx context.Context, order *model.SquarespaceOrder) error {
	if order.ID == uuid.Nil {
		order.ID = uuid.New()
	}
	order.UpdatedAt = time.Now()
	if order.CreatedAt.IsZero() {
		order.CreatedAt = time.Now()
	}
	if order.SyncedAt.IsZero() {
		order.SyncedAt = time.Now()
	}

	// Marshal addresses to JSON
	var billingJSON, shippingJSON []byte
	if order.BillingAddress != nil {
		billingJSON, _ = json.Marshal(order.BillingAddress)
	}
	if order.ShippingAddress != nil {
		shippingJSON, _ = json.Marshal(order.ShippingAddress)
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO squarespace_orders (
			id, squarespace_order_id, order_number, customer_email, customer_name, channel,
			subtotal_cents, shipping_cents, tax_cents, discount_cents, refunded_cents, grand_total_cents,
			currency, fulfillment_status, billing_address_json, shipping_address_json,
			created_on, modified_on, is_processed, project_id, synced_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (squarespace_order_id) DO UPDATE SET
			order_number = EXCLUDED.order_number,
			customer_email = EXCLUDED.customer_email,
			customer_name = EXCLUDED.customer_name,
			channel = EXCLUDED.channel,
			subtotal_cents = EXCLUDED.subtotal_cents,
			shipping_cents = EXCLUDED.shipping_cents,
			tax_cents = EXCLUDED.tax_cents,
			discount_cents = EXCLUDED.discount_cents,
			refunded_cents = EXCLUDED.refunded_cents,
			grand_total_cents = EXCLUDED.grand_total_cents,
			currency = EXCLUDED.currency,
			fulfillment_status = EXCLUDED.fulfillment_status,
			billing_address_json = EXCLUDED.billing_address_json,
			shipping_address_json = EXCLUDED.shipping_address_json,
			modified_on = EXCLUDED.modified_on,
			synced_at = EXCLUDED.synced_at,
			updated_at = EXCLUDED.updated_at
	`, order.ID, order.SquarespaceOrderID, order.OrderNumber, order.CustomerEmail, order.CustomerName, order.Channel,
		order.SubtotalCents, order.ShippingCents, order.TaxCents, order.DiscountCents, order.RefundedCents, order.GrandTotalCents,
		order.Currency, order.FulfillmentStatus, billingJSON, shippingJSON,
		order.CreatedOn, order.ModifiedOn, order.IsProcessed, order.ProjectID, order.SyncedAt, order.CreatedAt, order.UpdatedAt)
	return err
}

// GetOrderBySquarespaceID retrieves an order by its Squarespace order ID.
func (r *SquarespaceRepository) GetOrderBySquarespaceID(ctx context.Context, squarespaceOrderID string) (*model.SquarespaceOrder, error) {
	var order model.SquarespaceOrder
	var billingJSON, shippingJSON []byte
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, squarespace_order_id, order_number, customer_email, customer_name, channel,
			subtotal_cents, shipping_cents, tax_cents, discount_cents, refunded_cents, grand_total_cents,
			currency, fulfillment_status, billing_address_json, shipping_address_json,
			created_on, modified_on, is_processed, project_id, synced_at, created_at, updated_at
		FROM squarespace_orders
		WHERE squarespace_order_id = ?
	`, squarespaceOrderID),
		&order.ID, &order.SquarespaceOrderID, &order.OrderNumber, &order.CustomerEmail, &order.CustomerName, &order.Channel,
		&order.SubtotalCents, &order.ShippingCents, &order.TaxCents, &order.DiscountCents, &order.RefundedCents, &order.GrandTotalCents,
		&order.Currency, &order.FulfillmentStatus, &billingJSON, &shippingJSON,
		&order.CreatedOn, &order.ModifiedOn, &order.IsProcessed, &order.ProjectID, &order.SyncedAt, &order.CreatedAt, &order.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Unmarshal addresses
	if len(billingJSON) > 0 {
		json.Unmarshal(billingJSON, &order.BillingAddress)
	}
	if len(shippingJSON) > 0 {
		json.Unmarshal(shippingJSON, &order.ShippingAddress)
	}

	return &order, nil
}

// GetOrderByID retrieves an order by its internal UUID.
func (r *SquarespaceRepository) GetOrderByID(ctx context.Context, id uuid.UUID) (*model.SquarespaceOrder, error) {
	var order model.SquarespaceOrder
	var billingJSON, shippingJSON []byte
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, squarespace_order_id, order_number, customer_email, customer_name, channel,
			subtotal_cents, shipping_cents, tax_cents, discount_cents, refunded_cents, grand_total_cents,
			currency, fulfillment_status, billing_address_json, shipping_address_json,
			created_on, modified_on, is_processed, project_id, synced_at, created_at, updated_at
		FROM squarespace_orders
		WHERE id = ?
	`, id),
		&order.ID, &order.SquarespaceOrderID, &order.OrderNumber, &order.CustomerEmail, &order.CustomerName, &order.Channel,
		&order.SubtotalCents, &order.ShippingCents, &order.TaxCents, &order.DiscountCents, &order.RefundedCents, &order.GrandTotalCents,
		&order.Currency, &order.FulfillmentStatus, &billingJSON, &shippingJSON,
		&order.CreatedOn, &order.ModifiedOn, &order.IsProcessed, &order.ProjectID, &order.SyncedAt, &order.CreatedAt, &order.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Unmarshal addresses
	if len(billingJSON) > 0 {
		json.Unmarshal(billingJSON, &order.BillingAddress)
	}
	if len(shippingJSON) > 0 {
		json.Unmarshal(shippingJSON, &order.ShippingAddress)
	}

	return &order, nil
}

// ListOrders retrieves orders with optional filtering.
func (r *SquarespaceRepository) ListOrders(ctx context.Context, processed *bool, limit, offset int) ([]model.SquarespaceOrder, error) {
	query := `
		SELECT id, squarespace_order_id, order_number, customer_email, customer_name, channel,
			subtotal_cents, shipping_cents, tax_cents, discount_cents, refunded_cents, grand_total_cents,
			currency, fulfillment_status, billing_address_json, shipping_address_json,
			created_on, modified_on, is_processed, project_id, synced_at, created_at, updated_at
		FROM squarespace_orders
	`
	var args []interface{}

	if processed != nil {
		query += " WHERE is_processed = ?"
		args = append(args, *processed)
	}

	query += " ORDER BY created_on DESC"

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

	var orders []model.SquarespaceOrder
	for rows.Next() {
		var order model.SquarespaceOrder
		var billingJSON, shippingJSON []byte
		err := scanRow(rows,
			&order.ID, &order.SquarespaceOrderID, &order.OrderNumber, &order.CustomerEmail, &order.CustomerName, &order.Channel,
			&order.SubtotalCents, &order.ShippingCents, &order.TaxCents, &order.DiscountCents, &order.RefundedCents, &order.GrandTotalCents,
			&order.Currency, &order.FulfillmentStatus, &billingJSON, &shippingJSON,
			&order.CreatedOn, &order.ModifiedOn, &order.IsProcessed, &order.ProjectID, &order.SyncedAt, &order.CreatedAt, &order.UpdatedAt)
		if err != nil {
			return nil, err
		}

		// Unmarshal addresses
		if len(billingJSON) > 0 {
			json.Unmarshal(billingJSON, &order.BillingAddress)
		}
		if len(shippingJSON) > 0 {
			json.Unmarshal(shippingJSON, &order.ShippingAddress)
		}

		orders = append(orders, order)
	}

	return orders, rows.Err()
}

// UpdateOrderProcessed marks an order as processed.
func (r *SquarespaceRepository) UpdateOrderProcessed(ctx context.Context, id uuid.UUID, projectID *uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE squarespace_orders
		SET is_processed = 1, project_id = ?, updated_at = ?
		WHERE id = ?
	`, projectID, time.Now(), id)
	return err
}

// ---- Order Item Methods ----

// SaveOrderItem creates or updates an order item.
func (r *SquarespaceRepository) SaveOrderItem(ctx context.Context, item *model.SquarespaceOrderItem) error {
	if item.ID == uuid.Nil {
		item.ID = uuid.New()
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO squarespace_order_items (
			id, order_id, squarespace_item_id, product_id, variant_id, product_name,
			sku, quantity, unit_price_cents, currency, image_url, variant_options_json, template_id, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (squarespace_item_id) DO UPDATE SET
			product_name = EXCLUDED.product_name,
			sku = EXCLUDED.sku,
			quantity = EXCLUDED.quantity,
			unit_price_cents = EXCLUDED.unit_price_cents,
			template_id = EXCLUDED.template_id
	`, item.ID, item.OrderID, item.SquarespaceItemID, item.ProductID, item.VariantID, item.ProductName,
		item.SKU, item.Quantity, item.UnitPriceCents, item.Currency, item.ImageURL, item.VariantOptions, item.TemplateID, item.CreatedAt)
	return err
}

// GetOrderItems retrieves all items for an order.
func (r *SquarespaceRepository) GetOrderItems(ctx context.Context, orderID uuid.UUID) ([]model.SquarespaceOrderItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, order_id, squarespace_item_id, product_id, variant_id, product_name,
			sku, quantity, unit_price_cents, currency, image_url, variant_options_json, template_id, created_at
		FROM squarespace_order_items
		WHERE order_id = ?
		ORDER BY created_at
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.SquarespaceOrderItem
	for rows.Next() {
		var item model.SquarespaceOrderItem
		err := scanRow(rows,
			&item.ID, &item.OrderID, &item.SquarespaceItemID, &item.ProductID, &item.VariantID, &item.ProductName,
			&item.SKU, &item.Quantity, &item.UnitPriceCents, &item.Currency, &item.ImageURL, &item.VariantOptions, &item.TemplateID, &item.CreatedAt)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

// ---- Product Methods ----

// SaveProduct creates or updates a Squarespace product.
func (r *SquarespaceRepository) SaveProduct(ctx context.Context, product *model.SquarespaceProduct) error {
	if product.ID == uuid.Nil {
		product.ID = uuid.New()
	}
	product.UpdatedAt = time.Now()
	if product.CreatedAt.IsZero() {
		product.CreatedAt = time.Now()
	}
	if product.SyncedAt.IsZero() {
		product.SyncedAt = time.Now()
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO squarespace_products (
			id, squarespace_product_id, name, description, url, type, is_visible,
			tags_json, created_on, modified_on, synced_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (squarespace_product_id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			url = EXCLUDED.url,
			type = EXCLUDED.type,
			is_visible = EXCLUDED.is_visible,
			tags_json = EXCLUDED.tags_json,
			modified_on = EXCLUDED.modified_on,
			synced_at = EXCLUDED.synced_at,
			updated_at = EXCLUDED.updated_at
	`, product.ID, product.SquarespaceProductID, product.Name, product.Description, product.URL, product.Type, product.IsVisible,
		product.Tags, product.CreatedOn, product.ModifiedOn, product.SyncedAt, product.CreatedAt, product.UpdatedAt)
	return err
}

// GetProductBySquarespaceID retrieves a product by its Squarespace product ID.
func (r *SquarespaceRepository) GetProductBySquarespaceID(ctx context.Context, squarespaceProductID string) (*model.SquarespaceProduct, error) {
	var product model.SquarespaceProduct
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, squarespace_product_id, name, description, url, type, is_visible,
			tags_json, created_on, modified_on, synced_at, created_at, updated_at
		FROM squarespace_products
		WHERE squarespace_product_id = ?
	`, squarespaceProductID),
		&product.ID, &product.SquarespaceProductID, &product.Name, &product.Description, &product.URL, &product.Type, &product.IsVisible,
		&product.Tags, &product.CreatedOn, &product.ModifiedOn, &product.SyncedAt, &product.CreatedAt, &product.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &product, err
}

// GetProductByID retrieves a product by its internal UUID.
func (r *SquarespaceRepository) GetProductByID(ctx context.Context, id uuid.UUID) (*model.SquarespaceProduct, error) {
	var product model.SquarespaceProduct
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, squarespace_product_id, name, description, url, type, is_visible,
			tags_json, created_on, modified_on, synced_at, created_at, updated_at
		FROM squarespace_products
		WHERE id = ?
	`, id),
		&product.ID, &product.SquarespaceProductID, &product.Name, &product.Description, &product.URL, &product.Type, &product.IsVisible,
		&product.Tags, &product.CreatedOn, &product.ModifiedOn, &product.SyncedAt, &product.CreatedAt, &product.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &product, err
}

// ListProducts retrieves products with optional limit/offset.
func (r *SquarespaceRepository) ListProducts(ctx context.Context, limit, offset int) ([]model.SquarespaceProduct, error) {
	query := `
		SELECT id, squarespace_product_id, name, description, url, type, is_visible,
			tags_json, created_on, modified_on, synced_at, created_at, updated_at
		FROM squarespace_products
		ORDER BY name
	`
	var args []interface{}

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

	var products []model.SquarespaceProduct
	for rows.Next() {
		var product model.SquarespaceProduct
		err := scanRow(rows,
			&product.ID, &product.SquarespaceProductID, &product.Name, &product.Description, &product.URL, &product.Type, &product.IsVisible,
			&product.Tags, &product.CreatedOn, &product.ModifiedOn, &product.SyncedAt, &product.CreatedAt, &product.UpdatedAt)
		if err != nil {
			return nil, err
		}
		products = append(products, product)
	}

	return products, rows.Err()
}

// ---- Product Variant Methods ----

// SaveProductVariant creates or updates a product variant.
func (r *SquarespaceRepository) SaveProductVariant(ctx context.Context, variant *model.SquarespaceProductVariant) error {
	if variant.ID == uuid.Nil {
		variant.ID = uuid.New()
	}
	if variant.CreatedAt.IsZero() {
		variant.CreatedAt = time.Now()
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO squarespace_product_variants (
			id, product_id, squarespace_variant_id, sku, price_cents, sale_price_cents,
			on_sale, stock_quantity, stock_unlimited, attributes_json, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (squarespace_variant_id) DO UPDATE SET
			sku = EXCLUDED.sku,
			price_cents = EXCLUDED.price_cents,
			sale_price_cents = EXCLUDED.sale_price_cents,
			on_sale = EXCLUDED.on_sale,
			stock_quantity = EXCLUDED.stock_quantity,
			stock_unlimited = EXCLUDED.stock_unlimited,
			attributes_json = EXCLUDED.attributes_json
	`, variant.ID, variant.ProductID, variant.SquarespaceVariantID, variant.SKU, variant.PriceCents, variant.SalePriceCents,
		variant.OnSale, variant.StockQuantity, variant.StockUnlimited, variant.Attributes, variant.CreatedAt)
	return err
}

// GetProductVariants retrieves all variants for a product.
func (r *SquarespaceRepository) GetProductVariants(ctx context.Context, productID uuid.UUID) ([]model.SquarespaceProductVariant, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, product_id, squarespace_variant_id, sku, price_cents, sale_price_cents,
			on_sale, stock_quantity, stock_unlimited, attributes_json, created_at
		FROM squarespace_product_variants
		WHERE product_id = ?
		ORDER BY created_at
	`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var variants []model.SquarespaceProductVariant
	for rows.Next() {
		var variant model.SquarespaceProductVariant
		err := scanRow(rows,
			&variant.ID, &variant.ProductID, &variant.SquarespaceVariantID, &variant.SKU, &variant.PriceCents, &variant.SalePriceCents,
			&variant.OnSale, &variant.StockQuantity, &variant.StockUnlimited, &variant.Attributes, &variant.CreatedAt)
		if err != nil {
			return nil, err
		}
		variants = append(variants, variant)
	}

	return variants, rows.Err()
}

// ---- Product-Template Link Methods ----

// SaveProductTemplate creates or updates a product-template link.
func (r *SquarespaceRepository) SaveProductTemplate(ctx context.Context, link *model.SquarespaceProductTemplate) error {
	if link.ID == uuid.Nil {
		link.ID = uuid.New()
	}
	if link.CreatedAt.IsZero() {
		link.CreatedAt = time.Now()
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO squarespace_product_templates (id, squarespace_product_id, template_id, sku, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (squarespace_product_id, template_id) DO UPDATE SET
			sku = EXCLUDED.sku
	`, link.ID, link.SquarespaceProductID, link.TemplateID, link.SKU, link.CreatedAt)
	return err
}

// GetProductTemplate retrieves a product-template link.
func (r *SquarespaceRepository) GetProductTemplate(ctx context.Context, squarespaceProductID string, templateID uuid.UUID) (*model.SquarespaceProductTemplate, error) {
	var link model.SquarespaceProductTemplate
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, squarespace_product_id, template_id, sku, created_at
		FROM squarespace_product_templates
		WHERE squarespace_product_id = ? AND template_id = ?
	`, squarespaceProductID, templateID),
		&link.ID, &link.SquarespaceProductID, &link.TemplateID, &link.SKU, &link.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &link, err
}

// GetTemplatesForProduct retrieves all templates linked to a product.
func (r *SquarespaceRepository) GetTemplatesForProduct(ctx context.Context, squarespaceProductID string) ([]model.SquarespaceProductTemplate, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, squarespace_product_id, template_id, sku, created_at
		FROM squarespace_product_templates
		WHERE squarespace_product_id = ?
	`, squarespaceProductID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []model.SquarespaceProductTemplate
	for rows.Next() {
		var link model.SquarespaceProductTemplate
		err := scanRow(rows,
			&link.ID, &link.SquarespaceProductID, &link.TemplateID, &link.SKU, &link.CreatedAt)
		if err != nil {
			return nil, err
		}
		links = append(links, link)
	}

	return links, rows.Err()
}

// GetProductTemplatesBySKU retrieves product-template links by SKU.
func (r *SquarespaceRepository) GetProductTemplatesBySKU(ctx context.Context, sku string) ([]model.SquarespaceProductTemplate, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, squarespace_product_id, template_id, sku, created_at
		FROM squarespace_product_templates
		WHERE sku = ?
	`, sku)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []model.SquarespaceProductTemplate
	for rows.Next() {
		var link model.SquarespaceProductTemplate
		err := scanRow(rows,
			&link.ID, &link.SquarespaceProductID, &link.TemplateID, &link.SKU, &link.CreatedAt)
		if err != nil {
			return nil, err
		}
		links = append(links, link)
	}

	return links, rows.Err()
}

// DeleteProductTemplate removes a product-template link.
func (r *SquarespaceRepository) DeleteProductTemplate(ctx context.Context, squarespaceProductID string, templateID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM squarespace_product_templates
		WHERE squarespace_product_id = ? AND template_id = ?
	`, squarespaceProductID, templateID)
	return err
}
