package api

import (
	"encoding/json"
	"net/http"
	"strings"
)

type planChangeRequest struct { Plan string `json:"plan"` }

func AdminChangePlan(w http.ResponseWriter, r *http.Request, targetUserID string) {
	admin, ok := requireAdministrator(w, r)
	if !ok { return }
	var request planChangeRequest
	if err := json.NewDecoder(http.MaxBytesReader(w,r.Body,4096)).Decode(&request); err != nil { estimateWriteError(w,http.StatusBadRequest,"invalid request"); return }
	request.Plan=strings.TrimSpace(request.Plan)
	if request.Plan==""||len(request.Plan)>50 { estimateWriteError(w,http.StatusBadRequest,"invalid plan"); return }
	pool, err := getDB(r.Context())
	if err != nil||pool==nil { estimateWriteError(w,http.StatusServiceUnavailable,"postgresql is unavailable"); return }
	command,err:=pool.Exec(r.Context(),`UPDATE users SET plan=$1,updated_at=now() WHERE id=$2`,request.Plan,targetUserID)
	if err!=nil||command.RowsAffected()==0 { estimateWriteError(w,http.StatusNotFound,"user not found"); return }
	_ = writeAuditLog(r.Context(),r,admin.ID,"admin.user.plan_changed","user",targetUserID,map[string]any{"plan":request.Plan})
	estimateWriteJSON(w,http.StatusOK,map[string]any{"user_id":targetUserID,"plan":request.Plan})
}
