package api

import (
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

func TestSafeReturnToAcceptsOnlyLocalPaths(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{name: "dashboard", value: "/dashboard", expected: "/dashboard"},
		{name: "path with query", value: "/reports?id=rep_1", expected: "/reports?id=rep_1"},
		{name: "absolute URL", value: "https://evil.example/path", expected: "/dashboard"},
		{name: "scheme relative", value: "//evil.example/path", expected: "/dashboard"},
		{name: "encoded scheme relative", value: "/%2fevil.example/path", expected: "/dashboard"},
		{name: "backslash", value: "/\\evil.example", expected: "/dashboard"},
		{name: "control character", value: "/dashboard\nLocation: https://evil.example", expected: "/dashboard"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if actual := safeReturnTo(test.value); actual != test.expected {
				t.Fatalf("safeReturnTo(%q) = %q, expected %q", test.value, actual, test.expected)
			}
		})
	}
}

func TestTelegramOIDCConfigUsesOfficialIssuerAndBasicAuth(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("AUTH_TELEGRAM_ENABLED", "true")
	t.Setenv("API_PUBLIC_BASE_URL", "https://api.smetacheck.kg")
	t.Setenv("TELEGRAM_OIDC_ISSUER", telegramOIDCIssuer)
	t.Setenv("TELEGRAM_OIDC_CLIENT_ID", "telegram-client")
	t.Setenv("TELEGRAM_OIDC_CLIENT_SECRET", "telegram-secret")
	t.Setenv("TELEGRAM_OIDC_REDIRECT_URL", "https://api.smetacheck.kg/v1/auth/telegram/callback")

	cfg, err := loadOIDCProviderConfig("telegram")
	if err != nil {
		t.Fatalf("loadOIDCProviderConfig returned error: %v", err)
	}
	if cfg.Issuer != telegramOIDCIssuer {
		t.Fatalf("unexpected Telegram issuer: %q", cfg.Issuer)
	}
	if cfg.AuthStyle != oauth2.AuthStyleInHeader {
		t.Fatalf("Telegram token exchange must use HTTP Basic authentication")
	}
	if strings.Join(cfg.Scopes, " ") != "openid profile" {
		t.Fatalf("unexpected Telegram scopes: %v", cfg.Scopes)
	}
}

func TestTelegramOIDCRejectsCustomIssuer(t *testing.T) {
	t.Setenv("AUTH_TELEGRAM_ENABLED", "true")
	t.Setenv("TELEGRAM_OIDC_ISSUER", "https://attacker.example")
	t.Setenv("TELEGRAM_OIDC_CLIENT_ID", "telegram-client")
	t.Setenv("TELEGRAM_OIDC_CLIENT_SECRET", "telegram-secret")
	t.Setenv("TELEGRAM_OIDC_REDIRECT_URL", "https://api.smetacheck.kg/v1/auth/telegram/callback")

	if _, err := loadOIDCProviderConfig("telegram"); err == nil {
		t.Fatal("expected custom Telegram issuer to be rejected")
	}
}

func TestGoogleOIDCConfigUsesVerifiedEmailScopes(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("AUTH_GOOGLE_ENABLED", "true")
	t.Setenv("API_PUBLIC_BASE_URL", "https://api.smetacheck.kg")
	t.Setenv("GOOGLE_OIDC_CLIENT_ID", "google-client")
	t.Setenv("GOOGLE_OIDC_CLIENT_SECRET", "google-secret")
	t.Setenv("GOOGLE_OIDC_REDIRECT_URL", "https://api.smetacheck.kg/v1/auth/google/callback")

	cfg, err := loadOIDCProviderConfig("google")
	if err != nil {
		t.Fatalf("loadOIDCProviderConfig returned error: %v", err)
	}
	if cfg.Issuer != googleOIDCIssuer {
		t.Fatalf("unexpected Google issuer: %q", cfg.Issuer)
	}
	if strings.Join(cfg.Scopes, " ") != "openid email profile" {
		t.Fatalf("unexpected Google scopes: %v", cfg.Scopes)
	}
}

func TestOIDCRedirectMustMatchProductionAPIBase(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("API_PUBLIC_BASE_URL", "https://api.smetacheck.kg")

	if err := validateOIDCRedirectURL("https://api.smetacheck.kg/v1/auth/google/callback", "google"); err != nil {
		t.Fatalf("expected valid redirect URL, got %v", err)
	}
	if err := validateOIDCRedirectURL("https://other.example/v1/auth/google/callback", "google"); err == nil {
		t.Fatal("expected redirect on another origin to be rejected")
	}
	if err := validateOIDCRedirectURL("http://api.smetacheck.kg/v1/auth/google/callback", "google"); err == nil {
		t.Fatal("expected insecure redirect URL to be rejected")
	}
}

func TestValidateOIDCClaims(t *testing.T) {
	claims, err := validateOIDCClaims("google", oidcClaims{
		Subject:       "google-subject",
		Email:         "USER@EXAMPLE.COM",
		EmailVerified: true,
		Name:          " Test User ",
		Picture:       "https://images.example/avatar.png",
		Nonce:         "expected-nonce",
	}, "expected-nonce")
	if err != nil {
		t.Fatalf("validateOIDCClaims returned error: %v", err)
	}
	if claims.Email != "user@example.com" {
		t.Fatalf("unexpected normalized email: %q", claims.Email)
	}
	if claims.Name != "Test User" {
		t.Fatalf("unexpected normalized name: %q", claims.Name)
	}

	_, err = validateOIDCClaims("google", oidcClaims{
		Subject: "google-subject",
		Email:   "user@example.com",
		Nonce:   "expected-nonce",
	}, "expected-nonce")
	if err == nil {
		t.Fatal("expected Google account without verified email to be rejected")
	}

	_, err = validateOIDCClaims("telegram", oidcClaims{
		Subject: "telegram-subject",
		Name:    "Telegram User",
		Nonce:   "wrong-nonce",
	}, "expected-nonce")
	if err == nil {
		t.Fatal("expected nonce mismatch to be rejected")
	}
}

func TestAuthProviderFeatureFlags(t *testing.T) {
	t.Setenv("AUTH_GOOGLE_ENABLED", "false")
	t.Setenv("AUTH_TELEGRAM_ENABLED", "true")
	if authProviderEnabled("google") {
		t.Fatal("Google provider should be disabled")
	}
	if !authProviderEnabled("telegram") {
		t.Fatal("Telegram provider should be enabled")
	}
}
