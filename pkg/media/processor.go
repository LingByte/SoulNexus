package media

import (
	"context"
	"fmt"
	"sync"
)

// ProcessorPriority defines processing priority
type ProcessorPriority int

const (
	PriorityLow    ProcessorPriority = 0
	PriorityNormal ProcessorPriority = 50
	PriorityHigh   ProcessorPriority = 100
)

// ProcessorCondition determines if a processor should handle an event
type ProcessorCondition func(ctx context.Context, event *MediaEvent) bool

// Processor handles media events
type Processor interface {
	// Name returns the processor name
	Name() string

	// Priority returns processing priority (higher = processed first)
	Priority() ProcessorPriority

	// CanHandle checks if this processor can handle the event
	CanHandle(ctx context.Context, event *MediaEvent) bool

	// Process handles the event
	Process(ctx context.Context, session *MediaSession, event *MediaEvent) error
}

// ProcessorRegistry manages registered processors
type ProcessorRegistry struct {
	processors []Processor
	mu         sync.RWMutex
}

// NewProcessorRegistry creates a new processor registry
func NewProcessorRegistry() *ProcessorRegistry {
	return &ProcessorRegistry{
		processors: make([]Processor, 0),
	}
}

// Register adds a processor to the registry
func (pr *ProcessorRegistry) Register(processor Processor) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	// Insert in priority order (higher priority first)
	inserted := false
	for i, p := range pr.processors {
		if processor.Priority() > p.Priority() {
			pr.processors = append(pr.processors[:i], append([]Processor{processor}, pr.processors[i:]...)...)
			inserted = true
			break
		}
	}
	if !inserted {
		pr.processors = append(pr.processors, processor)
	}
}

// Unregister removes a processor
func (pr *ProcessorRegistry) Unregister(name string) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	for i, p := range pr.processors {
		if p.Name() == name {
			pr.processors = append(pr.processors[:i], pr.processors[i+1:]...)
			break
		}
	}
}

// GetProcessors returns all processors that can handle the event, in priority order
func (pr *ProcessorRegistry) GetProcessors(ctx context.Context, event *MediaEvent) []Processor {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	var result []Processor
	for _, p := range pr.processors {
		if p.CanHandle(ctx, event) {
			result = append(result, p)
		}
	}
	return result
}

// GetAllProcessors returns all registered processors
func (pr *ProcessorRegistry) GetAllProcessors() []Processor {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	result := make([]Processor, len(pr.processors))
	copy(result, pr.processors)
	return result
}

// BaseProcessor provides default implementation for common processor functionality
type BaseProcessor struct {
	name      string
	priority  ProcessorPriority
	condition ProcessorCondition
}

// NewBaseProcessor creates a base processor
func NewBaseProcessor(name string, priority ProcessorPriority) *BaseProcessor {
	return &BaseProcessor{
		name:     name,
		priority: priority,
	}
}

// WithCondition sets a condition for the processor
func (bp *BaseProcessor) WithCondition(condition ProcessorCondition) *BaseProcessor {
	bp.condition = condition
	return bp
}

// Name returns the processor name
func (bp *BaseProcessor) Name() string {
	return bp.name
}

// Priority returns the processor priority
func (bp *BaseProcessor) Priority() ProcessorPriority {
	return bp.priority
}

// CanHandle checks if the processor can handle the event
func (bp *BaseProcessor) CanHandle(ctx context.Context, event *MediaEvent) bool {
	if bp.condition != nil {
		return bp.condition(ctx, event)
	}
	return true
}

// FuncProcessor is a processor implemented as a function
type FuncProcessor struct {
	*BaseProcessor
	processFunc func(ctx context.Context, session *MediaSession, event *MediaEvent) error
}

// NewFuncProcessor creates a function-based processor
func NewFuncProcessor(name string, priority ProcessorPriority, fn func(ctx context.Context, session *MediaSession, event *MediaEvent) error) *FuncProcessor {
	return &FuncProcessor{
		BaseProcessor: NewBaseProcessor(name, priority),
		processFunc:   fn,
	}
}

// Process executes the processor function
func (fp *FuncProcessor) Process(ctx context.Context, session *MediaSession, event *MediaEvent) error {
	return fp.processFunc(ctx, session, event)
}

// PacketProcessor is a specialized processor for packet events
type PacketProcessor struct {
	*BaseProcessor
	processPacket func(ctx context.Context, session *MediaSession, packet MediaPacket) error
}

// NewPacketProcessor creates a packet processor
func NewPacketProcessor(name string, priority ProcessorPriority, fn func(ctx context.Context, session *MediaSession, packet MediaPacket) error) *PacketProcessor {
	return &PacketProcessor{
		BaseProcessor: NewBaseProcessor(name, priority),
		processPacket: fn,
	}
}

// CanHandle checks if this is a packet event
func (pp *PacketProcessor) CanHandle(ctx context.Context, event *MediaEvent) bool {
	if !pp.BaseProcessor.CanHandle(ctx, event) {
		return false
	}
	return event.Type == EventTypePacket
}

// Process handles packet events
func (pp *PacketProcessor) Process(ctx context.Context, session *MediaSession, event *MediaEvent) error {
	if packet, ok := event.Payload.(MediaPacket); ok {
		return pp.processPacket(ctx, session, packet)
	}
	return fmt.Errorf("event payload is not a MediaPacket")
}
