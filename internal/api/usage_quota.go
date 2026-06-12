package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

type UsageDelta struct {
	UploadFiles  int64
	UploadBytes  int64
	AIJobs       int64
	BatchJobs    int64
	StorageBytes int64
}

type UsageQuotaLimits struct {
	MonthlyUploadFiles int64 `json:"monthly_upload_files"`
	MonthlyUploadBytes int64 `json:"monthly_upload_bytes"`
	MonthlyAIJobs      int64 `json:"monthly_ai_jobs"`
	MonthlyBatchJobs   int64 `json:"monthly_batch_jobs"`
	StorageBytes       int64 `json:"storage_bytes"`
}

type UsageQuotaSnapshot struct {
	PeriodStart time.Time        `json:"period_start"`
	Used        UsageQuotaValues `json:"used"`
	Limits      UsageQuotaLimits `json:"limits"`
}

type UsageQuotaValues struct {
	UploadFiles int64 `json:"upload_files"`
	UploadBytes int64 `json:"upload_bytes"`
	AIJobs      int64 `json:"ai_jobs"`
	BatchJobs   int64 `json:"batch_jobs"`
	StorageBytes int64 `json:"storage_bytes"`
}

type UsageQuotaError struct {
	Resource  string
	Limit     int64
	Used      int64
	Requested int64
}

func (err *UsageQuotaError) Error() string {
	return fmt.Sprintf("%s quota exceeded", err.Resource)
}

func usageQuotaLimits() UsageQuotaLimits {
	const megabyte = int64(1024 * 1024)
	return UsageQuotaLimits{
		MonthlyUploadFiles: envInt64("QUOTA_MONTHLY_UPLOAD_FILES", 100),
		MonthlyUploadBytes: envInt64("QUOTA_MONTHLY_UPLOAD_MB", 2048) * megabyte,
		MonthlyAIJobs:      envInt64("QUOTA_MONTHLY_AI_JOBS", 200),
		MonthlyBatchJobs:   envInt64("QUOTA_MONTHLY_BATCHES", 50),
		StorageBytes:       envInt64("QUOTA_STORAGE_MB", 4096) * megabyte,
	}
}

