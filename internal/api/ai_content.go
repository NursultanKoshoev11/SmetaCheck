package api

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const defaultAIRowsPerChunk = 250

type aiChunkInput struct {
	EstimateID    string         `json:"estimate_id"`
	FileName      string         `json:"file_name"`
	ChunkIndex    int            `json:"chunk_index"`
	ChunkCount    int            `json:"chunk_count"`
	BackendScore  int            `json:"backend_score"`
	BackendTotal  float64        `json:"backend_total"`
	Rows          []EstimateItem `json:"rows"`
	BackendIssues []Finding      `json:"backend_issues"`
}

func buildAIProviderInputs(estimates []Estimate) []AIProviderInput {
	result := make([]AIProviderInput, 0, len(estimates))
	for _, estimate := range estimates {
		result = append(result, AIProviderInput{
			Estimate:       estimate,
			MIMEType:       estimateMIMEType(estimate.FileName),
			NormalizedText: normalizedEstimateText(estimate),
			NormalizedRows: estimate.ItemsCount,
		})
	}
	return result
}

func normalizedEstimateText(estimate Estimate) string {
	payload := map[string]any{
		"estimate_id":    estimate.ID,
		"file_name":      estimate.FileName,
		"backend_score":  estimate.Score,
		"backend_total":  estimate.TotalAmount,
		"items_count":    estimate.ItemsCount,
		"rows":           estimate.Items,
		"backend_issues": estimate.Findings,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return string(encoded)
}

func splitEstimateForAI(estimate Estimate) []aiChunkInput {
	rowsPerChunk := int(envInt64("AI_ROWS_PER_CHUNK", defaultAIRowsPerChunk))
	if rowsPerChunk < 25 {
		rowsPerChunk = 25
	}
	if rowsPerChunk > 1000 {
		rowsPerChunk = 1000
	}
	if len(estimate.Items) == 0 {
		return []aiChunkInput{{
			EstimateID:    estimate.ID,
			FileName:      estimate.FileName,
			ChunkIndex:    1,
			ChunkCount:    1,
			BackendScore:  estimate.Score,
			BackendTotal:  estimate.TotalAmount,
			BackendIssues: estimate.Findings,
		}}
	}
	count := (len(estimate.Items) + rowsPerChunk - 1) / rowsPerChunk
	chunks := make([]aiChunkInput, 0, count)
	for start, index := 0, 1; start < len(estimate.Items); start, index = start+rowsPerChunk, index+1 {
		end := start + rowsPerChunk
		if end > len(estimate.Items) {
			end = len(estimate.Items)
		}
		chunks = append(chunks, aiChunkInput{
			EstimateID:    estimate.ID,
			FileName:      estimate.FileName,
			ChunkIndex:    index,
			ChunkCount:    count,
			BackendScore:  estimate.Score,
			BackendTotal:  estimate.TotalAmount,
			Rows:          estimate.Items[start:end],
			BackendIssues: findingsForRows(estimate.Findings, estimate.Items[start:end]),
		})
	}
	return chunks
}

func findingsForRows(findings []Finding, rows []EstimateItem) []Finding {
	if len(rows) == 0 {
		return findings
	}
	minRow, maxRow := rows[0].Row, rows[0].Row
	for _, row := range rows[1:] {
		if row.Row < minRow {
			minRow = row.Row
		}
		if row.Row > maxRow {
			maxRow = row.Row
		}
	}
	result := make([]Finding, 0)
	for _, finding := range findings {
		row := extractRowNumber(finding.Detail)
		if row == 0 || (row >= minRow && row <= maxRow) {
			result = append(result, finding)
		}
	}
	return result
}

func mergeAIFileAnalyses(estimate Estimate, chunks []AIFileAnalysis, inputMode string) AIFileAnalysis {
	result := AIFileAnalysis{
		EstimateID:       estimate.ID,
		FileName:         estimate.FileName,
		RiskLevel:        "Низкий",
		DataQualityScore: 100,
		ExtractedTotal:   estimate.TotalAmount,
		RowsAnalyzed:     estimate.ItemsCount,
		Findings:         make([]AIProviderFinding, 0),
		Recommendations: make([]string, 0),
		InputMode:        inputMode,
	}
	seenFindings := map[string]struct{}{}
	seenRecommendations := map[string]struct{}{}
	summaries := make([]string, 0, len(chunks))
	qualityTotal, qualityCount := 0, 0
	for _, chunk := range chunks {
		result.RiskLevel = higherRiskLevel(result.RiskLevel, chunk.RiskLevel)
		if chunk.DataQualityScore >= 0 && chunk.DataQualityScore <= 100 {
			qualityTotal += chunk.DataQualityScore
			qualityCount++
		}
		if strings.TrimSpace(chunk.Summary) != "" {
			summaries = append(summaries, strings.TrimSpace(chunk.Summary))
		}
		for _, finding := range chunk.Findings {
			finding.FileID = estimate.ID
			finding.Severity = normalizeSeverity(finding.Severity)
			key := normalizedAIFindingKey(finding)
			if _, exists := seenFindings[key]; exists {
				continue
			}
			seenFindings[key] = struct{}{}
			result.Findings = append(result.Findings, finding)
		}
		for _, recommendation := range chunk.Recommendations {
			key := strings.ToLower(strings.TrimSpace(recommendation))
			if key == "" {
				continue
			}
			if _, exists := seenRecommendations[key]; exists {
				continue
			}
			seenRecommendations[key] = struct{}{}
			result.Recommendations = append(result.Recommendations, strings.TrimSpace(recommendation))
		}
	}
	if qualityCount > 0 {
		result.DataQualityScore = qualityTotal / qualityCount
	}
	if len(summaries) > 0 {
		result.Summary = strings.Join(summaries, " ")
		if len(result.Summary) > 4000 {
			result.Summary = result.Summary[:4000]
		}
	}
	sort.SliceStable(result.Findings, func(i, j int) bool {
		if result.Findings[i].Row == result.Findings[j].Row {
			return severityRank(result.Findings[i].Severity) > severityRank(result.Findings[j].Severity)
		}
		return result.Findings[i].Row < result.Findings[j].Row
	})
	return result
}

func normalizedAIFindingKey(finding AIProviderFinding) string {
	return fmt.Sprintf("%s|%d|%s|%s", finding.FileID, finding.Row,
		strings.ToLower(strings.TrimSpace(finding.Category)),
		strings.ToLower(strings.TrimSpace(finding.Title)))
}

func estimateMIMEType(fileName string) string {
	lower := strings.ToLower(fileName)
	switch {
	case strings.HasSuffix(lower, ".pdf"):
		return "application/pdf"
	case strings.HasSuffix(lower, ".xlsx"):
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case strings.HasSuffix(lower, ".xlsm"):
		return "application/vnd.ms-excel.sheet.macroenabled.12"
	case strings.HasSuffix(lower, ".csv"):
		return "text/csv"
	default:
		return "application/octet-stream"
	}
}

func higherRiskLevel(a, b string) string {
	if riskLevelRank(b) > riskLevelRank(a) {
		return b
	}
	return a
}

func riskLevelRank(value string) int {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "высокий", "high":
		return 4
	case "средний", "medium":
		return 3
	case "низкий", "low":
		return 2
	case "не определён", "unknown":
		return 1
	default:
		return 0
	}
}

func severityRank(value string) int {
	switch normalizeSeverity(value) {
	case "High":
		return 4
	case "Medium":
		return 3
	case "Low":
		return 2
	default:
		return 1
	}
}
