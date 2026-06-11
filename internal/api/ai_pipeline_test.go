package api

import "testing"

func TestSplitEstimateForAICoversEveryRow(t *testing.T) {
	t.Setenv("AI_ROWS_PER_CHUNK", "25")
	items := make([]EstimateItem, 61)
	for index := range items {
		items[index] = EstimateItem{Row: index + 1, Name: "Item", Quantity: 1, UnitPrice: 1, Total: 1}
	}
	estimate := Estimate{ID: "est_rows", FileName: "estimate.csv", Items: items, ItemsCount: len(items)}
	chunks := splitEstimateForAI(estimate)
	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(chunks))
	}
	rows := 0
	for index, chunk := range chunks {
		rows += len(chunk.Rows)
		if chunk.ChunkIndex != index+1 || chunk.ChunkCount != 3 {
			t.Fatalf("unexpected chunk metadata: %+v", chunk)
		}
	}
	if rows != len(items) {
		t.Fatalf("expected all %d rows, got %d", len(items), rows)
	}
}

func TestAIInputDoesNotContainBackendConclusions(t *testing.T) {
	estimate := Estimate{
		ID: "est_independent", FileName: "estimate.csv", Score: 10, TotalAmount: 999,
		ItemsCount: 1,
		Items: []EstimateItem{{Row: 2, Name: "Brick", Quantity: 2, UnitPrice: 3, Total: 6}},
		Findings: []Finding{{Title: "Backend issue", Severity: "High", Detail: "row 2"}},
	}
	text := normalizedEstimateText(estimate)
	for _, forbidden := range []string{"backend_score", "backend_total", "backend_issues", "Backend issue"} {
		if containsText(text, forbidden) {
			t.Fatalf("AI input contains backend conclusion %q: %s", forbidden, text)
		}
	}
}

func TestBackendFallbackIsNotCountedAsAIConfirmation(t *testing.T) {
	estimate := Estimate{
		ID: "est_fallback", FileName: "estimate.csv", Score: 70, ItemsCount: 1,
		Findings: []Finding{{Title: "Wrong total", Severity: "High", Detail: "row 2"}},
	}
	fallback := rulesFileAnalysis(AIProviderInput{Estimate: estimate})
	fallback.InputMode = "backend_fallback"
	fallback.Warning = "provider unavailable"
	report := reconcileFile(estimate, fallback)
	if report.AgreementScore != 0 {
		t.Fatalf("fallback must not count as independent agreement, got %d", report.AgreementScore)
	}
	if report.AIRiskLevel != "Не определён" {
		t.Fatalf("fallback must not expose AI risk, got %q", report.AIRiskLevel)
	}
	if len(report.Findings) != 1 || report.Findings[0].Reconciliation != "backend_only_ai_unavailable" {
		t.Fatalf("unexpected fallback reconciliation: %+v", report.Findings)
	}
}

func TestIndependentAIAgreementIncludesCoverage(t *testing.T) {
	estimate := Estimate{
		ID: "est_coverage", FileName: "estimate.csv", Score: 70, ItemsCount: 100,
		Findings: []Finding{{Title: "Wrong total", Severity: "High", Detail: "row 2"}},
	}
	ai := AIFileAnalysis{
		EstimateID: estimate.ID, FileName: estimate.FileName, InputMode: "normalized_chunks",
		RiskLevel: "Высокий", RowsAnalyzed: 50,
		Findings: []AIProviderFinding{{Row: 2, Category: "arithmetic", Title: "Wrong total", Severity: "High", Detail: "row 2"}},
	}
	report := reconcileFile(estimate, ai)
	if report.AgreementScore != 50 {
		t.Fatalf("expected 50%% coverage-adjusted agreement, got %d", report.AgreementScore)
	}
	if report.Findings[0].Reconciliation != "confirmed_by_both" {
		t.Fatalf("expected confirmed finding, got %+v", report.Findings[0])
	}
}

func TestDecodeRawPDFPreservesIndependentRowCount(t *testing.T) {
	data := []byte(`{"summary":"ok","risk_level":"Средний","data_quality_score":80,"extracted_total":100,"rows_analyzed":12,"findings":[],"recommendations":[],"input_mode":"raw_pdf","warning":"","estimate_id":"x","file_name":"x.pdf"}`)
	report, err := decodeAIFileAnalysis(data, Estimate{ID: "est_pdf", FileName: "x.pdf", ItemsCount: 0}, "raw_pdf")
	if err != nil {
		t.Fatal(err)
	}
	if report.RowsAnalyzed != 12 {
		t.Fatalf("raw PDF rows were incorrectly clamped: %d", report.RowsAnalyzed)
	}
}

func containsText(value, needle string) bool {
	for index := 0; index+len(needle) <= len(value); index++ {
		if value[index:index+len(needle)] == needle {
			return true
		}
	}
	return false
}
