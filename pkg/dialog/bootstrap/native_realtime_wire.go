package bootstrap

import (
	"context"
	"strings"
	"time"

	dialogaudio "github.com/LingByte/SoulNexus/pkg/dialog/audio"
	"github.com/LingByte/SoulNexus/pkg/dialog/callbinding"
	"github.com/LingByte/SoulNexus/pkg/dialog/providers"
	stageknow "github.com/LingByte/SoulNexus/pkg/dialog/stages/knowledge"
	stagenlu "github.com/LingByte/SoulNexus/pkg/dialog/stages/nlu"
	stagespeaker "github.com/LingByte/SoulNexus/pkg/dialog/stages/speaker"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
	"github.com/LingByte/SoulNexus/pkg/dialog/voiceattach"
	"go.uber.org/zap"
)

func init() {
	voiceattach.SetNativeRealtimeHooks(voiceattach.NativeRealtimeHooks{
		BuildSystemPrompt: buildNativeRealtimeSystemPrompt,
		PrepareCall:       prepareNativeRealtimeCall,
	})
}

func buildNativeRealtimeSystemPrompt(_ context.Context, env voiceattach.VoiceEnv, callID string) string {
	stagenlu.PrepareCallNLUBinding(callID, tenantcfg.VoiceEnv(env), nil)
	kbBinding := stageknow.ResolveBinding(callID)
	operatorCore := mergeRealtimeInstructions(
		tenantcfg.EffectiveRealtimeOperatorCore(tenantcfg.VoiceEnv(env)),
		providers.PopCallSystemPrompt(callID),
	)
	rulesBlock := providers.AugmentToolRules("", kbBinding.Enabled)
	base := mergeRealtimeInstructions(operatorCore, rulesBlock)
	base = mergeRealtimeInstructions(base, stagenlu.IntentCatalogHint(callID))
	base = mergeRealtimeInstructions(base, stagespeaker.AssistantBoundPromptHint(env.TenantID, env.AssistantID))
	base = mergeRealtimeInstructions(base, providers.WallClockPromptHint(time.Now()))
	base = callbinding.EnrichSystemPrompt(base, callID)
	return dialogaudio.ApplyNoiseHint(base, dialogaudio.GlobalCallNoise.HintForCall(callID))
}

func prepareNativeRealtimeCall(_ context.Context, callID string, env voiceattach.VoiceEnv, lg *zap.Logger) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	stageknow.PrepareCallKnowledgeBinding(callID, tenantcfg.VoiceEnv(env), env.TenantID, stageknow.SearchConfigFromVoiceEnv(tenantcfg.VoiceEnv(env)), lg)
	stagenlu.PrepareCallNLUBinding(callID, tenantcfg.VoiceEnv(env), lg)
	_ = stagespeaker.PrepareCallSpeakerBinding(callID, env.TenantID, env.AssistantID, lg)
	if lg != nil {
		lg.Debug("native realtime: call preparation complete", zap.String("call_id", callID))
	}
}

func mergeRealtimeInstructions(parts ...string) string {
	var b strings.Builder
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(p)
	}
	return b.String()
}
