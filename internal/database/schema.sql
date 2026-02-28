-- Daedalus SQLite Schema
-- Consolidated from PostgreSQL migrations 001-013

-- Files table (content-addressed storage)
CREATE TABLE IF NOT EXISTS files (
    id TEXT PRIMARY KEY,
    hash TEXT NOT NULL UNIQUE,
    original_name TEXT NOT NULL,
    content_type TEXT DEFAULT 'application/octet-stream',
    size_bytes INTEGER NOT NULL,
    storage_path TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_files_hash ON files(hash);

-- Materials table (catalog)
CREATE TABLE IF NOT EXISTS materials (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    manufacturer TEXT DEFAULT '',
    color TEXT DEFAULT '',
    color_hex TEXT DEFAULT '',
    density REAL DEFAULT 1.24,
    cost_per_kg REAL DEFAULT 0,
    print_temp TEXT,
    bed_temp TEXT,
    notes TEXT DEFAULT '',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_materials_type ON materials(type);

-- Printers table
CREATE TABLE IF NOT EXISTS printers (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    model TEXT DEFAULT '',
    manufacturer TEXT DEFAULT '',
    connection_type TEXT NOT NULL DEFAULT 'manual',
    connection_uri TEXT DEFAULT '',
    api_key TEXT DEFAULT '',
    serial_number TEXT DEFAULT '',
    status TEXT NOT NULL DEFAULT 'offline',
    build_volume TEXT,
    nozzle_diameter REAL DEFAULT 0.4,
    location TEXT DEFAULT '',
    notes TEXT DEFAULT '',
    min_material_percent INTEGER DEFAULT 10,
    cost_per_hour_cents INTEGER DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_printers_status ON printers(status);

-- Templates/Recipes table
CREATE TABLE IF NOT EXISTS templates (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    sku TEXT UNIQUE,
    tags TEXT DEFAULT '[]',
    material_type TEXT NOT NULL,
    estimated_material_grams REAL DEFAULT 0,
    preferred_printer_id TEXT REFERENCES printers(id),
    allow_any_printer INTEGER DEFAULT 0,
    quantity_per_order INTEGER NOT NULL DEFAULT 1,
    post_process_checklist TEXT DEFAULT '[]',
    is_active INTEGER DEFAULT 1,
    printer_constraints TEXT DEFAULT '{}',
    print_profile TEXT DEFAULT 'standard',
    estimated_print_seconds INTEGER DEFAULT 0,
    labor_minutes INTEGER NOT NULL DEFAULT 0,
    sale_price_cents INTEGER NOT NULL DEFAULT 0,
    material_cost_per_gram_cents INTEGER NOT NULL DEFAULT 0,
    version INTEGER DEFAULT 1,
    archived_at TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_templates_sku ON templates(sku);
CREATE INDEX IF NOT EXISTS idx_templates_active ON templates(is_active);

-- Projects table (Product Catalog - extended with template-like fields)
CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    target_date TEXT,
    tags TEXT DEFAULT '[]',
    template_id TEXT REFERENCES templates(id),  -- Legacy: kept for migration
    source TEXT DEFAULT 'manual',
    external_order_id TEXT,
    customer_notes TEXT DEFAULT '',
    -- Template-like fields for product catalog
    sku TEXT,
    price_cents INTEGER,
    printer_type TEXT,
    allowed_printer_ids TEXT DEFAULT '[]',  -- JSON array of printer UUIDs
    default_settings TEXT DEFAULT '{}',     -- JSON object for print settings
    notes TEXT DEFAULT '',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_projects_updated_at ON projects(updated_at);
CREATE INDEX IF NOT EXISTS idx_projects_template ON projects(template_id);
CREATE INDEX IF NOT EXISTS idx_projects_external_order ON projects(external_order_id);
CREATE INDEX IF NOT EXISTS idx_projects_sku ON projects(sku);

-- Tasks table (Work Instances - created when processing orders)
CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id),
    order_id TEXT REFERENCES orders(id),
    order_item_id TEXT REFERENCES order_items(id),
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',  -- pending, in_progress, completed, cancelled
    quantity INTEGER NOT NULL DEFAULT 1,
    notes TEXT DEFAULT '',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at TEXT,
    completed_at TEXT,
    pickup_date TEXT
);

CREATE INDEX IF NOT EXISTS idx_tasks_project ON tasks(project_id);
CREATE INDEX IF NOT EXISTS idx_tasks_order ON tasks(order_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);

-- Task checklist items
CREATE TABLE IF NOT EXISTS task_checklist_items (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    part_id TEXT REFERENCES parts(id),
    sort_order INTEGER NOT NULL DEFAULT 0,
    completed INTEGER NOT NULL DEFAULT 0,
    completed_at TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_task_checklist_task ON task_checklist_items(task_id);

-- Parts table
CREATE TABLE IF NOT EXISTS parts (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    quantity INTEGER NOT NULL DEFAULT 1,
    status TEXT NOT NULL DEFAULT 'design',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_parts_project_id ON parts(project_id);

-- Designs table (versioned, immutable)
CREATE TABLE IF NOT EXISTS designs (
    id TEXT PRIMARY KEY,
    part_id TEXT NOT NULL REFERENCES parts(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    file_id TEXT NOT NULL REFERENCES files(id),
    file_name TEXT NOT NULL,
    file_hash TEXT NOT NULL,
    file_size_bytes INTEGER NOT NULL,
    file_type TEXT NOT NULL,
    notes TEXT DEFAULT '',
    slice_profile TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(part_id, version)
);

CREATE INDEX IF NOT EXISTS idx_designs_part_id ON designs(part_id);

-- Template to Design mapping (many-to-many)
CREATE TABLE IF NOT EXISTS template_designs (
    id TEXT PRIMARY KEY,
    template_id TEXT NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
    design_id TEXT NOT NULL REFERENCES designs(id),
    is_primary INTEGER DEFAULT 0,
    quantity INTEGER NOT NULL DEFAULT 1,
    sequence_order INTEGER DEFAULT 0,
    notes TEXT DEFAULT '',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(template_id, design_id)
);

CREATE INDEX IF NOT EXISTS idx_template_designs_template ON template_designs(template_id);

-- Material spools table (inventory)
CREATE TABLE IF NOT EXISTS material_spools (
    id TEXT PRIMARY KEY,
    material_id TEXT NOT NULL REFERENCES materials(id),
    initial_weight REAL NOT NULL DEFAULT 1000,
    remaining_weight REAL NOT NULL DEFAULT 1000,
    purchase_date TEXT,
    purchase_cost REAL DEFAULT 0,
    location TEXT DEFAULT '',
    status TEXT NOT NULL DEFAULT 'new',
    notes TEXT DEFAULT '',
    expense_item_id TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_spools_material_id ON material_spools(material_id);
CREATE INDEX IF NOT EXISTS idx_spools_status ON material_spools(status);
CREATE INDEX IF NOT EXISTS idx_spools_expense_item ON material_spools(expense_item_id);

-- Print jobs table
CREATE TABLE IF NOT EXISTS print_jobs (
    id TEXT PRIMARY KEY,
    design_id TEXT NOT NULL REFERENCES designs(id),
    printer_id TEXT REFERENCES printers(id),
    material_spool_id TEXT REFERENCES material_spools(id),
    project_id TEXT REFERENCES projects(id),
    task_id TEXT REFERENCES tasks(id),  -- Link to task (work instance)
    status TEXT NOT NULL DEFAULT 'queued',
    progress REAL DEFAULT 0,
    started_at TEXT,
    completed_at TEXT,
    outcome TEXT,
    notes TEXT DEFAULT '',
    recipe_id TEXT REFERENCES templates(id),
    attempt_number INTEGER NOT NULL DEFAULT 1,
    parent_job_id TEXT REFERENCES print_jobs(id),
    failure_category TEXT,
    estimated_seconds INTEGER,
    actual_seconds INTEGER,
    material_used_grams REAL,
    cost_cents INTEGER,
    printer_time_cost_cents INTEGER,
    material_cost_cents INTEGER,
    material_snapshot TEXT,
    priority INTEGER NOT NULL DEFAULT 0,
    auto_dispatch_enabled INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_print_jobs_design_id ON print_jobs(design_id);
CREATE INDEX IF NOT EXISTS idx_print_jobs_printer_id ON print_jobs(printer_id);
CREATE INDEX IF NOT EXISTS idx_print_jobs_status ON print_jobs(status);
CREATE INDEX IF NOT EXISTS idx_print_jobs_created_at ON print_jobs(created_at);
CREATE INDEX IF NOT EXISTS idx_print_jobs_parent ON print_jobs(parent_job_id);
CREATE INDEX IF NOT EXISTS idx_print_jobs_recipe ON print_jobs(recipe_id);
CREATE INDEX IF NOT EXISTS idx_print_jobs_project ON print_jobs(project_id);
CREATE INDEX IF NOT EXISTS idx_print_jobs_task ON print_jobs(task_id);
CREATE INDEX IF NOT EXISTS idx_print_jobs_material_spool ON print_jobs(material_spool_id);

-- Dispatch requests table (for auto-dispatch confirmation)
CREATE TABLE IF NOT EXISTS dispatch_requests (
    id TEXT PRIMARY KEY,
    job_id TEXT NOT NULL REFERENCES print_jobs(id) ON DELETE CASCADE,
    printer_id TEXT NOT NULL REFERENCES printers(id),
    status TEXT NOT NULL DEFAULT 'pending',  -- pending, confirmed, rejected, expired
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TEXT NOT NULL,
    responded_at TEXT,
    reason TEXT
);
CREATE INDEX IF NOT EXISTS idx_dispatch_requests_status ON dispatch_requests(status);
CREATE INDEX IF NOT EXISTS idx_dispatch_requests_printer ON dispatch_requests(printer_id);
CREATE INDEX IF NOT EXISTS idx_dispatch_requests_job ON dispatch_requests(job_id);

-- Auto-dispatch settings per printer
CREATE TABLE IF NOT EXISTS auto_dispatch_settings (
    printer_id TEXT PRIMARY KEY REFERENCES printers(id) ON DELETE CASCADE,
    enabled INTEGER NOT NULL DEFAULT 0,
    require_confirmation INTEGER NOT NULL DEFAULT 1,
    auto_start INTEGER NOT NULL DEFAULT 0,
    timeout_minutes INTEGER NOT NULL DEFAULT 30,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Job events table (append-only, immutable audit log)
CREATE TABLE IF NOT EXISTS job_events (
    id TEXT PRIMARY KEY,
    job_id TEXT NOT NULL REFERENCES print_jobs(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    occurred_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    status TEXT,
    progress REAL,
    printer_id TEXT,
    error_code TEXT,
    error_message TEXT,
    actor_type TEXT NOT NULL DEFAULT 'system',
    actor_id TEXT,
    metadata TEXT DEFAULT '{}',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_job_events_job_id ON job_events(job_id);
CREATE INDEX IF NOT EXISTS idx_job_events_occurred_at ON job_events(occurred_at);
CREATE INDEX IF NOT EXISTS idx_job_events_type ON job_events(event_type);
CREATE INDEX IF NOT EXISTS idx_job_events_job_occurred ON job_events(job_id, occurred_at);

-- Expenses table
CREATE TABLE IF NOT EXISTS expenses (
    id TEXT PRIMARY KEY,
    occurred_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    vendor TEXT DEFAULT '',
    subtotal_cents INTEGER NOT NULL DEFAULT 0,
    tax_cents INTEGER NOT NULL DEFAULT 0,
    shipping_cents INTEGER NOT NULL DEFAULT 0,
    total_cents INTEGER NOT NULL DEFAULT 0,
    currency TEXT NOT NULL DEFAULT 'USD',
    category TEXT NOT NULL DEFAULT 'other',
    notes TEXT DEFAULT '',
    receipt_file_id TEXT REFERENCES files(id),
    receipt_file_path TEXT DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending',
    raw_ocr_text TEXT DEFAULT '',
    raw_ai_response TEXT,
    confidence INTEGER DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_expenses_occurred_at ON expenses(occurred_at);
CREATE INDEX IF NOT EXISTS idx_expenses_category ON expenses(category);
CREATE INDEX IF NOT EXISTS idx_expenses_status ON expenses(status);
CREATE INDEX IF NOT EXISTS idx_expenses_vendor ON expenses(vendor);

-- Expense line items
CREATE TABLE IF NOT EXISTS expense_items (
    id TEXT PRIMARY KEY,
    expense_id TEXT NOT NULL REFERENCES expenses(id) ON DELETE CASCADE,
    description TEXT NOT NULL DEFAULT '',
    quantity REAL NOT NULL DEFAULT 1,
    unit_price_cents INTEGER NOT NULL DEFAULT 0,
    total_price_cents INTEGER NOT NULL DEFAULT 0,
    sku TEXT DEFAULT '',
    vendor_item_id TEXT DEFAULT '',
    category TEXT NOT NULL DEFAULT 'other',
    metadata TEXT DEFAULT '{}',
    matched_spool_id TEXT REFERENCES material_spools(id),
    matched_material_id TEXT REFERENCES materials(id),
    confidence INTEGER DEFAULT 0,
    action_taken TEXT DEFAULT 'none',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_expense_items_expense_id ON expense_items(expense_id);
CREATE INDEX IF NOT EXISTS idx_expense_items_category ON expense_items(category);
CREATE INDEX IF NOT EXISTS idx_expense_items_matched_spool ON expense_items(matched_spool_id);

-- Sales table
CREATE TABLE IF NOT EXISTS sales (
    id TEXT PRIMARY KEY,
    occurred_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    channel TEXT NOT NULL DEFAULT 'other',
    platform TEXT DEFAULT '',
    gross_cents INTEGER NOT NULL DEFAULT 0,
    fees_cents INTEGER NOT NULL DEFAULT 0,
    shipping_charged_cents INTEGER DEFAULT 0,
    shipping_cost_cents INTEGER DEFAULT 0,
    tax_collected_cents INTEGER DEFAULT 0,
    net_cents INTEGER NOT NULL DEFAULT 0,
    currency TEXT NOT NULL DEFAULT 'USD',
    project_id TEXT REFERENCES projects(id),
    customer_id TEXT REFERENCES customers(id) ON DELETE SET NULL,
    order_reference TEXT DEFAULT '',
    customer_name TEXT DEFAULT '',
    item_description TEXT DEFAULT '',
    quantity INTEGER NOT NULL DEFAULT 1,
    notes TEXT DEFAULT '',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sales_occurred_at ON sales(occurred_at);
CREATE INDEX IF NOT EXISTS idx_sales_channel ON sales(channel);
CREATE INDEX IF NOT EXISTS idx_sales_project_id ON sales(project_id);

-- Etsy integration
CREATE TABLE IF NOT EXISTS etsy_integration (
    id TEXT PRIMARY KEY,
    shop_id INTEGER UNIQUE NOT NULL,
    shop_name TEXT NOT NULL,
    user_id INTEGER NOT NULL,
    access_token TEXT NOT NULL,
    refresh_token TEXT NOT NULL,
    token_expires_at TEXT NOT NULL,
    scopes TEXT DEFAULT '[]',
    is_active INTEGER DEFAULT 1,
    last_sync_at TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Etsy OAuth states
CREATE TABLE IF NOT EXISTS etsy_oauth_states (
    state TEXT PRIMARY KEY,
    code_verifier TEXT NOT NULL,
    redirect_uri TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_etsy_oauth_states_created ON etsy_oauth_states(created_at);

-- Etsy receipts
CREATE TABLE IF NOT EXISTS etsy_receipts (
    id TEXT PRIMARY KEY,
    etsy_receipt_id INTEGER UNIQUE NOT NULL,
    etsy_shop_id INTEGER NOT NULL,
    buyer_user_id INTEGER,
    buyer_email TEXT,
    name TEXT,
    status TEXT NOT NULL,
    message_from_buyer TEXT,
    is_shipped INTEGER DEFAULT 0,
    is_paid INTEGER DEFAULT 0,
    is_gift INTEGER DEFAULT 0,
    gift_message TEXT,
    grandtotal_cents INTEGER NOT NULL,
    subtotal_cents INTEGER NOT NULL,
    total_price_cents INTEGER NOT NULL,
    total_shipping_cost_cents INTEGER DEFAULT 0,
    total_tax_cost_cents INTEGER DEFAULT 0,
    discount_cents INTEGER DEFAULT 0,
    currency TEXT DEFAULT 'USD',
    shipping_name TEXT,
    shipping_address_first_line TEXT,
    shipping_address_second_line TEXT,
    shipping_city TEXT,
    shipping_state TEXT,
    shipping_zip TEXT,
    shipping_country_code TEXT,
    create_timestamp TEXT,
    update_timestamp TEXT,
    is_processed INTEGER DEFAULT 0,
    project_id TEXT REFERENCES projects(id),
    synced_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_etsy_receipts_etsy_id ON etsy_receipts(etsy_receipt_id);
CREATE INDEX IF NOT EXISTS idx_etsy_receipts_processed ON etsy_receipts(is_processed);
CREATE INDEX IF NOT EXISTS idx_etsy_receipts_shop ON etsy_receipts(etsy_shop_id);

-- Etsy receipt items
CREATE TABLE IF NOT EXISTS etsy_receipt_items (
    id TEXT PRIMARY KEY,
    etsy_receipt_item_id INTEGER UNIQUE NOT NULL,
    receipt_id TEXT NOT NULL REFERENCES etsy_receipts(id) ON DELETE CASCADE,
    etsy_listing_id INTEGER NOT NULL,
    etsy_transaction_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    quantity INTEGER NOT NULL DEFAULT 1,
    price_cents INTEGER NOT NULL,
    shipping_cost_cents INTEGER DEFAULT 0,
    sku TEXT,
    variations TEXT DEFAULT '[]',
    is_digital INTEGER DEFAULT 0,
    template_id TEXT REFERENCES templates(id),
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_etsy_receipt_items_receipt ON etsy_receipt_items(receipt_id);
CREATE INDEX IF NOT EXISTS idx_etsy_receipt_items_sku ON etsy_receipt_items(sku);

-- Etsy sync state
CREATE TABLE IF NOT EXISTS etsy_sync_state (
    id TEXT PRIMARY KEY,
    shop_id INTEGER UNIQUE NOT NULL,
    last_receipt_sync_at TEXT,
    last_listing_sync_at TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Etsy listings
CREATE TABLE IF NOT EXISTS etsy_listings (
    id TEXT PRIMARY KEY,
    etsy_listing_id INTEGER UNIQUE NOT NULL,
    etsy_shop_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    state TEXT NOT NULL,
    quantity INTEGER NOT NULL DEFAULT 0,
    url TEXT,
    views INTEGER DEFAULT 0,
    num_favorers INTEGER DEFAULT 0,
    is_customizable INTEGER DEFAULT 0,
    is_personalizable INTEGER DEFAULT 0,
    tags TEXT DEFAULT '[]',
    has_variations INTEGER DEFAULT 0,
    price_cents INTEGER,
    currency TEXT DEFAULT 'USD',
    skus TEXT DEFAULT '[]',
    synced_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_etsy_listings_etsy_id ON etsy_listings(etsy_listing_id);
CREATE INDEX IF NOT EXISTS idx_etsy_listings_state ON etsy_listings(state);

-- Etsy listing to template mapping
CREATE TABLE IF NOT EXISTS etsy_listing_templates (
    id TEXT PRIMARY KEY,
    etsy_listing_id INTEGER NOT NULL REFERENCES etsy_listings(etsy_listing_id) ON DELETE CASCADE,
    template_id TEXT NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
    sku TEXT,
    sync_inventory INTEGER DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(etsy_listing_id, template_id)
);

CREATE INDEX IF NOT EXISTS idx_etsy_listing_templates_sku ON etsy_listing_templates(sku);

-- Etsy webhook events
CREATE TABLE IF NOT EXISTS etsy_webhook_events (
    id TEXT PRIMARY KEY,
    event_type TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id INTEGER,
    shop_id INTEGER,
    payload TEXT NOT NULL,
    signature TEXT,
    processed INTEGER DEFAULT 0,
    processed_at TEXT,
    error TEXT,
    received_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_etsy_webhook_events_type ON etsy_webhook_events(event_type);
CREATE INDEX IF NOT EXISTS idx_etsy_webhook_events_processed ON etsy_webhook_events(processed);

-- Squarespace Integration
CREATE TABLE IF NOT EXISTS squarespace_integration (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL,
    site_title TEXT,
    api_key TEXT NOT NULL,  -- Encrypted
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
    updated_at TEXT NOT NULL,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_squarespace_orders_order_id ON squarespace_orders(squarespace_order_id);
CREATE INDEX IF NOT EXISTS idx_squarespace_orders_processed ON squarespace_orders(is_processed);

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
    created_at TEXT NOT NULL,
    FOREIGN KEY (order_id) REFERENCES squarespace_orders(id) ON DELETE CASCADE,
    FOREIGN KEY (template_id) REFERENCES templates(id) ON DELETE SET NULL
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

CREATE INDEX IF NOT EXISTS idx_squarespace_products_product_id ON squarespace_products(squarespace_product_id);

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
    created_at TEXT NOT NULL,
    FOREIGN KEY (product_id) REFERENCES squarespace_products(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS squarespace_product_templates (
    id TEXT PRIMARY KEY,
    squarespace_product_id TEXT NOT NULL,
    template_id TEXT NOT NULL,
    sku TEXT,
    created_at TEXT NOT NULL,
    UNIQUE(squarespace_product_id, template_id),
    FOREIGN KEY (template_id) REFERENCES templates(id) ON DELETE CASCADE
);

-- Recipe materials
CREATE TABLE IF NOT EXISTS recipe_materials (
    id TEXT PRIMARY KEY,
    recipe_id TEXT NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
    material_type TEXT NOT NULL,
    color_spec TEXT,
    weight_grams REAL NOT NULL DEFAULT 0,
    ams_position INTEGER,
    sequence_order INTEGER DEFAULT 0,
    notes TEXT DEFAULT '',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(recipe_id, sequence_order)
);

CREATE INDEX IF NOT EXISTS idx_recipe_materials_recipe ON recipe_materials(recipe_id);

-- Recipe supplies (non-printed BOM items for templates/recipes)
CREATE TABLE IF NOT EXISTS recipe_supplies (
    id TEXT PRIMARY KEY,
    recipe_id TEXT NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    unit_cost_cents INTEGER NOT NULL DEFAULT 0,
    quantity INTEGER NOT NULL DEFAULT 1,
    material_id TEXT REFERENCES materials(id),
    notes TEXT DEFAULT '',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_recipe_supplies_recipe ON recipe_supplies(recipe_id);

-- Settings (key-value store for app configuration)
CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Bambu Cloud authentication storage
CREATE TABLE IF NOT EXISTS bambu_cloud_auth (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL,
    access_token TEXT NOT NULL,
    refresh_token TEXT NOT NULL DEFAULT '',
    mqtt_username TEXT NOT NULL,
    expires_at TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Project supplies (non-printed BOM items)
CREATE TABLE IF NOT EXISTS project_supplies (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    unit_cost_cents INTEGER NOT NULL DEFAULT 0,
    quantity INTEGER NOT NULL DEFAULT 1,
    notes TEXT DEFAULT '',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_project_supplies_project ON project_supplies(project_id);

-- ============================================
-- Low-Spool Threshold Alerts (Phase 1)
-- ============================================

-- Alert dismissals for user dismissal tracking
CREATE TABLE IF NOT EXISTS alert_dismissals (
    id TEXT PRIMARY KEY,
    alert_type TEXT NOT NULL,  -- 'low_spool', 'empty_spool', 'order_due', etc.
    entity_id TEXT NOT NULL,   -- spool_id, printer_id, order_id, etc.
    dismissed_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    dismissed_until TEXT       -- NULL = permanent, or timestamp for snooze
);
CREATE INDEX IF NOT EXISTS idx_alert_dismissals_entity ON alert_dismissals(alert_type, entity_id);

-- ============================================
-- Customers
-- ============================================

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

-- ============================================
-- Unified Orders (Phase 2)
-- ============================================

CREATE TABLE IF NOT EXISTS orders (
    id TEXT PRIMARY KEY,
    source TEXT NOT NULL,           -- 'manual', 'etsy', 'squarespace', 'shopify', 'quote'
    source_order_id TEXT,           -- External order ID
    customer_name TEXT NOT NULL,
    customer_email TEXT,
    customer_id TEXT REFERENCES customers(id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'pending',  -- pending, in_progress, completed, shipped, cancelled
    priority INTEGER NOT NULL DEFAULT 0,
    due_date TEXT,
    notes TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TEXT,
    shipped_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_source ON orders(source, source_order_id);
CREATE INDEX IF NOT EXISTS idx_orders_due_date ON orders(due_date);

CREATE TABLE IF NOT EXISTS order_items (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    template_id TEXT REFERENCES templates(id),  -- Legacy: kept for migration
    project_id TEXT REFERENCES projects(id),    -- New: link to project (product catalog)
    sku TEXT,
    quantity INTEGER NOT NULL DEFAULT 1,
    notes TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_order_items_order ON order_items(order_id);
CREATE INDEX IF NOT EXISTS idx_order_items_sku ON order_items(sku);
CREATE INDEX IF NOT EXISTS idx_order_items_project ON order_items(project_id);

-- Order events for history tracking
CREATE TABLE IF NOT EXISTS order_events (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,  -- 'created', 'synced', 'item_added', 'job_started', 'completed', etc.
    message TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_order_events_order ON order_events(order_id);

-- ============================================
-- Quotes
-- ============================================

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
    project_id TEXT REFERENCES projects(id) ON DELETE SET NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_quote_line_items_option ON quote_line_items(option_id);

CREATE TABLE IF NOT EXISTS quote_events (
    id TEXT PRIMARY KEY,
    quote_id TEXT NOT NULL REFERENCES quotes(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    message TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_quote_events_quote ON quote_events(quote_id);

-- ============================================
-- Shopify Integration (Phase 3)
-- ============================================

CREATE TABLE IF NOT EXISTS shopify_credentials (
    id TEXT PRIMARY KEY,
    shop_domain TEXT NOT NULL UNIQUE,
    access_token TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS shopify_orders (
    id TEXT PRIMARY KEY,
    shopify_order_id TEXT NOT NULL UNIQUE,
    order_id TEXT REFERENCES orders(id),  -- Link to unified Order
    shop_domain TEXT NOT NULL,
    order_number TEXT,
    customer_name TEXT,
    customer_email TEXT,
    total_cents INTEGER,
    status TEXT,
    synced_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_shopify_orders_shopify_id ON shopify_orders(shopify_order_id);
CREATE INDEX IF NOT EXISTS idx_shopify_orders_order ON shopify_orders(order_id);

CREATE TABLE IF NOT EXISTS shopify_order_items (
    id TEXT PRIMARY KEY,
    shopify_order_id TEXT NOT NULL REFERENCES shopify_orders(id) ON DELETE CASCADE,
    shopify_line_item_id TEXT NOT NULL,
    sku TEXT,
    title TEXT,
    quantity INTEGER NOT NULL,
    price_cents INTEGER,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_shopify_order_items_order ON shopify_order_items(shopify_order_id);

CREATE TABLE IF NOT EXISTS shopify_product_templates (
    id TEXT PRIMARY KEY,
    shopify_product_id TEXT NOT NULL,
    template_id TEXT NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
    sku TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(shopify_product_id, template_id)
);

-- ============================================
-- Tags for File Versioning (Phase 5)
-- ============================================

CREATE TABLE IF NOT EXISTS tags (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    color TEXT DEFAULT '#6b7280',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS part_tags (
    part_id TEXT NOT NULL REFERENCES parts(id) ON DELETE CASCADE,
    tag_id TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (part_id, tag_id)
);

CREATE TABLE IF NOT EXISTS design_tags (
    design_id TEXT NOT NULL REFERENCES designs(id) ON DELETE CASCADE,
    tag_id TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (design_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_part_tags_part ON part_tags(part_id);
CREATE INDEX IF NOT EXISTS idx_part_tags_tag ON part_tags(tag_id);
CREATE INDEX IF NOT EXISTS idx_design_tags_design ON design_tags(design_id);
CREATE INDEX IF NOT EXISTS idx_design_tags_tag ON design_tags(tag_id);

-- ============================================
-- Views
-- ============================================

-- Current job status (replaces PostgreSQL DISTINCT ON)
CREATE VIEW IF NOT EXISTS job_current_status AS
SELECT je.job_id, je.status, je.progress,
       je.occurred_at AS status_changed_at,
       je.error_code, je.error_message
FROM job_events je
WHERE je.status IS NOT NULL
  AND je.occurred_at = (
      SELECT MAX(je2.occurred_at) FROM job_events je2
      WHERE je2.job_id = je.job_id AND je2.status IS NOT NULL
  );

-- Job status durations
CREATE VIEW IF NOT EXISTS job_status_durations AS
WITH status_changes AS (
    SELECT
        job_id,
        status,
        occurred_at,
        LEAD(occurred_at) OVER (PARTITION BY job_id ORDER BY occurred_at) AS next_occurred_at
    FROM job_events
    WHERE status IS NOT NULL
)
SELECT
    job_id,
    status,
    occurred_at AS started_at,
    next_occurred_at AS ended_at,
    (strftime('%s', COALESCE(next_occurred_at, datetime('now'))) - strftime('%s', occurred_at)) AS duration_seconds
FROM status_changes;

-- Beta Feedback
CREATE TABLE IF NOT EXISTS feedback (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL DEFAULT 'general',
    message TEXT NOT NULL,
    contact TEXT,
    page TEXT,
    app_version TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
