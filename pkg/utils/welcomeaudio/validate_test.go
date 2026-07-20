// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package welcomeaudio

import (
	"context"
	"encoding/binary"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// minimalWAV builds a 12-byte RIFF/WAVE header followed by an empty
// fmt/data shell. Enough bytes to pass ValidateBytes; not necessarily
// decodeable. We use this as the "looks like a WAV" sentinel.
func minimalWAVHeader() []byte {
	b := make([]byte, 16)
	copy(b[0:4], "RIFF")
	binary.LittleEndian.PutUint32(b[4:8], 0)
	copy(b[8:12], "WAVE")
	return b
}

func TestValidateBytes(t *testing.T) {
	if err := ValidateBytes(minimalWAVHeader()); err != nil {
		t.Errorf("valid WAV header: %v", err)
	}
	if err := ValidateBytes([]byte("not a wav at all")); !errors.Is(err, ErrNotAudio) {
		t.Errorf("non-WAV: got %v want ErrNotAudio", err)
	}
	if err := ValidateBytes([]byte("RIFF")); !errors.Is(err, ErrNotAudio) {
		t.Errorf("too-short: got %v want ErrNotAudio", err)
	}
}

func TestValidateURL_SchemeRejected(t *testing.T) {
	for _, raw := range []string{"", "  ", "ftp://example.com/a.wav", "file:///etc/passwd", "://broken"} {
		if err := ValidateURL(context.Background(), raw); !errors.Is(err, ErrUnsupportedScheme) {
			t.Errorf("scheme %q: got %v want ErrUnsupportedScheme", raw, err)
		}
	}
}

// TestValidateURL_WAVMagicAccepted starts a local HTTP server that
// returns a minimal RIFF/WAVE blob with Content-Type=audio/wav. The
// validator should accept this regardless of HEAD support.
func TestValidateURL_WAVMagicAccepted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/wav")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(minimalWAVHeader())
	}))
	defer srv.Close()
	if err := ValidateURL(context.Background(), srv.URL+"/welcome.wav"); err != nil {
		t.Errorf("valid wav url: %v", err)
	}
}

// TestValidateURL_NonWAVRejected verifies that even with a friendly
// audio/wav Content-Type, a non-RIFF body is rejected. Body content
// is the authoritative signal.
func TestValidateURL_NonWAVRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/wav")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ID3\x04\x00\x00\x00\x00\x00\x00not-wav"))
	}))
	defer srv.Close()
	if err := ValidateURL(context.Background(), srv.URL+"/fake.wav"); !errors.Is(err, ErrNotAudio) {
		t.Errorf("non-wav body with audio CT: got %v want ErrNotAudio", err)
	}
}

// TestValidateURL_5xxRejected covers the unreachable path.
func TestValidateURL_5xxRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	err := ValidateURL(context.Background(), srv.URL+"/x")
	if !errors.Is(err, ErrUnreachable) {
		t.Errorf("5xx: got %v want ErrUnreachable", err)
	}
}

// TestFetchPCM_CachesAfterFirstHit verifies the second call hits cache
// (no extra HTTP request observed).
func TestFetchPCM_CachesAfterFirstHit(t *testing.T) {
	PurgeCache()
	defer PurgeCache()

	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Content-Type", "audio/wav")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(minimalWAVHeader())
	}))
	defer srv.Close()

	decode := func(raw []byte, sr int) ([]byte, error) {
		return []byte{0x01, 0x02, 0x03, 0x04}, nil
	}
	if _, err := FetchPCM(context.Background(), srv.URL+"/a.wav", 16000, decode); err != nil {
		t.Fatalf("first fetch: %v", err)
	}
	if _, err := FetchPCM(context.Background(), srv.URL+"/a.wav", 16000, decode); err != nil {
		t.Fatalf("second fetch: %v", err)
	}
	if hits != 1 {
		t.Errorf("expected 1 HTTP hit (second served from cache), got %d", hits)
	}
}

// TestFetchPCM_UnreachableSurfaceErr ensures network failures map to
// ErrUnreachable so callers can fall back to scripts/welcome.wav.
func TestFetchPCM_UnreachableSurfaceErr(t *testing.T) {
	PurgeCache()
	defer PurgeCache()
	decode := func(raw []byte, sr int) ([]byte, error) { return nil, nil }
	// 127.0.0.1:1 is reliably unreachable on every CI runner.
	_, err := FetchPCM(context.Background(), "http://127.0.0.1:1/missing.wav", 16000, decode)
	if !errors.Is(err, ErrUnreachable) {
		t.Errorf("unreachable: got %v want ErrUnreachable", err)
	}
}

// TestFetchPCM_NonWAVRejected prevents misconfigured CDNs from poisoning
// the cache with non-audio content.
func TestFetchPCM_NonWAVRejected(t *testing.T) {
	PurgeCache()
	defer PurgeCache()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(strings.Repeat("X", 64)))
	}))
	defer srv.Close()
	decode := func(raw []byte, sr int) ([]byte, error) { return nil, nil }
	_, err := FetchPCM(context.Background(), srv.URL+"/x.wav", 16000, decode)
	if !errors.Is(err, ErrNotAudio) {
		t.Errorf("non-wav body: got %v want ErrNotAudio", err)
	}
}
