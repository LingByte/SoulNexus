package access

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const DefaultRecoveryCodeCount = 8

// GenerateRecoveryCodes returns one-time backup codes (plain text, show once to user).
func GenerateRecoveryCodes(count int) ([]string, error) {
	if count <= 0 {
		count = DefaultRecoveryCodeCount
	}
	out := make([]string, 0, count)
	for len(out) < count {
		n, err := rand.Int(rand.Reader, big.NewInt(900000000000))
		if err != nil {
			return nil, err
		}
		code := fmt.Sprintf("%04d-%04d", n.Int64()/10000, n.Int64()%10000)
		out = append(out, code)
	}
	return out, nil
}

func normalizeRecoveryCode(code string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(code), " ", ""))
}

// HashRecoveryCodes stores bcrypt hashes as JSON array string.
func HashRecoveryCodes(codes []string) (string, error) {
	hashes := make([]string, 0, len(codes))
	for _, code := range codes {
		h, err := bcrypt.GenerateFromPassword([]byte(normalizeRecoveryCode(code)), bcrypt.DefaultCost)
		if err != nil {
			return "", err
		}
		hashes = append(hashes, string(h))
	}
	b, err := json.Marshal(hashes)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ConsumeRecoveryCode validates code against stored hashes JSON; returns updated hashes JSON.
func ConsumeRecoveryCode(code, hashesJSON string) (string, bool) {
	code = normalizeRecoveryCode(code)
	if code == "" || strings.TrimSpace(hashesJSON) == "" {
		return hashesJSON, false
	}
	var hashes []string
	if err := json.Unmarshal([]byte(hashesJSON), &hashes); err != nil || len(hashes) == 0 {
		return hashesJSON, false
	}
	for i, h := range hashes {
		if bcrypt.CompareHashAndPassword([]byte(h), []byte(code)) != nil {
			continue
		}
		hashes = append(hashes[:i], hashes[i+1:]...)
		b, err := json.Marshal(hashes)
		if err != nil {
			return hashesJSON, true
		}
		return string(b), true
	}
	return hashesJSON, false
}
