package workflow

import (
	"encoding/json"
	"fmt"
	"strings"
)

// EndNode the last node for the whole workflow
type EndNode struct {
	Node               // basic node
	ExitMessage string // the message to exit
}

func (e *EndNode) Complete() error {
	if e.ExitMessage != "" {
		fmt.Println("Exit Message:", e.ExitMessage)
	}
	return nil
}

func (e *EndNode) Base() *Node {
	return &e.Node
}

func (e *EndNode) Run(ctx *WorkflowContext) ([]string, error) {
	// End node automatically receives data from upstream nodes
	// and organizes them as workflow final output according to OutputParams

	// Try to get data from upstream nodes
	// End node automatically collects data from all available context data
	allData := make(map[string]interface{})

	// Try to get common output keys from context
	for i := 0; i < 20; i++ {
		key := fmt.Sprintf("output-%d", i)
		if val, ok := ctx.ResolveValue(key); ok {
			allData[key] = val
		}
	}

	// Also try to get data from node-specific keys (format: nodeId.outputName)
	// We need to flatten nested structures like {"nodeId": {"outputName": value}}
	for key, val := range ctx.NodeData {
		allData[key] = val

		// If the value is a map, also add its nested keys with dot notation
		if nestedMap, ok := val.(map[string]interface{}); ok {
			for nestedKey, nestedVal := range nestedMap {
				flatKey := fmt.Sprintf("%s.%s", key, nestedKey)
				allData[flatKey] = nestedVal
				if ctx != nil {
					ctx.AddLog("debug", fmt.Sprintf("EndNode: flattened %s -> %s", key, flatKey), e.ID, e.Name)
				}
			}
		}
	}

	// End node uses OutputParams to define the final output structure
	// OutputParams defines the field names for the workflow result
	if len(e.OutputParams) > 0 {
		// Map data to outputs according to OutputParams
		// OutputParams format: { "outputFieldName": "sourceKey" }
		// If sourceKey is empty or same as outputFieldName, try to find the value automatically
		outputs := make(map[string]interface{})
		for outputName, sourceKey := range e.OutputParams {
			// Try to get value from sourceKey
			if sourceKey != "" && sourceKey != outputName {
				if val, ok := ctx.ResolveValue(sourceKey); ok {
					outputs[outputName] = val
				} else {
					// Debug: log what we're looking for
					ctx.AddLog("debug", fmt.Sprintf("EndNode: ResolveValue failed for %s (sourceKey=%s). Available keys in allData: %v", outputName, sourceKey, getMapKeys(allData)), e.ID, e.Name)
					outputs[outputName] = nil
				}
			} else {
				// If sourceKey is empty or same as outputName, try to find value automatically
				// First try to get from allData using outputName
				if val, ok := allData[outputName]; ok {
					outputs[outputName] = val
				} else {
					// Try to get from context using outputName
					if val, ok := ctx.ResolveValue(outputName); ok {
						outputs[outputName] = val
					} else {
						// Try to get from any node output that might match
						found := false
						for key, val := range allData {
							if key == outputName || strings.HasSuffix(key, "."+outputName) {
								outputs[outputName] = val
								found = true
								break
							}
						}
						if !found {
							// Debug: log what we're looking for
							ctx.AddLog("debug", fmt.Sprintf("EndNode: Could not find %s in allData. Available keys: %v", outputName, getMapKeys(allData)), e.ID, e.Name)
							outputs[outputName] = nil
						}
					}
				}
			}
		}
		// Store outputs as workflow result
		ctx.SetData("workflow_result", outputs)

		// Log final output with values
		outputJSON, err := json.Marshal(outputs)
		if err == nil {
			ctx.AddLog("info", fmt.Sprintf("Output: %s", string(outputJSON)), e.ID, e.Name)
		}
	} else {
		// If no OutputParams defined, store all collected data as workflow result
		if len(allData) > 0 {
			ctx.SetData("workflow_result", allData)

			// Log final output with values
			outputJSON, err := json.Marshal(allData)
			if err == nil {
				ctx.AddLog("info", fmt.Sprintf("Output: %s", string(outputJSON)), e.ID, e.Name)
			}
		} else {
			ctx.SetData("workflow_result", map[string]interface{}{})
		}
	}

	if err := e.Complete(); err != nil {
		return nil, err
	}
	return nil, nil
}
