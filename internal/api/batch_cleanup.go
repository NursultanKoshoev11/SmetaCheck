package api

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type expiredBatchFile struct {
	ID         string
	OwnerID    string
	BatchID    string
	EstimateID string
	FilePath   string
	FileSize   int64
}

func CleanupExpiredBatchFiles(ctx context.Context) (int, error) {
	retentionDays := envInt64("BATCH_FILE_RETENTION_DAYS", 30)
	if retentionDays <= 0 {
		return 0, nil
	}
	if retentionDays > 3650 {
		retentionDays = 3650
	}

	pool, err := getDB(ctx)
	if err != nil {
		return 0, err
	}
	if pool == nil {
		return 0, fmt.Errorf("postgresql is not configured")
	}

	cutoff := time.Now().UTC().Add(-time.Duration(retentionDays) * 24 * time.Hour)
	rows, err := pool.Query(ctx, `
		SELECT f.id, b.owner_id, f.batch_id, COALESCE(f.estimate_id, ''), f.file_path, f.file_size
		FROM analysis_batch_files f
		JOIN analysis_batches b ON b.id = f.batch_id
		WHERE b.status IN ('completed', 'failed')
		  AND COALESCE(b.completed_at, b.created_at) < $1
		  AND f.file_path <> ''
		ORDER BY b.completed_at NULLS FIRST
		LIMIT 500
	`, cutoff)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	files := make([]expiredBatchFile, 0)
	for rows.Next() {
		var file expiredBatchFile
		if err := rows.Scan(&file.ID, &file.OwnerID, &file.BatchID, &file.EstimateID, &file.FilePath, &file.FileSize); err != nil {
			return 0, err
		}
		files = append(files, file)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	removed := 0
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return removed, err
		}
		if err := removeStoredBatchFile(file.FilePath); err != nil {
			return removed, fmt.Errorf("remove batch file %s: %w", file.ID, err)
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return removed, err
		}
		result, updateErr := tx.Exec(ctx, `
			UPDATE analysis_batch_files
			SET file_path = ''
			WHERE id = $1 AND file_path = $2
		`, file.ID, file.FilePath)
		if updateErr == nil && result.RowsAffected() == 1 && file.EstimateID != "" {
			_, updateErr = tx.Exec(ctx, `
				UPDATE estimates
				SET file_path = ''
				WHERE id = $1 AND file_path = $2
			`, file.EstimateID, file.FilePath)
		}
		if updateErr == nil && result.RowsAffected() == 1 {
			_, updateErr = tx.Exec(ctx, `
				UPDATE account_storage_usage
				SET storage_bytes=GREATEST(storage_bytes-$2,0),updated_at=now()
				WHERE user_id=$1
			`, file.OwnerID, file.FileSize)
		}
		if updateErr == nil && result.RowsAffected() == 1 {
			_, updateErr = tx.Exec(ctx, `
				INSERT INTO audit_logs (id, user_id, action, resource_type, resource_id)
				VALUES ($1, $2, 'analysis_batch.file_purged', 'analysis_batch', $3)
			`, newDatabaseID("aud"), file.OwnerID, file.BatchID)
		}
		if updateErr != nil {
			_ = tx.Rollback(ctx)
			return removed, updateErr
		}
		if err := tx.Commit(ctx); err != nil {
			return removed, err
		}
		if result.RowsAffected() == 1 {
			removed++
		}
	}
	return removed, nil
}

func removeStoredBatchFile(path string) error {
	root, err := filepath.Abs(estimateUploadDir())
	if err != nil {
		return err
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(root, absolute)
	if err != nil {
		return err
	}
	if relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("refusing to delete path outside upload directory")
	}
	if err := os.Remove(absolute); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
