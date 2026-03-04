package workflow

// NodeType basic node type
type NodeType string

const (
	NodeTypeStart          NodeType = "start"
	NodeTypeEnd            NodeType = "end"
	NodeTypeTask           NodeType = "task"
	NodeTypeGateway        NodeType = "gateway"
	NodeTypeEvent          NodeType = "event"
	NodeTypeSubflow        NodeType = "subflow"
	NodeTypeCondition      NodeType = "condition"
	NodeTypeParallel       NodeType = "parallel"
	NodeTypeWait           NodeType = "wait"
	NodeTypeTimer          NodeType = "timer"
	NodeTypeScript         NodeType = "script"
	NodeTypePlugin         NodeType = "plugin"
	NodeTypeWorkflowPlugin NodeType = "workflow_plugin"
	NodeTypeAIChat         NodeType = "ai_chat"
)

func (nt NodeType) String() string {
	return string(nt)
}

type NodeStatus string

const (
	NodeStatusPending   NodeStatus = "pending"
	NodeStatusRunning   NodeStatus = "running"
	NodeStatusCompleted NodeStatus = "completed"
	NodeStatusFailed    NodeStatus = "failed"
	NodeStatusSkipped   NodeStatus = "skipped"
)
