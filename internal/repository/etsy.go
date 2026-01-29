package repository

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/hyperion/printfarm/internal/crypto"
	"github.com/hyperion/printfarm/internal/model"
)

// EtsyRepository handles Etsy integration database operations.
type EtsyRepository struct {
	db *sql.DB
}

// NewEtsyRepository creates a new EtsyRepository.
func NewEtsyRepository(db *sql.DB) *EtsyRepository {
	return &EtsyRepository{db: db}
}

// SaveIntegration creates or updates an Etsy integration.
// Access and refresh tokens are encrypted before storage.
func (r *EtsyRepository) SaveIntegration(ctx context.Context, i *model.EtsyIntegration) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	i.CreatedAt = time.Now()
	i.UpdatedAt = time.Now()

	if i.Scopes == nil {
		i.Scopes = []string{}
	}

	// Encrypt tokens before storing
	accessToken := i.AccessToken
	refreshToken := i.RefreshToken
	if accessToken != "" {
		if encrypted, err := crypto.Encrypt(accessToken); err == nil {
			accessToken = encrypted
		} else {
			slog.Warn("failed to encrypt access token", "error", err)
		}
	}
	if refreshToken != "" {
		if encrypted, err := crypto.Encrypt(refreshToken); err == nil {
			refreshToken = encrypted
		} else {
			slog.Warn("failed to encrypt refresh token", "error", err)
		}
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO etsy_integration (id, shop_id, shop_name, user_id, access_token, refresh_token, token_expires_at, scopes, is_active, last_sync_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (shop_id) DO UPDATE SET
			shop_name = EXCLUDED.shop_name,
			user_id = EXCLUDED.user_id,
			access_token = EXCLUDED.access_token,
			refresh_token = EXCLUDED.refresh_token,
			token_expires_at = EXCLUDED.token_expires_at,
			scopes = EXCLUDED.scopes,
			is_active = EXCLUDED.is_active,
			updated_at = EXCLUDED.updated_at
	`, i.ID, i.ShopID, i.ShopName, i.UserID, accessToken, refreshToken, i.TokenExpiresAt, marshalStringArray(i.Scopes), i.IsActive, i.LastSyncAt, i.CreatedAt, i.UpdatedAt)
	return err
}

// GetIntegration retrieves the current Etsy integration (only one per install).
// Tokens are decrypted before returning.
func (r *EtsyRepository) GetIntegration(ctx context.Context) (*model.EtsyIntegration, error) {
	var i model.EtsyIntegration
	var scopesJSON []byte
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, shop_id, shop_name, user_id, access_token, refresh_token, token_expires_at, scopes, is_active, last_sync_at, created_at, updated_at
		FROM etsy_integration
		WHERE is_active = 1
		LIMIT 1
	`), &i.ID, &i.ShopID, &i.ShopName, &i.UserID, &i.AccessToken, &i.RefreshToken, &i.TokenExpiresAt, &scopesJSON, &i.IsActive, &i.LastSyncAt, &i.CreatedAt, &i.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	i.Scopes = unmarshalStringArray(scopesJSON)

	// Decrypt tokens
	if decrypted, err := crypto.Decrypt(i.AccessToken); err == nil {
		i.AccessToken = decrypted
	}
	if decrypted, err := crypto.Decrypt(i.RefreshToken); err == nil {
		i.RefreshToken = decrypted
	}

	return &i, nil
}

// UpdateTokens updates the access and refresh tokens.
// Tokens are encrypted before storage.
func (r *EtsyRepository) UpdateTokens(ctx context.Context, accessToken, refreshToken string, expiresAt time.Time) error {
	// Encrypt tokens before storing
	encAccessToken := accessToken
	encRefreshToken := refreshToken
	if accessToken != "" {
		if encrypted, err := crypto.Encrypt(accessToken); err == nil {
			encAccessToken = encrypted
		} else {
			slog.Warn("failed to encrypt access token", "error", err)
		}
	}
	if refreshToken != "" {
		if encrypted, err := crypto.Encrypt(refreshToken); err == nil {
			encRefreshToken = encrypted
		} else {
			slog.Warn("failed to encrypt refresh token", "error", err)
		}
	}

	_, err := r.db.ExecContext(ctx, `
		UPDATE etsy_integration
		SET access_token = ?, refresh_token = ?, token_expires_at = ?, updated_at = ?
		WHERE is_active = 1
	`, encAccessToken, encRefreshToken, expiresAt, time.Now())
	return err
}

