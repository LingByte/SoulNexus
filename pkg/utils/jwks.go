// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package utils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	jwtKeyManagerMu sync.RWMutex
	jwtKeyManager   *KeyManager
)

// InstallJWTKeyManager registers the KeyManager used to sign and verify access/refresh JWTs.
// Call from process bootstrap (e.g. after InitializeKeyManager); use nil to clear.
func InstallJWTKeyManager(km *KeyManager) {
	jwtKeyManagerMu.Lock()
	defer jwtKeyManagerMu.Unlock()
	jwtKeyManager = km
}

// JWTKeyManager returns the KeyManager installed for JWT auth, or nil.
func JWTKeyManager() *KeyManager {
	jwtKeyManagerMu.RLock()
	defer jwtKeyManagerMu.RUnlock()
	return jwtKeyManager
}

// IsLikelyCompactJWT returns true if s has three non-empty dot-separated segments (JWS compact form).
func IsLikelyCompactJWT(s string) bool {
	a, b, c, ok := splitJWTThreeParts(s)
	return ok && a != "" && b != "" && c != ""
}

func splitJWTThreeParts(s string) (h, p, sig string, ok bool) {
	i := strings.IndexByte(s, '.')
	if i <= 0 {
		return "", "", "", false
	}
	j := strings.IndexByte(s[i+1:], '.')
	if j < 0 {
		return "", "", "", false
	}
	j += i + 1
	if j >= len(s)-1 {
		return "", "", "", false
	}
	k := strings.IndexByte(s[j+1:], '.')
	if k >= 0 {
		return "", "", "", false
	}
	return s[:i], s[i+1 : j], s[j+1:], true
}

// KeyPair represents a signing key pair with metadata
type KeyPair struct {
	ID        string      `json:"kid"`
	Algorithm string      `json:"alg"`
	PublicKey interface{} `json:"-"` // rsa.PublicKey or ecdsa.PublicKey
	PrivateKey interface{} `json:"-"` // rsa.PrivateKey or ecdsa.PrivateKey
	CreatedAt time.Time   `json:"created_at"`
}

// JWKS represents a JSON Web Key Set
type JWKS struct {
	Keys []JSONWebKey `json:"keys"`
}

// JSONWebKey represents a single JSON Web Key
type JSONWebKey struct {
	Kty string `json:"kty"` // Key Type: RSA or EC
	Kid string `json:"kid"` // Key ID
	Alg string `json:"alg"` // Algorithm: RS256, ES256, etc.
	Use string `json:"use"` // Public Key Use: sig
	N   string `json:"n"`   // Modulus (for RSA)
	E   string `json:"e"`   // Exponent (for RSA)
	X   string `json:"x"`   // X coordinate (for EC)
	Y   string `json:"y"`   // Y coordinate (for EC)
	Crv string `json:"crv"` // Curve (for EC)
}

// KeyManager manages signing keys and JWKS
type KeyManager struct {
	mu         sync.RWMutex
	keys       map[string]*KeyPair // kid -> KeyPair
	currentKid string
	algorithm  string // RS256, ES256, etc.
}

// NewKeyManager creates a new key manager
func NewKeyManager(algorithm string) *KeyManager {
	return &KeyManager{
		keys:      make(map[string]*KeyPair),
		algorithm: algorithm,
	}
}

// GenerateKey generates a new key pair with the specified algorithm
func (km *KeyManager) GenerateKey() (*KeyPair, error) {
	km.mu.Lock()
	defer km.mu.Unlock()

	kid := generateKeyID()
	var pubKey, privKey interface{}
	var err error

	switch km.algorithm {
	case "RS256":
		privKey, err = rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, fmt.Errorf("failed to generate RSA key: %w", err)
		}
		pubKey = privKey.(*rsa.PrivateKey).Public()
	case "ES256":
		privKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("failed to generate ECDSA key: %w", err)
		}
		pubKey = privKey.(*ecdsa.PrivateKey).Public()
	default:
		return nil, fmt.Errorf("unsupported algorithm: %s", km.algorithm)
	}

	keyPair := &KeyPair{
		ID:         kid,
		Algorithm:  km.algorithm,
		PublicKey:  pubKey,
		PrivateKey: privKey,
		CreatedAt:  time.Now(),
	}

	km.keys[kid] = keyPair
	km.currentKid = kid

	return keyPair, nil
}

// GetCurrentKey returns the current signing key
func (km *KeyManager) GetCurrentKey() (*KeyPair, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	if km.currentKid == "" {
		return nil, fmt.Errorf("no current key available")
	}

	key, ok := km.keys[km.currentKid]
	if !ok {
		return nil, fmt.Errorf("current key not found")
	}

	return key, nil
}

