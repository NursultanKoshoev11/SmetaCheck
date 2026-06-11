package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

const aiPromptVersion = "smetacheck-estimate-audit-v2"

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
	InputMode        string         `json:"input_mode,omitempty"`
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
	provider, err := configuredAIProvider("", "")
	if err != nil {
		fallback := buildAISummary(estimate)
		fallback.Warning = "AI configuration error: " + err.Error()
		return fallback
	}
	inputHash := aiInputHash(estimate)
	if cached, found, err := pgLoadAIReport(ctx, ownerID, estimate.ID, inputHash, provider.Name(), provider.Model()); err == nil && found {
		return cached
	} else if err != nil {
		log.Printf("cannot load cached AI report estimate_id=%s: %v", estimate.ID, err)
	}

	batch, err := provider.AnalyzeBatch(ctx, buildAIProviderInputs([]Estimate{estimate}))
	if err != nil || len(batch.Files) == 0 {
		fallback := buildAISummary(estimate)
		if err != nil {
			fallback.Warning = "AI временно недоступен: " + err.Error()
		} else {
			fallback.Warning = "AI не вернул анализ файла."
		}
		return fallback
	}
	result := summaryFromAIFile(estimate, batch.Files[0], provider.Name(), provider.Model())
	if err := pgSaveAIReport(ctx, ownerID, inputHash, result); err != nil {
		log.Printf("cannot cache AI report estimate_id=%s: %v", estimate.ID, err)
	}
	return result
}

func summaryFromAIFile(estimate Estimate, file AIFileAnalysis, provider, model string) AISummaryResponse {
	keyRisks := make([]string, 0, len(file.Findings))
	costFlags := make([]string, 0)
	questions := make([]string, 0)
	for _, finding := range file.Findings {
		if normalizeSeverity(finding.Severity) == "High" || normalizeSeverity(finding.Severity) == "Medium" {
			keyRisks = append(keyRisks, finding.Title+": "+finding.Detail)
		}
		if finding.Category == "cost_anomaly" || finding.Category == "arithmetic" {
			costFlags = append(costFlags, finding.Detail)
		}
		if strings.TrimSpace(finding.SuggestedAction) != "" {
			questions = append(questions, finding.SuggestedAction)
		}
	}
	if len(keyRisks) > 10 { keyRisks = keyRisks[:10] }
	if len(costFlags) > 10 { costFlags = costFlags[:10] }
	if len(questions) > 10 { questions = questions[:10] }
	return AISummaryResponse{
		EstimateID: estimate.ID, ExecutiveBrief: file.Summary, RiskLevel: normalizeRiskLevel(file.RiskLevel),
		DataQualityScore: clampScore(file.DataQualityScore), KeyRisks: keyRisks,
		PriorityActions: deduplicateStrings(file.Recommendations), CostFlags: deduplicateStrings(costFlags),
		Questions: deduplicateStrings(questions), Recommendation: firstRecommendation(file.Recommendations),
		ChartData: chartFromAIFindings(file.Findings), AnalysisSource: provider,
		Provider: provider, Model: model, InputMode: file.InputMode,
		PromptVersion: aiPromptVersion, GeneratedAt: time.Now().UTC(), Warning: file.Warning,
	}
}

func firstRecommendation(values []string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" { return strings.TrimSpace(value) }
	}
	return "Проверьте объединённые замечания backend и AI до утверждения сметы."
}

