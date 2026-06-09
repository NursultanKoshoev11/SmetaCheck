package checker

import(
 "testing"
 "github.com/NursultanKoshoev11/SmetaCheck/internal/domain"
)

func TestAnalyze(t *testing.T){
 rows:=[]domain.EstimateRow{{Number:1,Name:"",Quantity:0}}
 if len(Analyze(rows))==0{t.Fail()}
}
