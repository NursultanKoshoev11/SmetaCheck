//go:build integration

package api

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestUsageReservationAndRollback(t *testing.T) {
	ctx, pool := integrationDatabase(t)
	setTestQuotaLimits(t, 2)

	userID := "usr_usage_" + fmt.Sprintf("%d", time.Now().UnixNano())
	if _, err := pool.Exec(ctx, `INSERT INTO users (id,full_name) VALUES ($1,'Usage test')`, userID); err != nil {
		t.Fatalf("insert test user: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM users WHERE id=$1`, userID)
	})

	delta := UsageDelta{UploadFiles: 1, UploadBytes: 100, AIJobs: 1, BatchJobs: 1, StorageBytes: 100}
	if err := reserveAccountUsage(ctx, userID, delta); err != nil {
		t.Fatalf("first reservation failed: %v", err)
	}
	if err := reserveAccountUsage(ctx, userID, delta); err != nil {
		t.Fatalf("second reservation at exact limit failed: %v", err)
	}
	if err := reserveAccountUsage(ctx, userID, UsageDelta{UploadFiles: 1}); err == nil {
		t.Fatal("expected third upload reservation to exceed quota")
	} else {
		var quotaErr *UsageQuotaError
		if !errors.As(err, &quotaErr) || quotaErr.Resource != "monthly_upload_files" {
			t.Fatalf("unexpected quota error: %v", err)
		}
	}

	snapshot, err := loadAccountUsage(ctx, userID)
	if err != nil {
		t.Fatalf("load usage snapshot: %v", err)
	}
	if snapshot.Used.UploadFiles != 2 || snapshot.Used.UploadBytes != 200 || snapshot.Used.StorageBytes != 200 {
		t.Fatalf("unexpected reserved usage: %+v", snapshot.Used)
	}
	if snapshot.Used.AIJobs != 2 || snapshot.Used.BatchJobs != 2 {
		t.Fatalf("unexpected AI or batch usage: %+v", snapshot.Used)
	}

	if err := rollbackAccountUsage(ctx, userID, delta); err != nil {
		t.Fatalf("rollback usage: %v", err)
	}
	snapshot, err = loadAccountUsage(ctx, userID)
	if err != nil {
		t.Fatalf("load usage after rollback: %v", err)
	}
	if snapshot.Used.UploadFiles != 1 || snapshot.Used.UploadBytes != 100 || snapshot.Used.StorageBytes != 100 {
		t.Fatalf("unexpected usage after rollback: %+v", snapshot.Used)
	}
}

func TestConcurrentUsageReservationsAreAtomic(t *testing.T) {
	ctx, pool := integrationDatabase(t)
	setTestQuotaLimits(t, 1)

	userID := "usr_usage_concurrent_" + fmt.Sprintf("%d", time.Now().UnixNano())
	if _, err := pool.Exec(ctx, `INSERT INTO users (id,full_name) VALUES ($1,'Concurrent usage test')`, userID); err != nil {
		t.Fatalf("insert test user: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM users WHERE id=$1`, userID)
	})

	start := make(chan struct{})
	results := make(chan error, 2)
	var waitGroup sync.WaitGroup
	for index := 0; index < 2; index++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			<-start
			requestCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			results <- reserveAccountUsage(requestCtx, userID, UsageDelta{UploadFiles: 1, UploadBytes: 100, StorageBytes: 100})
		}()
	}
	close(start)
	waitGroup.Wait()
	close(results)

	successes, quotaFailures := 0, 0
	for err := range results {
		if err == nil {
			successes++
			continue
		}
		var quotaErr *UsageQuotaError
		if errors.As(err, &quotaErr) {
			quotaFailures++
			continue
		}
		t.Fatalf("unexpected concurrent reservation error: %v", err)
	}
	if successes != 1 || quotaFailures != 1 {
		t.Fatalf("expected one success and one quota failure, got successes=%d quota_failures=%d", successes, quotaFailures)
	}

	snapshot, err := loadAccountUsage(ctx, userID)
	if err != nil {
		t.Fatalf("load concurrent usage: %v", err)
	}
	if snapshot.Used.UploadFiles != 1 || snapshot.Used.StorageBytes != 100 {
		t.Fatalf("concurrent usage counters exceeded quota: %+v", snapshot.Used)
	}
}

func setTestQuotaLimits(t *testing.T, uploadFiles int64) {
	t.Helper()
	t.Setenv("QUOTA_MONTHLY_UPLOAD_FILES", fmt.Sprintf("%d", uploadFiles))
	t.Setenv("QUOTA_MONTHLY_UPLOAD_MB", "10")
	t.Setenv("QUOTA_MONTHLY_AI_JOBS", "10")
	t.Setenv("QUOTA_MONTHLY_BATCHES", "10")
	t.Setenv("QUOTA_STORAGE_MB", "10")
}
