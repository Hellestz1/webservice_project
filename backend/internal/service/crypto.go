package service

import (
	"crypto/sha256"
	"encoding/hex"
)

func sha256Hex(value string) string {
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])
}
