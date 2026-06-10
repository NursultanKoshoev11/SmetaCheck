package api

import (
	"fmt"
	"net/http"
	"strings"
)

type AISummaryResponse struct {
	EstimateID      string   `json:"estimate_id"`
	ExecutiveBrief  string   `json:"executive_brief"`
	RiskLevel       string   `json:"risk_level"`
	KeyRisks        []string `json:"key_risks"`
	Questions       []string `json:"questions"`
	Recommendation  string   `json:"recommendation"`
}

func EstimateAISummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		estimateWriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v1/ai/estimate-summary/")
	id = strings.Trim(id, "/")
	if id == "" {
		estimateWriteError(w, http.StatusBadRequest, "estimate id is required")
		return
	}
	estimate, ok, err := findEstimate(id)
	if err != nil {
		estimateWriteError(w, http.StatusInternalServerError, "cannot load estimate")
		return
	}
	if !ok {
		estimateWriteError(w, http.StatusNotFound, "estimate not found")
		return
	}
	estimateWriteJSON(w, http.StatusOK, buildAISummary(estimate))
}

func buildAISummary(estimate Estimate) AISummaryResponse {
	high, medium := 0, 0
	keyRisks := make([]string, 0)
	for _, finding := range estimate.Findings {
		switch strings.ToLower(finding.Severity) {
		case "high":
			high++
		case "medium":
			medium++
		}
		if finding.Severity == "High" || finding.Severity == "Medium" {
			keyRisks = append(keyRisks, finding.Title+": "+finding.Detail)
		}
	}
	if len(keyRisks) > 6 {
		keyRisks = keyRisks[:6]
	}

	riskLevel := "Низкий"
	if high > 0 {
		riskLevel = "Высокий"
	} else if medium > 1 {
		riskLevel = "Средний"
	}

	brief := fmt.Sprintf("Смета %q проанализирована: найдено %d строк, сумма по найденным позициям %.2f, оценка %d/100.", estimate.FileName, estimate.ItemsCount, estimate.TotalAmount, estimate.Score)
	if high > 0 {
		brief += fmt.Sprintf(" Есть %d замечаний высокого риска, их нужно проверить до оплаты или утверждения бюджета.", high)
	} else if medium > 0 {
		brief += fmt.Sprintf(" Есть %d замечаний среднего риска, их желательно обсудить с подрядчиком.", medium)
	} else {
		brief += " Критичных рисков по базовым правилам не найдено, но финальную проверку всё равно должен сделать ответственный человек."
	}

	questions := []string{
		"Какие строки подрядчик должен уточнить до оплаты?",
		"Есть ли позиции без количества, цены, единицы измерения или суммы?",
		"Почему появились крупные суммы или возможные дубли?",
		"Совпадает ли итоговая сумма с условиями договора и объёмом работ?",
	}

	recommendation := "Используйте этот анализ как список вопросов к прорабу, сметчику или подрядчику. Сначала закройте замечания High, затем Medium, после этого можно повторно загрузить исправленную смету и сравнить версии."

	return AISummaryResponse{EstimateID: estimate.ID, ExecutiveBrief: brief, RiskLevel: riskLevel, KeyRisks: keyRisks, Questions: questions, Recommendation: recommendation}
}
