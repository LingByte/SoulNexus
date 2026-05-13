package utils

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSignal(t *testing.T) {
	var val string
	var eid uint
	eid = Sig().Connect("mock_test", func(sender any, params ...any) {
		val = sender.(string)
		assert.True(t, Sig().inLoop)
		Sig().Disconnect("mock_test", eid)
	})
	Sig().Emit("mock_test", "unittest")
	assert.Equal(t, val, "unittest")
	assert.Equal(t, 0, len(Sig().events))
	Sig().Clear("mock_test")
	assert.Equal(t, 0, len(Sig().sigHandlers))
}
