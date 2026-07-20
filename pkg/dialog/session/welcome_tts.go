package session

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
)

const welcomeFrameDur = 20 * time.Millisecond

// StartAssistantWelcomePrewarm is a no-op without welcome prewarm buffers.
// Welcome audio is synthesized on demand in PlayAssistantWelcome.
func StartAssistantWelcomePrewarm(callID string, env tenantcfg.VoiceEnv, bridgeRate int) {
	_ = callID
	_ = env
	_ = bridgeRate
}

// PlayAssistantWelcome synthesizes assistant welcome via providerHooks.BuildTTS
// and emits PCM16LE mono at bridgeRate through emitPCM.
//
// paceRealtime should be true for transports without a downlink pacer
// (WebSocket); WebRTC passes false and relies on its pacer.
func PlayAssistantWelcome(ctx context.Context, callID string, env tenantcfg.VoiceEnv, bridgeRate int, emitPCM func([]byte) error, paceRealtime bool) (string, error) {
	welcome := strings.TrimSpace(tenantcfg.ResolvedAssistantWelcome(env))
	if welcome == "" {
		return "", nil
	}
	if ResolveMode(env) == engine.ModeRealtime {
		return welcome, nil
	}
	if bridgeRate <= 0 {
		bridgeRate = 16000
	}
	if providerHooks.BuildTTS == nil {
		return welcome, fmt.Errorf("tts provider not wired")
	}
	ttsSvc, err := providerHooks.BuildTTS(env, bridgeRate, nil, strings.TrimSpace(callID))
	if err != nil {
		return welcome, fmt.Errorf("tts provider: %w", err)
	}
	onPCM := func(pcm []byte) error {
		if len(pcm) == 0 || emitPCM == nil {
			return nil
		}
		if err := emitPCM(pcm); err != nil {
			return err
		}
		if paceRealtime {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(welcomeFrameDur):
			}
		}
		return nil
	}
	if err := ttsSvc.Speak(ctx, welcome, onPCM); err != nil && ctx.Err() == nil {
		return welcome, err
	}
	_ = ttsSvc.Finalize(ctx, onPCM)
	return welcome, nil
}

func emitWelcomePCM(ctx context.Context, pcm []byte, bridgeRate int, emitPCM func([]byte) error, paceRealtime bool) error {
	if emitPCM == nil || len(pcm) == 0 {
		return nil
	}
	// mono s16le: bytes/frame = rate * 0.02 * 2
	frameBytes := bridgeRate / 50 * 2
	if frameBytes <= 0 {
		frameBytes = 640
	}
	for len(pcm) > 0 {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		n := frameBytes
		if n > len(pcm) {
			n = len(pcm)
		}
		chunk := append([]byte(nil), pcm[:n]...)
		pcm = pcm[n:]
		if err := emitPCM(chunk); err != nil {
			return err
		}
		if paceRealtime {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(welcomeFrameDur):
			}
		}
	}
	return nil
}
