package auth

import (
	"crypto/rand"
	"encoding/hex"
)

func GenerateToken(byteCount int) (string, error) {
	bytes := make([]byte, byteCount)

	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}
