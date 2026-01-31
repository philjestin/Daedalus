package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hyperion/printfarm/internal/model"
	"github.com/hyperion/printfarm/internal/realtime"
	"github.com/hyperion/printfarm/internal/repository"
)

// ShopifyService handles Shopify integration business logic.
type ShopifyService struct {
	shopifyRepo *repository.ShopifyRepository
	orderSvc    *OrderService
	templateSvc *TemplateService
	hub         *realtime.Hub
}

// NewShopifyService creates a new ShopifyService.
func NewShopifyService(
	shopifyRepo *repository.ShopifyRepository,
	orderSvc *OrderService,
	templateSvc *TemplateService,
	hub *realtime.Hub,
) *ShopifyService {
	return &ShopifyService{
		shopifyRepo: shopifyRepo,
		orderSvc:    orderSvc,
		templateSvc: templateSvc,
		hub:         hub,
	}
}

// ShopifyConfig holds configuration for Shopify OAuth.
type ShopifyConfig struct {
	APIKey      string
	APISecret   string
	RedirectURI string
	Scopes      []string
}

// GetAuthURL generates the Shopify OAuth authorization URL.
func (s *ShopifyService) GetAuthURL(shopDomain string, config ShopifyConfig) (string, error) {
	if shopDomain == "" {
		return "", fmt.Errorf("shop domain is required")
	}

	// Normalize shop domain
	shopDomain = normalizeShopDomain(shopDomain)

	scopes := strings.Join(config.Scopes, ",")
	if scopes == "" {
		scopes = "read_orders,read_products"
	}

	authURL := fmt.Sprintf(
		"https://%s/admin/oauth/authorize?client_id=%s&scope=%s&redirect_uri=%s",
		shopDomain,
		url.QueryEscape(config.APIKey),
		url.QueryEscape(scopes),
		url.QueryEscape(config.RedirectURI),
	)

	return authURL, nil
}

// HandleOAuthCallback processes the OAuth callback and saves credentials.
func (s *ShopifyService) HandleOAuthCallback(ctx context.Context, shopDomain, code string, config ShopifyConfig) error {
	shopDomain = normalizeShopDomain(shopDomain)

	// Exchange code for access token
	tokenURL := fmt.Sprintf("https://%s/admin/oauth/access_token", shopDomain)
	payload := fmt.Sprintf(
		"client_id=%s&client_secret=%s&code=%s",
		url.QueryEscape(config.APIKey),
		url.QueryEscape(config.APISecret),
		url.QueryEscape(code),
	)

	resp, err := http.Post(tokenURL, "application/x-www-form-urlencoded", strings.NewReader(payload))
	if err != nil {
		return fmt.Errorf("exchanging code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token exchange failed: %s", string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		Scope       string `json:"scope"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("parsing token response: %w", err)
	}

	// Save credentials
	creds := &model.ShopifyCredentials{
		ShopDomain:  shopDomain,
		AccessToken: tokenResp.AccessToken,
	}
	if err := s.shopifyRepo.SaveCredentials(ctx, creds); err != nil {
		return fmt.Errorf("saving credentials: %w", err)
	}

	slog.Info("Shopify store connected", "shop_domain", shopDomain)
	return nil
}

// GetStatus returns the current Shopify integration status.
func (s *ShopifyService) GetStatus(ctx context.Context) (*model.ShopifyIntegrationStatus, error) {
	creds, err := s.shopifyRepo.GetCredentials(ctx)
	if err != nil {
		return nil, err
	}

	status := &model.ShopifyIntegrationStatus{
		Connected: creds != nil,
	}

	if creds != nil {
		status.ShopDomain = creds.ShopDomain

		// Get order count
		orders, err := s.shopifyRepo.ListOrders(ctx, nil, 0, 0)
		if err == nil {
			status.OrderCount = len(orders)
		}
	}

	return status, nil
}

// Disconnect removes the Shopify integration.
func (s *ShopifyService) Disconnect(ctx context.Context) error {
	if err := s.shopifyRepo.DeleteCredentials(ctx); err != nil {
		return fmt.Errorf("deleting credentials: %w", err)
	}
	slog.Info("Shopify store disconnected")
	return nil
}

// SyncOrders fetches orders from Shopify and stores them.
func (s *ShopifyService) SyncOrders(ctx context.Context) (*model.SyncResult, error) {
	creds, err := s.shopifyRepo.GetCredentials(ctx)
	if err != nil {
		return nil, err
	}
	if creds == nil {
		return nil, fmt.Errorf("no Shopify integration found")
	}

	result := &model.SyncResult{}

	// Fetch orders from Shopify API
	ordersURL := fmt.Sprintf("https://%s/admin/api/2024-01/orders.json?status=any&limit=250", creds.ShopDomain)
	req, err := http.NewRequestWithContext(ctx, "GET", ordersURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Shopify-Access-Token", creds.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching orders: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fetching orders failed: %s", string(body))
	}

	var ordersResp struct {
		Orders []ShopifyAPIOrder `json:"orders"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ordersResp); err != nil {
		return nil, fmt.Errorf("parsing orders: %w", err)
	}

	result.TotalFetched = len(ordersResp.Orders)

	for _, apiOrder := range ordersResp.Orders {
		// Check if order already exists
		existing, err := s.shopifyRepo.GetOrderByShopifyID(ctx, fmt.Sprintf("%d", apiOrder.ID))
		if err != nil {
			slog.Error("error checking existing order", "order_id", apiOrder.ID, "error", err)
			result.Errors++
			continue
		}

		order := s.convertAPIOrder(apiOrder, creds.ShopDomain)
		if existing != nil {
			order.ID = existing.ID
			order.OrderID = existing.OrderID
			order.CreatedAt = existing.CreatedAt
			result.Updated++
		} else {
			result.Created++
		}

		if err := s.shopifyRepo.SaveOrder(ctx, order); err != nil {
			slog.Error("error saving order", "order_id", apiOrder.ID, "error", err)
			result.Errors++
			continue
		}

		// Save order items
		for _, apiItem := range apiOrder.LineItems {
			item := s.convertAPILineItem(order.ID, apiItem)
			if err := s.shopifyRepo.SaveOrderItem(ctx, item); err != nil {
				slog.Error("error saving order item", "item_id", apiItem.ID, "error", err)
			}
		}
	}

	slog.Info("Shopify order sync complete",
		"fetched", result.TotalFetched,
		"created", result.Created,
		"updated", result.Updated,
		"errors", result.Errors)

	return result, nil
}

// ShopifyAPIOrder represents a Shopify order from the API.
type ShopifyAPIOrder struct {
	ID              int64  `json:"id"`
	OrderNumber     int    `json:"order_number"`
	Name            string `json:"name"`
	Email           string `json:"email"`
	TotalPrice      string `json:"total_price"`
	FinancialStatus string `json:"financial_status"`
	FulfillmentStatus string `json:"fulfillment_status"`
	Customer        struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Email     string `json:"email"`
	} `json:"customer"`
	LineItems []ShopifyAPILineItem `json:"line_items"`
	CreatedAt string               `json:"created_at"`
}

