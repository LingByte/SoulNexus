package workflowdef

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	runtimewf "github.com/LingByte/SoulNexus/pkg/workflow"
	"gorm.io/gorm"
)

// BuildRuntimeWorkflow converts a persisted workflow definition into an executable workflow instance.
func BuildRuntimeWorkflow(def *models.WorkflowDefinition, db *gorm.DB) (*runtimewf.Workflow, error) {
	if def == nil {
		return nil, fmt.Errorf("workflow definition is nil")
	}
	if len(def.Definition.Nodes) == 0 {
		return nil, fmt.Errorf("workflow definition %s has no nodes", def.Name)
	}

	wf := runtimewf.NewWorkflow(fmt.Sprintf("definition-%d", def.ID))
	if db != nil {
		wf.Context = runtimewf.NewWorkflowContextWithDB(fmt.Sprintf("definition-%d", def.ID), db)
	} else {
		wf.Context = runtimewf.NewWorkflowContext(fmt.Sprintf("definition-%d", def.ID))
	}

	nodeRegistry := make(map[string]runtimewf.ExecutableNode, len(def.Definition.Nodes))
	startCount := 0
	endCount := 0

	for _, nodeSchema := range def.Definition.Nodes {
		if nodeSchema.ID == "" {
			return nil, fmt.Errorf("node id cannot be empty")
		}
		if _, exists := nodeRegistry[nodeSchema.ID]; exists {
			return nil, fmt.Errorf("duplicate node id %s", nodeSchema.ID)
		}
		baseNode := runtimewf.Node{
			ID:           nodeSchema.ID,
			Name:         nodeSchema.Name,
			Type:         runtimewf.NodeType(nodeSchema.Type),
			InputParams:  toNativeMap(nodeSchema.InputMap),
			OutputParams: toNativeMap(nodeSchema.OutputMap),
			Properties:   toNativeMap(nodeSchema.Properties),
		}
		execNode, err := instantiateNode(baseNode)
		if err != nil {
			return nil, fmt.Errorf("node %s: %w", nodeSchema.ID, err)
		}
		if baseNode.Type == runtimewf.NodeTypeStart {
			startCount++
		}
		if baseNode.Type == runtimewf.NodeTypeEnd {
			endCount++
		}
		nodeRegistry[baseNode.ID] = execNode
	}

	if startCount != 1 {
		return nil, fmt.Errorf("workflow requires exactly one start node, got %d", startCount)
	}
	if endCount == 0 {
		return nil, fmt.Errorf("workflow requires at least one end node")
	}

	// Build adjacency map for node successors based on edges
	successors := make(map[string][]string)
	for _, edge := range def.Definition.Edges {
		if edge.Source == "" || edge.Target == "" {
			// Skip invalid edges (should be caught by validation, but handle gracefully)
			continue
		}
		if _, ok := nodeRegistry[edge.Source]; !ok {
			return nil, fmt.Errorf("edge references unknown source node %s", edge.Source)
		}
		if _, ok := nodeRegistry[edge.Target]; !ok {
			return nil, fmt.Errorf("edge references unknown target node %s", edge.Target)
		}
		successors[edge.Source] = append(successors[edge.Source], edge.Target)
		// Assign edge-specific metadata (e.g., condition expressions) to nodes
		AssignEdgeMetadata(nodeRegistry[edge.Source], edge)
	}

	var startID, endID string
	for id, node := range nodeRegistry {
		if node.Base() == nil {
			continue
		}
		if next, ok := successors[id]; ok {
			node.Base().NextNodes = next
		}
		switch node.Base().Type {
		case runtimewf.NodeTypeStart:
			if startID == "" {
				startID = id
			}
		case runtimewf.NodeTypeEnd:
			if endID == "" {
				endID = id
			}
		case runtimewf.NodeTypeParallel:
			if parallelNode, ok := node.(*runtimewf.ParallelNode); ok {
				parallelNode.Branches = successors[id]
			}
		}
		wf.RegisterNode(node)
	}

	if startID == "" {
		return nil, fmt.Errorf("workflow definition %s missing start node", def.Name)
	}
	wf.SetStartNode(startID)
	if endID != "" {
		wf.SetEndNode(endID)
	}

	return wf, nil
}

// BuildRuntimeNode converts a single node definition into an executable node instance.
func BuildRuntimeNode(nodeSchema *models.WorkflowNodeSchema, graph *models.WorkflowGraph) (runtimewf.ExecutableNode, error) {
	if nodeSchema == nil {
		return nil, fmt.Errorf("node schema is nil")
	}
	if nodeSchema.ID == "" {
		return nil, fmt.Errorf("node id cannot be empty")
	}

	baseNode := runtimewf.Node{
		ID:           nodeSchema.ID,
		Name:         nodeSchema.Name,
		Type:         runtimewf.NodeType(nodeSchema.Type),
		InputParams:  toNativeMap(nodeSchema.InputMap),
		OutputParams: toNativeMap(nodeSchema.OutputMap),
		Properties:   toNativeMap(nodeSchema.Properties),
	}

	execNode, err := instantiateNode(baseNode)
	if err != nil {
		return nil, fmt.Errorf("node %s: %w", nodeSchema.ID, err)
	}

	return execNode, nil
}

