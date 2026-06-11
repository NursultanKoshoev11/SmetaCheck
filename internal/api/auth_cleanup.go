package api

import (
	"context"
	"log"
	"time"
)

func startAuthCleanup() {
	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()
		for {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			if err := cleanupAuthRecords(ctx); err != nil {
				log.Printf("auth cleanup failed: %v", err)
			}
			cancel()
			<-ticker.C
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
