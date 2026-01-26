-- Etsy Listings
CREATE TABLE IF NOT EXISTS etsy_listings (
    id UUID PRIMARY KEY,
    etsy_listing_id BIGINT UNIQUE NOT NULL,
    etsy_shop_id BIGINT NOT NULL,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    state VARCHAR(50) NOT NULL,
    quantity INT NOT NULL DEFAULT 0,
    url VARCHAR(500),
    views INT DEFAULT 0,
    num_favorers INT DEFAULT 0,
    is_customizable BOOLEAN DEFAULT false,
    is_personalizable BOOLEAN DEFAULT false,
    tags JSONB DEFAULT '[]',
    has_variations BOOLEAN DEFAULT false,
    price_cents INT,
    currency VARCHAR(10) DEFAULT 'USD',
    skus JSONB DEFAULT '[]',
    synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_etsy_listings_etsy_id ON etsy_listings(etsy_listing_id);
CREATE INDEX idx_etsy_listings_state ON etsy_listings(state);

-- Etsy Listing to Template mapping
CREATE TABLE IF NOT EXISTS etsy_listing_templates (
    id UUID PRIMARY KEY,
    etsy_listing_id BIGINT NOT NULL REFERENCES etsy_listings(etsy_listing_id) ON DELETE CASCADE,
    template_id UUID NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
    sku VARCHAR(255),
    sync_inventory BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(etsy_listing_id, template_id)
);

CREATE INDEX idx_etsy_listing_templates_sku ON etsy_listing_templates(sku);
