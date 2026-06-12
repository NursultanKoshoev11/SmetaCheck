package api

import (
	"context"
	"encoding/json"
	"net/http"
)

func writeAuditLog(ctx context.Context, r *http.Request, userID, action, resourceType, resourceID string, metadata map[string]any) error {
	pool, err := getDB(ctx)
	if err != nil || pool == nil {
		return err
	}
	payload, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	ipAddress := ""
	userAgent := ""
	if r != nil {
		ipAddress = requestIP(r)
		userAgent = r.UserAgent()
	}
	_, err = pool.Exec(ctx, `INSERT INTO audit_logs (id,user_id,action,resource_type,resource_id,ip_address,user_agent,metadata,outcome) VALUES ($1,NULLIF($2,''),$3,NULLIF($4,''),NULLIF($5,''),NULLIF($6,''),NULLIF($7,''),$8,'success')`, newDatabaseID("aud"), userID, action, resourceType, resourceID, ipAddress, userAgent, payload)
	return err
}
