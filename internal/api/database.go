package api

import (
	"context"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

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
			return
		}
		config, err := pgxpool.ParseConfig(dsn)
		if err != nil {
			dbErr = err
			return
		}
		config.MaxConns = 10
		config.MinConns = 1
		config.MaxConnLifetime = time.Hour
		config.HealthCheckPeriod = time.Minute
		pool, err := pgxpool.NewWithConfig(ctx, config)
		if err != nil {
			dbErr = err
			return
		}
		if err := pool.Ping(ctx); err != nil {
			dbErr = err
			pool.Close()
			return
		}
		dbPool = pool
	})
	return dbPool, dbErr
}

func migrateDatabase(ctx context.Context) error {
	if !databaseEnabled() {
		return nil
	}
	pool, err := getDB(ctx)
	if err != nil {
		return err
	}
	if pool == nil {
		return nil
	}
	data, err := os.ReadFile("db/migrations/001_initial_schema.sql")
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, string(data))
	return err
}

func initDatabaseForRun() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := migrateDatabase(ctx); err != nil {
		log.Fatalf("database migration failed: %v", err)
	}
}
