package api

import (
	"net/http"
	"os"
	"sort"
	"strings"
)

type AIProviderOption struct {
	Name      string   `json:"name"`
	Models    []string `json:"models"`
	Available bool     `json:"available"`
	RawPDF    bool     `json:"raw_pdf"`
}

func AIProviders(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAuthenticatedUser(w, r); !ok {
		return
	}

	options := []AIProviderOption{
		{Name: "rules", Models: []string{"deterministic-v2"}, Available: true, RawPDF: false},
		providerOptionFromFactory("openai", true),
		providerOptionFromFactory("gemini", true),
		providerOptionFromFactory("anthropic", false),
	}

	defaultProvider := strings.ToLower(strings.TrimSpace(os.Getenv("AI_PROVIDER")))
	if defaultProvider == "" {
		defaultProvider = "rules"
	}
	estimateWriteJSON(w, http.StatusOK, map[string]any{
		"default_provider": defaultProvider,
		"override_allowed": requestProviderOverrideAllowed(),
		"providers":        options,
	})
}

func providerOptionFromFactory(name string, rawPDF bool) AIProviderOption {
	modelsMap := allowedModelsForProvider(name)
	models := make([]string, 0, len(modelsMap))
	for model := range modelsMap {
		models = append(models, model)
	}
	sort.Strings(models)
	_, err := configuredAIProvider(name, "")
	return AIProviderOption{
		Name:      name,
		Models:    models,
		Available: err == nil,
		RawPDF:    rawPDF,
	}
}
