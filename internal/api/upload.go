package api

import(
 "net/http"
 "github.com/NursultanKoshoev11/SmetaCheck/internal/storage"
)

func Upload(w http.ResponseWriter,r *http.Request,s *storage.Store,dir string){
 id,err:=Accept(r,s,dir)
 if err!=nil{Error(w,400,err.Error());return}
 JSON(w,201,map[string]string{"id":id})
}
