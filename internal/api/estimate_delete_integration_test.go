//go:build integration

package api

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestEstimateDeletionIsOwnerScoped(t *testing.T) {
	ctx, pool := integrationDatabase(t)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	ownerID := "usr_delete_owner_" + suffix
	otherUserID := "usr_delete_other_" + suffix
	estimateID := "est_delete_" + suffix
	filePath := "/var/lib/smetacheck/uploads/" + estimateID + ".xlsx"
	reportPath := "/var/lib/smetacheck/reports/" + estimateID + ".txt"

	if _, err := pool.Exec(ctx, `
		INSERT INTO users (id, full_name) VALUES ($1,'Owner'),($2,'Other user')
	`, ownerID, otherUserID); err != nil {
		t.Fatalf("insert users: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM users WHERE id IN ($1,$2)`, ownerID, otherUserID)
	})

	if _, err := pool.Exec(ctx, `
		INSERT INTO estimates (id,owner_id,file_name,file_path,report_path)
		VALUES ($1,$2,'estimate.xlsx',$3,$4)
	`, estimateID, ownerID, filePath, reportPath); err != nil {
		t.Fatalf("insert estimate: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO estimate_items (id,estimate_id,row_number,name,total)
		VALUES ($1,$2,1,'Test item',100)
	`, "itm_delete_"+suffix, estimateID); err != nil {
		t.Fatalf("insert estimate child: %v", err)
	}

	_, _, found, err := pgDeleteEstimate(ctx, otherUserID, estimateID)
	if err != nil {
		t.Fatalf("delete by another owner returned error: %v", err)
	}
	if found {
		t.Fatal("another user must not be able to delete the estimate")
	}

	var estimateStillExists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM estimates WHERE id=$1)`, estimateID).Scan(&estimateStillExists); err != nil {
		t.Fatalf("check estimate after denied deletion: %v", err)
	}
	if !estimateStillExists {
		t.Fatal("estimate was deleted by a different owner")
	}

	deletedFilePath, deletedReportPath, found, err := pgDeleteEstimate(ctx, ownerID, estimateID)
	if err != nil {
		t.Fatalf("owner deletion returned error: %v", err)
	}
	if !found {
		t.Fatal("owner deletion did not find the estimate")
	}
	if deletedFilePath != filePath || deletedReportPath != reportPath {
		t.Fatalf("unexpected deleted storage paths: %q %q", deletedFilePath, deletedReportPath)
	}

	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM estimates WHERE id=$1)`, estimateID).Scan(&estimateStillExists); err != nil {
		t.Fatalf("check estimate after owner deletion: %v", err)
	}
	if estimateStillExists {
		t.Fatal("estimate still exists after owner deletion")
	}
	var childStillExists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM estimate_items WHERE estimate_id=$1)`, estimateID).Scan(&childStillExists); err != nil {
		t.Fatalf("check estimate child cascade: %v", err)
	}
	if childStillExists {
		t.Fatal("estimate child rows were not deleted by cascade")
	}
	assertEstimateDeletionAudit(t, ctx, pool, ownerID, estimateID)
}

func TestDeletingBatchEstimatePreservesBatchOwnedFile(t *testing.T) {
	ctx, pool := integrationDatabase(t)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	ownerID := "usr_batch_delete_" + suffix
	batchID := "bat_delete_" + suffix
	batchFileID := "bfl_delete_" + suffix
	estimateID := "est_batch_delete_" + suffix
	filePath := "/var/lib/smetacheck/uploads/" + batchFileID + ".xlsx"
	reportPath := "/var/lib/smetacheck/reports/" + estimateID + ".txt"

	if _, err := pool.Exec(ctx, `INSERT INTO users (id,full_name) VALUES ($1,'Batch owner')`, ownerID); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM users WHERE id=$1`, ownerID)
	})
	if _, err := pool.Exec(ctx, `
		INSERT INTO analysis_batches (id,owner_id,provider,model,status,file_count,completed_count,completed_at)
		VALUES ($1,$2,'rules','rules','completed',1,1,now())
	`, batchID, ownerID); err != nil {
		t.Fatalf("insert batch: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO estimates (id,owner_id,file_name,file_path,report_path)
		VALUES ($1,$2,'batch.xlsx',$3,$4)
	`, estimateID, ownerID, filePath, reportPath); err != nil {
		t.Fatalf("insert estimate: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO analysis_batch_files (
			id,batch_id,file_name,file_path,mime_type,file_size,position,status,estimate_id
		) VALUES ($1,$2,'batch.xlsx',$3,'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',100,0,'completed',$4)
	`, batchFileID, batchID, filePath, estimateID); err != nil {
		t.Fatalf("insert batch file: %v", err)
	}

	deletedFilePath, deletedReportPath, found, err := pgDeleteEstimate(ctx, ownerID, estimateID)
	if err != nil {
		t.Fatalf("delete batch estimate: %v", err)
	}
	if !found {
		t.Fatal("batch estimate was not found")
	}
	if deletedFilePath != "" {
		t.Fatalf("batch-owned upload must be retained for batch cleanup, got %q", deletedFilePath)
	}
	if deletedReportPath != reportPath {
		t.Fatalf("estimate report should still be removed, got %q", deletedReportPath)
	}

	var retainedPath string
	var linkedEstimateID *string
	if err := pool.QueryRow(ctx, `
		SELECT file_path, estimate_id FROM analysis_batch_files WHERE id=$1
	`, batchFileID).Scan(&retainedPath, &linkedEstimateID); err != nil {
		t.Fatalf("load batch file after estimate deletion: %v", err)
	}
	if retainedPath != filePath {
		t.Fatalf("batch file path was changed unexpectedly: %q", retainedPath)
	}
	if linkedEstimateID != nil {
		t.Fatalf("batch file estimate link must be cleared by ON DELETE SET NULL: %v", *linkedEstimateID)
	}
	assertEstimateDeletionAudit(t, ctx, pool, ownerID, estimateID)
}

func integrationDatabase(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	t.Setenv("DATABASE_URL", databaseURL)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	if err := migrateDatabase(ctx); err != nil {
		t.Fatalf("database migration failed: %v", err)
	}
	pool, err := getDB(ctx)
	if err != nil {
		t.Fatalf("getDB failed: %v", err)
	}
	return ctx, pool
}

func assertEstimateDeletionAudit(t *testing.T, ctx context.Context, pool *pgxpool.Pool, ownerID, estimateID string) {
	t.Helper()
	var auditExists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM audit_logs
			WHERE user_id=$1 AND action='estimate.deleted' AND resource_id=$2
		)
	`, ownerID, estimateID).Scan(&auditExists); err != nil {
		t.Fatalf("check deletion audit log: %v", err)
	}
	if !auditExists {
		t.Fatal("estimate deletion audit event was not recorded")
	}
}
