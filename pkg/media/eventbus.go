package media

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"go.uber.org/zap"
)

// EventType represents the type of event
type EventType string

const (
	EventTypePacket    EventType = "packet"
	EventTypeState     EventType = "state"
	EventTypeError     EventType = "error"
	EventTypeLifecycle EventType = "lifecycle"
)

// MediaEvent represents an event in the event bus
type MediaEvent struct {
	Type      EventType
	Timestamp time.Time
	SessionID string
	Payload   interface{}
	Metadata  map[string]interface{}
}

// EventHandler processes events from the event bus
type EventHandler func(ctx context.Context, event *MediaEvent) error

// EventBus manages event distribution using pub/sub pattern
type EventBus struct {
	subscribers map[EventType][]EventHandler
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	eventQueue  chan *MediaEvent
	workers     int
	wg          sync.WaitGroup
}

// NewEventBus creates a new event bus
func NewEventBus(ctx context.Context, queueSize, workers int) *EventBus {
	busCtx, cancel := context.WithCancel(ctx)
	bus := &EventBus{
		subscribers: make(map[EventType][]EventHandler),
		ctx:         busCtx,
		cancel:      cancel,
		eventQueue:  make(chan *MediaEvent, queueSize),
		workers:     workers,
	}

	// Start worker goroutines
	for i := 0; i < workers; i++ {
		bus.wg.Add(1)
		go bus.worker(i)
	}

	return bus
}

// Subscribe registers an event handler for a specific event type
func (eb *EventBus) Subscribe(eventType EventType, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.subscribers[eventType] = append(eb.subscribers[eventType], handler)
}

// Unsubscribe removes an event handler
func (eb *EventBus) Unsubscribe(eventType EventType, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	handlers := eb.subscribers[eventType]
	for i, h := range handlers {
		if fmt.Sprintf("%p", h) == fmt.Sprintf("%p", handler) {
			eb.subscribers[eventType] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}
}

// Publish sends an event to all subscribers
func (eb *EventBus) Publish(event *MediaEvent) {
	select {
	case eb.eventQueue <- event:
	default:
		logger.Warn("event bus queue full, dropping event",
			zap.String("type", string(event.Type)),
			zap.String("sessionID", event.SessionID))
	}
}

// worker processes events from the queue
func (eb *EventBus) worker(id int) {
	defer eb.wg.Done()
	for {
		select {
		case <-eb.ctx.Done():
			return
		case event, ok := <-eb.eventQueue:
			if !ok {
				return
			}
			eb.dispatch(event)
		}
	}
}

// dispatch sends event to all registered handlers
func (eb *EventBus) dispatch(event *MediaEvent) {
	eb.mu.RLock()
	handlers := eb.subscribers[event.Type]
	// Also dispatch to wildcard subscribers if any
	if event.Type != EventTypeLifecycle {
		handlers = append(handlers, eb.subscribers[EventTypeLifecycle]...)
	}
	eb.mu.RUnlock()

	for _, handler := range handlers {
		func(h EventHandler) {
			defer func() {
				if r := recover(); r != nil {
					logger.Error("event handler panic",
						zap.String("type", string(event.Type)),
						zap.String("sessionID", event.SessionID),
						zap.Any("error", r))
				}
			}()
			if err := h(eb.ctx, event); err != nil {
				logger.Error("event handler error",
					zap.String("type", string(event.Type)),
					zap.String("sessionID", event.SessionID),
					zap.Error(err))
			}
		}(handler)
	}
}

// Close stops the event bus
func (eb *EventBus) Close() {
	eb.cancel()
	close(eb.eventQueue)
	eb.wg.Wait()
}

// PublishPacket publishes a packet event
func (eb *EventBus) PublishPacket(sessionID string, packet MediaPacket, sender interface{}) {
	eb.Publish(&MediaEvent{
		Type:      EventTypePacket,
		Timestamp: time.Now(),
		SessionID: sessionID,
		Payload:   packet,
		Metadata: map[string]interface{}{
			"sender": sender,
		},
	})
}

// PublishState publishes a state change event
func (eb *EventBus) PublishState(sessionID string, state StateChange, sender interface{}) {
	eb.Publish(&MediaEvent{
		Type:      EventTypeState,
		Timestamp: time.Now(),
		SessionID: sessionID,
		Payload:   state,
		Metadata: map[string]interface{}{
			"sender": sender,
		},
	})
}

// PublishError publishes an error event
func (eb *EventBus) PublishError(sessionID string, err error, sender interface{}) {
	eb.Publish(&MediaEvent{
		Type:      EventTypeError,
		Timestamp: time.Now(),
		SessionID: sessionID,
		Payload:   err,
		Metadata: map[string]interface{}{
			"sender": sender,
		},
	})
}
