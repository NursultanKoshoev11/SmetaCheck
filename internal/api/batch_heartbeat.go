package api

import (
	"context"
	"fmt"
	"time"
)

func pgHeartbeatAnalysisBatch(ctx context.Context, batchID string) error {
	pool, err := getDB(ctx)
	if err != nil {
		return err
	}
	if pool == nil {
		return fmt.Errorf("postgresql is not configured")
	}
	command, err := pool.Exec(ctx, `
		UPDATE analysis_batches
		SET locked_at = now()
		WHERE id = $1 AND status = 'processing'
	`, batchID)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return fmt.Errorf("analysis batch lease is no longer owned by this worker")
	}
	return nil
}

func runWithBatchHeartbeat(
	parent context.Context,
	batchID string,
	work func(context.Context) (AIProviderBatchResponse, error),
) (AIProviderBatchResponse, error) {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	interval := envDuration("AI_BATCH_HEARTBEAT_INTERVAL", 30*time.Second)
	if interval < 5*time.Second {
		interval = 5 * time.Second
	}
	heartbeatErrors := make(chan error, 1)
	done := make(chan struct{})

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := pgHeartbeatAnalysisBatch(ctx, batchID); err != nil {
					select {
					case heartbeatErrors <- err:
					default:
					}
					cancel()
					return
				}
			}
		}
	}()

	result, workErr := work(ctx)
	cancel()
	<-done
	select {
	case heartbeatErr := <-heartbeatErrors:
		if workErr != nil {
			return AIProviderBatchResponse{}, fmt.Errorf("AI work failed: %v; heartbeat failed: %w", workErr, heartbeatErr)
		}
		return AIProviderBatchResponse{}, heartbeatErr
	default:
	}
	return result, workErr
}
