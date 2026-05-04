package listeners

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitAgentListener(t *testing.T) {
	assert.NotPanics(t, func() {
		InitAgentListener()
	})
}

func TestAgentListener_Initialization(t *testing.T) {
	for i := 0; i < 3; i++ {
		assert.NotPanics(t, func() {
			InitAgentListener()
		})
	}
}

func BenchmarkInitAgentListener(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		InitAgentListener()
	}
}
