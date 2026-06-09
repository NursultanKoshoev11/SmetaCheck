package report

import "github.com/NursultanKoshoev11/SmetaCheck/internal/domain"

type ChartPoint struct{ Label string `json:"label"`; Count int `json:"count"` }

func CountBySeverity(items []domain.Issue)[]ChartPoint{
 m:=map[string]int{}
 for _,x:=range items{m[x.Severity]++}
 out:=[]ChartPoint{}
 for k,v:=range m{out=append(out,ChartPoint{k,v})}
 return out
}
