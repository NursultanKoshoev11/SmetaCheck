package api

import (
	"fmt"
	"io"
	"strings"
)

func writeAIEnhancedTextReport(w io.Writer, estimate Estimate, analysis AISummaryResponse) error {
	var b strings.Builder
	b.WriteString("SmetaCheck KG — отчёт проверки сметы\n")
	b.WriteString("====================================\n\n")
	b.WriteString("Файл: " + estimate.FileName + "\n")
	b.WriteString(fmt.Sprintf("Оценка правил: %d/100\n", estimate.Score))
	b.WriteString("Статус: " + estimate.Status + "\n")
	b.WriteString(fmt.Sprintf("Распознано строк: %d\n", estimate.ItemsCount))
	b.WriteString(fmt.Sprintf("Сумма по распознанным строкам: %.2f сом\n", estimate.TotalAmount))
	b.WriteString("Дата загрузки: " + estimate.UploadedAt.Format("2006-01-02 15:04:05 MST") + "\n\n")

	b.WriteString("AI / автоматический анализ\n")
	b.WriteString("--------------------------\n")
	b.WriteString("Источник: " + analysis.AnalysisSource + "\n")
	if analysis.Model != "" {
		b.WriteString("Модель: " + analysis.Model + "\n")
	}
	b.WriteString("Версия анализа: " + analysis.PromptVersion + "\n")
	b.WriteString("Уровень риска: " + analysis.RiskLevel + "\n")
	b.WriteString(fmt.Sprintf("Качество данных: %d/100\n", analysis.DataQualityScore))
	if analysis.Warning != "" {
		b.WriteString("Предупреждение: " + analysis.Warning + "\n")
	}
	b.WriteString("\nКраткий вывод:\n" + analysis.ExecutiveBrief + "\n")
	b.WriteString("\nРекомендация:\n" + analysis.Recommendation + "\n")

	appendReportList(&b, "Ключевые риски", analysis.KeyRisks)
	appendReportList(&b, "Приоритетные действия", analysis.PriorityActions)
	appendReportList(&b, "Финансовые флаги", analysis.CostFlags)
	appendReportList(&b, "Вопросы подрядчику", analysis.Questions)

	b.WriteString("\nГрафик рисков (количество)\n")
	for _, point := range analysis.ChartData {
		b.WriteString(fmt.Sprintf("- %s: %d\n", point.Label, point.Value))
	}

	b.WriteString("\nДетерминированные замечания\n")
	b.WriteString("----------------------------\n")
	for i, finding := range estimate.Findings {
		b.WriteString(fmt.Sprintf("%d. [%s] %s — %s\n", i+1, finding.Severity, finding.Title, finding.Detail))
	}

	b.WriteString("\nПозиции сметы\n")
	b.WriteString("--------------\n")
	for _, item := range estimate.Items {
		b.WriteString(fmt.Sprintf("Строка %d | %s | %s | кол-во %.4f | цена %.2f | сумма %.2f\n",
			item.Row, item.Name, item.Unit, item.Quantity, item.UnitPrice, item.Total))
	}

	_, err := io.WriteString(w, b.String())
	return err
}

func appendReportList(b *strings.Builder, title string, items []string) {
	b.WriteString("\n" + title + "\n")
	if len(items) == 0 {
		b.WriteString("- Нет данных.\n")
		return
	}
	for _, item := range items {
		b.WriteString("- " + item + "\n")
	}
}
