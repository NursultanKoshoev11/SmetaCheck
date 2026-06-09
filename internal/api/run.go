package api

import (
 "log"
 "net/http"
)

func Run(){
 mux:=http.NewServeMux()
 mux.HandleFunc("/health",Health)
 log.Fatal(http.ListenAndServe(":8080",mux))
}
