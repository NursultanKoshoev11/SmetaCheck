CREATE TABLE IF NOT EXISTS org_users(
user_id uuid REFERENCES users(id),
org_id uuid REFERENCES organizations(id),
user_role text NOT NULL DEFAULT 'member',
created_at timestamptz DEFAULT now(),
PRIMARY KEY(user_id,org_id)
);
