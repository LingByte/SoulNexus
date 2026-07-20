// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"flag"
	"log"

	"github.com/LingByte/lingllm/protocol/voice/asr"
)

// DenoiserConfig 降噪器配置
type DenoiserConfig struct {
	Type   string // "none", "simple", "rnnoise"
	Simple *asr.SimpleDenoiserConfig
}

// RegisterDenoiserFlags 注册降噪器相关的命令行标志
func RegisterDenoiserFlags() *DenoiserConfig {
	config := &DenoiserConfig{
		Simple: &asr.SimpleDenoiserConfig{
			AECEnable:     true,
			AGCEnable:     true,
			SampleRate:    16000,
			Channels:      1,
			BitsPerSample: 16,
		},
	}

	flag.StringVar(&config.Type, "denoiser", "simple", "降噪器类型: none, simple, rnnoise")
	flag.BoolVar(&config.Simple.AECEnable, "aec", true, "启用 AEC (回声消除)")
	flag.BoolVar(&config.Simple.AGCEnable, "agc", true, "启用 AGC (自动增益控制)")
	flag.IntVar(&config.Simple.SampleRate, "sample_rate", 16000, "采样率")
	flag.IntVar(&config.Simple.Channels, "channels", 1, "声道数")
	flag.IntVar(&config.Simple.BitsPerSample, "bits_per_sample", 16, "位深")

	return config
}

// CreateDenoiser 创建降噪器组件
func (c *DenoiserConfig) CreateDenoiser() (interface{}, error) {
	factory := asr.NewDenoiserFactory()

	switch c.Type {
	case "none":
		return factory.CreateDenoiser(asr.DenoiserTypeNone, nil)
	case "simple":
		return factory.CreateDenoiser(asr.DenoiserTypeSimple, c.Simple)
	case "rnnoise":
		return factory.CreateDenoiser(asr.DenoiserTypeRNNoise, nil)
	default:
		log.Fatalf("未知的降噪器类型: %s", c.Type)
		return nil, nil
	}
}

// LogConfig 记录配置信息
func (c *DenoiserConfig) LogConfig() {
	log.Printf("降噪器配置:")
	log.Printf("  类型: %s", c.Type)

	if c.Type == "simple" {
		log.Printf("  AEC: %v", c.Simple.AECEnable)
		log.Printf("  AGC: %v", c.Simple.AGCEnable)
		log.Printf("  采样率: %d Hz", c.Simple.SampleRate)
		log.Printf("  声道数: %d", c.Simple.Channels)
		log.Printf("  位深: %d-bit", c.Simple.BitsPerSample)
	}
}
