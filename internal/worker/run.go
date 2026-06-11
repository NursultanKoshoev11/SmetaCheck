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
	cleanupInterval := 6 * time.Hour
	if value := os.Getenv("BATCH_CLEANUP_INTERVAL"); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil && parsed >= time.Minute {
			cleanupInterval = parsed
		}
	}
	log.Printf("smetacheck analysis worker started poll_interval=%s cleanup_interval=%s", pollInterval, cleanupInterval)

	pollTicker := time.NewTicker(pollInterval)
	defer pollTicker.Stop()
	cleanupTicker := time.NewTicker(cleanupInterval)
	defer cleanupTicker.Stop()

	cleanupCtx, cleanupCancel := context.WithTimeout(ctx, 2*time.Minute)
	if removed, err := api.CleanupExpiredBatchFiles(cleanupCtx); err != nil {
		log.Printf("batch file cleanup error: %v", err)
	} else if removed > 0 {
		log.Printf("batch file cleanup removed=%d", removed)
	}
	cleanupCancel()

	for {
		select {
		case <-ctx.Done():
			log.Println("smetacheck analysis worker stopped")
			return
		case <-cleanupTicker.C:
			cleanupCtx, cleanupCancel := context.WithTimeout(ctx, 2*time.Minute)
			removed, err := api.CleanupExpiredBatchFiles(cleanupCtx)
			cleanupCancel()
			if err != nil {
				log.Printf("batch file cleanup error: %v", err)
			} else if removed > 0 {
				log.Printf("batch file cleanup removed=%d", removed)
			}
		case <-pollTicker.C:
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
