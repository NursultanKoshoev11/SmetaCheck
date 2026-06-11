package api

import (
	"context"
	"fmt"
	"time"
)

// ProcessNextAnalysisBatchSafe prevents two workers from processing the same batch
// concurrently and bounds the total processing time for a claimed batch.
func ProcessNextAnalysisBatchSafe(ctx context.Context) (bool, error) {
	batch, ownerID, found, err := pgClaimNextAnalysisBatch(ctx)
	if err != nil || !found {
		return found, err
	}

	release, acquired, err := acquireBatchAdvisoryLock(ctx, batch.ID)
	if err != nil {
		_ = pgFailAnalysisBatch(ctx, ownerID, batch.ID, err.Error())
		return true, fmt.Errorf("acquire batch lock: %w", err)
	}
	if !acquired {
		return true, nil
	}
	defer release()

	timeout := envDuration("AI_BATCH_TIMEOUT", 30*time.Minute)
	if timeout < time.Minute {
		timeout = time.Minute
	}
	batchCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := processClaimedAnalysisBatch(batchCtx, ownerID, batch); err != nil {
		_ = pgFailAnalysisBatch(context.Background(), ownerID, batch.ID, err.Error())
		return true, err
	}
	return true, nil
}
