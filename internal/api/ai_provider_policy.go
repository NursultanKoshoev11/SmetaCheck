package api

import (
	"fmt"
	"os"
	"strings"
)

func validateRequestedAIModel(provider, requestedModel, resolvedModel string) error {
	if strings.TrimSpace(requestedModel) == "" {
		return nil
	}
	if !requestProviderOverrideAllowed() {
		return fmt.Errorf("AI provider override is disabled")
	}
	allowed := allowedModelsForProvider(provider)
	if len(allowed) == 0 {
		return fmt.Errorf("model override is not configured for provider %s", provider)
	}
	if _, ok := allowed[strings.ToLower(strings.TrimSpace(resolvedModel))]; !ok {
		return fmt.Errorf("model %q is not allowed for provider %s", resolvedModel, provider)
	}
	return nil
}

func allowedModelsForProvider(provider string) map[string]struct{} {
	provider = strings.ToLower(strings.TrimSpace(provider))
	var key, fallback string
	switch provider {
	case "rules":
		return map[string]struct{}{"deterministic-v2": struct{}{}}
	case "openai":
		key = "AI_ALLOWED_OPENAI_MODELS"
		fallback = envString("OPENAI_MODEL", "gpt-4.1-mini")
	case "gemini":
		key = "AI_ALLOWED_GEMINI_MODELS"
		fallback = envString("GEMINI_MODEL", "gemini-2.5-flash")
	case "anthropic", "claude":
		key = "AI_ALLOWED_ANTHROPIC_MODELS"
		fallback = envString("ANTHROPIC_MODEL", "claude-sonnet-4-5")
	default:
		return map[string]struct{}{}
	}

	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		value = fallback
	}
	result := make(map[string]struct{})
	for _, item := range strings.Split(value, ",") {
		model := strings.ToLower(strings.TrimSpace(item))
		if model != "" && safeModelName.MatchString(model) {
			result[model] = struct{}{}
		}
	}
	return result
}
