package api

import(
 "io"
 "mime/multipart"
 "os"
 "path/filepath"
)

func SaveFile(f multipart.File,h *multipart.FileHeader)(string,error){
 dir:="data/uploads"
 if err:=os.MkdirAll(dir,0750);err!=nil{return "",err}
 p:=filepath.Join(dir,filepath.Base(h.Filename))
 out,err:=os.Create(p);if err!=nil{return "",err}
 defer out.Close()
 _,err=io.Copy(out,f)
 return p,err
}
