package api

import (
	"context"
	"fmt"
)

func pgReleaseAnalysisBatchClaim(ctx context.Context, batchID string) error {
	pool, err := getDB(ctx)
	if err != nil {
		return err
	}
	if pool == nil {
		return fmt.Errorf("postgresql is not configured")
	}
	_, err = pool.Exec(ctx, `
		UPDATE analysis_batches
		SET status = 'pending', locked_at = NULL
		WHERE id = $1 AND status = 'processing'
	`, batchID)
	return err
}
