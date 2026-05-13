// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package sfu

import (
	"encoding/binary"
	"testing"
)

// Tests for the small pure-function helpers across the package. Kept in
// one file because each individually is ~5 lines.

func TestPtrString(t *testing.T) {
	if ptrString("") != nil {
		t.Error("empty should be nil")
	}
	v := ptrString("x")
	if v == nil || *v != "x" {
		t.Error("non-empty should return pointer")
	}
}

func TestOrDefault(t *testing.T) {
	if orDefault("", "fb") != "fb" {
		t.Error("empty → fallback")
	}
	if orDefault("v", "fb") != "v" {
		t.Error("nonempty → value")
	}
}

func TestAppendU16U32(t *testing.T) {
	buf := appendU16LE(nil, 0x1234)
	if len(buf) != 2 || binary.LittleEndian.Uint16(buf) != 0x1234 {
		t.Errorf("u16 le wrong: %x", buf)
	}
	buf = appendU32LE(nil, 0x11223344)
	if len(buf) != 4 || binary.LittleEndian.Uint32(buf) != 0x11223344 {
		t.Errorf("u32 le wrong: %x", buf)
	}
}

func TestWrapWAVRecording(t *testing.T) {
	pcm := []byte{0x01, 0x00, 0x02, 0x00, 0x03, 0x00}
	wav := wrapWAVRecording(pcm, 48000, 1)
	if string(wav[:4]) != "RIFF" || string(wav[8:12]) != "WAVE" {
		t.Fatalf("RIFF/WAVE header missing: %x", wav[:16])
	}
	if string(wav[12:16]) != "fmt " {
		t.Fatalf("fmt chunk missing")
	}
	if string(wav[36:40]) != "data" {
		t.Fatalf("data chunk missing")
	}
	if binary.LittleEndian.Uint16(wav[22:24]) != 1 {
		t.Errorf("channels not 1")
	}
	if binary.LittleEndian.Uint32(wav[24:28]) != 48000 {
		t.Errorf("sample rate not 48000")
	}
	if binary.LittleEndian.Uint32(wav[40:44]) != uint32(len(pcm)) {
		t.Errorf("data len mismatch")
	}
	if string(wav[44:]) != string(pcm) {
		t.Errorf("pcm body not appended")
	}
}

func TestICEServersForClient(t *testing.T) {
	out := iceServersForClient(nil)
	if len(out) != 0 {
		t.Error("nil → empty slice")
	}
}

func TestNewParticipantIDUnique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		id := newParticipantID()
		if len(id) != 16 {
			t.Errorf("id len = %d", len(id))
		}
		if seen[id] {
			t.Errorf("collision: %s", id)
		}
		seen[id] = true
	}
}

func TestPermissionsDefault(t *testing.T) {
	p := DefaultPermissions()
	if !p.CanPublish || !p.CanSubscribe || !p.CanPublishData {
		t.Errorf("defaults should be full access: %+v", p)
	}
	if p.IsRecorder {
		t.Error("recorder flag should not be set by default")
	}
}

func TestNewAccessTokenValidation(t *testing.T) {
	cases := []struct {
		name   string
		secret string
		claims AccessTokenClaims
		want   bool // true=error expected
	}{
		{"empty secret", "", AccessTokenClaims{Room: "r", Identity: "i"}, true},
		{"empty room", "s", AccessTokenClaims{Identity: "i"}, true},
		{"empty identity", "s", AccessTokenClaims{Room: "r"}, true},
		{"defaults applied", "s", AccessTokenClaims{Room: "r", Identity: "i"}, false},
	}
	for _, tc := range cases {
		_, err := NewAccessToken(tc.secret, tc.claims)
		if (err != nil) != tc.want {
			t.Errorf("%s: err=%v want_err=%v", tc.name, err, tc.want)
		}
	}
}

func TestParseAccessTokenErrors(t *testing.T) {
	if _, err := ParseAccessToken("s", ""); err == nil {
		t.Error("empty token must error")
	}
	if _, err := ParseAccessToken("s", "onlyonepart"); err == nil {
		t.Error("missing dot must error")
	}
	if _, err := ParseAccessToken("s", "***.***"); err == nil {
		t.Error("bad b64 must error")
	}
}
