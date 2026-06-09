package auth

import "errors"

var ErrProviderNotConfigured = errors.New("auth provider not configured")
var ErrInvalidCredentials = errors.New("invalid credentials")
