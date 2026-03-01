-- Link quote line items to projects (products)
ALTER TABLE quote_line_items ADD COLUMN project_id TEXT REFERENCES projects(id) ON DELETE SET NULL;
