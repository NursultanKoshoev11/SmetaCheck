package worker

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func Run() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	interval := 10 * time.Second
	log.Printf("smetacheck worker started interval=%s", interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("smetacheck worker stopped")
			return
		case <-ticker.C:
			log.Println("smetacheck worker heartbeat: estimate processing is not implemented yet")
		}
	}
}
