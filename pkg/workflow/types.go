package workflow

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"time"

	"gorm.io/gorm"
)

// StringMap is a string→string map used in workflow graph schemas.
type StringMap map[string]string

// WorkflowGraph captures nodes and edges for a workflow definition.
type WorkflowGraph struct {
	Nodes []WorkflowNodeSchema `json:"nodes"`
	Edges []WorkflowEdgeSchema `json:"edges"`
}

// WorkflowNodeSchema represents a single node within the workflow graph.
type WorkflowNodeSchema struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Type       string    `json:"type"`
	InputMap   StringMap `json:"inputMap,omitempty"`
	OutputMap  StringMap `json:"outputMap,omitempty"`
	Properties StringMap `json:"properties,omitempty"`
}

// WorkflowEdgeSchema links nodes together.
type WorkflowEdgeSchema struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// WorkflowDefinition describes a reusable workflow template.
type WorkflowDefinition struct {
	ID         uint          `json:"id" gorm:"primaryKey"`
	Name       string        `json:"name" gorm:"size:128;not null"`
	Definition WorkflowGraph `json:"definition" gorm:"type:json"`
}

// NodePluginPort describes one IO port on a node plugin.
type NodePluginPort struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Required bool        `json:"required"`
	Default  interface{} `json:"default,omitempty"`
}

// NodePluginDefinition is the executable shape of a node plugin.
type NodePluginDefinition struct {
	Type    string            `json:"type"`
	Inputs  []NodePluginPort  `json:"inputs"`
	Outputs []NodePluginPort  `json:"outputs"`
	Runtime NodePluginRuntime `json:"runtime"`
}

// NodePluginRuntime configures how a plugin executes.
type NodePluginRuntime struct {
	Type    string                 `json:"type"` // script | http | builtin
	Config  map[string]interface{} `json:"config"`
	Timeout int                    `json:"timeout,omitempty"`
}

// NodePlugin is a reusable workflow node plugin definition.
type NodePlugin struct {
	ID          uint                 `json:"id" gorm:"primaryKey"`
	DisplayName string               `json:"displayName" gorm:"size:128;not null"`
	Definition  NodePluginDefinition `json:"definition" gorm:"type:json"`
	CreatedAt   time.Time            `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt   time.Time            `json:"updatedAt" gorm:"autoUpdateTime"`
	DeletedAt   gorm.DeletedAt       `json:"-" gorm:"index"`
}

// WorkflowPluginParameter describes one plugin IO parameter.
type WorkflowPluginParameter struct {
	Name string `json:"name"`
}

// WorkflowPluginIOSchema holds plugin parameter definitions.
type WorkflowPluginIOSchema struct {
	Parameters []WorkflowPluginParameter `json:"parameters"`
}

// WorkflowPlugin publishes a workflow as a reusable plugin.
type WorkflowPlugin struct {
	ID               uint                   `json:"id" gorm:"primaryKey"`
	DisplayName      string                 `json:"displayName" gorm:"size:256;not null"`
	OutputSchema     WorkflowPluginIOSchema `json:"outputSchema" gorm:"type:json"`
	WorkflowSnapshot WorkflowGraph          `json:"workflowSnapshot" gorm:"type:json"`
	CreatedAt        time.Time              `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt        time.Time              `json:"updatedAt" gorm:"autoUpdateTime"`
	DeletedAt        gorm.DeletedAt         `json:"-" gorm:"index"`
}
