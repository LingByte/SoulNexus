package voiceattach

import (
	"context"
	"fmt"

	dialogrealtime "github.com/LingByte/SoulNexus/pkg/dialog/realtime"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
	"github.com/LingByte/lingllm/realtime"
	_ "github.com/LingByte/lingllm/realtime/aliyunomni"
	_ "github.com/LingByte/lingllm/realtime/volcdialogue"
	"go.uber.org/zap"
)

// NativeRealtimeHooks wires conversation-owned realtime preparation.
type NativeRealtimeHooks struct {
	BuildSystemPrompt func(ctx context.Context, env VoiceEnv, callID string) string
	PrepareCall       func(ctx context.Context, callID string, env VoiceEnv, lg *zap.Logger)
}

var nativeRealtimeHooks NativeRealtimeHooks

// SetNativeRealtimeHooks installs realtime call-scoped hooks (once at bootstrap).
func SetNativeRealtimeHooks(h NativeRealtimeHooks) {
	nativeRealtimeHooks = h
}

func buildNativeRealtimeAgent(
	ctx context.Context,
	env tenantcfg.VoiceEnv,
	callID string,
	bridgeRate int,
	lg *zap.Logger,
) (dialogrealtime.AgentBuilder, dialogrealtime.PacerConfig, error) {
	if !tenantcfg.RealtimeReady(env) {
		return nil, dialogrealtime.PacerConfig{}, fmt.Errorf("voiceattach: realtime credentials unusable")
	}
	sysPrompt := tenantcfg.EffectiveRealtimeOperatorCore(env)
	if nativeRealtimeHooks.BuildSystemPrompt != nil {
		if s := nativeRealtimeHooks.BuildSystemPrompt(ctx, VoiceEnv(env), callID); s != "" {
			sysPrompt = s
		}
	}
	cfg := nativeRealtimeBuilderConfig{
		CredentialCfg: env.RealtimeConfigRaw,
		Options: realtime.Options{
			SystemPrompt:     sysPrompt,
			Voice:            tenantcfg.RealtimeVoice(env),
			InputSampleRate:  tenantcfg.RealtimeInputSampleRate,
			OutputSampleRate: tenantcfg.RealtimeOutputSampleRate(env),
			Temperature:      tenantcfg.RealtimeTemperature(env),
		},
		Session:  lookupNativeRealtimeSession(callID),
		MediaCtx: nil,
	}
	if cfg.Session != nil {
		cfg.MediaCtx = cfg.Session.mediaCtx
	}
	cfg.ObserveVendorEvent = wireNativeRealtimeToolsAndTransfer(callID, env, &cfg.Options, lg)
	// Catalog usage hints may append to Options.SystemPrompt inside the wire helper.
	sysPrompt = cfg.Options.SystemPrompt
	if cfg.Session != nil {
		cfg.Session.setInstructionBase(sysPrompt)
	}
	if bridgeRate <= 0 {
		bridgeRate = 16000
	}
	pacer := dialogrealtime.PacerConfig{
		SampleRate:      bridgeRate,
		FrameMillis:     20,
		// 1 frame (~20ms) keeps jitter absorbed without adding ~60ms first-audio delay.
		PrebufferFrames: 1,
	}
	return newNativeRealtimeBuilder(cfg), pacer, nil
}
