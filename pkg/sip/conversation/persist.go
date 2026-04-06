package conversation

import (
	"context"
	"sync"

	"github.com/LingByte/SoulNexus/pkg/logger"
)

var sipTurnPersist func(ctx context.Context, callID, userText, assistantText, asrProvider, llmModel, ttsProvider string)

var warnTurnPersistOnce sync.Once

// SetSIPTurnPersist registers a callback after a successful ASR→LLM reply (before TTS); cmd/sip wires it to sippersist.Store.SaveConversationTurn (appends one turn in sip_calls.turns JSON).
// Related lifecycle: INVITE/ACK → optional AI turns → BYE → sippersist.OnBye uploads PCMU/PCMA or Opus→WAV via config.GlobalStore and updates sip_calls.
func SetSIPTurnPersist(fn func(ctx context.Context, callID, userText, assistantText, asrProvider, llmModel, ttsProvider string)) {
	sipTurnPersist = fn
}

func persistSIPTurn(ctx context.Context, callID, userText, assistantText, asrProvider, llmModel, ttsProvider string) {
	if callID == "" {
		return
	}
	if sipTurnPersist == nil {
		warnTurnPersistOnce.Do(func() {
			if logger.Lg != nil {
				logger.Lg.Warn("sip: AI dialog turns are not persisted (SetSIPTurnPersist not wired — usually missing DSN in sip process)")
			}
		})
		return
	}
	sipTurnPersist(ctx, callID, userText, assistantText, asrProvider, llmModel, ttsProvider)
}
