package auth

type GoogleProfile struct {
	Email string
	Name string
	Sub string
	AvatarURL string
}

func GoogleProfileFromToken(idToken string) (GoogleProfile, error) {
	return GoogleProfile{}, ErrProviderNotConfigured
}
