package providers

import (
	"context"

	"github.com/LingByte/lingllm/protocol"
)

// usageAccumulatingModel sums token usage across multiple Chat rounds (tool calling).
type usageAccumulatingModel struct {
	inner protocol.ChatModel
	usage protocol.TokenUsage
}

func wrapUsageAccumulatingModel(inner protocol.ChatModel) *usageAccumulatingModel {
	if inner == nil {
		return nil
	}
	return &usageAccumulatingModel{inner: inner}
}

func (m *usageAccumulatingModel) Name() string {
	if m == nil || m.inner == nil {
		return ""
	}
	return m.inner.Name()
}

func (m *usageAccumulatingModel) Chat(ctx context.Context, req protocol.ChatRequest) (*protocol.ChatResponse, error) {
	resp, err := m.inner.Chat(ctx, req)
	if err == nil && resp != nil {
		m.add(resp.Usage)
	}
	return resp, err
}

func (m *usageAccumulatingModel) StreamChat(ctx context.Context, req protocol.ChatRequest) (protocol.ChatStream, error) {
	return m.inner.StreamChat(ctx, req)
}

func (m *usageAccumulatingModel) add(u protocol.TokenUsage) {
	if u.TotalTokens <= 0 && u.PromptTokens <= 0 && u.CompletionTokens <= 0 {
		return
	}
	m.usage.PromptTokens += u.PromptTokens
	m.usage.CompletionTokens += u.CompletionTokens
	if u.TotalTokens > 0 {
		m.usage.TotalTokens += u.TotalTokens
	} else {
		m.usage.TotalTokens += u.PromptTokens + u.CompletionTokens
	}
}

func (m *usageAccumulatingModel) accumulatedUsage() protocol.TokenUsage {
	if m == nil {
		return protocol.TokenUsage{}
	}
	return m.usage
}
