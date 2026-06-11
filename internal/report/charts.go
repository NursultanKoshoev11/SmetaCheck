package report

import "github.com/NursultanKoshoev11/SmetaCheck/internal/domain"

type ChartPoint struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

func CountBySeverity(items []domain.Issue) []ChartPoint {
	counts := map[domain.Severity]int{}
	for _, item := range items {
		counts[item.Severity]++
	}
	result := make([]ChartPoint, 0, len(counts))
	for severity, count := range counts {
		result = append(result, ChartPoint{Label: string(severity), Count: count})
	}
	return result
}
