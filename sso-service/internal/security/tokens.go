package security

import (
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	Scope string `json:"scope,omitempty"`
	jwt.RegisteredClaims
}

func ParseRSAPrivateKeyFromPEM(privatePEM string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(privatePEM))
	if block == nil {
		return nil, errors.New("invalid private key pem")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func ParseRSAPublicKeyFromPEM(publicPEM string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(publicPEM))
	if block == nil {
		return nil, errors.New("invalid public key pem")
	}
	return x509.ParsePKCS1PublicKey(block.Bytes)
}

func BuildAccessToken(privateKey *rsa.PrivateKey, kid, issuer, audience, userID, scope string, ttl time.Duration) (string, string, error) {
	now := time.Now()
	jti := uuid.NewString()
	claims := Claims{
		Scope: scope,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   userID,
			Audience:  jwt.ClaimStrings{audience},
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        jti,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	signed, err := token.SignedString(privateKey)
	if err != nil {
		return "", "", err
	}
	return signed, jti, nil
}

func HashToken(raw string) string {
	digest := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(digest[:])
}

func S256CodeChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
