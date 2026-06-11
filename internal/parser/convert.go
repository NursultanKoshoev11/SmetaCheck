package parser

import "github.com/NursultanKoshoev11/SmetaCheck/internal/domain"

func ToEstimateRows(src [][]string) []domain.EstimateRow {
	out := []domain.EstimateRow{}
	for i, row := range src {
		if len(row) < 5 {
			continue
		}
		out = append(out, domain.EstimateRow{Number: i + 1, Name: row[0], Unit: row[1]})
	}
	return out
}
