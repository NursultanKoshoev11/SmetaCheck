package api

import (
	"net/http"
	"os"
	"strconv"
	"strings"
)

type authProviderStatus struct {
	EmailLogin        bool `json:"email_login"`
	EmailRegistration bool `json:"email_registration"`
	PasswordReset     bool `json:"password_reset"`
	Google            bool `json:"google"`
	Telegram          bool `json:"telegram"`
}

func AuthProviders(w http.ResponseWriter, r *http.Request) {
	status := authProviderStatus{
		EmailLogin:        true,
		EmailRegistration: smtpConfigured(),
		PasswordReset:     smtpConfigured(),
		Google:            googleOIDCConfigured(),
		Telegram:          telegramOIDCConfigured(),
	}
	estimateWriteJSON(w, http.StatusOK, map[string]any{"providers": status})
}

func smtpConfigured() bool {
	if strings.TrimSpace(os.Getenv("SMTP_HOST")) == "" ||
		strings.TrimSpace(os.Getenv("SMTP_PORT")) == "" ||
		strings.TrimSpace(os.Getenv("SMTP_FROM")) == "" {
		return false
	}
	username := strings.TrimSpace(os.Getenv("SMTP_USERNAME"))
	password := strings.TrimSpace(os.Getenv("SMTP_PASSWORD"))
	if os.Getenv("APP_ENV") == "production" {
		return username != "" && password != ""
	}
	return (username == "") == (password == "")
}

func authProviderEnabled(providerName string) bool {
	var key string
	switch providerName {
	case "google":
		key = "AUTH_GOOGLE_ENABLED"
	case "telegram":
		key = "AUTH_TELEGRAM_ENABLED"
	default:
		return false
	}
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return true
	}
	enabled, err := strconv.ParseBool(value)
	return err == nil && enabled
}

func googleOIDCConfigured() bool {
	if !authProviderEnabled("google") {
		return false
	}
	_, err := loadOIDCProviderConfig("google")
	return err == nil
}

func telegramOIDCConfigured() bool {
	if !authProviderEnabled("telegram") {
		return false
	}
	_, err := loadOIDCProviderConfig("telegram")
	return err == nil
}

func googleRedirectURL() string {
	if value := strings.TrimSpace(os.Getenv("GOOGLE_OIDC_REDIRECT_URL")); value != "" {
		return value
	}
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("API_PUBLIC_BASE_URL")), "/")
	if base == "" {
		return ""
	}
	return base + "/v1/auth/google/callback"
}

func telegramRedirectURL() string {
	if value := strings.TrimSpace(os.Getenv("TELEGRAM_OIDC_REDIRECT_URL")); value != "" {
		return value
	}
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("API_PUBLIC_BASE_URL")), "/")
	if base == "" {
		return ""
	}
	return base + "/v1/auth/telegram/callback"
}
