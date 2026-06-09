package api

import(
 "log"
 "net/http"
 "os"
)

func Run(){
 mux:=http.NewServeMux()
 mux.HandleFunc("/health",Health)
 mux.HandleFunc("/v1/auth/register",AuthRegister)
 mux.HandleFunc("/v1/auth/login",AuthLogin)
 addr:=os.Getenv("HTTP_ADDR")
 if addr==""{addr=":8080"}
 log.Fatal(http.ListenAndServe(addr,mux))
}
