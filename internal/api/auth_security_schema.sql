ALTER TABLE auth_sessions
    ADD COLUMN IF NOT EXISTS previous_refresh_token_hash TEXT;

ALTER TABLE auth_sessions
    ADD COLUMN IF NOT EXISTS rotated_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_auth_sessions_previous_refresh_hash
    ON auth_sessions(previous_refresh_token_hash)
    WHERE previous_refresh_token_hash IS NOT NULL;

CREATE TABLE IF NOT EXISTS auth_refresh_token_history (
    token_hash TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES auth_sessions(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rotated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    ip_address TEXT,
    user_agent TEXT
);

CREATE INDEX IF NOT EXISTS idx_auth_refresh_history_session
    ON auth_refresh_token_history(session_id, rotated_at DESC);

CREATE INDEX IF NOT EXISTS idx_auth_refresh_history_user
    ON auth_refresh_token_history(user_id, rotated_at DESC);

CREATE INDEX IF NOT EXISTS idx_auth_refresh_history_expires
    ON auth_refresh_token_history(expires_at);

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

ALTER TABLE users ADD COLUMN IF NOT EXISTS plan TEXT NOT NULL DEFAULT 'free';
ALTER TABLE users ADD COLUMN IF NOT EXISTS quota_files INTEGER NOT NULL DEFAULT 10;
ALTER TABLE users ADD COLUMN IF NOT EXISTS disabled_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS mfa_enabled BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE users ADD COLUMN IF NOT EXISTS mfa_secret_encrypted TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS mfa_pending_secret_encrypted TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS mfa_enabled_at TIMESTAMPTZ;

ALTER TABLE auth_sessions ADD COLUMN IF NOT EXISTS device_fingerprint TEXT;
ALTER TABLE auth_sessions ADD COLUMN IF NOT EXISTS mfa_verified_at TIMESTAMPTZ;

ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS outcome TEXT NOT NULL DEFAULT 'success';
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS target_user_id TEXT;

CREATE INDEX IF NOT EXISTS idx_audit_logs_action_created ON audit_logs(action, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs(resource_type, resource_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_target_user ON audit_logs(target_user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_auth_sessions_device ON auth_sessions(user_id, device_fingerprint);
