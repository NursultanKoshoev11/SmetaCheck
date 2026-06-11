CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE,
    password_hash TEXT,
    full_name TEXT,
    avatar_url TEXT,
    role TEXT NOT NULL DEFAULT 'owner',
    email_verified_at TIMESTAMPTZ,
    failed_login_attempts INTEGER NOT NULL DEFAULT 0,
    locked_until TIMESTAMPTZ,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE users ALTER COLUMN email DROP NOT NULL;
ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL;
ALTER TABLE users ADD COLUMN IF NOT EXISTS full_name TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'owner';
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS failed_login_attempts INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS locked_until TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE users ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_lower
    ON users (lower(email)) WHERE email IS NOT NULL;

CREATE TABLE IF NOT EXISTS auth_identities (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider TEXT NOT NULL CHECK (provider IN ('email','google','telegram')),
    provider_subject TEXT NOT NULL,
    provider_email TEXT,
    provider_username TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (provider, provider_subject)
);

CREATE TABLE IF NOT EXISTS auth_sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash TEXT NOT NULL UNIQUE,
    ip_address TEXT,
    user_agent TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS auth_tokens (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    purpose TEXT NOT NULL CHECK (purpose IN ('verify_email','reset_password')),
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    owner_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    location TEXT,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS estimates (
    id TEXT PRIMARY KEY,
    owner_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    project_id TEXT REFERENCES projects(id) ON DELETE SET NULL,
    file_name TEXT NOT NULL,
    file_path TEXT NOT NULL,
    report_path TEXT,
    status TEXT NOT NULL DEFAULT 'ready',
    score INTEGER NOT NULL DEFAULT 0 CHECK (score >= 0 AND score <= 100),
    file_size BIGINT NOT NULL DEFAULT 0,
    items_count INTEGER NOT NULL DEFAULT 0,
    total_amount NUMERIC(16,2) NOT NULL DEFAULT 0,
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE estimates ADD COLUMN IF NOT EXISTS owner_id TEXT REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE estimates ADD COLUMN IF NOT EXISTS project_id TEXT REFERENCES projects(id) ON DELETE SET NULL;
ALTER TABLE estimates ADD COLUMN IF NOT EXISTS file_path TEXT NOT NULL DEFAULT '';
ALTER TABLE estimates ADD COLUMN IF NOT EXISTS report_path TEXT;
ALTER TABLE estimates ADD COLUMN IF NOT EXISTS score INTEGER NOT NULL DEFAULT 0;
ALTER TABLE estimates ADD COLUMN IF NOT EXISTS file_size BIGINT NOT NULL DEFAULT 0;
ALTER TABLE estimates ADD COLUMN IF NOT EXISTS items_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE estimates ADD COLUMN IF NOT EXISTS total_amount NUMERIC(16,2) NOT NULL DEFAULT 0;
ALTER TABLE estimates ADD COLUMN IF NOT EXISTS uploaded_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE estimates ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE estimates ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE TABLE IF NOT EXISTS estimate_items (
    id TEXT PRIMARY KEY,
    estimate_id TEXT NOT NULL REFERENCES estimates(id) ON DELETE CASCADE,
    row_number INTEGER NOT NULL,
    name TEXT,
    unit TEXT,
    quantity NUMERIC(16,4) NOT NULL DEFAULT 0,
    unit_price NUMERIC(16,2) NOT NULL DEFAULT 0,
    total NUMERIC(16,2) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS findings (
    id TEXT PRIMARY KEY,
    estimate_id TEXT NOT NULL REFERENCES estimates(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    severity TEXT NOT NULL CHECK (severity IN ('Info','Low','Medium','High')),
    detail TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS compare_results (
    id TEXT PRIMARY KEY,
    owner_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    base_file_name TEXT NOT NULL,
    new_file_name TEXT NOT NULL,
    base_total NUMERIC(16,2) NOT NULL DEFAULT 0,
    new_total NUMERIC(16,2) NOT NULL DEFAULT 0,
    delta_total NUMERIC(16,2) NOT NULL DEFAULT 0,
    added_count INTEGER NOT NULL DEFAULT 0,
    removed_count INTEGER NOT NULL DEFAULT 0,
    changed_count INTEGER NOT NULL DEFAULT 0,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS ai_reports (
    id TEXT PRIMARY KEY,
    owner_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    estimate_id TEXT NOT NULL REFERENCES estimates(id) ON DELETE CASCADE,
    input_hash TEXT NOT NULL,
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    prompt_version TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (owner_id, estimate_id, input_hash, provider, model, prompt_version)
);

CREATE TABLE IF NOT EXISTS audit_logs (
    id TEXT PRIMARY KEY,
    user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    action TEXT NOT NULL,
    resource_type TEXT,
    resource_id TEXT,
    ip_address TEXT,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_auth_identities_user_id ON auth_identities(user_id);
CREATE INDEX IF NOT EXISTS idx_auth_sessions_user_id ON auth_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_auth_sessions_expires_at ON auth_sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_auth_tokens_user_id ON auth_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_auth_tokens_expires_at ON auth_tokens(expires_at);
CREATE INDEX IF NOT EXISTS idx_projects_owner_id ON projects(owner_id);
CREATE INDEX IF NOT EXISTS idx_estimates_owner_id ON estimates(owner_id);
CREATE INDEX IF NOT EXISTS idx_estimates_project_id ON estimates(project_id);
CREATE INDEX IF NOT EXISTS idx_estimates_uploaded_at ON estimates(uploaded_at DESC);
CREATE INDEX IF NOT EXISTS idx_estimate_items_estimate_id ON estimate_items(estimate_id);
CREATE INDEX IF NOT EXISTS idx_findings_estimate_id ON findings(estimate_id);
CREATE INDEX IF NOT EXISTS idx_findings_severity ON findings(severity);
CREATE INDEX IF NOT EXISTS idx_compare_results_owner_id ON compare_results(owner_id);
CREATE INDEX IF NOT EXISTS idx_compare_results_created_at ON compare_results(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_reports_estimate_id ON ai_reports(estimate_id);
CREATE INDEX IF NOT EXISTS idx_ai_reports_owner_id ON ai_reports(owner_id);
CREATE INDEX IF NOT EXISTS idx_ai_reports_created_at ON ai_reports(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at DESC);