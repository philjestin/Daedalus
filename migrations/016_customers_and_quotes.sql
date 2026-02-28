-- Customers table (reusable across quotes and orders)
CREATE TABLE IF NOT EXISTS customers (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT,
    company TEXT,
    phone TEXT,
    notes TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_customers_email ON customers(email);
CREATE INDEX IF NOT EXISTS idx_customers_name ON customers(name);

-- Add customer_id to orders for backfill
ALTER TABLE orders ADD COLUMN customer_id TEXT REFERENCES customers(id) ON DELETE SET NULL;

-- Quotes table
CREATE TABLE IF NOT EXISTS quotes (
    id TEXT PRIMARY KEY,
    quote_number TEXT NOT NULL UNIQUE,
    customer_id TEXT NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    status TEXT NOT NULL DEFAULT 'draft',  -- draft, sent, accepted, rejected, expired
    title TEXT NOT NULL,
    notes TEXT,
    valid_until TEXT,
    accepted_option_id TEXT,
    order_id TEXT REFERENCES orders(id) ON DELETE SET NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    sent_at TEXT,
    accepted_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_quotes_status ON quotes(status);
CREATE INDEX IF NOT EXISTS idx_quotes_customer ON quotes(customer_id);
CREATE INDEX IF NOT EXISTS idx_quotes_quote_number ON quotes(quote_number);

-- Quote options table
CREATE TABLE IF NOT EXISTS quote_options (
    id TEXT PRIMARY KEY,
    quote_id TEXT NOT NULL REFERENCES quotes(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    sort_order INTEGER NOT NULL DEFAULT 0,
    total_cents INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_quote_options_quote ON quote_options(quote_id);

-- Quote line items table
CREATE TABLE IF NOT EXISTS quote_line_items (
    id TEXT PRIMARY KEY,
    option_id TEXT NOT NULL REFERENCES quote_options(id) ON DELETE CASCADE,
    type TEXT NOT NULL DEFAULT 'other',  -- printing, post_processing, consulting, design, other
    description TEXT NOT NULL,
    quantity REAL NOT NULL DEFAULT 1,
    unit TEXT NOT NULL DEFAULT 'each',  -- hours, units, grams, each
    unit_price_cents INTEGER NOT NULL DEFAULT 0,
    total_cents INTEGER NOT NULL DEFAULT 0,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_quote_line_items_option ON quote_line_items(option_id);

-- Quote events table (append-only audit log)
CREATE TABLE IF NOT EXISTS quote_events (
    id TEXT PRIMARY KEY,
    quote_id TEXT NOT NULL REFERENCES quotes(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    message TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_quote_events_quote ON quote_events(quote_id);
