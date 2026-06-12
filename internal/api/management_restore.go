package api

import "net/http"

func AdminRestoreUser(w http.ResponseWriter, r *http.Request, targetUserID string) {
	admin, ok := requireAdministrator(w, r)
	if !ok { return }
	pool, err := getDB(r.Context())
	if err != nil||pool==nil { estimateWriteError(w,http.StatusServiceUnavailable,"postgresql is unavailable"); return }
	command,err:=pool.Exec(r.Context(),`UPDATE users SET disabled_at=NULL,failed_login_attempts=0,locked_until=NULL,updated_at=now() WHERE id=$1`,targetUserID)
	if err!=nil||command.RowsAffected()==0 { estimateWriteError(w,http.StatusNotFound,"user not found"); return }
	_ = writeAuditLog(r.Context(),r,admin.ID,"admin.user.restored","user",targetUserID,nil)
	estimateWriteJSON(w,http.StatusOK,map[string]any{"user_id":targetUserID,"restored":true})
}
