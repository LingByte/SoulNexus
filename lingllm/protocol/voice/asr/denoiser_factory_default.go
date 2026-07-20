// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

//go:build !rnnoise

package asr

import "fmt"

// isRNNoiseAvailable 检查 RNNoise 是否可用 (默认: 不可用)
func isRNNoiseAvailable() bool {
	return false
}

// newRNNoiseDenoiserComponent 创建 RNNoise 组件 (默认: 返回错误)
func newRNNoiseDenoiserComponent() (interface{}, error) {
	return nil, fmt.Errorf("RNNoise denoiser not available: build with -tags rnnoise")
}
