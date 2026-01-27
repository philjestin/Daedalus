-- Add pricing fields to templates/recipes for margin calculation

-- Labor time in minutes (flat per SKU)
ALTER TABLE templates ADD COLUMN IF NOT EXISTS labor_minutes INT NOT NULL DEFAULT 0;

-- Base sale price in cents
ALTER TABLE templates ADD COLUMN IF NOT EXISTS sale_price_cents INT NOT NULL DEFAULT 0;

-- Material cost per gram in cents (cached from material at recipe creation, can be overridden)
ALTER TABLE templates ADD COLUMN IF NOT EXISTS material_cost_per_gram_cents INT NOT NULL DEFAULT 0;

COMMENT ON COLUMN templates.labor_minutes IS 'Manual labor time per unit in minutes';
COMMENT ON COLUMN templates.sale_price_cents IS 'Base sale price per unit in cents';
COMMENT ON COLUMN templates.material_cost_per_gram_cents IS 'Material cost per gram in cents (for margin calc)';
