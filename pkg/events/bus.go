package events

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"go.uber.org/zap"
)

// Event is a published bus event.
type Event struct {
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Source    string                 `json:"source"`
}

// EventHandler handles a published event.
type EventHandler func(event Event) error

// EventBus is an in-process pub/sub bus.
type EventBus struct {
	handlers       map[string][]EventHandler
	publishedTypes map[string]time.Time
	mu             sync.RWMutex
}

var (
	globalEventBus *EventBus
	once           sync.Once
)

// GetEventBus returns the process-wide event bus.
func GetEventBus() *EventBus {
	once.Do(func() {
		globalEventBus = &EventBus{
			handlers:       make(map[string][]EventHandler),
			publishedTypes: make(map[string]time.Time),
		}
	})
	return globalEventBus
}

// Subscribe registers a handler for eventType ("*" matches all).
func (bus *EventBus) Subscribe(eventType string, handler EventHandler) {
	bus.mu.Lock()
	defer bus.mu.Unlock()
	if bus.handlers == nil {
		bus.handlers = make(map[string][]EventHandler)
	}
	bus.handlers[eventType] = append(bus.handlers[eventType], handler)
}

// Unsubscribe removes all handlers for eventType.
func (bus *EventBus) Unsubscribe(eventType string) {
	bus.mu.Lock()
	defer bus.mu.Unlock()
	delete(bus.handlers, eventType)
}

// Publish dispatches an event to matching handlers asynchronously.
func (bus *EventBus) Publish(event Event) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	bus.mu.Lock()
	if bus.publishedTypes == nil {
		bus.publishedTypes = make(map[string]time.Time)
	}
	if _, exists := bus.publishedTypes[event.Type]; !exists {
		bus.publishedTypes[event.Type] = event.Timestamp
	}
	handlers := append([]EventHandler{}, bus.handlers[event.Type]...)
	handlers = append(handlers, bus.handlers["*"]...)
	bus.mu.Unlock()

	if len(handlers) == 0 {
		return
	}
	for _, handler := range handlers {
		go func(h EventHandler) {
			if err := h(event); err != nil {
				logger.Error("Event handler failed",
					zap.String("eventType", event.Type),
					zap.Error(err))
			}
		}(handler)
	}
}

// GetPublishedEventTypes returns first-seen timestamps for published types.
func (bus *EventBus) GetPublishedEventTypes() map[string]time.Time {
	bus.mu.RLock()
	defer bus.mu.RUnlock()
	result := make(map[string]time.Time, len(bus.publishedTypes))
	for k, v := range bus.publishedTypes {
		result[k] = v
	}
	return result
}

// PublishEvent publishes to the global bus.
func PublishEvent(eventType string, data map[string]interface{}, source string) {
	GetEventBus().Publish(Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
		Source:    source,
	})
}
