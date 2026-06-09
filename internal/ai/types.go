package ai

import "github.com/NursultanKoshoev11/SmetaCheck/internal/domain"

type ReportInput struct{
 FileName string `json:"file_name"`
 Rows []domain.EstimateRow `json:"rows"`
 Issues []domain.Issue `json:"issues"`
}

type ReportOutput struct{
 Summary string `json:"summary"`
 Recommendations []string `json:"recommendations"`
}
