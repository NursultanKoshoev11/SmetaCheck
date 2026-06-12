package api

import (
	"context"
	"log"
	"time"
)

func startAuthCleanup(parent context.Context) {
	go func() {
		runCleanup := func() {
			ctx, cancel := context.WithTimeout(parent, 20*time.Second)
			defer cancel()
			if err := cleanupAuthRecords(ctx); err != nil && parent.Err() == nil {
				log.Printf("auth cleanup failed: %v", err)
			}
		}

		runCleanup()
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-parent.Done():
				return
			case <-ticker.C:
				runCleanup()
			}
		}
	}()
}

func cleanupAuthRecords(ctx context.Context) error {
	pool, err := getDB(ctx)
	if err != nil {
		return err
	}
	if pool == nil {
		return nil
	}
	statements := []string{
		`DELETE FROM oauth_states WHERE expires_at < now()`,
		`DELETE FROM auth_tokens WHERE expires_at < now() OR (consumed_at IS NOT NULL AND consumed_at < now() - interval '7 days')`,
		`DELETE FROM auth_sessions WHERE expires_at < now() OR (revoked_at IS NOT NULL AND revoked_at < now() - interval '7 days')`,
		`DELETE FROM auth_rate_limits WHERE updated_at < now() - interval '2 days'`,
	}
	for _, statement := range statements {
		if _, err := pool.Exec(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}
