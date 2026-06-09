package report

import "time"

type HistoryItem struct{
 ID string `json:"id"`
 FileName string `json:"file_name"`
 Status string `json:"status"`
 CreatedAt time.Time `json:"created_at"`
}
