package auth

type EmailLogin struct {
	Email string `json:"email"`
	Password string `json:"password"`
}

type AuthResult struct {
	UserID string `json:"user_id"`
	Token string `json:"token"`
}
