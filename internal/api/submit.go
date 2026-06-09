package api

import(
 "net/http"
 "github.com/NursultanKoshoev11/SmetaCheck/internal/storage"
)

func Estimates(s *storage.Store,dir string)http.HandlerFunc{
 return func(w http.ResponseWriter,r *http.Request){
  if r.Method!=http.MethodPost{Error(w,405,"method");return}
  err:=r.ParseMultipartForm(25<<20)
  if err!=nil{Error(w,400,"bad form");return}
  _,h,err:=r.FormFile("file")
  if err!=nil{Error(w,400,"file required");return}
  if err=ValidateUpload(h);err!=nil{Error(w,400,err.Error());return}
  p,err:=SaveFile(h,dir)
  if err!=nil{Error(w,500,"save failed");return}
  id,err:=s.CreateEstimate(r.Context(),h.Filename,p)
  if