package api

import (
	"fmt"
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

	if os.Getenv("APP_ENV") != "production" {
		return nil
	}

	checks := map[string][]string{
		"DATABASE_URL":            {"change_me", "smetacheck_change_me", "replace_with"},
		"JWT_SECRET":              {"change_me", "replace_with"},
		"TELEGRAM_WEBHOOK_SECRET": {"change_me", "replace_with"},
	}
	for key, blockedValues := range checks {
		value := strings.TrimSpace(os.Getenv(key))
		if value == "" {
			return fmt.Errorf("%s must be set in production", key)
		}
		for _, blocked := range blockedValues {
			if strings.Contains(strings.ToLower(value), blocked) {
				return fmt.Errorf("%s contains an unsafe placeholder value", key)
			}
		}
	}
	if len(jwtSecret) < 64 {
		return fmt.Errorf("JWT_SECRET must be at least 64 characters in production")
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
