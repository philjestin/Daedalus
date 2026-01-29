package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/hyperion/printfarm/internal/model"
	"github.com/hyperion/printfarm/internal/repository"
	"github.com/hyperion/printfarm/internal/squarespace"
)

// SquarespaceService handles Squarespace integration business logic.
type SquarespaceService struct {
	repo        *repository.SquarespaceRepository
	templateSvc *TemplateService
}

// NewSquarespaceService creates a new SquarespaceService.
func NewSquarespaceService(repo *repository.SquarespaceRepository, templateSvc *TemplateService) *SquarespaceService {
	return &SquarespaceService{
		repo:        repo,
		templateSvc: templateSvc,
	}
}

// Connect validates the API key and saves the integration.
func (s *SquarespaceService) Connect(ctx context.Context, apiKey string) (*model.SquarespaceIntegration, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	// Create client and validate key by fetching site info
	client := squarespace.NewClient(apiKey)
	website, err := client.GetWebsite(ctx)
	if err != nil {
		return nil, fmt.Errorf("validating API key: %w", err)
	}

	// Save integration
	integration := &model.SquarespaceIntegration{
		SiteID:    website.ID,
		SiteTitle: website.Title,
		APIKey:    apiKey,
		IsActive:  true,
	}
	if err := s.repo.SaveIntegration(ctx, integration); err != nil {
		return nil, fmt.Errorf("saving integration: %w", err)
	}

	slog.Info("Squarespace site connected", "site_id", website.ID, "site_title", website.Title)
	return integration, nil
}

// Disconnect removes the Squarespace integration.
func (s *SquarespaceService) Disconnect(ctx context.Context) error {
	if err := s.repo.DeleteIntegration(ctx); err != nil {
		return fmt.Errorf("deleting integration: %w", err)
	}
	slog.Info("Squarespace site disconnected")
	return nil
}

// GetStatus returns the current Squarespace integration status.
func (s *SquarespaceService) GetStatus(ctx context.Context) (*model.SquarespaceIntegration, error) {
	return s.repo.GetIntegration(ctx)
}

// getClient creates an authenticated Squarespace client.
func (s *SquarespaceService) getClient(ctx context.Context) (*squarespace.Client, error) {
	integration, err := s.repo.GetIntegration(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting integration: %w", err)
	}
	if integration == nil {
		return nil, fmt.Errorf("no Squarespace integration found")
	}

	return squarespace.NewClient(integration.APIKey), nil
}

// ---- Order Methods ----

// SyncOrders fetches orders from Squarespace and stores them.
func (s *SquarespaceService) SyncOrders(ctx context.Context) (*model.SyncResult, error) {
	client, err := s.getClient(ctx)
	if err != nil {
		return nil, err
	}

	integration, err := s.repo.GetIntegration(ctx)
	if err != nil {
		return nil, err
	}

	result := &model.SyncResult{}
	opts := &squarespace.OrdersOptions{}

	// Only fetch orders modified since last sync
	if integration.LastOrderSyncAt != nil {
		opts.ModifiedAfter = integration.LastOrderSyncAt
	}

	// Paginate through all orders
	for {
		resp, err := client.GetOrders(ctx, opts)
		if err != nil {
			slog.Error("failed to fetch orders", "error", err)
			result.Errors++
			break
		}

		for _, apiOrder := range resp.Result {
			result.TotalFetched++

			// Check if order already exists
			existing, err := s.repo.GetOrderBySquarespaceID(ctx, apiOrder.ID)
			if err != nil {
				slog.Error("failed to check existing order", "order_id", apiOrder.ID, "error", err)
				result.Errors++
				continue
			}

			order := s.convertOrder(apiOrder)
			if existing != nil {
				order.ID = existing.ID
				order.IsProcessed = existing.IsProcessed
				order.ProjectID = existing.ProjectID
				order.CreatedAt = existing.CreatedAt
				result.Updated++
			} else {
				result.Created++
			}

			if err := s.repo.SaveOrder(ctx, order); err != nil {
				slog.Error("failed to save order", "order_id", apiOrder.ID, "error", err)
				result.Errors++
				continue
			}

			// Save order items
			for _, apiItem := range apiOrder.LineItems {
				item := s.convertOrderItem(order.ID, apiItem)
				if err := s.repo.SaveOrderItem(ctx, item); err != nil {
					slog.Error("failed to save order item", "item_id", apiItem.ID, "error", err)
					result.Errors++
				}
			}
		}

		// Check for more pages
		if !resp.Pagination.HasNextPage {
			break
		}
		opts.Cursor = resp.Pagination.NextPageCursor
	}

	// Update last sync timestamp
	now := time.Now()
	if err := s.repo.UpdateLastSync(ctx, &now, nil); err != nil {
		slog.Error("failed to update last sync", "error", err)
	}

	slog.Info("Squarespace order sync complete",
		"fetched", result.TotalFetched,
		"created", result.Created,
		"updated", result.Updated,
		"errors", result.Errors)

	return result, nil
}

