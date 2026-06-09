package xlsx

import "github.com/xuri/excelize/v2"

func Open(path string)([][]string,error){
 f,err:=excelize.OpenFile(path)
 if err!=nil{return nil,err}
 defer f.Close()
 sheets:=f.GetSheetList()
 if len(sheets)==0{return nil,nil}
 return f.GetRows(sheets[0])
}
