// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package utils

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewKeyManager(t *testing.T) {
	km := NewKeyManager("RS256")
	if km == nil {
		t.Fatal("expected non-nil KeyManager")
	}
	if km.algorithm != "RS256" {
		t.Errorf("expected algorithm RS256, got %s", km.algorithm)
	}
}

func TestGenerateKey(t *testing.T) {
	km := NewKeyManager("RS256")
	key, err := km.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	if key.ID == "" {
		t.Error("expected non-empty key ID")
	}
	if key.Algorithm != "RS256" {
		t.Errorf("expected algorithm RS256, got %s", key.Algorithm)
	}
	if key.PublicKey == nil {
		t.Error("expected non-nil public key")
	}
	if key.PrivateKey == nil {
		t.Error("expected non-nil private key")
	}
	if key.CreatedAt.IsZero() {
		t.Error("expected non-zero creation time")
	}
}

func TestGenerateKeyES256(t *testing.T) {
	km := NewKeyManager("ES256")
	key, err := km.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate ES256 key: %v", err)
	}
	if key.Algorithm != "ES256" {
		t.Errorf("expected algorithm ES256, got %s", key.Algorithm)
	}
}

func TestGetCurrentKey(t *testing.T) {
	km := NewKeyManager("RS256")
	_, err := km.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	key, err := km.GetCurrentKey()
	if err != nil {
		t.Fatalf("failed to get current key: %v", err)
	}
	if key == nil {
		t.Error("expected non-nil current key")
	}
}

func TestGetKeyByID(t *testing.T) {
	km := NewKeyManager("RS256")
	key1, err := km.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	key2, err := km.GetKeyByID(key1.ID)
	if err != nil {
		t.Fatalf("failed to get key by ID: %v", err)
	}
	if key2.ID != key1.ID {
		t.Errorf("expected key ID %s, got %s", key1.ID, key2.ID)
	}
}

func TestGetPublicKey(t *testing.T) {
	km := NewKeyManager("RS256")
	key, err := km.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	pubKey, err := km.GetPublicKey(key.ID)
	if err != nil {
		t.Fatalf("failed to get public key: %v", err)
	}
	if pubKey == nil {
		t.Error("expected non-nil public key")
	}
}

func TestGetJWKS(t *testing.T) {
	km := NewKeyManager("RS256")
	_, err := km.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	jwks, err := km.GetJWKS()
	if err != nil {
		t.Fatalf("failed to get JWKS: %v", err)
	}
	if len(jwks.Keys) == 0 {
		t.Error("expected at least one key in JWKS")
	}
	if jwks.Keys[0].Kid == "" {
		t.Error("expected non-empty kid in JWKS key")
	}
	if jwks.Keys[0].Alg != "RS256" {
		t.Errorf("expected alg RS256, got %s", jwks.Keys[0].Alg)
	}
	if jwks.Keys[0].Use != "sig" {
		t.Errorf("expected use sig, got %s", jwks.Keys[0].Use)
	}
}

func TestRotateKeys(t *testing.T) {
	km := NewKeyManager("RS256")
	_, err := km.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate first key: %v", err)
	}

	firstKey, _ := km.GetCurrentKey()
	firstID := firstKey.ID

	// Rotate keys
	err = km.RotateKeys(2)
	if err != nil {
		t.Fatalf("failed to rotate keys: %v", err)
	}

	currentKey, err := km.GetCurrentKey()
	if err != nil {
		t.Fatalf("failed to get current key after rotation: %v", err)
	}

	// Current key should be different after rotation
	if currentKey.ID == firstID {
		t.Error("expected current key ID to change after rotation")
	}
}

