package outbound

// This file only handles decoding and validating hybrid script JSON into data structures.
// Step execution lives in hybrid_script_run.go.

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/constants"
)

// HybridScript is a deterministic skeleton where each step has bounded retries/timeouts.
type HybridScript struct {
	ID               string       `json:"id"`
	Version          string       `json:"version"`
	StartID          string       `json:"start_id"`
	MaxTurns         int          `json:"max_turns"`
	SilenceTimeoutMS int          `json:"silence_timeout_ms"`
	EndIntents       []string     `json:"end_intents"`
	Steps            []HybridStep `json:"steps"`
}

type HybridStep struct {
	ID               string             `json:"id"`
	Type             string             `json:"type"`
	Prompt           string             `json:"prompt"`
	NextID           string             `json:"next_id"`
	Retry            int                `json:"retry"`
	TimeoutMS        int                `json:"timeout_ms"`
	Conditions       map[string]string  `json:"conditions"`
	FallbackID       string             `json:"fallback_id"`
	ListenTimeoutMS  int                `json:"listen_timeout_ms"`
	ListenFallbackID string             `json:"listen_fallback_id"`
	LLMInstruction   string             `json:"llm_instruction"`
	Transitions      []HybridTransition `json:"transitions"`
	// DTMFTransitions: listen steps only — map keypad (0-9 * #) to next_id without ASR/LLM.
	DTMFTransitions []HybridDTMFTransition `json:"dtmf_transitions"`
}

// HybridDTMFTransition binds one DTMF key to the next script step (IVR-style).
type HybridDTMFTransition struct {
	Digit  string `json:"digit"`   // single: 0-9, *, #
	NextID string `json:"next_id"` // required when digit matches
}

type HybridTransition struct {
	// Description is natural-language guidance for LLM-only listen routing (preferred over contains/equals).
	Description string `json:"description"`
	Intent      string `json:"intent"`
	Contains    string `json:"contains"`
	Equals      string `json:"equals"`
	NextID      string `json:"next_id"`
}

// ParseHybridScript unmarshals raw JSON and validates required fields and step types.
func ParseHybridScript(raw string) (HybridScript, error) {
	var s HybridScript
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &s); err != nil {
		return HybridScript{}, err
	}
	if s.ID == "" {
		return HybridScript{}, fmt.Errorf("hybrid script id is required")
	}
	if s.StartID == "" {
		return HybridScript{}, fmt.Errorf("hybrid script start_id is required")
	}
	seen := map[string]struct{}{}
	allowedTypes := map[string]struct{}{
		constants.SIPScriptStepSay:       {},
		constants.SIPScriptStepListen:    {},
		constants.SIPScriptStepLLMReply:  {},
		constants.SIPScriptStepCondition: {},
		constants.SIPScriptStepEnd:       {},
	}
	for _, st := range s.Steps {
		id := strings.TrimSpace(st.ID)
		if id == "" {
			return HybridScript{}, fmt.Errorf("hybrid script step id is required")
		}
		stepType := strings.TrimSpace(st.Type)
		if stepType == "" {
			return HybridScript{}, fmt.Errorf("hybrid script step type is required: %s", id)
		}
		if _, ok := allowedTypes[stepType]; !ok {
			return HybridScript{}, fmt.Errorf("hybrid script unsupported step type %q: %s", stepType, id)
		}
		if _, ok := seen[id]; ok {
			return HybridScript{}, fmt.Errorf("hybrid script duplicate step id: %s", id)
		}
		seen[id] = struct{}{}
	}
	for _, st := range s.Steps {
		if strings.TrimSpace(st.Type) != constants.SIPScriptStepListen {
			if len(st.DTMFTransitions) > 0 {
				return HybridScript{}, fmt.Errorf("hybrid script step %s: dtmf_transitions only allowed on listen steps", st.ID)
			}
			continue
		}
		for ti, tr := range st.Transitions {
			if strings.TrimSpace(tr.NextID) == "" {
				return HybridScript{}, fmt.Errorf("hybrid script listen step %s transition[%d]: next_id is required", st.ID, ti)
			}
		}
		seenD := map[string]struct{}{}
		for ti, dt := range st.DTMFTransitions {
			d := normalizeDTMFKey(dt.Digit)
			if d == "" {
				return HybridScript{}, fmt.Errorf("hybrid script listen step %s dtmf_transitions[%d]: digit must be 0-9, *, or #", st.ID, ti)
			}
			if _, dup := seenD[d]; dup {
				return HybridScript{}, fmt.Errorf("hybrid script listen step %s: duplicate dtmf digit %q", st.ID, d)
			}
			seenD[d] = struct{}{}
			if strings.TrimSpace(dt.NextID) == "" {
				return HybridScript{}, fmt.Errorf("hybrid script listen step %s dtmf_transitions[%d]: next_id is required", st.ID, ti)
			}
		}
	}
	return s, nil
}

func normalizeDTMFKey(d string) string {
	d = strings.TrimSpace(d)
	if len(d) != 1 {
		return ""
	}
	switch d[0] {
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '*', '#':
		return string(d[0])
	default:
		return ""
	}
}

// DTMFDigitToNextMap builds digit → next_id for script listen (scriptlisten.TryConsumeDTMF).
func DTMFDigitToNextMap(step HybridStep) map[string]string {
	m := make(map[string]string, len(step.DTMFTransitions))
	for _, dt := range step.DTMFTransitions {
		d := normalizeDTMFKey(dt.Digit)
		if d == "" {
			continue
		}
		if nx := strings.TrimSpace(dt.NextID); nx != "" {
			m[d] = nx
		}
	}
	return m
}
