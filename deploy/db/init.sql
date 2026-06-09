CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE TABLE users(id uuid PRIMARY KEY DEFAULT gen_random_uuid(),email text UNIQUE NOT NULL,password_hash text NOT NULL);
CREATE TABLE estimates(id uuid PRIMARY KEY DEFAULT gen_random_uuid(),user_id uuid REFERENCES users(id),file_name text NOT NULL,status text NOT NULL DEFAULT 'queued');
