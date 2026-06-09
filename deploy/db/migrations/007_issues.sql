CREATE TABLE IF NOT EXISTS issues(
id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
estimate_id uuid REFERENCES estimates(id),
row_no int,
severity text NOT NULL,
issue_type text NOT NULL,
message text NOT NULL,
recommendation text,
created_at timestamptz DEFAULT now()
);
