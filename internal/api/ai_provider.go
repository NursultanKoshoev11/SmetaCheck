package api

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type AIProviderInput struct {
	Estimate         Estimate
	MIMEType         string
	NormalizedText   string
	NormalizedRows   int
	ContentTruncated bool
}

type EstimateAIProvider interface {
	Name() string
	Model() string
	AnalyzeBatch(ctx context.Context, inputs []AIProviderInput) (AIProviderBatchResponse, error)
}

var safeModelName = regexp.MustCompile(`^[A-Za-z0-9._:/-]{1,160}$`)

func configuredAIProvider(requestedProvider, requestedModel string) (EstimateAIProvider, error) {
	providerName := strings.ToLower(strings.TrimSpace(requestedProvider))
	if providerName == "" {
		providerName = strings.ToLower(strings.TrimSpace(os.Getenv("AI_PROVIDER")))
	}
	if providerName == "" {
		providerName = "rules"
	}

	model := strings.TrimSpace(requestedModel)
	if model != "" && !safeModelName.MatchString(model) {
		return nil, fmt.Errorf("invalid AI model name")
	}

	switch providerName {
	case "rules":
		return rulesAIProvider{}, nil
	case "openai":
		if strings.TrimSpace(os.Getenv("OPENAI_API_KEY")) == "" {
			return nil, fmt.Errorf("OPENAI_API_KEY is not configured")
		}
		if model == "" {
			model = envString("OPENAI_MODEL", "gpt-5.5")
		}
		return &openAIProvider{model: model}, nil
	case "gemini":
		if strings.TrimSpace(os.Getenv("GEMINI_API_KEY")) == "" {
			return nil, fmt.Errorf("GEMINI_API_KEY is not configured")
		}
		if model == "" {
			model = envString("GEMINI_MODEL", "gemini-3.5-flash")
		}
		return &geminiAIProvider{model: model}, nil
	case "anthropic", "claude":
		if strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY")) == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY is not configured")
		}
		if model == "" {
			model = envString("ANTHROPIC_MODEL", "claude-sonnet-4-6")
		}
		return &anthropicAIProvider{model: model}, nil
	default:
		return nil, fmt.Errorf("unsupported AI provider %q", providerName)
	}
}

func envString(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func requestProviderOverrideAllowed() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("AI_ALLOW_REQUEST_PROVIDER_OVERRIDE")), "true")
}

func resolveRequestedProvider(provider, model string) (string, string) {
	if !requestProviderOverrideAllowed() {
		return "", ""
	}
	return strings.TrimSpace(provider), strings.TrimSpace(model)
}

type rulesAIProvider struct{}

func (rulesAIProvider) Name() string  { return "rules" }
func (rulesAIProvider) Model() string { return "deterministic-v2" }

func (rulesAIProvider) AnalyzeBatch(_ context.Context, inputs []AIProviderInput) (AIProviderBatchResponse, error) {
	result := AIProviderBatchResponse{
		Provider:      "rules",
		Model:         "deterministic-v2",
		PromptVersion: aiPromptVersion,
		Files:         make([]AISummaryResponse, 0, len(inputs)),
	}
	for _, input := range inputs {
		report := buildAISummary(input.Estimate)
		report.InputMode = "backend_rules"
		result.Files = append(result.Files, report)
	}
	result.BatchSummary = buildRuleBatchSummary(inputs)
	result.Recommendations = []string{"Исправьте замечания высокого и среднего риска и повторите проверку."}
	return result, nil
}

func buildRuleBatchSummary(inputs []AIProviderInput) string {
	if len(inputs) == 0 {
		return "Файлы для анализа не переданы."
	}
	var total float64
	rows := 0
	for _, input := range inputs {
		total += input.Estimate.TotalAmount
		rows += input.Estimate.ItemsCount
	}
	return fmt.Sprintf("Проверено файлов: %d, распознано строк: %d, общая сумма по backend-расчётам: %.2f сом.", len(inputs), rows, total)
}
