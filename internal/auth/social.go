package auth

type GoogleLogin struct {
	IDToken string `json:"id_token"`
}

type TelegramLogin struct {
	Payload map[string]string `json:"payload"`
}
