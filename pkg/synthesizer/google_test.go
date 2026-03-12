package synthesizer

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithGoogleTTS(t *testing.T) {
	languageCode := os.Getenv("GOOGLE_LANGUAGE_CODE")
	if languageCode == "" {
		t.Skip("missing parameters")
	}

	opt := NewGoogleTTSOption(languageCode)
	svc := NewGoogleService(opt)

	ctx := context.Background()
	h := &testSynthesisHandler{}
	err := svc.Synthesize(ctx, h, "hello lingecho")
	assert.Nil(t, err)
}
