package api

import "time"

type AIProviderFinding struct {
	FileID          string `json:"file_id"`
	Row             int    `json:"row"`
	Category        string `json:"category"`
	Title           string `json:"title"`
	Severity        string `json:"severity"`
	Detail          string `json:"detail"`
	Evidence        string `json:"evidence"`
	SuggestedAction string `json:"suggested_action"`
}

type AIFileAnalysis struct {
	EstimateID       string              `json:"estimate_id"`
	FileName         string              `json:"file_name"`
	Summary          string              `json:"summary"`
	RiskLevel        string              `json:"risk_level"`
	DataQualityScore int                 `json:"data_quality_score"`
	ExtractedTotal   float64             `json:"extracted_total"`
	RowsAnalyzed     int                 `json:"rows_analyzed"`
	Findings         []AIProviderFinding `json:"findings"`
	Recommendations []string            `json:"recommendations"`
	InputMode        string              `json:"input_mode"`
	Warning          string              `json:"warning,omitempty"`
}

type AIProviderBatchResponse struct {
	Provider        string           `json:"provider"`
	Model           string           `json:"model"`
	PromptVersion   string           `json:"prompt_version"`
	BatchSummary    string           `json:"batch_summary"`
	Files           []AIFileAnalysis `json:"files"`
	Recommendations []string         `json:"recommendations"`
	GeneratedAt     time.Time        `json:"generated_at"`
}

type UnifiedFinding struct {
	FileID          string   `json:"file_id"`
	Row             int      `json:"row"`
	Category        string   `json:"category"`
	Title           string   `json:"title"`
	Severity        string   `json:"severity"`
	Detail          string   `json:"detail"`
	SuggestedAction string   `json:"suggested_action"`
	Sources         []string `json:"sources"`
	Reconciliation  string   `json:"reconciliation"`
}

type UnifiedFileReport struct {
	EstimateID       string              `json:"estimate_id"`
	FileName         string              `json:"file_name"`
	BackendScore     int                 `json:"backend_score"`
	AIRiskLevel      string              `json:"ai_risk_level"`
	FinalRiskLevel   string              `json:"final_risk_level"`
	AgreementScore   int                 `json:"agreement_score"`
	Summary          string              `json:"summary"`
	TotalAmount      float64             `json:"total_amount"`
	AIExtractedTotal float64             `json:"ai_extracted_total"`
	Findings         []UnifiedFinding    `json:"findings"`
	BackendChart     []AIChartPoint      `json:"backend_chart"`
	AIChart          []AIChartPoint      `json:"ai_chart"`
	UnifiedChart     []AIChartPoint      `json:"unified_chart"`
	Recommendations []string            `json:"recommendations"`
	AIAnalysis       AIFileAnalysis      `json:"ai_analysis"`
}

type CrossFileComparison struct {
	Type        string  `json:"type"`
	FileA       string  `json:"file_a"`
	FileB       string  `json:"file_b"`
	Description string  `json:"description"`
	Delta       float64 `json:"delta"`
}

type UnifiedBatchReport struct {
	BatchID          string                `json:"batch_id"`
	Provider         string                `json:"provider"`
	Model            string                `json:"model"`
	Status           string                `json:"status"`
	Summary          string                `json:"summary"`
	OverallRiskLevel string                `json:"overall_risk_level"`
	TotalFiles       int                   `json:"total_files"`
	TotalRows        int                   `json:"total_rows"`
	TotalAmount      float64               `json:"total_amount"`
	Files            []UnifiedFileReport   `json:"files"`
	CrossFile        []CrossFileComparison `json:"cross_file"`
	UnifiedChart     []AIChartPoint        `json:"unified_chart"`
	Recommendations []string               `json:"recommendations"`
	GeneratedAt      time.Time              `json:"generated_at"`
}

type AnalysisBatch struct {
	ID             string              `json:"id"`
	Status         string              `json:"status"`
	Provider       string              `json:"provider"`
	Model          string              `json:"model"`
	FileCount      int                 `json:"file_count"`
	CompletedCount int                 `json:"completed_count"`
	Attempts       int                 `json:"attempts"`
	ErrorMessage   string              `json:"error_message,omitempty"`
	Report         *UnifiedBatchReport `json:"report,omitempty"`
	CreatedAt      time.Time           `json:"created_at"`
	StartedAt      *time.Time          `json:"started_at,omitempty"`
	CompletedAt    *time.Time          `json:"completed_at,omitempty"`
}

type AnalysisBatchFile struct {
	ID         string `json:"id"`
	BatchID    string `json:"batch_id"`
	FileName   string `json:"file_name"`
	FilePath   string `json:"-"`
	MIMEType   string `json:"mime_type"`
	FileSize   int64  `json:"file_size"`
	Status     string `json:"status"`
	EstimateID string `json:"estimate_id,omitempty"`
	Error      string `json:"error,omitempty"`
}
