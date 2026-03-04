package workflow

import (
	"encoding/json"
	"fmt"
)

// StartNode the first node for the whole workflow
type StartNode struct {
	Node                // basic node
	TriggerEvent string // start event
}

// Trigger starts the workflow execution
func (s *StartNode) Trigger(ctx *WorkflowContext) error {
	if ctx == nil {
		return fmt.Errorf("workflow context is nil")
	}
	// 不添加日志，让节点的 Run 方法处理
	return nil
}

func (s *StartNode) Base() *Node {
	return &s.Node
}

func (s *StartNode) Run(ctx *WorkflowContext) ([]string, error) {
	if err := s.Trigger(ctx); err != nil {
		return nil, err
	}

	// Start node receives initial parameters from context.Parameters (workflow input)
	// and passes them to downstream nodes as output data

	// Get input parameters from context.Parameters
	var inputs map[string]interface{}
	if len(s.InputParams) > 0 {
		// For start node, inputs come from context.Parameters (user-provided values)
		inputs = make(map[string]interface{})
		for alias, source := range s.InputParams {
			// Try to resolve from source first (in case it's a reference like "start-xxx.city")
			if val, ok := ctx.ResolveValue(source); ok {
				inputs[alias] = val
			} else if ctx.Parameters != nil {
				// Fall back to getting from context.Parameters using the alias
				if val, ok := ctx.Parameters[alias]; ok {
					inputs[alias] = val
				} else {
					// Try to get from context.Parameters using the source (in case source is a simple key)
					if val, ok := ctx.Parameters[source]; ok {
						inputs[alias] = val
					} else {
						inputs[alias] = nil
					}
				}
			} else {
				inputs[alias] = nil
			}
		}
	} else {
		// If no InputParams defined, use all context.Parameters as inputs
		if ctx.Parameters != nil {
			inputs = ctx.Parameters
		} else {
			inputs = make(map[string]interface{})
		}
	}

	// Start node passes input parameters to downstream nodes
	// Store input parameters in context so downstream nodes can access them
	if inputs != nil && len(inputs) > 0 {
		// Store each input parameter in context using the parameter name as key
		// This allows downstream nodes to access these values
		for key, value := range inputs {
			// Store in context using the input parameter name
			ctx.SetData(key, value)
			// Also store in a common output key for backward compatibility
			ctx.SetData(fmt.Sprintf("output-%s", key), value)
		}
		// Store all inputs together for easy access
		ctx.SetData("output-0", inputs)

		// Log input parameters with their values
		inputJSON, err := json.Marshal(inputs)
		if err == nil {
			ctx.AddLog("info", fmt.Sprintf("Input: %s", string(inputJSON)), s.ID, s.Name)
		}
	} else {
		// No inputs, set default start data
		startData := map[string]interface{}{
			"started":  true,
			"workflow": ctx.WorkflowID,
		}
		ctx.SetData("output-0", startData)
	}

	return s.NextNodes, nil
}
