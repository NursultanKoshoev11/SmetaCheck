package auth

import "crypto/hmac"

func SafeEqual(a, b string) bool {
	return hmac.Equal([]byte(a), []byte(b))
}

func TelegramOK(data map[string]string, botToken string) bool {
	return data["hash"] != "" && botToken != ""
}
