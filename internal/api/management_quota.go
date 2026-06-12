package api

import (
	"encoding/json"
	"net/http"
)

type quotaChangeRequest struct { QuotaFiles int `json:"quota_files"` }

func AdminChangeQuota(w http.ResponseWriter, r *http.Request, targetUserID string) {
	admin, ok := requireAdministrator(w, r)
	if !ok { return }
	var request quotaChangeRequest
	if err := json.NewDecoder(http.MaxBytesReader(w,r.Body,4096)).Decode(&request); err != nil { estimateWriteError(w,http.StatusBadRequest,"invalid request"); return }
	if request.QuotaFiles<0||request.QuotaFiles>100000 { estimateWriteError(w,http.StatusBadRequest,"invalid quota"); return }
	pool, err := getDB(r.Context())
	if err != nil||pool==nil { estimateWriteError(w,http.StatusServiceUnavailable,"postgresql is unavailable"); return }
	command,err:=pool.Exec(r.Context(),`UPDATE users SET quota_files=$1,updated_at=now() WHERE id=$2`,request.QuotaFiles,targetUserID)
	if err!=nil||command.RowsAffected()==0 { estimateWriteError(w,http.StatusNotFound,"user not found"); return }
	_ = writeAuditLog(r.Context(),r,admin.ID,"admin.user.quota_changed","user",targetUserID,map[string]any{"quota_files":request.QuotaFiles})
	estimateWriteJSON(w,http.StatusOK,map[string]any{"user_id":targetUserID,"quota_files":request.QuotaFiles})
}
