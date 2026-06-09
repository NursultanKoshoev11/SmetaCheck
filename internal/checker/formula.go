package checker

import "github.com/NursultanKoshoev11/SmetaCheck/internal/domain"

func formulaRule(r domain.EstimateRow) []domain.Issue{
 if r.Total==0{return nil}
 want:=r.Quantity*r.UnitPrice
 if !closeMoney(want,r.Total){return []domain.Issue{makeIssue(r.Number,domain.Warning,"sum_mismatch","Total does not match quantity and price","Check formula.")}}
 return nil
}
