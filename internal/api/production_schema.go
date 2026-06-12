package api

import _ "embed"

//go:embed production_schema.sql
var productionReadinessSchema string

func init() {
	embeddedUsageSchema += "\n" + productionReadinessSchema
}
