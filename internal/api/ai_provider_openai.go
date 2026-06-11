package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type openAIProvider struct{ model string }

func (p *openAIProvider) Name() string  { return "openai" }
func (p *openAIProvider) Model() string { return p.model }

func (p *openAIProvider) AnalyzeBatch(ctx context.Context, inputs []AIProviderInput) (AIProviderBatchResponse, error) {
	result := AIProviderBatchResponse{Provider: p.Name(), Model: p.Model(), PromptVersion: aiPromptVersion, Files: make([]AIFileAnalysis, 0, len(inputs)), GeneratedAt: time.Now().UTC()}
	for _, input := range inputs {
		chunks := splitEstimateForAI(input.Estimate)
		parts := make([]AIFileAnalysis, 0, len(chunks))
		for _, chunk := range chunks {
			analysis, err := p.analyzeChunk(ctx, input.Estimate, chunk)
			if err != nil {
				return AIProviderBatchResponse{}, fmt.Errorf("openai %s chunk %d/%d: %w", input.Estimate.FileName, chunk.ChunkIndex, chunk.ChunkCount, err)
			}
			parts = append(parts, analysis)
		}
		result.Files = append(result.Files, mergeAIFileAnalyses(input.Estimate, parts, "normalized_chunks"))
	}
	result.BatchSummary = providerBatchSummary(result.Files)
	result.Recommendations = providerBatchRecommendations(result.Files)
	return result, nil
}

func (p *openAIProvider) analyzeChunk(ctx context.Context, estimate Estimate, chunk aiChunkInput) (AIFileAnalysis, error) {
	chunkJSON, err := json.Marshal(chunk)
	if err != nil {
		return AIFileAnalysis{}, err
	}
	payload := map[string]any{
		"model": p.model,
		"input": []map[string]any{{"role": "user", "content": []map[string]string{{"type": "input_text", "text": aiAuditInstructions() + "\nДанные chunk:\n" + string(chunkJSON)}}}},
		"text": map[string]any{"format": map[string]any{"type": "json_schema", "name": "smetacheck_file_analysis", "strict": true, "schema": aiFileAnalysisJSONSchema()}},
		"max_output_tokens": int(envInt64("AI_MAX_OUTPUT_TOKENS", 3000)),
		"store": false,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return AIFileAnalysis{}, err
	}
	endpoint := strings.TrimRight(envString("OPENAI_BASE_URL", defaultOpenAIBaseURL), "/") + "/responses"
	responseBody, _, err := doAIJSONRequest(ctx, http.MethodPost, endpoint, map[string]string{"Authorization": "Bearer " + strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))}, body)
	if err != nil {
		return AIFileAnalysis{}, err
	}
	var response openAIResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return AIFileAnalysis{}, fmt.Errorf("decode openai response: %w", err)
	}
	text := extractOpenAIOutputText(response)
	if text == "" {
		return AIFileAnalysis{}, fmt.Errorf("openai returned no structured output")
	}
	return decodeAIFileAnalysis([]byte(text), estimate, "normalized_chunk")
}

func providerBatchSummary(files []AIFileAnalysis) string {
	if len(files) == 0 { return "AI не вернул анализ файлов." }
	high, medium, rows := 0, 0, 0
	var total float64
	for _, file := range files {
		rows += file.RowsAnalyzed
		total += file.ExtractedTotal
		for _, finding := range file.Findings {
			switch normalizeSeverity(finding.Severity) { case "High": high++; case "Medium": medium++ }
		}
	}
	return fmt.Sprintf("AI проверил %d файлов и %d строк. High: %d, Medium: %d. AI-сумма: %.2f сом.", len(files), rows, high, medium, total)
}

func providerBatchRecommendations(files []AIFileAnalysis) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0)
	for _, file := range files {
		for _, item := range file.Recommendations {
			key := strings.ToLower(strings.TrimSpace(item))
			if key == "" { continue }
			if _, ok := seen[key]; ok { continue }
			seen[key] = struct{}{}
			result = append(result, strings.TrimSpace(item))
			if len(result) == 20 { return result }
		}
	}
	return result
}
