package workflow

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// KnowledgeBaseNode recalls documents from a bound knowledge namespace.
type KnowledgeBaseNode struct {
	*Node
	Config *KnowledgeBaseConfig
}

// KnowledgeBaseConfig configuration for knowledge base recall node.
type KnowledgeBaseConfig struct {
	NamespaceID    string  `json:"namespaceId"`
	NamespaceName  string  `json:"namespaceName,omitempty"`
	InputVariable  string  `json:"inputVariable"`
	OutputVariable string  `json:"outputVariable"`
	TopK           int     `json:"topK"`
	MinScore       float64 `json:"minScore"`
	OutputFormat   string  `json:"outputFormat"` // hits | text_block
}

func (k *KnowledgeBaseNode) Base() *Node {
	return k.Node
}

func (k *KnowledgeBaseNode) Run(ctx *WorkflowContext) ([]string, error) {
	ctx.SetNodeStatus(k.ID, NodeStatusRunning, nil)

	inputs, err := k.PrepareInputs(ctx)
	if err != nil {
		ctx.SetNodeStatus(k.ID, NodeStatusFailed, err)
		return nil, fmt.Errorf("prepare inputs failed: %w", err)
	}

	outputs, err := k.executeRecall(ctx, inputs)
	if err != nil {
		ctx.SetNodeStatus(k.ID, NodeStatusFailed, err)
		return nil, fmt.Errorf("knowledge recall failed: %w", err)
	}

	k.PersistOutputs(ctx, outputs)
	ctx.SetNodeStatus(k.ID, NodeStatusCompleted, nil)
	return k.NextNodes, nil
}

func (k *KnowledgeBaseNode) executeRecall(ctx *WorkflowContext, inputs map[string]interface{}) (map[string]interface{}, error) {
	if k.Config == nil {
		return nil, fmt.Errorf("knowledge base config is nil")
	}
	if strings.TrimSpace(k.Config.NamespaceID) == "" {
		return nil, fmt.Errorf("namespaceId is required")
	}

	query := ""
	if k.Config.InputVariable != "" {
		if val, ok := inputs[k.Config.InputVariable]; ok && val != nil {
			query = strings.TrimSpace(fmt.Sprintf("%v", val))
		}
	}
	if query == "" {
		return nil, fmt.Errorf("input variable '%s' not found or empty", k.Config.InputVariable)
	}

	if ctx.KnowledgeRecaller == nil {
		return nil, fmt.Errorf("knowledge recaller is not configured")
	}
	if ctx.GroupID == 0 {
		return nil, fmt.Errorf("workflow group id is required for knowledge recall")
	}

	topK := k.Config.TopK
	if topK <= 0 {
		topK = 5
	}
	outputFormat := strings.TrimSpace(k.Config.OutputFormat)
	if outputFormat == "" {
		outputFormat = "text_block"
	}

	hits, textBlock, err := ctx.KnowledgeRecaller.RecallKnowledge(
		context.Background(),
		ctx.GroupID,
		k.Config.NamespaceID,
		query,
		topK,
		k.Config.MinScore,
		outputFormat,
	)
	if err != nil {
		return nil, err
	}

	outputKey := k.Config.OutputVariable
	if outputKey == "" {
		outputKey = "kb_result"
	}

	var output interface{}
	switch outputFormat {
	case "hits":
		output = hits
	default:
		output = textBlock
	}

	ctx.AddLog("info", fmt.Sprintf("Knowledge recall query=%q hits=%d", query, len(hits)), k.ID, k.Name)

	return map[string]interface{}{
		outputKey: output,
	}, nil
}

func NewKnowledgeBaseNode(node *Node, config *KnowledgeBaseConfig) *KnowledgeBaseNode {
	return &KnowledgeBaseNode{Node: node, Config: config}
}

func ParseKnowledgeBaseConfig(properties map[string]interface{}) (*KnowledgeBaseConfig, error) {
	if len(properties) == 0 {
		return nil, fmt.Errorf("knowledge base node requires properties")
	}

	config := &KnowledgeBaseConfig{
		NamespaceID:    propertyString(properties, "namespaceId", "namespace_id"),
		NamespaceName:  propertyString(properties, "namespaceName", "namespace_name"),
		InputVariable:  propertyString(properties, "inputVariable", "input_variable"),
		OutputVariable: propertyString(properties, "outputVariable", "output_variable"),
		OutputFormat:   propertyString(properties, "outputFormat", "output_format"),
		TopK:           propertyInt(properties, 5, "topK", "top_k"),
		MinScore:       propertyFloat(properties, 0, "minScore", "min_score"),
	}

	if strings.TrimSpace(config.NamespaceID) == "" {
		return nil, fmt.Errorf("namespaceId is required")
	}
	if strings.TrimSpace(config.InputVariable) == "" {
		return nil, fmt.Errorf("inputVariable is required")
	}
	if strings.TrimSpace(config.OutputVariable) == "" {
		return nil, fmt.Errorf("outputVariable is required")
	}

	if config.TopK <= 0 {
		config.TopK = 5
	}
	if config.TopK > 20 {
		config.TopK = 20
	}
	if strings.TrimSpace(config.OutputFormat) == "" {
		config.OutputFormat = "text_block"
	}

	return config, nil
}

func propertyString(props map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if v, ok := props[key]; ok && v != nil {
			if s := strings.TrimSpace(fmt.Sprintf("%v", v)); s != "" {
				return s
			}
		}
	}
	return ""
}

func propertyInt(props map[string]interface{}, def int, keys ...string) int {
	for _, key := range keys {
		v, ok := props[key]
		if !ok || v == nil {
			continue
		}
		switch n := v.(type) {
		case int:
			return n
		case int32:
			return int(n)
		case int64:
			return int(n)
		case float64:
			return int(n)
		case float32:
			return int(n)
		case json.Number:
			if i, err := n.Int64(); err == nil {
				return int(i)
			}
		case string:
			if i, err := strconv.Atoi(strings.TrimSpace(n)); err == nil {
				return i
			}
		}
	}
	return def
}

func propertyFloat(props map[string]interface{}, def float64, keys ...string) float64 {
	for _, key := range keys {
		v, ok := props[key]
		if !ok || v == nil {
			continue
		}
		switch n := v.(type) {
		case float64:
			return n
		case float32:
			return float64(n)
		case int:
			return float64(n)
		case int64:
			return float64(n)
		case json.Number:
			if f, err := n.Float64(); err == nil {
				return f
			}
		case string:
			if f, err := strconv.ParseFloat(strings.TrimSpace(n), 64); err == nil {
				return f
			}
		}
	}
	return def
}
