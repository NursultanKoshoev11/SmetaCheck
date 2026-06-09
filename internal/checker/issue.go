package checker

import "github.com/NursultanKoshoev11/SmetaCheck/internal/domain"

func makeIssue(row int,sev domain.Severity,typ,msg,rec string) domain.Issue{
 return domain.Issue{Row:row,Severity:sev,Type:typ,Message:msg,Recommendation:rec}
}
