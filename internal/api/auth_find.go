package api

import(
 "context"
 "github.com/jackc/pgx/v5/pgxpool"
)

func findUser(ctx context.Context,db *pgxpool.Pool,email string)(authUser,error){
 var u authUser
 q:="select id,email,password_hash from users where email=$1"
 err:=db.QueryRow(ctx,q,normEmail(email)).Scan(&u.ID,&u.Email,&u.Hash)
 return u,err
}
