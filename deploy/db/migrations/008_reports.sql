CREATE TABLE IF NOT EXISTS reports(
id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
estimate_id uuid REFERENCES estimates(id),
html_path text,
pdf_path text,
risk_score int DEFAULT 0,
created_at timestamptz DEFAULT now()
);
