package access

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("my-secret-password")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	if hash == "" {
		t.Error("expected non-empty hash")
	}
	if hash == "my-secret-password" {
		t.Error("hash must not equal plaintext")
	}
}

func TestCheckPasswordMatch(t *testing.T) {
	hash, err := HashPassword("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	if !CheckPassword(hash, "correct-horse-battery-staple") {
		t.Error("expected password to match")
	}
}

func TestCheckPasswordMismatch(t *testing.T) {
	hash, err := HashPassword("original-password")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	if CheckPassword(hash, "wrong-password") {
		t.Error("expected password mismatch")
	}
}

func TestCheckPasswordEmptyInputs(t *testing.T) {
	hash, _ := HashPassword("some-password")
	if CheckPassword(hash, "") {
		t.Error("empty plain should never match")
	}
	if CheckPassword("", "some-password") {
		t.Error("empty hash should never match")
	}
}

func TestHashPasswordDeterministicOutput(t *testing.T) {
	// bcrypt salts are random, so each call produces a different hash
	h1, err := HashPassword("same-password")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	h2, err := HashPassword("same-password")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	if h1 == h2 {
		t.Error("expected different bcrypt hashes due to random salt")
	}
	// both should verify
	if !CheckPassword(h1, "same-password") {
		t.Error("h1 should verify")
	}
	if !CheckPassword(h2, "same-password") {
		t.Error("h2 should verify")
	}
}
