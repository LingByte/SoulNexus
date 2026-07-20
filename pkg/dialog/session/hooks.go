package session

import (
	"context"
	"errors"

	"github.com/LingByte/SoulNexus/pkg/dialog/cascaded"
	"github.com/LingByte/SoulNexus/pkg/dialog/turn"
	dialogrealtime "github.com/LingByte/SoulNexus/pkg/dialog/realtime"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
	"github.com/LingByte/SoulNexus/pkg/dialog/voiceruntime"
)

// ProviderHooks wires production ASR/LLM/TTS/realtime builders from voiceattach.
type ProviderHooks struct {
	BuildASR            func(env tenantcfg.VoiceEnv, bridgeRate int, callID string) (cascaded.ASRRecognizer, int, error)
	BuildLLM            func(ctx context.Context, env tenantcfg.VoiceEnv, sessionID string) (cascaded.LLMService, error)
	BuildTTS            func(env tenantcfg.VoiceEnv, bridgeRate int, recorderTap func([]byte), callID string) (cascaded.TTSService, error)
	BuildHotword        func(env tenantcfg.VoiceEnv) cascaded.TextRewriter
	BuildVAD            func(env tenantcfg.VoiceEnv) cascaded.BargeInDetector
	BuildTurnPersister  func(env tenantcfg.VoiceEnv, sessionID string) cascaded.TurnPersister
	BuildRealtimeAgent  func(ctx context.Context, env tenantcfg.VoiceEnv, callID string, bridgeRate int) (dialogrealtime.AgentBuilder, dialogrealtime.PacerConfig, error)
	PrepareVoiceRuntime func(env tenantcfg.VoiceEnv, pcmBridgeSR int) *voiceruntime.Runtime
}

var providerHooks ProviderHooks

// SetProviderHooks installs provider builders (once at bootstrap).
func SetProviderHooks(h ProviderHooks) {
	providerHooks = h
}

var (
	peekKnowledgeRetrievals func(callID string) []turn.KnowledgeRetrievalRecord
	takeKnowledgeRetrievals func(callID string) []turn.KnowledgeRetrievalRecord
)

// SetKnowledgeRetrievalHooks wires KB buffer peek/take for debug turn metrics.
func SetKnowledgeRetrievalHooks(
	peek func(callID string) []turn.KnowledgeRetrievalRecord,
	take func(callID string) []turn.KnowledgeRetrievalRecord,
) {
	peekKnowledgeRetrievals = peek
	takeKnowledgeRetrievals = take
}

// TakeKnowledgeRetrievalsFn exposes the take hook for dialog/chat persistence.
func TakeKnowledgeRetrievalsFn() func(callID string) []turn.KnowledgeRetrievalRecord {
	return takeKnowledgeRetrievals
}

// BuildLLMForChat builds the shared cascaded LLM for text/IM dialog turns.
func BuildLLMForChat(ctx context.Context, env tenantcfg.VoiceEnv, callID string) (cascaded.LLMService, error) {
	if providerHooks.BuildLLM == nil {
		return nil, errors.New("dialog/session: LLM provider not wired")
	}
	return providerHooks.BuildLLM(ctx, env, callID)
}
