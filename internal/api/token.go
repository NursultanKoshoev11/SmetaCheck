package api

import(
 "crypto/rand"
 "encoding/hex"
)

func newToken()string{
 b:=make([]byte,32)
 _,_ = rand.Read(b)
 return hex.EncodeToString(b)
}