// UpdateLastSync updates the last sync timestamp.
func (r *EtsyRepository) UpdateLastSync(ctx context.Context) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE etsy_integration
		SET last_sync_at = ?, updated_at = ?
		WHERE is_active = 1
	`, now, now)
	return err
}

// DeleteIntegration removes the Etsy integration.
func (r *EtsyRepository) DeleteIntegration(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM etsy_integration WHERE is_active = 1`)
	return err
}

// SaveOAuthState saves a pending OAuth state for PKCE verification.
func (r *EtsyRepository) SaveOAuthState(ctx context.Context, s *model.EtsyOAuthState) error {
	s.CreatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO etsy_oauth_states (state, code_verifier, redirect_uri, created_at)
		VALUES (?, ?, ?, ?)
	`, s.State, s.CodeVerifier, s.RedirectURI, s.CreatedAt)
	return err
}

// GetOAuthState retrieves a pending OAuth state.
func (r *EtsyRepository) GetOAuthState(ctx context.Context, state string) (*model.EtsyOAuthState, error) {
	var s model.EtsyOAuthState
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT state, code_verifier, redirect_uri, created_at
		FROM etsy_oauth_states
		WHERE state = ?
	`, state), &s.State, &s.CodeVerifier, &s.RedirectURI, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &s, err
}

// DeleteOAuthState removes a pending OAuth state.
func (r *EtsyRepository) DeleteOAuthState(ctx context.Context, state string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM etsy_oauth_states WHERE state = ?`, state)
	return err
}

// CleanupExpiredStates removes OAuth states older than 10 minutes.
func (r *EtsyRepository) CleanupExpiredStates(ctx context.Context) error {
	cutoff := time.Now().Add(-10 * time.Minute)
	_, err := r.db.ExecContext(ctx, `DELETE FROM etsy_oauth_states WHERE created_at < ?`, cutoff)
	return err
}

// ---- Receipt Methods ----

// SaveReceipt creates or updates an Etsy receipt.
func (r *EtsyRepository) SaveReceipt(ctx context.Context, receipt *model.EtsyReceipt) error {
	if receipt.ID == uuid.Nil {
		receipt.ID = uuid.New()
	}
	receipt.UpdatedAt = time.Now()
	if receipt.CreatedAt.IsZero() {
		receipt.CreatedAt = time.Now()
	}
	if receipt.SyncedAt.IsZero() {
		receipt.SyncedAt = time.Now()
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO etsy_receipts (
			id, etsy_receipt_id, etsy_shop_id, buyer_user_id, buyer_email, name, status,
			message_from_buyer, is_shipped, is_paid, is_gift, gift_message,
			grandtotal_cents, subtotal_cents, total_price_cents, total_shipping_cost_cents,
			total_tax_cost_cents, discount_cents, currency,
			shipping_name, shipping_address_first_line, shipping_address_second_line,
			shipping_city, shipping_state, shipping_zip, shipping_country_code,
			create_timestamp, update_timestamp, is_processed, project_id,
			synced_at, created_at, updated_at
		) VALUES (
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
		)
		ON CONFLICT (etsy_receipt_id) DO UPDATE SET
			status = EXCLUDED.status,
			is_shipped = EXCLUDED.is_shipped,
			is_paid = EXCLUDED.is_paid,
			update_timestamp = EXCLUDED.update_timestamp,
			synced_at = EXCLUDED.synced_at,
			updated_at = EXCLUDED.updated_at
	`, receipt.ID, receipt.EtsyReceiptID, receipt.EtsyShopID, receipt.BuyerUserID, receipt.BuyerEmail,
		receipt.Name, receipt.Status, receipt.MessageFromBuyer, receipt.IsShipped, receipt.IsPaid,
		receipt.IsGift, receipt.GiftMessage, receipt.GrandtotalCents, receipt.SubtotalCents,
		receipt.TotalPriceCents, receipt.TotalShippingCostCents, receipt.TotalTaxCostCents,
		receipt.DiscountCents, receipt.Currency, receipt.ShippingName, receipt.ShippingAddressFirstLine,
		receipt.ShippingAddressSecondLine, receipt.ShippingCity, receipt.ShippingState,
		receipt.ShippingZip, receipt.ShippingCountryCode, receipt.CreateTimestamp,
		receipt.UpdateTimestamp, receipt.IsProcessed, receipt.ProjectID, receipt.SyncedAt,
		receipt.CreatedAt, receipt.UpdatedAt)
	return err
}

