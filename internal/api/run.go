package api

import (
	"log"
	"net/http"
	"os"
	"time"
)

func Run() {
	if err := validateProductionConfig(); err != nil {
		log.Fatalf("configuration error: %v", err)
	}
	initDatabaseForRun()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", Health)
	mux.HandleFunc("/ready", Ready)

	mux.HandleFunc("/v1/auth/providers", requireMethod(http.MethodGet, AuthProviders))
	mux.HandleFunc("/v1/auth/register", requireMethod(http.MethodPost, AuthRegisterEmail))
	mux.HandleFunc("/v1/auth/login", requireMethod(http.MethodPost, AuthLoginEmail))
	mux.HandleFunc("/v1/auth/me", requireMethod(http.MethodGet, AuthMe))
	mux.HandleFunc("/v1/auth/refresh", requireMethod(http.MethodPost, AuthRefresh))
	mux.HandleFunc("/v1/auth/logout", requireMethod(http.MethodPost, AuthLogout))
	mux.HandleFunc("/v1/auth/email/verify", requireMethod(http.MethodGet, AuthVerifyEmail))
	mux.HandleFunc("/v1/auth/email/resend", requireMethod(http.MethodPost, AuthResendVerification))
	mux.HandleFunc("/v1/auth/password/forgot", requireMethod(http.MethodPost, AuthForgotPassword))
	mux.HandleFunc("/v1/auth/password/reset", requireMethod(http.MethodPost, AuthResetPassword))
	mux.HandleFunc("/v1/auth/google", requireMethod(http.MethodGet, AuthGoogleBegin))
	mux.HandleFunc("/v1/auth/google/callback", requireMethod(http.MethodGet, AuthGoogleCallback))
	mux.HandleFunc("/v1/auth/telegram", requireMethod(http.MethodGet, AuthTelegramBegin))
	mux.HandleFunc("/v1/auth/telegram/callback", requireMethod(http.MethodGet, AuthTelegramCallback))

	mux.HandleFunc("/v1/ai/estimate-summary/", requireMethod(http.MethodGet, EstimateAISummaryPostgres))
	mux.HandleFunc("/v1/analysis-batches", requireMethod(http.MethodPost, AnalysisBatchCreate))
	mux.HandleFunc("/v1/analysis-batches/", requireMethod(http.MethodGet, AnalysisBatchRouter))
	mux.HandleFunc("/v1/estimates", requireMethod(http.MethodGet, EstimateListPostgres))
	mux.HandleFunc("/v1/estimates/upload", requireMethod(http.MethodPost, EstimateUploadPostgres))
	mux.HandleFunc("/v1/estimates/compare", requireMethod(http.MethodPost, EstimateComparePostgres))
	mux.HandleFunc("/v1/estimates/", requireMethod(http.MethodGet, EstimateDetailRouterPostgres))

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" { addr = ":8080" }
	server := &http.Server{
		Addr: addr,
		Handler: recoverPanic(requestID(securityHeaders(cors(csrfProtection(maxBodyBytes(mux))))))),
		ReadHeaderTimeout: envDuration("SERVER_READ_HEADER_TIMEOUT", 5*time.Second),
		ReadTimeout: envDuration("SERVER_READ_TIMEOUT", 60*time.Second),
		WriteTimeout: envDuration("SERVER_WRITE_TIMEOUT", 60*time.Second),
		IdleTimeout: envDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
		MaxHeaderBytes: 1 << 20,
	}
	log.Printf("smetacheck api listening on %s", addr)
	log.Fatal(server.ListenAndServe())
}
