package memory

import (
	"strings"
	"testing"
	"time"

	"github.com/LingByte/lingllm/protocol"
)

func TestNewWorkingMemory(t *testing.T) {
	wm := NewWorkingMemory("round-1")

	if wm.GetRoundID() != "round-1" {
		t.Errorf("expected round-1, got %s", wm.GetRoundID())
	}

	if len(wm.GetMessages()) != 0 {
		t.Errorf("expected 0 messages, got %d", len(wm.GetMessages()))
	}
}

func TestAddMessage(t *testing.T) {
	wm := NewWorkingMemory("round-1")

	wm.AddMessage(protocol.RoleUser, "Hello")
	wm.AddMessage(protocol.RoleAssistant, "Hi there")

	messages := wm.GetMessages()
	if len(messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(messages))
	}

	if messages[0].Content != "Hello" {
		t.Errorf("expected 'Hello', got '%s'", messages[0].Content)
	}

	if messages[1].Content != "Hi there" {
		t.Errorf("expected 'Hi there', got '%s'", messages[1].Content)
	}
}

func TestAddThought(t *testing.T) {
	wm := NewWorkingMemory("round-1")

	wm.AddThought("I need to think about this")

	chain := wm.GetReActChain()
	if len(chain.Thoughts) != 1 {
		t.Errorf("expected 1 thought, got %d", len(chain.Thoughts))
	}

	if chain.Thoughts[0] != "I need to think about this" {
		t.Errorf("expected 'I need to think about this', got '%s'", chain.Thoughts[0])
	}

	// Thought should also be added as a message
	messages := wm.GetMessages()
	if len(messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(messages))
	}
}

func TestAddAction(t *testing.T) {
	wm := NewWorkingMemory("round-1")

	input := map[string]interface{}{"query": "test"}
	wm.AddAction("search", input)

	chain := wm.GetReActChain()
	if len(chain.Actions) != 1 {
		t.Errorf("expected 1 action, got %d", len(chain.Actions))
	}

	if chain.Actions[0].ToolName != "search" {
		t.Errorf("expected 'search', got '%s'", chain.Actions[0].ToolName)
	}
}

func TestAddObservation(t *testing.T) {
	wm := NewWorkingMemory("round-1")

	wm.AddObservation("Found 3 results")

	chain := wm.GetReActChain()
	if len(chain.Observations) != 1 {
		t.Errorf("expected 1 observation, got %d", len(chain.Observations))
	}

	if chain.Observations[0] != "Found 3 results" {
		t.Errorf("expected 'Found 3 results', got '%s'", chain.Observations[0])
	}

	// Observation should also be added as a message
	messages := wm.GetMessages()
	if len(messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(messages))
	}
}

func TestTempVars(t *testing.T) {
	wm := NewWorkingMemory("round-1")

	wm.SetTempVar("key1", "value1")
	wm.SetTempVar("key2", 42)

	if wm.GetTempVar("key1") != "value1" {
		t.Errorf("expected 'value1', got %v", wm.GetTempVar("key1"))
	}

	if wm.GetTempVar("key2") != 42 {
		t.Errorf("expected 42, got %v", wm.GetTempVar("key2"))
	}

	vars := wm.GetAllTempVars()
	if len(vars) != 2 {
		t.Errorf("expected 2 vars, got %d", len(vars))
	}
}

func TestGetStats(t *testing.T) {
	wm := NewWorkingMemory("round-1")

	wm.AddMessage(protocol.RoleUser, "Hello")
	wm.AddThought("Thinking...")
	wm.AddAction("search", map[string]interface{}{})
	wm.AddObservation("Result")
	wm.SetTempVar("key", "value")

	stats := wm.GetStats()

	if stats.RoundID != "round-1" {
		t.Errorf("expected 'round-1', got '%s'", stats.RoundID)
	}

	// AddThought and AddObservation also add messages, so we have 3 total
	if stats.MessageCount != 3 {
		t.Errorf("expected 3 messages, got %d", stats.MessageCount)
	}

	if stats.ThoughtCount != 1 {
		t.Errorf("expected 1 thought, got %d", stats.ThoughtCount)
	}

	if stats.ActionCount != 1 {
		t.Errorf("expected 1 action, got %d", stats.ActionCount)
	}

	if stats.ObservationCount != 1 {
		t.Errorf("expected 1 observation, got %d", stats.ObservationCount)
	}

	if stats.TempVarCount != 1 {
		t.Errorf("expected 1 temp var, got %d", stats.TempVarCount)
	}
}

