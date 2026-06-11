package api

import (
	"context"
	"fmt"
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
	if strings.EqualFold(dataType, "uuid") {
		return fmt.Errorf("legacy UUID schema detected; apply a reviewed migration before starting this version")
	}
	return nil
}
