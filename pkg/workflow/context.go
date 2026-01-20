package workflow

import (
	"strings"
	"time"
)

// LogSender interface for sending logs via WebSocket or other channels
type LogSender interface {
	SendLog(log ExecutionLog) error
}

type WorkflowContext struct {
	WorkflowID  string                 // workflow id
	CurrentNode string                 // current node id
	Status      NodeStatus             // node status
	Parameters  map[string]interface{} // global parameters or data
	NodeData    map[string]interface{} // storage node data
	NodeStatus  map[string]NodeStatus  // all node status
	History     []NodeExecutionRecord  // execution history
	Logs        []ExecutionLog         // execution logs for frontend display
	LogSender   LogSender              // optional log sender for real-time streaming
}

// ExecutionLog represents a log entry for frontend terminal display
type ExecutionLog struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"` // info, success, warning, error, debug
	Message   string `json:"message"`
	NodeID    string `json:"nodeId,omitempty"`
	NodeName  string `json:"nodeName,omitempty"`
}

// NodeExecutionRecord track node execution timeline
type NodeExecutionRecord struct {
	NodeID    string
	Status    NodeStatus
	Timestamp time.Time
	Error     string
}

// NewWorkflowContext helper to build context with initialized maps
func NewWorkflowContext(workflowID string) *WorkflowContext {
	return &WorkflowContext{
		WorkflowID: workflowID,
		Parameters: make(map[string]interface{}),
		NodeData:   make(map[string]interface{}),
		NodeStatus: make(map[string]NodeStatus),
		Logs:       make([]ExecutionLog, 0),
	}
}

// AddLog adds a log entry to the context and optionally sends it via LogSender
func (ctx *WorkflowContext) AddLog(level, message, nodeID, nodeName string) {
	if ctx == nil {
		return
	}
	if ctx.Logs == nil {
		ctx.Logs = make([]ExecutionLog, 0)
	}
	log := ExecutionLog{
		Timestamp: time.Now().Format("15:04:05.000"),
		Level:     level,
		Message:   message,
		NodeID:    nodeID,
		NodeName:  nodeName,
	}
	ctx.Logs = append(ctx.Logs, log)

	// Send log via WebSocket if LogSender is set
	if ctx.LogSender != nil {
		_ = ctx.LogSender.SendLog(log)
	}
}

// SetNodeStatus update node status & append history
func (ctx *WorkflowContext) SetNodeStatus(nodeID string, status NodeStatus, nodeErr error) {
	if ctx == nil {
		return
	}
	if ctx.NodeStatus == nil {
		ctx.NodeStatus = make(map[string]NodeStatus)
	}
	ctx.NodeStatus[nodeID] = status
	record := NodeExecutionRecord{
		NodeID:    nodeID,
		Status:    status,
		Timestamp: time.Now(),
	}
	if nodeErr != nil {
		record.Error = nodeErr.Error()
	}
	ctx.History = append(ctx.History, record)
}

// GetNodeStatus returns current status for a node
func (ctx *WorkflowContext) GetNodeStatus(nodeID string) NodeStatus {
	if ctx == nil || ctx.NodeStatus == nil {
		return ""
	}
	return ctx.NodeStatus[nodeID]
}

// ResolveValue fetch value from node data or global parameters
// Supports dot notation like "parameters.value" or "context.key" or "context.response.body.data.user.name"
// Supports deep nesting with multiple dots
func (ctx *WorkflowContext) ResolveValue(key string) (interface{}, bool) {
	if ctx == nil {
		return nil, false
	}

	// Support dot notation: "parameters.xxx" or "context.xxx" or "context.xxx.yyy.zzz"
	if strings.Contains(key, ".") {
		parts := strings.Split(key, ".")
		if len(parts) >= 2 {
			prefix := parts[0]
			path := parts[1:]

			switch prefix {
			case "parameters":
				if ctx.Parameters != nil {
					// Navigate through nested structure
					var current interface{} = ctx.Parameters
					for _, part := range path {
						if currentMap, ok := current.(map[string]interface{}); ok {
							if val, ok := currentMap[part]; ok {
								current = val
							} else {
								return nil, false
							}
						} else {
							return nil, false
						}
					}
					return current, true
				}
			case "context":
				if ctx.NodeData != nil {
					// Navigate through nested structure
					var current interface{} = ctx.NodeData
					for _, part := range path {
						if currentMap, ok := current.(map[string]interface{}); ok {
							if val, ok := currentMap[part]; ok {
								current = val
							} else {
								return nil, false
							}
						} else {
							return nil, false
						}
					}
					return current, true
				}
			default:
				// Try as node ID: "nodeId.output.field"
				// First part might be a node ID, rest is the path
				if ctx.NodeData != nil {
					if val, ok := ctx.NodeData[prefix]; ok {
						// Navigate through nested structure
						var current interface{} = val
						for _, part := range path {
							if currentMap, ok := current.(map[string]interface{}); ok {
								if val, ok := currentMap[part]; ok {
									current = val
								} else {
									return nil, false
								}
							} else {
								return nil, false
							}
						}
						return current, true
					}
				}
			}
		}
	}

	// Try direct key lookup in NodeData first
	if ctx.NodeData != nil {
		if val, ok := ctx.NodeData[key]; ok {
			return val, true
		}
	}

	// Then try Parameters
	if ctx.Parameters != nil {
		if val, ok := ctx.Parameters[key]; ok {
			return val, true
		}
	}

	return nil, false
}

// SetData writes data to node data map
func (ctx *WorkflowContext) SetData(key string, value interface{}) {
	if ctx == nil {
		return
	}
	if ctx.NodeData == nil {
		ctx.NodeData = make(map[string]interface{})
	}
	ctx.NodeData[key] = value
}
