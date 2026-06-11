package api

import (
	"crypto/subtle"
	"net/http"
	"strings"
	"time"
)

func setOAuthStateCookie(w http.ResponseWriter, provider, state string) {
	secure := cookieSecure()
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName(provider, secure),
		Value:    hashToken(state),
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600,
		Expires:  time.Now().UTC().Add(10 * time.Minute),
	})
}

func validateAndClearOAuthStateCookie(w http.ResponseWriter, r *http.Request, provider, state string) bool {
	secure := cookieSecure()
	name := oauthStateCookieName(provider, secure)
	cookie, err := r.Cookie(name)
	clearOAuthStateCookie(w, provider, secure)
	if err != nil || strings.TrimSpace(cookie.Value) == "" || strings.TrimSpace(state) == "" {
		return false
	}
	expected := hashToken(state)
	return subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(expected)) == 1
}

func clearOAuthStateCookie(w http.ResponseWriter, provider string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName(provider, secure),
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
	})
}

func oauthStateCookieName(provider string, secure bool) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if secure {
		return "__Host-smetacheck_oauth_" + provider
	}
	return "smetacheck_oauth_" + provider
}
