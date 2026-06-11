package api

import "context"

func PrepareDatabase(ctx context.Context) error {
	return migrateDatabase(ctx)
}
