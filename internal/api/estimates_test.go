package api

import (
	"os"
	"testing"
)

func TestAnalyzeCSVEstimate(t *testing.T) {
	path := t.TempDir() + "/estimate.csv"
	content := "Наименование,Ед,Количество,Цена,Сумма\nКирпич,шт,1000,12,12000\nЦемент,мешок,20,450,9000\nПесок,м3,5,1200,6000\n"
	if err := os.WriteFile(path, []byte(content), 0o640); err != nil {
		t.Fatal(err)
	}
	items, findings := analyzeEstimateFile(path, "estimate.csv", int64(len(content)))
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	if sumItemsTotal(items) != 27000 {
		t.Fatalf("expected total 27000, got %.2f", sumItemsTotal(items))
	}
	if len(findings) == 0 {
		t.Fatal("expected findings")
	}
}

func TestMismatchFinding(t *testing.T) {
	items := []EstimateItem{{Row: 2, Name: "Кирпич", Unit: "шт", Quantity: 10, UnitPrice: 12, Total: 100}}
	findings := validateItems(items)
	found := false
	for _, finding := range findings {
		if finding.Title == "Сумма строки не сходится" && finding.Severity == "High" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected mismatch high finding")
	}
}

func TestCompareEstimateItems(t *testing.T) {
	base := []EstimateItem{{Row: 2, Name: "Кирпич", Unit: "шт", Quantity: 100, UnitPrice: 10, Total: 1000}, {Row: 3, Name: "Песок", Unit: "м3", Quantity: 1, UnitPrice: 500, Total: 500}}
	updated := []EstimateItem{{Row: 2, Name: "Кирпич", Unit: "шт", Quantity: 120, UnitPrice: 10, Total: 1200}, {Row: 3, Name: "Арматура", Unit: "кг", Quantity: 10, UnitPrice: 80, Total: 800}}
	result := compareEstimateItems("base.csv", "new.csv", base, updated)
	if len(result.Added) != 1 || len(result.Removed) != 1 || len(result.Changed) != 1 {
		t.Fatalf("unexpected compare result: added=%d removed=%d changed=%d", len(result.Added), len(result.Removed), len(result.Changed))
	}
	if result.DeltaTotal != 500 {
		t.Fatalf("expected delta 500, got %.2f", result.DeltaTotal)
	}
}

func TestAISummary(t *testing.T) {
	estimate := Estimate{ID: "est_test", FileName: "test.csv", Score: 70, ItemsCount: 2, TotalAmount: 1000, Findings: []Finding{{Title: "Нет цены", Severity: "High", Detail: "Строка 2"}}}
	summary := buildAISummary(estimate)
	if summary.RiskLevel != "Высокий" {
		t.Fatalf("expected high risk, got %s", summary.RiskLevel)
	}
	if summary.ExecutiveBrief == "" || len(summary.Questions) == 0 {
		t.Fatal("summary is incomplete")
	}
}
