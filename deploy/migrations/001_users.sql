CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE TABLE IF NOT EXISTS users(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 email text UNIQUE NOT NULL,
 secret_digest text NOT NULL,
 full_name text NOT NULL DEFAULT '',
 role text NOT NULL DEFAULT 'user',
 created_at timestamptz NOT NULL DEFAULT now()
);
