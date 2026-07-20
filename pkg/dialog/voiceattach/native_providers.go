package voiceattach

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/LingByte/SoulNexus/pkg/dialog/callbinding"
	"github.com/LingByte/SoulNexus/pkg/dialog/cascaded"
	"github.com/LingByte/SoulNexus/pkg/dialog/providers"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
	"github.com/LingByte/SoulNexus/pkg/dialog/voiceruntime"
	sipasr "github.com/LingByte/SoulNexus/pkg/voice/asr"
	siptts "github.com/LingByte/SoulNexus/pkg/voice/tts"
	"go.uber.org/zap"
)

var (
	errNHooksNotWired  = errors.New("native cascaded LLM: hooks not wired")
	errNTTSNilPipeline = errors.New("native cascaded TTS: nil pipeline")
)

// NativeProviderHooks wires conversation-owned LLM / turn persistence into native attach.
type NativeProviderHooks struct {
	BuildLLM           func(ctx context.Context, env VoiceEnv, callID string, lg *zap.Logger) (cascaded.LLMService, error)
	BuildTurnPersister func(env VoiceEnv, callID string, lg *zap.Logger) cascaded.TurnPersister
}

var nativeProviderHooks NativeProviderHooks

// SetNativeProviderHooks installs LLM / persister builders (call once from conversation init).
func SetNativeProviderHooks(h NativeProviderHooks) { nativeProviderHooks = h }

func buildNativeCascadedASR(env VoiceEnv, bridgeRate int, callID string, lg *zap.Logger) (cascaded.ASRRecognizer, int, error) {
	invSource := ""
	if callID != "" {
		invSource = callbinding.GetAISource(callID)
	}
	pipe, asrRate, err := sipasr.BuildPipeline(sipasr.BuildOptions{
		Env:                env,
		Logger:             lg,
		InvocationTenantID: env.TenantID,
		InvocationCallID:   callID,
		InvocationSource:   invSource,
	})
	if err != nil {
		return nil, 0, err
	}
	if lg != nil && bridgeRate > 0 && asrRate > 0 && bridgeRate != asrRate {
		lg.Info("native cascaded: ASR input will be resampled",
			zap.Int("bridge_hz", bridgeRate),
			zap.Int("asr_hz", asrRate),
			zap.String("asr_model", env.ASRModelType),
		)
	}
	return pipe, asrRate, nil
}

func buildNativeCascadedLLM(ctx context.Context, env VoiceEnv, callID string, lg *zap.Logger) (cascaded.LLMService, error) {
	if nativeProviderHooks.BuildLLM == nil {
		return nil, errNHooksNotWired
	}
	return nativeProviderHooks.BuildLLM(ctx, env, callID, lg)
}

func buildNativeTurnPersister(env VoiceEnv, callID string, lg *zap.Logger) cascaded.TurnPersister {
	if nativeProviderHooks.BuildTurnPersister == nil {
		return nil
	}
	return nativeProviderHooks.BuildTurnPersister(env, callID, lg)
}

type nativeCascadedTTS struct {
	pipe    *siptts.Pipeline
	onPCM   atomic.Pointer[func(pcm []byte) error]
	logger  *zap.Logger
	ttsPrep func(string) string
	speakMu sync.Mutex
}

func (a *nativeCascadedTTS) Speak(ctx context.Context, text string, onPCM func(pcm []byte) error) error {
	if a == nil || a.pipe == nil {
		return errNTTSNilPipeline
	}
	a.speakMu.Lock()
	defer a.speakMu.Unlock()
	a.onPCM.Store(&onPCM)
	defer a.onPCM.Store(nil)
	a.pipe.Start(ctx)
	defer func() {
		if ctx.Err() != nil {
			a.pipe.Stop()
		}
	}()
	text = strings.TrimSpace(text)
	if a.ttsPrep != nil {
		text = strings.TrimSpace(a.ttsPrep(text))
	}
	if text == "" {
		return nil
	}
	if err := a.pipe.Speak(text); err != nil {
		if errors.Is(err, context.Canceled) || ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	}
	return nil
}

