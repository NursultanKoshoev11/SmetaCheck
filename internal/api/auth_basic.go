package api

import "net/http"

func AuthRegister(w http.ResponseWriter,r *http.Request){
 if r.Method!="POST"{Error(w,405,"method");return}
 JSON(w,202,map[string]string{"status":"ready"})
}

func AuthLogin(w http.ResponseWriter,r *http.Request){
 if r.Method!="POST"{Error(w,405,"method");return}
 JSON(w,202,map[string]string{"status":"ready"})
}