// GetReceiptByEtsyID retrieves a receipt by its Etsy receipt ID.
func (r *EtsyRepository) GetReceiptByEtsyID(ctx context.Context, etsyReceiptID int64) (*model.EtsyReceipt, error) {
	var receipt model.EtsyReceipt
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, etsy_receipt_id, etsy_shop_id, buyer_user_id, buyer_email, name, status,
			message_from_buyer, is_shipped, is_paid, is_gift, gift_message,
			grandtotal_cents, subtotal_cents, total_price_cents, total_shipping_cost_cents,
			total_tax_cost_cents, discount_cents, currency,
			shipping_name, shipping_address_first_line, shipping_address_second_line,
			shipping_city, shipping_state, shipping_zip, shipping_country_code,
			create_timestamp, update_timestamp, is_processed, project_id,
			synced_at, created_at, updated_at
		FROM etsy_receipts
		WHERE etsy_receipt_id = ?
	`, etsyReceiptID),
		&receipt.ID, &receipt.EtsyReceiptID, &receipt.EtsyShopID, &receipt.BuyerUserID,
		&receipt.BuyerEmail, &receipt.Name, &receipt.Status, &receipt.MessageFromBuyer,
		&receipt.IsShipped, &receipt.IsPaid, &receipt.IsGift, &receipt.GiftMessage,
		&receipt.GrandtotalCents, &receipt.SubtotalCents, &receipt.TotalPriceCents,
		&receipt.TotalShippingCostCents, &receipt.TotalTaxCostCents, &receipt.DiscountCents,
		&receipt.Currency, &receipt.ShippingName, &receipt.ShippingAddressFirstLine,
		&receipt.ShippingAddressSecondLine, &receipt.ShippingCity, &receipt.ShippingState,
		&receipt.ShippingZip, &receipt.ShippingCountryCode, &receipt.CreateTimestamp,
		&receipt.UpdateTimestamp, &receipt.IsProcessed, &receipt.ProjectID,
		&receipt.SyncedAt, &receipt.CreatedAt, &receipt.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &receipt, err
}

// GetReceiptByID retrieves a receipt by its internal UUID.
func (r *EtsyRepository) GetReceiptByID(ctx context.Context, id uuid.UUID) (*model.EtsyReceipt, error) {
	var receipt model.EtsyReceipt
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, etsy_receipt_id, etsy_shop_id, buyer_user_id, buyer_email, name, status,
			message_from_buyer, is_shipped, is_paid, is_gift, gift_message,
			grandtotal_cents, subtotal_cents, total_price_cents, total_shipping_cost_cents,
			total_tax_cost_cents, discount_cents, currency,
			shipping_name, shipping_address_first_line, shipping_address_second_line,
			shipping_city, shipping_state, shipping_zip, shipping_country_code,
			create_timestamp, update_timestamp, is_processed, project_id,
			synced_at, created_at, updated_at
		FROM etsy_receipts
		WHERE id = ?
	`, id),
		&receipt.ID, &receipt.EtsyReceiptID, &receipt.EtsyShopID, &receipt.BuyerUserID,
		&receipt.BuyerEmail, &receipt.Name, &receipt.Status, &receipt.MessageFromBuyer,
		&receipt.IsShipped, &receipt.IsPaid, &receipt.IsGift, &receipt.GiftMessage,
		&receipt.GrandtotalCents, &receipt.SubtotalCents, &receipt.TotalPriceCents,
		&receipt.TotalShippingCostCents, &receipt.TotalTaxCostCents, &receipt.DiscountCents,
		&receipt.Currency, &receipt.ShippingName, &receipt.ShippingAddressFirstLine,
		&receipt.ShippingAddressSecondLine, &receipt.ShippingCity, &receipt.ShippingState,
		&receipt.ShippingZip, &receipt.ShippingCountryCode, &receipt.CreateTimestamp,
		&receipt.UpdateTimestamp, &receipt.IsProcessed, &receipt.ProjectID,
		&receipt.SyncedAt, &receipt.CreatedAt, &receipt.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &receipt, err
}

