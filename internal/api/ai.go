package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"
)

const aiPromptVersion = "smetacheck-estimate-audit-v1"

type AIChartPoint struct {
	Label string `json:"label"`
	Value int    `json:"value"`
}

type AISummaryResponse struct {
	EstimateID       string         `json:"estimate_id"`
	ExecutiveBrief   string         `json:"executive_brief"`
	RiskLevel        string         `json:"risk_level"`
	DataQualityScore int            `json:"data_quality_score"`
	KeyRisks         []string       `json:"key_risks"`
	PriorityActions  []string       `json:"priority_actions"`
	CostFlags        []string       `json:"cost_flags"`
	Questions        []string       `json:"questions"`
	Recommendation   string         `json:"recommendation"`
	ChartData        []AIChartPoint `json:"chart_data"`
	AnalysisSource   string         `json:"analysis_source"`
	Provider         string         `json:"provider,omitempty"`
	Model            string         `json:"model,omitempty"`
	PromptVersion    string         `json:"prompt_version"`
	GeneratedAt      time.Time      `json:"generated_at"`
	Warning          string         `json:"warning,omitempty"`
}

type aiHashInput struct {
	PromptVersion string         `json:"prompt_version"`
	EstimateID    string         `json:"estimate_id"`
	FileName      string         `json:"file_name"`
	Score         int            `json:"score"`
	ItemsCount    int            `json:"items_count"`
	TotalAmount   float64        `json:"total_amount"`
	Items         []EstimateItem `json:"items"`
	Findings      []Finding      `json:"findings"`
}

func generateAISummary(ctx context.Context, ownerID string, estimate Estimate) AISummaryResponse {
	provider := strings.ToLower(strings.TrimSpace(os.Getenv("AI_PROVIDER")))
	if provider == "" || provider == "rules" {
		return buildAISummary(estimate)
	}
	if provider != "openai" {
		fallback := buildAISummary(estimate)
		fallback.Warning = "Неизвестный AI_PROVIDER. Использована автоматическая сводка по правилам."
		return fallback
	}

	inputHash := aiInputHash(estimate)
	if cached, found, err := pgLoadAIReport(ctx, ownerID, estimate.ID, inputHash); err == nil && found {
		return cached
	} else if err != nil {
		log.Printf("cannot load cached AI report estimate_id=%s: %v", estimate.ID, err)
	}

	report, err := requestOpenAIAnalysis(ctx, estimate)
	if err != nil {
		log.Printf("OpenAI analysis failed estimate_id=%s: %v", estimate.ID, err)
		fallback := buildAISummary(estimate)
		fallback.Warning = "AI временно недоступен. Показана автоматическая сводка по проверенным правилам."
		return fallback
	}

	report.EstimateID = estimate.ID
	report.AnalysisSource = "openai"
	report.Provider = "openai"
	report.Model = openAIModel()
	report.PromptVersion = aiPromptVersion
	report.GeneratedAt = time.Now().UTC()
	report.ChartData = buildRiskChart(estimate.Findings)
	report = normalizeAIReport(report, estimate)

	if err := pgSaveAIReport(ctx, ownerID, inputHash, report); err != nil {
		log.Printf("cannot cache AI report estimate_id=%s: %v", estimate.ID, err)
	}
	return report
}

func buildAISummary(estimate Estimate) AISummaryResponse {
	high, medium := 0, 0
	keyRisks := make([]string, 0)
	costFlags := make([]string, 0)
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
	if len(keyRisks) > 8 {
		keyRisks = keyRisks[:8]
	}

	largeItems := 0
	zeroValueItems := 0
	for _, item := range estimate.Items {
		if item.Total > 5000000 || item.UnitPrice > 1000000 {
			largeItems++
		}
		if item.Total <= 0 || item.Quantity <= 0 || item.UnitPrice <= 0 {
			zeroValueItems++
		}
	}
	if largeItems > 0 {
		costFlags = append(costFlags, pluralCount(largeItems, "крупная позиция требует", "крупные позиции требуют", "крупных позиций требуют")+" ручной проверки.")
	}
	if zeroValueItems > 0 {
		costFlags = append(costFlags, pluralCount(zeroValueItems, "позиция содержит", "позиции содержат", "позиций содержат")+" нулевые или пустые значения.")
	}
	if estimate.TotalAmount > 0 {
		costFlags = append(costFlags, moneySentence(estimate.TotalAmount))
	}

	dataQuality := estimate.Score
	if estimate.ItemsCount == 0 {
		dataQuality = 0
	}
	dataQuality = clampScore(dataQuality)

	riskLevel := "Низкий"
	if estimate.ItemsCount == 0 {
		riskLevel = "Не определён"
	} else if high > 0 || estimate.Score < 70 {
		riskLevel = "Высокий"
	} else if medium > 1 || estimate.Score < 85 {
		riskLevel = "Средний"
	}

	brief := ruleBasedBrief(estimate, high, medium, dataQuality)
	priorityActions := make([]string, 0)
	if high > 0 {
		priorityActions = append(priorityActions, "Сначала исправить High-замечания: пустые цены, неправильные суммы и критичные расхождения.")
	}
	if medium > 0 {
		priorityActions = append(priorityActions, "Затем проверить Medium-замечания: возможные дубли, крупные суммы и неполные данные.")
	}
	if zeroValueItems > 0 {
		priorityActions = append(priorityActions, "Попросить подрядчика заполнить нулевые или пустые количество, цену и сумму.")
	}
	if largeItems > 0 {
		priorityActions = append(priorityActions, "Запросить расшифровку крупных позиций и сверить их с объёмом работ.")
	}
	priorityActions = append(priorityActions, "После исправлений повторно загрузить смету и сравнить версии.")

	questions := buildRuleQuestions(estimate)
	return AISummaryResponse{
		EstimateID:       estimate.ID,
		ExecutiveBrief:   brief,
		RiskLevel:        riskLevel,
		DataQualityScore: dataQuality,
		KeyRisks:         keyRisks,
		PriorityActions:  priorityActions,
		CostFlags:        costFlags,
		Questions:        questions,
		Recommendation:   "Используйте сводку как список вопросов к прорабу, сметчику или подрядчику. Расчёты и найденные строки являются источником фактов; финальное решение принимает ответственный специалист.",
		ChartData:        buildRiskChart(estimate.Findings),
		AnalysisSource:   "rules",
		PromptVersion:    aiPromptVersion,
		GeneratedAt:      time.Now().UTC(),
	}
}

