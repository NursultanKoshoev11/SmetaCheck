package api

import "net/http"

func Dashboard(w http.ResponseWriter,r *http.Request){
 JSON(w,http.StatusOK,map[string]string{"status":"ready"})
}
