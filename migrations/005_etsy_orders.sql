-- Etsy Receipts (Orders)
CREATE TABLE IF NOT EXISTS etsy_receipts (
    id UUID PRIMARY KEY,
    etsy_receipt_id BIGINT UNIQUE NOT NULL,
    etsy_shop_id BIGINT NOT NULL,
    buyer_user_id BIGINT,
    buyer_email VARCHAR(255),
    name VARCHAR(255),
    status VARCHAR(50) NOT NULL,
    message_from_buyer TEXT,
    is_shipped BOOLEAN DEFAULT false,
    is_paid BOOLEAN DEFAULT false,
    is_gift BOOLEAN DEFAULT false,
    gift_message TEXT,
    grandtotal_cents INT NOT NULL,
    subtotal_cents INT NOT NULL,
    total_price_cents INT NOT NULL,
    total_shipping_cost_cents INT DEFAULT 0,
    total_tax_cost_cents INT DEFAULT 0,
    discount_cents INT DEFAULT 0,
    currency VARCHAR(10) DEFAULT 'USD',
    shipping_name VARCHAR(255),
    shipping_address_first_line VARCHAR(255),
    shipping_address_second_line VARCHAR(255),
    shipping_city VARCHAR(255),
    shipping_state VARCHAR(255),
    shipping_zip VARCHAR(50),
    shipping_country_code VARCHAR(10),
    create_timestamp TIMESTAMPTZ,
    update_timestamp TIMESTAMPTZ,
    is_processed BOOLEAN DEFAULT false,
    project_id UUID REFERENCES projects(id),
    synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_etsy_receipts_etsy_id ON etsy_receipts(etsy_receipt_id);
CREATE INDEX idx_etsy_receipts_processed ON etsy_receipts(is_processed);

-- Etsy Receipt Items (Line Items)
CREATE TABLE IF NOT EXISTS etsy_receipt_items (
    id UUID PRIMARY KEY,
    etsy_receipt_item_id BIGINT UNIQUE NOT NULL,
    receipt_id UUID NOT NULL REFERENCES etsy_receipts(id) ON DELETE CASCADE,
    etsy_listing_id BIGINT NOT NULL,
    etsy_transaction_id BIGINT NOT NULL,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    quantity INT NOT NULL DEFAULT 1,
    price_cents INT NOT NULL,
    shipping_cost_cents INT DEFAULT 0,
    sku VARCHAR(255),
    variations JSONB DEFAULT '[]',
    is_digital BOOLEAN DEFAULT false,
    template_id UUID REFERENCES templates(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_etsy_receipt_items_receipt ON etsy_receipt_items(receipt_id);
CREATE INDEX idx_etsy_receipt_items_sku ON etsy_receipt_items(sku);

-- Sync state tracking
CREATE TABLE IF NOT EXISTS etsy_sync_state (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id BIGINT UNIQUE NOT NULL,
    last_receipt_sync_at TIMESTAMPTZ,
    last_listing_sync_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
