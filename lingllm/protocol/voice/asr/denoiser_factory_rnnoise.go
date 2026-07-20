// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

//go:build rnnoise

package asr

// isRNNoiseAvailable 在 rnnoise build tag 下返回 true
func isRNNoiseAvailable() bool {
	return true
}

// newRNNoiseDenoiserComponent 在 rnnoise build tag 下创建 RNNoise 组件
func newRNNoiseDenoiserComponent() (interface{}, error) {
	return NewRNNoiseDenoiserComponent()
}
