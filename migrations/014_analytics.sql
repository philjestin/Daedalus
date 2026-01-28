-- Printer cost model: hourly rate for time-cost attribution
ALTER TABLE printers ADD COLUMN cost_per_hour_cents INTEGER NOT NULL DEFAULT 150;

-- Job cost breakdown: snapshot at completion (not recalculated)
ALTER TABLE print_jobs ADD COLUMN printer_time_cost_cents INTEGER;
ALTER TABLE print_jobs ADD COLUMN material_cost_cents INTEGER;
