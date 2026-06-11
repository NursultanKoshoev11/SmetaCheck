package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const maxAIResponseBytes = 4 * 1024 * 1024

var aiHTTPTransport = &http.Transport{
	Proxy:                 http.ProxyFromEnvironment,
	DialContext:           (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
	ForceAttemptHTTP2:     true,
	MaxIdleConns:          100,
	MaxIdleConnsPerHost:   20,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: time.Second,
	ResponseHeaderTimeout: 40 * time.Second,
}

func doAIJSONRequest(ctx context.Context, method, endpoint string, headers map[string]string, payload []byte) ([]byte, http.Header, error) {
	attempts := int(envInt64("AI_HTTP_MAX_ATTEMPTS", 3))
	if attempts < 1 {
		attempts = 1
	}
	if attempts > 5 {
		attempts = 5
	}
	timeout := envDuration("AI_TIMEOUT", 45*time.Second)
	client := &http.Client{Timeout: timeout, Transport: aiHTTPTransport}
	var lastErr error

	for attempt := 1; attempt <= attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(payload))
		if err != nil {
			return nil, nil, fmt.Errorf("create AI request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("X-Client-Request-Id", newRequestID())
		for key, value := range headers {
			req.Header.Set(key, value)
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("AI request failed: %w", err)
			if attempt < attempts && ctx.Err() == nil {
				if err := waitForRetry(ctx, retryDelay(attempt, "")); err != nil {
					return nil, nil, err
				}
				continue
			}
			return nil, nil, lastErr
		}

		body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxAIResponseBytes+1))
		resp.Body.Close()
		if readErr != nil {
			return nil, resp.Header, fmt.Errorf("read AI response: %w", readErr)
		}
		if len(body) > maxAIResponseBytes {
			return nil, resp.Header, fmt.Errorf("AI response exceeds %d bytes", maxAIResponseBytes)
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return body, resp.Header, nil
		}

		lastErr = fmt.Errorf("AI provider returned HTTP %d: %s", resp.StatusCode, compactErrorBody(body))
		if attempt >= attempts || !retryableAIStatus(resp.StatusCode) {
			return nil, resp.Header, lastErr
		}
		if err := waitForRetry(ctx, retryDelay(attempt, resp.Header.Get("Retry-After"))); err != nil {
			return nil, resp.Header, err
		}
	}
	return nil, nil, lastErr
}

func waitForRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func retryableAIStatus(status int) bool {
	return status == http.StatusTooManyRequests || status == http.StatusRequestTimeout || status >= 500
}

func retryDelay(attempt int, retryAfter string) time.Duration {
	if seconds, err := strconv.Atoi(strings.TrimSpace(retryAfter)); err == nil && seconds > 0 && seconds <= 30 {
		return time.Duration(seconds) * time.Second
	}
	delay := time.Duration(attempt*attempt) * 500 * time.Millisecond
	if delay > 5*time.Second {
		return 5 * time.Second
	}
	return delay
}
