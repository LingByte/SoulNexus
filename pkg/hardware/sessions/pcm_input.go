package sessions

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"fmt"
)

// PCMInputComponent 直接透传 PCM 数据。
type PCMInputComponent struct{}

func (p *PCMInputComponent) Name() string {
	return "pcm_input"
}

func (p *PCMInputComponent) Process(ctx context.Context, data interface{}) (interface{}, bool, error) {
	select {
	case <-ctx.Done():
		return nil, false, ctx.Err()
	default:
	}
	pcmData, ok := data.([]byte)
	if !ok {
		return nil, false, fmt.Errorf("invalid data type: expected []byte")
	}
	if len(pcmData) == 0 {
		return nil, false, nil
	}
	return pcmData, true, nil
}