func instantiateNode(base runtimewf.Node) (runtimewf.ExecutableNode, error) {
	switch base.Type {
	case runtimewf.NodeTypeStart:
		return &runtimewf.StartNode{Node: base}, nil
	case runtimewf.NodeTypeEnd:
		return &runtimewf.EndNode{Node: base}, nil
	case runtimewf.NodeTypeTask:
		taskNode := &runtimewf.TaskNode{Node: base}
		// Extract task configuration from properties
		if base.Properties != nil {
			// Parse task type
			if taskType, ok := base.Properties["task_type"]; ok {
				taskNode.TaskType = taskType
			} else if taskType, ok := base.Properties["type"]; ok {
				taskNode.TaskType = taskType
			}

			// Parse action (for backward compatibility)
			if action, ok := base.Properties["action"]; ok {
				taskNode.Action = action
			}

			// Parse task config (all properties except type/action)
			taskNode.Config = make(map[string]interface{})
			for k, v := range base.Properties {
				if k != "task_type" && k != "type" && k != "action" {
					// Try to parse as JSON if it looks like JSON
					var jsonValue interface{}
					if err := json.Unmarshal([]byte(v), &jsonValue); err == nil {
						taskNode.Config[k] = jsonValue
					} else {
						// If not valid JSON, use as string
						taskNode.Config[k] = v
					}
				}
			}
		}
		return taskNode, nil
	case runtimewf.NodeTypeGateway:
		gatewayNode := &runtimewf.GatewayNode{Node: base}
		// Extract condition/expression from properties if present
		if base.Properties != nil {
			// Check mode: 'value' (context key) or 'expression' (expression evaluation)
			mode := base.Properties["mode"]
			if mode == "expression" {
				// Expression mode: use expression field
				if expression, ok := base.Properties["expression"]; ok {
					gatewayNode.Expression = expression
				}
			} else {
				// Value mode (default): use condition field as context key
				if condition, ok := base.Properties["condition"]; ok {
					gatewayNode.Condition = condition
					// Debug: log the condition being set
					fmt.Printf("[DEBUG] GatewayNode %s: Setting condition from properties: '%s'\n", base.ID, condition)
				} else {
					keys := make([]string, 0, len(base.Properties))
					for k := range base.Properties {
						keys = append(keys, k)
					}
					fmt.Printf("[DEBUG] GatewayNode %s: No condition found in properties. Available keys: %v\n", base.ID, keys)
				}
			}
			// Check if result should be stored
			if storeResult, ok := base.Properties["store_result"]; ok {
				gatewayNode.StoreResult = storeResult == "true" || storeResult == "1"
			}
		} else {
			fmt.Printf("[DEBUG] GatewayNode %s: Properties is nil\n", base.ID)
		}
		return gatewayNode, nil
	case runtimewf.NodeTypeEvent:
		eventNode := &runtimewf.EventNode{Node: base}
		// Extract event configuration from properties
		if base.Properties != nil {
			// Parse event type
			if eventType, ok := base.Properties["event_type"]; ok {
				eventNode.EventType = eventType
			} else if eventType, ok := base.Properties["eventType"]; ok {
				eventNode.EventType = eventType
			}

			// Parse mode
			if mode, ok := base.Properties["mode"]; ok {
				eventNode.Mode = mode
			}

			// Parse event data (JSON string)
			if eventDataStr, ok := base.Properties["event_data"]; ok {
				var eventData map[string]interface{}
				if err := json.Unmarshal([]byte(eventDataStr), &eventData); err == nil {
					eventNode.EventData = eventData
				}
			} else if eventDataStr, ok := base.Properties["eventData"]; ok {
				var eventData map[string]interface{}
				if err := json.Unmarshal([]byte(eventDataStr), &eventData); err == nil {
					eventNode.EventData = eventData
				}
			}
		}
		return eventNode, nil
	case runtimewf.NodeTypeSubflow:
		return &runtimewf.SubflowNode{Node: base}, nil
	case runtimewf.NodeTypeCondition:
		// Map condition node to gateway node for backward compatibility
		gatewayNode := &runtimewf.GatewayNode{Node: base}
		if base.Properties != nil {
			// Migrate condition node properties to gateway node
			if expression, ok := base.Properties["expression"]; ok {
				gatewayNode.Expression = expression
			} else if condition, ok := base.Properties["condition"]; ok {
				gatewayNode.Expression = condition
			}
			// Condition nodes always store result
			gatewayNode.StoreResult = true
		}
		return gatewayNode, nil
	case runtimewf.NodeTypeParallel:
		return &runtimewf.ParallelNode{Node: base}, nil
	case runtimewf.NodeTypeWait:
		return &runtimewf.WaitNode{Node: base}, nil
	case runtimewf.NodeTypeTimer:
		timerNode := &runtimewf.TimerNode{Node: base}
		// Extract timer configuration from properties
		if base.Properties != nil {
			if delayStr, ok := base.Properties["delay"]; ok {
				if delayMs, err := strconv.Atoi(delayStr); err == nil {
					timerNode.Delay = time.Duration(delayMs) * time.Millisecond
				}
			}
			if repeatStr, ok := base.Properties["repeat"]; ok {
				timerNode.Repeat = repeatStr == "true"
			}
		}
		return timerNode, nil
	case runtimewf.NodeTypeScript:
		scriptNode := &runtimewf.ScriptNode{Node: base}
		// Extract script code from properties
		if base.Properties != nil {
			if code, ok := base.Properties["code"]; ok {
				scriptNode.Script = code
			}
			if language, ok := base.Properties["language"]; ok {
				// Store language in properties for reference
				// Currently only Go is supported
				_ = language
			}
		}
		return scriptNode, nil
	case runtimewf.NodeTypePlugin:
		pluginNode := &runtimewf.PluginNode{Node: base}
		// Extract plugin configuration from properties
		if base.Properties != nil {
			// Parse plugin ID
			if pluginIDStr, ok := base.Properties["plugin_id"]; ok {
				// TODO: Load plugin definition from database
				// For now, we'll need to implement plugin loading logic
				_ = pluginIDStr
			}

			// Parse user configuration
			if configStr, ok := base.Properties["config"]; ok {
				var config map[string]interface{}
				if err := json.Unmarshal([]byte(configStr), &config); err == nil {
					pluginNode.Config = config
				}
			}
		}
		return pluginNode, nil
	case runtimewf.NodeTypeWorkflowPlugin:
		// 工作流插件节点
		if base.Properties == nil {
			return nil, fmt.Errorf("workflow plugin node requires properties")
		}

		pluginIDStr, ok := base.Properties["pluginId"]
		if !ok {
			return nil, fmt.Errorf("workflow plugin node requires pluginId property")
		}

		pluginID, err := strconv.ParseUint(pluginIDStr, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid pluginId: %v", err)
		}

		// 解析用户配置的参数
		parameters := make(map[string]interface{})
		if parametersStr, ok := base.Properties["parameters"]; ok && parametersStr != "" {
			// Handle empty parameters or "[object Object]" case
			if parametersStr == "{}" || parametersStr == "[object Object]" {
				// Empty parameters, use empty map
				parameters = make(map[string]interface{})
			} else {
				if err := json.Unmarshal([]byte(parametersStr), &parameters); err != nil {
					return nil, fmt.Errorf("invalid parameters JSON: %v", err)
				}
			}
		}

		// 这里需要数据库连接，但在这个上下文中我们没有
		// 我们需要修改这个函数的签名来传递数据库连接
		// 暂时返回一个基础的工作流插件节点，稍后在执行时加载插件信息
		workflowPluginNode := &runtimewf.WorkflowPluginNode{
			Node:       base,
			PluginID:   uint(pluginID),
			Parameters: parameters,
		}
		return workflowPluginNode, nil
	case runtimewf.NodeTypeAIChat:
		// AI对话节点
		aiChatNode := &runtimewf.AIChatNode{Node: &base}
		// Extract AI chat configuration from properties
		if base.Properties != nil {
			// Convert map[string]string to map[string]interface{}
			properties := make(map[string]interface{})
			for k, v := range base.Properties {
				properties[k] = v
			}
			config, err := runtimewf.ParseAIChatConfig(properties)
			if err != nil {
				return nil, fmt.Errorf("parse AI chat config failed: %w", err)
			}
			aiChatNode.Config = config
		} else {
			return nil, fmt.Errorf("AI chat node requires properties")
		}
		return aiChatNode, nil
	default:
		return nil, fmt.Errorf("unsupported node type %s", base.Type)
	}
}

