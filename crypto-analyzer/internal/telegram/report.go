package telegram

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/NursultanKoshoev11/SmetaCheck/crypto-analyzer/internal/domain"
)

func FormatReport(appName string, run domain.ScanRun, signals []domain.Signal, topN int, location *time.Location) string {
	if topN <= 0 {
		topN = 10
	}
	if location == nil {
		location = time.UTC
	}
	buys := make([]domain.Signal, 0)
	sells := make([]domain.Signal, 0)
	for _, signal := range signals {
		switch signal.Action {
		case "BUY":
			buys = append(buys, signal)
		case "SELL":
			sells = append(sells, signal)
		}
	}
	sort.Slice(buys, func(i, j int) bool { return buys[i].Score > buys[j].Score })
	sort.Slice(sells, func(i, j int) bool { return sells[i].Score < sells[j].Score })
	if len(buys) > topN {
		buys = buys[:topN]
	}
	if len(sells) > topN {
		sells = sells[:topN]
	}

	var b strings.Builder
	b.WriteString("📊 " + appName + "\n")
	b.WriteString(time.Now().In(location).Format("02.01.2006 15:04 MST") + "\n\n")
	fmt.Fprintf(&b, "Сканирование: %d/%d монет\n", run.SymbolsAnalyzed, run.SymbolsSelected)
	fmt.Fprintf(&b, "Сильных сигналов: %d\n", len(buys)+len(sells))

	if len(buys) == 0 && len(sells) == 0 {
		b.WriteString("\n⚪ Сейчас статистически сильных сигналов нет. Решение: наблюдать.\n")
	}
	if len(buys) > 0 {
		b.WriteString("\n🟢 BUY-КАНДИДАТЫ\n")
		for i, signal := range buys {
			writeSignal(&b, i+1, signal)
		}
	}
	if len(sells) > 0 {
		b.WriteString("\n🔴 SELL-КАНДИДАТЫ\n")
		for i, signal := range sells {
			writeSignal(&b, i+1, signal)
		}
	}
	b.WriteString("\nВажно: это исследовательские сигналы по истории, а не гарантия и не автоматическая торговля. Проверяйте риск самостоятельно.")
	return b.String()
}

func FormatCoin(symbol string, signals []domain.Signal) string {
	var b strings.Builder
	b.WriteString("📈 История сигналов " + symbol + "\n")
	if len(signals) == 0 {
		b.WriteString("Данных пока нет.")
		return b.String()
	}
	for _, signal := range signals {
		fmt.Fprintf(&b, "\n%s | %s | score %+.3f | confidence %.0f | price %s\n",
			signal.CreatedAt.Format("02.01 15:04"), signal.Action, signal.Score, signal.Confidence, formatPrice(signal.Price))
	}
	return b.String()
}

func FormatStatus(run domain.ScanRun) string {
	if run.ID == 0 {
		return "Сканирования ещё не запускались."
	}
	return fmt.Sprintf("Последнее сканирование #%d\nСтатус: %s\nНачало: %s\nЗавершение: %s\nМонеты: %d/%d\nСигналы: %d\nОшибка: %s",
		run.ID, run.Status, run.StartedAt.Format(time.RFC3339), run.FinishedAt.Format(time.RFC3339),
		run.SymbolsAnalyzed, run.SymbolsSelected, run.SignalsCreated, emptyDash(run.ErrorMessage))
}

func HelpText() string {
	return "Команды:\n/report — последний отчёт\n/status — состояние сканера\n/coin BTCUSDT — история монеты\n/help — эта справка"
}

func writeSignal(b *strings.Builder, index int, signal domain.Signal) {
	fmt.Fprintf(b, "\n%d. %s\n", index, signal.Symbol)
	fmt.Fprintf(b, "Цена: %s\n", formatPrice(signal.Price))
	fmt.Fprintf(b, "Score: %+.3f | Confidence: %.0f/100\n", signal.Score, signal.Confidence)
	fmt.Fprintf(b, "Средний исход аналогов: %+.2f%% | Исторических точек: %d\n", signal.ExpectedReturn, signal.SampleCount)
	if signal.FundingRate != 0 {
		fmt.Fprintf(b, "Funding: %+.4f%%\n", signal.FundingRate*100)
	}
	if signal.OpenInterestPct != 0 {
		fmt.Fprintf(b, "Open interest 1h: %+.2f%%\n", signal.OpenInterestPct)
	}
	for _, reason := range signal.Reasons {
		fmt.Fprintf(b, "• %s\n", reason)
	}
}

func formatPrice(value float64) string {
	switch {
	case value >= 1000:
		return fmt.Sprintf("%.2f", value)
	case value >= 1:
		return fmt.Sprintf("%.4f", value)
	default:
		return fmt.Sprintf("%.8f", value)
	}
}

func emptyDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "—"
	}
	return value
}
