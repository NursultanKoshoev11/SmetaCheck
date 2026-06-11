package api

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func prepareLegacySchema(ctx context.Context, pool *pgxpool.Pool) error {
	var dataType string
	err := pool.QueryRow(ctx, `
		SELECT data_type
		FROM information_schema.columns
		WHERE table_schema = 'public'
		  AND table_name = 'users'
		  AND column_name = 'id'
	`).Scan(&dataType)
	if err == pgx.ErrNoRows {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect users.id type: %w", err)
	}
	if !strings.EqualFold(dataType, "uuid") {
		return nil
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production") {
		return fmt.Errorf("legacy UUID schema detected; production migration requires manual review")
	}
	_, err = pool.Exec(ctx, `
		DROP TABLE IF EXISTS audit_logs CASCADE;
		DROP TABLE IF EXISTS compare_results CASCADE;
		DROP TABLE IF EXISTS findings CASCADE;
		DROP TABLE IF EXISTS estimate_items CASCADE;
		DROP TABLE IF EXISTS estimates CASCADE;
		DROP TABLE IF EXISTS projects CASCADE;
		DROP TABLE IF EXISTS users CASCADE;
	`)
	if err != nil {
		return fmt.Errorf("reset legacy development schema: %w", err)
	}
	return nil
}
