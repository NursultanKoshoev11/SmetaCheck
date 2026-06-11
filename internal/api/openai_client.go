package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const defaultOpenAIBaseURL = "https://api.openai.com/v1"

type openAIInput struct {
	FileName      string         `json:"file_name"`
	ItemsCount    int            `json:"items_count"`
	TotalAmount   float64        `json:"total_amount"`
	RuleScore     int            `json:"rule_score"`
	Rows          []EstimateItem `json:"rows"`
	Findings      []Finding      `json:"findings"`
	RowsTruncated bool           `json:"rows_truncated"`
}

type openAIRequest struct {
	Model           string         `json:"model"`
	Instructions    string         `json:"instructions"`
	Input           any            `json:"input"`
	Text            map[string]any `json:"text"`
	MaxOutputTokens int            `json:"max_output_tokens"`
	Store           bool           `json:"store"`
}

type openAIResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error"`
	Output []struct {
		Type    string `json:"type"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
}

func requestOpenAIAnalysis(parent context.Context, estimate Estimate) (AISummaryResponse, error) {
	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if apiKey == "" {
		return AISummaryResponse{}, errors.New("OPENAI_API_KEY is not configured")
	}
	if estimate.ItemsCount == 0 {
		return AISummaryResponse{}, errors.New("estimate contains no parsed rows")
	}

	timeout := envDuration("AI_TIMEOUT", 35*time.Second)
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	maxRows := int(envInt64("AI_MAX_INPUT_ROWS", 200))
	if maxRows < 1 {
		maxRows = 1
	}
	if maxRows > 1000 {
		maxRows = 1000
	}
	rows := estimate.Items
	truncated := false
	if len(rows) > maxRows {
		rows = rows[:maxRows]
		truncated = true
	}

	input := openAIInput{
		FileName:      estimate.FileName,
		ItemsCount:    estimate.ItemsCount,
		TotalAmount:   estimate.TotalAmount,
		RuleScore:     estimate.Score,
		Rows:          rows,
		Findings:      estimate.Findings,
		RowsTruncated: truncated,
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return AISummaryResponse{}, fmt.Errorf("marshal AI input: %w", err)
	}

	requestPayload := openAIRequest{
		Model: openAIModel(),
		Instructions: "Ты эксперт по аудиту строительных смет. Анализируй только переданные нормализованные строки и findings. Не придумывай цены, объёмы, строки, нормативы или рыночные данные. Не меняй арифметические факты. Дай краткий отчёт на русском языке. Укажи конкретные строки из входных данных, когда это возможно. Если данных недостаточно, прямо скажи об этом. data_quality_score должен отражать качество переданных данных, а не уверенность модели. risk_level может быть только: Низкий, Средний, Высокий, Не определён.",
		Input: []map[string]any{
			{
				"role": "user",
				"content": []map[string]string{{
					"type": "input_text",
					"text": "Подготовь структурированный аудит этой сметы. Фактические данные:\n" + string(inputJSON),
				}},
			},
		},
		Text: map[string]any{
			"format": map[string]any{
				"type":   "json_schema",
				"name":   "smetacheck_estimate_audit",
				"strict": true,
				"schema": aiResponseJSONSchema(),
			},
		},
		MaxOutputTokens: int(envInt64("AI_MAX_OUTPUT_TOKENS", 1800)),
		Store:           false,
	}
	body, err := json.Marshal(requestPayload)
	if err != nil {
		return AISummaryResponse{}, fmt.Errorf("marshal OpenAI request: %w", err)
	}

	endpoint := strings.TrimRight(openAIBaseURL(), "/") + "/responses"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return AISummaryResponse{}, fmt.Errorf("create OpenAI request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-Request-Id", newRequestID())

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return AISummaryResponse{}, fmt.Errorf("OpenAI request failed: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return AISummaryResponse{}, fmt.Errorf("read OpenAI response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return AISummaryResponse{}, fmt.Errorf("OpenAI returned HTTP %d: %s", resp.StatusCode, compactErrorBody(responseBody))
	}

	var apiResponse openAIResponse
	if err := json.Unmarshal(responseBody, &apiResponse); err != nil {
		return AISummaryResponse{}, fmt.Errorf("decode OpenAI response: %w", err)
	}
	if apiResponse.Error != nil {
		return AISummaryResponse{}, fmt.Errorf("OpenAI error: %s", apiResponse.Error.Message)
	}
	text := extractOpenAIOutputText(apiResponse)
	if text == "" {
		return AISummaryResponse{}, fmt.Errorf("OpenAI response does not contain output_text")
	}

	var report AISummaryResponse
	if err := json.Unmarshal([]byte(text), &report); err != nil {
		return AISummaryResponse{}, fmt.Errorf("decode structured AI report: %w", err)
	}
	return report, nil
}

func aiResponseJSONSchema() map[string]any {
	stringArray := map[string]any{
		"type":     "array",
		"items":    map[string]any{"type": "string"},
		"maxItems": 10,
	}
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]any{
			"executive_brief":    map[string]any{"type": "string"},
			"risk_level":         map[string]any{"type": "string", "enum": []string{"Низкий", "Средний", "Высокий", "Не определён"}},
			"data_quality_score": map[string]any{"type": "integer", "minimum": 0, "maximum": 100},
			"key_risks":          stringArray,
			"priority_actions":   stringArray,
			"cost_flags":         stringArray,
			"questions":          stringArray,
			"recommendation":     map[string]any{"type": "string"},
		},
		"required": []string{
			"executive_brief",
			"risk_level",
			"data_quality_score",
			"key_risks",
			"priority_actions",
			"cost_flags",
			"questions",
			"recommendation",
		},
	}
}

func extractOpenAIOutputText(response openAIResponse) string {
	for _, output := range response.Output {
		if output.Type != "message" {
			continue
		}
		for _, content := range output.Content {
			if content.Type == "output_text" && strings.TrimSpace(content.Text) != "" {
				return strings.TrimSpace(content.Text)
			}
		}
	}
	return ""
}

func openAIModel() string {
	if value := strings.TrimSpace(os.Getenv("OPENAI_MODEL")); value != "" {
		return value
	}
	return "gpt-4o-mini"
}

func openAIBaseURL() string {
	if value := strings.TrimSpace(os.Getenv("OPENAI_BASE_URL")); value != "" {
		return value
	}
	return defaultOpenAIBaseURL
}

func compactErrorBody(body []byte) string {
	text := strings.Join(strings.Fields(string(body)), " ")
	if len(text) > 500 {
		text = text[:500]
	}
	return text
}
