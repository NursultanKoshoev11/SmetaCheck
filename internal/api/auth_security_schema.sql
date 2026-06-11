ALTER TABLE auth_sessions
    ADD COLUMN IF NOT EXISTS previous_refresh_token_hash TEXT;

ALTER TABLE auth_sessions
    ADD COLUMN IF NOT EXISTS rotated_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_auth_sessions_previous_refresh_hash
    ON auth_sessions(previous_refresh_token_hash)
    WHERE previous_refresh_token_hash IS NOT NULL;

CREATE TABLE IF NOT EXISTS auth_rate_limits (
    key_hash TEXT NOT NULL,
    action TEXT NOT NULL,
    window_started_at TIMESTAMPTZ NOT NULL,
    request_count INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (key_hash, action, window_started_at)
);

CREATE INDEX IF NOT EXISTS idx_auth_rate_limits_updated_at
    ON auth_rate_limits(updated_at);
