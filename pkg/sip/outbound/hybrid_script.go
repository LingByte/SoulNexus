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
	ScriptStepLLMReply  = "llm_reply"
	ScriptStepCondition = "condition"
	ScriptStepEnd       = "end"
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
	ID              string            `json:"id"`
	Type            string            `json:"type"`
	Prompt          string            `json:"prompt"`
	NextID          string            `json:"next_id"`
	Retry           int               `json:"retry"`
	TimeoutMS       int               `json:"timeout_ms"`
	Conditions      map[string]string `json:"conditions"`
	FallbackID      string            `json:"fallback_id"`
	ListenTimeoutMS int               `json:"listen_timeout_ms"`
	ListenFallbackID string           `json:"listen_fallback_id"`
	LLMInstruction  string            `json:"llm_instruction"`
	Transitions     []HybridTransition `json:"transitions"`
}

type HybridTransition struct {
	Intent   string `json:"intent"`
	Contains string `json:"contains"`
	Equals   string `json:"equals"`
	NextID   string `json:"next_id"`
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

type ListenResult struct {
	InputText string
	ReplyText string
}

type RuntimeHooks struct {
	OnSay        func(ctx context.Context, leg EstablishedLeg, prompt string) error
	OnListen     func(ctx context.Context, leg EstablishedLeg, timeout time.Duration) (ListenResult, error)
	OnLLMReply   func(ctx context.Context, leg EstablishedLeg, userText, instruction string) (string, error)
	IsEndIntent  func(input string, script HybridScript) bool
}

// HybridScriptRunner is an MVP runner: it enforces step ordering and logs script traces.
// Voice input/output integration is delegated to ASR/TTS pipeline through system prompt.
type HybridScriptRunner struct {
	Script   HybridScript
	Recorder ScriptRunRecorder
	Hooks    RuntimeHooks
}

func NewHybridScriptRunner(script HybridScript, recorder ScriptRunRecorder) *HybridScriptRunner {
	return &HybridScriptRunner{Script: script, Recorder: recorder}
}

func (r *HybridScriptRunner) WithHooks(h RuntimeHooks) *HybridScriptRunner {
	if r == nil {
		return nil
	}
	r.Hooks = h
	return r
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
	seen := map[string]struct{}{}
	for _, st := range s.Steps {
		id := strings.TrimSpace(st.ID)
		if id == "" {
			return HybridScript{}, fmt.Errorf("hybrid script step id is required")
		}
		if _, ok := seen[id]; ok {
			return HybridScript{}, fmt.Errorf("hybrid script duplicate step id: %s", id)
		}
		seen[id] = struct{}{}
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
	maxSteps := len(r.Script.Steps) + 8
	if r.Script.MaxTurns > 0 {
		maxSteps = r.Script.MaxTurns
	}
	lastInput := ""
	lastReply := ""
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
			Result:        "started",
			InputText:     lastInput,
			OutputText:    step.Prompt,
		}); err != nil {
			return err
		}
		nextID := strings.TrimSpace(step.NextID)
		switch strings.TrimSpace(step.Type) {
		case ScriptStepSay:
			if p := strings.TrimSpace(step.Prompt); p != "" && r.Hooks.OnSay != nil {
				if err := r.Hooks.OnSay(ctx, leg, p); err != nil {
					_ = r.record(ctx, ScriptRunEvent{
						CallID:        leg.CallID,
						CorrelationID: leg.CorrelationID,
						ScriptID:      r.Script.ID,
						ScriptVersion: r.Script.Version,
						StepID:        step.ID,
						StepType:      step.Type,
						Result:        "failed",
						InputText:     lastInput,
						OutputText:    err.Error(),
					})
					return err
				}
			}
		case ScriptStepListen:
			timeoutMS := step.ListenTimeoutMS
			if timeoutMS <= 0 {
				timeoutMS = r.Script.SilenceTimeoutMS
			}
			if timeoutMS <= 0 {
				timeoutMS = 8000
			}
			if r.Hooks.OnListen != nil {
				res, err := r.Hooks.OnListen(ctx, leg, time.Duration(timeoutMS)*time.Millisecond)
				if err != nil {
					fallback := strings.TrimSpace(step.ListenFallbackID)
					if fallback == "" {
						fallback = strings.TrimSpace(step.FallbackID)
					}
					if fallback != "" {
						nextID = fallback
						_ = r.record(ctx, ScriptRunEvent{
							CallID:        leg.CallID,
							CorrelationID: leg.CorrelationID,
							ScriptID:      r.Script.ID,
							ScriptVersion: r.Script.Version,
							StepID:        step.ID,
							StepType:      step.Type,
							Result:        "timeout",
							InputText:     lastInput,
							OutputText:    err.Error(),
						})
					} else {
						return err
					}
				} else {
					lastInput = strings.TrimSpace(res.InputText)
					lastReply = strings.TrimSpace(res.ReplyText)
				}
			}
			if r.hitEndIntent(lastInput) {
				nextID = ""
				_ = r.record(ctx, ScriptRunEvent{
					CallID:        leg.CallID,
					CorrelationID: leg.CorrelationID,
					ScriptID:      r.Script.ID,
					ScriptVersion: r.Script.Version,
					StepID:        step.ID,
					StepType:      step.Type,
					Result:        "ended",
					InputText:     lastInput,
					OutputText:    "end intent matched",
				})
			} else if tNext := resolveTransition(step.Transitions, lastInput); tNext != "" {
				nextID = tNext
			}
		case ScriptStepLLMReply:
			reply := strings.TrimSpace(lastReply)
			if reply == "" && r.Hooks.OnLLMReply != nil {
				rp, err := r.Hooks.OnLLMReply(ctx, leg, lastInput, step.LLMInstruction)
				if err == nil {
					reply = strings.TrimSpace(rp)
				}
			}
			if reply != "" && r.Hooks.OnSay != nil {
				if err := r.Hooks.OnSay(ctx, leg, reply); err != nil {
					_ = r.record(ctx, ScriptRunEvent{
						CallID:        leg.CallID,
						CorrelationID: leg.CorrelationID,
						ScriptID:      r.Script.ID,
						ScriptVersion: r.Script.Version,
						StepID:        step.ID,
						StepType:      step.Type,
						Result:        "failed",
						InputText:     lastInput,
						OutputText:    err.Error(),
					})
					return err
				}
			}
		case ScriptStepCondition:
			if tNext := resolveTransition(step.Transitions, lastInput); tNext != "" {
				nextID = tNext
			} else if fb := strings.TrimSpace(step.FallbackID); fb != "" {
				nextID = fb
			}
		case ScriptStepEnd:
			return nil
		}
		if nextID == "" {
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
		current = nextID
	}
	return fmt.Errorf("hybrid script reached max steps")
}

