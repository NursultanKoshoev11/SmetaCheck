package api

import (
	"net/http"
	"strings"
	"time"
)

func authRateLimit(action string, limit int, window time.Duration, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if limit < 1 || window <= 0 {
			handler(w, r)
			return
		}
		pool, err := getDB(r.Context())
		if err != nil || pool == nil {
			estimateWriteError(w, http.StatusServiceUnavailable, "authentication service is unavailable")
			return
		}

		key := strings.TrimSpace(requestIP(r))
		if key == "" {
			key = "unknown"
		}
		keyHash := hashToken("auth-rate|" + action + "|" + key)
		now := time.Now().UTC()
		windowSeconds := int64(window / time.Second)
		if windowSeconds < 1 {
			windowSeconds = 1
		}
		windowStart := time.Unix((now.Unix()/windowSeconds)*windowSeconds, 0).UTC()

		var count int
		err = pool.QueryRow(r.Context(), `
			INSERT INTO auth_rate_limits (key_hash,action,window_started_at,request_count,updated_at)
			VALUES ($1,$2,$3,1,now())
			ON CONFLICT (key_hash,action,window_started_at)
			DO UPDATE SET request_count=auth_rate_limits.request_count+1,updated_at=now()
			RETURNING request_count
		`, keyHash, action, windowStart).Scan(&count)
		if err != nil {
			estimateWriteError(w, http.StatusServiceUnavailable, "authentication rate limiter is unavailable")
			return
		}

		remaining := limit - count
		if remaining < 0 {
			remaining = 0
		}
		resetAt := windowStart.Add(window)
		w.Header().Set("RateLimit-Limit", itoa(limit))
		w.Header().Set("RateLimit-Remaining", itoa(remaining))
		w.Header().Set("RateLimit-Reset", itoa64(resetAt.Unix()))

		if count > limit {
			retryAfter := int(time.Until(resetAt).Seconds())
			if retryAfter < 1 {
				retryAfter = 1
			}
			w.Header().Set("Retry-After", itoa(retryAfter))
			estimateWriteError(w, http.StatusTooManyRequests, "too many authentication requests; try again later")
			return
		}
		handler(w, r)
	}
}

func itoa(value int) string {
	return itoa64(int64(value))
}

func itoa64(value int64) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	var buffer [32]byte
	index := len(buffer)
	for value > 0 {
		index--
		buffer[index] = byte('0' + value%10)
		value /= 10
	}
	if negative {
		index--
		buffer[index] = '-'
	}
	return string(buffer[index:])
}
