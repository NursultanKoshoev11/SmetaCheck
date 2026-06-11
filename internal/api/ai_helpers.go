package api

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var rowNumberPattern = regexp.MustCompile(`(?i)(?:строк[аи]?|row)\s*[:№#-]?\s*(\d+)`)

func extractRowNumber(text string) int {
	match := rowNumberPattern.FindStringSubmatch(text)
	if len(match) != 2 {
		return 0
	}
	value, err := strconv.Atoi(match[1])
	if err != nil || value < 0 {
		return 0
	}
	return value
}

func decodeAIFileAnalysis(data []byte, estimate Estimate, inputMode string) (AIFileAnalysis, error) {
	var report AIFileAnalysis
	if err := json.Unmarshal(data, &report); err != nil {
		return AIFileAnalysis{}, fmt.Errorf("decode AI file analysis: %w", err)
	}
	report.EstimateID = estimate.ID
	report.FileName = estimate.FileName
	report.InputMode = inputMode
	report.RiskLevel = normalizeRiskLevel(report.RiskLevel)
	if report.DataQualityScore < 0 {
		report.DataQualityScore = 0
	}
	if report.DataQualityScore > 100 {
		report.DataQualityScore = 100
	}
	if report.RowsAnalyzed < 0 || report.RowsAnalyzed > estimate.ItemsCount {
		report.RowsAnalyzed = estimate.ItemsCount
	}
	if report.ExtractedTotal < 0 {
		report.ExtractedTotal = 0
	}
	if report.Findings == nil {
		report.Findings = []AIProviderFinding{}
	}
	for index := range report.Findings {
		report.Findings[index].FileID = estimate.ID
		report.Findings[index].Severity = normalizeSeverity(report.Findings[index].Severity)
		if report.Findings[index].Row < 0 {
			report.Findings[index].Row = 0
		}
		if strings.TrimSpace(report.Findings[index].Category) == "" {
			report.Findings[index].Category = "other"
		}
	}
	if report.Recommendations == nil {
		report.Recommendations = []string{}
	}
	return report, nil
}

func normalizeRiskLevel(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "high", "высокий":
		return "Высокий"
	case "medium", "средний":
		return "Средний"
	case "low", "низкий":
		return "Низкий"
	default:
		return "Не определён"
	}
}

func aiFileAnalysisJSONSchema() map[string]any {
	finding := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]any{
			"file_id":          map[string]any{"type": "string"},
			"row":              map[string]any{"type": "integer", "minimum": 0},
			"category":         map[string]any{"type": "string"},
			"title":            map[string]any{"type": "string"},
			"severity":         map[string]any{"type": "string", "enum": []string{"Info", "Low", "Medium", "High"}},
			"detail":           map[string]any{"type": "string"},
			"evidence":         map[string]any{"type": "string"},
			"suggested_action": map[string]any{"type": "string"},
		},
		"required": []string{"file_id", "row", "category", "title", "severity", "detail", "evidence", "suggested_action"},
	}
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]any{
			"estimate_id":       map[string]any{"type": "string"},
			"file_name":         map[string]any{"type": "string"},
			"summary":           map[string]any{"type": "string"},
			"risk_level":        map[string]any{"type": "string", "enum": []string{"Низкий", "Средний", "Высокий", "Не определён"}},
			"data_quality_score": map[string]any{"type": "integer", "minimum": 0, "maximum": 100},
			"extracted_total":   map[string]any{"type": "number", "minimum": 0},
			"rows_analyzed":     map[string]any{"type": "integer", "minimum": 0},
			"findings":          map[string]any{"type": "array", "items": finding, "maxItems": 100},
			"recommendations":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "maxItems": 20},
			"input_mode":        map[string]any{"type": "string"},
			"warning":           map[string]any{"type": "string"},
		},
		"required": []string{"estimate_id", "file_name", "summary", "risk_level", "data_quality_score", "extracted_total", "rows_analyzed", "findings", "recommendations", "input_mode", "warning"},
	}
}

func aiAuditInstructions() string {
	return "Ты независимый аудитор строительных смет. Анализируй все переданные данные полностью. Не доверяй backend findings автоматически: проверь строки самостоятельно, затем используй backend findings только как дополнительный сигнал. Не придумывай отсутствующие цены, нормы, объёмы или рыночные данные. Денежные вычисления перепроверяй по переданным количеству, цене и сумме. Каждое замечание привязывай к file_id и row, если строка известна. Возвращай только JSON по заданной схеме на русском языке."
}
