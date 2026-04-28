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

// RefreshPayload is embedded in long-lived refresh tokens.
type RefreshPayload struct {
	UserID uint   `json:"uid"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}

type refreshClaims struct {
	UserID uint   `json:"uid"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// SignRefreshToken issues a refresh JWT (distinct issuer from access).
func SignRefreshToken(p RefreshPayload, secret string, ttl time.Duration) (string, error) {
	if len(secret) < 8 {
		return "", errors.New("jwt: refresh signing secret too short (min 8 bytes)")
	}
	if ttl <= 0 {
		return "", errors.New("jwt: ttl must be positive")
	}
	now := time.Now()
	claims := refreshClaims{
		UserID: p.UserID,
		Email:  p.Email,
		Role:   p.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    RefreshIssuer,
			Subject:   fmt.Sprintf("refresh:%d", p.UserID),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-30 * time.Second)),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, &claims)
	return t.SignedString([]byte(secret))
}

// SignRefreshTokenWithKey signs a refresh token using a KeyManager (RS256/ES256 with kid)
func SignRefreshTokenWithKey(p RefreshPayload, keyManager *KeyManager, ttl time.Duration) (string, error) {
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
	claims := refreshClaims{
		UserID: p.UserID,
		Email:  p.Email,
		Role:   p.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    RefreshIssuer,
			Subject:   fmt.Sprintf("refresh:%d", p.UserID),
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

// ParseRefreshToken validates a refresh JWT.
func ParseRefreshToken(tokenString, secret string) (*RefreshPayload, error) {
	if tokenString == "" || len(secret) < 8 {
		return nil, ErrInvalidToken
	}
	token, err := jwt.ParseWithClaims(tokenString, &refreshClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %q", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil || token == nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	rc, ok := token.Claims.(*refreshClaims)
	if !ok {
		return nil, ErrInvalidToken
	}
	if rc.Issuer != RefreshIssuer {
		return nil, ErrInvalidToken
	}
	return &RefreshPayload{UserID: rc.UserID, Email: rc.Email, Role: rc.Role}, nil
}

// ParseRefreshTokenWithKey validates a refresh token using KeyManager (JWKS with kid)
func ParseRefreshTokenWithKey(tokenString string, keyManager *KeyManager) (*RefreshPayload, error) {
	if tokenString == "" || keyManager == nil {
		return nil, ErrInvalidToken
	}

	token, err := jwt.ParseWithClaims(tokenString, &refreshClaims{}, func(t *jwt.Token) (interface{}, error) {
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

	rc, ok := token.Claims.(*refreshClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	if rc.Issuer != RefreshIssuer {
		return nil, ErrInvalidToken
	}

	return &RefreshPayload{UserID: rc.UserID, Email: rc.Email, Role: rc.Role}, nil
}
