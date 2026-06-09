package auth

import "time"

type Session struct{
 UserID string `json:"user_id"`
 AccessToken string `json:"access_token"`
 RefreshToken string `json:"refresh_token"`
 ExpiresAt time.Time `json:"expires_at"`
}
