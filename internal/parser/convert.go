package parser

import "github.com/NursultanKoshoev11/SmetaCheck/internal/domain"

func ToEstimateRows(src [][]string)[]domain.EstimateRow{
 out:=[]domain.EstimateRow{}
 for i,r:=range src{if len(r)<5{continue};out=append(out,domain.EstimateRow{RowNumber:i+1,Name:r[0],Unit:r[1]})}
 return out
}
