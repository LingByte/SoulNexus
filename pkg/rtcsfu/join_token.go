// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package rtcsfu

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// MintJoinToken creates a signed token bound to roomID with absolute expiry.
// Format: base64url("v1|room_id|expUnix") + "." + base64url(HMAC-SHA256(secret, payload)).
func MintJoinToken(secret, roomID string, exp time.Time) (token string, err error) {
	if strings.TrimSpace(secret) == "" || strings.TrimSpace(roomID) == "" {
		return "", errors.New("rtcsfu: secret and room_id are required")
	}
	if exp.Before(time.Now()) {
		return "", errors.New("rtcsfu: expiry must be in the future")
	}
	payload := fmt.Sprintf("v1|%s|%d", roomID, exp.Unix())
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	sig := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." +
		base64.RawURLEncoding.EncodeToString(sig), nil
}

// VerifyJoinToken checks HMAC, room binding, and expiry.
func VerifyJoinToken(secret, token, expectedRoom string) error {
	if strings.TrimSpace(secret) == "" || strings.TrimSpace(token) == "" {
		return errors.New("rtcsfu: missing secret or token")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return errors.New("rtcsfu: malformed token")
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return fmt.Errorf("rtcsfu: payload decode: %w", err)
	}
	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return fmt.Errorf("rtcsfu: sig decode: %w", err)
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payloadBytes)
	want := mac.Sum(nil)
	if len(sigBytes) != len(want) || !hmac.Equal(sigBytes, want) {
		return errors.New("rtcsfu: invalid token signature")
	}
	fields := strings.Split(string(payloadBytes), "|")
	if len(fields) != 3 || fields[0] != "v1" {
		return errors.New("rtcsfu: invalid token payload")
	}
	room := fields[1]
	if room != expectedRoom {
		return errors.New("rtcsfu: token room mismatch")
	}
	expUnix, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil {
		return errors.New("rtcsfu: invalid token expiry")
	}
	if time.Now().Unix() > expUnix {
		return errors.New("rtcsfu: token expired")
	}
	return nil
}
