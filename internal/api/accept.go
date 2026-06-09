package api

import(
 "errors"
 "net/http"
 "github.com/NursultanKoshoev11/SmetaCheck/internal/storage"
)

func Accept(r *http.Request,s *storage.Store,dir string)(string,error){
 if r.ParseMultipartForm(32<<20)!=nil{return "",errors.New("bad multipart")}
 f,h,e:=r.FormFile("file");if e!=nil{return "",errors.New("file required")}
 defer f.Close()
 _,e=SaveFile(dir,h.Filename,f);if e!=nil{return "",e}
 return s.CreateEstimate(r.Context(),h.Filename)
}
