CREATE TABLE IF NOT EXISTS feedback (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL DEFAULT 'general',
    message TEXT NOT NULL,
    contact TEXT,
    page TEXT,
    app_version TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
