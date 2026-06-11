package api

import (
	"math"
	"sort"
	"strings"
)

func reconcileFile(estimate Estimate, aiFile AIFileAnalysis) UnifiedFileReport {
	backend := backendProviderFindings(estimate)
	if !isIndependentAIAnalysis(aiFile) {
		unified := make([]UnifiedFinding, 0, len(backend))
		for _, item := range backend {
			unified = append(unified, UnifiedFinding{
				FileID: estimate.ID, Row: item.Row, Category: item.Category, Title: item.Title,
				Severity: item.Severity, Detail: item.Detail, SuggestedAction: item.SuggestedAction,
				Sources: []string{"backend"}, Reconciliation: "backend_only_ai_unavailable",
			})
		}
		sortUnifiedFindings(unified)
		backendRisk := riskFromProviderFindings(backend)
		if backendRisk == "" { backendRisk = "Не определён" }
		summary := strings.TrimSpace(aiFile.Warning)
		if summary == "" { summary = "Независимый AI-анализ недоступен. Итог построен только по backend-проверкам." }
		return UnifiedFileReport{
			EstimateID: estimate.ID, FileName: estimate.FileName, BackendScore: estimate.Score,
			AIRiskLevel: "Не определён", FinalRiskLevel: backendRisk, AgreementScore: 0,
			Summary: summary, TotalAmount: estimate.TotalAmount, AIExtractedTotal: 0,
			Findings: unified, BackendChart: buildRiskChart(estimate.Findings),
			AIChart: chartFromAIFindings(nil), UnifiedChart: chartFromUnifiedFindings(unified),
			Recommendations: deduplicateStrings(aiFile.Recommendations), AIAnalysis: aiFile,
		}
	}

	unified := make([]UnifiedFinding, 0, len(backend)+len(aiFile.Findings))
	usedAI := make([]bool, len(aiFile.Findings))
	agreements := 0
	for _, item := range backend {
		match := bestAIMatch(item, aiFile.Findings, usedAI)
		if match >= 0 {
			usedAI[match] = true
			other := aiFile.Findings[match]
			agreements++
			unified = append(unified, UnifiedFinding{
				FileID: estimate.ID, Row: chooseRow(item.Row, other.Row),
				Category: chooseNonEmpty(item.Category, other.Category),
				Title: chooseNonEmpty(other.Title, item.Title), Severity: maxSeverity(item.Severity, other.Severity),
				Detail: mergeDetails(item.Detail, other.Detail),
				SuggestedAction: chooseNonEmpty(other.SuggestedAction, item.SuggestedAction),
				Sources: []string{"backend", "ai"}, Reconciliation: "confirmed_by_both",
			})
			continue
		}
		unified = append(unified, UnifiedFinding{
			FileID: estimate.ID, Row: item.Row, Category: item.Category, Title: item.Title,
			Severity: item.Severity, Detail: item.Detail, SuggestedAction: item.SuggestedAction,
			Sources: []string{"backend"}, Reconciliation: "backend_only",
		})
	}
	for index, item := range aiFile.Findings {
		if usedAI[index] { continue }
		unified = append(unified, UnifiedFinding{
			FileID: estimate.ID, Row: item.Row, Category: item.Category, Title: item.Title,
			Severity: normalizeSeverity(item.Severity), Detail: item.Detail,
			SuggestedAction: item.SuggestedAction, Sources: []string{"ai"},
			Reconciliation: "ai_only_needs_review",
		})
	}
	sortUnifiedFindings(unified)

	denominator := len(backend) + len(aiFile.Findings) - agreements
	agreementScore := 100
	if denominator > 0 { agreementScore = int(math.Round(float64(agreements) / float64(denominator) * 100)) }
	coverage := 1.0
	if estimate.ItemsCount > 0 { coverage = math.Min(1, float64(aiFile.RowsAnalyzed)/float64(estimate.ItemsCount)) }
	agreementScore = int(math.Round(float64(agreementScore) * coverage))

	finalRisk := higherRiskLevel(riskFromProviderFindings(backend), aiFile.RiskLevel)
	if finalRisk == "" { finalRisk = "Не определён" }
	return UnifiedFileReport{
		EstimateID: estimate.ID, FileName: estimate.FileName, BackendScore: estimate.Score,
		AIRiskLevel: normalizeRiskLevel(aiFile.RiskLevel), FinalRiskLevel: finalRisk,
		AgreementScore: agreementScore, Summary: aiFile.Summary, TotalAmount: estimate.TotalAmount,
		AIExtractedTotal: aiFile.ExtractedTotal, Findings: unified,
		BackendChart: buildRiskChart(estimate.Findings), AIChart: chartFromAIFindings(aiFile.Findings),
		UnifiedChart: chartFromUnifiedFindings(unified),
		Recommendations: deduplicateStrings(aiFile.Recommendations), AIAnalysis: aiFile,
	}
}

