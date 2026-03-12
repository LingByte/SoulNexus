package workflow

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"fmt"
	"time"

	"github.com/LingByte/SoulNexus/pkg/events"
)

// EventNode reacts to external/internal events
// Can be used to:
// 1. Publish events to the event bus (mode: "publish")
// 2. Wait for external events (mode: "wait") - similar to WaitNode but event-specific
type EventNode struct {
	Node
	EventType string                 // Event type to publish or wait for
	Mode      string                 // "publish" or "wait" (default: "publish")
	EventData map[string]interface{} // Data to include in published event
	Handler   func(ctx *WorkflowContext, payload map[string]interface{}) error
}

// PublishEvent publishes an event to the event bus
func (e *EventNode) PublishEvent(ctx *WorkflowContext, eventData map[string]interface{}) error {
	if e.EventType == "" {
		return fmt.Errorf("event type is required for publish mode")
	}

	// Merge event data from config and context
	data := make(map[string]interface{})
	if e.EventData != nil {
		for k, v := range e.EventData {
			data[k] = v
		}
	}
	// Merge provided event data
	for k, v := range eventData {
		data[k] = v
	}
	// Add workflow context data
	if ctx != nil {
		if ctx.NodeData != nil {
			for k, v := range ctx.NodeData {
				data["_context_"+k] = v
			}
		}
		data["_workflow_id"] = ctx.WorkflowID
		data["_node_id"] = e.ID
		data["_node_name"] = e.Name
	}

	// Publish event to events bus
	// This will:
	// 1. Trigger workflows configured with event triggers (via WorkflowEventListener)
	// 2. Trigger any code that subscribes to events bus (via events.Subscribe)
	events.PublishEvent(e.EventType, data, fmt.Sprintf("workflow:%s:node:%s", ctx.WorkflowID, e.ID))

	if ctx != nil {
		ctx.AddLog("info", fmt.Sprintf("Published event: %s", e.EventType), e.ID, e.Name)
		ctx.AddLog("debug", fmt.Sprintf("Event data: %v", data), e.ID, e.Name)
		ctx.AddLog("debug", fmt.Sprintf("Event will trigger workflows configured with event trigger: %s", e.EventType), e.ID, e.Name)
		ctx.AddLog("debug", "To listen to this event, use: events.GetEventBus().Subscribe(\""+e.EventType+"\", handler)", e.ID, e.Name)
	} else {
		fmt.Printf("Event node %s published event %s with data: %v\n", e.Name, e.EventType, data)
	}

	return nil
}

func (e *EventNode) Trigger(ctx *WorkflowContext, payload map[string]interface{}) error {
	if ctx != nil {
		ctx.AddLog("info", fmt.Sprintf("Event node %s triggered with type %s", e.Name, e.EventType), e.ID, e.Name)
	} else {
		fmt.Printf("Event node %s triggered with type %s\n", e.Name, e.EventType)
	}

	if e.Handler != nil {
		return e.Handler(ctx, payload)
	}
	return nil
}

func (e *EventNode) Base() *Node {
	return &e.Node
}

func (e *EventNode) Run(ctx *WorkflowContext) ([]string, error) {
	mode := e.Mode
	if mode == "" {
		// Try to get from properties
		if e.Properties != nil {
			if m, ok := e.Properties["mode"]; ok {
				mode = m
			}
		}
		if mode == "" {
			mode = "publish" // Default to publish
		}
	}

	switch mode {
	case "publish":
		// Get inputs as event data
		inputs, err := e.Node.PrepareInputs(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare inputs: %w", err)
		}

		// Publish event
		if err := e.PublishEvent(ctx, inputs); err != nil {
			return nil, fmt.Errorf("failed to publish event: %w", err)
		}

		// Store published event info in context
		if ctx != nil {
			if ctx.NodeData == nil {
				ctx.NodeData = make(map[string]interface{})
			}
			ctx.NodeData["published_event_type"] = e.EventType
			ctx.NodeData["published_event_time"] = time.Now()
		}

	case "wait":
		// Wait for event (similar to WaitNode but event-specific)
		if e.EventType == "" {
			return nil, fmt.Errorf("event type is required for wait mode")
		}

		if ctx != nil {
			ctx.AddLog("info", fmt.Sprintf("Waiting for event: %s", e.EventType), e.ID, e.Name)
			ctx.AddLog("warning", "Event waiting is not fully implemented. Workflow will continue immediately.", e.ID, e.Name)
		}
		// TODO: Implement actual event waiting mechanism
		// This would require:
		// 1. Registering a handler for the event type
		// 2. Suspending workflow execution
		// 3. Resuming when event is received
		// For now, just log and continue

	default:
		// Legacy behavior: just trigger
		if err := e.Trigger(ctx, nil); err != nil {
			return nil, err
		}
	}

	return e.NextNodes, nil
}
