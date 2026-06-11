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