func isIndependentAIAnalysis(aiFile AIFileAnalysis) bool {
	mode := strings.ToLower(strings.TrimSpace(aiFile.InputMode))
	return mode != "" && mode != "backend_rules" && mode != "backend_fallback"
}

func sortUnifiedFindings(findings []UnifiedFinding) {
	sort.SliceStable(findings, func(i, j int) bool {
		left, right := severityRank(findings[i].Severity), severityRank(findings[j].Severity)
		if left != right { return left > right }
		return findings[i].Row < findings[j].Row
	})
}

func backendProviderFindings(estimate Estimate) []AIProviderFinding {
	result := make([]AIProviderFinding, 0, len(estimate.Findings))
	for _, finding := range estimate.Findings {
		result = append(result, AIProviderFinding{
			FileID: estimate.ID, Row: extractRowNumber(finding.Detail),
			Category: backendFindingCategory(finding), Title: finding.Title,
			Severity: normalizeSeverity(finding.Severity), Detail: finding.Detail,
			Evidence: finding.Detail, SuggestedAction: suggestedActionForFinding(finding),
		})
	}
	return result
}

func bestAIMatch(backend AIProviderFinding, ai []AIProviderFinding, used []bool) int {
	bestIndex, bestScore := -1, 0
	for index, candidate := range ai {
		if used[index] { continue }
		sameRow := backend.Row > 0 && candidate.Row > 0 && candidate.Row == backend.Row
		overlap := tokenOverlap(backend.Title+" "+backend.Detail, candidate.Title+" "+candidate.Detail)
		if !sameRow && overlap < 0.35 { continue }
		score := 0
		if sameRow { score += 6 }
		if strings.EqualFold(strings.TrimSpace(backend.Category), strings.TrimSpace(candidate.Category)) { score += 3 }
		if overlap >= 0.35 { score += 4 }
		if score > bestScore { bestScore, bestIndex = score, index }
	}
	if bestScore < 6 { return -1 }
	return bestIndex
}

func tokenOverlap(a, b string) float64 {
	left, right := tokenSet(a), tokenSet(b)
	if len(left) == 0 || len(right) == 0 { return 0 }
	intersection := 0
	for token := range left { if _, ok := right[token]; ok { intersection++ } }
	return float64(intersection) / float64(len(left)+len(right)-intersection)
}

func tokenSet(value string) map[string]struct{} {
	result := map[string]struct{}{}
	for _, token := range strings.Fields(strings.ToLower(value)) {
		token = strings.Trim(token, ".,:;!?()[]{}\"'—-_")
		if len([]rune(token)) >= 3 { result[token] = struct{}{} }
	}
	return result
}

func chooseRow(a, b int) int { if a > 0 { return a }; return b }
func chooseNonEmpty(a, b string) string { if strings.TrimSpace(a) != "" { return strings.TrimSpace(a) }; return strings.TrimSpace(b) }
func mergeDetails(a, b string) string { if strings.TrimSpace(a) == strings.TrimSpace(b) { return strings.TrimSpace(a) }; return strings.TrimSpace(a)+" AI: "+strings.TrimSpace(b) }
func maxSeverity(a, b string) string { if severityRank(b) > severityRank(a) { return normalizeSeverity(b) }; return normalizeSeverity(a) }
