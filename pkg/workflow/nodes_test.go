package workflow

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkflowLinearExecution(t *testing.T) {
	ctx := NewWorkflowContext("wf-linear")
	ctx.Parameters["approval_flag"] = true
	ctx.Parameters["request_message"] = "ping"

	start := &StartNode{
		Node: Node{
			ID:        "start",
			Name:      "Start",
			Type:      NodeTypeStart,
			NextNodes: []string{"task"},
		},
		TriggerEvent: "manual",
	}

	task := &TaskNode{
		Node: Node{
			ID:        "task",
			Name:      "Task",
			Type:      NodeTypeTask,
			NextNodes: []string{"gateway"},
			InputParams: map[string]string{
				"message": "request_message",
			},
			OutputParams: map[string]string{
				"approved": "task.approved",
			},
		},
		Action: "demo",
		Handler: func(ctx *WorkflowContext, inputs map[string]interface{}) (map[string]interface{}, error) {
			msg := inputs["message"].(string)
			return map[string]interface{}{
				"approved": msg == "ping",
			}, nil
		},
	}

	gateway := &GatewayNode{
		Node: Node{
			ID:        "gateway",
			Name:      "Gateway",
			Type:      NodeTypeGateway,
			NextNodes: []string{"end"},
		},
		Condition:       "task.approved",
		TrueNextNodeID:  "end",
		FalseNextNodeID: "task",
	}

	end := &EndNode{
		Node: Node{
			ID:   "end",
			Name: "End",
			Type: NodeTypeEnd,
		},
		ExitMessage: "done",
	}

	wf := NewWorkflow("wf-linear")
	wf.Context = ctx
	wf.SetStartNode("start")
	wf.SetEndNode("end")
	for _, node := range []ExecutableNode{start, task, gateway, end} {
		wf.RegisterNode(node)
	}

	require.NoError(t, wf.Execute())
	require.Equal(t, NodeStatusCompleted, ctx.GetNodeStatus("end"))
	require.Len(t, ctx.History, 8) // four nodes * (running+completed)
	val, ok := ctx.ResolveValue("task.approved")
	require.True(t, ok)
	require.Equal(t, true, val)
}

func TestWorkflowMissingInputFails(t *testing.T) {
	ctx := NewWorkflowContext("wf-missing")

	start := &StartNode{
		Node: Node{
			ID:        "start",
			Name:      "Start",
			Type:      NodeTypeStart,
			NextNodes: []string{"task"},
		},
	}

	task := &TaskNode{
		Node: Node{
			ID:        "task",
			Name:      "Task",
			Type:      NodeTypeTask,
			NextNodes: []string{"end"},
			InputParams: map[string]string{
				"must": "missing.key",
			},
		},
	}

	end := &EndNode{
		Node: Node{
			ID:   "end",
			Name: "End",
			Type: NodeTypeEnd,
		},
	}

	wf := NewWorkflow("wf-missing")
	wf.Context = ctx
	wf.SetStartNode("start")
	wf.SetEndNode("end")
	for _, node := range []ExecutableNode{start, task, end} {
		wf.RegisterNode(node)
	}

	err := wf.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing input")
	require.Equal(t, NodeStatusFailed, ctx.GetNodeStatus("task"))
}
