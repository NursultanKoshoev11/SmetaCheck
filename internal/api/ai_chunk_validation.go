package api

import "fmt"

func validateAIChunkResult(result AIFileAnalysis, chunk aiChunkInput) AIFileAnalysis {
	expectedRows := len(chunk.Rows)
	if result.RowsAnalyzed < 0 {
		result.RowsAnalyzed = 0
	}
	if result.RowsAnalyzed > expectedRows {
		result.RowsAnalyzed = expectedRows
	}
	if result.RowsAnalyzed < expectedRows {
		result.Warning = appendWarning(result.Warning, fmt.Sprintf(
			"AI подтвердил анализ %d из %d строк части %d/%d.",
			result.RowsAnalyzed, expectedRows, chunk.ChunkIndex, chunk.ChunkCount,
		))
	}
	if result.ExtractedTotal < 0 {
		result.ExtractedTotal = 0
	}
	return result
}