// ShopifyAPILineItem represents a line item from the Shopify API.
type ShopifyAPILineItem struct {
	ID        int64  `json:"id"`
	ProductID int64  `json:"product_id"`
	VariantID int64  `json:"variant_id"`
	Title     string `json:"title"`
	SKU       string `json:"sku"`
	Quantity  int    `json:"quantity"`
	Price     string `json:"price"`
}

// convertAPIOrder converts a Shopify API order to our model.
func (s *ShopifyService) convertAPIOrder(api ShopifyAPIOrder, shopDomain string) *model.ShopifyOrder {
	customerName := fmt.Sprintf("%s %s", api.Customer.FirstName, api.Customer.LastName)
	if customerName == " " {
		customerName = api.Email
	}

	totalCents := parseMoneyToCents(api.TotalPrice)

	return &model.ShopifyOrder{
		ShopifyOrderID: fmt.Sprintf("%d", api.ID),
		ShopDomain:     shopDomain,
		OrderNumber:    fmt.Sprintf("#%d", api.OrderNumber),
		CustomerName:   customerName,
		CustomerEmail:  api.Customer.Email,
		TotalCents:     totalCents,
		Status:         api.FinancialStatus,
		SyncedAt:       time.Now(),
	}
}

// convertAPILineItem converts a Shopify API line item to our model.
func (s *ShopifyService) convertAPILineItem(orderID uuid.UUID, api ShopifyAPILineItem) *model.ShopifyOrderItem {
	priceCents := parseMoneyToCents(api.Price)

	return &model.ShopifyOrderItem{
		ShopifyOrderID:    orderID,
		ShopifyLineItemID: fmt.Sprintf("%d", api.ID),
		SKU:               api.SKU,
		Title:             api.Title,
		Quantity:          api.Quantity,
		PriceCents:        priceCents,
	}
}

