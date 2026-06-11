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

type anthropicAIProvider struct{ model string }

func (p *anthropicAIProvider) Name() string  { return "anthropic" }
func (p *anthropicAIProvider) Model() string { return p.model }

func (p *anthropicAIProvider) AnalyzeBatch(ctx context.Context, inputs []AIProviderInput) (AIProviderBatchResponse, error) {
	result := AIProviderBatchResponse{Provider: p.Name(), Model: p.Model(), PromptVersion: aiPromptVersion, Files: make([]AIFileAnalysis, 0, len(inputs)), GeneratedAt: time.Now().UTC()}
	for _, input := range inputs {
		chunks := splitEstimateForAI(input.Estimate)
		parts := make([]AIFileAnalysis, 0, len(chunks))
		for _, chunk := range chunks {
			analysis, err := p.analyzeChunk(ctx, input.Estimate, chunk)
			if err != nil {
				return AIProviderBatchResponse{}, fmt.Errorf("anthropic %s chunk %d/%d: %w", input.Estimate.FileName, chunk.ChunkIndex, chunk.ChunkCount, err)
			}
			parts = append(parts, analysis)
		}
		result.Files = append(result.Files, mergeAIFileAnalyses(input.Estimate, parts, "normalized_chunks"))
	}
	result.BatchSummary = providerBatchSummary(result.Files)
	result.Recommendations = providerBatchRecommendations(result.Files)
	return result, nil
}

func (p *anthropicAIProvider) analyzeChunk(ctx context.Context, estimate Estimate, chunk aiChunkInput) (AIFileAnalysis, error) {
	chunkJSON, err := json.Marshal(chunk)
	if err != nil {
		return AIFileAnalysis{}, err
	}
	payload := map[string]any{
		"model":      p.model,
		"max_tokens": int(envInt64("AI_MAX_OUTPUT_TOKENS", 3000)),
		"temperature": 0,
		"system": aiAuditInstructions(),
		"messages": []map[string]any{{
			"role": "user",
			"content": []map[string]string{{"type": "text", "text": "Проанализируй chunk сметы и вызови инструмент submit_estimate_analysis. Данные:\n" + string(chunkJSON)}},
		}},
		"tools": []map[string]any{{
			"name":         "submit_estimate_analysis",
			"description":  "Return the complete structured estimate audit.",
			"input_schema": aiFileAnalysisJSONSchema(),
		}},
		"tool_choice": map[string]string{"type": "tool", "name": "submit_estimate_analysis"},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return AIFileAnalysis{}, err
	}
	endpoint := strings.TrimRight(envString("ANTHROPIC_BASE_URL", "https://api.anthropic.com/v1"), "/") + "/messages"
	responseBody, _, err := doAIJSONRequest(ctx, http.MethodPost, endpoint, map[string]string{
		"x-api-key":         strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY")),
		"anthropic-version": envString("ANTHROPIC_VERSION", "2023-06-01"),
	}, body)
	if err != nil {
		return AIFileAnalysis{}, err
	}
	var response struct {
		Content []struct {
			Type  string          `json:"type"`
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return AIFileAnalysis{}, fmt.Errorf("decode anthropic response: %w", err)
	}
	for _, content := range response.Content {
		if content.Type == "tool_use" && content.Name == "submit_estimate_analysis" && len(content.Input) > 0 {
			return decodeAIFileAnalysis(content.Input, estimate, "normalized_chunk")
		}
	}
	return AIFileAnalysis{}, fmt.Errorf("anthropic returned no submit_estimate_analysis tool call")
}
