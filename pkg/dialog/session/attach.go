package session

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/dialog/callbinding"
	"github.com/LingByte/SoulNexus/pkg/dialog/cascaded"
	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
	dialogrealtime "github.com/LingByte/SoulNexus/pkg/dialog/realtime"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
	"github.com/LingByte/SoulNexus/pkg/dialog/voiceruntime"
)

const defaultVADBargeIn = true

// ResolveMode picks cascaded-native / realtime / hybrid from tenant VoiceEnv.
func ResolveMode(env tenantcfg.VoiceEnv) engine.Mode {
	if strings.EqualFold(env.VoiceMode, "hybrid") && tenantcfg.RealtimeReady(env) && tenantcfg.PipelineUsable(env) {
		return engine.ModeHybrid
	}
	if strings.EqualFold(env.VoiceMode, "realtime") && tenantcfg.RealtimeReady(env) {
		return engine.ModeRealtime
	}
	return engine.ModeCascadedNative
}

// AttachEngine binds the dialog engine to port using tenant env.
func AttachEngine(ctx context.Context, p AttachParams) (engine.Detach, error) {
	if p.Port == nil {
		return nil, errors.New("dialog/session: nil MediaPort")
	}
	mode := p.Mode
	if !mode.IsValid() {
		mode = ResolveMode(p.Env)
	}
	lg := p.Logger
	if lg == nil {
		lg = engine.NopLogger{}
	}
	// Attribute AI invocation logs for voice sessions (debug/embed sources are set earlier).
	if callID := strings.TrimSpace(p.Port.CallID()); callID != "" {
		if callbinding.GetAISource(callID) == "" && !IsDebugCall(callID) {
			callbinding.SetAISource(callID, "voice")
		}
		if tid := p.Env.TenantID; tid > 0 {
			callbinding.SetTenantID(callID, tid)
		}
	}

	switch mode {
	case engine.ModeCascadedNative, engine.ModeCascaded:
		return attachCascadedNative(ctx, p, lg)
	case engine.ModeRealtime:
		return attachRealtimeNative(ctx, p, lg)
	case engine.ModeHybrid:
		return attachHybrid(ctx, p, lg)
	default:
		return nil, fmt.Errorf("dialog/session: unsupported mode %q", string(mode))
	}
}

// attachHybrid runs realtime for audio. Parallel ASR sidecar sidecar path is disabled.
func attachHybrid(ctx context.Context, p AttachParams, lg engine.Logger) (engine.Detach, error) {
	if !tenantcfg.RealtimeReady(p.Env) {
		return nil, errors.New("dialog/session: hybrid requires realtime credentials")
	}
	if !tenantcfg.PipelineUsable(p.Env) {
		return nil, errors.New("dialog/session: hybrid requires ASR/LLM/TTS pipeline credentials")
	}
	var stopASR func()
	detach, err := attachRealtimeNative(ctx, p, lg)
	if err != nil {
		if stopASR != nil {
			stopASR()
		}
		return nil, err
	}
	return func(detachCtx context.Context) error {
		if stopASR != nil {
			stopASR()
		}
		if detach != nil {
			return detach(detachCtx)
		}
		return nil
	}, nil
}

func voiceRuntimeForAttach(p AttachParams) *voiceruntime.Runtime {
	if providerHooks.PrepareVoiceRuntime == nil {
		return nil
	}
	// Debug voice sessions skip native uplink denoise/VAD — it is unnecessary
	// for browser PCM tests and has caused process crashes on some hosts.
	if IsDebugCall(p.Port.CallID()) {
		return nil
	}
	sr := p.Port.SampleRate()
	if sr <= 0 {
		sr = 16000
	}
	return providerHooks.PrepareVoiceRuntime(p.Env, sr)
}

