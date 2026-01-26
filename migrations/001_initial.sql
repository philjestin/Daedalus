-- Migration: 001_initial
-- Description: Initial database schema

-- Projects table
CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT DEFAULT '',
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    target_date TIMESTAMPTZ,
    tags TEXT[] DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_projects_status ON projects(status);
CREATE INDEX idx_projects_updated_at ON projects(updated_at DESC);

-- Parts table
CREATE TABLE IF NOT EXISTS parts (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT DEFAULT '',
    quantity INT NOT NULL DEFAULT 1,
    status VARCHAR(50) NOT NULL DEFAULT 'design',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_parts_project_id ON parts(project_id);

-- Files table (for content-addressed storage)
CREATE TABLE IF NOT EXISTS files (
    id UUID PRIMARY KEY,
    hash VARCHAR(64) NOT NULL UNIQUE,
    original_name VARCHAR(255) NOT NULL,
    content_type VARCHAR(100) DEFAULT 'application/octet-stream',
    size_bytes BIGINT NOT NULL,
    storage_path VARCHAR(500) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_files_hash ON files(hash);

-- Designs table (versioned, immutable)
CREATE TABLE IF NOT EXISTS designs (
    id UUID PRIMARY KEY,
    part_id UUID NOT NULL REFERENCES parts(id) ON DELETE CASCADE,
    version INT NOT NULL,
    file_id UUID NOT NULL REFERENCES files(id),
    file_name VARCHAR(255) NOT NULL,
    file_hash VARCHAR(64) NOT NULL,
    file_size_bytes BIGINT NOT NULL,
    file_type VARCHAR(20) NOT NULL,
    notes TEXT DEFAULT '',
    slice_profile JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(part_id, version)
);

CREATE INDEX idx_designs_part_id ON designs(part_id);

-- Printers table
CREATE TABLE IF NOT EXISTS printers (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    model VARCHAR(255) DEFAULT '',
    manufacturer VARCHAR(255) DEFAULT '',
    connection_type VARCHAR(50) NOT NULL DEFAULT 'manual',
    connection_uri VARCHAR(500) DEFAULT '',
    api_key VARCHAR(255) DEFAULT '',
    status VARCHAR(50) NOT NULL DEFAULT 'offline',
    build_volume JSONB,
    nozzle_diameter DECIMAL(3,2) DEFAULT 0.4,
    location VARCHAR(255) DEFAULT '',
    notes TEXT DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_printers_status ON printers(status);

-- Materials table (catalog)
CREATE TABLE IF NOT EXISTS materials (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    manufacturer VARCHAR(255) DEFAULT '',
    color VARCHAR(100) DEFAULT '',
    color_hex VARCHAR(7) DEFAULT '',
    density DECIMAL(4,2) DEFAULT 1.24,
    cost_per_kg DECIMAL(10,2) DEFAULT 0,
    print_temp JSONB,
    bed_temp JSONB,
    notes TEXT DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_materials_type ON materials(type);

-- Material spools table (inventory)
CREATE TABLE IF NOT EXISTS material_spools (
    id UUID PRIMARY KEY,
    material_id UUID NOT NULL REFERENCES materials(id),
    initial_weight DECIMAL(10,2) NOT NULL DEFAULT 1000,
    remaining_weight DECIMAL(10,2) NOT NULL DEFAULT 1000,
    purchase_date DATE,
    purchase_cost DECIMAL(10,2) DEFAULT 0,
    location VARCHAR(255) DEFAULT '',
    status VARCHAR(50) NOT NULL DEFAULT 'new',
    notes TEXT DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_spools_material_id ON material_spools(material_id);
CREATE INDEX idx_spools_status ON material_spools(status);

-- Print jobs table (immutable records)
CREATE TABLE IF NOT EXISTS print_jobs (
    id UUID PRIMARY KEY,
    design_id UUID NOT NULL REFERENCES designs(id),
    printer_id UUID NOT NULL REFERENCES printers(id),
    material_spool_id UUID NOT NULL REFERENCES material_spools(id),
    status VARCHAR(50) NOT NULL DEFAULT 'queued',
    progress DECIMAL(5,2) DEFAULT 0,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    outcome JSONB,
    notes TEXT DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_print_jobs_design_id ON print_jobs(design_id);
CREATE INDEX idx_print_jobs_printer_id ON print_jobs(printer_id);
CREATE INDEX idx_print_jobs_status ON print_jobs(status);
CREATE INDEX idx_print_jobs_created_at ON print_jobs(created_at DESC);

-- Print events table (immutable audit log)
CREATE TABLE IF NOT EXISTS print_events (
    id UUID PRIMARY KEY,
    print_job_id UUID NOT NULL REFERENCES print_jobs(id),
    event_type VARCHAR(50) NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    data JSONB
);

CREATE INDEX idx_print_events_job_id ON print_events(print_job_id);
CREATE INDEX idx_print_events_timestamp ON print_events(timestamp);

