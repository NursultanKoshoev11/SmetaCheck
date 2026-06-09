package api

import(
 "io"
 "mime/multipart"
 "os"
 "path/filepath"
)

func SaveFile(dir,name string,src multipart.File) (string,error){
 if err:=os.MkdirAll(dir,0750);err!=nil{return "",err}
 dst:=filepath.Join(dir,filepath.Base(name))
 f,err:=os.Create(dst);if err!=nil{return "",err}
 defer f.Close()
 _,err=io.Copy(f,src)
 return dst,err
}
