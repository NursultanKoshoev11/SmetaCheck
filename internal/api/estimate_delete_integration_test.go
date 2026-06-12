//go:build integration

package api

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestEstimateDeletionIsOwnerScoped(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	t.Setenv("DATABASE_URL", databaseURL)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := migrateDatabase(ctx); err != nil {
		t.Fatalf("database migration failed: %v", err)
	}
	pool, err := getDB(ctx)
	if err != nil {
		t.Fatalf("getDB failed: %v", err)
	}

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
