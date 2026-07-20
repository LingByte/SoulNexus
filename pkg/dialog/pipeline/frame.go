package pipeline

import (
	"time"

	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
	"github.com/LingByte/SoulNexus/pkg/dialog/provider/llm"
)

// Kind tags the payload type of a Frame. Stages branch on Kind to
// decide which fields are populated.
type Kind uint8

const (
	// KindPCM carries one frame of audio (input from user OR output
	// to user — direction is determined by which channel the frame
	// is travelling on, not by Kind).
	KindPCM Kind = iota + 1

	// KindTextInterim is a partial ASR transcript (best guess so far).
	// Stages that drive LLM/UI should debounce on these.
	KindTextInterim

	// KindTextFinal is the committed ASR transcript for one user
	// utterance. Triggers LLM dispatch in cascaded engines.
	KindTextFinal

	// KindAIText is an LLM-generated text fragment streaming toward
	// TTS. Multiple KindAIText frames per turn (one per delta).
	KindAIText

	// KindAITextDone closes the current AI-text turn. TTS Stage
	// flushes residual audio on this signal.
	KindAITextDone

	// KindToolCall asks the engine to invoke a tool (transfer /
	// hangup / lookup). Frame.ToolCall is populated.
	KindToolCall

	// KindBargeIn signals user-side voice activity detected during
	// AI output. Stages above TTS gate output until barge-in clears.
	KindBargeIn

	// KindUserHangup is raised by the transport-side adapter when
	// the user terminates the call. Stages perform graceful cleanup.
	KindUserHangup
)

// Frame is the single union type flowing along a Pipeline. Only the
// fields documented for each Kind are valid; others are zero.
type Frame struct {
	// Kind tags the payload type. See the Kind constants.
	Kind Kind

	// PCM is set for KindPCM frames.
	PCM engine.PCMFrame

	// Text is set for KindTextInterim / KindTextFinal / KindAIText.
	Text string

	// ToolCall is set for KindToolCall frames. Reuses the LLM
	// provider's ToolCall shape because realtime providers also
	// emit calls in the same shape (see multimodal.Event).
	ToolCall *llm.ToolCall

	// EmittedAt is the wall-clock time the producing stage finished
	// generating this frame. Used for analytics and end-to-end
	// latency measurement.
	EmittedAt time.Time

	// TurnID is a stable identifier for the turn this frame belongs
	// to. Stages can use it to group related frames; producers MUST
	// keep it consistent across all frames of one turn.
	TurnID string
}

// IsAudio reports whether this frame carries audio samples.
func (f Frame) IsAudio() bool { return f.Kind == KindPCM }

// IsText reports whether this frame carries any kind of text.
func (f Frame) IsText() bool {
	switch f.Kind {
	case KindTextInterim, KindTextFinal, KindAIText:
		return true
	default:
		return false
	}
}

// IsControl reports whether this frame is a control signal (no data
// payload — barge-in / hangup / tool call / done markers).
func (f Frame) IsControl() bool {
	switch f.Kind {
	case KindBargeIn, KindUserHangup, KindAITextDone, KindToolCall:
		return true
	default:
		return false
	}
}
