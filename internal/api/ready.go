package api

import (
	"context"
	"net/http"
	"time"
)

func Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	pool, err := getDB(ctx)
	if err != nil || pool == nil {
		estimateWriteError(w, http.StatusServiceUnavailable, "postgresql is not ready")
		return
	}
	if err := pool.Ping(ctx); err != nil {
		estimateWriteError(w, http.StatusServiceUnavailable, "postgresql ping failed")
		return
	}
	if err := databaseSchemaReady(ctx); err != nil {
		estimateWriteError(w, http.StatusServiceUnavailable, "database schema is not ready")
		return
	}
	estimateWriteJSON(w, http.StatusOK, map[string]any{
		"ok":             true,
		"service":        "api",
		"ready":          true,
		"database":       "postgresql",
		"schema_version": currentSchemaVersion,
	})
}
