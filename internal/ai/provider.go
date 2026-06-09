package ai

import "context"

type Provider interface{
 GenerateReport(ctx context.Context,in ReportInput)(ReportOutput,error)
}
