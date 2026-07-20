package memory

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/lingllm/protocol"
)

// WorkingMemory represents the working memory for the current round.
//
// L1: Working Memory (工作记忆，当前轮)
// Stores the current conversation context, reasoning chain, and temporary variables.
type WorkingMemory struct {
	// Messages in current round
	messages []protocol.Message

	// ReAct chain: Thought -> Action -> Observation
	thoughts     []string
	actions      []ToolAction
	observations []string

	// Temporary variables for current reasoning
	tempVars map[string]interface{}

	// Metadata
	roundID   string
	startTime time.Time
	maxSize   int // Max number of messages (default 20)

	// Context optimization
	compressionRatio float32 // How much to compress when overflow (default 0.5)
	priorityMessages []int   // Indices of high-priority messages to keep

	mu sync.RWMutex
}

// ToolAction represents an action taken by the agent.
type ToolAction struct {
	ToolName  string
	Input     map[string]interface{}
	Timestamp time.Time
}

// ToolObservation represents the result of a tool action.
type ToolObservation struct {
	ToolName  string
	Output    string
	Error     string
	Timestamp time.Time
}

// NewWorkingMemory creates a new working memory.
func NewWorkingMemory(roundID string) *WorkingMemory {
	return &WorkingMemory{
		messages:         make([]protocol.Message, 0, 20),
		thoughts:         make([]string, 0),
		actions:          make([]ToolAction, 0),
		observations:     make([]string, 0),
		tempVars:         make(map[string]interface{}),
		roundID:          roundID,
		startTime:        time.Now(),
		maxSize:          20,
		compressionRatio: 0.5,
		priorityMessages: make([]int, 0),
	}
}

// AddMessage adds a message to the working memory.
func (w *WorkingMemory) AddMessage(role protocol.MessageRole, content string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.addMessageUnsafe(role, content)
}

// addMessageUnsafe adds a message without locking (internal use only).
func (w *WorkingMemory) addMessageUnsafe(role protocol.MessageRole, content string) {
	// Keep only maxSize messages
	if len(w.messages) >= w.maxSize {
		w.messages = w.messages[1:]
	}
	w.messages = append(w.messages, protocol.Message{
		Role:    role,
		Content: content,
	})
}

// AddThought adds a thought to the ReAct chain.
func (w *WorkingMemory) AddThought(thought string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.thoughts = append(w.thoughts, thought)
	w.addMessageUnsafe(protocol.RoleAssistant, "Thought: "+thought)
}

// AddAction adds an action to the ReAct chain.
func (w *WorkingMemory) AddAction(toolName string, input map[string]interface{}) {
	w.mu.Lock()
	defer w.mu.Unlock()
	action := ToolAction{
		ToolName:  toolName,
		Input:     input,
		Timestamp: time.Now(),
	}
	w.actions = append(w.actions, action)
}

// AddObservation adds an observation to the ReAct chain.
func (w *WorkingMemory) AddObservation(observation string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.observations = append(w.observations, observation)
	w.addMessageUnsafe(protocol.RoleTool, observation)
}

// GetMessages returns all messages in the current round.
func (w *WorkingMemory) GetMessages() []protocol.Message {
	w.mu.RLock()
	defer w.mu.RUnlock()

	messages := make([]protocol.Message, len(w.messages))
	copy(messages, w.messages)
	return messages
}

// GetReActChain returns the complete ReAct chain.
func (w *WorkingMemory) GetReActChain() ReActChain {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return ReActChain{
		Thoughts:     w.thoughts,
		Actions:      w.actions,
		Observations: w.observations,
	}
}

// SetTempVar sets a temporary variable.
func (w *WorkingMemory) SetTempVar(key string, value interface{}) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.tempVars[key] = value
}

// GetTempVar gets a temporary variable.
func (w *WorkingMemory) GetTempVar(key string) interface{} {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.tempVars[key]
}

// GetAllTempVars returns all temporary variables.
func (w *WorkingMemory) GetAllTempVars() map[string]interface{} {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.tempVars
}

// Clear clears all data (called after round completion).
func (w *WorkingMemory) Clear() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.messages = make([]protocol.Message, 0, 20)
	w.thoughts = make([]string, 0)
	w.actions = make([]ToolAction, 0)
	w.observations = make([]string, 0)
	w.tempVars = make(map[string]interface{})
}

// GetStats returns statistics about the working memory.
func (w *WorkingMemory) GetStats() WorkingMemoryStats {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.getStatsUnsafe()
}

// getStatsUnsafe returns statistics without locking (internal use only).
func (w *WorkingMemory) getStatsUnsafe() WorkingMemoryStats {
	return WorkingMemoryStats{
		RoundID:          w.roundID,
		MessageCount:     len(w.messages),
		ThoughtCount:     len(w.thoughts),
		ActionCount:      len(w.actions),
		ObservationCount: len(w.observations),
		TempVarCount:     len(w.tempVars),
		Duration:         time.Since(w.startTime),
		StartTime:        w.startTime,
	}
}

