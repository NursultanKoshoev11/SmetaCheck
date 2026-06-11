package api

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func analyzeBatchResilient(
	ctx context.Context,
	batchID string,
	provider EstimateAIProvider,
	inputs []AIProviderInput,
) (AIProviderBatchResponse, error) {
	result := AIProviderBatchResponse{
		Provider:      provider.Name(),
		Model:         provider.Model(),
		PromptVersion: aiPromptVersion,
		Files:         make([]AIFileAnalysis, 0, len(inputs)),
		GeneratedAt:   time.Now().UTC(),
	}

	for _, input := range inputs {
		if err := ctx.Err(); err != nil {
			return AIProviderBatchResponse{}, err
		}
		input := input
		partial, err := runWithBatchHeartbeat(ctx, batchID, func(callCtx context.Context) (AIProviderBatchResponse, error) {
			return provider.AnalyzeBatch(callCtx, []AIProviderInput{input})
		})
		if err != nil || len(partial.Files) == 0 {
			fallback := rulesFileAnalysis(input)
			fallback.Warning = fmt.Sprintf(
				"AI-провайдер %s не завершил анализ этого файла; использован backend-анализ. Причина: %s",
				provider.Name(), safeProviderError(err),
			)
			fallback.InputMode = "backend_fallback"
			result.Files = append(result.Files, fallback)
			continue
		}

		analysis := partial.Files[0]
		analysis.EstimateID = input.Estimate.ID
		analysis.FileName = input.Estimate.FileName
		if strings.TrimSpace(analysis.InputMode) == "" {
			analysis.InputMode = "normalized_chunks"
		}
		result.Files = append(result.Files, analysis)
	}

	result.BatchSummary = providerBatchSummary(result.Files)
	result.Recommendations = providerBatchRecommendations(result.Files)
	return result, nil
}

func safeProviderError(err error) string {
	if err == nil {
		return "пустой ответ"
	}
	message := strings.Join(strings.Fields(err.Error()), " ")
	if len(message) > 300 {
		message = message[:300]
	}
	return message
}
