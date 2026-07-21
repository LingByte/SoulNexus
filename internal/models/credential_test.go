package models

import (
	"strings"
	"testing"

	"github.com/LingByte/SoulNexus/internal/constants"
)

func TestCredentialClientIPAllowed(t *testing.T) {
	if !CredentialClientIPAllowed("203.0.113.9", "203.0.113.9") {
		t.Fatal("exact match")
	}
	if CredentialClientIPAllowed("203.0.113.9", "198.51.100.1") {
		t.Fatal("miss")
	}
}

func TestIssueAPIKeyAndMatch(t *testing.T) {
	full, prefix, hash, err := IssueAPIKeyForKind(constants.CredentialKindUserBundle)
	if err != nil {
		t.Fatal(err)
	}
	if !IsAPIKeyToken(full) {
		t.Fatalf("not api key: %q", full)
	}
	if !strings.HasPrefix(full, APIKeyPrefixUser) {
		t.Fatalf("want user prefix, got %q", full)
	}
	if prefix != APIKeyLookupPrefix(full) {
		t.Fatalf("prefix=%q full=%q", prefix, full)
	}
	if !APIKeyMatchesStoredHash(full, hash) {
		t.Fatal("hash mismatch")
	}
	if APIKeyMatchesStoredHash(full+"x", hash) {
		t.Fatal("should not match tampered key")
	}
	if IsAPIKeyToken("eyJhbGciOiJIUzI1NiJ9.xx.yy") {
		t.Fatal("jwt should not look like api key")
	}
}

func TestIsLegacyHMACCredential(t *testing.T) {
	if !IsLegacyHMACCredential(Credential{AccessKey: "ak_abc"}) {
		t.Fatal("expected legacy")
	}
	if IsLegacyHMACCredential(Credential{AccessKey: "soulnexus_abcdef123456"}) {
		t.Fatal("not legacy")
	}
}

func TestMaskAPIKeyPrefix(t *testing.T) {
	if MaskAPIKeyPrefix("soulnexus_abcdef") != "lingecho…" {
		t.Fatalf("%q", MaskAPIKeyPrefix("soulnexus_abcdef"))
	}
}

func TestIssueAPIKeyPrefix(t *testing.T) {
	full, prefix, _, err := IssueAPIKeyForKind(constants.CredentialKindPlatformBundle)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(full, APIKeyPrefixPlatform) {
		t.Fatalf("want prefix %q, got %q", APIKeyPrefixPlatform, full)
	}
	if prefix != APIKeyLookupPrefix(full) {
		t.Fatalf("lookup prefix=%q", prefix)
	}
	if IsAPIKeyToken("lex_abcdef1234567890") {
		// legacy tokens still recognized
	} else {
		t.Fatal("legacy lex_ should still look like api key")
	}
}
