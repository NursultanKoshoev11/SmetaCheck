package api

import (
	"net/http"
	"os"
	"strings"
)

func csrfProtection(next http.Handler) http.Handler {
	allowed := parseAllowedOrigins(os.Getenv("ALLOWED_ORIGINS"))
	if os.Getenv("APP_ENV") != "production" {
		allowed["http://localhost:3000"] = true
		allowed["http://127.0.0.1:3000"] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(r.Header.Get("Authorization"))), "bearer ") {
			next.ServeHTTP(w, r)
			return
		}

		_, hasAccessCookie := readAccessCookie(r)
		_, hasRefreshCookie := readRefreshCookie(r)
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		referer := strings.TrimSpace(r.Header.Get("Referer"))
		if origin == "" && referer == "" && !hasAccessCookie && !hasRefreshCookie {
			next.ServeHTTP(w, r)
			return
		}
		if origin == "" && referer != "" {
			for candidate := range allowed {
				if referer == candidate || strings.HasPrefix(referer, candidate+"/") {
					origin = candidate
					break
				}
			}
		}
		if origin == "" || !allowed[origin] {
			estimateWriteError(w, http.StatusForbidden, "request origin is not allowed")
			return
		}
		next.ServeHTTP(w, r)
	})
}
