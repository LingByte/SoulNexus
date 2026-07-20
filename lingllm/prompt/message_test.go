package prompt

import (
	"testing"

	"github.com/LingByte/lingllm/protocol"
)

func TestRole(t *testing.T) {
	t.Run("role constants", func(t *testing.T) {
		if RoleSystem != "system" {
			t.Errorf("expected RoleSystem to be 'system', got %s", RoleSystem)
		}
		if RoleUser != "user" {
			t.Errorf("expected RoleUser to be 'user', got %s", RoleUser)
		}
		if RoleAssistant != "assistant" {
			t.Errorf("expected RoleAssistant to be 'assistant', got %s", RoleAssistant)
		}
		if RoleTool != "tool" {
			t.Errorf("expected RoleTool to be 'tool', got %s", RoleTool)
		}
	})
}

func TestMessageCondition(t *testing.T) {
	t.Run("exists condition met", func(t *testing.T) {
		cond := MessageCondition{
			Field:    "user.name",
			Operator: "exists",
		}
		data := map[string]any{
			"user": map[string]any{"name": "Alice"},
		}
		if !cond.ConditionMet(data) {
			t.Error("expected condition to be met")
		}
	})

	t.Run("exists condition not met", func(t *testing.T) {
		cond := MessageCondition{
			Field:    "user.name",
			Operator: "exists",
		}
		data := map[string]any{
			"user": map[string]any{},
		}
		if cond.ConditionMet(data) {
			t.Error("expected condition not to be met")
		}
	})

	t.Run("not_exists condition met", func(t *testing.T) {
		cond := MessageCondition{
			Field:    "user.name",
			Operator: "not_exists",
		}
		data := map[string]any{
			"user": map[string]any{},
		}
		if !cond.ConditionMet(data) {
			t.Error("expected condition to be met")
		}
	})

	t.Run("eq condition met", func(t *testing.T) {
		cond := MessageCondition{
			Field:    "user.isAdmin",
			Operator: "eq",
			Value:    true,
		}
		data := map[string]any{
			"user": map[string]any{"isAdmin": true},
		}
		if !cond.ConditionMet(data) {
			t.Error("expected condition to be met")
		}
	})

	t.Run("eq condition not met", func(t *testing.T) {
		cond := MessageCondition{
			Field:    "user.isAdmin",
			Operator: "eq",
			Value:    true,
		}
		data := map[string]any{
			"user": map[string]any{"isAdmin": false},
		}
		if cond.ConditionMet(data) {
			t.Error("expected condition not to be met")
		}
	})

	t.Run("ne condition met", func(t *testing.T) {
		cond := MessageCondition{
			Field:    "user.role",
			Operator: "ne",
			Value:    "admin",
		}
		data := map[string]any{
			"user": map[string]any{"role": "user"},
		}
		if !cond.ConditionMet(data) {
			t.Error("expected condition to be met")
		}
	})

	t.Run("gt condition met", func(t *testing.T) {
		cond := MessageCondition{
			Field:    "count",
			Operator: "gt",
			Value:    5,
		}
		data := map[string]any{"count": 10}
		if !cond.ConditionMet(data) {
			t.Error("expected condition to be met")
		}
	})

	t.Run("gt condition not met", func(t *testing.T) {
		cond := MessageCondition{
			Field:    "count",
			Operator: "gt",
			Value:    5,
		}
		data := map[string]any{"count": 3}
		if cond.ConditionMet(data) {
			t.Error("expected condition not to be met")
		}
	})

	t.Run("lt condition met", func(t *testing.T) {
		cond := MessageCondition{
			Field:    "count",
			Operator: "lt",
			Value:    10,
		}
		data := map[string]any{"count": 5}
		if !cond.ConditionMet(data) {
			t.Error("expected condition to be met")
		}
	})

	t.Run("lt condition not met", func(t *testing.T) {
		cond := MessageCondition{
			Field:    "count",
			Operator: "lt",
			Value:    10,
		}
		data := map[string]any{"count": 15}
		if cond.ConditionMet(data) {
			t.Error("expected condition not to be met")
		}
	})
}

