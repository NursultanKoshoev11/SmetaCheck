package api

import "net/http"

func AuthRegister(w http.ResponseWriter,r *http.Request){
 if r.Method!="POST"{Error(w,http.StatusMethodNotAllowed,"method not allowed");return}
 JSON(w,http.StatusAccepted,map[string]string{"status":"auth_contract_ready"})
}

func AuthLogin(w http.ResponseWriter,r *http.Request){
 if r.Method!="POST"{Error(w,http.StatusMethodNotAllowed,"method not allowed");return}
 JSON(w,http.StatusAccepted,map[string]string{"status":"auth_contract