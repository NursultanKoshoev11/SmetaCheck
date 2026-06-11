CREATE TABLE IF NOT EXISTS oauth_states (
    state_hash TEXT PRIMARY KEY,
    provider TEXT NOT NULL CHECK (provider IN ('google','telegram')),
    nonce TEXT NOT NULL,
    code_verifier TEXT NOT NULL,
    return_to TEXT NOT NULL DEFAULT '/dashboard',
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_oauth_states_expires_at ON oauth_states(expires_at);
