package xlsx

import "strconv"

func Number(s string) float64{
 v,_:=strconv.ParseFloat(s,64)
 return v
}
