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

	provider := strings.ToLower(strings.TrimSpace(os.Getenv("AI_PROVIDER")))
	if provider == "" { provider = "rules" }
	keyByProvider := map[string]string{
		"rules": "",
		"openai": "OPENAI_API_KEY",
		"gemini": "GEMINI_API_KEY",
		"anthropic": "ANTHROPIC_API_KEY",
		"claude": "ANTHROPIC_API_KEY",
	}
	keyName, supported := keyByProvider[provider]
	if !supported {
		return fmt.Errorf("AI_PROVIDER must be rules, openai, gemini, anthropic or claude")
	}
	if keyName != "" && strings.TrimSpace(os.Getenv(keyName)) == "" {
		return fmt.Errorf("%s must be set when AI_PROVIDER=%s", keyName, provider)
	}
	if envInt64("MAX_BATCH_FILES",10)<1 || envInt64("MAX_BATCH_FILES",10)>50 {
		return fmt.Errorf("MAX_BATCH_FILES must be between 1 and 50")
	}
	if envInt64("AI_ROWS_PER_CHUNK",250)<25 || envInt64("AI_ROWS_PER_CHUNK",250)>1000 {
		return fmt.Errorf("AI_ROWS_PER_CHUNK must be between 25 and 1000")
	}

	if os.Getenv("APP_ENV") != "production" { return nil }
	checks := map[string][]string{
		"DATABASE_URL": {"change_me","smetacheck_change_me","replace_with"},
		"JWT_SECRET": {"change_me","replace_with"},
		"TELEGRAM_WEBHOOK_SECRET": {"change_me","replace_with"},
	}
	if keyName!="" { checks[keyName]=[]string{"change_me","replace_with","your_api_key"} }
	for key,blockedValues:=range checks {
		value:=strings.TrimSpace(os.Getenv(key))
		if value=="" { return fmt.Errorf("%s must be set in production",key) }
		for _,blocked:=range blockedValues {
			if strings.Contains(strings.ToLower(value),blocked) { return fmt.Errorf("%s contains an unsafe placeholder value",key) }
		}
	}
	if len(jwtSecret)<64 { return fmt.Errorf("JWT_SECRET must be at least 64 characters in production") }
	return nil
}

func envDuration(key string,fallback time.Duration) time.Duration {
	value:=strings.TrimSpace(os.Getenv(key));if value==""{return fallback}
	duration,err:=time.ParseDuration(value);if err!=nil{return fallback};return duration
}

func envInt64(key string,fallback int64) int64 {
	value:=strings.TrimSpace(os.Getenv(key));if value==""{return fallback}
	parsed,err:=strconv.ParseInt(value,10,64);if err!=nil{return fallback};return parsed
}
