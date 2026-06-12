package api

import (
	"errors"
	"testing"
	"time"
)

func TestQuotaExceededChecksEveryResource(t *testing.T) {
	used := UsageQuotaValues{
		UploadFiles:  9,
		UploadBytes:  90,
		AIJobs:       4,
		BatchJobs:    2,
		StorageBytes: 900,
	}
	limits := UsageQuotaLimits{
		MonthlyUploadFiles: 10,
		MonthlyUploadBytes: 100,
		MonthlyAIJobs:      5,
		MonthlyBatchJobs:   3,
		StorageBytes:       1000,
	}

	tests := []struct {
		name     string
		delta    UsageDelta
		resource string
	}{
		{"upload files", UsageDelta{UploadFiles: 2}, "monthly_upload_files"},
		{"upload bytes", UsageDelta{UploadBytes: 11}, "monthly_upload_bytes"},
		{"AI jobs", UsageDelta{AIJobs: 2}, "monthly_ai_jobs"},
		{"batch jobs", UsageDelta{BatchJobs: 2}, "monthly_batch_jobs"},
		{"storage", UsageDelta{StorageBytes: 101}, "storage_bytes"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := quotaExceeded(used, limits, test.delta)
			var quotaErr *UsageQuotaError
			if !errors.As(err, &quotaErr) {
				t.Fatalf("expected UsageQuotaError, got %v", err)
			}
			if quotaErr.Resource != test.resource {
				t.Fatalf("unexpected resource: %q", quotaErr.Resource)
			}
		})
	}
}

func TestQuotaAllowsExactRemainingCapacity(t *testing.T) {
	used := UsageQuotaValues{UploadFiles: 9, UploadBytes: 90, AIJobs: 4, BatchJobs: 2, StorageBytes: 900}
	limits := UsageQuotaLimits{MonthlyUploadFiles: 10, MonthlyUploadBytes: 100, MonthlyAIJobs: 5, MonthlyBatchJobs: 3, StorageBytes: 1000}
	delta := UsageDelta{UploadFiles: 1, UploadBytes: 10, AIJobs: 1, BatchJobs: 1, StorageBytes: 100}
	if err := quotaExceeded(used, limits, delta); err != nil {
		t.Fatalf("expected exact remaining capacity to be allowed, got %v", err)
	}
}

func TestCurrentUsagePeriodUsesUTCMonth(t *testing.T) {
	value := time.Date(2026, time.June, 30, 23, 59, 59, 0, time.FixedZone("UTC+6", 6*60*60))
	period := currentUsagePeriod(value)
	expected := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	if !period.Equal(expected) {
		t.Fatalf("unexpected usage period: %s", period)
	}
}
