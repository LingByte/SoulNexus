package outbound

import (
	"context"
	"testing"
	"time"
)

type inMemoryRecorder struct {
	events []ScriptRunEvent
}

func (r *inMemoryRecorder) Record(_ context.Context, event ScriptRunEvent) error {
	r.events = append(r.events, event)
	return nil
}

func TestParseHybridScript_ExtendedFields(t *testing.T) {
	raw := `{
		"id":"followup-v2",
		"start_id":"begin",
		"max_turns":6,
		"silence_timeout_ms":7000,
		"end_intents":["挂断","再见"],
		"steps":[
			{"id":"begin","type":"say","prompt":"你好","next_id":"listen_1"},
			{"id":"listen_1","type":"listen","listen_timeout_ms":3000,"listen_fallback_id":"end","next_id":"end"},
			{"id":"end","type":"end"}
		]
	}`
	s, err := ParseHybridScript(raw)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if s.MaxTurns != 6 {
		t.Fatalf("unexpected max_turns: %d", s.MaxTurns)
	}
	if s.SilenceTimeoutMS != 7000 {
		t.Fatalf("unexpected silence timeout: %d", s.SilenceTimeoutMS)
	}
	if len(s.EndIntents) != 2 {
		t.Fatalf("unexpected end_intents length: %d", len(s.EndIntents))
	}
}

func TestRunner_ListenTimeoutFallback(t *testing.T) {
	script, err := ParseHybridScript(`{
		"id":"timeout-flow",
		"start_id":"listen_1",
		"steps":[
			{"id":"listen_1","type":"listen","listen_timeout_ms":1,"listen_fallback_id":"bye","next_id":"reply"},
			{"id":"reply","type":"llm_reply","next_id":"bye"},
			{"id":"bye","type":"end"}
		]
	}`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	rec := &inMemoryRecorder{}
	r := NewHybridScriptRunner(script, rec).WithHooks(RuntimeHooks{
		OnListen: func(_ context.Context, _ EstablishedLeg, _ time.Duration) (ListenResult, error) {
			return ListenResult{}, context.DeadlineExceeded
		},
	})
	if err := r.Run(context.Background(), EstablishedLeg{CallID: "call-1"}); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if len(rec.events) == 0 {
		t.Fatalf("expected trace events")
	}
}

