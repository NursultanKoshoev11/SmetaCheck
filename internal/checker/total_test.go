package checker

import(
 "testing"
 "github.com/NursultanKoshoev11/SmetaCheck/internal/domain"
)

func TestTotalRule(t *testing.T){
 r:=domain.EstimateRow{Number:4,Name:"A",Quantity:1,UnitPrice:1,Total:-1}
 if len(totalRule(r))==0{t.Fail()}
}