func TestMessageTemplate(t *testing.T) {
	t.Run("basic creation", func(t *testing.T) {
		tpl := MustNewTemplate("test", "Hello, {{name}}")
		mt := NewMessageTemplate(RoleUser, tpl)

		if mt.Role != RoleUser {
			t.Errorf("expected RoleUser, got %s", mt.Role)
		}
		if mt.Template != tpl {
			t.Error("expected template to be set")
		}
		if mt.Metadata == nil {
			t.Error("expected non-nil metadata")
		}
	})

	t.Run("with metadata", func(t *testing.T) {
		tpl := MustNewTemplate("test", "Hello")
		mt := NewMessageTemplate(RoleUser, tpl)
		result := mt.WithMetadata("key", "value")

		if result != mt {
			t.Error("expected fluent interface")
		}
		if mt.Metadata["key"] != "value" {
			t.Error("expected metadata to be set")
		}
	})

	t.Run("with condition", func(t *testing.T) {
		tpl := MustNewTemplate("test", "Hello")
		mt := NewMessageTemplate(RoleUser, tpl)
		result := mt.WithCondition("user.isAdmin", "eq", true)

		if result != mt {
			t.Error("expected fluent interface")
		}
		if len(mt.Conditions) != 1 {
			t.Errorf("expected 1 condition, got %d", len(mt.Conditions))
		}
	})

	t.Run("should include without conditions", func(t *testing.T) {
		tpl := MustNewTemplate("test", "Hello")
		mt := NewMessageTemplate(RoleUser, tpl)

		if !mt.ShouldInclude(map[string]any{}) {
			t.Error("expected message to be included when no conditions")
		}
	})

	t.Run("should include with met condition", func(t *testing.T) {
		tpl := MustNewTemplate("test", "Admin Panel")
		mt := NewMessageTemplate(RoleUser, tpl)
		mt.WithCondition("user.isAdmin", "eq", true)

		data := map[string]any{
			"user": map[string]any{"isAdmin": true},
		}
		if !mt.ShouldInclude(data) {
			t.Error("expected message to be included")
		}
	})

	t.Run("should not include with unmet condition", func(t *testing.T) {
		tpl := MustNewTemplate("test", "Admin Panel")
		mt := NewMessageTemplate(RoleUser, tpl)
		mt.WithCondition("user.isAdmin", "eq", true)

		data := map[string]any{
			"user": map[string]any{"isAdmin": false},
		}
		if mt.ShouldInclude(data) {
			t.Error("expected message not to be included")
		}
	})

	t.Run("render message", func(t *testing.T) {
		tpl := MustNewTemplate("test", "Hello, {{name}}!")
		mt := NewMessageTemplate(RoleUser, tpl)

		msg, err := mt.Render(map[string]any{"name": "Alice"})
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}
		if msg == nil {
			t.Fatal("expected non-nil message")
		}
		if msg.Content != "Hello, Alice!" {
			t.Errorf("unexpected content: %s", msg.Content)
		}
		if msg.Role != protocol.RoleUser {
			t.Errorf("unexpected role: %s", msg.Role)
		}
	})

	t.Run("render returns nil when excluded", func(t *testing.T) {
		tpl := MustNewTemplate("test", "Admin Panel")
		mt := NewMessageTemplate(RoleUser, tpl)
		mt.WithCondition("user.isAdmin", "eq", true)

		msg, err := mt.Render(map[string]any{
			"user": map[string]any{"isAdmin": false},
		})
		if err != nil {
			t.Fatalf("Render should not error: %v", err)
		}
		if msg != nil {
			t.Error("expected nil message when condition not met")
		}
	})

	t.Run("render with complex data", func(t *testing.T) {
		tpl := MustNewTemplate("test", "User: {{user.name}}, Role: {{user.role}}")
		mt := NewMessageTemplate(RoleUser, tpl)

		data := map[string]any{
			"user": map[string]any{
				"name": "Alice",
				"role": "admin",
			},
		}

		msg, err := mt.Render(data)
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}
		if msg.Content != "User: Alice, Role: admin" {
			t.Errorf("unexpected content: %s", msg.Content)
		}
	})

	t.Run("render with array data", func(t *testing.T) {
		tpl := MustNewTemplate("test", "Items: {{items[0]}}, {{items[1]}}")
		mt := NewMessageTemplate(RoleUser, tpl)

		data := map[string]any{
			"items": []any{"first", "second"},
		}

		msg, err := mt.Render(data)
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}
		if msg.Content != "Items: first, second" {
			t.Errorf("unexpected content: %s", msg.Content)
		}
	})

	t.Run("render with multiple conditions", func(t *testing.T) {
		tpl := MustNewTemplate("test", "Premium content")
		mt := NewMessageTemplate(RoleUser, tpl)
		mt.WithCondition("user.isPremium", "eq", true)
		mt.WithCondition("user.isActive", "eq", true)

		// Both conditions met
		msg, err := mt.Render(map[string]any{
			"user": map[string]any{"isPremium": true, "isActive": true},
		})
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}
		if msg == nil {
			t.Error("expected message when both conditions met")
		}

		// One condition not met
		msg, err = mt.Render(map[string]any{
			"user": map[string]any{"isPremium": true, "isActive": false},
		})
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}
		if msg != nil {
			t.Error("expected nil when one condition not met")
		}
	})
}

func TestMessageHelpers(t *testing.T) {
	t.Run("system message", func(t *testing.T) {
		tpl := MustNewTemplate("test", "You are helpful")
		mt := SystemMessage(tpl)

		if mt.Role != RoleSystem {
			t.Errorf("expected RoleSystem, got %s", mt.Role)
		}
	})

	t.Run("user message", func(t *testing.T) {
		tpl := MustNewTemplate("test", "Hello")
		mt := UserMessage(tpl)

		if mt.Role != RoleUser {
			t.Errorf("expected RoleUser, got %s", mt.Role)
		}
	})

	t.Run("assistant message", func(t *testing.T) {
		tpl := MustNewTemplate("test", "Hello")
		mt := AssistantMessage(tpl)

		if mt.Role != RoleAssistant {
			t.Errorf("expected RoleAssistant, got %s", mt.Role)
		}
	})
}

