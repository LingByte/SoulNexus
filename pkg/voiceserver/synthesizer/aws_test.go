package synthesizer

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAws(t *testing.T) {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		t.Skip("aws region not set")
	}
	amazonTTSOption := NewAmazonTTSOption(region, "json", "111")

	ctx := context.Background()
	h := &testSynthesisHandler{}

	amazonService := NewAmazonService(amazonTTSOption)
	err := amazonService.Synthesize(ctx, h, "hello world")
	assert.Nil(t, err)
}
