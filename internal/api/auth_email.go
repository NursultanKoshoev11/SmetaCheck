package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

const (
	passwordHashCost = 12
	maxLoginAttempts = 5
	loginLockDuration = 15 * time.Minute
)

type tokenRequest struct { Token string `json:"token"` }
type passwordResetRequest struct { Token string `json:"token"`; Password string `json:"password"` }
type emailRequest struct { Email string `json:"email"` }

func AuthRegisterEmail(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := decodeJSONBody(r, &req); err != nil { estimateWriteError(w,http.StatusBadRequest,err.Error()); return }
	email, err := normalizeEmail(req.Email)
	if err != nil { estimateWriteError(w,http.StatusBadRequest,"valid email is required"); return }
	req.FullName = strings.TrimSpace(req.FullName)
	if len(req.FullName)<2 || len(req.FullName)>120 { estimateWriteError(w,http.StatusBadRequest,"full name must be between 2 and 120 characters"); return }
	if err := validatePassword(req.Password); err != nil { estimateWriteError(w,http.StatusBadRequest,err.Error()); return }

	pool, err := getDB(r.Context())
	if err != nil || pool == nil { estimateWriteError(w,http.StatusServiceUnavailable,"postgresql is unavailable"); return }
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password),passwordHashCost)
	if err != nil { estimateWriteError(w,http.StatusInternalServerError,"cannot secure password"); return }

	user := User{ID:newDatabaseID("usr"),Email:email,FullName:req.FullName,PasswordHash:string(hash),CreatedAt:time.Now().UTC()}
	tx, err := pool.Begin(r.Context())
	if err != nil { estimateWriteError(w,http.StatusInternalServerError,"cannot start registration"); return }
	defer tx.Rollback(r.Context())

	_, err = tx.Exec(r.Context(),`INSERT INTO users (id,email,password_hash,full_name,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$5)`,user.ID,user.Email,user.PasswordHash,user.FullName,user.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err,&pgErr) && pgErr.Code=="23505" { estimateWriteError(w,http.StatusConflict,"user already exists"); return }
		estimateWriteError(w,http.StatusInternalServerError,"cannot save user"); return
	}
	_, err = tx.Exec(r.Context(),`INSERT INTO auth_identities (id,user_id,provider,provider_subject,provider_email) VALUES ($1,$2,'email',$3,$3)`,newDatabaseID("idn"),user.ID,user.Email)
	if err != nil { estimateWriteError(w,http.StatusInternalServerError,"cannot save email identity"); return }
	token, err := createOneTimeTokenTx(r.Context(),tx,user.ID,"verify_email",24*time.Hour)
	if err != nil { estimateWriteError(w,http.StatusInternalServerError,"cannot create verification token"); return }
	if err := tx.Commit(r.Context()); err != nil { estimateWriteError(w,http.StatusInternalServerError,"cannot complete registration"); return }
	if err := sendVerificationEmail(user,token); err != nil { estimateWriteError(w,http.StatusServiceUnavailable,"account created but verification email could not be sent; use resend verification"); return }
	estimateWriteJSON(w,http.StatusCreated,map[string]any{"user":user,"verification_required":true})
}

