CREATE TABLE IF NOT EXISTS organizations(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 name text NOT NULL,
 owner_id uuid NOT NULL REFERENCES users(id),
 tariff text NOT NULL DEFAULT 'free',
 created_at timestamptz NOT NULL DEFAULT now()
);
