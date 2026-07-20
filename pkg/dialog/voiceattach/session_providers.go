package voiceattach

import (
	"context"

	"github.com/LingByte/SoulNexus/pkg/dialog/cascaded"
	dialogrealtime "github.com/LingByte/SoulNexus/pkg/dialog/realtime"
	"github.com/LingByte/SoulNexus/pkg/dialog/session"
	"github.com/LingByte/SoulNexus/pkg/dialog/providers"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
	"github.com/LingByte/SoulNexus/pkg/dialog/voiceruntime"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"go.uber.org/zap"
)

// SessionProviderHooks returns provider builders wired for pkg/dialog/session.
func SessionProviderHooks(lg *zap.Logger) session.ProviderHooks {
	if lg == nil {
		lg = logger.Lg
	}
	return session.ProviderHooks{
		BuildASR: func(env tenantcfg.VoiceEnv, bridgeRate int, callID string) (cascaded.ASRRecognizer, int, error) {
			return buildNativeCascadedASR(VoiceEnv(env), bridgeRate, callID, lg)
		},
		BuildLLM: func(ctx context.Context, env tenantcfg.VoiceEnv, sessionID string) (cascaded.LLMService, error) {
			return buildNativeCascadedLLM(ctx, VoiceEnv(env), sessionID, lg)
		},
		BuildTTS: func(env tenantcfg.VoiceEnv, bridgeRate int, tap func([]byte), callID string) (cascaded.TTSService, error) {
			return buildNativeCascadedTTS(VoiceEnv(env), bridgeRate, tap, callID, lg)
		},
		BuildHotword: func(env tenantcfg.VoiceEnv) cascaded.TextRewriter {
			bundle := providers.ParseHotwordBundle(env.HotWordsRaw, lg, env.TTSConfigRaw)
			if bundle.Corrector != nil {
				return bundle.Corrector
			}
			return newHotwordCorrector(lg)
		},
		BuildVAD: func(env tenantcfg.VoiceEnv) cascaded.BargeInDetector {
			return buildNativeVAD(VoiceEnv(env), lg)
		},
		BuildTurnPersister: func(env tenantcfg.VoiceEnv, sessionID string) cascaded.TurnPersister {
			return buildNativeTurnPersister(VoiceEnv(env), sessionID, lg)
		},
		BuildRealtimeAgent: func(ctx context.Context, env tenantcfg.VoiceEnv, callID string, bridgeRate int) (dialogrealtime.AgentBuilder, dialogrealtime.PacerConfig, error) {
			return buildNativeRealtimeAgent(ctx, env, callID, bridgeRate, lg)
		},
		PrepareVoiceRuntime: func(env tenantcfg.VoiceEnv, pcmBridgeSR int) *voiceruntime.Runtime {
			return voiceruntime.Prepare(context.Background(), env, pcmBridgeSR, lg)
		},
	}
}
