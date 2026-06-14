// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/lingllm/synthesizer"
	voicetts "github.com/LingByte/SoulNexus/pkg/voiceserver/voice/tts"
)

// SessionWarmTexts are ultra-short phrases used to open a vendor TTS websocket
// and prime DNS/TLS without polluting the dialog. Safe to synthesize at session
// start (xiaozhi hello / gateway connect).
func SessionWarmTexts() []string {
	return []string{"嗯，", "好。"}
}

// WrapCachedTTSService wraps a Service with the process PCM cache and optionally
// prewarms common phrases at boot when VOICE_TTS_PREWARM is enabled.
func WrapCachedTTSService(inner voicetts.Service, voiceKey string) voicetts.Service {
	if inner == nil {
		return nil
	}
	voiceKey = strings.TrimSpace(voiceKey)
	if voiceKey == "" {
		voiceKey = "default"
	}
	cached, err := voicetts.NewCachingService(inner, voicetts.CacheConfig{
		VoiceKey:   voiceKey,
		MaxRunes:   80,
		ChunkBytes: 0,
	})
	if err != nil {
		return inner
	}
	if TTSPrewarmEnabled() {
		go cached.Prewarm(context.Background(), PrewarmTexts(), func(t string, e error) {
			if e != nil {
				logger.Info(fmt.Sprintf("[tts-prewarm] %q: %v", t, e))
			}
		})
	}
	return cached
}

// WarmTTSConnection opens the vendor streaming path (Volcengine WS, etc.) by
// synthesizing short session warm phrases. Best-effort; errors are logged only.
func WarmTTSConnection(ctx context.Context, svc voicetts.Service, label string) {
	if svc == nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	for _, text := range SessionWarmTexts() {
		err := svc.SynthesizeStream(ctx, text, func(pcm []byte) error {
			return nil
		})
		if err != nil && ctx.Err() == nil {
			logger.Info(fmt.Sprintf("[tts-warm] %s %q: %v", label, text, err))
		}
	}
}

// EnsureVolcengineStreaming sets streaming=true on volcengine credential maps when absent.
func EnsureVolcengineStreaming(cfg synthesizer.TTSCredentialConfig) {
	if cfg == nil {
		return
	}
	p := strings.ToLower(strings.TrimSpace(stringFromAny(cfg["provider"])))
	if p != "volcengine" && p != "volcengine_clone" {
		return
	}
	if _, ok := cfg["streaming"]; !ok {
		cfg["streaming"] = true
	}
}

func stringFromAny(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}
