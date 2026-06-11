package api

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestAnalyzeExcelWorkbookReadsEverySheet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "multi.xlsx")
	book := excelize.NewFile()
	book.SetSheetName(book.GetSheetName(0), "Материалы")
	if _, err := book.NewSheet("Работы"); err != nil { t.Fatal(err) }
	for sheet, item := range map[string][]any{
		"Материалы": {"Кирпич", "шт", 100, 12, 1200},
		"Работы": {"Монтаж", "м2", 10, 500, 5000},
	} {
		headers := []any{"Наименование", "Ед", "Количество", "Цена", "Сумма"}
		for column, value := range headers {
			cell, _ := excelize.CoordinatesToCellName(column+1, 1)
			if err := book.SetCellValue(sheet, cell, value); err != nil { t.Fatal(err) }
		}
		for column, value := range item {
			cell, _ := excelize.CoordinatesToCellName(column+1, 2)
			if err := book.SetCellValue(sheet, cell, value); err != nil { t.Fatal(err) }
		}
	}
	if err := book.SaveAs(path); err != nil { t.Fatal(err) }
	_ = book.Close()

	items, findings := analyzeEstimateFile(path, "multi.xlsx", 1)
	if len(items) != 2 { t.Fatalf("expected 2 items, got %d", len(items)) }
	if sumItemsTotal(items) != 6200 { t.Fatalf("expected 6200, got %.2f", sumItemsTotal(items)) }
	names := items[0].Name + items[1].Name
	if !strings.Contains(names, "Материалы") || !strings.Contains(names, "Работы") {
		t.Fatalf("sheet names missing: %s", names)
	}
	for _, finding := range findings {
		if finding.Title == "Excel обработан полностью" { return }
	}
	t.Fatal("complete workbook finding missing")
}
