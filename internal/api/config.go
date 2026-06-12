package api

import (
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

func validateProductionConfig() error {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL must be set; JSON/file storage is not supported")
	}
	jwtSecret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if jwtSecret == "" {
		return fmt.Errorf("JWT_SECRET must be set")
	}
	if len(jwtSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}

	provider := strings.ToLower(strings.TrimSpace(os.Getenv("AI_PROVIDER")))
	if provider == "" {
		provider = "rules"
	}
	keyByProvider := map[string]string{
		"rules": "", "openai": "OPENAI_API_KEY", "gemini": "GEMINI_API_KEY",
		"anthropic": "ANTHROPIC_API_KEY", "claude": "ANTHROPIC_API_KEY",
	}
	keyName, supported := keyByProvider[provider]
	if !supported {
		return fmt.Errorf("AI_PROVIDER must be rules, openai, gemini, anthropic or claude")
	}
	if keyName != "" && strings.TrimSpace(os.Getenv(keyName)) == "" {
		return fmt.Errorf("%s must be set when AI_PROVIDER=%s", keyName, provider)
	}
	if envInt64("MAX_UPLOAD_MB", 25) < 1 || envInt64("MAX_UPLOAD_MB", 25) > 100 {
		return fmt.Errorf("MAX_UPLOAD_MB must be between 1 and 100")
	}
	if envInt64("MAX_BATCH_FILES", 10) < 1 || envInt64("MAX_BATCH_FILES", 10) > 50 {
		return fmt.Errorf("MAX_BATCH_FILES must be between 1 and 50")
	}
	if envInt64("AI_ROWS_PER_CHUNK", 250) < 25 || envInt64("AI_ROWS_PER_CHUNK", 250) > 1000 {
		return fmt.Errorf("AI_ROWS_PER_CHUNK must be between 25 and 1000")
	}
	if envInt64("DATABASE_MAX_CONNS", 10) < 2 || envInt64("DATABASE_MAX_CONNS", 10) > 100 {
		return fmt.Errorf("DATABASE_MAX_CONNS must be between 2 and 100")
	}
	if envInt64("DATABASE_MIN_CONNS", 1) < 0 || envInt64("DATABASE_MIN_CONNS", 1) > envInt64("DATABASE_MAX_CONNS", 10) {
		return fmt.Errorf("DATABASE_MIN_CONNS must be between 0 and DATABASE_MAX_CONNS")
	}
	if err := validateOptionalDuration("OAUTH_HTTP_TIMEOUT", 2*time.Second, 60*time.Second); err != nil {
		return err
	}
	if err := validateOptionalDuration("OAUTH_STATE_TTL", 2*time.Minute, 30*time.Minute); err != nil {
		return err
	}
	if err := validateOptionalDuration("SERVER_SHUTDOWN_TIMEOUT", time.Second, 2*time.Minute); err != nil {
		return err
	}
	if err := validateOptionalBoolean("AUTH_GOOGLE_ENABLED"); err != nil {
		return err
	}
	if err := validateOptionalBoolean("AUTH_TELEGRAM_ENABLED"); err != nil {
		return err
	}

	if os.Getenv("APP_ENV") != "production" {
		return nil
	}

	required := []string{
		"PUBLIC_BASE_URL", "API_PUBLIC_BASE_URL", "DATABASE_URL", "JWT_SECRET", "ALLOWED_ORIGINS",
		"SMTP_HOST", "SMTP_PORT", "SMTP_USERNAME", "SMTP_PASSWORD", "SMTP_FROM",
	}
	for _, key := range required {
		if err := validateRequiredProductionValue(key); err != nil {
			return err
		}
	}
	if authProviderEnabled("google") {
		for _, key := range []string{"GOOGLE_OIDC_CLIENT_ID", "GOOGLE_OIDC_CLIENT_SECRET", "GOOGLE_OIDC_REDIRECT_URL"} {
			if err := validateRequiredProductionValue(key); err != nil {
				return err
			}
		}
		if _, err := loadOIDCProviderConfig("google"); err != nil {
			return err
		}
	}
	if authProviderEnabled("telegram") {
		for _, key := range []string{"TELEGRAM_OIDC_CLIENT_ID", "TELEGRAM_OIDC_CLIENT_SECRET", "TELEGRAM_OIDC_REDIRECT_URL"} {
			if err := validateRequiredProductionValue(key); err != nil {
				return err
			}
		}
		if issuer := strings.TrimSpace(os.Getenv("TELEGRAM_OIDC_ISSUER")); issuer != "" && strings.TrimRight(issuer, "/") != telegramOIDCIssuer {
			return fmt.Errorf("TELEGRAM_OIDC_ISSUER must be %s", telegramOIDCIssuer)
		}
		if _, err := loadOIDCProviderConfig("telegram"); err != nil {
			return err
		}
	}
	if keyName != "" {
		if err := validateRequiredProductionValue(keyName); err != nil {
			return err
		}
	}
	if len(jwtSecret) < 64 {
		return fmt.Errorf("JWT_SECRET must be at least 64 characters in production")
	}
	if err := validateProductionBaseURL("PUBLIC_BASE_URL"); err != nil {
		return err
	}
	if err := validateProductionBaseURL("API_PUBLIC_BASE_URL"); err != nil {
		return err
	}
	if err := validateProductionOrigins(os.Getenv("ALLOWED_ORIGINS")); err != nil {
		return err
	}
	cookieSecure, err := strconv.ParseBool(strings.TrimSpace(os.Getenv("COOKIE_SECURE")))
	if err != nil || !cookieSecure {
		return fmt.Errorf("COOKIE_SECURE must be true in production")
	}
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("SMTP_TLS_MODE")))
	if mode == "" {
		mode = "starttls"
	}
	if mode != "starttls" && mode != "implicit" {
		return fmt.Errorf("SMTP_TLS_MODE must be starttls or implicit in production")
	}
	if _, err := mail.ParseAddress(strings.TrimSpace(os.Getenv("SMTP_FROM"))); err != nil {
		return fmt.Errorf("SMTP_FROM must be a valid email address")
	}
	return nil
}

