package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppName             string
	Environment         string
	HTTPAddr            string
	APIKey              string
	DatabaseURL         string
	BinanceSpotBaseURL  string
	BinanceFuturesURL   string
	QuoteAsset          string
	Intervals           []string
	MaxSymbols          int
	MinQuoteVolume      float64
	KlineLimit          int
	RequestConcurrency  int
	RequestDelay        time.Duration
	HTTPTimeout         time.Duration
	ScanMinute          int
	RunOnStartup        bool
	NearestNeighbors    int
	MinimumSamples      int
	MinimumProbability  float64
	MinimumEdge         float64
	SignalThreshold     float64
	ReportTopN          int
	LookaheadByInterval map[string]int
	TargetByInterval    map[string]float64
	WeightByInterval    map[string]float64
	EnableFutures       bool
	FuturesEnrichLimit  int
	TelegramToken       string
	TelegramChatID      string
	TelegramPolling     bool
	TelegramSendHolds   bool
	LogLevel            string
	ExcludedBaseAssets  map[string]struct{}
}

func Load() (Config, error) {
	cfg := Config{
		AppName:            env("APP_NAME", "Crypto Pattern Analyzer"),
		Environment:        env("APP_ENV", "production"),
		HTTPAddr:           env("HTTP_ADDR", ":8080"),
		APIKey:             strings.TrimSpace(os.Getenv("API_KEY")),
		DatabaseURL:        databaseURL(),
		BinanceSpotBaseURL: strings.TrimRight(env("BINANCE_SPOT_BASE_URL", "https://api.binance.com"), "/"),
		BinanceFuturesURL:  strings.TrimRight(env("BINANCE_FUTURES_BASE_URL", "https://fapi.binance.com"), "/"),
		QuoteAsset:         strings.ToUpper(env("QUOTE_ASSET", "USDT")),
		Intervals:          splitCSV(env("ANALYSIS_INTERVALS", "15m,1h,4h,1d")),
		MaxSymbols:         envInt("MAX_SYMBOLS", 150),
		MinQuoteVolume:     envFloat("MIN_24H_QUOTE_VOLUME", 1_000_000),
		KlineLimit:         envInt("KLINE_LIMIT", 1000),
		RequestConcurrency: envInt("REQUEST_CONCURRENCY", 6),
		RequestDelay:       envDuration("REQUEST_DELAY", 120*time.Millisecond),
		HTTPTimeout:        envDuration("HTTP_TIMEOUT", 20*time.Second),
		ScanMinute:         envInt("SCAN_MINUTE", 2),
		RunOnStartup:       envBool("RUN_ON_STARTUP", true),
		NearestNeighbors:   envInt("NEAREST_NEIGHBORS", 40),
		MinimumSamples:     envInt("MINIMUM_SAMPLES", 120),
		MinimumProbability: envFloat("MINIMUM_PROBABILITY", 0.58),
		MinimumEdge:        envFloat("MINIMUM_EDGE", 0.06),
		SignalThreshold:    envFloat("MULTITF_SIGNAL_THRESHOLD", 0.12),
		ReportTopN:         envInt("REPORT_TOP_N", 10),
		EnableFutures:      envBool("ENABLE_FUTURES_METRICS", true),
		FuturesEnrichLimit: envInt("FUTURES_ENRICH_LIMIT", 30),
		TelegramToken:      strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN")),
		TelegramChatID:     strings.TrimSpace(os.Getenv("TELEGRAM_CHAT_ID")),
		TelegramPolling:    envBool("TELEGRAM_COMMANDS_ENABLED", true),
		TelegramSendHolds:  envBool("TELEGRAM_SEND_EMPTY_REPORT", true),
		LogLevel:           strings.ToLower(env("LOG_LEVEL", "info")),
		LookaheadByInterval: map[string]int{
			"15m": envInt("LOOKAHEAD_15M", 16),
			"1h":  envInt("LOOKAHEAD_1H", 12),
			"4h":  envInt("LOOKAHEAD_4H", 6),
			"1d":  envInt("LOOKAHEAD_1D", 7),
		},
		TargetByInterval: map[string]float64{
			"15m": envFloat("TARGET_PCT_15M", 1.5),
			"1h":  envFloat("TARGET_PCT_1H", 3.0),
			"4h":  envFloat("TARGET_PCT_4H", 5.0),
			"1d":  envFloat("TARGET_PCT_1D", 8.0),
		},
		WeightByInterval: map[string]float64{
			"15m": envFloat("WEIGHT_15M", 0.15),
			"1h":  envFloat("WEIGHT_1H", 0.35),
			"4h":  envFloat("WEIGHT_4H", 0.35),
			"1d":  envFloat("WEIGHT_1D", 0.15),
		},
		ExcludedBaseAssets: makeSet(splitCSV(env("EXCLUDED_BASE_ASSETS", "USDC,FDUSD,TUSD,USDP,DAI,EUR,TRY,BRL,GBP,AUD,RUB,BIDR,IDRT,UAH,NGN,BVND"))),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL or POSTGRES_* settings are required")
	}
	if cfg.ScanMinute < 0 || cfg.ScanMinute > 59 {
		return Config{}, fmt.Errorf("SCAN_MINUTE must be between 0 and 59")
	}
	if cfg.KlineLimit < 200 || cfg.KlineLimit > 1000 {
		return Config{}, fmt.Errorf("KLINE_LIMIT must be between 200 and 1000")
	}
	if cfg.RequestConcurrency < 1 || cfg.RequestConcurrency > 30 {
		return Config{}, fmt.Errorf("REQUEST_CONCURRENCY must be between 1 and 30")
	}
	if cfg.NearestNeighbors < 10 {
		return Config{}, fmt.Errorf("NEAREST_NEIGHBORS must be at least 10")
	}
	if cfg.MinimumSamples < 80 {
		return Config{}, fmt.Errorf("MINIMUM_SAMPLES must be at least 80")
	}
	for _, interval := range cfg.Intervals {
		if _, ok := cfg.LookaheadByInterval[interval]; !ok {
			return Config{}, fmt.Errorf("unsupported ANALYSIS_INTERVALS value %q; supported: 15m,1h,4h,1d", interval)
		}
	}
	return cfg, nil
}

func databaseURL() string {
	if value := strings.TrimSpace(os.Getenv("DATABASE_URL")); value != "" {
		return value
	}
	user := env("POSTGRES_USER", "crypto")
	password := env("POSTGRES_PASSWORD", "crypto-change-me")
	host := env("POSTGRES_HOST", "postgres")
	port := env("POSTGRES_PORT", "5432")
	database := env("POSTGRES_DB", "crypto_analyzer")
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   host + ":" + port,
		Path:   database,
	}
	query := u.Query()
	query.Set("sslmode", env("POSTGRES_SSLMODE", "disable"))
	u.RawQuery = query.Encode()
	return u.String()
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return n
}

func envFloat(key string, fallback float64) float64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	n, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return n
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	n, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return n
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return d
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func makeSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[strings.ToUpper(strings.TrimSpace(value))] = struct{}{}
	}
	return set
}
