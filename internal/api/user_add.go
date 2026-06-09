package api

import(
 "context"
 "github.com/jackc/pgx/v5/pgxpool"
)

func addUser(ctx context.Context,db *pgxpool.Pool,email,hash string)(int64,error){
 var id int64
 stmt:="insert into users(email,password_hash) values($1,$2) returning id"
 err:=db.QueryRow(ctx,stmt,normEmail(email),hash).Scan(&id)
 return id,err
}
