package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/jackc/pgx/v5"
	"golang.org/x/oauth2"
)

const (
	googleOIDCIssuer   = "https://accounts.google.com"
	telegramOIDCIssuer = "https://oauth.telegram.org"
)

type oidcProviderConfig struct {
	Name         string
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
	AuthStyle    oauth2.AuthStyle
}

type oidcClaims struct {
	Subject           string `json:"sub"`
	Email             string `json:"email"`
	EmailVerified     bool   `json:"email_verified"`
	Name              string `json:"name"`
	Picture           string `json:"picture"`
	PreferredUsername string `json:"preferred_username"`
	Nonce             string `json:"nonce"`
}

func AuthGoogleBegin(w http.ResponseWriter, r *http.Request) {
	beginOIDC(w, r, "google")
}

func AuthGoogleCallback(w http.ResponseWriter, r *http.Request) {
	finishOIDC(w, r, "google")
}

func AuthTelegramBegin(w http.ResponseWriter, r *http.Request) {
	beginOIDC(w, r, "telegram")
}

func AuthTelegramCallback(w http.ResponseWriter, r *http.Request) {
	finishOIDC(w, r, "telegram")
}

func beginOIDC(w http.ResponseWriter, r *http.Request, providerName string) {
	cfg, err := loadOIDCProviderConfig(providerName)
	if err != nil {
		redirectOAuthFailure(w, r, providerName, "provider_unavailable", err)
		return
	}

	oauthCtx, cancel := oauthRequestContext(r.Context())
	provider, err := oidc.NewProvider(oauthCtx, cfg.Issuer)
	cancel()
	if err != nil {
		redirectOAuthFailure(w, r, providerName, "provider_unavailable", err)
		return
	}

	state, err := randomURLToken(32)
	if err != nil {
		redirectOAuthFailure(w, r, providerName, "internal_error", err)
		return
	}
	nonce, err := randomURLToken(32)
	if err != nil {
		redirectOAuthFailure(w, r, providerName, "internal_error", err)
		return
	}
	verifier := oauth2.GenerateVerifier()
	returnTo := safeReturnTo(r.URL.Query().Get("return_to"))

	pool, err := getDB(r.Context())
	if err != nil || pool == nil {
		redirectOAuthFailure(w, r, providerName, "service_unavailable", err)
		return
	}
	stateTTL := envDuration("OAUTH_STATE_TTL", 10*time.Minute)
	if stateTTL < 2*time.Minute || stateTTL > 30*time.Minute {
		stateTTL = 10 * time.Minute
	}
	_, err = pool.Exec(r.Context(), `
		INSERT INTO oauth_states (state_hash,provider,nonce,code_verifier,return_to,expires_at)
		VALUES ($1,$2,$3,$4,$5,$6)
	`, hashToken(state), providerName, nonce, verifier, returnTo, time.Now().UTC().Add(stateTTL))
	if err != nil {
		redirectOAuthFailure(w, r, providerName, "service_unavailable", err)
		return
	}

	oauthConfig := oauthConfigForProvider(cfg, provider)
	options := []oauth2.AuthCodeOption{
		oauth2.SetAuthURLParam("nonce", nonce),
		oauth2.S256ChallengeOption(verifier),
	}
	if providerName == "google" {
		options = append(options, oauth2.AccessTypeOnline)
	}
	http.Redirect(w, r, oauthConfig.AuthCodeURL(state, options...), http.StatusFound)
}

