package api

import(
 "net/http"
 "github.com/NursultanKoshoev11/SmetaCheck/internal/storage"
)

func Estimates(s *storage.Store,dir string)http.HandlerFunc{
 return func(w http.ResponseWriter,r *http.Request){
  if r.Method!=http.MethodPost{Error(w,405,"method not allowed");return}
  f,h,err:=r.FormFile("file");if err!=nil{Error(w,400,"file is required");return}
  defer f.Close()
  p,err:=SaveFile(f,h);if err!=nil{Error(w,500,"