func AuthLoginEmail(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := decodeJSONBody(r,&req); err != nil { estimateWriteError(w,http.StatusBadRequest,err.Error()); return }
	email, err := normalizeEmail(req.Email)
	if err != nil || req.Password=="" { estimateWriteError(w,http.StatusBadRequest,"email and password are required"); return }
	pool, err := getDB(r.Context())
	if err != nil || pool==nil { estimateWriteError(w,http.StatusServiceUnavailable,"postgresql is unavailable"); return }

	var user User
	var failed int
	var lockedUntil *time.Time
	err = pool.QueryRow(r.Context(),`SELECT id,COALESCE(email,''),COALESCE(full_name,''),COALESCE(avatar_url,''),COALESCE(password_hash,''),email_verified_at,failed_login_attempts,locked_until,created_at FROM users WHERE lower(email)=lower($1)`,email).
		Scan(&user.ID,&user.Email,&user.FullName,&user.AvatarURL,&user.PasswordHash,&user.EmailVerifiedAt,&failed,&lockedUntil,&user.CreatedAt)
	if errors.Is(err,pgx.ErrNoRows) {
		_ = bcrypt.CompareHashAndPassword([]byte("$2a$12$Qqn75TzxV9QYbWj7EThJHesjx78XjI3Q4F6KcQVrkM3QwTuHZnM5e"),[]byte(req.Password))
		estimateWriteError(w,http.StatusUnauthorized,"invalid email or password"); return
	}
	if err != nil { estimateWriteError(w,http.StatusInternalServerError,"cannot load user"); return }
	if lockedUntil!=nil && lockedUntil.After(time.Now().UTC()) { estimateWriteError(w,http.StatusTooManyRequests,"account is temporarily locked; try again later"); return }
	if user.PasswordHash=="" || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash),[]byte(req.Password))!=nil {
		failed++
		if failed>=maxLoginAttempts { _,_ = pool.Exec(r.Context(),`UPDATE users SET failed_login_attempts=0,locked_until=$1,updated_at=now() WHERE id=$2`,time.Now().UTC().Add(loginLockDuration),user.ID) } else { _,_ = pool.Exec(r.Context(),`UPDATE users SET failed_login_attempts=$1,updated_at=now() WHERE id=$2`,failed,user.ID) }
		estimateWriteError(w,http.StatusUnauthorized,"invalid email or password"); return
	}
	if user.EmailVerifiedAt==nil { estimateWriteError(w,http.StatusForbidden,"email verification is required"); return }
	_,_ = pool.Exec(r.Context(),`UPDATE users SET failed_login_attempts=0,locked_until=NULL,last_login_at=now(),updated_at=now() WHERE id=$1`,user.ID)
	if err := createBrowserSession(w,r,user); err != nil { estimateWriteError(w,http.StatusInternalServerError,"cannot create session"); return }
	estimateWriteJSON(w,http.StatusOK,authResponse{User:user})
}

func AuthVerifyEmail(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token=="" { estimateWriteError(w,http.StatusBadRequest,"verification token is required"); return }
	userID, err := consumeOneTimeToken(r.Context(),"verify_email",token)
	if err != nil { estimateWriteError(w,http.StatusBadRequest,"verification link is invalid or expired"); return }
	pool,_ := getDB(r.Context())
	_,err = pool.Exec(r.Context(),`UPDATE users SET email_verified_at=COALESCE(email_verified_at,now()),updated_at=now() WHERE id=$1`,userID)
	if err != nil { estimateWriteError(w,http.StatusInternalServerError,"cannot verify email"); return }
	redirectAuthResult(w,r,"/login?verified=1")
}

func AuthResendVerification(w http.ResponseWriter, r *http.Request) {
	var req emailRequest
	if err := decodeJSONBody(r,&req); err==nil {
		if email,normalizeErr := normalizeEmail(req.Email); normalizeErr==nil {
			pool,_ := getDB(r.Context())
			var user User
			err := pool.QueryRow(r.Context(),`SELECT id,email,COALESCE(full_name,''),COALESCE(avatar_url,''),email_verified_at,created_at FROM users WHERE lower(email)=lower($1)`,email).
				Scan(&user.ID,&user.Email,&user.FullName,&user.AvatarURL,&user.EmailVerifiedAt,&user.CreatedAt)
			if err==nil && user.EmailVerifiedAt==nil { if token,tokenErr:=createOneTimeToken(r.Context(),user.ID,"verify_email",24*time.Hour); tokenErr==nil { _=sendVerificationEmail(user,token) } }
		}
	}
	estimateWriteJSON(w,http.StatusAccepted,map[string]bool{"accepted":true})
}

func AuthForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req emailRequest
	if err := decodeJSONBody(r,&req); err==nil {
		if email,normalizeErr:=normalizeEmail(req.Email); normalizeErr==nil {
			pool,_:=getDB(r.Context())
			var user User
			err:=pool.QueryRow(r.Context(),`SELECT id,email,COALESCE(full_name,''),COALESCE(avatar_url,''),email_verified_at,created_at FROM users WHERE lower(email)=lower($1) AND password_hash IS NOT NULL`,email).
				Scan(&user.ID,&user.Email,&user.FullName,&user.AvatarURL,&user.EmailVerifiedAt,&user.CreatedAt)
			if err==nil { if token,tokenErr:=createOneTimeToken(r.Context(),user.ID,"reset_password",time.Hour); tokenErr==nil { _=sendPasswordResetEmail(user,token) } }
		}
	}
	estimateWriteJSON(w,http.StatusAccepted,map[string]bool{"accepted":true})
}

