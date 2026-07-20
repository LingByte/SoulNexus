package realtime

import (
	"errors"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
	"github.com/LingByte/lingllm/realtime"
	"go.uber.org/zap"
)

const proactiveGreetingSilenceMs = 100

// AgentInputRate is the caller PCM rate multimodal agents expect.
const AgentInputRate = tenantcfg.RealtimeInputSampleRate

// TriggerProactiveGreeting asks Omni to speak first after attach or welcome.
// When serverVAD is true, do not push silence/commit/response.create after a
// welcome clip — server VAD will reply once the caller speaks. When no welcome
// was played and assistantWelcome is set, one response.create is allowed so the
// model can open without waiting for user audio.
func TriggerProactiveGreeting(
	agent realtime.Agent,
	assistantWelcome string,
	lg *zap.Logger,
	callID string,
	serverVAD bool,
	afterWelcomeClip bool,
) {
	if agent == nil {
		return
	}
	hint := strings.TrimSpace(assistantWelcome)

	if serverVAD {
		if afterWelcomeClip {
			if lg != nil {
				lg.Info("realtime: proactive greeting skipped — welcome already played, awaiting caller (server VAD)",
					zap.String("call_id", callID))
			}
			return
		}
		if hint == "" {
			if lg != nil {
				lg.Debug("realtime: proactive greeting deferred to server VAD",
					zap.String("call_id", callID))
			}
			return
		}
		starter, ok := agent.(ResponseStarter)
		if !ok {
			if lg != nil {
				lg.Debug("realtime: proactive greeting deferred to server VAD",
					zap.String("call_id", callID))
			}
			return
		}
		if err := starter.CreateResponse(hint); err != nil {
			logProactiveFailure(lg, callID, "response.create", err)
		} else if lg != nil {
			lg.Info("realtime: proactive greeting requested (no welcome clip)",
				zap.String("call_id", callID),
				zap.Int("hint_chars", len([]rune(hint))))
		}
		return
	}

	silenceBytes := AgentInputRate * 2 * proactiveGreetingSilenceMs / 1000
	if silenceBytes < 2 {
		silenceBytes = 2
	}
	silence := make([]byte, silenceBytes)
	if err := agent.PushAudio(silence); err != nil {
		logProactiveFailure(lg, callID, "push silence", err)
		return
	}
	if err := agent.CommitInputAudio(); err != nil {
		logProactiveFailure(lg, callID, "commit silence", err)
		return
	}
	starter, ok := agent.(ResponseStarter)
	if !ok {
		if lg != nil {
			lg.Warn("realtime: proactive greeting skipped — no response.create",
				zap.String("call_id", callID))
		}
		return
	}
	if err := starter.CreateResponse(hint); err != nil {
		logProactiveFailure(lg, callID, "response.create", err)
		return
	}
	if lg != nil {
		lg.Info("realtime: proactive greeting requested",
			zap.String("call_id", callID),
			zap.Int("hint_chars", len([]rune(hint))))
	}
}

func logProactiveFailure(lg *zap.Logger, callID, step string, err error) {
	if lg == nil {
		return
	}
	if errors.Is(err, realtime.ErrAgentClosed) {
		lg.Warn("realtime: proactive greeting skipped — session closed",
			zap.String("call_id", callID), zap.String("step", step), zap.Error(err))
		return
	}
	lg.Warn("realtime: proactive greeting failed",
		zap.String("call_id", callID), zap.String("step", step), zap.Error(err))
}
