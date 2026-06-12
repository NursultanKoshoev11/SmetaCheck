package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

func endpointRateLimit(action string, limit int, window time.Duration, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if limit < 1 || window <= 0 {
			handler(w, r)
			return
		}
		pool, err := getDB(r.Context())
		if err != nil || pool == nil {
			estimateWriteError(w, http.StatusServiceUnavailable, "rate limiter is unavailable")
			return
		}
		identity := strings.TrimSpace(requestIP(r))
		if user, ok := currentRequestUser(r); ok && user.ID != "" {
			identity = "user:" + user.ID
		}
		if identity == "" {
			identity = "unknown"
		}
		keyHash := hashToken("endpoint-rate|" + action + "|" + identity)
		now := time.Now().UTC()
		seconds := int64(window / time.Second)
		if seconds < 1 { seconds = 1 }
		windowStart := time.Unix((now.Unix()/seconds)*seconds, 0).UTC()
		var count int
		err = pool.QueryRow(r.Context(), `
			INSERT INTO auth_rate_limits (key_hash,action,window_started_at,request_count,updated_at)
			VALUES ($1,$2,$3,1,now())
			ON CONFLICT (key_hash,action,window_started_at)
			DO UPDATE SET request_count=auth_rate_limits.request_count+1,updated_at=now()
			RETURNING request_count
		`, keyHash, "endpoint:"+action, windowStart).Scan(&count)
		if err != nil {
			estimateWriteError(w, http.StatusServiceUnavailable, "rate limiter is unavailable")
			return
		}
		remaining := limit - count
		if remaining < 0 { remaining = 0 }
		resetAt := windowStart.Add(window)
		w.Header().Set("RateLimit-Limit", strconv.Itoa(limit))
		w.Header().Set("RateLimit-Remaining", strconv.Itoa(remaining))
		w.Header().Set("RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))
		if count > limit {
			retryAfter := int(time.Until(resetAt).Seconds())
			if retryAfter < 1 { retryAfter = 1 }
			w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
			estimateWriteError(w, http.StatusTooManyRequests, "request limit exceeded; try again later")
			return
		}
		handler(w, r)
	}
}
