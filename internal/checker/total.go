package checker

import "github.com/NursultanKoshoev11/SmetaCheck/internal/domain"

func totalRule(r domain.EstimateRow) []domain.Issue{
 if r.Total<0{return []domain.Issue{makeIssue(r.Number,domain.Critical,"bad_total","Total is negative","Check total amount.")}}
 return nil
}
