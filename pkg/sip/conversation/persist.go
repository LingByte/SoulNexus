package conversation

import (
	"context"
	"sync"

	"github.com/LingByte/SoulNexus/pkg/logger"
)

var sipTurnPersist func(ctx context.Context, callID string, turn DialogTurn)

var warnTurnPersistOnce sync.Once

// SetSIPTurnPersist registers a callback to append one dialog turn onto sip_calls.turns.
// cmd/sip wires this to sipserver.Store.SaveConversationTurn.
//
// Call recording (inbound UAS and outbound UAC with CallSession registered):
//   - pkg/sip/session.CallSession: OnInputRTP (user) + OnOutputRTP (AI) → SN2 blob.
//   - BYE → TakeRecording → sipserver.OnBye → stereo WAV preferred (L=user R=AI per-leg decode, no mono ducking),
//     falling back to legacy mono mix if stereo build fails; upload → recording_url.
func SetSIPTurnPersist(fn func(ctx context.Context, callID string, turn DialogTurn)) {
	sipTurnPersist = fn
}

func persistSIPTurn(ctx context.Context, callID string, turn DialogTurn) {
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
	sipTurnPersist(ctx, callID, turn)
}
