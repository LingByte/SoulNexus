package access

import (
	"testing"
	"time"
)

func TestSignPlatformAccessTokenWithKey(t *testing.T) {
	km := NewKeyManager("RS256")
	_, err := km.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	payload := PlatformPayload{
		AdminID: 1,
		Email:   "admin@lingvoice.com",
		Role:    "super_admin",
	}

	token, err := SignPlatformAccessTokenWithKey(payload, km, time.Hour)
	if err != nil {
		t.Fatalf("failed to sign platform token: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}
}

func TestParsePlatformAccessTokenWithKey(t *testing.T) {
	km := NewKeyManager("RS256")
	_, err := km.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	payload := PlatformPayload{
		AdminID: 42,
		Email:   "admin@example.com",
		Role:    "admin",
	}

	token, err := SignPlatformAccessTokenWithKey(payload, km, time.Hour)
	if err != nil {
		t.Fatalf("failed to sign platform token: %v", err)
	}

	parsed, err := ParsePlatformAccessTokenWithKey(token, km)
	if err != nil {
		t.Fatalf("failed to parse platform token: %v", err)
	}

	if parsed.AdminID != payload.AdminID {
		t.Errorf("expected AdminID %d, got %d", payload.AdminID, parsed.AdminID)
	}
	if parsed.Email != payload.Email {
		t.Errorf("expected Email %q, got %q", payload.Email, parsed.Email)
	}
	if parsed.Role != payload.Role {
		t.Errorf("expected Role %q, got %q", payload.Role, parsed.Role)
	}
}

func TestPlatformTokenIsolatedFromAccessToken(t *testing.T) {
	km := NewKeyManager("RS256")
	_, err := km.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	// Sign a platform token
	pt, err := SignPlatformAccessTokenWithKey(PlatformPayload{AdminID: 1, Email: "p@x.co", Role: "admin"}, km, time.Hour)
	if err != nil {
		t.Fatalf("failed to sign platform token: %v", err)
	}

	// Platform token must NOT parse as a tenant access token
	_, err = ParseAccessTokenWithKey(pt, km)
	if err == nil {
		t.Error("platform token should NOT be parseable as access token (issuer mismatch)")
	}

	// Tenant access token must NOT parse as platform token
	at, err := SignAccessTokenWithKey(AccessPayload{UserID: 7, Email: "u@x.co", Role: "user"}, km, time.Hour)
	if err != nil {
		t.Fatalf("failed to sign access token: %v", err)
	}
	_, err = ParsePlatformAccessTokenWithKey(at, km)
	if err == nil {
		t.Error("access token should NOT be parseable as platform token (issuer mismatch)")
	}
}

func TestPlatformTokenNilKeyManager(t *testing.T) {
	_, err := SignPlatformAccessTokenWithKey(PlatformPayload{AdminID: 1, Email: "a@b.co"}, nil, time.Hour)
	if err == nil {
		t.Error("expected error for nil key manager")
	}
}

func TestPlatformTokenInvalidPayload(t *testing.T) {
	km := NewKeyManager("RS256")
	_, _ = km.GenerateKey()

	// Missing AdminID
	_, err := SignPlatformAccessTokenWithKey(PlatformPayload{Email: "a@b.co"}, km, time.Hour)
	if err == nil {
		t.Error("expected error for missing AdminID")
	}

	// Missing Email
	_, err = SignPlatformAccessTokenWithKey(PlatformPayload{AdminID: 1}, km, time.Hour)
	if err == nil {
		t.Error("expected error for missing Email")
	}
}

func TestParsePlatformTokenEmptyInput(t *testing.T) {
	_, err := ParsePlatformAccessTokenWithKey("", nil)
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestPlatformTokenES256(t *testing.T) {
	km := NewKeyManager("ES256")
	_, err := km.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate ES256 key: %v", err)
	}

	payload := PlatformPayload{
		AdminID: 99,
		Email:   "ec@example.com",
		Role:    "admin",
	}

	token, err := SignPlatformAccessTokenWithKey(payload, km, 30*time.Minute)
	if err != nil {
		t.Fatalf("failed to sign ES256 platform token: %v", err)
	}

	parsed, err := ParsePlatformAccessTokenWithKey(token, km)
	if err != nil {
		t.Fatalf("failed to parse ES256 platform token: %v", err)
	}

	if parsed.AdminID != 99 {
		t.Errorf("expected AdminID 99, got %d", parsed.AdminID)
	}
}
