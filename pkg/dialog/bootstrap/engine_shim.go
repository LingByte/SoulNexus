package bootstrap

import (
	"github.com/LingByte/SoulNexus/pkg/dialog/cascaded"
	"github.com/LingByte/SoulNexus/pkg/dialog/providers"
	"github.com/LingByte/SoulNexus/pkg/dialog/voiceattach"
	"go.uber.org/zap"
)

func init() {
	voiceattach.SetHooks(voiceattach.Hooks{
		RecordTurn: RecordDialogTurn,
		NewHotwordCorrector: func(lg *zap.Logger) cascaded.TextRewriter {
			c := providers.NewHotwordCorrector(lg)
			if c == nil {
				return nil
			}
			return c
		},
	})
	voiceattach.SetNativeProviderHooks(voiceattach.NativeProviderHooks{
		BuildLLM:           buildNativeCascadedLLM,
		BuildTurnPersister: buildNativeTurnPersister,
	})
}
