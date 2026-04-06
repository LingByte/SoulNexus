package outbound

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	ScriptStepSay       = "say"
	ScriptStepListen    = "listen"
	ScriptStepCondition = "condition"
	ScriptStepEnd       = "end"
)

// HybridScript is a deterministic skeleton where each step has bounded retries/timeouts.
type HybridScript struct {
	ID      string       `json:"id"`
	Version string       `json:"version"`
	StartID string       `json:"start_id"`
	Steps   []HybridStep `json:"steps"`
}

type HybridStep struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	Prompt      string            `json:"prompt"`
	NextID      string            `json:"next_id"`
	Retry       int               `json:"retry"`
	TimeoutMS   int               `json:"timeout_ms"`
	Conditions  map[string]string `json:"conditions"`
	FallbackID  string            `json:"fallback_id"`
}

type ScriptRunRecorder interface {
	Record(ctx context.Context, event ScriptRunEvent) error
}

type ScriptRunEvent struct {
	CallID        string
	CorrelationID string
	ScriptID      string
	ScriptVersion string
	StepID        string
	StepType      string
	Result        string
	InputText     string
	OutputText    string
}

// HybridScriptRunner is an MVP runner: it enforces step ordering and logs script traces.
// Voice input/output integration is delegated to ASR/TTS pipeline through system prompt.
type HybridScriptRunner struct {
	Script   HybridScript
	Recorder ScriptRunRecorder
}

func NewHybridScriptRunner(script HybridScript, recorder ScriptRunRecorder) *HybridScriptRunner {
	return &HybridScriptRunner{Script: script, Recorder: recorder}
}

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
	return s, nil
}

func (r *HybridScriptRunner) Run(ctx context.Context, leg EstablishedLeg) error {
	if r == nil {
		return nil
	}
	steps := make(map[string]HybridStep, len(r.Script.Steps))
	for _, s := range r.Script.Steps {
		steps[s.ID] = s
	}
	current := r.Script.StartID
	maxSteps := len(r.Script.Steps) + 4
	for i := 0; i < maxSteps; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		step, ok := steps[current]
		if !ok {
			return fmt.Errorf("hybrid script step not found: %s", current)
		}
		if err := r.record(ctx, ScriptRunEvent{
			CallID:        leg.CallID,
			CorrelationID: leg.CorrelationID,
			ScriptID:      r.Script.ID,
			ScriptVersion: r.Script.Version,
			StepID:        step.ID,
			StepType:      step.Type,
			Result:        "ok",
			OutputText:    step.Prompt,
		}); err != nil {
			return err
		}
		if step.Type == ScriptStepEnd || step.NextID == "" {
			return nil
		}
		if step.TimeoutMS > 0 {
			timer := time.NewTimer(time.Duration(step.TimeoutMS) * time.Millisecond)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
		current = step.NextID
	}
	return fmt.Errorf("hybrid script reached max steps")
}

func (r *HybridScriptRunner) record(ctx context.Context, event ScriptRunEvent) error {
	if r == nil || r.Recorder == nil {
		return nil
	}
	return r.Recorder.Record(ctx, event)
}

