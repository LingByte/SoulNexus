// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package utils

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrInvalidToken is returned when the token is malformed, expired, or signature-invalid.
	ErrInvalidToken = errors.New("jwt: invalid token")
)

const (
	AccessIssuer  = "lingvoice"
	RefreshIssuer = "lingvoice.refresh"
)

// AccessPayload is the application data carried in an access token.
type AccessPayload struct {
	UserID uint   `json:"uid"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}

type accessClaims struct {
	UserID uint   `json:"uid"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// SignAccessToken issues an HS256 JWT with subject user:<id> and standard time claims.
func SignAccessToken(p AccessPayload, secret string, ttl time.Duration) (string, error) {
	if len(secret) < 8 {
		return "", errors.New("jwt: signing secret too short (min 8 bytes)")
	}
	if ttl <= 0 {
		return "", errors.New("jwt: ttl must be positive")
	}
	now := time.Now()
	claims := accessClaims{
		UserID: p.UserID,
		Email:  p.Email,
		Role:   p.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    AccessIssuer,
			Subject:   fmt.Sprintf("user:%d", p.UserID),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-30 * time.Second)),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, &claims)
	return t.SignedString([]byte(secret))
}

// SignAccessTokenWithKey signs an access token using a KeyManager (RS256/ES256 with kid)
func SignAccessTokenWithKey(p AccessPayload, keyManager *KeyManager, ttl time.Duration) (string, error) {
	if keyManager == nil {
		return "", errors.New("jwt: key manager is nil")
	}
	if ttl <= 0 {
		return "", errors.New("jwt: ttl must be positive")
	}

	keyPair, err := keyManager.GetCurrentKey()
	if err != nil {
		return "", fmt.Errorf("jwt: failed to get signing key: %w", err)
	}

	now := time.Now()
	claims := accessClaims{
		UserID: p.UserID,
		Email:  p.Email,
		Role:   p.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    AccessIssuer,
			Subject:   fmt.Sprintf("user:%d", p.UserID),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-30 * time.Second)),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}

	var token *jwt.Token
	switch keyPair.Algorithm {
	case "RS256":
		token = jwt.NewWithClaims(jwt.SigningMethodRS256, &claims)
	case "ES256":
		token = jwt.NewWithClaims(jwt.SigningMethodES256, &claims)
	default:
		return "", fmt.Errorf("jwt: unsupported algorithm: %s", keyPair.Algorithm)
	}

	token.Header["kid"] = keyPair.ID

	return token.SignedString(keyPair.PrivateKey)
}

// ParseAccessToken validates signature and expiry and returns the embedded payload.
func ParseAccessToken(tokenString, secret string) (*AccessPayload, error) {
	if tokenString == "" || len(secret) < 8 {
		return nil, ErrInvalidToken
	}
	token, err := jwt.ParseWithClaims(tokenString, &accessClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %q", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil || token == nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	ac, ok := token.Claims.(*accessClaims)
	if !ok {
		return nil, ErrInvalidToken
	}
	if ac.Issuer != AccessIssuer {
		return nil, ErrInvalidToken
	}
	return &AccessPayload{
		UserID: ac.UserID,
		Email:  ac.Email,
		Role:   ac.Role,
	}, nil
}

// ParseAccessTokenWithKey validates an access token using KeyManager (JWKS with kid)
func ParseAccessTokenWithKey(tokenString string, keyManager *KeyManager) (*AccessPayload, error) {
	if tokenString == "" || keyManager == nil {
		return nil, ErrInvalidToken
	}

	token, err := jwt.ParseWithClaims(tokenString, &accessClaims{}, func(t *jwt.Token) (interface{}, error) {
		// Extract kid from header
		kid, ok := t.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("missing kid in token header")
		}

		// Get public key from KeyManager
		publicKey, err := keyManager.GetPublicKey(kid)
		if err != nil {
			return nil, fmt.Errorf("failed to get public key for kid %s: %w", kid, err)
		}

		// Verify algorithm matches
		var expectedAlg string
		switch publicKey.(type) {
		case *rsa.PublicKey:
			expectedAlg = "RS256"
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method %q, expected RS256", t.Header["alg"])
			}
		case *ecdsa.PublicKey:
			expectedAlg = "ES256"
			if _, ok := t.Method.(*jwt.SigningMethodECDSA); !ok {
				return nil, fmt.Errorf("unexpected signing method %q, expected ES256", t.Header["alg"])
			}
		default:
			return nil, fmt.Errorf("unsupported public key type")
		}

		if t.Header["alg"] != expectedAlg {
			return nil, fmt.Errorf("unexpected algorithm %q, expected %s", t.Header["alg"], expectedAlg)
		}

		return publicKey, nil
	})

	if err != nil || token == nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	ac, ok := token.Claims.(*accessClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	if ac.Issuer != AccessIssuer {
		return nil, ErrInvalidToken
	}

	return &AccessPayload{
		UserID: ac.UserID,
		Email:  ac.Email,
		Role:   ac.Role,
	}, nil
}
