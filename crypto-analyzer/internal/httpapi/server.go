package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/NursultanKoshoev11/SmetaCheck/crypto-analyzer/internal/store"
)

type Server struct {
	httpServer *http.Server
	store      *store.Store
	apiKey     string
	logger     *slog.Logger
}

func New(addr, apiKey string, database *store.Store, logger *slog.Logger) *Server {
	s := &Server{store: database, apiKey: strings.TrimSpace(apiKey), logger: logger}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /api/v1/signals/latest", s.auth(s.latestSignals))
	mux.HandleFunc("GET /api/v1/scans/latest", s.auth(s.latestScan))
	mux.HandleFunc("GET /api/v1/coins/{symbol}", s.auth(s.coinHistory))
	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           securityHeaders(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return s
}

func (s *Server) Start() error {
	s.logger.Info("HTTP API listening", "addr", s.httpServer.Addr)
	err := s.httpServer.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) error { return s.httpServer.Shutdown(ctx) }

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	if err := s.store.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"status": "unhealthy", "database": "down"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "database": "postgresql", "time": time.Now().UTC()})
}

func (s *Server) latestSignals(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r, 20)
	signals, err := s.store.LatestSignals(r.Context(), limit)
	if err != nil {
		s.logger.Error("latest signals query failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database query failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"signals": signals, "count": len(signals)})
}

func (s *Server) latestScan(w http.ResponseWriter, r *http.Request) {
	run, err := s.store.LatestScan(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database query failed"})
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (s *Server) coinHistory(w http.ResponseWriter, r *http.Request) {
	symbol := strings.ToUpper(strings.TrimSpace(r.PathValue("symbol")))
	if symbol == "" || len(symbol) > 30 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid symbol"})
		return
	}
	signals, err := s.store.LatestSignalsForSymbol(r.Context(), symbol, parseLimit(r, 30))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database query failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"symbol": symbol, "signals": signals, "count": len(signals)})
}

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.apiKey == "" {
			next(w, r)
			return
		}
		provided := strings.TrimSpace(r.Header.Get("X-API-Key"))
		if provided == "" {
			provided = strings.TrimPrefix(strings.TrimSpace(r.Header.Get("Authorization")), "Bearer ")
		}
		if subtleEqual(provided, s.apiKey) {
			next(w, r)
			return
		}
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
}

func parseLimit(r *http.Request, fallback int) int {
	value, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || value <= 0 || value > 200 {
		return fallback
	}
	return value
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

func subtleEqual(a, b string) bool {
	if len(a) != len(b) || len(a) == 0 {
		return false
	}
	var result byte
	for i := range a {
		result |= a[i] ^ b[i]
	}
	return result == 0
}
