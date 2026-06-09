package checker

import "github.com/NursultanKoshoev11/SmetaCheck/internal/domain"

func priceRule(r domain.EstimateRow) []domain.Issue{
 if r.UnitPrice<0{return []domain.Issue{makeIssue(r.Number,domain.Critical,"bad_price","Unit price is negative","Check unit price.")}}
 return nil
}
