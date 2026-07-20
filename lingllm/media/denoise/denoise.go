// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package denoise

/*
#include "denoise.h"
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// DenoiseConfig 降噪配置
type DenoiseConfig struct {
	// AEC (Acoustic Echo Cancellation) 回声消除
	AECEnable bool
	// AGC (Automatic Gain Control) 自动增益控制
	AGCEnable bool
	// 采样率 (Hz)
	SampleRate int
	// 声道数
	Channels int
	// 位深 (bits)
	BitsPerSample int
}

// DenoiseProcessor 降噪处理器
type DenoiseProcessor struct {
	handle C.denoise_handle_t
	config DenoiseConfig
}

// NewDenoiseProcessor 创建降噪处理器
func NewDenoiseProcessor(config *DenoiseConfig) (*DenoiseProcessor, error) {
	if config == nil {
		config = &DenoiseConfig{
			AECEnable:     true,
			AGCEnable:     true,
			SampleRate:    16000,
			Channels:      1,
			BitsPerSample: 16,
		}
	}

	// 创建C配置结构体
	cConfig := C.denoise_config_t{
		aec_enable:      C.bool(config.AECEnable),
		agc_enable:      C.bool(config.AGCEnable),
		sample_rate:     C.int(config.SampleRate),
		channels:        C.int(config.Channels),
		bits_per_sample: C.int(config.BitsPerSample),
	}

	// 调用C函数创建处理器
	handle := C.denoise_create(&cConfig)
	if handle == nil {
		return nil, fmt.Errorf("failed to create denoise processor")
	}

	return &DenoiseProcessor{
		handle: handle,
		config: *config,
	}, nil
}

// Process 处理音频数据
// input: 输入音频数据 (PCM格式)
// output: 输出音频数据 (降噪后)
// 返回处理的字节数和错误
func (p *DenoiseProcessor) Process(input []byte) ([]byte, error) {
	if p.handle == nil {
		return nil, fmt.Errorf("denoise processor not initialized")
	}

	if len(input) == 0 {
		return nil, fmt.Errorf("input data is empty")
	}

	// 分配输出缓冲区 (与输入大小相同)
	output := make([]byte, len(input))

	// 调用C函数处理
	inputPtr := (*C.uint8_t)(unsafe.Pointer(&input[0]))
	outputPtr := (*C.uint8_t)(unsafe.Pointer(&output[0]))
	inputLen := C.int(len(input))

	ret := C.denoise_process(p.handle, inputPtr, inputLen, outputPtr)
	if ret < 0 {
		return nil, fmt.Errorf("denoise process failed: %d", ret)
	}

	return output[:ret], nil
}

// ProcessInPlace 原地处理音频数据
func (p *DenoiseProcessor) ProcessInPlace(data []byte) error {
	if p.handle == nil {
		return fmt.Errorf("denoise processor not initialized")
	}

	if len(data) == 0 {
		return fmt.Errorf("data is empty")
	}

	dataPtr := (*C.uint8_t)(unsafe.Pointer(&data[0]))
	dataLen := C.int(len(data))

	ret := C.denoise_process_inplace(p.handle, dataPtr, dataLen)
	if ret < 0 {
		return fmt.Errorf("denoise process in-place failed: %d", ret)
	}

	return nil
}

// Reset 重置处理器状态
func (p *DenoiseProcessor) Reset() error {
	if p.handle == nil {
		return fmt.Errorf("denoise processor not initialized")
	}

	ret := C.denoise_reset(p.handle)
	if ret < 0 {
		return fmt.Errorf("denoise reset failed: %d", ret)
	}

	return nil
}

// GetConfig 获取当前配置
func (p *DenoiseProcessor) GetConfig() DenoiseConfig {
	return p.config
}

// SetAECEnable 设置AEC启用状态
func (p *DenoiseProcessor) SetAECEnable(enable bool) error {
	if p.handle == nil {
		return fmt.Errorf("denoise processor not initialized")
	}

	ret := C.denoise_set_aec_enable(p.handle, C.bool(enable))
	if ret < 0 {
		return fmt.Errorf("set AEC enable failed: %d", ret)
	}

	p.config.AECEnable = enable
	return nil
}

// SetAGCEnable 设置AGC启用状态
func (p *DenoiseProcessor) SetAGCEnable(enable bool) error {
	if p.handle == nil {
		return fmt.Errorf("denoise processor not initialized")
	}

	ret := C.denoise_set_agc_enable(p.handle, C.bool(enable))
	if ret < 0 {
		return fmt.Errorf("set AGC enable failed: %d", ret)
	}

	p.config.AGCEnable = enable
	return nil
}

// Close 关闭处理器
func (p *DenoiseProcessor) Close() error {
	if p.handle == nil {
		return nil
	}

	ret := C.denoise_destroy(p.handle)
	p.handle = nil

	if ret < 0 {
		return fmt.Errorf("denoise destroy failed: %d", ret)
	}

	return nil
}

// Version 获取降噪库版本
func Version() string {
	cVersion := C.denoise_version()
	return C.GoString(cVersion)
}
