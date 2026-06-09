package api

import(
 "context"
 "strings"
 "github.com/jackc/pgx/v5/pgxpool"
)

type authUser struct{ID int64; Email string; Hash string}

func normEmail(v string)string{return strings.ToLower(strings.TrimSpace(v))}

func findUser(ctx context.Context,db *pgxpool.Pool,email string)(authUser,error){
 var u authUser
 err:=db.QueryRow(ctx,"select id,email,password_hash from users where email=$1",normEmail(email)).Scan(&u.ID,&u.Email,&u.Hash)
 return