func validateRequiredProductionValue(key string) error {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fmt.Errorf("%s must be set in production", key)
	}
	lower := strings.ToLower(value)
	for _, blocked := range []string{"change_me", "replace_with", "your_api_key", "example.com"} {
		if strings.Contains(lower, blocked) {
			return fmt.Errorf("%s contains an unsafe placeholder value", key)
		}
	}
	return nil
}

func validateProductionBaseURL(key string) error {
	value := strings.TrimSpace(os.Getenv(key))
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return fmt.Errorf("%s must be an absolute https URL", key)
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return fmt.Errorf("%s must contain only an https origin", key)
	}
	if isLocalHostname(parsed.Hostname()) {
		return fmt.Errorf("%s cannot use a local address in production", key)
	}
	return nil
}

func validateProductionOrigins(value string) error {
	seen := 0
	for _, rawOrigin := range strings.Split(value, ",") {
		origin := strings.TrimSpace(rawOrigin)
		if origin == "" {
			continue
		}
		seen++
		if origin == "*" || strings.Contains(origin, "*") {
			return fmt.Errorf("ALLOWED_ORIGINS cannot contain wildcards in production")
		}
		parsed, err := url.Parse(origin)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
			return fmt.Errorf("ALLOWED_ORIGINS entries must be absolute https origins")
		}
		if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
			return fmt.Errorf("ALLOWED_ORIGINS entries cannot contain paths, credentials, queries or fragments")
		}
		if isLocalHostname(parsed.Hostname()) {
			return fmt.Errorf("ALLOWED_ORIGINS cannot contain local addresses in production")
		}
	}
	if seen == 0 {
		return fmt.Errorf("ALLOWED_ORIGINS must contain at least one production origin")
	}
	return nil
}

func isLocalHostname(hostname string) bool {
	hostname = strings.ToLower(strings.TrimSpace(hostname))
	if hostname == "localhost" || strings.HasSuffix(hostname, ".localhost") {
		return true
	}
	if address := net.ParseIP(hostname); address != nil {
		return address.IsLoopback() || address.IsUnspecified() || address.IsPrivate()
	}
	return false
}

func validateOptionalBoolean(key string) error {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return nil
	}
	if _, err := strconv.ParseBool(value); err != nil {
		return fmt.Errorf("%s must be true or false", key)
	}
	return nil
}

func validateOptionalDuration(key string, minimum, maximum time.Duration) error {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return fmt.Errorf("%s must be a valid duration", key)
	}
	if duration < minimum || duration > maximum {
		return fmt.Errorf("%s must be between %s and %s", key, minimum, maximum)
	}
	return nil
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return duration
}

func envInt64(key string, fallback int64) int64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}