// ListReceipts retrieves receipts with optional filtering.
func (r *EtsyRepository) ListReceipts(ctx context.Context, processed *bool, limit, offset int) ([]model.EtsyReceipt, error) {
	query := `
		SELECT id, etsy_receipt_id, etsy_shop_id, buyer_user_id, buyer_email, name, status,
			message_from_buyer, is_shipped, is_paid, is_gift, gift_message,
			grandtotal_cents, subtotal_cents, total_price_cents, total_shipping_cost_cents,
			total_tax_cost_cents, discount_cents, currency,
			shipping_name, shipping_address_first_line, shipping_address_second_line,
			shipping_city, shipping_state, shipping_zip, shipping_country_code,
			create_timestamp, update_timestamp, is_processed, project_id,
			synced_at, created_at, updated_at
		FROM etsy_receipts
	`
	var args []interface{}

	if processed != nil {
		query += " WHERE is_processed = ?"
		args = append(args, *processed)
	}

	query += " ORDER BY create_timestamp DESC"

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

	var receipts []model.EtsyReceipt
	for rows.Next() {
		var receipt model.EtsyReceipt
		err := scanRow(rows,
			&receipt.ID, &receipt.EtsyReceiptID, &receipt.EtsyShopID, &receipt.BuyerUserID,
			&receipt.BuyerEmail, &receipt.Name, &receipt.Status, &receipt.MessageFromBuyer,
			&receipt.IsShipped, &receipt.IsPaid, &receipt.IsGift, &receipt.GiftMessage,
			&receipt.GrandtotalCents, &receipt.SubtotalCents, &receipt.TotalPriceCents,
			&receipt.TotalShippingCostCents, &receipt.TotalTaxCostCents, &receipt.DiscountCents,
			&receipt.Currency, &receipt.ShippingName, &receipt.ShippingAddressFirstLine,
			&receipt.ShippingAddressSecondLine, &receipt.ShippingCity, &receipt.ShippingState,
			&receipt.ShippingZip, &receipt.ShippingCountryCode, &receipt.CreateTimestamp,
			&receipt.UpdateTimestamp, &receipt.IsProcessed, &receipt.ProjectID,
			&receipt.SyncedAt, &receipt.CreatedAt, &receipt.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		receipts = append(receipts, receipt)
	}

	return receipts, rows.Err()
}

// UpdateReceiptProcessed marks a receipt as processed.
func (r *EtsyRepository) UpdateReceiptProcessed(ctx context.Context, id uuid.UUID, projectID *uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE etsy_receipts
		SET is_processed = 1, project_id = ?, updated_at = ?
		WHERE id = ?
	`, projectID, time.Now(), id)
	return err
}

// ---- Receipt Item Methods ----

// SaveReceiptItem creates or updates a receipt item.
func (r *EtsyRepository) SaveReceiptItem(ctx context.Context, item *model.EtsyReceiptItem) error {
	if item.ID == uuid.Nil {
		item.ID = uuid.New()
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO etsy_receipt_items (
			id, etsy_receipt_item_id, receipt_id, etsy_listing_id, etsy_transaction_id,
			title, description, quantity, price_cents, shipping_cost_cents,
			sku, variations, is_digital, template_id, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (etsy_receipt_item_id) DO UPDATE SET
			quantity = EXCLUDED.quantity,
			sku = EXCLUDED.sku,
			template_id = EXCLUDED.template_id
	`, item.ID, item.EtsyReceiptItemID, item.ReceiptID, item.EtsyListingID,
		item.EtsyTransactionID, item.Title, item.Description, item.Quantity,
		item.PriceCents, item.ShippingCostCents, item.SKU, item.Variations,
		item.IsDigital, item.TemplateID, item.CreatedAt)
	return err
}

