package checker

import "math"

func closeMoney(a,b float64) bool{
 return math.Abs(a-b) <= 0.01
}
