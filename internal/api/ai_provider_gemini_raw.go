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

type geminiRawProvider struct{ model string }

func (p *geminiRawProvider) Name() string  { return "gemini" }
func (p *geminiRawProvider) Model() string { return p.model }

func (p *geminiRawProvider) AnalyzeBatch(ctx context.Context, inputs []AIProviderInput) (AIProviderBatchResponse, error) {
	result := AIProviderBatchResponse{
		Provider:      p.Name(),
		Model:         p.Model(),
		PromptVersion: aiPromptVersion,
		Files:         make([]AIFileAnalysis, 0, len(inputs)),
		GeneratedAt:   time.Now().UTC(),
	}
	delegate := &geminiAIProvider{model: p.model}
	for _, input := range inputs {
		if !shouldUseRawDocument(input) {
			partial, err := delegate.AnalyzeBatch(ctx, []AIProviderInput{input})
			if err != nil || len(partial.Files) == 0 {
				if err == nil {
					err = fmt.Errorf("gemini returned no file analysis")
				}
				return AIProviderBatchResponse{}, err
			}
			result.Files = append(result.Files, partial.Files[0])
			continue
		}
		analysis, err := p.analyzeRawPDF(ctx, input)
		if err != nil {
			return AIProviderBatchResponse{}, fmt.Errorf("gemini raw PDF %s: %w", input.Estimate.FileName, err)
		}
		result.Files = append(result.Files, analysis)
	}
	result.BatchSummary = providerBatchSummary(result.Files)
	result.Recommendations = providerBatchRecommendations(result.Files)
	return result, nil
}

func (p *geminiRawProvider) analyzeRawPDF(ctx context.Context, input AIProviderInput) (AIFileAnalysis, error) {
	encoded, err := readRawDocumentBase64(input)
	if err != nil {
		return AIFileAnalysis{}, err
	}
	payload := map[string]any{
		"contents": []map[string]any{{
			"role": "user",
			"parts": []map[string]any{
				{
					"inline_data": map[string]string{
						"mime_type": "application/pdf",
						"data":      encoded,
					},
				},
				{"text": rawDocumentPrompt(input)},
			},
		}},
		"generationConfig": map[string]any{
			"responseMimeType":   "application/json",
			"responseJsonSchema": aiFileAnalysisJSONSchema(),
			"maxOutputTokens":    int(envInt64("AI_MAX_OUTPUT_TOKENS", 3000)),
			"temperature":        0.1,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return AIFileAnalysis{}, err
	}
	base := strings.TrimRight(envString("GEMINI_BASE_URL", "https://generativelanguage.googleapis.com/v1beta"), "/")
	endpoint := base + "/models/" + url.PathEscape(p.model) + ":generateContent"
	responseBody, _, err := doAIJSONRequest(ctx, http.MethodPost, endpoint, map[string]string{
		"x-goog-api-key": strings.TrimSpace(os.Getenv("GEMINI_API_KEY")),
	}, body)
	if err != nil {
		return AIFileAnalysis{}, err
	}
	var response struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return AIFileAnalysis{}, fmt.Errorf("decode gemini response: %w", err)
	}
	if len(response.Candidates) == 0 || len(response.Candidates[0].Content.Parts) == 0 {
		return AIFileAnalysis{}, fmt.Errorf("gemini returned no structured output")
	}
	text := strings.TrimSpace(response.Candidates[0].Content.Parts[0].Text)
	return decodeAIFileAnalysis([]byte(text), input.Estimate, "raw_pdf")
}
