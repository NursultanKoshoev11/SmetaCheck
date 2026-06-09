package ai

import (
 "context"
 "fmt"
)

type NoopProvider struct{}

func (NoopProvider) GenerateReport(ctx context.Context,in ReportInput)(ReportOutput,error){
 return ReportOutput{},fmt.Errorf("ai provider is not configured")
}
