ALTER TABLE estimates ADD COLUMN IF NOT EXISTS file_path text;
ALTER TABLE estimates ADD COLUMN IF NOT EXISTS created_at timestamptz DEFAULT now();
