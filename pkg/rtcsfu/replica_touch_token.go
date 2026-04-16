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

// MintReplicaTouchToken creates a short-lived HMAC token bound to a replica node id (for POST .../replica/touch).
// Format: base64url("v1|rtouch|node_id|expUnix") + "." + base64url(HMAC-SHA256(secret, payload)).
func MintReplicaTouchToken(secret, nodeID string, exp time.Time) (string, error) {
	if strings.TrimSpace(secret) == "" || strings.TrimSpace(nodeID) == "" {
		return "", errors.New("rtcsfu: touch secret and node_id are required")
	}
	if exp.Before(time.Now()) {
		return "", errors.New("rtcsfu: touch token expiry must be in the future")
	}
	payload := fmt.Sprintf("v1|rtouch|%s|%d", nodeID, exp.Unix())
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	sig := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." +
		base64.RawURLEncoding.EncodeToString(sig), nil
}

// VerifyReplicaTouchToken checks HMAC, node binding, and expiry.
func VerifyReplicaTouchToken(secret, token, expectedNodeID string) error {
	if strings.TrimSpace(secret) == "" || strings.TrimSpace(token) == "" {
		return errors.New("rtcsfu: missing touch secret or token")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return errors.New("rtcsfu: malformed touch token")
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return fmt.Errorf("rtcsfu: touch payload decode: %w", err)
	}
	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return fmt.Errorf("rtcsfu: touch sig decode: %w", err)
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payloadBytes)
	want := mac.Sum(nil)
	if len(sigBytes) != len(want) || !hmac.Equal(sigBytes, want) {
		return errors.New("rtcsfu: invalid touch token signature")
	}
	fields := strings.Split(string(payloadBytes), "|")
	if len(fields) != 4 || fields[0] != "v1" || fields[1] != "rtouch" {
		return errors.New("rtcsfu: invalid touch token payload")
	}
	node := fields[2]
	if node != expectedNodeID {
		return errors.New("rtcsfu: touch token node mismatch")
	}
	expUnix, err := strconv.ParseInt(fields[3], 10, 64)
	if err != nil {
		return errors.New("rtcsfu: invalid touch token expiry")
	}
	if time.Now().Unix() > expUnix {
		return errors.New("rtcsfu: touch token expired")
	}
	return nil
}
