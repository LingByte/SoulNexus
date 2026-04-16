// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package rtcsfu

import (
	"testing"
	"time"
)

func TestReplicaTouchToken_RoundTrip(t *testing.T) {
	const sec = "touch-secret-9"
	exp := time.Now().Add(2 * time.Minute).UTC()
	tok, err := MintReplicaTouchToken(sec, "rep-a", exp)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyReplicaTouchToken(sec, tok, "rep-a"); err != nil {
		t.Fatal(err)
	}
	if err := VerifyReplicaTouchToken(sec, tok, "rep-b"); err == nil {
		t.Fatal("expected node mismatch")
	}
	if err := VerifyReplicaTouchToken("wrong", tok, "rep-a"); err == nil {
		t.Fatal("expected bad secret")
	}
}
