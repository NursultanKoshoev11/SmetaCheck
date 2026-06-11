package worker

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/NursultanKoshoev11/SmetaCheck/internal/api"
)

func Run() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	startupCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	if err := api.PrepareDatabase(startupCtx); err != nil {
		cancel()
		log.Fatalf("worker database initialization failed: %v", err)
	}
	cancel()

	pollInterval := 2 * time.Second
	if value := os.Getenv("WORKER_POLL_INTERVAL"); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil && parsed >= 250*time.Millisecond {
			pollInterval = parsed
		}
	}
	log.Printf("smetacheck analysis worker started poll_interval=%s", pollInterval)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Println("smetacheck analysis worker stopped")
			return
		case <-ticker.C:
			for {
				claimed, err := api.ProcessNextAnalysisBatchSafe(ctx)
				if err != nil {
					log.Printf("analysis batch processing error: %v", err)
					break
				}
				if !claimed {
					break
				}
			}
		}
	}
}
