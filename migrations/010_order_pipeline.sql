-- 010_order_pipeline.sql
-- Make the production pipeline functional: Orders create Projects, Projects create Jobs from Recipes, Jobs run to completion.

-- Allow jobs without printer/spool assignment (pending assignment)
ALTER TABLE print_jobs
    ALTER COLUMN printer_id DROP NOT NULL,
    ALTER COLUMN material_spool_id DROP NOT NULL;

-- Add project_id to link jobs directly to projects
ALTER TABLE print_jobs
    ADD COLUMN IF NOT EXISTS project_id UUID REFERENCES projects(id);

-- Index for project job queries
CREATE INDEX IF NOT EXISTS idx_print_jobs_project ON print_jobs(project_id) WHERE project_id IS NOT NULL;

-- Add shipping tracking to projects
ALTER TABLE projects
    ADD COLUMN IF NOT EXISTS shipped_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS tracking_number VARCHAR(255);
