package outbound

// This file runs a validated HybridScript: step dispatch, hooks, tracing, and listen timing helpers.
// JSON parsing and schema checks are in hybrid_script_parse.go.

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"go.uber.org/zap"
)

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
	// DurationMS is wall ms since the script runner entered this step (set before Record).
	DurationMS int64
}

type ListenResult struct {
	InputText string
	ReplyText string
	// DTMFDigit is set when the waiter resolved this listen via keypad (RFC2833 / SIP INFO), e.g. "1".
	DTMFDigit string
}

type RuntimeHooks struct {
	OnSay func(ctx context.Context, leg EstablishedLeg, prompt string) error
	// notBefore marks the moment listen step starts; consumers should ignore older ASR turns.
	// step is the current listen HybridStep (for DTMF bindings).
	OnListen func(ctx context.Context, leg EstablishedLeg, timeout time.Duration, notBefore time.Time, step HybridStep) (ListenResult, error)
	OnLLMReply  func(ctx context.Context, leg EstablishedLeg, userText, instruction string) (string, error)
	IsEndIntent func(input string, script HybridScript) bool
}

// HybridScriptRunner enforces step ordering and logs script traces; I/O is delegated via RuntimeHooks.
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
	lastSayPrompt := ""
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
		stepStart := time.Now()
		startIn, startOut := lastInput, step.Prompt
		if strings.TrimSpace(step.Type) == constants.SIPScriptStepListen {
			startIn, startOut = "", "-"
		}
		if err := r.record(ctx, stepStart, ScriptRunEvent{
			CallID:        leg.CallID,
			CorrelationID: leg.CorrelationID,
			ScriptID:      r.Script.ID,
			ScriptVersion: r.Script.Version,
			StepID:        step.ID,
			StepType:      step.Type,
			Result:        constants.SIPScriptRunStarted,
			InputText:     startIn,
			OutputText:    startOut,
		}); err != nil {
			return err
		}
		nextID := strings.TrimSpace(step.NextID)
		switch strings.TrimSpace(step.Type) {
		case constants.SIPScriptStepSay:
			if p := strings.TrimSpace(step.Prompt); p != "" && r.Hooks.OnSay != nil {
				if err := r.Hooks.OnSay(ctx, leg, p); err != nil {
					_ = r.record(ctx, stepStart, ScriptRunEvent{
						CallID:        leg.CallID,
						CorrelationID: leg.CorrelationID,
						ScriptID:      r.Script.ID,
						ScriptVersion: r.Script.Version,
						StepID:        step.ID,
						StepType:      step.Type,
						Result:        constants.SIPScriptRunFailed,
						InputText:     lastInput,
						OutputText:    err.Error(),
					})
					return err
				}
				lastSayPrompt = p
			}
		case constants.SIPScriptStepListen:
			timeoutMS := step.ListenTimeoutMS
			if timeoutMS <= 0 {
				timeoutMS = r.Script.SilenceTimeoutMS
			}
			if timeoutMS <= 0 {
				timeoutMS = 8000
			}
			if lastSayPrompt != "" {
				timeoutMS += estimatePromptTailMS(lastSayPrompt)
			}
			var dtmfUsed bool
			if r.Hooks.OnListen != nil {
				listenStartedAt := time.Now()
				res, err := r.Hooks.OnListen(ctx, leg, time.Duration(timeoutMS)*time.Millisecond, listenStartedAt, step)
				if err != nil {
					fallback := strings.TrimSpace(step.ListenFallbackID)
					if fallback == "" {
						fallback = strings.TrimSpace(step.FallbackID)
					}
					if fallback != "" {
						nextID = fallback
						_ = r.record(ctx, stepStart, ScriptRunEvent{
							CallID:        leg.CallID,
							CorrelationID: leg.CorrelationID,
							ScriptID:      r.Script.ID,
							ScriptVersion: r.Script.Version,
							StepID:        step.ID,
							StepType:      step.Type,
							Result:        constants.SIPScriptRunTimeout,
							InputText:     lastInput,
							OutputText:    err.Error(),
						})
					} else {
						return err
					}
				} else {
					if d := strings.TrimSpace(res.DTMFDigit); d != "" {
						dtmfUsed = true
						lastInput = "dtmf:" + d
						if nid := matchDTMFNextID(step, d); nid != "" {
							nextID = nid
						}
						_ = r.record(ctx, stepStart, ScriptRunEvent{
							CallID:        leg.CallID,
							CorrelationID: leg.CorrelationID,
							ScriptID:      r.Script.ID,
							ScriptVersion: r.Script.Version,
							StepID:        step.ID,
							StepType:      step.Type,
							Result:        constants.SIPScriptRunMatched,
							InputText:     lastInput,
							OutputText:    "dtmf -> " + nextID,
						})
					} else {
						lastInput = strings.TrimSpace(res.InputText)
						lastReply = strings.TrimSpace(res.ReplyText)
						out := lastReply
						if out == "" {
							out = "-"
						}
						_ = r.record(ctx, stepStart, ScriptRunEvent{
							CallID:        leg.CallID,
							CorrelationID: leg.CorrelationID,
							ScriptID:      r.Script.ID,
							ScriptVersion: r.Script.Version,
							StepID:        step.ID,
							StepType:      step.Type,
							Result:        constants.SIPScriptRunMatched,
							InputText:     lastInput,
							OutputText:    out,
						})
					}
				}
			}
			lastSayPrompt = ""
			if !dtmfUsed && r.hitEndIntent(lastInput) {
				nextID = ""
				_ = r.record(ctx, stepStart, ScriptRunEvent{
					CallID:        leg.CallID,
					CorrelationID: leg.CorrelationID,
					ScriptID:      r.Script.ID,
					ScriptVersion: r.Script.Version,
					StepID:        step.ID,
					StepType:      step.Type,
					Result:        constants.SIPScriptRunEnded,
					InputText:     lastInput,
					OutputText:    "end intent matched",
				})
			} else if !dtmfUsed {
				picked, branchErr := resolveListenBranches(ctx, leg, step, lastInput)
				if branchErr != nil {
					return r.finishWithRouteApology(ctx, leg, step, branchErr)
				}
				if picked != "" {
					nextID = picked
				}
			}
		case constants.SIPScriptStepLLMReply:
			reply := strings.TrimSpace(lastReply)
			if reply == "" && r.Hooks.OnLLMReply != nil {
				rp, err := r.Hooks.OnLLMReply(ctx, leg, lastInput, step.LLMInstruction)
				if err == nil {
					reply = strings.TrimSpace(rp)
				}
			}
			if reply != "" && r.Hooks.OnSay != nil {
				if err := r.Hooks.OnSay(ctx, leg, reply); err != nil {
					_ = r.record(ctx, stepStart, ScriptRunEvent{
						CallID:        leg.CallID,
						CorrelationID: leg.CorrelationID,
						ScriptID:      r.Script.ID,
						ScriptVersion: r.Script.Version,
						StepID:        step.ID,
						StepType:      step.Type,
						Result:        constants.SIPScriptRunFailed,
						InputText:     lastInput,
						OutputText:    err.Error(),
					})
					return err
				}
			}
		case constants.SIPScriptStepCondition:
			if tNext := resolveTransition(step.Transitions, lastInput); tNext != "" {
				nextID = tNext
			} else if fb := strings.TrimSpace(step.FallbackID); fb != "" {
				nextID = fb
			}
		case constants.SIPScriptStepEnd:
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

func (r *HybridScriptRunner) record(ctx context.Context, stepStart time.Time, event ScriptRunEvent) error {
	if r == nil || r.Recorder == nil {
		return nil
	}
	if !stepStart.IsZero() {
		event.DurationMS = time.Since(stepStart).Milliseconds()
	}
	return r.Recorder.Record(ctx, event)
}

// finishWithRouteApology speaks SIP_SCRIPT_LLM_FAIL_PROMPT (default apology) and ends the script run successfully.
func (r *HybridScriptRunner) finishWithRouteApology(ctx context.Context, leg EstablishedLeg, step HybridStep, routeErr error) error {
	if r == nil {
		return nil
	}
	routeApologyStart := time.Now()
	if logger.Lg != nil {
		logger.Lg.Warn("hybrid script listen route aborted",
			zap.String("call_id", leg.CallID),
			zap.String("step_id", step.ID),
			zap.Error(routeErr))
	}
	msg := strings.TrimSpace(utils.GetEnv(constants.EnvSIPScriptLLMFailPrompt))
	if msg == "" {
		msg = "对不起，打扰了。"
	}
	_ = r.record(ctx, routeApologyStart, ScriptRunEvent{
		CallID:        leg.CallID,
		CorrelationID: leg.CorrelationID,
		ScriptID:      r.Script.ID,
		ScriptVersion: r.Script.Version,
		StepID:        step.ID,
		StepType:      step.Type,
		Result:        constants.SIPScriptRunRouteFailed,
		InputText:     routeErr.Error(),
		OutputText:    msg,
	})
	if r.Hooks.OnSay != nil {
		if err := r.Hooks.OnSay(ctx, leg, msg); err != nil {
			return err
		}
	}
	return nil
}

func matchDTMFNextID(step HybridStep, digit string) string {
	d := normalizeDTMFKey(digit)
	if d == "" {
		return ""
	}
	for _, dt := range step.DTMFTransitions {
		if normalizeDTMFKey(dt.Digit) == d {
			return strings.TrimSpace(dt.NextID)
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

func estimatePromptTailMS(prompt string) int {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return 0
	}
	v := strings.TrimSpace(strings.ToLower(utils.GetEnv(constants.EnvSIPScriptListenAfterTTSTail)))
	switch v {
	case "0", "false", "off", "no":
		return 0
	}
	maxMS := 2000
	if s := strings.TrimSpace(utils.GetEnv(constants.EnvSIPScriptListenTailMSMax)); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			maxMS = n
		}
	}
	minMS := 400
	if s := strings.TrimSpace(utils.GetEnv(constants.EnvSIPScriptListenTailMSMin)); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 0 {
			minMS = n
		}
	}
	n := utf8.RuneCountInString(prompt)
	ms := n * 140
	if ms < minMS {
		ms = minMS
	}
	if ms > maxMS {
		ms = maxMS
	}
	return ms
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
