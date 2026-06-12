package api

import "net/http"

type auditResponseWriter struct {
	http.ResponseWriter
	status int
}

func (writer *auditResponseWriter) WriteHeader(status int) {
	writer.status=status
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *auditResponseWriter) Write(data []byte) (int,error) {
	if writer.status==0 { writer.status=http.StatusOK }
	return writer.ResponseWriter.Write(data)
}

func EstimateRouterAudited(w http.ResponseWriter, r *http.Request) {
	user, authenticated := currentRequestUser(r)
	writer:=&auditResponseWriter{ResponseWriter:w}
	EstimateRouterWithReportLimit(writer,r)
	if !authenticated||writer.status<200||writer.status>=300 { return }
	resourceID:=estimateIDFromPath(r.URL.Path)
	if resourceID=="" { return }
	switch r.Method {
	case http.MethodGet:
		_ = writeAuditLog(r.Context(),r,user.ID,"estimate.data_viewed","estimate",resourceID,nil)
	case http.MethodDelete:
		_ = writeAuditLog(r.Context(),r,user.ID,"estimate.file_deleted","estimate",resourceID,nil)
	}
}

func estimateIDFromPath(path string) string {
	trimmed:=path
	prefix:="/v1/estimates/"
	if len(trimmed)<len(prefix)||trimmed[:len(prefix)]!=prefix { return "" }
	trimmed=trimmed[len(prefix):]
	for index,value:=range trimmed {
		if value=='/' { return trimmed[:index] }
	}
	return trimmed
}
