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

func releaseStorageUsageBestEffort(userID string, storageBytes int64) {
	if storageBytes <= 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rollbackAccountUsage(ctx, userID, UsageDelta{StorageBytes: storageBytes}); err != nil {
		log.Printf("storage usage release failed user_id=%s bytes=%d err=%v", userID, storageBytes, err)
	}
}
