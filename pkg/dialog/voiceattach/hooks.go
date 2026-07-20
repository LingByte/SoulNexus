package voiceattach

import (
	"context"

	"github.com/LingByte/SoulNexus/pkg/dialog/cascaded"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
	"github.com/LingByte/SoulNexus/pkg/dialog/turn"
	"go.uber.org/zap"
)

// VoiceEnv is tenant voice configuration loaded for attach.
type VoiceEnv = tenantcfg.VoiceEnv

// Hooks wires conversation-owned voice attach paths into the dialog engine seam.
type Hooks struct {
	RecordTurn          func(context.Context, string, turn.Turn)
	NewHotwordCorrector func(*zap.Logger) cascaded.TextRewriter
}

var hooks Hooks

// SetHooks installs conversation callbacks (call once from bootstrap).
func SetHooks(h Hooks) { hooks = h }

func recordDialogTurn(ctx context.Context, callID string, t turn.Turn) {
	if hooks.RecordTurn != nil {
		hooks.RecordTurn(ctx, callID, t)
	}
}

func newHotwordCorrector(lg *zap.Logger) cascaded.TextRewriter {
	if hooks.NewHotwordCorrector == nil {
		return nil
	}
	return hooks.NewHotwordCorrector(lg)
}
