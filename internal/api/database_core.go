package api

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var embeddedSchema string

//go:embed oauth_schema.sql
var embeddedOAuthSchema string

//go:embed auth_security_schema.sql
var embeddedAuthSecuritySchema string

var (
	dbPool *pgxpool.Pool
	dbOnce sync.Once
	dbErr error
)

func databaseEnabled() bool {
	return strings.TrimSpace(os.Getenv("DATABASE_URL")) != ""
}

func getDB(ctx context.Context) (*pgxpool.Pool, error) {
	dbOnce.Do(func() {
		dsn := strings.TrimSpace(os.Getenv("DATABASE_URL"))
		if dsn == "" {
			dbErr = fmt.Errorf("DATABASE_URL is required")
			return
		}
		cfg, err := pgxpool.ParseConfig(dsn)
		if err != nil {
			dbErr = err
			return
		}
		cfg.MaxConns = 10
		cfg.MinConns = 1
		cfg.MaxConnLifetime = time.Hour
		pool, err := pgxpool.NewWithConfig(ctx, cfg)
		if err != nil {
			dbErr = err
			return
		}
		if err := pool.Ping(ctx); err != nil {
			pool.Close()
			dbErr = err
			return
		}
		dbPool = pool
	})
	return dbPool, dbErr
}

func migrateDatabase(ctx context.Context) error {
	pool, err := getDB(ctx)
	if err != nil {
		return err
	}
	if pool == nil {
		return fmt.Errorf("postgresql is unavailable")
	}
	if err := prepareLegacySchema(ctx, pool); err != nil {
		return err
	}
	if strings.TrimSpace(embeddedSchema) == "" || strings.TrimSpace(embeddedOAuthSchema) == "" || strings.TrimSpace(embeddedAuthSecuritySchema) == "" {
		return fmt.Errorf("embedded database schema is incomplete")
	}
	for _, migration := range []string{embeddedSchema, embeddedOAuthSchema, embeddedAuthSecuritySchema} {
		if _, err := pool.Exec(ctx, migration); err != nil {
			return err
		}
	}
	return nil
}

func initDatabaseForRun() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := migrateDatabase(ctx); err != nil {
		log.Fatalf("database migration failed: %v", err)
	}
}