func TestGetContext(t *testing.T) {
	wm := NewWorkingMemory("round-1")

	wm.AddMessage(protocol.RoleUser, "Hello")
	wm.AddThought("Thinking")
	wm.AddAction("search", map[string]interface{}{})
	wm.AddObservation("Result")
	wm.SetTempVar("key", "value")

	ctx := wm.GetContext()

	if ctx.RoundID != "round-1" {
		t.Errorf("expected 'round-1', got '%s'", ctx.RoundID)
	}

	// AddThought and AddObservation also add messages, so we have 3 total
	if len(ctx.Messages) != 3 {
		t.Errorf("expected 3 messages, got %d", len(ctx.Messages))
	}

	if len(ctx.ReActChain.Thoughts) != 1 {
		t.Errorf("expected 1 thought, got %d", len(ctx.ReActChain.Thoughts))
	}

	if len(ctx.TempVars) != 1 {
		t.Errorf("expected 1 temp var, got %d", len(ctx.TempVars))
	}
}

func TestClear(t *testing.T) {
	wm := NewWorkingMemory("round-1")

	wm.AddMessage(protocol.RoleUser, "Hello")
	wm.AddThought("Thinking")
	wm.SetTempVar("key", "value")

	wm.Clear()

	if len(wm.GetMessages()) != 0 {
		t.Errorf("expected 0 messages after clear, got %d", len(wm.GetMessages()))
	}

	chain := wm.GetReActChain()
	if len(chain.Thoughts) != 0 {
		t.Errorf("expected 0 thoughts after clear, got %d", len(chain.Thoughts))
	}

	if len(wm.GetAllTempVars()) != 0 {
		t.Errorf("expected 0 temp vars after clear, got %d", len(wm.GetAllTempVars()))
	}
}

func TestMaxSize(t *testing.T) {
	wm := NewWorkingMemory("round-1")
	wm.SetMaxSize(3)

	wm.AddMessage(protocol.RoleUser, "msg1")
	wm.AddMessage(protocol.RoleUser, "msg2")
	wm.AddMessage(protocol.RoleUser, "msg3")
	wm.AddMessage(protocol.RoleUser, "msg4")

	messages := wm.GetMessages()
	if len(messages) != 3 {
		t.Errorf("expected 3 messages (max size), got %d", len(messages))
	}

	if messages[0].Content != "msg2" {
		t.Errorf("expected 'msg2' (oldest kept), got '%s'", messages[0].Content)
	}

	if messages[2].Content != "msg4" {
		t.Errorf("expected 'msg4' (newest), got '%s'", messages[2].Content)
	}
}

func TestOptimizeContext(t *testing.T) {
	wm := NewWorkingMemory("round-1")
	wm.SetMaxSize(5)
	wm.SetCompressionRatio(0.5)

	// Add system message
	wm.AddMessage(protocol.RoleSystem, "You are a helpful assistant")

	// Add many messages to exceed max size
	for i := 1; i <= 10; i++ {
		wm.AddMessage(protocol.RoleUser, "msg"+string(rune(48+i)))
	}

	wm.OptimizeContext()

	messages := wm.GetMessages()
	if len(messages) > wm.maxSize {
		t.Errorf("expected max %d messages after optimization, got %d", wm.maxSize, len(messages))
	}

	// After optimization, should have some messages
	if len(messages) == 0 {
		t.Errorf("expected messages after optimization")
	}
}

func TestToPrompt(t *testing.T) {
	wm := NewWorkingMemory("round-1")

	wm.AddMessage(protocol.RoleUser, "Hello")
	wm.AddThought("Let me think")
	wm.AddAction("search", map[string]interface{}{"q": "test"})
	wm.AddObservation("Found results")
	wm.SetTempVar("context", "important")

	prompt := wm.ToPrompt()

	if !strings.Contains(prompt, "Reasoning Chain") {
		t.Errorf("expected 'Reasoning Chain' in prompt")
	}

	if !strings.Contains(prompt, "Conversation History") {
		t.Errorf("expected 'Conversation History' in prompt")
	}

	if !strings.Contains(prompt, "Context Variables") {
		t.Errorf("expected 'Context Variables' in prompt")
	}

	if !strings.Contains(prompt, "Let me think") {
		t.Errorf("expected thought in prompt")
	}

	if !strings.Contains(prompt, "search") {
		t.Errorf("expected action in prompt")
	}

	if !strings.Contains(prompt, "Found results") {
		t.Errorf("expected observation in prompt")
	}
}

func TestConcurrency(t *testing.T) {
	wm := NewWorkingMemory("round-1")

	// Add messages concurrently
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(id int) {
			wm.AddMessage(protocol.RoleUser, "msg")
			wm.SetTempVar("key", id)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic and should have some messages
	messages := wm.GetMessages()
	if len(messages) == 0 {
		t.Errorf("expected messages after concurrent adds")
	}
}

func TestDuration(t *testing.T) {
	wm := NewWorkingMemory("round-1")

	time.Sleep(100 * time.Millisecond)

	duration := wm.GetDuration()
	if duration < 100*time.Millisecond {
		t.Errorf("expected duration >= 100ms, got %v", duration)
	}
}