// GetReceiptItems retrieves all items for a receipt.
func (r *EtsyRepository) GetReceiptItems(ctx context.Context, receiptID uuid.UUID) ([]model.EtsyReceiptItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, etsy_receipt_item_id, receipt_id, etsy_listing_id, etsy_transaction_id,
			title, description, quantity, price_cents, shipping_cost_cents,
			sku, variations, is_digital, template_id, created_at
		FROM etsy_receipt_items
		WHERE receipt_id = ?
		ORDER BY created_at
	`, receiptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.EtsyReceiptItem
	for rows.Next() {
		var item model.EtsyReceiptItem
		err := scanRow(rows,
			&item.ID, &item.EtsyReceiptItemID, &item.ReceiptID, &item.EtsyListingID,
			&item.EtsyTransactionID, &item.Title, &item.Description, &item.Quantity,
			&item.PriceCents, &item.ShippingCostCents, &item.SKU, &item.Variations,
			&item.IsDigital, &item.TemplateID, &item.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

// ---- Sync State Methods ----

// GetSyncState retrieves the sync state for a shop.
func (r *EtsyRepository) GetSyncState(ctx context.Context, shopID int64) (*model.EtsySyncState, error) {
	var state model.EtsySyncState
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, shop_id, last_receipt_sync_at, last_listing_sync_at, created_at, updated_at
		FROM etsy_sync_state
		WHERE shop_id = ?
	`, shopID), &state.ID, &state.ShopID, &state.LastReceiptSyncAt,
		&state.LastListingSyncAt, &state.CreatedAt, &state.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &state, err
}

// SaveSyncState creates or updates the sync state for a shop.
func (r *EtsyRepository) SaveSyncState(ctx context.Context, state *model.EtsySyncState) error {
	if state.ID == uuid.Nil {
		state.ID = uuid.New()
	}
	state.UpdatedAt = time.Now()
	if state.CreatedAt.IsZero() {
		state.CreatedAt = time.Now()
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO etsy_sync_state (id, shop_id, last_receipt_sync_at, last_listing_sync_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (shop_id) DO UPDATE SET
			last_receipt_sync_at = EXCLUDED.last_receipt_sync_at,
			last_listing_sync_at = EXCLUDED.last_listing_sync_at,
			updated_at = EXCLUDED.updated_at
	`, state.ID, state.ShopID, state.LastReceiptSyncAt, state.LastListingSyncAt,
		state.CreatedAt, state.UpdatedAt)
	return err
}

// ---- Listing Methods ----

// SaveListing creates or updates an Etsy listing.
func (r *EtsyRepository) SaveListing(ctx context.Context, listing *model.EtsyListing) error {
	if listing.ID == uuid.Nil {
		listing.ID = uuid.New()
	}
	listing.UpdatedAt = time.Now()
	if listing.CreatedAt.IsZero() {
		listing.CreatedAt = time.Now()
	}
	if listing.SyncedAt.IsZero() {
		listing.SyncedAt = time.Now()
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO etsy_listings (
			id, etsy_listing_id, etsy_shop_id, title, description, state, quantity,
			url, views, num_favorers, is_customizable, is_personalizable, tags,
			has_variations, price_cents, currency, skus, synced_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (etsy_listing_id) DO UPDATE SET
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			state = EXCLUDED.state,
			quantity = EXCLUDED.quantity,
			url = EXCLUDED.url,
			views = EXCLUDED.views,
			num_favorers = EXCLUDED.num_favorers,
			has_variations = EXCLUDED.has_variations,
			price_cents = EXCLUDED.price_cents,
			skus = EXCLUDED.skus,
			synced_at = EXCLUDED.synced_at,
			updated_at = EXCLUDED.updated_at
	`, listing.ID, listing.EtsyListingID, listing.EtsyShopID, listing.Title,
		listing.Description, listing.State, listing.Quantity, listing.URL,
		listing.Views, listing.NumFavorers, listing.IsCustomizable, listing.IsPersonalizable,
		listing.Tags, listing.HasVariations, listing.PriceCents, listing.Currency,
		listing.SKUs, listing.SyncedAt, listing.CreatedAt, listing.UpdatedAt)
	return err
}

// GetListingByEtsyID retrieves a listing by its Etsy listing ID.
func (r *EtsyRepository) GetListingByEtsyID(ctx context.Context, etsyListingID int64) (*model.EtsyListing, error) {
	var listing model.EtsyListing
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, etsy_listing_id, etsy_shop_id, title, description, state, quantity,
			url, views, num_favorers, is_customizable, is_personalizable, tags,
			has_variations, price_cents, currency, skus, synced_at, created_at, updated_at
		FROM etsy_listings
		WHERE etsy_listing_id = ?
	`, etsyListingID),
		&listing.ID, &listing.EtsyListingID, &listing.EtsyShopID, &listing.Title,
		&listing.Description, &listing.State, &listing.Quantity, &listing.URL,
		&listing.Views, &listing.NumFavorers, &listing.IsCustomizable, &listing.IsPersonalizable,
		&listing.Tags, &listing.HasVariations, &listing.PriceCents, &listing.Currency,
		&listing.SKUs, &listing.SyncedAt, &listing.CreatedAt, &listing.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &listing, err
}

// GetListingByID retrieves a listing by its internal UUID.
func (r *EtsyRepository) GetListingByID(ctx context.Context, id uuid.UUID) (*model.EtsyListing, error) {
	var listing model.EtsyListing
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, etsy_listing_id, etsy_shop_id, title, description, state, quantity,
			url, views, num_favorers, is_customizable, is_personalizable, tags,
			has_variations, price_cents, currency, skus, synced_at, created_at, updated_at
		FROM etsy_listings
		WHERE id = ?
	`, id),
		&listing.ID, &listing.EtsyListingID, &listing.EtsyShopID, &listing.Title,
		&listing.Description, &listing.State, &listing.Quantity, &listing.URL,
		&listing.Views, &listing.NumFavorers, &listing.IsCustomizable, &listing.IsPersonalizable,
		&listing.Tags, &listing.HasVariations, &listing.PriceCents, &listing.Currency,
		&listing.SKUs, &listing.SyncedAt, &listing.CreatedAt, &listing.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &listing, err
}

