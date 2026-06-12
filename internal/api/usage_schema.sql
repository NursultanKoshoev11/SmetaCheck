CREATE TABLE IF NOT EXISTS account_storage_usage (
    user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    storage_bytes BIGINT NOT NULL DEFAULT 0 CHECK (storage_bytes >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS account_usage_monthly (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    period_start DATE NOT NULL,
    upload_files BIGINT NOT NULL DEFAULT 0 CHECK (upload_files >= 0),
    upload_bytes BIGINT NOT NULL DEFAULT 0 CHECK (upload_bytes >= 0),
    ai_jobs BIGINT NOT NULL DEFAULT 0 CHECK (ai_jobs >= 0),
    batch_jobs BIGINT NOT NULL DEFAULT 0 CHECK (batch_jobs >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, period_start)
);

CREATE INDEX IF NOT EXISTS idx_account_usage_monthly_period
    ON account_usage_monthly(period_start);

INSERT INTO account_storage_usage (user_id, storage_bytes)
SELECT owner_id, COALESCE(SUM(file_size), 0)::BIGINT
FROM (
    SELECT owner_id, file_path, MAX(file_size)::BIGINT AS file_size
    FROM (
        SELECT owner_id, file_path, file_size
        FROM estimates
        WHERE file_path <> ''
        UNION ALL
        SELECT b.owner_id, f.file_path, f.file_size
        FROM analysis_batch_files f
        JOIN analysis_batches b ON b.id = f.batch_id
        WHERE f.file_path <> ''
    ) stored_files
    GROUP BY owner_id, file_path
) distinct_files
GROUP BY owner_id
ON CONFLICT (user_id) DO NOTHING;
