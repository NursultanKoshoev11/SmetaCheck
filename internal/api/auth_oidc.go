package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/jackc/pgx/v5"
	"golang.org/x/oauth2"
)

type oidcProviderConfig struct {
	Name         string
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
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
		redirectOAuthError(w, r, err.Error())
		return
	}
	provider, err := oidc.NewProvider(r.Context(), cfg.Issuer)
	if err != nil {
		redirectOAuthError(w, r, "identity provider is unavailable")
		return
	}
	state, err := randomURLToken(32)
	if err != nil {
		redirectOAuthError(w, r, "cannot create OAuth state")
		return
	}
	nonce, err := randomURLToken(32)
	if err != nil {
		redirectOAuthError(w, r, "cannot create OAuth nonce")
		return
	}
	verifier := oauth2.GenerateVerifier()
	returnTo := safeReturnTo(r.URL.Query().Get("return_to"))
	pool, err := getDB(r.Context())
	if err != nil || pool == nil {
		redirectOAuthError(w, r, "postgresql is unavailable")
		return
	}
	_, err = pool.Exec(r.Context(), `
		INSERT INTO oauth_states (state_hash,provider,nonce,code_verifier,return_to,expires_at)
		VALUES ($1,$2,$3,$4,$5,$6)
	`, hashToken(state), providerName, nonce, verifier, returnTo, time.Now().UTC().Add(10*time.Minute))
	if err != nil {
		redirectOAuthError(w, r, "cannot save OAuth state")
		return
	}

	oauthConfig := oauth2.Config{
		ClientID: cfg.ClientID, ClientSecret: cfg.ClientSecret,
		Endpoint: provider.Endpoint(), RedirectURL: cfg.RedirectURL, Scopes: cfg.Scopes,
	}
	authURL := oauthConfig.AuthCodeURL(state,
		oauth2.AccessTypeOnline,
		oauth2.SetAuthURLParam("nonce", nonce),
		oauth2.S256ChallengeOption(verifier),
	)
	http.Redirect(w, r, authURL, http.StatusFound)
}

func finishOIDC(w http.ResponseWriter, r *http.Request, providerName string) {
	if providerError := strings.TrimSpace(r.URL.Query().Get("error")); providerError != "" {
		redirectOAuthError(w, r, "sign-in was cancelled or denied")
		return
	}
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if state == "" || code == "" {
		redirectOAuthError(w, r, "OAuth callback is incomplete")
		return
	}

	pool, err := getDB(r.Context())
	if err != nil || pool == nil {
		redirectOAuthError(w, r, "postgresql is unavailable")
		return
	}
	var nonce, verifier, returnTo string
	err = pool.QueryRow(r.Context(), `
		DELETE FROM oauth_states
		WHERE state_hash=$1 AND provider=$2 AND expires_at>now()
		RETURNING nonce,code_verifier,return_to
	`, hashToken(state), providerName).Scan(&nonce, &verifier, &returnTo)
	if err != nil {
		redirectOAuthError(w, r, "OAuth state is invalid or expired")
		return
	}

	cfg, err := loadOIDCProviderConfig(providerName)
	if err != nil {
		redirectOAuthError(w, r, err.Error())
		return
	}
	provider, err := oidc.NewProvider(r.Context(), cfg.Issuer)
	if err != nil {
		redirectOAuthError(w, r, "identity provider is unavailable")
		return
	}
	oauthConfig := oauth2.Config{
		ClientID: cfg.ClientID, ClientSecret: cfg.ClientSecret,
		Endpoint: provider.Endpoint(), RedirectURL: cfg.RedirectURL, Scopes: cfg.Scopes,
	}
	token, err := oauthConfig.Exchange(r.Context(), code, oauth2.VerifierOption(verifier))
	if err != nil {
		redirectOAuthError(w, r, "authorization code exchange failed")
		return
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		redirectOAuthError(w, r, "identity token is missing")
		return
	}
	idToken, err := provider.Verifier(&oidc.Config{ClientID: cfg.ClientID}).Verify(r.Context(), rawIDToken)
	if err != nil {
		redirectOAuthError(w, r, "identity token verification failed")
		return
	}
	var claims oidcClaims
	if err := idToken.Claims(&claims); err != nil {
		redirectOAuthError(w, r, "identity claims are invalid")
		return
	}
	if claims.Subject == "" || claims.Nonce != nonce {
		redirectOAuthError(w, r, "identity token nonce mismatch")
		return
	}
	if providerName == "google" && (!claims.EmailVerified || strings.TrimSpace(claims.Email) == "") {
		redirectOAuthError(w, r, "Google account email is not verified")
		return
	}

	user, err := upsertOIDCIdentity(r.Context(), providerName, claims)
	if err != nil {
		redirectOAuthError(w, r, "cannot link external account")
		return
	}
	if err := createBrowserSession(w, r, user); err != nil {
		redirectOAuthError(w, r, "cannot create browser session")
		return
	}
	redirectToFrontend(w, r, safeReturnTo(returnTo))
}

