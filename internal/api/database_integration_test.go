//go:build integration

package api

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

func TestProductionDatabaseSchema(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	t.Setenv("DATABASE_URL", databaseURL)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := migrateDatabase(ctx); err != nil {
		t.Fatalf("initial database migration failed: %v", err)
	}

	pool, err := getDB(ctx)
	if err != nil {
		t.Fatalf("getDB failed: %v", err)
	}
	if pool == nil {
		t.Fatal("getDB returned a nil pool")
	}

	expectedTables := []string{
		"users",
		"auth_identities",
		"auth_sessions",
		"auth_tokens",
		"oauth_states",
		"auth_rate_limits",
		"projects",
		"estimates",
		"analysis_batches",
		"analysis_batch_files",
		"audit_logs",
		"schema_migrations",
	}
	for _, table := range expectedTables {
		var exists bool
		if err := pool.QueryRow(ctx, `SELECT to_regclass($1) IS NOT NULL`, "public."+table).Scan(&exists); err != nil {
			t.Fatalf("check table %s: %v", table, err)
		}
		if !exists {
			t.Fatalf("required table %s was not created", table)
		}
	}

	if err := databaseSchemaReady(ctx); err != nil {
		t.Fatalf("database schema readiness failed: %v", err)
	}

	var migrationRecorded bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)`, currentSchemaVersion).Scan(&migrationRecorded); err != nil {
		t.Fatalf("read schema migration version: %v", err)
	}
	if !migrationRecorded {
		t.Fatalf("schema version %d was not recorded", currentSchemaVersion)
	}

	stateHash := fmt.Sprintf("integration-state-%d", time.Now().UnixNano())
	if _, err := pool.Exec(ctx, `
		INSERT INTO oauth_states (state_hash,provider,nonce,code_verifier,return_to,expires_at)
		VALUES ($1,'google','nonce','verifier','/dashboard',now()+interval '5 minutes')
	`, stateHash); err != nil {
		t.Fatalf("oauth_states is not writable: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM oauth_states WHERE state_hash=$1`, stateHash); err != nil {
		t.Fatalf("oauth_states cleanup failed: %v", err)
	}

	rateKey := fmt.Sprintf("integration-rate-%d", time.Now().UnixNano())
	if _, err := pool.Exec(ctx, `
		INSERT INTO auth_rate_limits (key_hash,action,window_started_at,request_count)
		VALUES ($1,'integration',date_trunc('minute',now()),1)
	`, rateKey); err != nil {
		t.Fatalf("auth_rate_limits is not writable: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM auth_rate_limits WHERE key_hash=$1`, rateKey); err != nil {
		t.Fatalf("auth_rate_limits cleanup failed: %v", err)
	}

	var waitGroup sync.WaitGroup
	errorChannel := make(chan error, 2)
	for index := 0; index < 2; index++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			migrationCtx, migrationCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer migrationCancel()
			errorChannel <- migrateDatabase(migrationCtx)
		}()
	}
	waitGroup.Wait()
	close(errorChannel)
	for migrationErr := range errorChannel {
		if migrationErr != nil {
			t.Fatalf("concurrent migration failed: %v", migrationErr)
		}
	}
}