// ListListings retrieves listings with optional state filtering.
func (r *EtsyRepository) ListListings(ctx context.Context, state string, limit, offset int) ([]model.EtsyListing, error) {
	query := `
		SELECT id, etsy_listing_id, etsy_shop_id, title, description, state, quantity,
			url, views, num_favorers, is_customizable, is_personalizable, tags,
			has_variations, price_cents, currency, skus, synced_at, created_at, updated_at
		FROM etsy_listings
	`
	var args []interface{}

	if state != "" {
		query += " WHERE state = ?"
		args = append(args, state)
	}

	query += " ORDER BY title"

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

	var listings []model.EtsyListing
	for rows.Next() {
		var listing model.EtsyListing
		err := scanRow(rows,
			&listing.ID, &listing.EtsyListingID, &listing.EtsyShopID, &listing.Title,
			&listing.Description, &listing.State, &listing.Quantity, &listing.URL,
			&listing.Views, &listing.NumFavorers, &listing.IsCustomizable, &listing.IsPersonalizable,
			&listing.Tags, &listing.HasVariations, &listing.PriceCents, &listing.Currency,
			&listing.SKUs, &listing.SyncedAt, &listing.CreatedAt, &listing.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		listings = append(listings, listing)
	}

	return listings, rows.Err()
}

// ---- Listing Template Link Methods ----

// SaveListingTemplate creates or updates a listing-template link.
func (r *EtsyRepository) SaveListingTemplate(ctx context.Context, link *model.EtsyListingTemplate) error {
	if link.ID == uuid.Nil {
		link.ID = uuid.New()
	}
	if link.CreatedAt.IsZero() {
		link.CreatedAt = time.Now()
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO etsy_listing_templates (id, etsy_listing_id, template_id, sku, sync_inventory, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (etsy_listing_id, template_id) DO UPDATE SET
			sku = EXCLUDED.sku,
			sync_inventory = EXCLUDED.sync_inventory
	`, link.ID, link.EtsyListingID, link.TemplateID, link.SKU, link.SyncInventory, link.CreatedAt)
	return err
}

// GetListingTemplate retrieves a listing-template link.
func (r *EtsyRepository) GetListingTemplate(ctx context.Context, etsyListingID int64, templateID uuid.UUID) (*model.EtsyListingTemplate, error) {
	var link model.EtsyListingTemplate
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, etsy_listing_id, template_id, sku, sync_inventory, created_at
		FROM etsy_listing_templates
		WHERE etsy_listing_id = ? AND template_id = ?
	`, etsyListingID, templateID),
		&link.ID, &link.EtsyListingID, &link.TemplateID, &link.SKU,
		&link.SyncInventory, &link.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &link, err
}

// GetListingTemplatesBySKU retrieves listing-template links by SKU.
func (r *EtsyRepository) GetListingTemplatesBySKU(ctx context.Context, sku string) ([]model.EtsyListingTemplate, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, etsy_listing_id, template_id, sku, sync_inventory, created_at
		FROM etsy_listing_templates
		WHERE sku = ?
	`, sku)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []model.EtsyListingTemplate
	for rows.Next() {
		var link model.EtsyListingTemplate
		err := scanRow(rows,
			&link.ID, &link.EtsyListingID, &link.TemplateID, &link.SKU,
			&link.SyncInventory, &link.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		links = append(links, link)
	}

	return links, rows.Err()
}

// GetTemplatesForListing retrieves all templates linked to a listing.
func (r *EtsyRepository) GetTemplatesForListing(ctx context.Context, etsyListingID int64) ([]model.EtsyListingTemplate, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, etsy_listing_id, template_id, sku, sync_inventory, created_at
		FROM etsy_listing_templates
		WHERE etsy_listing_id = ?
	`, etsyListingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []model.EtsyListingTemplate
	for rows.Next() {
		var link model.EtsyListingTemplate
		err := scanRow(rows,
			&link.ID, &link.EtsyListingID, &link.TemplateID, &link.SKU,
			&link.SyncInventory, &link.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		links = append(links, link)
	}

	return links, rows.Err()
}

// DeleteListingTemplate removes a listing-template link.
func (r *EtsyRepository) DeleteListingTemplate(ctx context.Context, etsyListingID int64, templateID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM etsy_listing_templates
		WHERE etsy_listing_id = ? AND template_id = ?
	`, etsyListingID, templateID)
	return err
}

// ---- Webhook Event Methods ----

// SaveWebhookEvent creates a new webhook event.
func (r *EtsyRepository) SaveWebhookEvent(ctx context.Context, event *model.EtsyWebhookEvent) error {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.ReceivedAt.IsZero() {
		event.ReceivedAt = time.Now()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO etsy_webhook_events (
			id, event_type, resource_type, resource_id, shop_id, payload,
			signature, processed, processed_at, error, received_at, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, event.ID, event.EventType, event.ResourceType, event.ResourceID,
		event.ShopID, event.Payload, event.Signature, event.Processed,
		event.ProcessedAt, event.Error, event.ReceivedAt, event.CreatedAt)
	return err
}

// GetWebhookEventByID retrieves a webhook event by ID.
func (r *EtsyRepository) GetWebhookEventByID(ctx context.Context, id uuid.UUID) (*model.EtsyWebhookEvent, error) {
	var event model.EtsyWebhookEvent
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, event_type, resource_type, resource_id, shop_id, payload,
			signature, processed, processed_at, error, received_at, created_at
		FROM etsy_webhook_events
		WHERE id = ?
	`, id),
		&event.ID, &event.EventType, &event.ResourceType, &event.ResourceID,
		&event.ShopID, &event.Payload, &event.Signature, &event.Processed,
		&event.ProcessedAt, &event.Error, &event.ReceivedAt, &event.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &event, err
}

// ListWebhookEvents retrieves webhook events with optional filtering.
func (r *EtsyRepository) ListWebhookEvents(ctx context.Context, eventType string, limit, offset int) ([]model.EtsyWebhookEvent, error) {
	query := `
		SELECT id, event_type, resource_type, resource_id, shop_id, payload,
			signature, processed, processed_at, error, received_at, created_at
		FROM etsy_webhook_events
	`
	var args []interface{}

	if eventType != "" {
		query += " WHERE event_type = ?"
		args = append(args, eventType)
	}

	query += " ORDER BY received_at DESC"

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

	var events []model.EtsyWebhookEvent
	for rows.Next() {
		var event model.EtsyWebhookEvent
		err := scanRow(rows,
			&event.ID, &event.EventType, &event.ResourceType, &event.ResourceID,
			&event.ShopID, &event.Payload, &event.Signature, &event.Processed,
			&event.ProcessedAt, &event.Error, &event.ReceivedAt, &event.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

// UpdateWebhookEventProcessed marks a webhook event as processed.
func (r *EtsyRepository) UpdateWebhookEventProcessed(ctx context.Context, id uuid.UUID, errorMsg string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE etsy_webhook_events
		SET processed = 1, processed_at = ?, error = ?
		WHERE id = ?
	`, now, errorMsg, id)
	return err
}
