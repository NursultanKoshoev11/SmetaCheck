package report

import (
 "context"
 "github.com/NursultanKoshoev11/SmetaCheck/internal/ai"
)

func Explain(ctx context.Context,p ai.Provider,in ai.ReportInput)(ai.ReportOutput,error){
 return p.GenerateReport(ctx,in)
}
