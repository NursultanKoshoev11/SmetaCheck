package admin

type Summary struct{
 Users int `json:"users"`
 Payments int `json:"payments"`
 Files int `json:"files"`
 Errors int `json:"errors"`
}