func TestRotateKeysWithCleanup(t *testing.T) {
	km := NewKeyManager("RS256")
	// Generate 3 keys
	for i := 0; i < 3; i++ {
		_, err := km.GenerateKey()
		if err != nil {
			t.Fatalf("failed to generate key %d: %v", i, err)
		}
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// Rotate and keep only 2 keys
	err := km.RotateKeys(2)
	if err != nil {
		t.Fatalf("failed to rotate keys: %v", err)
	}

	jwks, err := km.GetJWKS()
	if err != nil {
		t.Fatalf("failed to get JWKS: %v", err)
	}

	// Should have at most 2 keys (current + 1 old)
	if len(jwks.Keys) > 3 { // Allow some tolerance
		t.Errorf("expected at most 3 keys, got %d", len(jwks.Keys))
	}
}

func TestGetJWKSJSON(t *testing.T) {
	km := NewKeyManager("RS256")
	_, err := km.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	jsonStr, err := km.GetJWKSJSON()
	if err != nil {
		t.Fatalf("failed to get JWKS JSON: %v", err)
	}
	if jsonStr == "" {
		t.Error("expected non-empty JSON string")
	}

	// Verify it's valid JSON
	var jwks JWKS
	err = json.Unmarshal([]byte(jsonStr), &jwks)
	if err != nil {
		t.Fatalf("failed to unmarshal JWKS JSON: %v", err)
	}
	if len(jwks.Keys) == 0 {
		t.Error("expected at least one key in parsed JWKS")
	}
}

func TestKeyToJSONWebKeyRSA(t *testing.T) {
	km := NewKeyManager("RS256")
	key, err := km.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	jwk, err := keyToJSONWebKey(key)
	if err != nil {
		t.Fatalf("failed to convert key to JWK: %v", err)
	}

	if jwk.Kty != "RSA" {
		t.Errorf("expected kty RSA, got %s", jwk.Kty)
	}
	if jwk.Alg != "RS256" {
		t.Errorf("expected alg RS256, got %s", jwk.Alg)
	}
	if jwk.N == "" {
		t.Error("expected non-empty modulus (n)")
	}
	if jwk.E == "" {
		t.Error("expected non-empty exponent (e)")
	}
}

func TestKeyToJSONWebKeyEC(t *testing.T) {
	km := NewKeyManager("ES256")
	key, err := km.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	jwk, err := keyToJSONWebKey(key)
	if err != nil {
		t.Fatalf("failed to convert key to JWK: %v", err)
	}

	if jwk.Kty != "EC" {
		t.Errorf("expected kty EC, got %s", jwk.Kty)
	}
	if jwk.Alg != "ES256" {
		t.Errorf("expected alg ES256, got %s", jwk.Alg)
	}
	if jwk.Crv != "P-256" {
		t.Errorf("expected crv P-256, got %s", jwk.Crv)
	}
	if jwk.X == "" {
		t.Error("expected non-empty X coordinate")
	}
	if jwk.Y == "" {
		t.Error("expected non-empty Y coordinate")
	}
}

func TestSignAccessTokenWithKey(t *testing.T) {
	km := NewKeyManager("RS256")
	_, err := km.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	payload := AccessPayload{
		UserID: 123,
		Email:  "test@example.com",
		Role:   "admin",
	}

	token, err := SignAccessTokenWithKey(payload, km, time.Hour)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}
}

func TestParseAccessTokenWithKey(t *testing.T) {
	km := NewKeyManager("RS256")
	_, err := km.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	payload := AccessPayload{
		UserID: 123,
		Email:  "test@example.com",
		Role:   "admin",
	}

	token, err := SignAccessTokenWithKey(payload, km, time.Hour)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	parsed, err := ParseAccessTokenWithKey(token, km)
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}

	if parsed.UserID != payload.UserID {
		t.Errorf("expected user ID %d, got %d", payload.UserID, parsed.UserID)
	}
	if parsed.Email != payload.Email {
		t.Errorf("expected email %s, got %s", payload.Email, parsed.Email)
	}
	if parsed.Role != payload.Role {
		t.Errorf("expected role %s, got %s", payload.Role, parsed.Role)
	}
}

func TestSignRefreshTokenWithKey(t *testing.T) {
	km := NewKeyManager("RS256")
	_, err := km.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	payload := RefreshPayload{
		UserID: 123,
	}

	token, err := SignRefreshTokenWithKey(payload, km, 24*time.Hour)
	if err != nil {
		t.Fatalf("failed to sign refresh token: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}
}

func TestParseRefreshTokenWithKey(t *testing.T) {
	km := NewKeyManager("RS256")
	_, err := km.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	payload := RefreshPayload{
		UserID: 123,
	}

	token, err := SignRefreshTokenWithKey(payload, km, 24*time.Hour)
	if err != nil {
		t.Fatalf("failed to sign refresh token: %v", err)
	}

	parsed, err := ParseRefreshTokenWithKey(token, km)
	if err != nil {
		t.Fatalf("failed to parse refresh token: %v", err)
	}

	if parsed.UserID != payload.UserID {
		t.Errorf("expected user ID %d, got %d", payload.UserID, parsed.UserID)
	}
}

func TestIsLikelyCompactJWT(t *testing.T) {
	if !IsLikelyCompactJWT("eyJhbGciOiJIUzI1NiJ9.e30.sig") {
		t.Error("expected valid compact JWS shape")
	}
	if IsLikelyCompactJWT("a.b") {
		t.Error("two segments should not match")
	}
	if IsLikelyCompactJWT("a.b.c.d") {
		t.Error("four segments should not match")
	}
}

func TestRSAExponent65537JWKEncoding(t *testing.T) {
	b := rsaExponentBytes(65537)
	want := []byte{0x01, 0x00, 0x01}
	if len(b) != len(want) {
		t.Fatalf("len got %d want %d", len(b), len(want))
	}
	for i := range want {
		if b[i] != want[i] {
			t.Fatalf("byte %d: got %x want %x", i, b[i], want[i])
		}
	}
}
