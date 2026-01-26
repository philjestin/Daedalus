-- Project Templates
CREATE TABLE IF NOT EXISTS templates (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT DEFAULT '',
    sku VARCHAR(255) UNIQUE,                    -- For future Etsy SKU matching
    tags TEXT[] DEFAULT '{}',
    material_type VARCHAR(50) NOT NULL,         -- pla, petg, abs, asa, tpu
    estimated_material_grams DECIMAL(10,2) DEFAULT 0,
    preferred_printer_id UUID REFERENCES printers(id),
    allow_any_printer BOOLEAN DEFAULT false,
    quantity_per_order INT NOT NULL DEFAULT 1,  -- 1 order = N parts
    post_process_checklist JSONB DEFAULT '[]',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_templates_sku ON templates(sku);
CREATE INDEX idx_templates_active ON templates(is_active);

-- Template to Design mapping (many-to-many)
CREATE TABLE IF NOT EXISTS template_designs (
    id UUID PRIMARY KEY,
    template_id UUID NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
    design_id UUID NOT NULL REFERENCES designs(id),
    is_primary BOOLEAN DEFAULT false,
    quantity INT NOT NULL DEFAULT 1,
    sequence_order INT DEFAULT 0,
    notes TEXT DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(template_id, design_id)
);

CREATE INDEX idx_template_designs_template ON template_designs(template_id);

-- Add template tracking to projects
ALTER TABLE projects
    ADD COLUMN IF NOT EXISTS template_id UUID REFERENCES templates(id),
    ADD COLUMN IF NOT EXISTS source VARCHAR(50) DEFAULT 'manual',
    ADD COLUMN IF NOT EXISTS external_order_id VARCHAR(255),
    ADD COLUMN IF NOT EXISTS customer_notes TEXT DEFAULT '';

CREATE INDEX idx_projects_template ON projects(template_id);
CREATE INDEX idx_projects_external_order ON projects(external_order_id);
