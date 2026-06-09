package api

import(
 "net/http"
 "github.com/NursultanKoshoev11/SmetaCheck/internal/storage"
)

func Estimates(s *storage.Store,dir string)http.HandlerFunc{
 return func(w http.ResponseWriter,r *http.Request){
  if r.Method!=http.MethodPost{Error(w,405,"method");return}
  JSON(w,202,EstimateResponse{ID:"pending",Status:"queued"})
 }
}
