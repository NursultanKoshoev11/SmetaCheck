package api

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"time"
)

func validTOTP(secret, code string, now time.Time) bool {
	for offset := int64(-1); offset <= 1; offset++ {
		if totpCode(secret, now.Unix()/30+offset) == code { return true }
	}
	return false
}

func totpCode(secret string, counter int64) string {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if err != nil { return "" }
	var message [8]byte
	binary.BigEndian.PutUint64(message[:], uint64(counter))
	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(message[:])
	digest := mac.Sum(nil)
	offset := digest[len(digest)-1] & 0x0f
	value := (uint32(digest[offset])&0x7f)<<24 | uint32(digest[offset+1])<<16 | uint32(digest[offset+2])<<8 | uint32(digest[offset+3])
	return fmt.Sprintf("%06d", value%1000000)
}

func encryptMFASecret(value string) (string, error) {
	block, err := mfaCipher()
	if err != nil { return "", err }
	gcm, err := cipher.NewGCM(block)
	if err != nil { return "", err }
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil { return "", err }
	sealed := gcm.Seal(nonce, nonce, []byte(value), nil)
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}

func decryptMFASecret(value string) (string, error) {
	block, err := mfaCipher()
	if err != nil { return "", err }
	gcm, err := cipher.NewGCM(block)
	if err != nil { return "", err }
	data, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || len(data) < gcm.NonceSize() { return "", fmt.Errorf("invalid encrypted MFA secret") }
	plain, err := gcm.Open(nil, data[:gcm.NonceSize()], data[gcm.NonceSize():], nil)
	if err != nil { return "", err }
	return string(plain), nil
}

func mfaCipher() (cipher.Block, error) {
	keyText := strings.TrimSpace(os.Getenv("MFA_ENCRYPTION_KEY"))
	if keyText == "" { return nil, fmt.Errorf("MFA encryption is not configured") }
	key := sha256.Sum256([]byte(keyText))
	return aes.NewCipher(key[:])
}
