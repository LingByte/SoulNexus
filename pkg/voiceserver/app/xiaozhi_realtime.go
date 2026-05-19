// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/realtime"
	_ "github.com/LingByte/SoulNexus/pkg/realtime/aliyunomni" // register aliyun_omni provider
	"github.com/LingByte/SoulNexus/pkg/utils"
)

// XiaozhiRealtimeFactory builds per-session pkg/realtime.Agent instances from
// REALTIME_* environment variables.
type XiaozhiRealtimeFactory struct {
	credential    map[string]any
	instructions  string
	voice         string
	inputSR       int
	outputSR      int
	temperature   float64
	providerLabel string
}

// NewXiaozhiRealtimeFactory reads REALTIME_* env and validates the provider
// is registered. Returns an error when apiKey or provider is missing.
func NewXiaozhiRealtimeFactory() (*XiaozhiRealtimeFactory, error) {
	provider := strings.ToLower(strings.TrimSpace(firstNonEmpty(
		utils.GetEnv("REALTIME_PROVIDER"),
		"aliyun_omni",
	)))
	apiKey := strings.TrimSpace(firstNonEmpty(
		utils.GetEnv("REALTIME_API_KEY"),
		utils.GetEnv("DASHSCOPE_API_KEY"),
	))
	if apiKey == "" {
		return nil, fmt.Errorf("missing REALTIME_API_KEY (or DASHSCOPE_API_KEY)")
	}
	if realtime.Lookup(provider) == nil {
		return nil, fmt.Errorf("unknown REALTIME_PROVIDER %q (registered: %v)", provider, realtime.RegisteredProviders())
	}

	inputSR := envIntDefault("REALTIME_INPUT_SAMPLE_RATE", 16000)
	outputSR := envIntDefault("REALTIME_OUTPUT_SAMPLE_RATE", 24000)
	if inputSR <= 0 {
		inputSR = 16000
	}
	if outputSR <= 0 {
		outputSR = 24000
	}

	cfg := map[string]any{
		"provider": provider,
		"apiKey":   apiKey,
	}
	if v := strings.TrimSpace(utils.GetEnv("REALTIME_MODEL")); v != "" {
		cfg["model"] = v
	}
	if v := strings.TrimSpace(utils.GetEnv("REALTIME_BASE_URL")); v != "" {
		cfg["baseUrl"] = v
	}
	if v := strings.TrimSpace(utils.GetEnv("REALTIME_VOICE")); v != "" {
		cfg["voice"] = v
	}
	if v := strings.TrimSpace(firstNonEmpty(
		utils.GetEnv("REALTIME_INSTRUCTIONS"),
		utils.GetEnv("REALTIME_SYSTEM_PROMPT"),
	)); v != "" {
		cfg["instructions"] = v
	}
	if ms := envIntDefault("REALTIME_DIAL_TIMEOUT_MS", 10000); ms > 0 {
		cfg["dialTimeoutMs"] = ms
	}
	// Cloud server VAD (turn_detection). Omit env vars to use aliyunomni defaults
	// (threshold 0.75, silence 1000 ms — less sensitive than DashScope 0.5 / 800 ms).
	if v := strings.TrimSpace(utils.GetEnv("REALTIME_VAD_TYPE")); v != "" {
		cfg["vadType"] = v
	}
	if utils.GetEnv("REALTIME_VAD_THRESHOLD") != "" {
		cfg["vadThreshold"] = utils.GetFloatEnvWithDefault("REALTIME_VAD_THRESHOLD", 0.75)
	}
	if utils.GetEnv("REALTIME_VAD_SILENCE_DURATION_MS") != "" {
		cfg["vadSilenceDurationMs"] = envIntDefault("REALTIME_VAD_SILENCE_DURATION_MS", 1000)
	}
	if utils.GetEnv("REALTIME_VAD_PREFIX_PADDING_MS") != "" {
		cfg["vadPrefixPaddingMs"] = envIntDefault("REALTIME_VAD_PREFIX_PADDING_MS", 300)
	}
	if utils.GetEnv("REALTIME_VAD_INTERRUPT_RESPONSE") != "" {
		cfg["vadInterruptResponse"] = utils.GetBoolEnv("REALTIME_VAD_INTERRUPT_RESPONSE")
	}
	if utils.GetEnv("REALTIME_VAD_CREATE_RESPONSE") != "" {
		cfg["vadCreateResponse"] = utils.GetBoolEnv("REALTIME_VAD_CREATE_RESPONSE")
	}

	instructions := stringField(cfg, "instructions")
	voice := stringField(cfg, "voice")

	var temperature float64
	if utils.GetEnv("REALTIME_TEMPERATURE") != "" {
		temperature = utils.GetFloatEnvWithDefault("REALTIME_TEMPERATURE", 0)
	}

	return &XiaozhiRealtimeFactory{
		credential:    cfg,
		instructions:  instructions,
		voice:         voice,
		inputSR:       inputSR,
		outputSR:      outputSR,
		temperature:   temperature,
		providerLabel: provider,
	}, nil
}

// Provider returns the realtime provider slug (for logs).
func (f *XiaozhiRealtimeFactory) Provider() string {
	if f == nil {
		return ""
	}
	return f.providerLabel
}

// NewAgent implements xiaozhi.RealtimeAgentFactory.
func (f *XiaozhiRealtimeFactory) NewAgent(ctx context.Context, _ string, onEvent func(realtime.Event)) (realtime.Agent, int, int, error) {
	if f == nil {
		return nil, 0, 0, fmt.Errorf("nil realtime factory")
	}
	if onEvent == nil {
		return nil, 0, 0, fmt.Errorf("realtime: OnEvent is required")
	}
	opts := realtime.Options{
		SystemPrompt:     f.instructions,
		Voice:            f.voice,
		InputSampleRate:  f.inputSR,
		OutputSampleRate: f.outputSR,
		Temperature:      f.temperature,
		OnEvent:          onEvent,
	}
	agent, err := realtime.NewAgentFromCredential(f.credential, opts)
	if err != nil {
		return nil, 0, 0, err
	}
	if err := agent.Start(ctx); err != nil {
		_ = agent.Close()
		return nil, 0, 0, err
	}
	return agent, f.inputSR, f.outputSR, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func envIntDefault(key string, def int) int {
	if utils.GetEnv(key) == "" {
		return def
	}
	return int(utils.GetIntEnv(key))
}

func stringField(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return strings.TrimSpace(s)
		}
	}
	return ""
}
