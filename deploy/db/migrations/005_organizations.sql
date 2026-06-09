CREATE TABLE IF NOT EXISTS organizations(
id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
name text NOT NULL,
owner_id uuid REFERENCES users(id),
created_at timestamptz DEFAULT now()
);
