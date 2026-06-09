package api

import "net/http"

func Organizations(w http.ResponseWriter,r *http.Request){
 JSON(w,http.StatusOK,map[string]string{"status":"ready"})
}
