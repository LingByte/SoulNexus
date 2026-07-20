package turn

import "time"

// Turn is one ASR→assistant dialog turn for audit / debug metrics.
type Turn struct {
	ASRText     string
	LLMText     string
	ASRProvider string
	TTSProvider string
	LLMModel    string
	// Trigger is how ASR fired this turn (currently always "final"; legacy values may appear in historical rows).
	Trigger string
	// ScriptStepID / RouteIntent are optional (script routing, future hooks).
	ScriptStepID string
	RouteIntent  string
	LLMFirstMs   int       // first LLM token latency (ms) when streaming; for non-stream ≈ LLMWallMs
	LLMWallMs    int       // LLM Query / QueryStream wall time (ms)
	TTSMs        int       // cumulative ttsPipe.Speak wall time (ms)
	PipelineMs   int       // wall time for the whole ASR→LLM→TTS goroutine slice
	At           time.Time // zero → persistence uses time.Now()
	// TurnGroupID groups streaming TTS segments into one logical turn row.
	TurnGroupID string
	// KnowledgeRetrievals is populated when search_knowledge_base ran during this turn.
	KnowledgeRetrievals []KnowledgeRetrievalRecord
}

// StreamTimings is measured inside streamLLMToTTS (LLM + TTS only, not ASR).
type StreamTimings struct {
	LLMFirstMs int
	LLMWallMs  int
	TTSMs      int
}
