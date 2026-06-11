package api

import (
	"fmt"
	"strings"
)

type AISummaryResponse struct {
	EstimateID       string   `json:"estimate_id"`
	ExecutiveBrief   string   `json:"executive_brief"`
	RiskLevel        string   `json:"risk_level"`
	DataQualityScore int      `json:"data_quality_score"`
	KeyRisks         []string `json:"key_risks"`
	PriorityActions  []string `json:"priority_actions"`
	CostFlags        []string `json:"cost_flags"`
	Questions        []string `json:"questions"`
	Recommendation   string   `json:"recommendation"`
}

func buildAISummary(estimate Estimate) AISummaryResponse {
	high, medium := 0, 0
	keyRisks := make([]string, 0)
	costFlags := make([]string, 0)
	for _, finding := range estimate.Findings {
		severity := strings.ToLower(finding.Severity)
		switch severity {
		case "high":
			high++
		case "medium":
			medium++
		}
		if finding.Severity == "High" || finding.Severity == "Medium" {
			keyRisks = append(keyRisks, finding.Title+": "+finding.Detail)
		}
	}
	if len(keyRisks) > 8 { keyRisks = keyRisks[:8] }

	largeItems := 0
	zeroValueItems := 0
	for _, item := range estimate.Items {
		if item.Total > 5000000 || item.UnitPrice > 1000000 { largeItems++ }
		if item.Total <= 0 || item.Quantity <= 0 || item.UnitPrice <= 0 { zeroValueItems++ }
	}
	if largeItems > 0 { costFlags = append(costFlags, fmt.Sprintf("Найдено %d крупных позиций, их нужно проверить вручную.", largeItems)) }
	if zeroValueItems > 0 { costFlags = append(costFlags, fmt.Sprintf("Найдено %d позиций с нулевыми или пустыми значениями.", zeroValueItems)) }
	if estimate.TotalAmount > 0 { costFlags = append(costFlags, fmt.Sprintf("Общая сумма по распознанным позициям: %.2f.", estimate.TotalAmount)) }

	dataQuality := estimate.Score
	if estimate.ItemsCount == 0 { dataQuality = 20 }
	if dataQuality < 0 { dataQuality = 0 }
	if dataQuality > 100 { dataQuality = 100 }

	riskLevel := "Низкий"
	if high > 0 || estimate.Score < 70 {
		riskLevel = "Высокий"
	} else if medium > 1 || estimate.Score < 85 {
		riskLevel = "Средний"
	}

	brief := fmt.Sprintf("Смета %q проанализирована: найдено %d строк, сумма по найденным позициям %.2f, оценка %d/100, качество данных %d/100.", estimate.FileName, estimate.ItemsCount, estimate.TotalAmount, estimate.Score, dataQuality)
	if high > 0 {
		brief += fmt.Sprintf(" Найдено %d замечаний высокого риска: перед оплатой или утверждением бюджета их нужно закрыть первыми.", high)
	} else if medium > 0 {
		brief += fmt.Sprintf(" Найдено %d замечаний среднего риска: их нужно обсудить с подрядчиком до финального согласования.", medium)
	} else {
		brief += " Критичных рисков по базовым правилам не найдено, но финальную проверку всё равно должен сделать ответственный человек."
	}

	priorityActions := []string{}
	if high > 0 { priorityActions = append(priorityActions, "Сначала исправить все High-замечания: пустые цены, неправильные суммы, критичные расхождения.") }
	if medium > 0 { priorityActions = append(priorityActions, "После этого проверить Medium-замечания: дубли, крупные суммы, неполные данные.") }
	if zeroValueItems > 0 { priorityActions = append(priorityActions, "Попросить подрядчика заполнить нулевые или пустые значения по количеству, цене и сумме.") }
	if largeItems > 0 { priorityActions = append(priorityActions, "Запросить расшифровку по крупным позициям и сверить их с объёмом работ.") }
	priorityActions = append(priorityActions, "Повторно загрузить исправленную смету и сравнить версии через SmetaCheck.")

	questions := []string{
		"Какие строки подрядчик должен уточнить до оплаты?",
		"Есть ли позиции без количества, цены, единицы измерения или суммы?",
		"Почему появились крупные суммы или возможные дубли?",
		"Совпадает ли итоговая сумма с договором и объёмом работ?",
		"Какие позиции изменились после последней версии сметы?",
	}

	recommendation := "Используйте AI-анализ как рабочий список вопросов к прорабу, сметчику или подрядчику. Сначала закройте High, затем Medium, после этого повторите проверку и сохраните финальный отчёт в PDF."

	return AISummaryResponse{EstimateID: estimate.ID, ExecutiveBrief: brief, RiskLevel: riskLevel, DataQualityScore: dataQuality, KeyRisks: keyRisks, PriorityActions: priorityActions, CostFlags: costFlags, Questions: questions, Recommendation: recommendation}
}
