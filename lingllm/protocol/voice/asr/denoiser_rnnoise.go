// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

//go:build rnnoise

package asr

import (
	"fmt"

	"github.com/LingByte/lingllm/media/rnnoise"
)

// RNNoiseDenoiser RNNoise 降噪器实现
// 注意: RNNoise 固定采样率 48kHz，声道数 1，位深 16
type RNNoiseDenoiser struct {
	denoiser *rnnoise.Denoiser
}

// NewRNNoiseDenoiser 创建 RNNoise 降噪器
func NewRNNoiseDenoiser() (*RNNoiseDenoiser, error) {
	denoiser, err := rnnoise.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create RNNoise denoiser: %w", err)
	}

	return &RNNoiseDenoiser{
		denoiser: denoiser,
	}, nil
}

// Process 处理 PCM 数据
// 注意: RNNoise 要求固定大小的帧 (480 samples @ 48kHz = 960 bytes)
func (r *RNNoiseDenoiser) Process(pcm []byte) []byte {
	if r == nil || r.denoiser == nil || len(pcm) == 0 {
		return pcm
	}

	// 检查帧大小
	expectedSize := rnnoise.FrameBytes()
	if len(pcm) != expectedSize {
		// 帧大小不匹配，返回原始数据
		return pcm
	}

	output, err := r.denoiser.ProcessPCM16LE(pcm)
	if err != nil {
		// 处理失败时返回原始数据
		return pcm
	}

	return output
}

// Close 关闭处理器
func (r *RNNoiseDenoiser) Close() {
	if r == nil || r.denoiser == nil {
		return
	}
	r.denoiser.Close()
}

// RNNoiseDenoiserComponent RNNoise 降噪器组件
type RNNoiseDenoiserComponent struct {
	denoiser *RNNoiseDenoiser
}

// NewRNNoiseDenoiserComponent 创建 RNNoise 降噪器组件
func NewRNNoiseDenoiserComponent() (*RNNoiseDenoiserComponent, error) {
	denoiser, err := NewRNNoiseDenoiser()
	if err != nil {
		return nil, err
	}

	return &RNNoiseDenoiserComponent{
		denoiser: denoiser,
	}, nil
}

// Name 返回组件名称
func (c *RNNoiseDenoiserComponent) Name() string {
	return "rnnoise_denoiser"
}

// Process 处理 PCM 数据
func (c *RNNoiseDenoiserComponent) Process(ctx interface{}, data interface{}) (interface{}, bool, error) {
	pcm, ok := data.([]byte)
	if !ok {
		return nil, false, fmt.Errorf("%w: expected []byte, got %T", ErrInvalidDataType, data)
	}

	if len(pcm) == 0 || c == nil || c.denoiser == nil {
		return pcm, true, nil
	}

	return c.denoiser.Process(pcm), true, nil
}

// Close 关闭组件
func (c *RNNoiseDenoiserComponent) Close() {
	if c == nil || c.denoiser == nil {
		return
	}
	c.denoiser.Close()
}
