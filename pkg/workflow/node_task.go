package workflow

import (
	"encoding/json"
	"fmt"
)

type TaskNode struct {
	Node                                 // basic node
	TaskType      string                 // task type (http, transform, set_variable, delay, log)
	Action        string                 // action description (deprecated, use TaskType)
	RequiredInput map[string]interface{} // needed inputs
	Result        string
	Handler       func(ctx *WorkflowContext, inputs map[string]interface{}) (map[string]interface{}, error)
	Config        map[string]interface{} // task configuration
}

// Execute performs task execution using registered executors
func (t *TaskNode) Execute(ctx *WorkflowContext, inputs map[string]interface{}) (map[string]interface{}, error) {
	// Determine task type
	taskType := t.TaskType
	if taskType == "" {
		// Fallback to Action for backward compatibility
		taskType = t.Action
	}
	if taskType == "" {
		// Try to get from config
		if t.Config != nil {
			if tt, ok := t.Config["task_type"].(string); ok {
				taskType = tt
			} else if tt, ok := t.Config["type"].(string); ok {
				taskType = tt
			}
		}
	}
	if taskType == "" {
		taskType = "log" // Default to log task
	}

	// Get executor
	executor := GetTaskExecutor(taskType)
	if executor == nil {
		// Fallback to default behavior
		message := fmt.Sprintf("Executing task: %s (type: %s)", t.Name, taskType)
		if ctx != nil {
			ctx.AddLog("warning", fmt.Sprintf("No executor found for task type '%s', using default behavior", taskType), t.ID, t.Name)
			ctx.AddLog("info", message, t.ID, t.Name)
		} else {
			fmt.Printf("%s\n", message)
		}
		result := map[string]interface{}{
			"result": fmt.Sprintf("%s completed", t.Name),
		}
		if ctx != nil {
			ctx.AddLog("success", fmt.Sprintf("Task completed: %s", t.Name), t.ID, t.Name)
		}
		return result, nil
	}

	// Prepare config
	config := make(map[string]interface{})
	if t.Config != nil {
		// Deep copy config
		configBytes, _ := json.Marshal(t.Config)
		json.Unmarshal(configBytes, &config)
	}
	config["task_type"] = taskType

	// Execute task
	if ctx != nil {
		ctx.AddLog("info", fmt.Sprintf("Executing task: %s (type: %s)", t.Name, taskType), t.ID, t.Name)
	}

	outputs, err := executor.Execute(ctx, config, inputs)
	if err != nil {
		if ctx != nil {
			ctx.AddLog("error", fmt.Sprintf("Task execution failed: %v", err), t.ID, t.Name)
		}
		return nil, fmt.Errorf("task execution failed: %w", err)
	}

	if ctx != nil {
		ctx.AddLog("success", fmt.Sprintf("Task completed: %s", t.Name), t.ID, t.Name)
	}

	return outputs, nil
}

func (t *TaskNode) Base() *Node {
	return &t.Node
}

func (t *TaskNode) Run(ctx *WorkflowContext) ([]string, error) {
	inputs, err := t.Node.PrepareInputs(ctx)
	if err != nil {
		return nil, err
	}
	var outputs map[string]interface{}
	if t.Handler != nil {
		outputs, err = t.Handler(ctx, inputs)
	} else {
		outputs, err = t.Execute(ctx, inputs)
	}
	if err != nil {
		return nil, err
	}

	// Store outputs with node ID prefix for easy access
	if ctx != nil && ctx.NodeData != nil && outputs != nil {
		// Store outputs under node ID for reference: context.{nodeId}.{outputKey}
		nodeOutputs := make(map[string]interface{})
		for k, v := range outputs {
			nodeOutputs[k] = v
		}
		ctx.NodeData[t.ID] = nodeOutputs
		// Also store individual keys for backward compatibility
		for k, v := range outputs {
			ctx.NodeData[t.ID+"."+k] = v
		}
	}

	t.Node.PersistOutputs(ctx, outputs)
	return t.NextNodes, nil
}