func AuthResetPassword(w http.ResponseWriter, r *http.Request) {
	var req passwordResetRequest
	if err:=decodeJSONBody(r,&req); err!=nil { estimateWriteError(w,http.StatusBadRequest,err.Error()); return }
	if err:=validatePassword(req.Password); err!=nil { estimateWriteError(w,http.StatusBadRequest,err.Error()); return }
	userID,err:=consumeOneTimeToken(r.Context(),"reset_password",strings.TrimSpace(req.Token))
	if err!=nil { estimateWriteError(w,http.StatusBadRequest,"reset link is invalid or expired"); return }
	hash,err:=bcrypt.GenerateFromPassword([]byte(req.Password),passwordHashCost)
	if err!=nil { estimateWriteError(w,http.StatusInternalServerError,"cannot secure password"); return }
	pool,_:=getDB(r.Context())
	_,err=pool.Exec(r.Context(),`UPDATE users SET password_hash=$1,failed_login_attempts=0,locked_until=NULL,updated_at=now() WHERE id=$2`,string(hash),userID)
	if err!=nil { estimateWriteError(w,http.StatusInternalServerError,"cannot reset password"); return }
	revokeAllUserSessions(r,userID)
	clearSessionCookies(w)
	estimateWriteJSON(w,http.StatusOK,map[string]bool{"password_reset":true})
}

func createOneTimeToken(ctx context.Context,userID,purpose string,ttl time.Duration)(string,error){
	pool,err:=getDB(ctx); if err!=nil{return "",err}
	token,err:=randomURLToken(32); if err!=nil{return "",err}
	_,err=pool.Exec(ctx,`DELETE FROM auth_tokens WHERE user_id=$1 AND purpose=$2 AND consumed_at IS NULL`,userID,purpose); if err!=nil{return "",err}
	_,err=pool.Exec(ctx,`INSERT INTO auth_tokens (id,user_id,purpose,token_hash,expires_at) VALUES ($1,$2,$3,$4,$5)`,newDatabaseID("atk"),userID,purpose,hashToken(token),time.Now().UTC().Add(ttl))
	return token,err
}

func createOneTimeTokenTx(ctx context.Context,tx pgx.Tx,userID,purpose string,ttl time.Duration)(string,error){
	token,err:=randomURLToken(32); if err!=nil{return "",err}
	_,err=tx.Exec(ctx,`INSERT INTO auth_tokens (id,user_id,purpose,token_hash,expires_at) VALUES ($1,$2,$3,$4,$5)`,newDatabaseID("atk"),userID,purpose,hashToken(token),time.Now().UTC().Add(ttl))
	return token,err
}

func consumeOneTimeToken(ctx context.Context,purpose,token string)(string,error){
	if token==""{return "",fmt.Errorf("token is required")}
	pool,err:=getDB(ctx); if err!=nil{return "",err}
	var userID string
	err=pool.QueryRow(ctx,`UPDATE auth_tokens SET consumed_at=now() WHERE token_hash=$1 AND purpose=$2 AND consumed_at IS NULL AND expires_at>now() RETURNING user_id`,hashToken(token),purpose).Scan(&userID)
	return userID,err
}

func validatePassword(password string)error{ if len(password)<12{return fmt.Errorf("password must be at least 12 characters")}; if len(password)>128{return fmt.Errorf("password must be at most 128 characters")}; return nil }
func normalizeEmail(value string)(string,error){ value=strings.ToLower(strings.TrimSpace(value)); address,err:=mail.ParseAddress(value); if err!=nil||strings.ToLower(address.Address)!=value||len(value)>254{return "",fmt.Errorf("invalid email")}; return value,nil }

func decodeJSONBody(r *http.Request,target any)error{
	reader:=io.LimitReader(r.Body,64*1024)
	decoder:=json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err:=decoder.Decode(target); err!=nil{return fmt.Errorf("invalid request body")}
	var extra any
	if err:=decoder.Decode(&extra); err!=io.EOF{return fmt.Errorf("request body must contain one JSON object")}
	return nil
}

func redirectAuthResult(w http.ResponseWriter,r *http.Request,path string){ base:=strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL")),"/"); if base==""{estimateWriteJSON(w,http.StatusOK,map[string]bool{"ok":true});return}; http.Redirect(w,r,base+path,http.StatusSeeOther) }
