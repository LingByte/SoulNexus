package workflow

import (
	"errors"
	"fmt"
)

var (
	// ErrStartNodeMissing returned when workflow lacks start node
	ErrStartNodeMissing = errors.New("workflow start node is not defined")
)

// Workflow orchestrates nodes
type Workflow struct {
	ID          string
	Context     *WorkflowContext
	Nodes       map[string]ExecutableNode
	StartNodeID string
	EndNodeID   string
	MaxSteps    int
}

// NewWorkflow creates workflow with sane defaults
func NewWorkflow(id string) *Workflow {
	return &Workflow{
		ID:       id,
		Nodes:    make(map[string]ExecutableNode),
		MaxSteps: 1000,
	}
}

// RegisterNode adds node definition into workflow
func (wf *Workflow) RegisterNode(node ExecutableNode) {
	if node == nil || node.Base() == nil {
		return
	}
	if wf.Nodes == nil {
		wf.Nodes = make(map[string]ExecutableNode)
	}
	wf.Nodes[node.Base().ID] = node
}

// SetStartNode define start node id
func (wf *Workflow) SetStartNode(nodeID string) {
	wf.StartNodeID = nodeID
}

// SetEndNode define end node id
func (wf *Workflow) SetEndNode(nodeID string) {
	wf.EndNodeID = nodeID
}

// Execute runs the workflow from start to end.
// It processes nodes in a breadth-first manner, executing each node and following its next nodes.
// Returns an error if the workflow exceeds max steps, a node fails, or the end node is not reached.
func (wf *Workflow) Execute() error {
	if err := wf.ensureReady(); err != nil {
		return err
	}
	if wf.Context == nil {
		return fmt.Errorf("workflow context is nil")
	}

	queue := []string{wf.StartNodeID}
	steps := 0
	visited := make(map[string]bool) // Track visited nodes to detect cycles

	for len(queue) > 0 {
		if wf.MaxSteps > 0 && steps >= wf.MaxSteps {
			return fmt.Errorf("workflow exceeded max steps %d", wf.MaxSteps)
		}
		steps++

		currentNodeID := queue[0]
		queue = queue[1:]

		// Detect cycles
		if visited[currentNodeID] {
			return fmt.Errorf("workflow cycle detected at node %s", currentNodeID)
		}
		visited[currentNodeID] = true

		node, ok := wf.Nodes[currentNodeID]
		if !ok {
			return fmt.Errorf("node %s not registered", currentNodeID)
		}

		wf.Context.CurrentNode = currentNodeID
		wf.Context.SetNodeStatus(currentNodeID, NodeStatusRunning, nil)

		nextNodes, err := node.Run(wf.Context)
		if err != nil {
			wf.Context.SetNodeStatus(currentNodeID, NodeStatusFailed, err)
			wf.Context.AddLog("error", fmt.Sprintf("Node execution failed: %s", err.Error()), currentNodeID, node.Base().Name)
			return fmt.Errorf("node %s execution failed: %w", currentNodeID, err)
		}

		wf.Context.SetNodeStatus(currentNodeID, NodeStatusCompleted, nil)
		wf.Context.AddLog("success", fmt.Sprintf("Node completed: %s", node.Base().Name), currentNodeID, node.Base().Name)

		if currentNodeID == wf.EndNodeID {
			return nil
		}

		queue = append(queue, nextNodes...)
	}

	if wf.EndNodeID != "" {
		return fmt.Errorf("workflow finished without reaching end node %s", wf.EndNodeID)
	}

	return nil
}

func (wf *Workflow) ensureReady() error {
	if wf.StartNodeID == "" {
		return ErrStartNodeMissing
	}
	if wf.Nodes == nil || len(wf.Nodes) == 0 {
		return fmt.Errorf("workflow %s has no registered nodes", wf.ID)
	}
	if _, ok := wf.Nodes[wf.StartNodeID]; !ok {
		return fmt.Errorf("start node %s not registered", wf.StartNodeID)
	}
	if wf.EndNodeID != "" {
		if _, ok := wf.Nodes[wf.EndNodeID]; !ok {
			return fmt.Errorf("end node %s not registered", wf.EndNodeID)
		}
	}
	if wf.Context == nil {
		wf.Context = NewWorkflowContext(wf.ID)
	}
	return nil
}
