package wxcrypt

import (
	"crypto/rand"
	"encoding/base64"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	aesKey := base64.StdEncoding.EncodeToString(key)
	aesKey = aesKey[:43] // WeChat EncodingAESKey is 43 chars without padding

	c, err := New("testtoken", aesKey, "wx_app_id")
	if err != nil {
		t.Fatal(err)
	}
	plain := []byte(`<xml><Content>hello</Content></xml>`)
	enc, err := c.encrypt(plain)
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.decrypt(enc)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(plain) {
		t.Fatalf("got %q want %q", got, plain)
	}
}

func TestNewRejectsBadKey(t *testing.T) {
	if _, err := New("t", "short", "app"); err == nil {
		t.Fatal("expected error for short EncodingAESKey")
	}
}
