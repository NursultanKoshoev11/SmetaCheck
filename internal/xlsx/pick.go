package xlsx

func Pick(r []string,i int) string{
 if i<0||i>=len(r){return ""}
 return r[i]
}