// convertOrder converts a Squarespace API order to our model.
func (s *SquarespaceService) convertOrder(apiOrder squarespace.Order) *model.SquarespaceOrder {
	order := &model.SquarespaceOrder{
		SquarespaceOrderID: apiOrder.ID,
		OrderNumber:        apiOrder.OrderNumber,
		CustomerEmail:      apiOrder.CustomerEmail,
		CustomerName:       squarespace.CustomerName(apiOrder.BillingAddress),
		Channel:            apiOrder.Channel,
		SubtotalCents:      squarespace.MoneyToCents(apiOrder.SubtotalPrice),
		ShippingCents:      squarespace.MoneyToCents(apiOrder.ShippingTotal),
		TaxCents:           squarespace.MoneyToCents(apiOrder.TaxTotal),
		DiscountCents:      squarespace.MoneyToCents(apiOrder.DiscountTotal),
		RefundedCents:      squarespace.MoneyToCents(apiOrder.RefundedTotal),
		GrandTotalCents:    squarespace.MoneyToCents(apiOrder.GrandTotal),
		Currency:           apiOrder.GrandTotal.Currency,
		FulfillmentStatus:  apiOrder.FulfillmentStatus,
		SyncedAt:           time.Now(),
	}

	// Convert addresses
	order.BillingAddress = &model.SquarespaceAddress{
		FirstName:   apiOrder.BillingAddress.FirstName,
		LastName:    apiOrder.BillingAddress.LastName,
		Address1:    apiOrder.BillingAddress.Address1,
		Address2:    apiOrder.BillingAddress.Address2,
		City:        apiOrder.BillingAddress.City,
		State:       apiOrder.BillingAddress.State,
		PostalCode:  apiOrder.BillingAddress.PostalCode,
		CountryCode: apiOrder.BillingAddress.CountryCode,
		Phone:       apiOrder.BillingAddress.Phone,
	}
	order.ShippingAddress = &model.SquarespaceAddress{
		FirstName:   apiOrder.ShippingAddress.FirstName,
		LastName:    apiOrder.ShippingAddress.LastName,
		Address1:    apiOrder.ShippingAddress.Address1,
		Address2:    apiOrder.ShippingAddress.Address2,
		City:        apiOrder.ShippingAddress.City,
		State:       apiOrder.ShippingAddress.State,
		PostalCode:  apiOrder.ShippingAddress.PostalCode,
		CountryCode: apiOrder.ShippingAddress.CountryCode,
		Phone:       apiOrder.ShippingAddress.Phone,
	}

	// Parse timestamps
	if apiOrder.CreatedOn != "" {
		if t, err := time.Parse(time.RFC3339, apiOrder.CreatedOn); err == nil {
			order.CreatedOn = &t
		}
	}
	if apiOrder.ModifiedOn != "" {
		if t, err := time.Parse(time.RFC3339, apiOrder.ModifiedOn); err == nil {
			order.ModifiedOn = &t
		}
	}

	return order
}

