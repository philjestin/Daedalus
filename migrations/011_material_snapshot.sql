-- 011_material_snapshot.sql
-- Add material snapshot and depletion guardrails support

-- Store AMS tray state snapshot when job starts
ALTER TABLE print_jobs
    ADD COLUMN IF NOT EXISTS material_snapshot JSONB;

-- Add minimum remaining percentage threshold for material warnings
-- Stored per-printer so different printers can have different thresholds
ALTER TABLE printers
    ADD COLUMN IF NOT EXISTS min_material_percent INT DEFAULT 10;

-- Add index for querying jobs by material snapshot data
CREATE INDEX IF NOT EXISTS idx_print_jobs_material_snapshot ON print_jobs USING GIN (material_snapshot) WHERE material_snapshot IS NOT NULL;
