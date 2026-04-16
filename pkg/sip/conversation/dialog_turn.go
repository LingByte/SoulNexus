package conversation

import "time"

// DialogTurn is persisted to sip_calls.turns (JSON) via SetSIPTurnPersist → sipserver.Store.SaveConversationTurn.
type DialogTurn struct {
	ASRText     string
	LLMText     string
	ASRProvider string
	TTSProvider string
	LLMModel    string
	// Trigger is how ASR fired this turn: final | partial | partial-timeout (script / partial ASR).
	Trigger string
	// ScriptStepID / RouteIntent are optional (script routing, future hooks).
	ScriptStepID string
	RouteIntent  string
	LLMFirstMs   int // first LLM token latency (ms) when streaming; for non-stream ≈ LLMWallMs
	LLMWallMs    int // LLM Query / QueryStream wall time (ms); streaming includes time while TTS flush runs inside callbacks
	TTSMs        int // cumulative ttsPipe.Speak wall time (ms)
	PipelineMs   int // wall time for the whole ASR→LLM→TTS goroutine slice (caller-filled)
	At           time.Time // zero → persistence uses time.Now()
}

// StreamTurnTimings is measured inside streamLLMToTTS (LLM + TTS only, not ASR).
type StreamTurnTimings struct {
	LLMFirstMs int
	LLMWallMs  int
	TTSMs      int
}
