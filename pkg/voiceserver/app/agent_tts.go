// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/LingByte/SoulNexus/internal/models/auth"
	svcmodels "github.com/LingByte/SoulNexus/internal/models/server"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/lingllm/recognizer"
	"github.com/LingByte/lingllm/synthesizer"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/sessionctx"
	voicetts "github.com/LingByte/SoulNexus/pkg/voiceserver/voice/tts"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AgentAwareFactory struct {
	DB       *gorm.DB
	Base     pipelineFactory
	Registry *sessionctx.TTSSpecRegistry
	Log      *zap.Logger

	mu    sync.Mutex
	cache map[string]cachedTTS
}

type pipelineFactory interface {
	NewASR(ctx context.Context, callID string) (recognizer.SpeechRecognitionEngine, int, error)
	TTS(ctx context.Context, callID string) (voicetts.Service, int, error)
}

type cachedTTS struct {
	svc voicetts.Service
	sr  int
}

func NewAgentAwareFactory(db *gorm.DB, base pipelineFactory, reg *sessionctx.TTSSpecRegistry, log *zap.Logger) *AgentAwareFactory {
	if reg == nil {
		reg = sessionctx.DefaultRegistry
	}
	if log == nil {
		log = VoiceLogger("agent-tts")
	}
	return &AgentAwareFactory{
		DB:       db,
		Base:     base,
		Registry: reg,
		Log:      log,
		cache:    make(map[string]cachedTTS),
	}
}

func (f *AgentAwareFactory) NewASR(ctx context.Context, callID string) (recognizer.SpeechRecognitionEngine, int, error) {
	return f.Base.NewASR(ctx, callID)
}

func (f *AgentAwareFactory) TTS(ctx context.Context, callID string) (voicetts.Service, int, error) {
	if f == nil || f.Base == nil {
		return nil, 0, fmt.Errorf("agent tts: nil factory")
	}
	f.mu.Lock()
	if c, ok := f.cache[callID]; ok {
		f.mu.Unlock()
		return c.svc, c.sr, nil
	}
	f.mu.Unlock()

	if f.DB == nil {
		return f.Base.TTS(ctx, callID)
	}
	spec := f.Registry.Get(callID)
	if spec == nil {
		return f.Base.TTS(ctx, callID)
	}
	synth, sr, err := buildSynthesisServiceFromDialog(ctx, f.DB, spec, f.Log)
	if err != nil {
		f.Log.Warn("agent tts: fallback to env TTS", zap.String("call_id", callID), zap.Error(err))
		return f.Base.TTS(ctx, callID)
	}
	voiceKey := ttsVoiceKeyFromSpec(spec)
	svc := WrapCachedTTSService(voicetts.FromSynthesisService(synth), voiceKey)
	if svc == nil {
		return f.Base.TTS(ctx, callID)
	}
	f.mu.Lock()
	f.cache[callID] = cachedTTS{svc: svc, sr: sr}
	f.mu.Unlock()
	go WarmTTSConnection(ctx, svc, fmt.Sprintf("call=%s", callID))
	return svc, sr, nil
}

func (f *AgentAwareFactory) Release(callID string) {
	if f == nil {
		return
	}
	f.Registry.Delete(callID)
	f.mu.Lock()
	delete(f.cache, callID)
	f.mu.Unlock()
}

// SwitchSpeaker updates the per-call dialog payload speaker and rebuilds
// the cached TTS service for the call.
func (f *AgentAwareFactory) SwitchSpeaker(ctx context.Context, callID, speakerID string) (voicetts.Service, int, error) {
	if f == nil {
		return nil, 0, fmt.Errorf("agent tts: nil factory")
	}
	speakerID = strings.TrimSpace(speakerID)
	if speakerID == "" {
		return nil, 0, fmt.Errorf("agent tts: empty speaker_id")
	}
	spec := f.Registry.Get(callID)
	if spec == nil {
		return nil, 0, fmt.Errorf("agent tts: no dialog payload for call %s", callID)
	}
	spec.Speaker = speakerID
	f.Registry.Put(callID, spec)
	f.mu.Lock()
	delete(f.cache, callID)
	f.mu.Unlock()
	svc, sr, err := f.TTS(ctx, callID)
	if err != nil {
		return nil, 0, err
	}
	return svc, sr, nil
}