func upsertOIDCIdentity(ctx context.Context, providerName string, claims oidcClaims) (User, error) {
	pool, err := getDB(ctx)
	if err != nil { return User{}, err }
	tx, err := pool.Begin(ctx)
	if err != nil { return User{}, err }
	defer tx.Rollback(ctx)

	var user User
	err = tx.QueryRow(ctx, `
		SELECT u.id,COALESCE(u.email,''),COALESCE(u.full_name,''),COALESCE(u.avatar_url,''),u.email_verified_at,u.created_at
		FROM auth_identities i JOIN users u ON u.id=i.user_id
		WHERE i.provider=$1 AND i.provider_subject=$2
	`, providerName, claims.Subject).Scan(&user.ID,&user.Email,&user.FullName,&user.AvatarURL,&user.EmailVerifiedAt,&user.CreatedAt)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) { return User{}, err }

	verifiedEmail := ""
	if claims.EmailVerified { verifiedEmail = strings.ToLower(strings.TrimSpace(claims.Email)) }
	if errors.Is(err, pgx.ErrNoRows) {
		if verifiedEmail != "" {
			err = tx.QueryRow(ctx, `SELECT id,COALESCE(email,''),COALESCE(full_name,''),COALESCE(avatar_url,''),email_verified_at,created_at FROM users WHERE lower(email)=lower($1)`, verifiedEmail).
				Scan(&user.ID,&user.Email,&user.FullName,&user.AvatarURL,&user.EmailVerifiedAt,&user.CreatedAt)
		}
		if verifiedEmail == "" || errors.Is(err, pgx.ErrNoRows) {
			now := time.Now().UTC()
			user = User{ID:newDatabaseID("usr"),Email:verifiedEmail,FullName:oidcDisplayName(claims),AvatarURL:claims.Picture,CreatedAt:now}
			if verifiedEmail != "" { user.EmailVerifiedAt = &now }
			_, err = tx.Exec(ctx, `
				INSERT INTO users (id,email,full_name,avatar_url,email_verified_at,last_login_at,created_at,updated_at)
				VALUES ($1,$2,$3,$4,$5,now(),$6,$6)
			`, user.ID, nullableString(user.Email), user.FullName, nullableString(user.AvatarURL), user.EmailVerifiedAt, now)
			if err != nil { return User{}, err }
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO auth_identities (id,user_id,provider,provider_subject,provider_email,provider_username)
			VALUES ($1,$2,$3,$4,$5,$6)
		`, newDatabaseID("idn"), user.ID, providerName, claims.Subject, nullableString(verifiedEmail), nullableString(claims.PreferredUsername))
		if err != nil { return User{}, err }
	}

	name := oidcDisplayName(claims)
	_, err = tx.Exec(ctx, `
		UPDATE users SET
			full_name=CASE WHEN $2<>'' THEN $2 ELSE full_name END,
			avatar_url=CASE WHEN $3<>'' THEN $3 ELSE avatar_url END,
			email=COALESCE(email,$4),
			email_verified_at=CASE WHEN $4 IS NOT NULL THEN COALESCE(email_verified_at,now()) ELSE email_verified_at END,
			last_login_at=now(),updated_at=now()
		WHERE id=$1
	`, user.ID, name, claims.Picture, nullableString(verifiedEmail))
	if err != nil { return User{}, err }
	if err := tx.Commit(ctx); err != nil { return User{}, err }

	user.FullName = firstNonEmpty(name, user.FullName)
	user.AvatarURL = firstNonEmpty(claims.Picture, user.AvatarURL)
	if user.Email == "" { user.Email = verifiedEmail }
	return user, nil
}

func loadOIDCProviderConfig(providerName string) (oidcProviderConfig, error) {
	apiBase := strings.TrimRight(strings.TrimSpace(os.Getenv("API_PUBLIC_BASE_URL")), "/")
	switch providerName {
	case "google":
		cfg := oidcProviderConfig{
			Name:"google", Issuer:"https://accounts.google.com",
			ClientID:strings.TrimSpace(os.Getenv("GOOGLE_OIDC_CLIENT_ID")),
			ClientSecret:os.Getenv("GOOGLE_OIDC_CLIENT_SECRET"),
			RedirectURL:strings.TrimSpace(os.Getenv("GOOGLE_OIDC_REDIRECT_URL")),
			Scopes:[]string{oidc.ScopeOpenID,"email","profile"},
		}
		if cfg.RedirectURL=="" && apiBase!="" { cfg.RedirectURL=apiBase+"/v1/auth/google/callback" }
		if cfg.ClientID=="" || cfg.ClientSecret=="" || cfg.RedirectURL=="" { return cfg, fmt.Errorf("Google OIDC is not configured") }
		return cfg,nil
	case "telegram":
		cfg := oidcProviderConfig{
			Name:"telegram", Issuer:strings.TrimRight(strings.TrimSpace(os.Getenv("TELEGRAM_OIDC_ISSUER")),"/"),
			ClientID:strings.TrimSpace(os.Getenv("TELEGRAM_OIDC_CLIENT_ID")),
			ClientSecret:os.Getenv("TELEGRAM_OIDC_CLIENT_SECRET"),
			RedirectURL:strings.TrimSpace(os.Getenv("TELEGRAM_OIDC_REDIRECT_URL")),
			Scopes:[]string{oidc.ScopeOpenID,"profile"},
		}
		if cfg.RedirectURL=="" && apiBase!="" { cfg.RedirectURL=apiBase+"/v1/auth/telegram/callback" }
		if cfg.Issuer=="" || cfg.ClientID=="" || cfg.ClientSecret=="" || cfg.RedirectURL=="" { return cfg, fmt.Errorf("Telegram OIDC is not configured") }
		return cfg,nil
	default:
		return oidcProviderConfig{},fmt.Errorf("unsupported identity provider")
	}
}

func safeReturnTo(value string) string {
	value = strings.TrimSpace(value)
	if value=="" || !strings.HasPrefix(value,"/") || strings.HasPrefix(value,"//") { return "/dashboard" }
	return value
}

func redirectOAuthError(w http.ResponseWriter, r *http.Request, message string) {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL")), "/")
	if base=="" { estimateWriteError(w,http.StatusBadRequest,message); return }
	http.Redirect(w,r,base+"/login?oauth_error="+urlQueryEscape(message),http.StatusSeeOther)
}

func redirectToFrontend(w http.ResponseWriter, r *http.Request, path string) {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL")), "/")
	if base=="" { estimateWriteJSON(w,http.StatusOK,map[string]bool{"authenticated":true}); return }
	http.Redirect(w,r,base+path,http.StatusSeeOther)
}

func nullableString(value string) any { if strings.TrimSpace(value)=="" { return nil }; return strings.TrimSpace(value) }
func firstNonEmpty(values ...string) string { for _,value:=range values { if strings.TrimSpace(value)!="" { return strings.TrimSpace(value) } }; return "" }
func oidcDisplayName(claims oidcClaims) string { return firstNonEmpty(claims.Name,claims.PreferredUsername,"Пользователь SmetaCheck") }
func urlQueryEscape(value string) string { replacer:=strings.NewReplacer("%","%25"," ","%20","?","%3F","&","%26","=","%3D","#","%23"); return replacer.Replace(value) }
