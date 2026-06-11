package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type geminiAIProvider struct{ model string }

func (p *geminiAIProvider) Name() string  { return "gemini" }
func (p *geminiAIProvider) Model() string { return p.model }

func (p *geminiAIProvider) AnalyzeBatch(ctx context.Context, inputs []AIProviderInput) (AIProviderBatchResponse, error) {
	result := AIProviderBatchResponse{Provider: p.Name(), Model: p.Model(), PromptVersion: aiPromptVersion, Files: make([]AIFileAnalysis, 0, len(inputs)), GeneratedAt: time.Now().UTC()}
	for _, input := range inputs {
		chunks := splitEstimateForAI(input.Estimate)
		parts := make([]AIFileAnalysis, 0, len(chunks))
		for _, chunk := range chunks {
			analysis, err := p.analyzeChunk(ctx, input.Estimate, chunk)
			if err != nil {
				return AIProviderBatchResponse{}, fmt.Errorf("gemini %s chunk %d/%d: %w", input.Estimate.FileName, chunk.ChunkIndex, chunk.ChunkCount, err)
			}
			parts = append(parts, analysis)
		}
		result.Files = append(result.Files, mergeAIFileAnalyses(input.Estimate, parts, "normalized_chunks"))
	}
	result.BatchSummary = providerBatchSummary(result.Files)
	result.Recommendations = providerBatchRecommendations(result.Files)
	return result, nil
}

func (p *geminiAIProvider) analyzeChunk(ctx context.Context, estimate Estimate, chunk aiChunkInput) (AIFileAnalysis, error) {
	chunkJSON, err := json.Marshal(chunk)
	if err != nil { return AIFileAnalysis{}, err }
	payload := map[string]any{
		"contents": []map[string]any{{"role": "user", "parts": []map[string]string{{"text": aiAuditInstructions() + "\nДанные chunk:\n" + string(chunkJSON)}}}},
		"generationConfig": map[string]any{
			"responseMimeType": "application/json",
			"responseJsonSchema": aiFileAnalysisJSONSchema(),
			"maxOutputTokens": int(envInt64("AI_MAX_OUTPUT_TOKENS", 3000)),
			"temperature": 0.1,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil { return AIFileAnalysis{}, err }
	base := strings.TrimRight(envString("GEMINI_BASE_URL", "https://generativelanguage.googleapis.com/v1beta"), "/")
	endpoint := base + "/models/" + url.PathEscape(p.model) + ":generateContent"
	responseBody, _, err := doAIJSONRequest(ctx, http.MethodPost, endpoint, map[string]string{"x-goog-api-key": strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))}, body)
	if err != nil { return AIFileAnalysis{}, err }
	var response struct {
		Candidates []struct {
			Content struct { Parts []struct { Text string `json:"text"` } `json:"parts"` } `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		PromptFeedback any `json:"promptFeedback"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return AIFileAnalysis{}, fmt.Errorf("decode gemini response: %w", err)
	}
	if len(response.Candidates) == 0 || len(response.Candidates[0].Content.Parts) == 0 {
		return AIFileAnalysis{}, fmt.Errorf("gemini returned no structured output")
	}
	text := strings.TrimSpace(response.Candidates[0].Content.Parts[0].Text)
	return decodeAIFileAnalysis([]byte(text), estimate, "normalized_chunk")
}
