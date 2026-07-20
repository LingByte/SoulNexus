// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package asr

import (
	"fmt"

	"github.com/LingByte/lingllm/media/denoise"
)

// SimpleDenoiserConfig 简单降噪器配置
type SimpleDenoiserConfig struct {
	AECEnable     bool
	AGCEnable     bool
	SampleRate    int
	Channels      int
	BitsPerSample int
}

// SimpleDenoiser 简单降噪器实现
type SimpleDenoiser struct {
	processor *denoise.DenoiseProcessor
}

// NewSimpleDenoiser 创建简单降噪器
func NewSimpleDenoiser(config *SimpleDenoiserConfig) (*SimpleDenoiser, error) {
	if config == nil {
		config = &SimpleDenoiserConfig{
			AECEnable:     true,
			AGCEnable:     true,
			SampleRate:    16000,
			Channels:      1,
			BitsPerSample: 16,
		}
	}

	processor, err := denoise.NewDenoiseProcessor(&denoise.DenoiseConfig{
		AECEnable:     config.AECEnable,
		AGCEnable:     config.AGCEnable,
		SampleRate:    config.SampleRate,
		Channels:      config.Channels,
		BitsPerSample: config.BitsPerSample,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create simple denoiser: %w", err)
	}

	return &SimpleDenoiser{
		processor: processor,
	}, nil
}

// Process 处理 PCM 数据
func (s *SimpleDenoiser) Process(pcm []byte) []byte {
	if s == nil || s.processor == nil || len(pcm) == 0 {
		return pcm
	}

	output, err := s.processor.Process(pcm)
	if err != nil {
		// 处理失败时返回原始数据
		return pcm
	}

	return output
}

// Close 关闭处理器
func (s *SimpleDenoiser) Close() error {
	if s == nil || s.processor == nil {
		return nil
	}
	return s.processor.Close()
}

// SimpleDenoiserComponent 简单降噪器组件
type SimpleDenoiserComponent struct {
	denoiser *SimpleDenoiser
}

// NewSimpleDenoiserComponent 创建简单降噪器组件
func NewSimpleDenoiserComponent(config *SimpleDenoiserConfig) (*SimpleDenoiserComponent, error) {
	denoiser, err := NewSimpleDenoiser(config)
	if err != nil {
		return nil, err
	}

	return &SimpleDenoiserComponent{
		denoiser: denoiser,
	}, nil
}

// Name 返回组件名称
func (c *SimpleDenoiserComponent) Name() string {
	return "simple_denoiser"
}

// Process 处理 PCM 数据
func (c *SimpleDenoiserComponent) Process(ctx interface{}, data interface{}) (interface{}, bool, error) {
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
func (c *SimpleDenoiserComponent) Close() error {
	if c == nil || c.denoiser == nil {
		return nil
	}
	return c.denoiser.Close()
}
