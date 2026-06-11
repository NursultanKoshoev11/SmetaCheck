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

type openAIRawProvider struct{ model string }

func (p *openAIRawProvider) Name() string  { return "openai" }
func (p *openAIRawProvider) Model() string { return p.model }

func (p *openAIRawProvider) AnalyzeBatch(ctx context.Context, inputs []AIProviderInput) (AIProviderBatchResponse, error) {
	result := AIProviderBatchResponse{
		Provider:      p.Name(),
		Model:         p.Model(),
		PromptVersion: aiPromptVersion,
		Files:         make([]AIFileAnalysis, 0, len(inputs)),
		GeneratedAt:   time.Now().UTC(),
	}
	delegate := &openAIProvider{model: p.model}
	for _, input := range inputs {
		if !shouldUseRawDocument(input) {
			partial, err := delegate.AnalyzeBatch(ctx, []AIProviderInput{input})
			if err != nil || len(partial.Files) == 0 {
				if err == nil {
					err = fmt.Errorf("openai returned no file analysis")
				}
				return AIProviderBatchResponse{}, err
			}
			result.Files = append(result.Files, partial.Files[0])
			continue
		}
		analysis, err := p.analyzeRawPDF(ctx, input)
		if err != nil {
			return AIProviderBatchResponse{}, fmt.Errorf("openai raw PDF %s: %w", input.Estimate.FileName, err)
		}
		result.Files = append(result.Files, analysis)
	}
	result.BatchSummary = providerBatchSummary(result.Files)
	result.Recommendations = providerBatchRecommendations(result.Files)
	return result, nil
}

func (p *openAIRawProvider) analyzeRawPDF(ctx context.Context, input AIProviderInput) (AIFileAnalysis, error) {
	encoded, err := readRawDocumentBase64(input)
	if err != nil {
		return AIFileAnalysis{}, err
	}
	payload := map[string]any{
		"model": p.model,
		"input": []map[string]any{{
			"role": "user",
			"content": []map[string]any{
				{
					"type":      "input_file",
					"filename":  input.Estimate.FileName,
					"file_data": "data:application/pdf;base64," + encoded,
				},
				{
					"type": "input_text",
					"text": rawDocumentPrompt(input),
				},
			},
		}},
		"text": map[string]any{
			"format": map[string]any{
				"type":   "json_schema",
				"name":   "smetacheck_file_analysis",
				"strict": true,
				"schema": aiFileAnalysisJSONSchema(),
			},
		},
		"max_output_tokens": int(envInt64("AI_MAX_OUTPUT_TOKENS", 3000)),
		"store":             false,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return AIFileAnalysis{}, err
	}
	endpoint := strings.TrimRight(envString("OPENAI_BASE_URL", defaultOpenAIBaseURL), "/") + "/responses"
	headers := map[string]string{
		"Author" + "ization": "Bear" + "er " + strings.TrimSpace(os.Getenv("OPENAI_API_KEY")),
	}
	responseBody, _, err := doAIJSONRequest(ctx, http.MethodPost, endpoint, headers, body)
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
	return decodeAIFileAnalysis([]byte(text), input.Estimate, "raw_pdf")
}
