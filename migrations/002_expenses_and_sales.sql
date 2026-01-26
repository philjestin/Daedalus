-- Migration: 002_expenses_and_sales
-- Description: Add expense tracking, receipt parsing, and sales/profit tracking

-- Expense categories enum-like check
-- Categories: filament, parts, tools, shipping, marketplace_fees, subscription, other

-- Expenses table (accounting source of truth)
CREATE TABLE IF NOT EXISTS expenses (
    id UUID PRIMARY KEY,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    vendor VARCHAR(255) DEFAULT '',
    subtotal_cents INT NOT NULL DEFAULT 0,
    tax_cents INT NOT NULL DEFAULT 0,
    shipping_cents INT NOT NULL DEFAULT 0,
    total_cents INT NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    category VARCHAR(50) NOT NULL DEFAULT 'other',
    notes TEXT DEFAULT '',

    -- Receipt storage
    receipt_file_id UUID REFERENCES files(id),
    receipt_file_path VARCHAR(500) DEFAULT '',

    -- AI parsing metadata
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, confirmed, rejected
    raw_ocr_text TEXT DEFAULT '',
    raw_ai_response JSONB,
    confidence INT DEFAULT 0, -- 0-100

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_expenses_occurred_at ON expenses(occurred_at DESC);
CREATE INDEX idx_expenses_category ON expenses(category);
CREATE INDEX idx_expenses_status ON expenses(status);
CREATE INDEX idx_expenses_vendor ON expenses(vendor);

-- Expense line items table
CREATE TABLE IF NOT EXISTS expense_items (
    id UUID PRIMARY KEY,
    expense_id UUID NOT NULL REFERENCES expenses(id) ON DELETE CASCADE,
    description TEXT NOT NULL DEFAULT '',
    quantity DECIMAL(10,2) NOT NULL DEFAULT 1,
    unit_price_cents INT NOT NULL DEFAULT 0,
    total_price_cents INT NOT NULL DEFAULT 0,
    sku VARCHAR(255) DEFAULT '',
    vendor_item_id VARCHAR(255) DEFAULT '',
    category VARCHAR(50) NOT NULL DEFAULT 'other',

    -- Parsed attributes for filament items
    metadata JSONB DEFAULT '{}', -- {brand, material_type, color, weight_grams, diameter_mm}

    -- Matching to inventory
    matched_spool_id UUID REFERENCES material_spools(id),
    matched_material_id UUID REFERENCES materials(id),

    -- Per-item confidence
    confidence INT DEFAULT 0,

    -- Action taken
    action_taken VARCHAR(50) DEFAULT 'none', -- none, created_spool, matched_spool, skipped

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_expense_items_expense_id ON expense_items(expense_id);
CREATE INDEX idx_expense_items_category ON expense_items(category);
CREATE INDEX idx_expense_items_matched_spool ON expense_items(matched_spool_id);

-- Sales table (revenue tracking)
CREATE TABLE IF NOT EXISTS sales (
    id UUID PRIMARY KEY,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    channel VARCHAR(50) NOT NULL DEFAULT 'other', -- marketplace, etsy, website, direct, other
    platform VARCHAR(100) DEFAULT '', -- "Facebook Marketplace", "Etsy", etc.

    -- Revenue breakdown
    gross_cents INT NOT NULL DEFAULT 0,
    fees_cents INT NOT NULL DEFAULT 0, -- marketplace fees, payment processing
    shipping_charged_cents INT DEFAULT 0, -- what customer paid for shipping
    shipping_cost_cents INT DEFAULT 0, -- actual shipping label cost
    tax_collected_cents INT DEFAULT 0,
    net_cents INT NOT NULL DEFAULT 0, -- gross - fees - shipping_cost
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',

    -- Link to project/order
    project_id UUID REFERENCES projects(id),
    order_reference VARCHAR(255) DEFAULT '', -- external order ID
    customer_name VARCHAR(255) DEFAULT '',

    -- Item details
    item_description TEXT DEFAULT '',
    quantity INT NOT NULL DEFAULT 1,

    notes TEXT DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sales_occurred_at ON sales(occurred_at DESC);
CREATE INDEX idx_sales_channel ON sales(channel);
CREATE INDEX idx_sales_project_id ON sales(project_id);

-- Add expense_item_id to material_spools to track which purchase created the spool
ALTER TABLE material_spools
    ADD COLUMN IF NOT EXISTS expense_item_id UUID REFERENCES expense_items(id);

-- Add index
CREATE INDEX IF NOT EXISTS idx_spools_expense_item ON material_spools(expense_item_id);
