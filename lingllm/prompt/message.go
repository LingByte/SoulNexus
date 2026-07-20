package prompt

import (
	"fmt"

	"github.com/LingByte/lingllm/protocol"
)

// Role represents the role in a conversation.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// MessageTemplate represents a template for a single message.
type MessageTemplate struct {
	Role       Role
	Template   *Template
	Metadata   map[string]any
	Conditions []MessageCondition
}

// MessageCondition defines when a message should be included.
type MessageCondition struct {
	Field    string
	Operator string // "eq", "ne", "gt", "lt", "exists", "not_exists"
	Value    any
}

// ConditionMet checks if the condition is met.
func (c *MessageCondition) ConditionMet(data map[string]any) bool {
	val := getNestedValue(data, c.Field)
	switch c.Operator {
	case "exists":
		return val != nil
	case "not_exists":
		return val == nil
	case "eq":
		return fmt.Sprintf("%v", val) == fmt.Sprintf("%v", c.Value)
	case "ne":
		return fmt.Sprintf("%v", val) != fmt.Sprintf("%v", c.Value)
	case "gt":
		if n1, ok := val.(int); ok {
			if n2, ok := c.Value.(int); ok {
				return n1 > n2
			}
		}
	case "lt":
		if n1, ok := val.(int); ok {
			if n2, ok := c.Value.(int); ok {
				return n1 < n2
			}
		}
	}
	return false
}

// NewMessageTemplate creates a new message template.
func NewMessageTemplate(role Role, template *Template) *MessageTemplate {
	return &MessageTemplate{
		Role:       role,
		Template:   template,
		Metadata:   make(map[string]any),
		Conditions: make([]MessageCondition, 0),
	}
}

// WithMetadata adds metadata to the message template.
func (mt *MessageTemplate) WithMetadata(key string, value any) *MessageTemplate {
	if mt.Metadata == nil {
		mt.Metadata = make(map[string]any)
	}
	mt.Metadata[key] = value
	return mt
}

// WithCondition adds a condition for when this message should be included.
func (mt *MessageTemplate) WithCondition(field, operator string, value any) *MessageTemplate {
	mt.Conditions = append(mt.Conditions, MessageCondition{
		Field:    field,
		Operator: operator,
		Value:    value,
	})
	return mt
}

// ShouldInclude checks if this message template should be included.
func (mt *MessageTemplate) ShouldInclude(data map[string]any) bool {
	for _, cond := range mt.Conditions {
		if !cond.ConditionMet(data) {
			return false
		}
	}
	return true
}

// Render renders the message template to a protocol.Message.
func (mt *MessageTemplate) Render(data map[string]any) (*protocol.Message, error) {
	if !mt.ShouldInclude(data) {
		return nil, nil
	}

	content, err := mt.Template.Render(data)
	if err != nil {
		return nil, fmt.Errorf("render message template: %w", err)
	}

	return &protocol.Message{
		Role:    protocol.MessageRole(mt.Role),
		Content: content,
	}, nil
}

// SystemMessage creates a system message template.
func SystemMessage(template *Template) *MessageTemplate {
	return NewMessageTemplate(RoleSystem, template)
}

// UserMessage creates a user message template.
func UserMessage(template *Template) *MessageTemplate {
	return NewMessageTemplate(RoleUser, template)
}

// AssistantMessage creates an assistant message template.
func AssistantMessage(template *Template) *MessageTemplate {
	return NewMessageTemplate(RoleAssistant, template)
}

// Conversation represents a complete conversation with multiple message templates.
type Conversation struct {
	name        string
	messages    []*MessageTemplate
	strictOrder bool
	metadata    map[string]any
}

// NewConversation creates a new conversation.
func NewConversation(name string) *Conversation {
	return &Conversation{
		name:     name,
		messages: make([]*MessageTemplate, 0),
		metadata: make(map[string]any),
	}
}

// Name returns the conversation name.
func (c *Conversation) Name() string { return c.name }

// AddMessage adds a message template to the conversation.
func (c *Conversation) AddMessage(msg *MessageTemplate) *Conversation {
	c.messages = append(c.messages, msg)
	return c
}

// AddSystem adds a system message.
func (c *Conversation) AddSystem(template *Template) *Conversation {
	return c.AddMessage(SystemMessage(template))
}

// AddUser adds a user message.
func (c *Conversation) AddUser(template *Template) *Conversation {
	return c.AddMessage(UserMessage(template))
}

// AddAssistant adds an assistant message.
func (c *Conversation) AddAssistant(template *Template) *Conversation {
	return c.AddMessage(AssistantMessage(template))
}

// AddConditional adds a message with a condition.
func (c *Conversation) AddConditional(msg *MessageTemplate) *Conversation {
	return c.AddMessage(msg)
}

// WithMetadata adds conversation metadata.
func (c *Conversation) WithMetadata(key string, value any) *Conversation {
	c.metadata[key] = value
	return c
}

// WithStrictOrder enforces strict message ordering.
func (c *Conversation) WithStrictOrder() *Conversation {
	c.strictOrder = true
	return c
}

// Render renders all messages in the conversation.
func (c *Conversation) Render(data map[string]any) ([]protocol.Message, error) {
	var messages []protocol.Message
	for i, msg := range c.messages {
		if !msg.ShouldInclude(data) {
			continue
		}

		content, err := msg.Template.Render(data)
		if err != nil {
			return nil, fmt.Errorf("message %d render error: %w", i, err)
		}

		messages = append(messages, protocol.Message{
			Role:    protocol.MessageRole(msg.Role),
			Content: content,
		})
	}
	return messages, nil
}

// ToChatRequest converts the conversation to a ChatRequest.
func (c *Conversation) ToChatRequest(model string, data map[string]any) (*protocol.ChatRequest, error) {
	messages, err := c.Render(data)
	if err != nil {
		return nil, err
	}

	return &protocol.ChatRequest{
		Model:    model,
		Messages: messages,
	}, nil
}
