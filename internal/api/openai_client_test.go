package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestOpenAIAnalysis(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/responses" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected authorization header: %q", got)
		}

		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request["store"] != false {
			t.Fatal("OpenAI request must set store=false")
		}
		text, _ := json.Marshal(request["input"])
		if !strings.Contains(string(text), "Кирпич") {
			t.Fatal("parsed estimate rows were not sent to the model")
		}
		if strings.Contains(string(text), "file_path") {
			t.Fatal("local file paths must not be sent to the model")
		}

		structured := `{"executive_brief":"Проверена одна позиция.","risk_level":"Средний","data_quality_score":88,"key_risks":["Строка 2 требует проверки"],"priority_actions":["Проверить цену"],"cost_flags":[],"questions":["Почему указана эта цена?"],"recommendation":"Проверить строку до оплаты."}`
		response := map[string]any{
			"id":     "resp_test",
			"status": "completed",
			"output": []any{
				map[string]any{
					"type": "message",
					"content": []any{
						map[string]any{"type": "output_text", "text": structured},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("OPENAI_BASE_URL", server.URL)
	t.Setenv("OPENAI_MODEL", "test-model")
	t.Setenv("AI_TIMEOUT", "2s")

	estimate := Estimate{
		ID:          "est_test",
		FileName:    "estimate.csv",
		Score:       88,
		ItemsCount:  1,
		TotalAmount: 12000,
		Items: []EstimateItem{
			{Row: 2, Name: "Кирпич", Unit: "шт", Quantity: 1000, UnitPrice: 12, Total: 12000},
		},
		Findings: []Finding{
			{Title: "Крупная сумма", Severity: "Medium", Detail: "Строка 2"},
		},
	}

	report, err := requestOpenAIAnalysis(t.Context(), estimate)
	if err != nil {
		t.Fatal(err)
	}
	if report.RiskLevel != "Средний" || report.DataQualityScore != 88 {
		t.Fatalf("unexpected report: %+v", report)
	}
	if report.ExecutiveBrief == "" || len(report.PriorityActions) != 1 {
		t.Fatalf("incomplete report: %+v", report)
	}
}

func TestBuildAISummaryWithNoParsedRows(t *testing.T) {
	report := buildAISummary(Estimate{ID: "est_empty", FileName: "estimate.pdf", Score: 94})
	if report.RiskLevel != "Не определён" {
		t.Fatalf("expected unknown risk, got %q", report.RiskLevel)
	}
	if report.DataQualityScore != 0 {
		t.Fatalf("expected zero data quality, got %d", report.DataQualityScore)
	}
	if report.AnalysisSource != "rules" {
		t.Fatalf("expected rules fallback, got %q", report.AnalysisSource)
	}
}
