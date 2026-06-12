package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidRequestID(t *testing.T) {
	valid := []string{"request-123", "trace_id.456", strings.Repeat("a", 128)}
	for _, value := range valid {
		if !validRequestID(value) {
			t.Fatalf("expected request ID %q to be valid", value)
		}
	}

	invalid := []string{"", "contains space", "line\nbreak", "slash/value", strings.Repeat("a", 129)}
	for _, value := range invalid {
		if validRequestID(value) {
			t.Fatalf("expected request ID %q to be invalid", value)
		}
	}
}

func TestRequestIDMiddlewareReplacesInvalidValue(t *testing.T) {
	handler := requestID(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("X-Request-ID", "invalid request id")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	generated := response.Header().Get("X-Request-ID")
	if !validRequestID(generated) || generated == "invalid request id" {
		t.Fatalf("middleware did not replace invalid request ID: %q", generated)
	}
}

func TestDefaultRequestBodyLimit(t *testing.T) {
	handler := maxBodyBytes(readAllTestHandler())
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/login", strings.NewReader(strings.Repeat("x", 1024*1024+1)))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected HTTP 413, got %d", response.Code)
	}
}

func TestBatchTotalRequestBodyLimit(t *testing.T) {
	t.Setenv("MAX_UPLOAD_MB", "25")
	t.Setenv("MAX_BATCH_FILES", "10")
	t.Setenv("MAX_BATCH_TOTAL_MB", "2")
	handler := maxBodyBytes(readAllTestHandler())
	request := httptest.NewRequest(http.MethodPost, "/v1/analysis-batches", strings.NewReader(strings.Repeat("x", 2*1024*1024+1)))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected HTTP 413, got %d", response.Code)
	}
}

func readAllTestHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.ReadAll(r.Body); err != nil {
			http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