// AssignEdgeMetadata assigns edge-specific metadata to nodes (e.g., condition expressions, next node IDs)
func AssignEdgeMetadata(node runtimewf.ExecutableNode, edge models.WorkflowEdgeSchema) {
	if node == nil {
		return
	}
	switch n := node.(type) {
	case *runtimewf.GatewayNode:
		switch edge.Type {
		case models.WorkflowEdgeTypeTrue:
			n.TrueNextNodeID = edge.Target
		case models.WorkflowEdgeTypeFalse:
			n.FalseNextNodeID = edge.Target
		}
		if edge.Condition != "" && n.Condition == "" {
			n.Condition = edge.Condition
		}
	case *runtimewf.ConditionNode:
		// ConditionNode is deprecated, but handle for backward compatibility
		switch edge.Type {
		case models.WorkflowEdgeTypeTrue:
			if n.Properties == nil {
				n.Properties = map[string]string{}
			}
			n.Properties["true_next"] = edge.Target
		case models.WorkflowEdgeTypeFalse:
			if n.Properties == nil {
				n.Properties = map[string]string{}
			}
			n.Properties["false_next"] = edge.Target
		}
		if edge.Condition != "" && n.Expression == "" {
			n.Expression = edge.Condition
		}
	}
}

func toNativeMap(sm models.StringMap) map[string]string {
	if len(sm) == 0 {
		return nil
	}
	out := make(map[string]string, len(sm))
	for k, v := range sm {
		out[k] = v
	}
	return out
}
