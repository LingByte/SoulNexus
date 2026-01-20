package workflow

import "time"

// TimerNode schedules recurring operations
type TimerNode struct {
	Node
	Interval time.Duration
	Timeout  time.Duration
}

func (t *TimerNode) StartTimer(ctx *WorkflowContext) {
	if t.Interval <= 0 || t.Timeout <= 0 {
		return
	}
	timer := time.NewTimer(t.Timeout)
	tick := time.NewTicker(t.Interval)
	defer timer.Stop()
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			// placeholder to hook timer ticks
		case <-timer.C:
			return
		}
	}
}

func (t *TimerNode) Base() *Node {
	return &t.Node
}

func (t *TimerNode) Run(ctx *WorkflowContext) ([]string, error) {
	t.StartTimer(ctx)
	return t.NextNodes, nil
}
