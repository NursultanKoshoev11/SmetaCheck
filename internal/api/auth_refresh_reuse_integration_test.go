//go:build integration

package api

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestRefreshRotationAndReuseDetection(t *testing.T) {
	ctx, pool := integrationDatabase(t)
	t.Setenv("REFRESH_REUSE_GRACE", "5s")

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	userID := "usr_refresh_" + suffix
	sessionID := "ses_refresh_" + suffix
	oldHash := hashToken("old-refresh-" + suffix)
	newHash := hashToken("new-refresh-" + suffix)
	expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)

	if _, err := pool.Exec(ctx, `INSERT INTO users (id,email,full_name) VALUES ($1,$2,'Refresh User')`, userID, "refresh-"+suffix+"@example.test"); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM users WHERE id=$1`, userID)
	})
	if _, err := pool.Exec(ctx, `
		INSERT INTO auth_sessions (id,user_id,refresh_token_hash,ip_address,user_agent,expires_at)
		VALUES ($1,$2,$3,'10.0.0.1','test-browser',$4)
	`, sessionID, userID, oldHash, expiresAt); err != nil {
		t.Fatalf("insert session: %v", err)
	}

	rotated, err := rotateRefreshToken(ctx, oldHash, newHash, expiresAt, "10.0.0.1", "test-browser")
	if err != nil {
		t.Fatalf("rotate refresh token: %v", err)
	}
	if rotated.Result != refreshRotationSuccess || rotated.SessionID != sessionID || rotated.User.ID != userID {
		t.Fatalf("unexpected rotation result: %+v", rotated)
	}

	var storedCurrent, storedPrevious string
	var rotatedAt *time.Time
	if err := pool.QueryRow(ctx, `
		SELECT refresh_token_hash,COALESCE(previous_refresh_token_hash,''),rotated_at
		FROM auth_sessions WHERE id=$1
	`, sessionID).Scan(&storedCurrent, &storedPrevious, &rotatedAt); err != nil {
		t.Fatalf("load rotated session: %v", err)
	}
	if storedCurrent != newHash || storedPrevious != oldHash || rotatedAt == nil {
		t.Fatalf("session rotation was not persisted correctly: current=%q previous=%q rotated_at=%v", storedCurrent, storedPrevious, rotatedAt)
	}
	var historyExists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM auth_refresh_token_history
			WHERE token_hash=$1 AND session_id=$2 AND user_id=$3
		)
	`, oldHash, sessionID, userID).Scan(&historyExists); err != nil {
		t.Fatalf("check refresh history: %v", err)
	}
	if !historyExists {
		t.Fatal("rotated refresh token was not saved to history")
	}

	concurrent, err := rotateRefreshToken(ctx, oldHash, hashToken("ignored-concurrent"), expiresAt, "10.0.0.1", "test-browser")
	if err != nil {
		t.Fatalf("classify concurrent refresh: %v", err)
	}
	if concurrent.Result != refreshRotationConcurrent {
		t.Fatalf("expected concurrent refresh result, got %+v", concurrent)
	}
	var revokedAt *time.Time
	if err := pool.QueryRow(ctx, `SELECT revoked_at FROM auth_sessions WHERE id=$1`, sessionID).Scan(&revokedAt); err != nil {
		t.Fatalf("load session after concurrent refresh: %v", err)
	}
	if revokedAt != nil {
		t.Fatal("legitimate concurrent refresh must not revoke the session")
	}

	reused, err := rotateRefreshToken(ctx, oldHash, hashToken("ignored-reuse"), expiresAt, "203.0.113.50", "attacker-agent")
	if err != nil {
		t.Fatalf("detect refresh token reuse: %v", err)
	}
	if reused.Result != refreshRotationReused {
		t.Fatalf("expected refresh token reuse result, got %+v", reused)
	}
	if err := pool.QueryRow(ctx, `SELECT revoked_at FROM auth_sessions WHERE id=$1`, sessionID).Scan(&revokedAt); err != nil {
		t.Fatalf("load revoked session: %v", err)
	}
	if revokedAt == nil {
		t.Fatal("reused refresh token did not revoke the compromised session")
	}
	var auditExists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM audit_logs
			WHERE user_id=$1 AND action='auth.refresh_token_reused'
			  AND resource_type='auth_session' AND resource_id=$2
		)
	`, userID, sessionID).Scan(&auditExists); err != nil {
		t.Fatalf("check refresh reuse audit event: %v", err)
	}
	if !auditExists {
		t.Fatal("refresh token reuse audit event was not recorded")
	}
}

func TestRefreshHistoryDetectsTokenOlderThanPreviousRotation(t *testing.T) {
	ctx, pool := integrationDatabase(t)
	t.Setenv("REFRESH_REUSE_GRACE", "0s")

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	userID := "usr_refresh_history_" + suffix
	sessionID := "ses_refresh_history_" + suffix
	firstHash := hashToken("first-" + suffix)
	secondHash := hashToken("second-" + suffix)
	thirdHash := hashToken("third-" + suffix)
	expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)

	if _, err := pool.Exec(ctx, `INSERT INTO users (id,full_name) VALUES ($1,'Refresh History User')`, userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM users WHERE id=$1`, userID)
	})
	if _, err := pool.Exec(ctx, `
		INSERT INTO auth_sessions (id,user_id,refresh_token_hash,expires_at)
		VALUES ($1,$2,$3,$4)
	`, sessionID, userID, firstHash, expiresAt); err != nil {
		t.Fatalf("insert session: %v", err)
	}

	if result, err := rotateRefreshToken(ctx, firstHash, secondHash, expiresAt, "10.0.0.1", "browser"); err != nil || result.Result != refreshRotationSuccess {
		t.Fatalf("first rotation failed: result=%+v err=%v", result, err)
	}
	if result, err := rotateRefreshToken(ctx, secondHash, thirdHash, expiresAt, "10.0.0.1", "browser"); err != nil || result.Result != refreshRotationSuccess {
		t.Fatalf("second rotation failed: result=%+v err=%v", result, err)
	}

	result, err := rotateRefreshToken(ctx, firstHash, hashToken("unused"), expiresAt, "198.51.100.10", "stolen-client")
	if err != nil {
		t.Fatalf("replay oldest token: %v", err)
	}
	if result.Result != refreshRotationReused {
		t.Fatalf("old token from complete history was not detected: %+v", result)
	}
}
