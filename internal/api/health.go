package api

import "net/http"

func Health(w http.ResponseWriter,r *http.Request){
 w.Header().Set("Content-Type","application/json")
 w.WriteHeader(http.StatusOK)
 _,_=w.Write([]byte(`{"ok":true,"service":"api"}`))
}