func aiInputHash(estimate Estimate) string {
	payload, err := json.Marshal(aiHashInput{
		PromptVersion: aiPromptVersion,
		EstimateID:    estimate.ID,
		FileName:      estimate.FileName,
		Score:         estimate.Score,
		ItemsCount:    estimate.ItemsCount,
		TotalAmount:   estimate.TotalAmount,
		Items:         estimate.Items,
		Findings:      estimate.Findings,
	})
	if err != nil {
		payload = []byte(estimate.ID + aiPromptVersion)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func buildRiskChart(findings []Finding) []AIChartPoint {
	counts := map[string]int{"High": 0, "Medium": 0, "Low": 0, "Info": 0}
	for _, finding := range findings {
		if _, ok := counts[finding.Severity]; ok {
			counts[finding.Severity]++
		}
	}
	return []AIChartPoint{
		{Label: "High", Value: counts["High"]},
		{Label: "Medium", Value: counts["Medium"]},
		{Label: "Low", Value: counts["Low"]},
		{Label: "Info", Value: counts["Info"]},
	}
}

func normalizeAIReport(report AISummaryResponse, estimate Estimate) AISummaryResponse {
	report.DataQualityScore = clampScore(report.DataQualityScore)
	if estimate.ItemsCount == 0 {
		report.RiskLevel = "Не определён"
		report.DataQualityScore = 0
	}
	switch report.RiskLevel {
	case "Низкий", "Средний", "Высокий", "Не определён":
	default:
		report.RiskLevel = buildAISummary(estimate).RiskLevel
	}
	fallback := buildAISummary(estimate)
	if strings.TrimSpace(report.ExecutiveBrief) == "" {
		report.ExecutiveBrief = fallback.ExecutiveBrief
	}
	if strings.TrimSpace(report.Recommendation) == "" {
		report.Recommendation = fallback.Recommendation
	}
	if report.KeyRisks == nil {
		report.KeyRisks = []string{}
	}
	if report.PriorityActions == nil {
		report.PriorityActions = fallback.PriorityActions
	}
	if report.CostFlags == nil {
		report.CostFlags = fallback.CostFlags
	}
	if report.Questions == nil {
		report.Questions = fallback.Questions
	}
	return report
}

func clampScore(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func ruleBasedBrief(estimate Estimate, high, medium, dataQuality int) string {
	if estimate.ItemsCount == 0 {
		return "Система не смогла распознать позиции сметы. AI-анализ финансовых рисков недоступен, пока файл не будет корректно прочитан."
	}
	brief := "Смета проанализирована по распознанным строкам и детерминированным проверкам."
	if high > 0 {
		brief += " Есть замечания высокого риска, которые нужно проверить до оплаты или утверждения бюджета."
	} else if medium > 0 {
		brief += " Есть замечания среднего риска, которые нужно обсудить с подрядчиком."
	} else {
		brief += " Критичных рисков по базовым правилам не найдено."
	}
	if dataQuality < 70 {
		brief += " Качество исходных данных низкое, поэтому выводы требуют дополнительной ручной проверки."
	}
	return brief
}

func buildRuleQuestions(estimate Estimate) []string {
	questions := make([]string, 0)
	for _, finding := range estimate.Findings {
		if finding.Severity != "High" && finding.Severity != "Medium" {
			continue
		}
		questions = append(questions, finding.Title+": "+finding.Detail)
		if len(questions) == 6 {
			break
		}
	}
	if len(questions) == 0 {
		questions = append(questions, "Совпадает ли итоговая сумма с договором и фактическим объёмом работ?")
	}
	return questions
}

func pluralCount(count int, one, few, many string) string {
	word := many
	mod10, mod100 := count%10, count%100
	if mod10 == 1 && mod100 != 11 {
		word = one
	} else if mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14) {
		word = few
	}
	return strings.TrimSpace(strings.Join([]string{itoa(count), word}, " "))
}

func itoa(value int) string {
	return json.Number(string(rune('0' + value))).String()
}

func moneySentence(total float64) string {
	return "Общая сумма по распознанным позициям сохранена в отчёте и должна быть сверена с итогом исходного документа."
}
