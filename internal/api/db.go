package api

import(
 "context"
 "os"
 "sync"
 "github.com/jackc/pgx/v5/pgxpool"
)

var poolOnce sync.Once
var poolVal *pgxpool.Pool
var poolErr error

func DB(ctx context.Context)(*pgxpool.Pool,error){
 poolOnce.Do(func(){
  poolVal,poolErr=pgxpool.New(ctx,os.Getenv("DATABASE_URL"))
 })
 return poolVal,poolErr
}
