package api

import (
	"net/http"
	"strings"
	"time"
)

func EstimateRouterWithReportLimit(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet && strings.HasSuffix(strings.TrimRight(r.URL.Path, "/"), "/report") {
		endpointRateLimit("report", int(envInt64("RATE_LIMIT_REPORT_PER_HOUR", 20)), time.Hour, EstimateRouterWithUsage)(w, r)
		return
	}
	EstimateRouterWithUsage(w, r)
}
