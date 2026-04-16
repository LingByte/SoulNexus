// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package rtcsfu

import (
	"testing"
	"time"
)

func TestMintJoinTokenRoundTrip(t *testing.T) {
	const secret = "test-secret-32bytes-long!!!!"
	exp := time.Now().Add(time.Hour).Truncate(time.Second)
	tok, err := MintJoinToken(secret, "room-a", exp)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyJoinToken(secret, tok, "room-a"); err != nil {
		t.Fatal(err)
	}
	if err := VerifyJoinToken(secret, tok, "room-b"); err == nil {
		t.Fatal("expected room mismatch")
	}
}

func TestVerifyJoinTokenWrongSecret(t *testing.T) {
	tok, err := MintJoinToken("good-secret", "room-x", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if VerifyJoinToken("other-secret", tok, "room-x") == nil {
		t.Fatal("expected verify failure")
	}
}