func currentUsagePeriod(now time.Time) time.Time {
	utc := now.UTC()
	return time.Date(utc.Year(), utc.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func reserveAccountUsage(ctx context.Context, userID string, delta UsageDelta) error {
	if userID == "" {
		return fmt.Errorf("user id is required for usage reservation")
	}
	if delta.UploadFiles < 0 || delta.UploadBytes < 0 || delta.AIJobs < 0 || delta.BatchJobs < 0 || delta.StorageBytes < 0 {
		return fmt.Errorf("usage reservation cannot be negative")
	}
	if delta == (UsageDelta{}) {
		return nil
	}

	pool, err := getDB(ctx)
	if err != nil {
		return err
	}
	if pool == nil {
		return fmt.Errorf("postgresql is unavailable")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1)::bigint)`, "usage|"+userID); err != nil {
		return err
	}
	period := currentUsagePeriod(time.Now())
	if _, err := tx.Exec(ctx, `
		INSERT INTO account_storage_usage (user_id,storage_bytes)
		VALUES ($1,0)
		ON CONFLICT (user_id) DO NOTHING
	`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO account_usage_monthly (user_id,period_start)
		VALUES ($1,$2)
		ON CONFLICT (user_id,period_start) DO NOTHING
	`, userID, period); err != nil {
		return err
	}

	var used UsageQuotaValues
	if err := tx.QueryRow(ctx, `
		SELECT m.upload_files,m.upload_bytes,m.ai_jobs,m.batch_jobs,s.storage_bytes
		FROM account_usage_monthly m
		JOIN account_storage_usage s ON s.user_id=m.user_id
		WHERE m.user_id=$1 AND m.period_start=$2
		FOR UPDATE OF m,s
	`, userID, period).Scan(&used.UploadFiles, &used.UploadBytes, &used.AIJobs, &used.BatchJobs, &used.StorageBytes); err != nil {
		return err
	}
	limits := usageQuotaLimits()
	if quotaErr := quotaExceeded(used, limits, delta); quotaErr != nil {
		return quotaErr
	}
	if _, err := tx.Exec(ctx, `
		UPDATE account_usage_monthly SET
			upload_files=upload_files+$3,
			upload_bytes=upload_bytes+$4,
			ai_jobs=ai_jobs+$5,
			batch_jobs=batch_jobs+$6,
			updated_at=now()
		WHERE user_id=$1 AND period_start=$2
	`, userID, period, delta.UploadFiles, delta.UploadBytes, delta.AIJobs, delta.BatchJobs); err != nil {
		return err
	}
	if delta.StorageBytes > 0 {
		if _, err := tx.Exec(ctx, `
			UPDATE account_storage_usage
			SET storage_bytes=storage_bytes+$2,updated_at=now()
			WHERE user_id=$1
		`, userID, delta.StorageBytes); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func quotaExceeded(used UsageQuotaValues, limits UsageQuotaLimits, delta UsageDelta) error {
	checks := []struct {
		resource  string
		used      int64
		requested int64
		limit     int64
	}{
		{"monthly_upload_files", used.UploadFiles, delta.UploadFiles, limits.MonthlyUploadFiles},
		{"monthly_upload_bytes", used.UploadBytes, delta.UploadBytes, limits.MonthlyUploadBytes},
		{"monthly_ai_jobs", used.AIJobs, delta.AIJobs, limits.MonthlyAIJobs},
		{"monthly_batch_jobs", used.BatchJobs, delta.BatchJobs, limits.MonthlyBatchJobs},
		{"storage_bytes", used.StorageBytes, delta.StorageBytes, limits.StorageBytes},
	}
	for _, check := range checks {
		if check.requested > 0 && (check.limit <= 0 || check.used > check.limit-check.requested) {
			return &UsageQuotaError{Resource: check.resource, Limit: check.limit, Used: check.used, Requested: check.requested}
		}
	}
	return nil
}

func rollbackAccountUsage(ctx context.Context, userID string, delta UsageDelta) error {
	if userID == "" || delta == (UsageDelta{}) {
		return nil
	}
	pool, err := getDB(ctx)
	if err != nil {
		return err
	}
	if pool == nil {
		return fmt.Errorf("postgresql is unavailable")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1)::bigint)`, "usage|"+userID); err != nil {
		return err
	}
	period := currentUsagePeriod(time.Now())
	if _, err := tx.Exec(ctx, `
		UPDATE account_usage_monthly SET
			upload_files=GREATEST(upload_files-$3,0),
			upload_bytes=GREATEST(upload_bytes-$4,0),
			ai_jobs=GREATEST(ai_jobs-$5,0),
			batch_jobs=GREATEST(batch_jobs-$6,0),
			updated_at=now()
		WHERE user_id=$1 AND period_start=$2
	`, userID, period, delta.UploadFiles, delta.UploadBytes, delta.AIJobs, delta.BatchJobs); err != nil {
		return err
	}
	if delta.StorageBytes > 0 {
		if _, err := tx.Exec(ctx, `
			UPDATE account_storage_usage
			SET storage_bytes=GREATEST(storage_bytes-$2,0),updated_at=now()
			WHERE user_id=$1
		`, userID, delta.StorageBytes); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func loadAccountUsage(ctx context.Context, userID string) (UsageQuotaSnapshot, error) {
	pool, err := getDB(ctx)
	if err != nil {
		return UsageQuotaSnapshot{}, err
	}
	if pool == nil {
		return UsageQuotaSnapshot{}, fmt.Errorf("postgresql is unavailable")
	}
	period := currentUsagePeriod(time.Now())
	snapshot := UsageQuotaSnapshot{PeriodStart: period, Limits: usageQuotaLimits()}
	err = pool.QueryRow(ctx, `
		SELECT
			COALESCE(m.upload_files,0),COALESCE(m.upload_bytes,0),
			COALESCE(m.ai_jobs,0),COALESCE(m.batch_jobs,0),
			COALESCE(s.storage_bytes,0)
		FROM users u
		LEFT JOIN account_usage_monthly m ON m.user_id=u.id AND m.period_start=$2
		LEFT JOIN account_storage_usage s ON s.user_id=u.id
		WHERE u.id=$1
	`, userID, period).Scan(
		&snapshot.Used.UploadFiles,
		&snapshot.Used.UploadBytes,
		&snapshot.Used.AIJobs,
		&snapshot.Used.BatchJobs,
		&snapshot.Used.StorageBytes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return UsageQuotaSnapshot{}, fmt.Errorf("user not found")
	}
	return snapshot, err
}

func AccountUsage(w http.ResponseWriter, r *http.Request) {
	user, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	snapshot, err := loadAccountUsage(r.Context(), user.ID)
	if err != nil {
		estimateWriteError(w, http.StatusInternalServerError, "cannot load account usage")
		return
	}
	estimateWriteJSON(w, http.StatusOK, map[string]any{"usage": snapshot})
}

func writeUsageQuotaError(w http.ResponseWriter, err error) bool {
	var quotaErr *UsageQuotaError
	if !errors.As(err, &quotaErr) {
		return false
	}
	w.Header().Set("X-Quota-Resource", quotaErr.Resource)
	w.Header().Set("X-Quota-Limit", itoa64(quotaErr.Limit))
	w.Header().Set("X-Quota-Used", itoa64(quotaErr.Used))
	estimateWriteJSON(w, http.StatusTooManyRequests, map[string]any{
		"error":     "usage quota exceeded",
		"resource":  quotaErr.Resource,
		"limit":     quotaErr.Limit,
		"used":      quotaErr.Used,
		"requested": quotaErr.Requested,
	})
	return true
}
