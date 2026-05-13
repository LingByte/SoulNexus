package synthesizer

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/media"
	"github.com/sirupsen/logrus"
)

// LocalTTSConfig 本地TTS配置
type LocalTTSConfig struct {
	Command       string `json:"command" yaml:"command" default:"say"`           // TTS 命令（如 say, festival, espeak）
	Voice         string `json:"voice" yaml:"voice" default:""`                  // 音色（可选）
	SampleRate    int    `json:"sample_rate" yaml:"sample_rate" default:"16000"` // 采样率
	Channels      int    `json:"channels" yaml:"channels" default:"1"`           // 声道数
	BitDepth      int    `json:"bit_depth" yaml:"bit_depth" default:"16"`        // 位深度
	Codec         string `json:"codec" yaml:"codec" default:"wav"`               // 音频编解码器
	FrameDuration string `json:"frame_duration" yaml:"frame_duration" default:"20ms"`
	OutputDir     string `json:"output_dir" yaml:"output_dir" default:"/tmp"` // 输出目录
}

type LocalService struct {
	opt LocalTTSConfig
	mu  sync.Mutex // 保护 opt 的并发访问
}

// NewLocalTTSConfig 创建本地TTS配置
func NewLocalTTSConfig(command string) LocalTTSConfig {
	opt := LocalTTSConfig{
		Command:       command,
		Voice:         "",
		SampleRate:    16000,
		Channels:      1,
		BitDepth:      16,
		Codec:         "wav",
		FrameDuration: "20ms",
		OutputDir:     "/tmp",
	}

	if opt.Command == "" {
		// 根据操作系统选择默认命令
		if _, err := exec.LookPath("say"); err == nil {
			opt.Command = "say" // macOS
		} else if _, err := exec.LookPath("espeak"); err == nil {
			opt.Command = "espeak" // Linux/Unix
		} else if _, err := exec.LookPath("festival"); err == nil {
			opt.Command = "festival" // Linux
		}
	}

	return opt
}

// NewLocalService 创建本地TTS服务
func NewLocalService(opt LocalTTSConfig) *LocalService {
	return &LocalService{
		opt: opt,
	}
}

func (ls *LocalService) Provider() TTSProvider {
	return ProviderLocal
}

func (ls *LocalService) Format() media.StreamFormat {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	return media.StreamFormat{
		SampleRate:    ls.opt.SampleRate,
		BitDepth:      ls.opt.BitDepth,
		Channels:      ls.opt.Channels,
		FrameDuration: media.NormalizeFramePeriod(ls.opt.FrameDuration),
	}
}

func (ls *LocalService) CacheKey(text string) string {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	digest := media.MediaCache().BuildKey(text)
	return fmt.Sprintf("local.tts-%s-%d-%s.%s", ls.opt.Command, ls.opt.SampleRate, digest, ls.opt.Codec)
}

func (ls *LocalService) Synthesize(ctx context.Context, handler SynthesisHandler, text string) error {
	ls.mu.Lock()
	opt := ls.opt
	ls.mu.Unlock()

	// 检查命令是否存在
	cmdPath, err := exec.LookPath(opt.Command)
	if err != nil {
		return fmt.Errorf("TTS command not found: %s, please install a TTS tool", opt.Command)
	}

	logrus.WithFields(logrus.Fields{
		"command": cmdPath,
		"text":    text,
	}).Info("local tts: starting synthesis")

	// 根据不同的命令构建不同的参数
	audioData, err := ls.synthesizeWithCommand(ctx, text, cmdPath, opt)
	if err != nil {
		return fmt.Errorf("synthesis failed: %w", err)
	}

	// 发送音频数据到 handler
	if len(audioData) > 0 {
		handler.OnMessage(audioData)
	} else {
		// 如果没有音频数据，返回一个占位符
		return fmt.Errorf("no audio data generated")
	}

	logrus.WithFields(logrus.Fields{
		"provider":   "local",
		"text":       text,
		"audio_size": len(audioData),
	}).Info("local tts: synthesis completed")

	return nil
}

