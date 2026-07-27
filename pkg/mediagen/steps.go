package mediagen

import (
	"time"
)

type stepTracker struct {
	steps []JobStep
}

func newStepTracker() *stepTracker {
	return &stepTracker{steps: make([]JobStep, 0, 8)}
}

func (t *stepTracker) begin(name string) int {
	now := time.Now()
	t.steps = append(t.steps, JobStep{
		Name:      name,
		Status:    "running",
		StartedAt: &now,
	})
	return len(t.steps) - 1
}

func (t *stepTracker) endOK(idx int, message string, meta map[string]any) {
	t.end(idx, "succeeded", message, meta)
}

func (t *stepTracker) endFail(idx int, err error, meta map[string]any) {
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	t.end(idx, "failed", msg, meta)
}

func (t *stepTracker) end(idx int, status, message string, meta map[string]any) {
	if t == nil || idx < 0 || idx >= len(t.steps) {
		return
	}
	now := time.Now()
	step := &t.steps[idx]
	step.Status = status
	step.FinishedAt = &now
	if step.StartedAt != nil {
		step.DurationMs = now.Sub(*step.StartedAt).Milliseconds()
	}
	step.Message = message
	if meta != nil {
		step.Meta = meta
	}
}

func (t *stepTracker) snapshot() []JobStep {
	if t == nil {
		return nil
	}
	out := make([]JobStep, len(t.steps))
	copy(out, t.steps)
	return out
}
