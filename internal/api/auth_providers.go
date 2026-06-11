package api

import (
	"net/http"
	"os"
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
	return strings.TrimSpace(os.Getenv("SMTP_HOST")) != "" &&
		strings.TrimSpace(os.Getenv("SMTP_PORT")) != "" &&
		strings.TrimSpace(os.Getenv("SMTP_FROM")) != ""
}

func googleOIDCConfigured() bool {
	return strings.TrimSpace(os.Getenv("GOOGLE_OIDC_CLIENT_ID")) != "" &&
		strings.TrimSpace(os.Getenv("GOOGLE_OIDC_CLIENT_SECRET")) != "" &&
		googleRedirectURL() != ""
}

func telegramOIDCConfigured() bool {
	return strings.TrimSpace(os.Getenv("TELEGRAM_OIDC_ISSUER")) != "" &&
		strings.TrimSpace(os.Getenv("TELEGRAM_OIDC_CLIENT_ID")) != "" &&
		strings.TrimSpace(os.Getenv("TELEGRAM_OIDC_CLIENT_SECRET")) != "" &&
		telegramRedirectURL() != ""
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
