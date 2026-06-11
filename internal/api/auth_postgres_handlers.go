package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

func AuthRegisterPostgres(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		estimateWriteError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.FullName = strings.TrimSpace(req.FullName)
	if req.Email == "" || !strings.Contains(req.Email, "@") {
		estimateWriteError(w, http.StatusBadRequest, "valid email is required")
		return
	}
	if len(req.Password) < 8 {
		estimateWriteError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	pool, err := getDB(r.Context())
	if err != nil || pool == nil {
		estimateWriteError(w, http.StatusServiceUnavailable, "postgresql is unavailable")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		estimateWriteError(w, http.StatusInternalServerError, "cannot hash password")
		return
	}
	user := User{
		ID: newDatabaseID("usr"), Email: req.Email, FullName: req.FullName,
		PasswordHash: string(hash), CreatedAt: time.Now().UTC(),
	}
	_, err = pool.Exec(r.Context(), `
		INSERT INTO users (id, email, password_hash, full_name, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$5)
	`, user.ID, user.Email, user.PasswordHash, user.FullName, user.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			estimateWriteError(w, http.StatusConflict, "user already exists")
			return
		}
		estimateWriteError(w, http.StatusInternalServerError, "cannot save user")
		return
	}
	token, err := createAuthToken(user)
	if err != nil {
		estimateWriteError(w, http.StatusInternalServerError, "cannot create token")
		return
	}
	estimateWriteJSON(w, http.StatusCreated, authResponse{Token: token, User: user})
}

func AuthLoginPostgres(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		estimateWriteError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" || req.Password == "" {
		estimateWriteError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	pool, err := getDB(r.Context())
	if err != nil || pool == nil {
		estimateWriteError(w, http.StatusServiceUnavailable, "postgresql is unavailable")
		return
	}
	var user User
	err = pool.QueryRow(r.Context(), `
		SELECT id, email, COALESCE(full_name,''), password_hash, created_at
		FROM users WHERE email = $1
	`, req.Email).Scan(&user.ID, &user.Email, &user.FullName, &user.PasswordHash, &user.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		estimateWriteError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	if err != nil {
		estimateWriteError(w, http.StatusInternalServerError, "cannot load user")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		estimateWriteError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	token, err := createAuthToken(user)
	if err != nil {
		estimateWriteError(w, http.StatusInternalServerError, "cannot create token")
		return
	}
	estimateWriteJSON(w, http.StatusOK, authResponse{Token: token, User: user})
}

func AuthMe(w http.ResponseWriter, r *http.Request) {
	user, ok := currentRequestUser(r)
	if !ok {
		estimateWriteError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	estimateWriteJSON(w, http.StatusOK, map[string]any{"user": user})
}