// convertOrderItem converts a Squarespace API line item to our model.
func (s *SquarespaceService) convertOrderItem(orderID uuid.UUID, apiItem squarespace.LineItem) *model.SquarespaceOrderItem {
	item := &model.SquarespaceOrderItem{
		OrderID:           orderID,
		SquarespaceItemID: apiItem.ID,
		ProductID:         apiItem.ProductID,
		VariantID:         apiItem.VariantID,
		ProductName:       apiItem.ProductName,
		SKU:               apiItem.SKU,
		Quantity:          apiItem.Quantity,
		UnitPriceCents:    squarespace.MoneyToCents(apiItem.UnitPricePaid),
		Currency:          apiItem.UnitPricePaid.Currency,
		ImageURL:          apiItem.ImageURL,
	}

	// Store variant options as JSON
	if len(apiItem.VariantOptions) > 0 {
		if data, err := json.Marshal(apiItem.VariantOptions); err == nil {
			item.VariantOptions = data
		}
	}

	return item
}

// ListOrders retrieves orders with optional filtering.
func (s *SquarespaceService) ListOrders(ctx context.Context, processed *bool, limit, offset int) ([]model.SquarespaceOrder, error) {
	orders, err := s.repo.ListOrders(ctx, processed, limit, offset)
	if err != nil {
		return nil, err
	}

	// Load items for each order
	for i := range orders {
		items, err := s.repo.GetOrderItems(ctx, orders[i].ID)
		if err != nil {
			slog.Warn("failed to load order items", "order_id", orders[i].ID, "error", err)
			continue
		}
		orders[i].Items = items
	}

	return orders, nil
}

// GetOrder retrieves a single order by ID with items.
func (s *SquarespaceService) GetOrder(ctx context.Context, id uuid.UUID) (*model.SquarespaceOrder, error) {
	order, err := s.repo.GetOrderByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, nil
	}

	// Load items
	items, err := s.repo.GetOrderItems(ctx, order.ID)
	if err != nil {
		return nil, err
	}
	order.Items = items

	return order, nil
}

