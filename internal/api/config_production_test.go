package api

import (
	"strings"
	"testing"
)

func setValidProductionEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", "production")
	t.Setenv("DATABASE_URL", "postgres://user:password@postgres:5432/smetacheck?sslmode=disable")
	t.Setenv("DATABASE_MAX_CONNS", "10")
	t.Setenv("DATABASE_MIN_CONNS", "1")
	t.Setenv("JWT_SECRET", strings.Repeat("j", 64))
	t.Setenv("PUBLIC_BASE_URL", "https://app.smetacheck.kg")
	t.Setenv("API_PUBLIC_BASE_URL", "https://api.smetacheck.kg")
	t.Setenv("ALLOWED_ORIGINS", "https://smetacheck.kg,https://app.smetacheck.kg")
	t.Setenv("COOKIE_SECURE", "true")
	t.Setenv("SMTP_HOST", "smtp.smetacheck.kg")
	t.Setenv("SMTP_PORT", "587")
	t.Setenv("SMTP_TLS_MODE", "starttls")
	t.Setenv("SMTP_USERNAME", "smtp-user")
	t.Setenv("SMTP_PASSWORD", "smtp-password")
	t.Setenv("SMTP_FROM", "no-reply@smetacheck.kg")
	t.Setenv("AUTH_GOOGLE_ENABLED", "false")
	t.Setenv("AUTH_TELEGRAM_ENABLED", "false")
	t.Setenv("AI_PROVIDER", "rules")
	t.Setenv("MAX_UPLOAD_MB", "25")
	t.Setenv("MAX_BATCH_FILES", "10")
	t.Setenv("MAX_BATCH_TOTAL_MB", "200")
	t.Setenv("MAX_COMPARE_TOTAL_MB", "60")
	t.Setenv("AI_ROWS_PER_CHUNK", "250")
}

func TestValidProductionConfiguration(t *testing.T) {
	setValidProductionEnvironment(t)
	if err := validateProductionConfig(); err != nil {
		t.Fatalf("expected valid production configuration, got %v", err)
	}
}

func TestProductionRejectsInsecureSMTP(t *testing.T) {
	setValidProductionEnvironment(t)
	t.Setenv("SMTP_TLS_MODE", "none")
	if err := validateProductionConfig(); err == nil || !strings.Contains(err.Error(), "SMTP_TLS_MODE") {
		t.Fatalf("expected SMTP TLS validation error, got %v", err)
	}
}

func TestProductionRequiresSecureCookies(t *testing.T) {
	setValidProductionEnvironment(t)
	t.Setenv("COOKIE_SECURE", "false")
	if err := validateProductionConfig(); err == nil || !strings.Contains(err.Error(), "COOKIE_SECURE") {
		t.Fatalf("expected secure cookie validation error, got %v", err)
	}
}

func TestProductionRejectsInsecureOrigin(t *testing.T) {
	setValidProductionEnvironment(t)
	t.Setenv("ALLOWED_ORIGINS", "http://app.smetacheck.kg")
	if err := validateProductionConfig(); err == nil || !strings.Contains(err.Error(), "ALLOWED_ORIGINS") {
		t.Fatalf("expected origin validation error, got %v", err)
	}
}

func TestProductionRejectsWildcardOrigin(t *testing.T) {
	setValidProductionEnvironment(t)
	t.Setenv("ALLOWED_ORIGINS", "https://*.smetacheck.kg")
	if err := validateProductionConfig(); err == nil || !strings.Contains(err.Error(), "wildcards") {
		t.Fatalf("expected wildcard origin validation error, got %v", err)
	}
}

func TestProductionRejectsLocalPublicURL(t *testing.T) {
	setValidProductionEnvironment(t)
	t.Setenv("PUBLIC_BASE_URL", "https://127.0.0.1")
	if err := validateProductionConfig(); err == nil || !strings.Contains(err.Error(), "local address") {
		t.Fatalf("expected local public URL validation error, got %v", err)
	}
}

func TestProductionRejectsExcessiveUploadLimit(t *testing.T) {
	setValidProductionEnvironment(t)
	t.Setenv("MAX_UPLOAD_MB", "101")
	if err := validateProductionConfig(); err == nil || !strings.Contains(err.Error(), "MAX_UPLOAD_MB") {
		t.Fatalf("expected upload limit validation error, got %v", err)
	}
}

func TestProductionRejectsExcessiveBatchTotalLimit(t *testing.T) {
	setValidProductionEnvironment(t)
	t.Setenv("MAX_BATCH_TOTAL_MB", "501")
	if err := validateProductionConfig(); err == nil || !strings.Contains(err.Error(), "MAX_BATCH_TOTAL_MB") {
		t.Fatalf("expected batch total limit validation error, got %v", err)
	}
}

func TestProductionRejectsExcessiveCompareTotalLimit(t *testing.T) {
	setValidProductionEnvironment(t)
	t.Setenv("MAX_COMPARE_TOTAL_MB", "201")
	if err := validateProductionConfig(); err == nil || !strings.Contains(err.Error(), "MAX_COMPARE_TOTAL_MB") {
		t.Fatalf("expected compare total limit validation error, got %v", err)
	}
}
