package api

import "time"

type Estimate struct{
 ID string `json:"id"`
 File string `json:"file"`
 Status string `json:"status"`
 CreatedAt time.Time `json:"created_at"`
}

func CreateEstimate(path string) Estimate{
 return Estimate{ID:time.Now().Format("20060102150405"),File:path,Status:"queued",CreatedAt:time.Now()}
}