// ProcessOrder creates a project from a Squarespace order.
func (s *SquarespaceService) ProcessOrder(ctx context.Context, orderID uuid.UUID) (*model.Project, error) {
	order, err := s.GetOrder(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("getting order: %w", err)
	}
	if order == nil {
		return nil, fmt.Errorf("order not found")
	}
	if order.IsProcessed {
		return nil, fmt.Errorf("order already processed")
	}

	// Find templates for order items by SKU
	var templateID *uuid.UUID
	for _, item := range order.Items {
		if item.SKU == "" {
			continue
		}

		// Look up template links by SKU
		links, err := s.repo.GetProductTemplatesBySKU(ctx, item.SKU)
		if err != nil {
			slog.Warn("failed to lookup template by SKU", "sku", item.SKU, "error", err)
			continue
		}
		if len(links) > 0 {
			templateID = &links[0].TemplateID
			break
		}
	}

	// Build project name
	projectName := fmt.Sprintf("Squarespace #%s", order.OrderNumber)
	if order.CustomerName != "" {
		projectName += " - " + order.CustomerName
	}

	externalOrderID := fmt.Sprintf("squarespace-%s", order.SquarespaceOrderID)

	// Create project
	var project *model.Project

	// If we have a template, use the template service to instantiate
	if templateID != nil && s.templateSvc != nil {
		opts := CreateFromTemplateOptions{
			OrderQuantity:   1,
			Source:          "squarespace",
			ExternalOrderID: externalOrderID,
		}
		proj, _, err := s.templateSvc.CreateProjectFromTemplate(ctx, *templateID, opts)
		if err != nil {
			return nil, fmt.Errorf("creating project from template: %w", err)
		}
		project = proj
	} else {
		// Just create a basic project without template instantiation
		project = &model.Project{
			ID:              uuid.New(),
			Name:            projectName,
			Source:          "squarespace",
			ExternalOrderID: externalOrderID,
			TemplateID:      templateID,
			Tags:            []string{},
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
	}

	// Mark order as processed
	if err := s.repo.UpdateOrderProcessed(ctx, order.ID, &project.ID); err != nil {
		return nil, fmt.Errorf("marking order processed: %w", err)
	}

	slog.Info("Squarespace order processed", "order_id", order.ID, "project_id", project.ID)
	return project, nil
}

// ---- Product Methods ----

// SyncProducts fetches products from Squarespace and stores them.
func (s *SquarespaceService) SyncProducts(ctx context.Context) (*model.SyncResult, error) {
	client, err := s.getClient(ctx)
	if err != nil {
		return nil, err
	}

	integration, err := s.repo.GetIntegration(ctx)
	if err != nil {
		return nil, err
	}

	result := &model.SyncResult{}
	opts := &squarespace.ProductsOptions{}

	// Only fetch products modified since last sync
	if integration.LastProductSyncAt != nil {
		opts.ModifiedAfter = integration.LastProductSyncAt
	}

	// Paginate through all products
	for {
		resp, err := client.GetProducts(ctx, opts)
		if err != nil {
			slog.Error("failed to fetch products", "error", err)
			result.Errors++
			break
		}

		for _, apiProduct := range resp.Result {
			result.TotalFetched++

			// Check if product already exists
			existing, err := s.repo.GetProductBySquarespaceID(ctx, apiProduct.ID)
			if err != nil {
				slog.Error("failed to check existing product", "product_id", apiProduct.ID, "error", err)
				result.Errors++
				continue
			}

			product := s.convertProduct(apiProduct)
			if existing != nil {
				product.ID = existing.ID
				product.CreatedAt = existing.CreatedAt
				result.Updated++
			} else {
				result.Created++
			}

			if err := s.repo.SaveProduct(ctx, product); err != nil {
				slog.Error("failed to save product", "product_id", apiProduct.ID, "error", err)
				result.Errors++
				continue
			}

			// Save variants
			for _, apiVariant := range apiProduct.Variants {
				variant := s.convertProductVariant(product.ID, apiVariant)
				if err := s.repo.SaveProductVariant(ctx, variant); err != nil {
					slog.Error("failed to save product variant", "variant_id", apiVariant.ID, "error", err)
					result.Errors++
				}
			}
		}

		// Check for more pages
		if !resp.Pagination.HasNextPage {
			break
		}
		opts.Cursor = resp.Pagination.NextPageCursor
	}

	// Update last sync timestamp
	now := time.Now()
	if err := s.repo.UpdateLastSync(ctx, nil, &now); err != nil {
		slog.Error("failed to update last sync", "error", err)
	}

	slog.Info("Squarespace product sync complete",
		"fetched", result.TotalFetched,
		"created", result.Created,
		"updated", result.Updated,
		"errors", result.Errors)

	return result, nil
}

// convertProduct converts a Squarespace API product to our model.
func (s *SquarespaceService) convertProduct(apiProduct squarespace.Product) *model.SquarespaceProduct {
	product := &model.SquarespaceProduct{
		SquarespaceProductID: apiProduct.ID,
		Name:                 apiProduct.Name,
		Description:          apiProduct.Description,
		URL:                  apiProduct.URL,
		Type:                 apiProduct.Type,
		IsVisible:            apiProduct.IsVisible,
		SyncedAt:             time.Now(),
	}

	// Store tags as JSON
	if len(apiProduct.Tags) > 0 {
		product.Tags, _ = json.Marshal(apiProduct.Tags)
	}

	// Parse timestamps
	if apiProduct.CreatedOn != "" {
		if t, err := time.Parse(time.RFC3339, apiProduct.CreatedOn); err == nil {
			product.CreatedOn = &t
		}
	}
	if apiProduct.ModifiedOn != "" {
		if t, err := time.Parse(time.RFC3339, apiProduct.ModifiedOn); err == nil {
			product.ModifiedOn = &t
		}
	}

	return product
}

// convertProductVariant converts a Squarespace API variant to our model.
func (s *SquarespaceService) convertProductVariant(productID uuid.UUID, apiVariant squarespace.ProductVariant) *model.SquarespaceProductVariant {
	variant := &model.SquarespaceProductVariant{
		ProductID:            productID,
		SquarespaceVariantID: apiVariant.ID,
		SKU:                  apiVariant.SKU,
		PriceCents:           squarespace.MoneyToCents(apiVariant.Pricing.BasePrice),
		SalePriceCents:       squarespace.MoneyToCents(apiVariant.Pricing.SalePrice),
		OnSale:               apiVariant.Pricing.OnSale,
		StockQuantity:        apiVariant.Stock.Quantity,
		StockUnlimited:       apiVariant.Stock.Unlimited,
	}

	// Store attributes as JSON
	if len(apiVariant.Attributes) > 0 {
		variant.Attributes, _ = json.Marshal(apiVariant.Attributes)
	}

	return variant
}

// ListProducts retrieves products with optional limit/offset.
func (s *SquarespaceService) ListProducts(ctx context.Context, limit, offset int) ([]model.SquarespaceProduct, error) {
	products, err := s.repo.ListProducts(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	// Load variants for each product
	for i := range products {
		variants, err := s.repo.GetProductVariants(ctx, products[i].ID)
		if err != nil {
			slog.Warn("failed to load product variants", "product_id", products[i].ID, "error", err)
			continue
		}
		products[i].Variants = variants
	}

	return products, nil
}

// GetProduct retrieves a single product by ID with variants.
func (s *SquarespaceService) GetProduct(ctx context.Context, id uuid.UUID) (*model.SquarespaceProduct, error) {
	product, err := s.repo.GetProductByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, nil
	}

	// Load variants
	variants, err := s.repo.GetProductVariants(ctx, product.ID)
	if err != nil {
		return nil, err
	}
	product.Variants = variants

	return product, nil
}

// LinkProductToTemplate links a Squarespace product to a template.
func (s *SquarespaceService) LinkProductToTemplate(ctx context.Context, productID uuid.UUID, templateID uuid.UUID, sku string) error {
	product, err := s.repo.GetProductByID(ctx, productID)
	if err != nil {
		return fmt.Errorf("getting product: %w", err)
	}
	if product == nil {
		return fmt.Errorf("product not found")
	}

	link := &model.SquarespaceProductTemplate{
		SquarespaceProductID: product.SquarespaceProductID,
		TemplateID:           templateID,
		SKU:                  sku,
	}
	return s.repo.SaveProductTemplate(ctx, link)
}

// UnlinkProductFromTemplate removes a product-template link.
func (s *SquarespaceService) UnlinkProductFromTemplate(ctx context.Context, productID uuid.UUID, templateID uuid.UUID) error {
	product, err := s.repo.GetProductByID(ctx, productID)
	if err != nil {
		return fmt.Errorf("getting product: %w", err)
	}
	if product == nil {
		return fmt.Errorf("product not found")
	}

	return s.repo.DeleteProductTemplate(ctx, product.SquarespaceProductID, templateID)
}

// GetProductTemplates retrieves templates linked to a product.
func (s *SquarespaceService) GetProductTemplates(ctx context.Context, productID uuid.UUID) ([]model.SquarespaceProductTemplate, error) {
	product, err := s.repo.GetProductByID(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("getting product: %w", err)
	}
	if product == nil {
		return nil, fmt.Errorf("product not found")
	}

	return s.repo.GetTemplatesForProduct(ctx, product.SquarespaceProductID)
}
