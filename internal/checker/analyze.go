package checker

import "github.com/NursultanKoshoev11/SmetaCheck/internal/domain"

func Analyze(rows []domain.EstimateRow) []domain.Issue{
 out:=[]domain.Issue{}
 for _,r:=range rows{
  if r.Name==""{out=append(out,makeIssue(r.Number,domain.Critical,"empty_name","Work name is empty","Fill the work name."))}
  if r.Quantity<=0{out=append(out,makeIssue(r.Number,domain.Critical,"bad_quantity","Quantity must be positive","Check quantity."))}
 }
 return out
}
