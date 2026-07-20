package access

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const PlatformAccessIssuer = "lingvoice.platform"

// PlatformPayload is embedded in platform-admin access tokens (isolated issuer from tenant JWT).
type PlatformPayload struct {
	AdminID        uint   `json:"aid"`
	Email          string `json:"email"`
	Role           string `json:"role"`
	SessionID      string `json:"sid,omitempty"`
	DeviceRecordID uint   `json:"did,omitempty"`
}

type platformClaims struct {
	AdminID        uint   `json:"aid"`
	Email          string `json:"email"`
	Role           string `json:"role"`
	SessionID      string `json:"sid,omitempty"`
	DeviceRecordID uint   `json:"did,omitempty"`
	jwt.RegisteredClaims
}

// SignPlatformAccessTokenWithKey issues a signed platform JWT (same key infra as tenants, different issuer/claims).
func SignPlatformAccessTokenWithKey(p PlatformPayload, keyManager *KeyManager, ttl time.Duration) (string, error) {
	if keyManager == nil {
		return "", errJWTKeyManagerNil
	}
	if ttl <= 0 {
		return "", errJWTTTLNotPositive
	}
	if p.AdminID == 0 || strings.TrimSpace(p.Email) == "" {
		return "", errJWTInvalidPlatform
	}
	keyPair, err := keyManager.GetCurrentKey()
	if err != nil {
		return "", fmt.Errorf("jwt: signing key: %w", err)
	}
	now := time.Now()
	claims := platformClaims{
		AdminID:        p.AdminID,
		Email:          strings.TrimSpace(p.Email),
		Role:           strings.TrimSpace(p.Role),
		SessionID:      p.SessionID,
		DeviceRecordID: p.DeviceRecordID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    PlatformAccessIssuer,
			Subject:   fmt.Sprintf("platform_admin:%d", p.AdminID),
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

// ParsePlatformAccessTokenWithKey validates issuer/signature/expiry for a platform admin token.
func ParsePlatformAccessTokenWithKey(tokenString string, keyManager *KeyManager) (*PlatformPayload, error) {
	if tokenString == "" || keyManager == nil {
		return nil, ErrInvalidToken
	}
	token, err := jwt.ParseWithClaims(tokenString, &platformClaims{}, func(t *jwt.Token) (interface{}, error) {
		kid, ok := t.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("missing kid")
		}
		publicKey, err := keyManager.GetPublicKey(kid)
		if err != nil {
			return nil, err
		}
		switch publicKey.(type) {
		case *rsa.PublicKey:
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
		case *ecdsa.PublicKey:
			if _, ok := t.Method.(*jwt.SigningMethodECDSA); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
		default:
			return nil, fmt.Errorf("unsupported public key type")
		}
		return publicKey, nil
	})
	if err != nil || token == nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	pc, ok := token.Claims.(*platformClaims)
	if !ok {
		return nil, ErrInvalidToken
	}
	if pc.Issuer != PlatformAccessIssuer {
		return nil, ErrInvalidToken
	}
	if pc.AdminID == 0 {
		return nil, ErrInvalidToken
	}
	return &PlatformPayload{
		AdminID:        pc.AdminID,
		Email:          pc.Email,
		Role:           pc.Role,
		SessionID:      pc.SessionID,
		DeviceRecordID: pc.DeviceRecordID,
	}, nil
}
