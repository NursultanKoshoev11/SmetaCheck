package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func pgCreateAnalysisBatch(ctx context.Context, ownerID string, batch AnalysisBatch, files []AnalysisBatchFile) error {
	pool, err := getDB(ctx)
	if err != nil { return err }
	if pool == nil { return fmt.Errorf("postgresql is not configured") }
	tx, err := pool.Begin(ctx)
	if err != nil { return err }
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO analysis_batches (id, owner_id, provider, model, status, file_count, completed_count, attempts, created_at)
		VALUES ($1,$2,$3,$4,'pending',$5,0,0,$6)
	`, batch.ID, ownerID, batch.Provider, batch.Model, len(files), batch.CreatedAt)
	if err != nil { return err }
	for position, file := range files {
		_, err = tx.Exec(ctx, `
			INSERT INTO analysis_batch_files (id, batch_id, file_name, file_path, mime_type, file_size, position, status)
			VALUES ($1,$2,$3,$4,$5,$6,$7,'pending')
		`, file.ID, batch.ID, file.FileName, file.FilePath, file.MIMEType, file.FileSize, position)
		if err != nil { return err }
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit_logs (id,user_id,action,resource_type,resource_id) VALUES ($1,$2,'analysis_batch.created','analysis_batch',$3)`, newDatabaseID("aud"), ownerID, batch.ID)
	if err != nil { return err }
	return tx.Commit(ctx)
}

func pgFindAnalysisBatch(ctx context.Context, ownerID, batchID string) (AnalysisBatch, []AnalysisBatchFile, bool, error) {
	pool, err := getDB(ctx)
	if err != nil { return AnalysisBatch{}, nil, false, err }
	if pool == nil { return AnalysisBatch{}, nil, false, fmt.Errorf("postgresql is not configured") }
	batch, found, err := scanAnalysisBatch(pool.QueryRow(ctx, `
		SELECT id,status,provider,model,file_count,completed_count,attempts,COALESCE(error_message,''),report,created_at,started_at,completed_at
		FROM analysis_batches WHERE id=$1 AND owner_id=$2
	`, batchID, ownerID))
	if err != nil || !found { return batch, nil, found, err }
	files, err := pgLoadAnalysisBatchFiles(ctx, batchID)
	return batch, files, true, err
}

func pgLoadAnalysisBatchFiles(ctx context.Context, batchID string) ([]AnalysisBatchFile, error) {
	pool, err := getDB(ctx)
	if err != nil { return nil, err }
	rows, err := pool.Query(ctx, `
		SELECT id,batch_id,file_name,file_path,mime_type,file_size,status,COALESCE(estimate_id,''),COALESCE(error_message,'')
		FROM analysis_batch_files WHERE batch_id=$1 ORDER BY position
	`, batchID)
	if err != nil { return nil, err }
	defer rows.Close()
	result := make([]AnalysisBatchFile, 0)
	for rows.Next() {
		var file AnalysisBatchFile
		if err := rows.Scan(&file.ID,&file.BatchID,&file.FileName,&file.FilePath,&file.MIMEType,&file.FileSize,&file.Status,&file.EstimateID,&file.Error); err != nil { return nil, err }
		result = append(result, file)
	}
	return result, rows.Err()
}

func pgClaimNextAnalysisBatch(ctx context.Context) (AnalysisBatch, string, bool, error) {
	pool, err := getDB(ctx)
	if err != nil { return AnalysisBatch{}, "", false, err }
	tx, err := pool.Begin(ctx)
	if err != nil { return AnalysisBatch{}, "", false, err }
	defer tx.Rollback(ctx)
	var batchID, ownerID string
	err = tx.QueryRow(ctx, `
		SELECT id,owner_id FROM analysis_batches
		WHERE (status='pending' OR (status='processing' AND locked_at < now()-interval '10 minutes'))
		  AND attempts < $1
		ORDER BY created_at
		FOR UPDATE SKIP LOCKED LIMIT 1
	`, envInt64("AI_BATCH_MAX_ATTEMPTS", 3)).Scan(&batchID,&ownerID)
	if errors.Is(err, pgx.ErrNoRows) { return AnalysisBatch{}, "", false, nil }
	if err != nil { return AnalysisBatch{}, "", false, err }
	_, err = tx.Exec(ctx, `UPDATE analysis_batches SET status='processing',attempts=attempts+1,locked_at=now(),started_at=COALESCE(started_at,now()),error_message=NULL WHERE id=$1`, batchID)
	if err != nil { return AnalysisBatch{}, "", false, err }
	if err := tx.Commit(ctx); err != nil { return AnalysisBatch{}, "", false, err }
	batch, _, found, err := pgFindAnalysisBatch(ctx, ownerID, batchID)
	return batch, ownerID, found, err
}

