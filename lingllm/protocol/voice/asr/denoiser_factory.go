// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package asr

import (
	"fmt"

	"github.com/LingByte/lingllm/media/denoise"
)

// DenoiserType 降噪器类型
type DenoiserType string

const (
	// DenoiserTypeNone 无降噪
	DenoiserTypeNone DenoiserType = "none"

	// DenoiserTypeSimple — AGC/noise gate only (not far-end AEC; use audioProcessConfig type=aec)
	DenoiserTypeSimple DenoiserType = "simple"
	// DenoiserTypeAEC — NLMS far-end AEC (wired via pkg/dialog/audio UplinkProcessor)
	DenoiserTypeAEC DenoiserType = "aec"

	// DenoiserTypeRNNoise RNNoise 降噪 (需要 rnnoise build tag)
	DenoiserTypeRNNoise DenoiserType = "rnnoise"

	// DenoiserTypeLedenoise Rust nnnoiseless (需要 ledenoise build tag)
	DenoiserTypeLedenoise DenoiserType = "ledenoise"
)

// DenoiserFactory 降噪器工厂
type DenoiserFactory struct{}

// CreateDenoiser 创建降噪器组件
// 支持的类型:
//   - "none": 无降噪 (返回 nil)
//   - "simple": 简易幅度/AGC 占位（不是远端回声消除 / Speex AEC）
//   - "rnnoise": RNNoise 降噪 (需要 rnnoise build tag)
//
// config 参数:
//   - 对于 "simple": *SimpleDenoiserConfig
//   - 对于 "rnnoise": nil (RNNoise 配置固定)
//   - 对于 "none": nil
func (f *DenoiserFactory) CreateDenoiser(denoiserType DenoiserType, config interface{}) (interface{}, error) {
	switch denoiserType {
	case DenoiserTypeNone:
		return nil, nil

	case DenoiserTypeSimple:
		simpleConfig, ok := config.(*SimpleDenoiserConfig)
		if config != nil && !ok {
			return nil, fmt.Errorf("invalid config type for simple denoiser: expected *SimpleDenoiserConfig, got %T", config)
		}
		return NewSimpleDenoiserComponent(simpleConfig)

	case DenoiserTypeRNNoise:
		// 检查是否编译了 rnnoise 支持
		if !isRNNoiseAvailable() {
			return nil, fmt.Errorf("RNNoise denoiser not available: build with -tags rnnoise")
		}
		return newRNNoiseDenoiserComponent()

	case DenoiserTypeLedenoise, "native", "nnnoiseless":
		return NewLedenoiseDenoiserComponent(config)

	default:
		return nil, fmt.Errorf("unknown denoiser type: %s", denoiserType)
	}
}

// GetAvailableDenoiserTypes 获取可用的降噪器类型
func (f *DenoiserFactory) GetAvailableDenoiserTypes() []DenoiserType {
	types := []DenoiserType{
		DenoiserTypeNone,
		DenoiserTypeSimple,
	}

	// 如果编译了 rnnoise 支持，添加到列表
	if isRNNoiseAvailable() {
		types = append(types, DenoiserTypeRNNoise)
	}
	if denoise.LedenoiseEnabled() {
		types = append(types, DenoiserTypeLedenoise)
	}

	return types
}

// NewDenoiserFactory 创建降噪器工厂
func NewDenoiserFactory() *DenoiserFactory {
	return &DenoiserFactory{}
}
