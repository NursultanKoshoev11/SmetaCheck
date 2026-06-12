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

const (
	databaseMigrationLockKey int64 = 0x534d45544143484b
	currentSchemaVersion     int64 = 2026061201
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
	dbErr  error
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
		cfg.MaxConns = int32(envInt64("DATABASE_MAX_CONNS", 10))
		if cfg.MaxConns < 2 {
			cfg.MaxConns = 2
		}
		cfg.MinConns = int32(envInt64("DATABASE_MIN_CONNS", 1))
		if cfg.MinConns < 0 {
			cfg.MinConns = 0
		}
		if cfg.MinConns > cfg.MaxConns {
			cfg.MinConns = cfg.MaxConns
		}
		cfg.MaxConnLifetime = envDuration("DATABASE_MAX_CONN_LIFETIME", time.Hour)
		cfg.MaxConnIdleTime = envDuration("DATABASE_MAX_CONN_IDLE_TIME", 15*time.Minute)
		cfg.HealthCheckPeriod = envDuration("DATABASE_HEALTH_CHECK_PERIOD", 30*time.Second)

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

	lockConnection, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire database migration connection: %w", err)
	}
	defer lockConnection.Release()
	if _, err := lockConnection.Exec(ctx, `SELECT pg_advisory_lock($1)`, databaseMigrationLockKey); err != nil {
		return fmt.Errorf("acquire database migration lock: %w", err)
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, unlockErr := lockConnection.Exec(unlockCtx, `SELECT pg_advisory_unlock($1)`, databaseMigrationLockKey); unlockErr != nil {
			log.Printf("database migration unlock failed: %v", unlockErr)
		}
	}()

	if err := prepareLegacySchema(ctx, pool); err != nil {
		return err
	}
	if strings.TrimSpace(embeddedSchema) == "" || strings.TrimSpace(embeddedOAuthSchema) == "" || strings.TrimSpace(embeddedAuthSecuritySchema) == "" {
		return fmt.Errorf("embedded database schema is incomplete")
	}
	for _, migration := range []string{embeddedSchema, embeddedOAuthSchema, embeddedAuthSecuritySchema} {
		if _, err := lockConnection.Exec(ctx, migration); err != nil {
			return fmt.Errorf("apply embedded database migration: %w", err)
		}
	}
	if _, err := lockConnection.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version BIGINT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`); err != nil {
		return fmt.Errorf("create schema migration metadata: %w", err)
	}
	if _, err := lockConnection.Exec(ctx, `
		INSERT INTO schema_migrations (version)
		VALUES ($1)
		ON CONFLICT (version) DO NOTHING
	`, currentSchemaVersion); err != nil {
		return fmt.Errorf("record schema migration version: %w", err)
	}
	return nil
}

func databaseSchemaReady(ctx context.Context) error {
	pool, err := getDB(ctx)
	if err != nil {
		return err
	}
	if pool == nil {
		return fmt.Errorf("postgresql is unavailable")
	}
	var ready bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM schema_migrations WHERE version = $1
		)
	`, currentSchemaVersion).Scan(&ready); err != nil {
		return err
	}
	if !ready {
		return fmt.Errorf("database schema version %d is not applied", currentSchemaVersion)
	}
	return nil
}

func closeDatabase() {
	if dbPool != nil {
		dbPool.Close()
	}
}

func initDatabaseForRun() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := migrateDatabase(ctx); err != nil {
		log.Fatalf("database migration failed: %v", err)
	}
}
