package api

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func Run() {
	if err := validateProductionConfig(); err != nil {
		log.Fatalf("configuration error: %v", err)
	}
	initDatabaseForRun()

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	startAuthCleanup(rootCtx)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", Health)
	mux.HandleFunc("/ready", Ready)

	mux.HandleFunc("/v1/auth/providers", requireMethod(http.MethodGet, AuthProviders))
	mux.HandleFunc("/v1/auth/register", requireMethod(http.MethodPost, authRateLimit("register", 5, time.Hour, AuthRegisterEmail)))
	mux.HandleFunc("/v1/auth/login", requireMethod(http.MethodPost, authRateLimit("login", 10, 15*time.Minute, AuthLoginEmail)))
	mux.HandleFunc("/v1/auth/me", requireMethod(http.MethodGet, AuthMe))
	mux.HandleFunc("/v1/auth/refresh", requireMethod(http.MethodPost, authRateLimit("refresh", 60, 15*time.Minute, AuthRefresh)))
	mux.HandleFunc("/v1/auth/logout", requireMethod(http.MethodPost, AuthLogoutSecure))
	mux.HandleFunc("/v1/auth/email/verify", requireMethod(http.MethodGet, AuthVerifyEmail))
	mux.HandleFunc("/v1/auth/email/resend", requireMethod(http.MethodPost, authRateLimit("resend_verification", 5, time.Hour, AuthResendVerification)))
	mux.HandleFunc("/v1/auth/password/forgot", requireMethod(http.MethodPost, authRateLimit("forgot_password", 5, time.Hour, AuthForgotPassword)))
	mux.HandleFunc("/v1/auth/password/reset", requireMethod(http.MethodPost, authRateLimit("reset_password", 10, time.Hour, AuthResetPassword)))
	mux.HandleFunc("/v1/auth/google", requireMethod(http.MethodGet, authRateLimit("google_begin", 20, 15*time.Minute, AuthGoogleBeginSecure)))
	mux.HandleFunc("/v1/auth/google/callback", requireMethod(http.MethodGet, AuthGoogleCallbackSecure))
	mux.HandleFunc("/v1/auth/telegram", requireMethod(http.MethodGet, authRateLimit("telegram_begin", 20, 15*time.Minute, AuthTelegramBeginSecure)))
	mux.HandleFunc("/v1/auth/telegram/callback", requireMethod(http.MethodGet, AuthTelegramCallbackSecure))

	mux.HandleFunc("/v1/ai/providers", requireMethod(http.MethodGet, AIProviders))
	mux.HandleFunc("/v1/ai/estimate-summary/", requireMethod(http.MethodGet, EstimateAISummaryPostgres))
	mux.HandleFunc("/v1/analysis-batches", requireMethod(http.MethodPost, AnalysisBatchCreate))
	mux.HandleFunc("/v1/analysis-batches/", requireMethod(http.MethodGet, AnalysisBatchRouter))
	mux.HandleFunc("/v1/estimates", requireMethod(http.MethodGet, EstimateListPostgres))
	mux.HandleFunc("/v1/estimates/upload", requireMethod(http.MethodPost, EstimateUploadPostgres))
	mux.HandleFunc("/v1/estimates/compare", requireMethod(http.MethodPost, EstimateComparePostgres))
	mux.HandleFunc("/v1/estimates/", requireMethod(http.MethodGet, EstimateDetailRouterPostgres))

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	server := &http.Server{
		Addr:              addr,
		Handler:           recoverPanic(requestID(securityHeaders(cors(csrfProtection(maxBodyBytes(mux)))))),
		ReadHeaderTimeout: envDuration("SERVER_READ_HEADER_TIMEOUT", 5*time.Second),
		ReadTimeout:       envDuration("SERVER_READ_TIMEOUT", 60*time.Second),
		WriteTimeout:      envDuration("SERVER_WRITE_TIMEOUT", 60*time.Second),
		IdleTimeout:       envDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
		MaxHeaderBytes:    1 << 20,
	}

	serveErr := make(chan error, 1)
	go func() {
		log.Printf("smetacheck api listening on %s", addr)
		serveErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		closeDatabase()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("api server failed: %v", err)
		}
	case <-rootCtx.Done():
		shutdownTimeout := envDuration("SERVER_SHUTDOWN_TIMEOUT", 20*time.Second)
		if shutdownTimeout < time.Second {
			shutdownTimeout = 20 * time.Second
		}
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		shutdownErr := server.Shutdown(shutdownCtx)
		cancel()
		listenerErr := <-serveErr
		closeDatabase()
		if shutdownErr != nil {
			log.Printf("api graceful shutdown failed: %v", shutdownErr)
		}
		if listenerErr != nil && !errors.Is(listenerErr, http.ErrServerClosed) {
			log.Printf("api server stopped with error: %v", listenerErr)
		}
		log.Println("smetacheck api stopped")
	}
}