// parseMoneyToCents parses a money string like "19.99" to cents.
func parseMoneyToCents(money string) int {
	var cents int
	fmt.Sscanf(money, "%d", &cents)
	// Simple conversion - proper implementation would handle decimals
	if strings.Contains(money, ".") {
		parts := strings.Split(money, ".")
		var dollars, pennies int
		fmt.Sscanf(parts[0], "%d", &dollars)
		if len(parts) > 1 {
			fmt.Sscanf(parts[1], "%d", &pennies)
			if len(parts[1]) == 1 {
				pennies *= 10
			}
		}
		cents = dollars*100 + pennies
	}
	return cents
}

// normalizeShopDomain ensures the shop domain is in the correct format.
func normalizeShopDomain(domain string) string {
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimSuffix(domain, "/")
	if !strings.HasSuffix(domain, ".myshopify.com") {
		domain = domain + ".myshopify.com"
	}
	return domain
}

// ListOrders retrieves stored Shopify orders.
func (s *ShopifyService) ListOrders(ctx context.Context, processed *bool, limit, offset int) ([]model.ShopifyOrder, error) {
	orders, err := s.shopifyRepo.ListOrders(ctx, processed, limit, offset)
	if err != nil {
		return nil, err
	}

	// Load items for each order
	for i := range orders {
		items, err := s.shopifyRepo.GetOrderItems(ctx, orders[i].ID)
		if err != nil {
			slog.Warn("failed to load order items", "order_id", orders[i].ID, "error", err)
			continue
		}
		orders[i].Items = items
	}

	return orders, nil
}

// GetOrder retrieves a single Shopify order by ID.
func (s *ShopifyService) GetOrder(ctx context.Context, id uuid.UUID) (*model.ShopifyOrder, error) {
	order, err := s.shopifyRepo.GetOrderByID(ctx, id)
	if err != nil || order == nil {
		return order, err
	}

	items, err := s.shopifyRepo.GetOrderItems(ctx, order.ID)
	if err != nil {
		return nil, err
	}
	order.Items = items

	return order, nil
}

// ProcessOrder creates a unified Order from a Shopify order.
func (s *ShopifyService) ProcessOrder(ctx context.Context, shopifyOrderID uuid.UUID) (*model.Order, error) {
	shopifyOrder, err := s.GetOrder(ctx, shopifyOrderID)
	if err != nil {
		return nil, fmt.Errorf("getting order: %w", err)
	}
	if shopifyOrder == nil {
		return nil, fmt.Errorf("order not found")
	}
	if shopifyOrder.OrderID != nil {
		return nil, fmt.Errorf("order already processed")
	}

	// Convert to order items with template lookup
	var orderItems []model.OrderItem
	for _, item := range shopifyOrder.Items {
		orderItem := model.OrderItem{
			SKU:      item.SKU,
			Quantity: item.Quantity,
		}

		// Look up template by SKU
		if item.SKU != "" {
			links, err := s.shopifyRepo.GetProductTemplatesBySKU(ctx, item.SKU)
			if err == nil && len(links) > 0 {
				orderItem.TemplateID = &links[0].TemplateID
			}
		}

		orderItems = append(orderItems, orderItem)
	}

	// Create unified order
	order, err := s.orderSvc.CreateFromExternalOrder(
		ctx,
		model.OrderSourceShopify,
		shopifyOrder.ShopifyOrderID,
		shopifyOrder.CustomerName,
		shopifyOrder.CustomerEmail,
		orderItems,
	)
	if err != nil {
		return nil, fmt.Errorf("creating unified order: %w", err)
	}

	// Link Shopify order to unified order
	if err := s.shopifyRepo.UpdateOrderProcessed(ctx, shopifyOrderID, &order.ID); err != nil {
		slog.Warn("failed to link Shopify order to unified order", "shopify_order_id", shopifyOrderID, "order_id", order.ID, "error", err)
	}

	slog.Info("processed Shopify order", "shopify_order_id", shopifyOrderID, "order_id", order.ID)
	return order, nil
}

// LinkProductToTemplate links a Shopify product to a template by SKU.
func (s *ShopifyService) LinkProductToTemplate(ctx context.Context, productID string, templateID uuid.UUID, sku string) error {
	link := &model.ShopifyProductTemplate{
		ShopifyProductID: productID,
		TemplateID:       templateID,
		SKU:              sku,
	}
	return s.shopifyRepo.SaveProductTemplate(ctx, link)
}

// UnlinkProductFromTemplate removes a product-template link.
func (s *ShopifyService) UnlinkProductFromTemplate(ctx context.Context, productID string, templateID uuid.UUID) error {
	return s.shopifyRepo.DeleteProductTemplate(ctx, productID, templateID)
}
