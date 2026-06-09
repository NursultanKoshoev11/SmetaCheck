package checker

import(
 "testing"
 "github.com/NursultanKoshoev11/SmetaCheck/internal/domain"
)

func TestFormulaRule(t *testing.T){
 r:=domain.EstimateRow{Number:2,Name:"A",Quantity:2,UnitPrice:10,Total:30}
 if len(formulaRule(r))==0{t.Fail()}
}

func TestPriceRule(t *testing.T){
 r:=domain.EstimateRow{Number:3,Name:"A",Quantity:1,UnitPrice:-1}
 if len(priceRule(r))==0{t.Fail()}
}
