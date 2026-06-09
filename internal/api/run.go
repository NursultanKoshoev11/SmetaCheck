package api

import(
 "context"
 "log"
 "net/http"
 "github.com/NursultanKoshoev11/SmetaCheck/internal/platform/config"
 pdb "github.com/NursultanKoshoev11/SmetaCheck/internal/platform/db"
 "github.com/NursultanKoshoev11/SmetaCheck/internal/storage"
)

func Run(){
 cfg:=config.Load()
 pool,err:=pdb.Open(context.Background(),cfg.DB)
 if err!=nil{log.Fatal(err)}
 s:=storage.New(pool)
 mux:=http.NewServeMux()
 mux.HandleFunc("/health",Health)
 mux.HandleFunc("/v1/auth/register",AuthRegister)
 mux.HandleFunc("/v1/auth/login",AuthLogin)
 mux.HandleFunc("/v1/estimates