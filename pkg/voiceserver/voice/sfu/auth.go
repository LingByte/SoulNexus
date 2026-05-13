// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package sfu

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// AccessToken is a compact signed token carrying everything the SFU
// needs to admit a participant into a room. Shape is deliberately
// similar to JWT but we avoid the dependency: header is implicit
// ("HS256" only), payload is our own struct, signature is HMAC-SHA256
// of `base64(payload) . ""` (no header segment) keyed with Config.AuthSecret.
//
// Token on the wire:
//
//	base64url(payload) + "." + base64url(signature)
//
// Example issuance (business backend, same Go module):
//
//	tok, _ := sfu.NewAccessToken(secret, sfu.AccessTokenClaims{
//	    Room: "team-standup",
//	    Identity: "alice@example.com",
//	    Name: "Alice",
//	    ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
//	    Permissions: &sfu.Permissions{CanPublish: true, CanSubscribe: true},
//	})
//
// ParseAccessToken verifies the signature + expiry and returns the
// claims. The SFU trusts the token's Room / Identity / Permissions
// verbatim.
type AccessToken struct {
	raw    string
	Claims AccessTokenClaims
}

// AccessTokenClaims is the token payload. Unix-second timestamps are
// used so tokens survive JSON/JS roundtrips without precision drift.
type AccessTokenClaims struct {
	// Room is the room name the holder is authorised to join. Required.
	Room string `json:"room"`
	// Identity uniquely identifies the participant across sessions
	// (e.g. account ID or email). Required. Must be ≤ 256 runes.
	Identity string `json:"identity"`
	// Name is a human-readable display name shown to other peers.
	// Optional; falls back to Identity.
	Name string `json:"name,omitempty"`
	// ExpiresAt is Unix seconds; the SFU rejects tokens past this
	// moment (with a small clock-skew grace window). Required.
	ExpiresAt int64 `json:"exp"`
	// IssuedAt is Unix seconds (optional, informational).
	IssuedAt int64 `json:"iat,omitempty"`
	// Permissions limits what the holder can do in the room. nil = full
	// access (publish + subscribe + publish data).
	Permissions *Permissions `json:"permissions,omitempty"`
	// Metadata is opaque data forwarded to other participants in their
	// ParticipantInfo — useful for attaching avatar URLs, roles, etc.
	// The SFU never parses it.
	Metadata string `json:"metadata,omitempty"`
}

// Permissions controls what a participant can do once inside a room.
// All fields default to true when the enclosing Permissions pointer is
// nil, giving "full access" semantics for unspecified tokens.
type Permissions struct {
	CanPublish    bool `json:"canPublish"`
	CanSubscribe  bool `json:"canSubscribe"`
	CanPublishData bool `json:"canPublishData"`
	// IsRecorder marks a server-side recorder bot; it gets subscribe
	// permission without counting against room capacity. Unused by the
	// browser-facing flow today; reserved for future bot integration.
	IsRecorder bool `json:"isRecorder,omitempty"`
}

// DefaultPermissions returns full-access permissions used when a token
// omits the Permissions field.
func DefaultPermissions() Permissions {
	return Permissions{
		CanPublish:     true,
		CanSubscribe:   true,
		CanPublishData: true,
	}
}

// clockSkewGrace is how much slack ParseAccessToken allows past
// ExpiresAt to tolerate minor client/server clock differences.
const clockSkewGrace = 30 * time.Second

// NewAccessToken mints a signed token for the supplied claims.
// ExpiresAt defaults to one hour from now if zero. Identity and Room
// are required; anything else can be left blank.
func NewAccessToken(secret string, claims AccessTokenClaims) (string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", errors.New("sfu: auth secret is required")
	}
	if strings.TrimSpace(claims.Room) == "" {
		return "", errors.New("sfu: token room is required")
	}
	if strings.TrimSpace(claims.Identity) == "" {
		return "", errors.New("sfu: token identity is required")
	}
	if claims.ExpiresAt == 0 {
		claims.ExpiresAt = time.Now().Add(time.Hour).Unix()
	}
	if claims.IssuedAt == 0 {
		claims.IssuedAt = time.Now().Unix()
	}
	payload, err := json.Marshal(&claims)
	if err != nil {
		return "", fmt.Errorf("sfu: marshal claims: %w", err)
	}
	b64 := base64.RawURLEncoding.EncodeToString(payload)
	sig := hmacSign(secret, b64)
	return b64 + "." + sig, nil
}

// ParseAccessToken verifies `tok` against `secret` and returns the
// claims. Errors if the signature doesn't match, the token is
// malformed, or it has expired beyond the clock-skew grace window.
func ParseAccessToken(secret, tok string) (*AccessToken, error) {
	if strings.TrimSpace(tok) == "" {
		return nil, errors.New("sfu: empty token")
	}
	parts := strings.Split(tok, ".")
	if len(parts) != 2 {
		return nil, errors.New("sfu: bad token format")
	}
	expected := hmacSign(secret, parts[0])
	// Constant-time compare to avoid leaking bytes via timing.
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return nil, errors.New("sfu: bad signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("sfu: decode payload: %w", err)
	}
	var claims AccessTokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("sfu: decode claims: %w", err)
	}
	if claims.ExpiresAt == 0 {
		return nil, errors.New("sfu: token missing expiry")
	}
	if time.Now().After(time.Unix(claims.ExpiresAt, 0).Add(clockSkewGrace)) {
		return nil, errors.New("sfu: token expired")
	}
	if strings.TrimSpace(claims.Room) == "" || strings.TrimSpace(claims.Identity) == "" {
		return nil, errors.New("sfu: token missing room or identity")
	}
	return &AccessToken{raw: tok, Claims: claims}, nil
}

// hmacSign returns the base64url-encoded HMAC-SHA256 of `msg` keyed
// with `secret`. Used both for issuance and verification so they agree
// on encoding exactly.
func hmacSign(secret, msg string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(msg))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
