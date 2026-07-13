package migrations

import _ "embed"

// InitialSQL contains the idempotent database schema used at startup.
//
//go:embed 001_init.sql
var InitialSQL string
