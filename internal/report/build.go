package report

import "github.com/NursultanKoshoev11/SmetaCheck/internal/domain"

func Build(items []domain.Issue) Summary{
 s:=Summary{Total:len(items)}
 for _,x:=range items{
  switch x.Severity{
  case domain.Critical:s.Critical++
  case domain.Warning:s.Warning++
  case domain.Info:s.Info++
  }
 }
 return s
}
