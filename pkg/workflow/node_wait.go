package workflow

import (
	"fmt"
	"time"
)

// WaitNode pauses the workflow until duration or event met
type WaitNode struct {
	Node
	Duration   time.Duration
	UntilEvent string
}

func (w *WaitNode) Wait(ctx *WorkflowContext) {
	if w.Duration > 0 {
		fmt.Printf("Wait node %s sleeping for %s\n", w.Name, w.Duration)
		time.Sleep(w.Duration)
		return
	}
	fmt.Printf("Wait node %s waiting for event %s\n", w.Name, w.UntilEvent)
}

func (w *WaitNode) Base() *Node {
	return &w.Node
}

func (w *WaitNode) Run(ctx *WorkflowContext) ([]string, error) {
	w.Wait(ctx)
	return w.NextNodes, nil
}
