package api

import (
	"strings"
	"testing"
)

func TestProductionConfigRequiresSMTPCredentials(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("DATABASE_URL", "postgres://user:password@localhost:5432/smetacheck")
	t.Setenv("JWT_SECRET", strings.Repeat("j", 64))
	t.Setenv("PUBLIC_BASE_URL", "https://app.smetacheck.kg")
	t.Setenv("API_PUBLIC_BASE_URL", "https://api.smetacheck.kg")
	t.Setenv("ALLOWED_ORIGINS", "https://app.smetacheck.kg")
	t.Setenv("SMTP_HOST", "smtp.example.test")
	t.Setenv("SMTP_PORT", "587")
	t.Setenv("SMTP_FROM", "no-reply@smetacheck.kg")
	t.Setenv("SMTP_USERNAME", "")
	t.Setenv("SMTP_PASSWORD", "")
	t.Setenv("GOOGLE_OIDC_CLIENT_ID", "google-client")
	t.Setenv("GOOGLE_OIDC_CLIENT_SECRET", "google-secret")
	t.Setenv("GOOGLE_OIDC_REDIRECT_URL", "https://api.smetacheck.kg/v1/auth/google/callback")
	t.Setenv("TELEGRAM_OIDC_ISSUER", "https://oauth.telegram.org")
	t.Setenv("TELEGRAM_OIDC_CLIENT_ID", "telegram-client")
	t.Setenv("TELEGRAM_OIDC_CLIENT_SECRET", "telegram-secret")
	t.Setenv("TELEGRAM_OIDC_REDIRECT_URL", "https://api.smetacheck.kg/v1/auth/telegram/callback")
	t.Setenv("AI_PROVIDER", "rules")

	err := validateProductionConfig()
	if err == nil || !strings.Contains(err.Error(), "SMTP_USERNAME") {
		t.Fatalf("expected missing SMTP_USERNAME error, got %v", err)
	}
}
