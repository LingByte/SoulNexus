package security

import "golang.org/x/crypto/bcrypt"

func HashPassword(raw string) (string, error) {
	out, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func VerifyPassword(hash, raw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(raw)) == nil
}
