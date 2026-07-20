package sms

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"testing"
)

func TestSha1Hex(t *testing.T) {
	t.Parallel()
	in := "hello"
	sum := sha1.Sum([]byte(in))
	want := hex.EncodeToString(sum[:])
	if got := sha1Hex(in); got != want {
		t.Errorf("sha1Hex() = %q, want %q", got, want)
	}
}

func TestSha256B64(t *testing.T) {
	t.Parallel()
	in := "secret+nonce+created"
	h := sha256.Sum256([]byte(in))
	want := base64.StdEncoding.EncodeToString(h[:])
	if got := sha256B64(in); got != want {
		t.Errorf("sha256B64() = %q, want %q", got, want)
	}
}

func TestMd5Hex(t *testing.T) {
	t.Parallel()
	in := "SIDtokenTS"
	sum := md5.Sum([]byte(in))
	want := hex.EncodeToString(sum[:])
	if got := md5Hex(in); got != want {
		t.Errorf("md5Hex() = %q, want %q", got, want)
	}
}

func TestMd5Hex_uppercaseYuntongxun(t *testing.T) {
	t.Parallel()
	// Yuntongxun uses strings.ToUpper(md5Hex(...))
	raw := "AC123token20260101120000"
	sig := md5Hex(raw)
	if len(sig) != 32 {
		t.Fatalf("md5 hex length = %d", len(sig))
	}
}
