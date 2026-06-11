package api

import (
	"fmt"
	"strings"
	"time"
)

func reconcileBatch(batchID string, estimates []Estimate, aiResult AIProviderBatchResponse) UnifiedBatchReport {
	aiByID := make(map[string]AIFileAnalysis, len(aiResult.Files))
	for _, file := range aiResult.Files { aiByID[file.EstimateID] = file }
	report := UnifiedBatchReport{
		BatchID: batchID, Provider: aiResult.Provider, Model: aiResult.Model,
		Status: "completed", TotalFiles: len(estimates), Files: make([]UnifiedFileReport,0,len(estimates)),
		CrossFile: make([]CrossFileComparison,0), Recommendations: append([]string(nil),aiResult.Recommendations...),
		GeneratedAt: time.Now().UTC(),
	}
	for _, estimate := range estimates {
		aiFile, ok := aiByID[estimate.ID]
		if !ok {
			aiFile = rulesFileAnalysis(AIProviderInput{Estimate: estimate})
			aiFile.Warning = "AI-провайдер не вернул результат для файла; использован backend-анализ."
		}
		fileReport := reconcileFile(estimate, aiFile)
		report.Files = append(report.Files,fileReport)
		report.TotalRows += estimate.ItemsCount
		report.TotalAmount += estimate.TotalAmount
		report.OverallRiskLevel = higherRiskLevel(report.OverallRiskLevel,fileReport.FinalRiskLevel)
	}
	if report.OverallRiskLevel=="" { report.OverallRiskLevel="Не определён" }
	report.CrossFile = compareBatchEstimates(estimates)
	report.UnifiedChart = aggregateUnifiedCharts(report.Files)
	if len(report.Recommendations)==0 { report.Recommendations=collectUnifiedRecommendations(report.Files) }
	report.Summary = fmt.Sprintf("Проверено файлов: %d. Backend обработал %d строк на сумму %.2f сом. Итоговый риск: %s. AI: %s / %s.", report.TotalFiles,report.TotalRows,report.TotalAmount,report.OverallRiskLevel,report.Provider,report.Model)
	return report
}

func compareBatchEstimates(estimates []Estimate) []CrossFileComparison {
	result:=make([]CrossFileComparison,0)
	for i:=0;i<len(estimates);i++ {
		for j:=i+1;j<len(estimates);j++ {
			a,b:=estimates[i],estimates[j]
			result=append(result,CrossFileComparison{Type:"total_delta",FileA:a.ID,FileB:b.ID,Description:fmt.Sprintf("Разница суммы между %s и %s.",a.FileName,b.FileName),Delta:b.TotalAmount-a.TotalAmount})
			shared:=crossFileDuplicateCount(a.Items,b.Items)
			if shared>0 { result=append(result,CrossFileComparison{Type:"shared_positions",FileA:a.ID,FileB:b.ID,Description:fmt.Sprintf("Общих или похожих позиций: %d.",shared),Delta:float64(shared)}) }
		}
	}
	return result
}

func crossFileDuplicateCount(a,b []EstimateItem) int {
	keys:=map[string]struct{}{}
	for _,item:=range a { if key:=normalizeDuplicateKey(item.Name,item.Unit); key!="" { keys[key]=struct{}{} } }
	seen:=map[string]struct{}{}
	count:=0
	for _,item:=range b {
		key:=normalizeDuplicateKey(item.Name,item.Unit)
		if _,ok:=keys[key];ok { if _,done:=seen[key];!done { seen[key]=struct{}{};count++ } }
	}
	return count
}

func chartFromAIFindings(findings []AIProviderFinding) []AIChartPoint {
	counts:=map[string]int{"High":0,"Medium":0,"Low":0,"Info":0}
	for _,item:=range findings { counts[normalizeSeverity(item.Severity)]++ }
	return chartFromCounts(counts)
}
func chartFromUnifiedFindings(findings []UnifiedFinding) []AIChartPoint {
	counts:=map[string]int{"High":0,"Medium":0,"Low":0,"Info":0}
	for _,item:=range findings { counts[normalizeSeverity(item.Severity)]++ }
	return chartFromCounts(counts)
}
func chartFromCounts(counts map[string]int) []AIChartPoint { return []AIChartPoint{{Label:"High",Value:counts["High"]},{Label:"Medium",Value:counts["Medium"]},{Label:"Low",Value:counts["Low"]},{Label:"Info",Value:counts["Info"]}} }
func aggregateUnifiedCharts(files []UnifiedFileReport) []AIChartPoint {
	counts:=map[string]int{"High":0,"Medium":0,"Low":0,"Info":0}
	for _,file:=range files { for _,point:=range file.UnifiedChart { counts[point.Label]+=point.Value } }
	return chartFromCounts(counts)
}
func riskFromProviderFindings(findings []AIProviderFinding) string {
	risk:="Низкий"
	for _,item:=range findings { switch normalizeSeverity(item.Severity) { case "High":return "Высокий";case "Medium":risk="Средний" } }
	return risk
}
func deduplicateStrings(values []string) []string {
	seen:=map[string]struct{}{}
	result:=make([]string,0,len(values))
	for _,value:=range values { value=strings.TrimSpace(value);if value==""{continue};key:=strings.ToLower(value);if _,ok:=seen[key];ok{continue};seen[key]=struct{}{};result=append(result,value) }
	return result
}
func collectUnifiedRecommendations(files []UnifiedFileReport) []string { values:=make([]string,0);for _,file:=range files{values=append(values,file.Recommendations...)};return deduplicateStrings(values) }
