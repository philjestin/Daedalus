-- Migration: 009_immutable_job_history
-- Description: Transform print jobs into immutable records with append-only event history
-- Rule: Printer state is ephemeral. Jobs are forever.

-- Enhance print_jobs table for immutability and retry tracking
ALTER TABLE print_jobs
    ADD COLUMN IF NOT EXISTS recipe_id UUID REFERENCES templates(id),
    ADD COLUMN IF NOT EXISTS attempt_number INT NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS parent_job_id UUID REFERENCES print_jobs(id),
    ADD COLUMN IF NOT EXISTS failure_category VARCHAR(50),
    ADD COLUMN IF NOT EXISTS estimated_seconds INT,
    ADD COLUMN IF NOT EXISTS actual_seconds INT,
    ADD COLUMN IF NOT EXISTS material_used_grams DECIMAL(10,2),
    ADD COLUMN IF NOT EXISTS cost_cents INT;

-- Index for retry chains
CREATE INDEX IF NOT EXISTS idx_print_jobs_parent ON print_jobs(parent_job_id) WHERE parent_job_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_print_jobs_recipe ON print_jobs(recipe_id) WHERE recipe_id IS NOT NULL;

-- Drop old print_events and recreate with proper structure
-- (keeping old data would require migration, but for dev this is fine)
DROP TABLE IF EXISTS print_events CASCADE;

-- Job events table (append-only, immutable audit log)
CREATE TABLE job_events (
    id UUID PRIMARY KEY,
    job_id UUID NOT NULL REFERENCES print_jobs(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Event-specific data
    status VARCHAR(50),              -- resulting status after this event
    progress DECIMAL(5,2),           -- progress at time of event (0-100)
    printer_id UUID,                 -- printer context (for assignment/transfer events)

    -- Error/failure context
    error_code VARCHAR(50),
    error_message TEXT,

    -- Actor tracking (who/what triggered this event)
    actor_type VARCHAR(20) NOT NULL DEFAULT 'system',  -- 'user', 'system', 'printer', 'webhook'
    actor_id VARCHAR(255),           -- user ID, printer serial, webhook source

    -- Flexible metadata
    metadata JSONB DEFAULT '{}',

    -- Denormalized for query efficiency
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for common query patterns
CREATE INDEX idx_job_events_job_id ON job_events(job_id);
CREATE INDEX idx_job_events_occurred_at ON job_events(occurred_at DESC);
CREATE INDEX idx_job_events_type ON job_events(event_type);
CREATE INDEX idx_job_events_job_occurred ON job_events(job_id, occurred_at DESC);

-- Composite index for status timeline queries
CREATE INDEX idx_job_events_status_timeline ON job_events(job_id, occurred_at) WHERE status IS NOT NULL;

-- Index for failure analysis
CREATE INDEX idx_job_events_errors ON job_events(error_code) WHERE error_code IS NOT NULL;

-- Materialized view for current job status (computed from events)
-- This can be refreshed periodically or we can compute in application
CREATE OR REPLACE VIEW job_current_status AS
SELECT DISTINCT ON (je.job_id)
    je.job_id,
    je.status,
    je.progress,
    je.occurred_at as status_changed_at,
    je.error_code,
    je.error_message
FROM job_events je
WHERE je.status IS NOT NULL
ORDER BY je.job_id, je.occurred_at DESC;

-- Analytics view: job duration by status
CREATE OR REPLACE VIEW job_status_durations AS
WITH status_changes AS (
    SELECT
        job_id,
        status,
        occurred_at,
        LEAD(occurred_at) OVER (PARTITION BY job_id ORDER BY occurred_at) as next_occurred_at
    FROM job_events
    WHERE status IS NOT NULL
)
SELECT
    job_id,
    status,
    occurred_at as started_at,
    next_occurred_at as ended_at,
    EXTRACT(EPOCH FROM (COALESCE(next_occurred_at, NOW()) - occurred_at)) as duration_seconds
FROM status_changes;

-- Comments
COMMENT ON TABLE job_events IS 'Append-only event log for print jobs. Never delete or update rows.';
COMMENT ON COLUMN job_events.event_type IS 'Event types: queued, assigned, uploaded, printing, progress, paused, resumed, completed, failed, cancelled, retried';
COMMENT ON COLUMN job_events.status IS 'Job status after this event. Null for events that dont change status (e.g., progress updates)';
COMMENT ON COLUMN job_events.actor_type IS 'What triggered this event: user, system, printer, webhook';

COMMENT ON COLUMN print_jobs.attempt_number IS 'Which attempt this is (1 = first try, 2+ = retries)';
COMMENT ON COLUMN print_jobs.parent_job_id IS 'Reference to original job if this is a retry';
COMMENT ON COLUMN print_jobs.failure_category IS 'Categorized failure reason: mechanical, filament, adhesion, thermal, user_cancelled, unknown';