func buildSynthesisServiceFromDialog(ctx context.Context, db *gorm.DB, spec *sessionctx.DialogPayload, log *zap.Logger) (synthesizer.AudioSynthesisEngine, int, error) {
	_ = ctx
	cred, err := auth.GetUserCredentialByApiSecretAndApiKey(db, strings.TrimSpace(spec.APIKey), strings.TrimSpace(spec.APISecret))
	if err != nil {
		return nil, 0, err
	}
	if cred == nil {
		return nil, 0, fmt.Errorf("invalid api credentials")
	}
	agentID, err := strconv.ParseInt(strings.TrimSpace(spec.AgentID), 10, 64)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid agentId: %w", err)
	}
	var agent svcmodels.Agent
	if err := db.First(&agent, agentID).Error; err != nil {
		return nil, 0, fmt.Errorf("agent not found: %w", err)
	}
	if cred.GroupID != agent.GroupID {
		return nil, 0, fmt.Errorf("credential/agent group mismatch")
	}

	speaker := strings.TrimSpace(spec.Speaker)
	if speaker == "" {
		speaker = strings.TrimSpace(agent.Speaker)
	}
	voiceCloneID := spec.VoiceCloneID
	if voiceCloneID == nil || *voiceCloneID <= 0 {
		voiceCloneID = agent.VoiceCloneID
	}
	ttsProvider := strings.TrimSpace(spec.TtsProvider)
	if ttsProvider == "" {
		ttsProvider = strings.TrimSpace(agent.TtsProvider)
	}
	if ttsProvider == "" {
		ttsProvider = cred.GetTTSProvider()
	}
	if ttsProvider == "" {
		return nil, 0, fmt.Errorf("tts provider not configured")
	}

	if voiceCloneID != nil && *voiceCloneID > 0 {
		if svc, err := buildVoiceCloneSynthesis(db, int64(*voiceCloneID), log); err == nil && svc != nil {
			sr := 16000
			if v := utils.GetIntEnv("VOLCENGINE_CLONE_SAMPLE_RATE"); v > 0 {
				sr = int(v)
			}
			return svc, sr, nil
		} else if err != nil {
			log.Warn("agent tts: voice clone failed, using credential TTS", zap.Int("voiceCloneId", *voiceCloneID), zap.Error(err))
		}
	}

	ttsConfig := make(synthesizer.TTSCredentialConfig)
	ttsConfig["provider"] = ttsProvider
	if cred.TtsConfig != nil {
		for k, v := range cred.TtsConfig {
			ttsConfig[k] = v
		}
	}
	ttsConfig["provider"] = ttsProvider
	if speaker != "" {
		ttsConfig["voiceType"] = speaker
		ttsConfig["voice_type"] = speaker
	}
	if lang := strings.TrimSpace(spec.Language); lang != "" {
		ttsConfig["language"] = lang
	}
	EnsureVolcengineStreaming(ttsConfig)

	svc, err := synthesizer.NewAudioSynthesisEngineFromCredential(ttsConfig)
	if err != nil {
		return nil, 0, err
	}
	logger.Info(fmt.Sprintf("[agent-tts] call agent=%d provider=%s speaker=%s", agentID, ttsProvider, speaker))
	return svc, 16000, nil
}

func buildVoiceCloneSynthesis(db *gorm.DB, voiceCloneID int64, log *zap.Logger) (synthesizer.AudioSynthesisEngine, error) {
	voiceClone, err := svcmodels.GetVoiceCloneByID(db, voiceCloneID)
	if err != nil || voiceClone == nil {
		return nil, fmt.Errorf("voice clone %d: %w", voiceCloneID, err)
	}
	if strings.TrimSpace(voiceClone.AssetID) == "" {
		return nil, fmt.Errorf("voice clone %d: asset not trained", voiceCloneID)
	}
	if strings.ToLower(strings.TrimSpace(voiceClone.Provider)) != "volcengine" {
		return nil, fmt.Errorf("voice clone %d: provider %s not supported on voice plane", voiceCloneID, voiceClone.Provider)
	}

	engine, err := voicetts.NewVolcengineCloneEngineFromEnv(voiceClone.AssetID)
	if err != nil {
		return nil, err
	}
	log.Info("[agent-tts] using volcengine voice clone (env credentials)",
		zap.Int64("id", voiceCloneID),
		zap.String("assetID", voiceClone.AssetID),
	)
	return engine, nil
}

func ttsVoiceKeyFromSpec(spec *sessionctx.DialogPayload) string {
	if spec == nil {
		return "default"
	}
	if spec.VoiceCloneID != nil && *spec.VoiceCloneID > 0 {
		return fmt.Sprintf("clone-%d", *spec.VoiceCloneID)
	}
	speaker := strings.TrimSpace(spec.Speaker)
	provider := strings.TrimSpace(spec.TtsProvider)
	if speaker != "" || provider != "" {
		return fmt.Sprintf("%s-%s-%s", strings.TrimSpace(spec.AgentID), provider, speaker)
	}
	return fmt.Sprintf("agent-%s", strings.TrimSpace(spec.AgentID))
}
