package api

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
)

const defaultAIRowsPerChunk = 250

type aiChunkInput struct {
	EstimateID string         `json:"estimate_id"`
	FileName   string         `json:"file_name"`
	ChunkIndex int            `json:"chunk_index"`
	ChunkCount int            `json:"chunk_count"`
	Rows       []EstimateItem `json:"rows"`
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
		"estimate_id": estimate.ID,
		"file_name":   estimate.FileName,
		"items_count": estimate.ItemsCount,
		"rows":        estimate.Items,
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
			EstimateID: estimate.ID,
			FileName:   estimate.FileName,
			ChunkIndex: 1,
			ChunkCount: 1,
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
			EstimateID: estimate.ID,
			FileName:   estimate.FileName,
			ChunkIndex: index,
			ChunkCount: count,
			Rows:       estimate.Items[start:end],
		})
	}
	return chunks
}

func mergeAIFileAnalyses(estimate Estimate, chunks []AIFileAnalysis, inputMode string) AIFileAnalysis {
	result := AIFileAnalysis{
		EstimateID:       estimate.ID,
		FileName:         estimate.FileName,
		RiskLevel:        "Низкий",
		DataQualityScore: 100,
		Findings:         make([]AIProviderFinding, 0),
		Recommendations: make([]string, 0),
		InputMode:        inputMode,
	}
	seenFindings := map[string]struct{}{}
	seenRecommendations := map[string]struct{}{}
	summaries := make([]string, 0, len(chunks))
	qualityWeightedTotal := 0
	qualityWeight := 0
	aiTotal := 0.0
	rowsAnalyzed := 0
	missingTotals := 0

	for _, chunk := range chunks {
		result.RiskLevel = higherRiskLevel(result.RiskLevel, chunk.RiskLevel)
		rowWeight := chunk.RowsAnalyzed
		if rowWeight < 1 {
			rowWeight = 1
		}
		if chunk.DataQualityScore >= 0 && chunk.DataQualityScore <= 100 {
			qualityWeightedTotal += chunk.DataQualityScore * rowWeight
			qualityWeight += rowWeight
		}
		if strings.TrimSpace(chunk.Summary) != "" {
			summaries = append(summaries, strings.TrimSpace(chunk.Summary))
		}
		if chunk.RowsAnalyzed > 0 {
			rowsAnalyzed += chunk.RowsAnalyzed
		}
		if chunk.ExtractedTotal > 0 || chunk.RowsAnalyzed == 0 {
			aiTotal += chunk.ExtractedTotal
		} else {
			missingTotals++
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

	if qualityWeight > 0 {
		result.DataQualityScore = int(math.Round(float64(qualityWeightedTotal) / float64(qualityWeight)))
	}
	if rowsAnalyzed > estimate.ItemsCount {
		rowsAnalyzed = estimate.ItemsCount
	}
	result.RowsAnalyzed = rowsAnalyzed
	result.ExtractedTotal = aiTotal
	if missingTotals > 0 {
		result.Warning = appendWarning(result.Warning, fmt.Sprintf("AI не вернул независимую сумму для %d частей файла.", missingTotals))
	}
	if rowsAnalyzed < estimate.ItemsCount {
		result.Warning = appendWarning(result.Warning, fmt.Sprintf(
			"AI подтвердил анализ %d из %d распознанных строк.", rowsAnalyzed, estimate.ItemsCount,
		))
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

func appendWarning(current, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return strings.TrimSpace(current)
	}
	current = strings.TrimSpace(current)
	if current == "" {
		return value
	}
	return current + " " + value
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
