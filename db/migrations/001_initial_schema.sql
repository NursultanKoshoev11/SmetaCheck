-- SmetaCheck KG initial production schema
-- Apply with: psql "$DATABASE_URL" -f db/migrations/001_initial_schema.sql

CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    full_name TEXT,
    role TEXT NOT NULL DEFAULT 'owner',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
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
    owner_id TEXT REFERENCES users(id) ON DELETE SET NULL,
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
    owner_id TEXT REFERENCES users(id) ON DELETE SET NULL,
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

CREATE INDEX IF NOT EXISTS idx_projects_owner_id ON projects(owner_id);
CREATE INDEX IF NOT EXISTS idx_estimates_owner_id ON estimates(owner_id);
CREATE INDEX IF NOT EXISTS idx_estimates_project_id ON estimates(project_id);
CREATE INDEX IF NOT EXISTS idx_estimates_uploaded_at ON estimates(uploaded_at DESC);
CREATE INDEX IF NOT EXISTS idx_estimate_items_estimate_id ON estimate_items(estimate_id);
CREATE INDEX IF NOT EXISTS idx_findings_estimate_id ON findings(estimate_id);
CREATE INDEX IF NOT EXISTS idx_findings_severity ON findings(severity);
CREATE INDEX IF NOT EXISTS idx_compare_results_created_at ON compare_results(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at DESC);
