package api

import (
	"net/http"
	"strings"
)

func AdminManagementRouter(w http.ResponseWriter, r *http.Request) {
	path:=strings.Trim(strings.TrimPrefix(r.URL.Path,"/v1/admin/"),"/")
	parts:=strings.Split(path,"/")
	if len(parts)!=3||parts[0]!="users"||parts[1]=="" {
		estimateWriteError(w,http.StatusNotFound,"administrative endpoint not found")
		return
	}
	targetUserID:=parts[1]
	switch {
	case parts[2]=="plan"&&r.Method==http.MethodPatch:
		AdminChangePlan(w,r,targetUserID)
	case parts[2]=="quota"&&r.Method==http.MethodPatch:
		AdminChangeQuota(w,r,targetUserID)
	case parts[2]=="restore"&&r.Method==http.MethodPost:
		AdminRestoreUser(w,r,targetUserID)
	default:
		estimateWriteError(w,http.StatusMethodNotAllowed,"administrative action is not allowed")
	}
}