func TestConversation(t *testing.T) {
	t.Run("basic creation", func(t *testing.T) {
		conv := NewConversation("test")
		if conv == nil {
			t.Fatal("expected non-nil conversation")
		}
		if conv.Name() != "test" {
			t.Errorf("expected name 'test', got %s", conv.Name())
		}
	})

	t.Run("add message", func(t *testing.T) {
		conv := NewConversation("test")
		tpl := MustNewTemplate("test", "Hello")
		msg := NewMessageTemplate(RoleUser, tpl)
		result := conv.AddMessage(msg)

		if result != conv {
			t.Error("expected fluent interface")
		}
	})

	t.Run("add system", func(t *testing.T) {
		conv := NewConversation("test")
		tpl := MustNewTemplate("sys", "You are helpful")
		result := conv.AddSystem(tpl)

		if result != conv {
			t.Error("expected fluent interface")
		}
	})

	t.Run("add user", func(t *testing.T) {
		conv := NewConversation("test")
		tpl := MustNewTemplate("user", "{{question}}")
		result := conv.AddUser(tpl)

		if result != conv {
			t.Error("expected fluent interface")
		}
	})

	t.Run("add assistant", func(t *testing.T) {
		conv := NewConversation("test")
		tpl := MustNewTemplate("assistant", "I am helpful")
		result := conv.AddAssistant(tpl)

		if result != conv {
			t.Error("expected fluent interface")
		}
	})

	t.Run("add conditional", func(t *testing.T) {
		conv := NewConversation("test")
		tpl := MustNewTemplate("admin", "Admin Panel")
		msg := NewMessageTemplate(RoleUser, tpl)
		msg.WithCondition("user.isAdmin", "eq", true)
		result := conv.AddConditional(msg)

		if result != conv {
			t.Error("expected fluent interface")
		}
	})

	t.Run("with metadata", func(t *testing.T) {
		conv := NewConversation("test")
		result := conv.WithMetadata("key", "value")

		if result != conv {
			t.Error("expected fluent interface")
		}
		if conv.metadata["key"] != "value" {
			t.Error("expected metadata to be set")
		}
	})

	t.Run("with strict order", func(t *testing.T) {
		conv := NewConversation("test")
		result := conv.WithStrictOrder()

		if result != conv {
			t.Error("expected fluent interface")
		}
		if !conv.strictOrder {
			t.Error("expected strictOrder to be true")
		}
	})

	t.Run("render empty conversation", func(t *testing.T) {
		conv := NewConversation("test")
		messages, err := conv.Render(map[string]any{})
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}
		if len(messages) != 0 {
			t.Errorf("expected 0 messages, got %d", len(messages))
		}
	})

	t.Run("render with messages", func(t *testing.T) {
		conv := NewConversation("test")
		conv.AddSystem(MustNewTemplate("sys", "You are helpful"))
		conv.AddUser(MustNewTemplate("user", "What is {{topic}}?"))
		conv.AddAssistant(MustNewTemplate("assistant", "I can help with {{topic}}"))

		messages, err := conv.Render(map[string]any{"topic": "Go"})
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}
		if len(messages) != 3 {
			t.Errorf("expected 3 messages, got %d", len(messages))
		}
	})

	t.Run("render filters conditional messages", func(t *testing.T) {
		conv := NewConversation("test")
		conv.AddUser(MustNewTemplate("user", "Regular message"))
		conv.AddConditional(NewMessageTemplate(RoleUser, MustNewTemplate("admin", "Admin Panel")).
			WithCondition("user.isAdmin", "eq", true))

		// Without condition met
		messages, err := conv.Render(map[string]any{})
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}
		if len(messages) != 1 {
			t.Errorf("expected 1 message, got %d", len(messages))
		}

		// With condition met
		messages, err = conv.Render(map[string]any{
			"user": map[string]any{"isAdmin": true},
		})
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}
		if len(messages) != 2 {
			t.Errorf("expected 2 messages when condition met, got %d", len(messages))
		}
	})

	t.Run("to chat request", func(t *testing.T) {
		conv := NewConversation("test")
		conv.AddUser(MustNewTemplate("user", "Hello {{name}}"))

		req, err := conv.ToChatRequest("claude", map[string]any{"name": "Alice"})
		if err != nil {
			t.Fatalf("ToChatRequest failed: %v", err)
		}
		if req.Model != "claude" {
			t.Errorf("expected model 'claude', got %s", req.Model)
		}
		if len(req.Messages) != 1 {
			t.Errorf("expected 1 message, got %d", len(req.Messages))
		}
	})
}