func buildAISummary(estimate Estimate) AISummaryResponse {
	high, medium := 0, 0
	keyRisks := make([]string, 0)
	costFlags := make([]string, 0)
	for _, finding := range estimate.Findings {
		switch strings.ToLower(finding.Severity) { case "high": high++; case "medium": medium++ }
		if finding.Severity == "High" || finding.Severity == "Medium" { keyRisks = append(keyRisks, finding.Title+": "+finding.Detail) }
	}
	if len(keyRisks)>8 { keyRisks=keyRisks[:8] }
	largeItems,zeroValueItems:=0,0
	for _,item:=range estimate.Items {
		if item.Total>5000000||item.UnitPrice>1000000 {largeItems++}
		if item.Total<=0||item.Quantity<=0||item.UnitPrice<=0 {zeroValueItems++}
	}
	if largeItems>0 {costFlags=append(costFlags,fmt.Sprintf("Крупных позиций, требующих ручной проверки: %d.",largeItems))}
	if zeroValueItems>0 {costFlags=append(costFlags,fmt.Sprintf("Позиций с нулевыми или пустыми значениями: %d.",zeroValueItems))}
	if estimate.TotalAmount>0 {costFlags=append(costFlags,fmt.Sprintf("Сумма по распознанным позициям: %.2f сом.",estimate.TotalAmount))}
	dataQuality:=clampScore(estimate.Score);if estimate.ItemsCount==0 {dataQuality=0}
	riskLevel:="Низкий";if estimate.ItemsCount==0 {riskLevel="Не определён"} else if high>0||estimate.Score<70 {riskLevel="Высокий"} else if medium>1||estimate.Score<85 {riskLevel="Средний"}
	priority:=make([]string,0)
	if high>0 {priority=append(priority,"Исправить все High-замечания.")}
	if medium>0 {priority=append(priority,"Проверить все Medium-замечания.")}
	priority=append(priority,"После исправлений повторно загрузить смету и сравнить версии.")
	return AISummaryResponse{
		EstimateID:estimate.ID,ExecutiveBrief:ruleBasedBrief(estimate,high,medium,dataQuality),RiskLevel:riskLevel,
		DataQualityScore:dataQuality,KeyRisks:keyRisks,PriorityActions:priority,CostFlags:costFlags,
		Questions:buildRuleQuestions(estimate),Recommendation:"Используйте backend-анализ как проверяемый источник расчётов.",
		ChartData:buildRiskChart(estimate.Findings),AnalysisSource:"rules",Provider:"rules",Model:"deterministic-v2",
		InputMode:"backend_rules",PromptVersion:aiPromptVersion,GeneratedAt:time.Now().UTC(),
	}
}

func aiInputHash(estimate Estimate) string {
	payload,err:=json.Marshal(aiHashInput{PromptVersion:aiPromptVersion,EstimateID:estimate.ID,FileName:estimate.FileName,Score:estimate.Score,ItemsCount:estimate.ItemsCount,TotalAmount:estimate.TotalAmount,Items:estimate.Items,Findings:estimate.Findings})
	if err!=nil {payload=[]byte(estimate.ID+aiPromptVersion)}
	sum:=sha256.Sum256(payload);return hex.EncodeToString(sum[:])
}

func buildRiskChart(findings []Finding) []AIChartPoint {
	counts:=map[string]int{"High":0,"Medium":0,"Low":0,"Info":0}
	for _,finding:=range findings {if _,ok:=counts[finding.Severity];ok{counts[finding.Severity]++}}
	return chartFromCounts(counts)
}
func clampScore(value int) int {if value<0{return 0};if value>100{return 100};return value}
func ruleBasedBrief(estimate Estimate,high,medium,dataQuality int) string {
	if estimate.ItemsCount==0{return "Система не смогла распознать позиции сметы."}
	brief:=fmt.Sprintf("Распознано %d позиций на сумму %.2f сом.",estimate.ItemsCount,estimate.TotalAmount)
	if high>0 {brief+=fmt.Sprintf(" High: %d.",high)} else if medium>0 {brief+=fmt.Sprintf(" Medium: %d.",medium)} else {brief+=" Критичных рисков не найдено."}
	if dataQuality<70 {brief+=" Качество данных низкое."};return brief
}
func buildRuleQuestions(estimate Estimate) []string {
	result:=make([]string,0)
	for _,finding:=range estimate.Findings {if finding.Severity=="High"||finding.Severity=="Medium"{result=append(result,finding.Title+": "+finding.Detail);if len(result)==6{break}}}
	if len(result)==0 {result=append(result,"Совпадает ли итоговая сумма с договором и объёмом работ?")};return result
}