func attachCascadedNative(ctx context.Context, p AttachParams, lg engine.Logger) (engine.Detach, error) {
	if !tenantcfg.PipelineUsable(p.Env) {
		return nil, errors.New("dialog/session: tenant pipeline credentials unusable")
	}
	if providerHooks.BuildASR == nil || providerHooks.BuildLLM == nil || providerHooks.BuildTTS == nil {
		return nil, errors.New("dialog/session: provider hooks not wired")
	}
	voiceRT := voiceRuntimeForAttach(p)

	bridgeRate := p.Port.SampleRate()
	if bridgeRate <= 0 {
		bridgeRate = 16000
	}
	callID := p.Port.CallID()
	asrSvc, asrRate, err := providerHooks.BuildASR(p.Env, bridgeRate, callID)
	if err != nil {
		return nil, fmt.Errorf("dialog/session: ASR: %w", err)
	}
	if asrRate <= 0 {
		asrRate = bridgeRate
	}
	if bridgeRate != asrRate {
		asrSvc = cascaded.WrapASRResampler(asrSvc, bridgeRate, asrRate)
	}
	// PCM ports wrap ASR uplink preprocessing here when available.
	if voiceRT != nil && !p.UplinkOnPort {
		asrSvc = cascaded.WrapASRPreprocessor(asrSvc, voiceRT.ProcessUplinkPCM)
	}
	llmSvc, err := providerHooks.BuildLLM(ctx, p.Env, callID)
	if err != nil {
		return nil, fmt.Errorf("dialog/session: LLM: %w", err)
	}
	recorderTap := func([]byte) {}
	if p.TTSRecorderTap != nil {
		recorderTap = p.TTSRecorderTap
	}
	ttsSvc, err := providerHooks.BuildTTS(p.Env, p.Port.SampleRate(), recorderTap, callID)
	if err != nil {
		return nil, fmt.Errorf("dialog/session: TTS: %w", err)
	}
	var hotword cascaded.TextRewriter
	if providerHooks.BuildHotword != nil {
		hotword = providerHooks.BuildHotword(p.Env)
	}
	var persister cascaded.TurnPersister
	if !p.SkipCallPersist && providerHooks.BuildTurnPersister != nil {
		persister = providerHooks.BuildTurnPersister(p.Env, p.Port.CallID())
	}
	if p.OnTurn != nil {
		base := persister
		persister = cascaded.TurnPersisterFunc(func(ctx context.Context, rec cascaded.TurnRecord) {
			if base != nil {
				base.PersistTurn(ctx, rec)
			}
			p.OnTurn(rec)
		})
	}

	cfg := engine.Config{
		Mode:     engine.ModeCascadedNative,
		CallID:   p.Port.CallID(),
		TenantID: p.Port.TenantID(),
	}
	opts := []cascaded.Option{
		cascaded.WithASRRecognizer(asrSvc),
		cascaded.WithLLMService(llmSvc),
		cascaded.WithTTSService(ttsSvc),
		cascaded.WithTextRewriter(hotword),
		cascaded.WithTurnPersister(persister),
	}
	// Browser debug WebSocket/WebRTC already skips denoise; also skip VAD
	// barge-in — mic echo / AEC residue during TTS falsely cancels the
	// remaining reply ("说一会就有些话没有说"). Production voice keeps VAD.
	if providerHooks.BuildVAD != nil && !IsDebugCall(p.Port.CallID()) {
		if det := providerHooks.BuildVAD(p.Env); det != nil {
			opts = append(opts, cascaded.WithVADDetector(det))
			if bp, ok := p.Port.(interface{ TriggerBargeIn() }); ok {
				opts = append(opts, cascaded.WithBargeInHandler(bp.TriggerBargeIn))
			}
			interruptPolicy := voiceruntime.InterruptionFromEnv(p.Env)
			opts = append(opts, cascaded.WithBargeInGrace(time.Duration(interruptPolicy.UnInterruptableAfterPlayStart)*time.Millisecond))
		}
	}
	eng := cascaded.New(cfg, opts...)
	return eng.Attach(ctx, p.Port, lg)
}

func attachRealtimeNative(ctx context.Context, p AttachParams, lg engine.Logger) (engine.Detach, error) {
	if !tenantcfg.RealtimeReady(p.Env) {
		return nil, errors.New("dialog/session: tenant realtime credentials unusable")
	}
	if providerHooks.BuildRealtimeAgent == nil {
		return nil, errors.New("dialog/session: realtime agent builder not wired")
	}
	builder, pacerCfg, err := providerHooks.BuildRealtimeAgent(ctx, p.Env, p.Port.CallID(), p.Port.SampleRate())
	if err != nil {
		return nil, fmt.Errorf("dialog/session: realtime agent: %w", err)
	}
	var hotword cascaded.TextRewriter
	if providerHooks.BuildHotword != nil {
		hotword = providerHooks.BuildHotword(p.Env)
	}
	var persister cascaded.TurnPersister
	if !p.SkipCallPersist && providerHooks.BuildTurnPersister != nil {
		persister = providerHooks.BuildTurnPersister(p.Env, p.Port.CallID())
	}
	if p.OnTurn != nil {
		base := persister
		persister = cascaded.TurnPersisterFunc(func(ctx context.Context, rec cascaded.TurnRecord) {
			if base != nil {
				base.PersistTurn(ctx, rec)
			}
			p.OnTurn(rec)
		})
	}

	port := p.Port
	if !p.UplinkOnPort {
		if rt := voiceRuntimeForAttach(p); rt != nil {
			port = wrapUplinkPort(p.Port, rt.ProcessUplinkPCM)
		}
	}

	cfg := engine.Config{
		Mode:     engine.ModeRealtime,
		CallID:   p.Port.CallID(),
		TenantID: p.Port.TenantID(),
	}
	opts := []dialogrealtime.Option{
		dialogrealtime.WithPacer(pacerCfg),
	}
	if hotword != nil {
		opts = append(opts, dialogrealtime.WithTextRewriter(hotword))
	}
	if persister != nil {
		opts = append(opts, dialogrealtime.WithTurnPersister(persister))
	}
	if p.OnLiveTranscript != nil {
		opts = append(opts, dialogrealtime.WithPersistObserver(p.OnLiveTranscript))
	}
	eng := dialogrealtime.New(cfg, builder, opts...)
	return eng.Attach(ctx, port, lg)
}
