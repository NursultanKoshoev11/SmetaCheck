package api

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"
)

func shouldUseRawDocument(input AIProviderInput) bool {
	return strings.EqualFold(strings.TrimSpace(input.MIMEType), "application/pdf") && strings.TrimSpace(input.Estimate.FilePath) != ""
}

func readRawDocumentBase64(input AIProviderInput) (string, error) {
	maxBytes := envInt64("AI_MAX_RAW_FILE_MB", 20) * 1024 * 1024
	if maxBytes < 1024*1024 {
		maxBytes = 1024 * 1024
	}
	if maxBytes > 50*1024*1024 {
		maxBytes = 50 * 1024 * 1024
	}

	file, err := os.Open(input.Estimate.FilePath)
	if err != nil {
		return "", fmt.Errorf("open raw document: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return "", fmt.Errorf("read raw document: %w", err)
	}
	if int64(len(data)) > maxBytes {
		return "", fmt.Errorf("raw document exceeds AI_MAX_RAW_FILE_MB")
	}
	if len(data) < 5 || string(data[:5]) != "%PDF-" {
		return "", fmt.Errorf("raw document is not a valid PDF")
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func rawDocumentPrompt(input AIProviderInput) string {
	return aiAuditInstructions() + " Анализируй исходный PDF как самостоятельный документ. Проверь таблицы, итоги, пустые поля, дубли, несогласованности и визуально заметные проблемы. Не используй внешние рыночные цены. Если строка или номер позиции видимы, укажи их."
}
