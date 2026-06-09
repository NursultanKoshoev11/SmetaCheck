package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

func TelegramOK(data map[string]string, botToken string) bool {
	hash := data["hash"]
	delete(data, "hash")
	keys := make([]string, 0, len(data))
	for k := range data { keys = append(keys, k) }
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys { parts = append(parts, k+"="+data[k]) }
	secret := sha256.Sum256([]byte(botToken))
	mac := hmac.New(sha256.New, secret[:])
	mac.Write([]byte(strings.Join(parts, "\n