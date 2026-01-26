-- Recipe enhancements: transforms templates into full manufacturing recipes
-- This enables SKU -> recipe mapping for unified production

-- Enhance templates table with recipe capabilities
ALTER TABLE templates
    ADD COLUMN IF NOT EXISTS printer_constraints JSONB DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS print_profile VARCHAR(50) DEFAULT 'standard',
    ADD COLUMN IF NOT EXISTS estimated_print_seconds INT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS version INT DEFAULT 1,
    ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ;

-- Recipe materials (multi-material support for AMS)
CREATE TABLE IF NOT EXISTS recipe_materials (
    id UUID PRIMARY KEY,
    recipe_id UUID NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
    material_type VARCHAR(50) NOT NULL,
    color_spec JSONB,
    weight_grams DECIMAL(10,2) NOT NULL DEFAULT 0,
    ams_position INT,
    sequence_order INT DEFAULT 0,
    notes TEXT DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(recipe_id, sequence_order)
);

CREATE INDEX IF NOT EXISTS idx_recipe_materials_recipe ON recipe_materials(recipe_id);

-- Update table comments
COMMENT ON TABLE templates IS 'Recipes: SKU to manufacturing instructions';
COMMENT ON COLUMN templates.printer_constraints IS 'JSON: {min_bed_size, nozzle_diameters, requires_enclosure, requires_ams, printer_tags}';
COMMENT ON COLUMN templates.print_profile IS 'Symbolic slicer profile: standard, detailed, fast, strong, custom';
COMMENT ON COLUMN templates.estimated_print_seconds IS 'Estimated print time for cost calculation and scheduling';
COMMENT ON COLUMN templates.version IS 'Recipe version number for tracking changes';
COMMENT ON COLUMN templates.archived_at IS 'Timestamp when recipe was archived (soft delete)';

COMMENT ON TABLE recipe_materials IS 'Multi-material requirements for recipes';
COMMENT ON COLUMN recipe_materials.color_spec IS 'JSON: {mode: exact|category|any, hex, name}';
COMMENT ON COLUMN recipe_materials.ams_position IS 'AMS slot position (1-4) for multi-material prints';
COMMENT ON COLUMN recipe_materials.sequence_order IS 'Order in which materials are used';
