package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	FullName     string    `json:"full_name"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

type authResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

var userStoreMu sync.Mutex

func AuthRegister(w http.ResponseWriter, r *http.Request) {
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

	users, err := loadUsersLocked()
	if err != nil {
		estimateWriteError(w, http.StatusInternalServerError, "cannot load users")
		return
	}
	for _, user := range users {
		if user.Email == req.Email {
			estimateWriteError(w, http.StatusConflict, "user already exists")
			return
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		estimateWriteError(w, http.StatusInternalServerError, "cannot hash password")
		return
	}
	user := User{ID: newAuthID(), Email: req.Email, FullName: req.FullName, PasswordHash: string(hash), CreatedAt: time.Now().UTC()}
	users = append(users, user)
	if err := writeUsersLocked(users); err != nil {
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

func AuthLogin(w http.ResponseWriter, r *http.Request) {
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

	users, err := loadUsersLocked()
	if err != nil {
		estimateWriteError(w, http.StatusInternalServerError, "cannot load users")
		return
	}
	for _, user := range users {
		if user.Email == req.Email {
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
			return
		}
	}
	estimateWriteError(w, http.StatusUnauthorized, "invalid email or password")
}

func loadUsersLocked() ([]User, error) {
	userStoreMu.Lock()
	defer userStoreMu.Unlock()
	path := userStorePath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return []User{}, nil
	}
	if err != nil {
		return nil, err
	}
	var users []User
	if err := json.Unmarshal(data, &users); err != nil {
		return nil, err
	}
	return users, nil
}

func writeUsersLocked(users []User) error {
	path := userStorePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o640)
}

func userStorePath() string {
	return filepath.Join(estimateReportDir(), "users.json")
}

func createAuthToken(user User) (string, error) {
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if secret == "" {
		secret = "development_only_smetacheck_jwt_secret_replace_in_production_64_chars_minimum"
	}
	claims := jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"name":  user.FullName,
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

func newAuthID() string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "usr_" + hex.EncodeToString([]byte(time.Now().Format("20060102150405")))
	}
	return "usr_" + hex.EncodeToString(buf)
}
