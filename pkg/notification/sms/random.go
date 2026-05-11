package sms

import (
	"crypto/rand"
	"encoding/hex"
)

func randHex(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
