package api

import(
 "net/http"
 "github.com/NursultanKoshoev11/SmetaCheck/internal/storage"
)

func Estimates(s *storage.Store,dir string)http.HandlerFunc{
 return func(w http.ResponseWriter,r *http.Request){
  if r.Method!=http.MethodPost{Error(w,405,"method");return}
  if err:=r.ParseMultipartForm(25<<20);err!=nil{Error(w,400,"bad form");return}
  _,h,err:=r.FormFile("file")
  if err!=nil{Error(w,400,"file required");return}
  if err=Validate