// ReActChain represents the complete ReAct chain.
type ReActChain struct {
	Thoughts     []string
	Actions      []ToolAction
	Observations []string
}

// WorkingMemoryStats represents statistics about working memory.
type WorkingMemoryStats struct {
	RoundID          string
	MessageCount     int
	ThoughtCount     int
	ActionCount      int
	ObservationCount int
	TempVarCount     int
	Duration         time.Duration
	StartTime        time.Time
}

// WorkingMemoryContext represents the complete context for the current round.
type WorkingMemoryContext struct {
	RoundID    string
	Messages   []protocol.Message
	ReActChain ReActChain
	TempVars   map[string]interface{}
	Stats      WorkingMemoryStats
}

// GetContext returns the complete working memory context.
func (w *WorkingMemory) GetContext() WorkingMemoryContext {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return WorkingMemoryContext{
		RoundID:  w.roundID,
		Messages: w.messages,
		ReActChain: ReActChain{
			Thoughts:     w.thoughts,
			Actions:      w.actions,
			Observations: w.observations,
		},
		TempVars: w.tempVars,
		Stats:    w.getStatsUnsafe(),
	}
}

// ToPrompt converts working memory to a formatted prompt string.
func (w *WorkingMemory) ToPrompt() string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var prompt strings.Builder

	// Add ReAct chain
	if len(w.thoughts) > 0 || len(w.actions) > 0 || len(w.observations) > 0 {
		prompt.WriteString("=== Reasoning Chain ===\n")

		for i, thought := range w.thoughts {
			prompt.WriteString(fmt.Sprintf("Thought %d: %s\n", i+1, thought))
		}

		for i, action := range w.actions {
			prompt.WriteString(fmt.Sprintf("Action %d: %s(%v)\n", i+1, action.ToolName, action.Input))
		}

		for i, obs := range w.observations {
			prompt.WriteString(fmt.Sprintf("Observation %d: %s\n", i+1, obs))
		}

		prompt.WriteString("\n")
	}

	// Add messages
	if len(w.messages) > 0 {
		prompt.WriteString("=== Conversation History ===\n")
		for _, msg := range w.messages {
			prompt.WriteString(fmt.Sprintf("[%s]: %s\n", msg.Role, msg.Content))
		}
		prompt.WriteString("\n")
	}

	// Add temp vars
	if len(w.tempVars) > 0 {
		prompt.WriteString("=== Context Variables ===\n")
		for key, value := range w.tempVars {
			prompt.WriteString(fmt.Sprintf("%s: %v\n", key, value))
		}
	}

	return prompt.String()
}

// OptimizeContext optimizes the context when it exceeds capacity.
// Strategy: Keep system/user messages, compress middle messages, keep recent messages.
func (w *WorkingMemory) OptimizeContext() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(w.messages) <= w.maxSize {
		return
	}

	// Strategy: Keep first (system), last N, and compress middle
	optimized := make([]protocol.Message, 0, w.maxSize)

	// Keep system message if exists
	if len(w.messages) > 0 && w.messages[0].Role == protocol.RoleSystem {
		optimized = append(optimized, w.messages[0])
	}

	// Calculate how many recent messages to keep
	recentCount := int(float32(w.maxSize) * w.compressionRatio)
	if recentCount < 3 {
		recentCount = 3
	}

	// Add recent messages
	startIdx := len(w.messages) - recentCount
	if startIdx < 1 {
		startIdx = 1
	}

	// Add a summary message if we're compressing
	if startIdx > 1 {
		summaryCount := startIdx - 1
		if len(optimized) > 0 {
			optimized = append(optimized, protocol.Message{
				Role:    protocol.RoleSystem,
				Content: fmt.Sprintf("[Context compressed: %d messages summarized]", summaryCount),
			})
		}
	}

	// Add recent messages
	optimized = append(optimized, w.messages[startIdx:]...)

	w.messages = optimized
}

// MarkMessageAsPriority marks a message index as high-priority (won't be compressed).
func (w *WorkingMemory) MarkMessageAsPriority(index int) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if index >= 0 && index < len(w.messages) {
		w.priorityMessages = append(w.priorityMessages, index)
	}
}

// SetCompressionRatio sets the compression ratio (0.0-1.0) for context optimization.
func (w *WorkingMemory) SetCompressionRatio(ratio float32) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if ratio > 0 && ratio <= 1.0 {
		w.compressionRatio = ratio
	}
}

// SetMaxSize sets the maximum number of messages to keep.
func (w *WorkingMemory) SetMaxSize(size int) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if size > 0 {
		w.maxSize = size
	}
}

// GetRoundID returns the round ID.
func (w *WorkingMemory) GetRoundID() string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.roundID
}

// GetDuration returns the duration since the round started.
func (w *WorkingMemory) GetDuration() time.Duration {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return time.Since(w.startTime)
}