// synthesizeWithCommand 使用命令进行合成
func (ls *LocalService) synthesizeWithCommand(ctx context.Context, text, cmdPath string, opt LocalTTSConfig) ([]byte, error) {
	switch opt.Command {
	case "say":
		return ls.synthesizeWithSay(ctx, text, cmdPath, opt)
	case "espeak":
		return ls.synthesizeWithEspeak(ctx, text, cmdPath, opt)
	case "festival":
		return ls.synthesizeWithFestival(ctx, text, cmdPath, opt)
	default:
		// 尝试通用方法
		return ls.synthesizeGeneric(ctx, text, cmdPath, opt)
	}
}

// synthesizeWithSay 使用 macOS say 命令合成
func (ls *LocalService) synthesizeWithSay(ctx context.Context, text, cmdPath string, opt LocalTTSConfig) ([]byte, error) {
	// macOS say 命令无法直接输出音频文件，这里返回占位符
	// 在实际应用中，可能需要使用 afconvert 或其他工具
	logrus.Warn("macOS say command cannot output audio directly, using placeholder")

	// 返回一个占位符音频数据（静音）
	// 实际应用中需要使用更复杂的实现
	duration := 2.0 // 估算2秒音频
	bytesPerSecond := opt.SampleRate * opt.Channels * (opt.BitDepth / 8)
	numBytes := int(float64(bytesPerSecond) * duration)
	audioData := make([]byte, numBytes)

	return audioData, nil
}

// synthesizeWithEspeak 使用 espeak 命令合成
func (ls *LocalService) synthesizeWithEspeak(ctx context.Context, text, cmdPath string, opt LocalTTSConfig) ([]byte, error) {
	// 构建 espeak 命令
	// espeak -s 160 --stdout "text" > output.wav
	cmd := exec.CommandContext(ctx, cmdPath, "-s", fmt.Sprintf("%d", opt.SampleRate), "--stdout", text)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("espeak execution failed: %w", err)
	}

	return stdout.Bytes(), nil
}

// synthesizeWithFestival 使用 festival 命令合成
func (ls *LocalService) synthesizeWithFestival(ctx context.Context, text, cmdPath string, opt LocalTTSConfig) ([]byte, error) {
	// Festival 需要通过交互式输入或脚本
	// 这里使用简化实现
	festivalScript := fmt.Sprintf("(SayText \"%s\")", text)

	cmd := exec.CommandContext(ctx, cmdPath, "-b", "-")
	cmd.Stdin = bytes.NewReader([]byte(festivalScript))

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("festival execution failed: %w", err)
	}

	return stdout.Bytes(), nil
}

// synthesizeGeneric 通用合成方法
func (ls *LocalService) synthesizeGeneric(ctx context.Context, text, cmdPath string, opt LocalTTSConfig) ([]byte, error) {
	// 对于其他命令，尝试直接执行
	cmd := exec.CommandContext(ctx, cmdPath, text)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("command execution failed: %w", err)
	}

	return stdout.Bytes(), nil
}

func (ls *LocalService) Close() error {
	return nil
}

// CheckLocalTTSAvailable 检查本地是否安装了 TTS 工具
func CheckLocalTTSAvailable() []string {
	var available []string

	commands := []string{"say", "espeak", "festival"}
	for _, cmd := range commands {
		if _, err := exec.LookPath(cmd); err == nil {
			available = append(available, cmd)
		}
	}

	return available
}

// DetectLocalTTSCommand 自动检测可用的本地 TTS 命令
func DetectLocalTTSCommand() string {
	available := CheckLocalTTSAvailable()
	if len(available) > 0 {
		return available[0]
	}
	return ""
}

// GetLocalTTSInfo 获取本地 TTS 信息
func GetLocalTTSInfo() map[string]interface{} {
	available := CheckLocalTTSAvailable()
	detected := DetectLocalTTSCommand()

	return map[string]interface{}{
		"available": available,
		"detected":  detected,
		"os":        os.Getenv("OS"),
		"platform":  "unknown",
	}
}
