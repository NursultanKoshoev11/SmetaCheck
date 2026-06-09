package parser

import "github.com/xuri/excelize/v2"

func Rows(path string)([][]string,error){
 f,e:=excelize.OpenFile(path);if e!=nil{return nil,e}
 defer f.Close()
 sh:=f.GetSheetName(0)
 return f.GetRows(sh)
}