func (r *HybridScriptRunner) record(ctx context.Context, event ScriptRunEvent) error {
	if r == nil || r.Recorder == nil {
		return nil
	}
	return r.Recorder.Record(ctx, event)
}

func resolveTransition(transitions []HybridTransition, input string) string {
	if len(transitions) == 0 {
		return ""
	}
	in := strings.ToLower(strings.TrimSpace(input))
	for _, tr := range transitions {
		next := strings.TrimSpace(tr.NextID)
		if next == "" {
			continue
		}
		if eq := strings.ToLower(strings.TrimSpace(tr.Equals)); eq != "" && in == eq {
			return next
		}
		if c := strings.ToLower(strings.TrimSpace(tr.Contains)); c != "" && strings.Contains(in, c) {
			return next
		}
		if it := strings.ToLower(strings.TrimSpace(tr.Intent)); it != "" && strings.Contains(in, it) {
			return next
		}
	}
	return ""
}

func (r *HybridScriptRunner) hitEndIntent(input string) bool {
	if r == nil {
		return false
	}
	if r.Hooks.IsEndIntent != nil {
		return r.Hooks.IsEndIntent(input, r.Script)
	}
	in := strings.ToLower(strings.TrimSpace(input))
	if in == "" {
		return false
	}
	for _, it := range r.Script.EndIntents {
		if v := strings.ToLower(strings.TrimSpace(it)); v != "" && strings.Contains(in, v) {
			return true
		}
	}
	return false
}

