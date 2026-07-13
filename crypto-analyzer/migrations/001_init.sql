CREATE TABLE IF NOT EXISTS crypto_schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS crypto_scan_runs (
    id BIGSERIAL PRIMARY KEY,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ,
    status TEXT NOT NULL CHECK (status IN ('running','completed','failed')),
    symbols_selected INTEGER NOT NULL DEFAULT 0,
    symbols_analyzed INTEGER NOT NULL DEFAULT 0,
    signals_created INTEGER NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS crypto_symbols (
    symbol TEXT PRIMARY KEY,
    base_asset TEXT NOT NULL,
    quote_asset TEXT NOT NULL,
    quote_volume_24h DOUBLE PRECISION NOT NULL DEFAULT 0,
    active BOOLEAN NOT NULL DEFAULT true,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS crypto_candles (
    symbol TEXT NOT NULL REFERENCES crypto_symbols(symbol) ON DELETE CASCADE,
    interval TEXT NOT NULL,
    open_time TIMESTAMPTZ NOT NULL,
    close_time TIMESTAMPTZ NOT NULL,
    open DOUBLE PRECISION NOT NULL,
    high DOUBLE PRECISION NOT NULL,
    low DOUBLE PRECISION NOT NULL,
    close DOUBLE PRECISION NOT NULL,
    volume DOUBLE PRECISION NOT NULL,
    quote_volume DOUBLE PRECISION NOT NULL,
    trades BIGINT NOT NULL,
    taker_buy_volume DOUBLE PRECISION NOT NULL,
    PRIMARY KEY (symbol, interval, open_time)
);
CREATE INDEX IF NOT EXISTS crypto_candles_symbol_interval_time_idx
    ON crypto_candles(symbol, interval, open_time DESC);

CREATE TABLE IF NOT EXISTS crypto_interval_analyses (
    id BIGSERIAL PRIMARY KEY,
    scan_run_id BIGINT NOT NULL REFERENCES crypto_scan_runs(id) ON DELETE CASCADE,
    symbol TEXT NOT NULL REFERENCES crypto_symbols(symbol) ON DELETE CASCADE,
    interval TEXT NOT NULL,
    candle_open_time TIMESTAMPTZ NOT NULL,
    price DOUBLE PRECISION NOT NULL,
    action TEXT NOT NULL CHECK (action IN ('BUY','SELL','HOLD')),
    buy_probability DOUBLE PRECISION NOT NULL,
    sell_probability DOUBLE PRECISION NOT NULL,
    buy_baseline DOUBLE PRECISION NOT NULL,
    sell_baseline DOUBLE PRECISION NOT NULL,
    expected_return DOUBLE PRECISION NOT NULL,
    sample_count INTEGER NOT NULL,
    nearest_count INTEGER NOT NULL,
    average_distance DOUBLE PRECISION NOT NULL,
    score DOUBLE PRECISION NOT NULL,
    confidence DOUBLE PRECISION NOT NULL,
    lookahead_bars INTEGER NOT NULL,
    target_pct DOUBLE PRECISION NOT NULL,
    features JSONB NOT NULL,
    reasons JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (scan_run_id, symbol, interval)
);
CREATE INDEX IF NOT EXISTS crypto_interval_analyses_symbol_created_idx
    ON crypto_interval_analyses(symbol, created_at DESC);

CREATE TABLE IF NOT EXISTS crypto_signals (
    id BIGSERIAL PRIMARY KEY,
    scan_run_id BIGINT NOT NULL REFERENCES crypto_scan_runs(id) ON DELETE CASCADE,
    symbol TEXT NOT NULL REFERENCES crypto_symbols(symbol) ON DELETE CASCADE,
    action TEXT NOT NULL CHECK (action IN ('BUY','SELL','HOLD')),
    price DOUBLE PRECISION NOT NULL,
    score DOUBLE PRECISION NOT NULL,
    confidence DOUBLE PRECISION NOT NULL,
    expected_return DOUBLE PRECISION NOT NULL,
    sample_count INTEGER NOT NULL,
    funding_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
    open_interest DOUBLE PRECISION NOT NULL DEFAULT 0,
    open_interest_pct DOUBLE PRECISION NOT NULL DEFAULT 0,
    reasons JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (scan_run_id, symbol)
);
CREATE INDEX IF NOT EXISTS crypto_signals_created_idx ON crypto_signals(created_at DESC);
CREATE INDEX IF NOT EXISTS crypto_signals_symbol_created_idx ON crypto_signals(symbol, created_at DESC);
CREATE INDEX IF NOT EXISTS crypto_signals_action_score_idx ON crypto_signals(action, score);

CREATE TABLE IF NOT EXISTS crypto_telegram_deliveries (
    id BIGSERIAL PRIMARY KEY,
    scan_run_id BIGINT REFERENCES crypto_scan_runs(id) ON DELETE SET NULL,
    chat_id TEXT NOT NULL,
    message_kind TEXT NOT NULL,
    success BOOLEAN NOT NULL,
    response_code INTEGER NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO crypto_schema_migrations(version)
VALUES ('001_init')
ON CONFLICT (version) DO NOTHING;
