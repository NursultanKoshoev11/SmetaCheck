package api

import (
	"context"
	"log"
	"time"
)

func rollbackUsageBestEffort(userID string, delta UsageDelta) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rollbackAccountUsage(ctx, userID, delta); err != nil {
		log.Printf("usage reservation rollback failed user_id=%s err=%v", userID, err)
	}
}