func finishOIDC(w http.ResponseWriter, r *http.Request, providerName string) {
	if providerError := strings.TrimSpace(r.URL.Query().Get("error")); providerError != "" {
		redirectOAuthFailure(w, r, providerName, "cancelled", nil)
		return
	}
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if state == "" || code == "" || len(state) > 512 || len(code) > 4096 {
		redirectOAuthFailure(w, r, providerName, "invalid_callback", nil)
		return
	}

	pool, err := getDB(r.Context())
	if err != nil || pool == nil {
		redirectOAuthFailure(w, r, providerName, "service_unavailable", err)
		return
	}
	var nonce, verifier, returnTo string
	err = pool.QueryRow(r.Context(), `
		DELETE FROM oauth_states
		WHERE state_hash=$1 AND provider=$2 AND expires_at>now()
		RETURNING nonce,code_verifier,return_to
	`, hashToken(state), providerName).Scan(&nonce, &verifier, &returnTo)
	if err != nil {
		redirectOAuthFailure(w, r, providerName, "invalid_state", err)
		return
	}

	cfg, err := loadOIDCProviderConfig(providerName)
	if err != nil {
		redirectOAuthFailure(w, r, providerName, "provider_unavailable", err)
		return
	}
	oauthCtx, cancel := oauthRequestContext(r.Context())
	defer cancel()
	provider, err := oidc.NewProvider(oauthCtx, cfg.Issuer)
	if err != nil {
		redirectOAuthFailure(w, r, providerName, "provider_unavailable", err)
		return
	}
	oauthConfig := oauthConfigForProvider(cfg, provider)
	token, err := oauthConfig.Exchange(oauthCtx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		redirectOAuthFailure(w, r, providerName, "exchange_failed", err)
		return
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		redirectOAuthFailure(w, r, providerName, "invalid_token", nil)
		return
	}
	idToken, err := provider.Verifier(&oidc.Config{ClientID: cfg.ClientID}).Verify(oauthCtx, rawIDToken)
	if err != nil {
		redirectOAuthFailure(w, r, providerName, "invalid_token", err)
		return
	}
	var claims oidcClaims
	if err := idToken.Claims(&claims); err != nil {
		redirectOAuthFailure(w, r, providerName, "invalid_token", err)
		return
	}
	claims, err = validateOIDCClaims(providerName, claims, nonce)
	if err != nil {
		code := "invalid_token"
		if providerName == "google" && strings.Contains(err.Error(), "verified email") {
			code = "email_unverified"
		}
		redirectOAuthFailure(w, r, providerName, code, err)
		return
	}

	user, err := upsertOIDCIdentity(r.Context(), providerName, claims)
	if err != nil {
		redirectOAuthFailure(w, r, providerName, "account_link_failed", err)
		return
	}
	if err := createBrowserSession(w, r, user); err != nil {
		redirectOAuthFailure(w, r, providerName, "session_failed", err)
		return
	}
	recordOAuthSuccess(r, user.ID, providerName)
	redirectToFrontend(w, r, safeReturnTo(returnTo))
}

func oauthRequestContext(parent context.Context) (context.Context, context.CancelFunc) {
	timeout := envDuration("OAUTH_HTTP_TIMEOUT", 15*time.Second)
	if timeout < 2*time.Second || timeout > 60*time.Second {
		timeout = 15 * time.Second
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	client := &http.Client{Timeout: timeout}
	return oidc.ClientContext(ctx, client), cancel
}

func oauthConfigForProvider(cfg oidcProviderConfig, provider *oidc.Provider) oauth2.Config {
	endpoint := provider.Endpoint()
	if cfg.AuthStyle != oauth2.AuthStyleAutoDetect {
		endpoint.AuthStyle = cfg.AuthStyle
	}
	return oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     endpoint,
		RedirectURL:  cfg.RedirectURL,
		Scopes:       cfg.Scopes,
	}
}

func validateOIDCClaims(providerName string, claims oidcClaims, expectedNonce string) (oidcClaims, error) {
	claims.Subject = strings.TrimSpace(claims.Subject)
	if claims.Subject == "" || len(claims.Subject) > 512 {
		return claims, fmt.Errorf("identity subject is invalid")
	}
	if claims.Nonce == "" || claims.Nonce != expectedNonce {
		return claims, fmt.Errorf("identity token nonce mismatch")
	}

	claims.Name = truncateUTF8(strings.TrimSpace(claims.Name), 120)
	claims.PreferredUsername = truncateUTF8(strings.TrimSpace(claims.PreferredUsername), 120)
	claims.Picture = safeProfilePictureURL(claims.Picture)
	claims.Email = strings.ToLower(strings.TrimSpace(claims.Email))
	if claims.EmailVerified {
		normalized, err := normalizeEmail(claims.Email)
		if err != nil {
			return claims, fmt.Errorf("verified email claim is invalid")
		}
		claims.Email = normalized
	} else {
		claims.Email = ""
	}
	if providerName == "google" && claims.Email == "" {
		return claims, fmt.Errorf("Google account must provide a verified email")
	}
	return claims, nil
}

func safeProfilePictureURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 2048 {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return ""
	}
	return value
}

func truncateUTF8(value string, limit int) string {
	if limit < 1 || utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	return string(runes[:limit])
}

func upsertOIDCIdentity(ctx context.Context, providerName string, claims oidcClaims) (User, error) {
	pool, err := getDB(ctx)
	if err != nil {
		return User{}, err
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback(ctx)

	var user User
	identityFound := true
	err = tx.QueryRow(ctx, `
		SELECT u.id,COALESCE(u.email,''),COALESCE(u.full_name,''),COALESCE(u.avatar_url,''),u.email_verified_at,u.created_at
		FROM auth_identities i JOIN users u ON u.id=i.user_id
		WHERE i.provider=$1 AND i.provider_subject=$2
	`, providerName, claims.Subject).Scan(&user.ID, &user.Email, &user.FullName, &user.AvatarURL, &user.EmailVerifiedAt, &user.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		identityFound = false
	} else if err != nil {
		return User{}, err
	}

	verifiedEmail := ""
	if claims.EmailVerified {
		verifiedEmail = claims.Email
	}
	createdUser := false
	if !identityFound {
		if verifiedEmail != "" {
			err = tx.QueryRow(ctx, `
				SELECT id,COALESCE(email,''),COALESCE(full_name,''),COALESCE(avatar_url,''),email_verified_at,created_at
				FROM users WHERE lower(email)=lower($1)
			`, verifiedEmail).Scan(&user.ID, &user.Email, &user.FullName, &user.AvatarURL, &user.EmailVerifiedAt, &user.CreatedAt)
		}
		if verifiedEmail == "" || errors.Is(err, pgx.ErrNoRows) {
			now := time.Now().UTC()
			user = User{
				ID:        newDatabaseID("usr"),
				Email:     verifiedEmail,
				FullName:  oidcDisplayName(claims),
				AvatarURL: claims.Picture,
				CreatedAt: now,
			}
			if verifiedEmail != "" {
				user.EmailVerifiedAt = &now
			}
			tag, insertErr := tx.Exec(ctx, `
				INSERT INTO users (id,email,full_name,avatar_url,email_verified_at,last_login_at,created_at,updated_at)
				VALUES ($1,$2,$3,$4,$5,now(),$6,$6)
				ON CONFLICT DO NOTHING
			`, user.ID, nullableString(user.Email), user.FullName, nullableString(user.AvatarURL), user.EmailVerifiedAt, now)
			if insertErr != nil {
				return User{}, insertErr
			}
			createdUser = tag.RowsAffected() == 1
			if !createdUser {
				if verifiedEmail == "" {
					return User{}, fmt.Errorf("cannot create external account user")
				}
				err = tx.QueryRow(ctx, `
					SELECT id,COALESCE(email,''),COALESCE(full_name,''),COALESCE(avatar_url,''),email_verified_at,created_at
					FROM users WHERE lower(email)=lower($1)
				`, verifiedEmail).Scan(&user.ID, &user.Email, &user.FullName, &user.AvatarURL, &user.EmailVerifiedAt, &user.CreatedAt)
				if err != nil {
					return User{}, err
				}
			}
		} else if err != nil {
			return User{}, err
		}

		tag, err := tx.Exec(ctx, `
			INSERT INTO auth_identities (id,user_id,provider,provider_subject,provider_email,provider_username)
			VALUES ($1,$2,$3,$4,$5,$6)
			ON CONFLICT (provider,provider_subject) DO NOTHING
		`, newDatabaseID("idn"), user.ID, providerName, claims.Subject, nullableString(verifiedEmail), nullableString(claims.PreferredUsername))
		if err != nil {
			return User{}, err
		}
		if tag.RowsAffected() == 0 {
			var linked User
			err = tx.QueryRow(ctx, `
				SELECT u.id,COALESCE(u.email,''),COALESCE(u.full_name,''),COALESCE(u.avatar_url,''),u.email_verified_at,u.created_at
				FROM auth_identities i JOIN users u ON u.id=i.user_id
				WHERE i.provider=$1 AND i.provider_subject=$2
			`, providerName, claims.Subject).Scan(&linked.ID, &linked.Email, &linked.FullName, &linked.AvatarURL, &linked.EmailVerifiedAt, &linked.CreatedAt)
			if err != nil {
				return User{}, err
			}
			if createdUser && linked.ID != user.ID {
				_, _ = tx.Exec(ctx, `DELETE FROM users WHERE id=$1`, user.ID)
			}
			user = linked
		}
	}

	name := oidcDisplayName(claims)
	_, err = tx.Exec(ctx, `
		UPDATE auth_identities SET
			provider_email=COALESCE($3,provider_email),
			provider_username=COALESCE($4,provider_username),
			updated_at=now()
		WHERE provider=$1 AND provider_subject=$2
	`, providerName, claims.Subject, nullableString(verifiedEmail), nullableString(claims.PreferredUsername))
	if err != nil {
		return User{}, err
	}
	_, err = tx.Exec(ctx, `
		UPDATE users SET
			full_name=CASE WHEN $2<>'' THEN $2 ELSE full_name END,
			avatar_url=CASE WHEN $3<>'' THEN $3 ELSE avatar_url END,
			email=COALESCE(email,$4),
			email_verified_at=CASE WHEN $4 IS NOT NULL THEN COALESCE(email_verified_at,now()) ELSE email_verified_at END,
			last_login_at=now(),updated_at=now()
		WHERE id=$1
	`, user.ID, name, claims.Picture, nullableString(verifiedEmail))
	if err != nil {
		return User{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return User{}, err
	}

	user.FullName = firstNonEmpty(name, user.FullName)
	user.AvatarURL = firstNonEmpty(claims.Picture, user.AvatarURL)
	if user.Email == "" {
		user.Email = verifiedEmail
	}
	return user, nil
}

func loadOIDCProviderConfig(providerName string) (oidcProviderConfig, error) {
	if !authProviderEnabled(providerName) {
		return oidcProviderConfig{}, fmt.Errorf("%s authentication is disabled", providerName)
	}
	apiBase := strings.TrimRight(strings.TrimSpace(os.Getenv("API_PUBLIC_BASE_URL")), "/")
	switch providerName {
	case "google":
		cfg := oidcProviderConfig{
			Name:         "google",
			Issuer:       googleOIDCIssuer,
			ClientID:     strings.TrimSpace(os.Getenv("GOOGLE_OIDC_CLIENT_ID")),
			ClientSecret: strings.TrimSpace(os.Getenv("GOOGLE_OIDC_CLIENT_SECRET")),
			RedirectURL:  strings.TrimSpace(os.Getenv("GOOGLE_OIDC_REDIRECT_URL")),
			Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
			AuthStyle:    oauth2.AuthStyleAutoDetect,
		}
		if cfg.RedirectURL == "" && apiBase != "" {
			cfg.RedirectURL = apiBase + "/v1/auth/google/callback"
		}
		if cfg.ClientID == "" || cfg.ClientSecret == "" || cfg.RedirectURL == "" {
			return cfg, fmt.Errorf("Google OIDC is not configured")
		}
		if err := validateOIDCRedirectURL(cfg.RedirectURL, "google"); err != nil {
			return cfg, err
		}
		return cfg, nil
	case "telegram":
		issuer := strings.TrimRight(strings.TrimSpace(os.Getenv("TELEGRAM_OIDC_ISSUER")), "/")
		if issuer == "" {
			issuer = telegramOIDCIssuer
		}
		if issuer != telegramOIDCIssuer {
			return oidcProviderConfig{}, fmt.Errorf("TELEGRAM_OIDC_ISSUER must be %s", telegramOIDCIssuer)
		}
		cfg := oidcProviderConfig{
			Name:         "telegram",
			Issuer:       telegramOIDCIssuer,
			ClientID:     strings.TrimSpace(os.Getenv("TELEGRAM_OIDC_CLIENT_ID")),
			ClientSecret: strings.TrimSpace(os.Getenv("TELEGRAM_OIDC_CLIENT_SECRET")),
			RedirectURL:  strings.TrimSpace(os.Getenv("TELEGRAM_OIDC_REDIRECT_URL")),
			Scopes:       []string{oidc.ScopeOpenID, "profile"},
			AuthStyle:    oauth2.AuthStyleInHeader,
		}
		if cfg.RedirectURL == "" && apiBase != "" {
			cfg.RedirectURL = apiBase + "/v1/auth/telegram/callback"
		}
		if cfg.ClientID == "" || cfg.ClientSecret == "" || cfg.RedirectURL == "" {
			return cfg, fmt.Errorf("Telegram OIDC is not configured")
		}
		if err := validateOIDCRedirectURL(cfg.RedirectURL, "telegram"); err != nil {
			return cfg, err
		}
		return cfg, nil
	default:
		return oidcProviderConfig{}, fmt.Errorf("unsupported identity provider")
	}
}

func validateOIDCRedirectURL(value, providerName string) error {
	callbackPath := "/v1/auth/" + providerName + "/callback"
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("%s OIDC redirect URL is invalid", providerName)
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path != callbackPath {
		return fmt.Errorf("%s OIDC redirect URL must end with %s and contain no query or fragment", providerName, callbackPath)
	}
	if os.Getenv("APP_ENV") == "production" {
		if parsed.Scheme != "https" {
			return fmt.Errorf("%s OIDC redirect URL must use https in production", providerName)
		}
		apiBase := strings.TrimRight(strings.TrimSpace(os.Getenv("API_PUBLIC_BASE_URL")), "/")
		if apiBase == "" || value != apiBase+callbackPath {
			return fmt.Errorf("%s OIDC redirect URL must exactly match API_PUBLIC_BASE_URL%s", providerName, callbackPath)
		}
	}
	return nil
}

func safeReturnTo(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 512 || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") {
		return "/dashboard"
	}
	for _, character := range value {
		if character == '\\' || unicode.IsControl(character) {
			return "/dashboard"
		}
	}
	decoded, err := url.PathUnescape(value)
	if err != nil || strings.HasPrefix(decoded, "//") || strings.Contains(decoded, "\\") {
		return "/dashboard"
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.Fragment != "" {
		return "/dashboard"
	}
	return value
}

func redirectOAuthFailure(w http.ResponseWriter, r *http.Request, providerName, code string, cause error) {
	if cause != nil {
		log.Printf("oauth authentication failed provider=%s code=%s err=%v", providerName, code, cause)
	} else {
		log.Printf("oauth authentication failed provider=%s code=%s", providerName, code)
	}
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL")), "/")
	if base == "" {
		estimateWriteJSON(w, http.StatusBadRequest, map[string]string{
			"error":    "oauth_authentication_failed",
			"code":     code,
			"provider": providerName,
		})
		return
	}
	query := url.Values{}
	query.Set("oauth_error", code)
	if providerName != "" {
		query.Set("provider", providerName)
	}
	http.Redirect(w, r, base+"/login?"+query.Encode(), http.StatusSeeOther)
}

func redirectOAuthError(w http.ResponseWriter, r *http.Request, _ string) {
	redirectOAuthFailure(w, r, "", "invalid_state", nil)
}

func redirectToFrontend(w http.ResponseWriter, r *http.Request, path string) {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL")), "/")
	if base == "" {
		estimateWriteJSON(w, http.StatusOK, map[string]bool{"authenticated": true})
		return
	}
	http.Redirect(w, r, base+path, http.StatusSeeOther)
}

func recordOAuthSuccess(r *http.Request, userID, providerName string) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	pool, err := getDB(ctx)
	if err != nil || pool == nil {
		return
	}
	ipAddress := strings.TrimSpace(r.RemoteAddr)
	if host, _, splitErr := net.SplitHostPort(ipAddress); splitErr == nil {
		ipAddress = host
	}
	userAgent := truncateUTF8(strings.TrimSpace(r.UserAgent()), 512)
	_, _ = pool.Exec(ctx, `
		INSERT INTO audit_logs (id,user_id,action,resource_type,resource_id,ip_address,user_agent)
		VALUES ($1,$2,'auth.oauth.login','identity_provider',$3,$4,$5)
	`, newDatabaseID("aud"), userID, providerName, nullableString(ipAddress), nullableString(userAgent))
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.TrimSpace(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func oidcDisplayName(claims oidcClaims) string {
	return firstNonEmpty(claims.Name, claims.PreferredUsername, "Пользователь SmetaCheck")
}
