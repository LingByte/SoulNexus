package events

import (
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"go.uber.org/zap"
)

// Event system event
type Event struct {
	Type      string                 `json:"type"`      // Event type, e.g. "user.created", "order.paid"
	Timestamp time.Time              `json:"timestamp"` // Event timestamp
	Data      map[string]interface{} `json:"data"`      // Event data
	Source    string                 `json:"source"`    // Event source
}

// EventHandler event handler function
type EventHandler func(event Event) error

// EventBus event bus
type EventBus struct {
	handlers       map[string][]EventHandler
	publishedTypes map[string]time.Time // Record all published event types and their first publish time
	mu             sync.RWMutex
}

var globalEventBus *EventBus
var once sync.Once

// GetEventBus gets global event bus instance
func GetEventBus() *EventBus {
	once.Do(func() {
		globalEventBus = &EventBus{
			handlers:       make(map[string][]EventHandler),
			publishedTypes: make(map[string]time.Time),
		}
	})
	return globalEventBus
}

// Subscribe subscribes to events
func (bus *EventBus) Subscribe(eventType string, handler EventHandler) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	if bus.handlers == nil {
		bus.handlers = make(map[string][]EventHandler)
	}

	bus.handlers[eventType] = append(bus.handlers[eventType], handler)
	logger.Info("Event handler subscribed",
		zap.String("eventType", eventType))
}

// Unsubscribe unsubscribes from events (removes all handlers for the type)
func (bus *EventBus) Unsubscribe(eventType string) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	delete(bus.handlers, eventType)
	logger.Info("Event handlers unsubscribed",
		zap.String("eventType", eventType))
}

// Publish publishes an event
func (bus *EventBus) Publish(event Event) {
	// If timestamp is not set, use current time
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Record published event types
	bus.mu.Lock()
	if bus.publishedTypes == nil {
		bus.publishedTypes = make(map[string]time.Time)
	}
	if _, exists := bus.publishedTypes[event.Type]; !exists {
		bus.publishedTypes[event.Type] = event.Timestamp
	}
	bus.mu.Unlock()

	bus.mu.RLock()
	// Get all matching handlers
	handlers := bus.handlers[event.Type]
	// Also handle wildcard "*"
	wildcardHandlers := bus.handlers["*"]

	allHandlers := append(handlers, wildcardHandlers...)
	bus.mu.RUnlock()

	if len(allHandlers) == 0 {
		logger.Debug("No handlers for event",
			zap.String("eventType", event.Type))
		return
	}

	logger.Info("Publishing event",
		zap.String("eventType", event.Type),
		zap.Int("handlerCount", len(allHandlers)))

	// Execute all handlers asynchronously
	for _, handler := range allHandlers {
		go func(h EventHandler) {
			if err := h(event); err != nil {
				logger.Error("Event handler failed",
					zap.String("eventType", event.Type),
					zap.Error(err))
			}
		}(handler)
	}
}

// GetPublishedEventTypes gets all published event types
func (bus *EventBus) GetPublishedEventTypes() map[string]time.Time {
	bus.mu.RLock()
	defer bus.mu.RUnlock()

	result := make(map[string]time.Time)
	for k, v := range bus.publishedTypes {
		result[k] = v
	}
	return result
}

// PublishEvent convenience method: publish event
func PublishEvent(eventType string, data map[string]interface{}, source string) {
	bus := GetEventBus()
	bus.Publish(Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
		Source:    source,
	})
}
