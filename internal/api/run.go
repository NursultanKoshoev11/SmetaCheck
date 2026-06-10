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

	mux := http.NewServeMux()
	mux.HandleFunc("/health", Health)
	mux.HandleFunc("/ready", Ready)
	mux.HandleFunc("/v1/auth/register", requireMethod(http.MethodPost, AuthRegister))
	mux.HandleFunc("/v1/auth/login", requireMethod(http.MethodPost, AuthLogin))
	mux.HandleFunc("/v1/estimates", EstimateList)
	mux.HandleFunc("/v1/estimates/upload", requireMethod(http.MethodPost, EstimateUpload))
	mux.HandleFunc("/v1/estimates/compare", requireMethod(http.MethodPost, EstimateCompare))
	mux.HandleFunc("/v1/estimates/", EstimateDetailRouter)

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           recoverPanic(requestID(securityHeaders(cors(maxBodyBytes(mux))))),
		ReadHeaderTimeout: envDuration("SERVER_READ_HEADER_TIMEOUT", 5*time.Second),
		ReadTimeout:       envDuration("SERVER_READ_TIMEOUT", 30*time.Second),
		WriteTimeout:      envDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
		IdleTimeout:       envDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
		MaxHeaderBytes:    1 << 20,
	}

	log.Printf("smetacheck api listening on %s", addr)
	log.Fatal(server.ListenAndServe())
}
