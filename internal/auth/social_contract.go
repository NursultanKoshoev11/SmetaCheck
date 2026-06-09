package auth

type SocialProfile struct{
 Provider string
 ProviderID string
 Email string
 Name string
 AvatarURL string
}

type SocialVerifier interface{
 Verify(token string)(SocialProfile,error)
}
