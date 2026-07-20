package intentonnx

import "encoding/json"

// AnswerChannel selects exactly one user-visible response path for a single user turn.
type AnswerChannel int

const (
	// AnswerChannelIntent: use RouteOutput.Reply only; do not invoke an LLM for this turn.
	AnswerChannelIntent AnswerChannel = iota
	// AnswerChannelLLM: RouteOutput.Reply is empty; use your LLM once for this turn.
	AnswerChannelLLM
)

// MarshalJSON encodes the channel as "intent" or "llm" for logs and HTTP payloads.
func (a AnswerChannel) MarshalJSON() ([]byte, error) {
	switch a {
	case AnswerChannelIntent:
		return json.Marshal("intent")
	case AnswerChannelLLM:
		return json.Marshal("llm")
	default:
		return json.Marshal("unknown")
	}
}

// RouteOutput is the result of [Engine.Route]. Obey [AnswerChannel] so intent and LLM never both answer.
type RouteOutput struct {
	Channel AnswerChannel `json:"channel"`
	// Reply is the full user-facing text when Channel == [AnswerChannelIntent].
	// It is always empty when Channel == [AnswerChannelLLM].
	Reply string `json:"reply,omitempty"`

	Prediction Prediction `json:"prediction"`
}

// RouteOptions tweaks routing; zero value is safe defaults (uncertain → canned default_reply, not LLM).
type RouteOptions struct {
	DisableKeywordBias bool
	// LabelOverrides optional display names per class index (CSV split in demo); same length as classes ideally.
	LabelOverrides []string
	// UncertainMeansLLM: when true, low-confidence / ambiguous-margin cases that would use DefaultReply
	// instead yield AnswerChannelLLM with empty Reply (single channel = LLM only).
	UncertainMeansLLM bool
	// VoiceASRHints: when true, apply voice-oriented keyword rules (e.g. 人工+查 → soft-intent disambiguation).
	VoiceASRHints bool
}

// Prediction holds model + post-processing diagnostics (safe to log when Channel is LLM).
type Prediction struct {
	Text               string    `json:"text"`
	IntentIndex        int       `json:"intent_index"`
	IntentName         string    `json:"intent_name"`
	Confidence         float64   `json:"confidence"`
	Logits             []float32 `json:"logits,omitempty"`
	AdjustedLogits     []float32 `json:"adjusted_logits,omitempty"`
	KeywordBiasApplied bool      `json:"keyword_bias_applied"`
	Softmax            []float32 `json:"softmax,omitempty"`
	UsedConfigFallback bool      `json:"used_config_fallback"`
}
