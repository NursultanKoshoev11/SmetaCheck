CREATE TABLE IF NOT EXISTS plans(
code text PRIMARY KEY,
name text NOT NULL,
monthly_limit int NOT NULL,
price_kgs int NOT NULL DEFAULT 0
);
INSERT INTO plans(code,name,monthly_limit,price_kgs) VALUES
('free','Free',3,0),('pro','Pro',100,3990)
ON CONFLICT(code) DO NOTHING;