func (a *nativeCascadedTTS) Finalize(ctx context.Context, onPCM func(pcm []byte) error) error {
	if a == nil || a.pipe == nil {
		return nil
	}
	a.speakMu.Lock()
	defer a.speakMu.Unlock()
	a.onPCM.Store(&onPCM)
	defer a.onPCM.Store(nil)
	if err := a.pipe.Finalize(); err != nil {
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return err
	}
	return nil
}

func nativeTTSCloudSampleRate(env VoiceEnv, synthRate, pcmBridgeSR int) int {
	return tenantcfg.TTSCloudSampleRate(tenantcfg.VoiceEnv(env), synthRate, pcmBridgeSR)
}

func buildNativeCascadedTTS(env VoiceEnv, bridgeRate int, recorderTap func(pcm []byte), callID string, lg *zap.Logger) (cascaded.TTSService, error) {
	if env.TTSConfigRaw == nil {
		return nil, fmt.Errorf("native cascaded TTS: tenant TTSConfig missing")
	}
	ttsHandle, err := siptts.NewFromCredential(env.TTSConfigRaw)
	if err != nil {
		return nil, fmt.Errorf("native cascaded TTS: provider: %w", err)
	}
	ttsCloudSR := nativeTTSCloudSampleRate(env, ttsHandle.SampleRate, bridgeRate)
	bundle := providers.ParseHotwordBundle(env.HotWordsRaw, lg, env.TTSConfigRaw)
	rt := voiceruntime.Prepare(context.Background(), tenantcfg.VoiceEnv(env), bridgeRate, lg)
	effectMix := rt.EffectMix
	adapter := &nativeCascadedTTS{logger: lg}
	if bundle.TTS != nil {
		adapter.ttsPrep = bundle.TTS.Prepare
	}
	// BridgeRate streams resample; MediaSession/RTP paces downlink.
	// PaceRealtime=false so Speak returns when synthesis finishes (not wall-clock
	// audio duration) — the next segment can dial TTS while prior audio still plays.
	invSource := ""
	if callID != "" {
		invSource = callbinding.GetAISource(callID)
	}
	pipe, err := siptts.New(siptts.Config{
		Service:       siptts.BridgeRate(ttsHandle.Service, ttsCloudSR, bridgeRate),
		SampleRate:    bridgeRate,
		Channels:      1,
		FrameDuration: 20 * time.Millisecond,
		PaceRealtime:  false,
		SendPCMFrame: func(frame []byte) error {
			if len(frame) == 0 {
				return nil
			}
			if effectMix != nil {
				frame = effectMix.MixFrame(frame)
			}
			if recorderTap != nil {
				recorderTap(frame)
			}
			cbPtr := adapter.onPCM.Load()
			if cbPtr == nil || *cbPtr == nil {
				return nil
			}
			return (*cbPtr)(frame)
		},
		Logger:             lg,
		InvocationTenantID: env.TenantID,
		InvocationCallID:   callID,
		InvocationSource:   invSource,
	})
	if err != nil {
		return nil, fmt.Errorf("native cascaded TTS: pipeline: %w", err)
	}
	adapter.pipe = pipe
	// Warm TTS before first Speak: dial WS pool (2 conns) then a discarded
	// micro-synth so TLS+encode path is hot; refill pool afterward.
	go prewarmCascadedTTS(ttsHandle.Engine, ttsHandle.Service, lg)
	return adapter, nil
}

func prewarmCascadedTTS(engine any, svc siptts.Service, lg *zap.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	if warmer, ok := engine.(interface{ Prewarm(context.Context) }); ok {
		warmer.Prewarm(ctx)
	}
	if svc != nil {
		_ = svc.SynthesizeStream(ctx, "好", func([]byte) error { return nil })
	}
	if warmer, ok := engine.(interface{ Prewarm(context.Context) }); ok {
		warmer.Prewarm(ctx)
	}
	if lg != nil {
		lg.Debug("native cascaded TTS: prewarm done")
	}
}