// GetKeyByID returns a key pair by kid
func (km *KeyManager) GetKeyByID(kid string) (*KeyPair, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	key, ok := km.keys[kid]
	if !ok {
		return nil, fmt.Errorf("key not found: %s", kid)
	}

	return key, nil
}

// KeyCount returns how many signing keys are loaded (including retired keys kept for verification).
func (km *KeyManager) KeyCount() int {
	km.mu.RLock()
	defer km.mu.RUnlock()
	return len(km.keys)
}

// GetPublicKey returns the public key for a given kid
func (km *KeyManager) GetPublicKey(kid string) (interface{}, error) {
	key, err := km.GetKeyByID(kid)
	if err != nil {
		return nil, err
	}
	return key.PublicKey, nil
}

// GetJWKS returns the JWKS for all public keys
func (km *KeyManager) GetJWKS() (*JWKS, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	jwks := &JWKS{
		Keys: make([]JSONWebKey, 0, len(km.keys)),
	}

	for _, key := range km.keys {
		jwk, err := keyToJSONWebKey(key)
		if err != nil {
			return nil, fmt.Errorf("failed to convert key to JWK: %w", err)
		}
		jwks.Keys = append(jwks.Keys, jwk)
	}

	return jwks, nil
}

// RotateKeys generates a new key and optionally removes old keys
func (km *KeyManager) RotateKeys(keepOldKeys int) error {
	_, err := km.GenerateKey()
	if err != nil {
		return err
	}

	// Remove old keys if we have too many
	km.mu.Lock()
	defer km.mu.Unlock()

	if len(km.keys) > keepOldKeys {
		// Find and remove oldest keys
		type keyWithTime struct {
			kid  string
			time time.Time
		}
		keyTimes := make([]keyWithTime, 0, len(km.keys))
		for kid, key := range km.keys {
			keyTimes = append(keyTimes, keyWithTime{kid, key.CreatedAt})
		}

		// Sort by creation time (oldest first)
		for i := 0; i < len(keyTimes)-1; i++ {
			for j := i + 1; j < len(keyTimes); j++ {
				if keyTimes[i].time.After(keyTimes[j].time) {
					keyTimes[i], keyTimes[j] = keyTimes[j], keyTimes[i]
				}
			}
		}

		// Remove excess keys
		for i := 0; i < len(keyTimes)-keepOldKeys; i++ {
			if keyTimes[i].kid != km.currentKid {
				delete(km.keys, keyTimes[i].kid)
			}
		}
	}

	return nil
}

// keyToJSONWebKey converts a KeyPair to JSONWebKey
func keyToJSONWebKey(key *KeyPair) (JSONWebKey, error) {
	jwk := JSONWebKey{
		Kid: key.ID,
		Alg: key.Algorithm,
		Use: "sig",
	}

	switch k := key.PublicKey.(type) {
	case *rsa.PublicKey:
		jwk.Kty = "RSA"
		jwk.N = base64URLEncode(k.N.Bytes())
		jwk.E = base64URLEncode(rsaExponentBytes(k.E))
	case *ecdsa.PublicKey:
		jwk.Kty = "EC"
		jwk.Crv = "P-256"
		jwk.X = base64URLEncode(k.X.Bytes())
		jwk.Y = base64URLEncode(k.Y.Bytes())
	default:
		return JSONWebKey{}, fmt.Errorf("unsupported public key type: %T", key.PublicKey)
	}

	return jwk, nil
}

// generateKeyID generates a unique key ID
func generateKeyID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// base64URLEncode encodes bytes to base64 URL-safe format
func base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// rsaExponentBytes encodes the RSA public exponent as minimal unsigned big-endian bytes (JWK "e").
func rsaExponentBytes(e int) []byte {
	return new(big.Int).SetInt64(int64(e)).Bytes()
}

// SaveKeysToFile saves all keys to a JSON file
func (km *KeyManager) SaveKeysToFile(filename string) error {
	km.mu.RLock()
	defer km.mu.RUnlock()

	data := make(map[string]interface{})
	for kid, key := range km.keys {
		keyData := map[string]interface{}{
			"algorithm": key.Algorithm,
			"created_at": key.CreatedAt.Unix(),
		}

		// Serialize private key
		switch k := key.PrivateKey.(type) {
		case *rsa.PrivateKey:
			keyData["private_key"] = map[string]interface{}{
				"type": "RSA",
				"d":    base64.RawURLEncoding.EncodeToString(k.D.Bytes()),
				"p":    base64.RawURLEncoding.EncodeToString(k.Primes[0].Bytes()),
				"q":    base64.RawURLEncoding.EncodeToString(k.Primes[1].Bytes()),
			}
		case *ecdsa.PrivateKey:
			keyData["private_key"] = map[string]interface{}{
				"type": "EC",
				"d":    base64.RawURLEncoding.EncodeToString(k.D.Bytes()),
			}
		}

		data[kid] = keyData
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal keys: %w", err)
	}

	return os.WriteFile(filename, jsonData, 0600)
}

