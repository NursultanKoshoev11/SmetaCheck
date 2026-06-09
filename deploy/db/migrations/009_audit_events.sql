CREATE TABLE IF NOT EXISTS audit_events(
id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
actor_id uuid REFERENCES users(id),
action text NOT NULL,
entity text,
entity_id text,
created_at timestamptz DEFAULT now()
);
