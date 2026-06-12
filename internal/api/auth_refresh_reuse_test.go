package api

import (
	"testing"
	"time"
)

func TestRefreshReuseGracePeriodBounds(t *testing.T) {
	t.Setenv("REFRESH_REUSE_GRACE", "5s")
	if got := refreshReuseGracePeriod(); got != 5*time.Second {
		t.Fatalf("unexpected refresh reuse grace: %s", got)
	}
	t.Setenv("REFRESH_REUSE_GRACE", "2m")
	if got := refreshReuseGracePeriod(); got != 30*time.Second {
		t.Fatalf("refresh reuse grace must be capped at 30 seconds, got %s", got)
	}
	t.Setenv("REFRESH_REUSE_GRACE", "-1s")
	if got := refreshReuseGracePeriod(); got != 0 {
		t.Fatalf("negative refresh reuse grace must disable the grace period, got %s", got)
	}
}

func TestConstantTimeStringEqual(t *testing.T) {
	if !constantTimeStringEqual("same-client", "same-client") {
		t.Fatal("equal strings must compare equal")
	}
	if constantTimeStringEqual("same-client", "other-client") {
		t.Fatal("different strings must not compare equal")
	}
}