// LoadKeysFromFile loads keys from a JSON file
func (km *KeyManager) LoadKeysFromFile(filename string) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read keys file: %w", err)
	}

	var keysData map[string]map[string]interface{}
	if err := json.Unmarshal(data, &keysData); err != nil {
		return fmt.Errorf("failed to unmarshal keys: %w", err)
	}

	for kid, keyData := range keysData {
		algorithm, ok := keyData["algorithm"].(string)
		if !ok {
			return fmt.Errorf("missing algorithm for key %s", kid)
		}

		var privateKey interface{}
		privKeyData, ok := keyData["private_key"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("missing private_key for key %s", kid)
		}

		keyType, ok := privKeyData["type"].(string)
		if !ok {
			return fmt.Errorf("missing private_key type for key %s", kid)
		}

		switch keyType {
		case "RSA":
			dStr, ok := privKeyData["d"].(string)
			if !ok {
				return fmt.Errorf("missing d for RSA key %s", kid)
			}
			d, err := base64.RawURLEncoding.DecodeString(dStr)
			if err != nil {
				return fmt.Errorf("failed to decode d for RSA key %s: %w", kid, err)
			}

			pStr, ok := privKeyData["p"].(string)
			if !ok {
				return fmt.Errorf("missing p for RSA key %s", kid)
			}
			p, err := base64.RawURLEncoding.DecodeString(pStr)
			if err != nil {
				return fmt.Errorf("failed to decode p for RSA key %s: %w", kid, err)
			}

			qStr, ok := privKeyData["q"].(string)
			if !ok {
				return fmt.Errorf("missing q for RSA key %s", kid)
			}
			q, err := base64.RawURLEncoding.DecodeString(qStr)
			if err != nil {
				return fmt.Errorf("failed to decode q for RSA key %s: %w", kid, err)
			}

			// Reconstruct RSA private key
			primeP := new(big.Int).SetBytes(p)
			primeQ := new(big.Int).SetBytes(q)
			dInt := new(big.Int).SetBytes(d)

			privateKey = &rsa.PrivateKey{
				D:      dInt,
				Primes: []*big.Int{primeP, primeQ},
			}
			pk := privateKey.(*rsa.PrivateKey)
			pk.PublicKey.N = new(big.Int).Mul(primeP, primeQ)
			pk.PublicKey.E = 65537 // Standard RSA exponent
			if err := pk.Validate(); err != nil {
				return fmt.Errorf("invalid RSA key for kid %s: %w", kid, err)
			}
			pk.Precompute()

		case "EC":
			dStr, ok := privKeyData["d"].(string)
			if !ok {
				return fmt.Errorf("missing d for EC key %s", kid)
			}
			d, err := base64.RawURLEncoding.DecodeString(dStr)
			if err != nil {
				return fmt.Errorf("failed to decode d for EC key %s: %w", kid, err)
			}

			dInt := new(big.Int).SetBytes(d)
			curve := elliptic.P256()
			x, y := curve.ScalarBaseMult(d)

			privateKey = &ecdsa.PrivateKey{
				D:       dInt,
				PublicKey: ecdsa.PublicKey{
					Curve: curve,
					X:     x,
					Y:     y,
				},
			}
		default:
			return fmt.Errorf("unsupported key type %s for key %s", keyType, kid)
		}

		createdAt := time.Now()
		if ts, ok := keyData["created_at"].(float64); ok {
			createdAt = time.Unix(int64(ts), 0)
		}

		keyPair := &KeyPair{
			ID:         kid,
			Algorithm:  algorithm,
			PrivateKey: privateKey,
			PublicKey:  getPublicKeyFromPrivate(privateKey),
			CreatedAt:  createdAt,
		}

		km.keys[kid] = keyPair
	}

	if len(km.keys) > 0 {
		var latest time.Time
		var pick string
		for id, kp := range km.keys {
			if pick == "" || kp.CreatedAt.After(latest) {
				latest = kp.CreatedAt
				pick = id
			}
		}
		km.currentKid = pick
	}

	return nil
}

// getPublicKeyFromPrivate extracts public key from private key
func getPublicKeyFromPrivate(privateKey interface{}) interface{} {
	switch k := privateKey.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

// GetJWKSJSON returns the JWKS as JSON string
func (km *KeyManager) GetJWKSJSON() (string, error) {
	jwks, err := km.GetJWKS()
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(jwks)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JWKS: %w", err)
	}

	return string(data), nil
}

// JWKSHandler returns an HTTP handler function that serves the JWKS endpoint
// This can be used with Gin, Echo, or other web frameworks
func (km *KeyManager) JWKSHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour

		jwksJSON, err := km.GetJWKSJSON()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "failed to generate JWKS"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(jwksJSON))
	}
}
