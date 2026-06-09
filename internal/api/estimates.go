package api

import(
 "net/http"
 "github.com/NursultanKoshoev11/SmetaCheck/internal/storage"
)

func Estimates(s *storage.Store,dir string)http.HandlerFunc{
 return func(w http.ResponseWriter,r *http.Request){
  if r.Method==http.MethodPost{Upload(w,r,s,dir);return}
  JSON(w,200,map[string]string{"status":"ok"})
 }
}
