package api

import (
	"context"
	"fmt"
)

func acquireBatchAdvisoryLock(ctx context.Context, batchID string) (func(), bool, error) {
	pool, err := getDB(ctx)
	if err != nil {
		return nil, false, err
	}
	if pool == nil {
		return nil, false, fmt.Errorf("postgresql is not configured")
	}
	connection, err := pool.Acquire(ctx)
	if err != nil {
		return nil, false, err
	}
	var acquired bool
	if err := connection.QueryRow(ctx, "SELECT pg_try_advisory_lock(hashtext($1))", batchID).Scan(&acquired); err != nil {
		connection.Release()
		return nil, false, err
	}
	if !acquired {
		connection.Release()
		return nil, false, nil
	}
	release := func() {
		var ignored bool
		_ = connection.QueryRow(context.Background(), "SELECT pg_advisory_unlock(hashtext($1))", batchID).Scan(&ignored)
		connection.Release()
	}
	return release, true, nil
}
