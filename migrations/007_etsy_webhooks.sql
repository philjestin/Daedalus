-- Etsy Webhook Events (audit log)
CREATE TABLE IF NOT EXISTS etsy_webhook_events (
    id UUID PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    resource_id BIGINT,
    shop_id BIGINT,
    payload JSONB NOT NULL,
    signature VARCHAR(255),
    processed BOOLEAN DEFAULT false,
    processed_at TIMESTAMPTZ,
    error TEXT,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_etsy_webhook_events_type ON etsy_webhook_events(event_type);
CREATE INDEX idx_etsy_webhook_events_processed ON etsy_webhook_events(processed);