func pgUpdateBatchFile(ctx context.Context, fileID, status, estimateID, errorMessage string) error {
	pool, err := getDB(ctx)
	if err != nil { return err }
	_, err = pool.Exec(ctx, `UPDATE analysis_batch_files SET status=$2,estimate_id=NULLIF($3,''),error_message=NULLIF($4,'') WHERE id=$1`, fileID,status,estimateID,errorMessage)
	return err
}

func pgTouchAnalysisBatch(ctx context.Context, batchID string, completedCount int) error {
	pool, err := getDB(ctx)
	if err != nil { return err }
	_, err = pool.Exec(ctx, `UPDATE analysis_batches SET completed_count=$2,locked_at=now() WHERE id=$1`, batchID,completedCount)
	return err
}

func pgCompleteAnalysisBatch(ctx context.Context, ownerID, batchID string, report UnifiedBatchReport) error {
	pool, err := getDB(ctx)
	if err != nil { return err }
	payload, err := json.Marshal(report)
	if err != nil { return err }
	tx, err := pool.Begin(ctx)
	if err != nil { return err }
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `UPDATE analysis_batches SET status='completed',completed_count=file_count,report=$2,error_message=NULL,completed_at=now(),locked_at=NULL WHERE id=$1 AND owner_id=$3`, batchID,payload,ownerID)
	if err != nil { return err }
	_, err = tx.Exec(ctx, `INSERT INTO audit_logs (id,user_id,action,resource_type,resource_id) VALUES ($1,$2,'analysis_batch.completed','analysis_batch',$3)`, newDatabaseID("aud"),ownerID,batchID)
	if err != nil { return err }
	return tx.Commit(ctx)
}

func pgFailAnalysisBatch(ctx context.Context, ownerID, batchID, message string) error {
	pool, err := getDB(ctx)
	if err != nil { return err }
	maxAttempts := envInt64("AI_BATCH_MAX_ATTEMPTS",3)
	_, err = pool.Exec(ctx, `
		UPDATE analysis_batches SET status=CASE WHEN attempts >= $3 THEN 'failed' ELSE 'pending' END,error_message=$2,locked_at=NULL,completed_at=CASE WHEN attempts >= $3 THEN now() ELSE NULL END
		WHERE id=$1 AND owner_id=$4
	`, batchID,truncateError(message),maxAttempts,ownerID)
	return err
}

func scanAnalysisBatch(row pgx.Row) (AnalysisBatch, bool, error) {
	var batch AnalysisBatch
	var payload []byte
	err := row.Scan(&batch.ID,&batch.Status,&batch.Provider,&batch.Model,&batch.FileCount,&batch.CompletedCount,&batch.Attempts,&batch.ErrorMessage,&payload,&batch.CreatedAt,&batch.StartedAt,&batch.CompletedAt)
	if errors.Is(err,pgx.ErrNoRows) { return AnalysisBatch{},false,nil }
	if err != nil { return AnalysisBatch{},false,err }
	if len(payload)>0 {
		var report UnifiedBatchReport
		if err := json.Unmarshal(payload,&report); err != nil { return AnalysisBatch{},false,err }
		batch.Report=&report
	}
	return batch,true,nil
}

func truncateError(value string) string {
	value = fmt.Sprintf("%s", value)
	if len(value)>2000 { return value[:2000] }
	return value
}

var _ = time.Time{}
