-- Etsy Shop Integration (one shop per install for now)
CREATE TABLE IF NOT EXISTS etsy_integration (
    id UUID PRIMARY KEY,
    shop_id BIGINT UNIQUE NOT NULL,
    shop_name VARCHAR(255) NOT NULL,
    user_id BIGINT NOT NULL,
    access_token TEXT NOT NULL,
    refresh_token TEXT NOT NULL,
    token_expires_at TIMESTAMPTZ NOT NULL,
    scopes TEXT[] DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    last_sync_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Pending OAuth states (for PKCE verification)
CREATE TABLE IF NOT EXISTS etsy_oauth_states (
    state VARCHAR(64) PRIMARY KEY,
    code_verifier VARCHAR(128) NOT NULL,
    redirect_uri TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Auto-cleanup old states
CREATE INDEX IF NOT EXISTS idx_etsy_oauth_states_created ON etsy_oauth_states(created_at);
