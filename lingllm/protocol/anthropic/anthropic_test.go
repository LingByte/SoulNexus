package anthropic

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/LingByte/lingllm/protocol"
)

func TestNewClientRequiresAPIKey(t *testing.T) {
	_, err := NewClient(Config{})
	if err == nil {
		t.Fatal("expected error without api key")
	}
}

func TestNewClientDefaults(t *testing.T) {
	c, err := NewClient(Config{APIKey: "key"})
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	if c.cfg.BaseURL != defaultBaseURL {
		t.Errorf("unexpected base URL: %s", c.cfg.BaseURL)
	}
}

func TestChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "anthropic-key" {
			t.Errorf("unexpected api key header")
		}
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), `"stream":true`) {
			t.Errorf("chat request must not enable streaming: %s", body)
		}
		fmt.Fprint(w, `{
			"id":"msg_1",
			"model":"claude",
			"role":"assistant",
			"content":[{"type":"text","text":"Hello from Claude"}],
			"stop_reason":"end_turn",
			"usage":{"input_tokens":10,"output_tokens":5}
		}`)
	}))
	defer server.Close()

	client, _ := NewClient(Config{APIKey: "anthropic-key", BaseURL: server.URL})
	resp, err := client.Chat(context.Background(), protocol.ChatRequest{
		Model:    "claude",
		Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if resp.FirstContent() != "Hello from Claude" {
		t.Errorf("unexpected content: %s", resp.FirstContent())
	}
	if resp.Usage.TotalTokens != 15 {
		t.Errorf("unexpected usage: %+v", resp.Usage)
	}
}

func TestStreamChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"stream":true`) {
			t.Errorf("stream request must enable streaming: %s", body)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "event: content_block_delta\n")
		fmt.Fprint(w, "data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"Hi\"}}\n\n")
		fmt.Fprint(w, "event: message_stop\n")
		fmt.Fprint(w, "data: {\"type\":\"message_stop\"}\n\n")
	}))
	defer server.Close()

	client, _ := NewClient(Config{APIKey: "anthropic-key", BaseURL: server.URL})
	stream, err := client.StreamChat(context.Background(), protocol.ChatRequest{
		Model:    "claude",
		Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("StreamChat failed: %v", err)
	}
	defer stream.Close()

	chunk, err := stream.Recv()
	if err != nil && err != io.EOF {
		t.Fatalf("Recv failed: %v", err)
	}
	if chunk != nil && chunk.Delta != "Hi" {
		t.Errorf("unexpected delta: %s", chunk.Delta)
	}
}

func TestToMessagesAndSystemPrompt(t *testing.T) {
	msgs := toMessages([]protocol.Message{
		{Role: protocol.RoleSystem, Content: "be helpful"},
		{Role: protocol.RoleUser, Content: "hello"},
	})
	if len(msgs) != 1 || msgs[0].Role != "user" {
		t.Errorf("unexpected messages: %+v", msgs)
	}
	system := extractSystemPrompt([]protocol.Message{
		{Role: protocol.RoleSystem, Content: "a"},
		{Role: protocol.RoleSystem, Content: "b"},
	})
	if system != "a\nb" {
		t.Errorf("unexpected system prompt: %s", system)
	}
}

func TestToChatResponse(t *testing.T) {
	resp := response{
		ID: "id", Model: "claude", StopReason: "end_turn",
		Content: []textBlock{{Type: "text", Text: "line1"}, {Type: "text", Text: "line2"}},
		Usage:   usage{InputTokens: 3, OutputTokens: 7},
	}.toChatResponse()
	if !strings.Contains(resp.FirstContent(), "line1") || resp.Usage.TotalTokens != 10 {
		t.Errorf("unexpected chat response: %+v", resp)
	}
}

func TestMax(t *testing.T) {
	if max(5, 10) != 10 || max(10, 5) != 10 {
		t.Error("max failed")
	}
}

func TestFactoryRegistration(t *testing.T) {
	client, err := protocol.NewClient(protocol.ClientConfig{
		Provider: protocol.ProviderAnthropic,
		APIKey:   "key",
	})
	if err != nil {
		t.Fatalf("factory registration failed: %v", err)
	}
	if client.Name() != "anthropic" {
		t.Errorf("unexpected client name: %s", client.Name())
	}
}

func TestChatHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "fail", http.StatusBadRequest)
	}))
	defer server.Close()

	client, _ := NewClient(Config{APIKey: "key", BaseURL: server.URL})
	_, err := client.Chat(context.Background(), protocol.ChatRequest{
		Model: "claude", Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "hi"}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStreamChatHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "fail", http.StatusUnauthorized)
	}))
	defer server.Close()

	client, _ := NewClient(Config{APIKey: "key", BaseURL: server.URL})
	_, err := client.StreamChat(context.Background(), protocol.ChatRequest{
		Model: "claude", Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "hi"}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAnthropicStreamInvalidJSON(t *testing.T) {
	s := &anthropicStream{body: io.NopCloser(strings.NewReader("data: bad\n\n"))}
	_, err := s.Recv()
	if err == nil {
		t.Fatal("expected decode error")
	}
}

func TestAnthropicStreamMessageDelta(t *testing.T) {
	body := "event: message_delta\n" +
		"data: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"input_tokens\":1,\"output_tokens\":2}}\n\n"
	s := &anthropicStream{body: io.NopCloser(strings.NewReader(body)), model: "claude"}
	chunk, err := s.Recv()
	if err != nil || chunk.FinishReason != "end_turn" {
		t.Fatalf("unexpected chunk: %+v err=%v", chunk, err)
	}
}

func TestAnthropicStreamClose(t *testing.T) {
	s := &anthropicStream{}
	if err := s.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestToChatResponseSkipsNonTextBlocks(t *testing.T) {
	resp := response{
		Content: []textBlock{{Type: "image", Text: "ignored"}, {Type: "text", Text: "ok"}},
		Usage:   usage{},
	}.toChatResponse()
	if strings.TrimSpace(resp.FirstContent()) != "ok" {
		t.Errorf("unexpected content: %s", resp.FirstContent())
	}
}

func TestChatValidationError(t *testing.T) {
	client, _ := NewClient(Config{APIKey: "key"})
	_, err := client.Chat(context.Background(), protocol.ChatRequest{})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestStreamChatValidationError(t *testing.T) {
	client, _ := NewClient(Config{APIKey: "key"})
	_, err := client.StreamChat(context.Background(), protocol.ChatRequest{})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestBaseURLTrimSuffix(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"id":"1","model":"claude","role":"assistant","content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":1,"output_tokens":1}}`)
	}))
	defer server.Close()

	client, _ := NewClient(Config{APIKey: "key", BaseURL: server.URL + "/"})
	_, err := client.Chat(context.Background(), protocol.ChatRequest{
		Model: "claude", Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
}

func TestAnthropicStreamMetrics(t *testing.T) {
	now := time.Now()
	s := &anthropicStream{
		startAt: now, firstAt: now, endAt: now, model: "claude",
		usage:  protocol.TokenUsage{PromptTokens: 1, CompletionTokens: 2, TotalTokens: 3},
		chunks: 1, bytes: 10,
	}
	m := s.Metrics()
	if m.Provider != "anthropic" || m.TotalTokens != 3 {
		t.Errorf("unexpected metrics: %+v", m)
	}
}

func TestAnthropicStreamDone(t *testing.T) {
	s := &anthropicStream{body: io.NopCloser(strings.NewReader("event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")), model: "claude"}
	_, err := s.Recv()
	if err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
}

func TestAnthropicStreamReadLinePartialEOF(t *testing.T) {
	s := &anthropicStream{body: io.NopCloser(strings.NewReader("partial"))}
	line, err := s.readLine()
	if line != "partial" || err != io.EOF {
		t.Fatalf("unexpected: %q err=%v", line, err)
	}
}

func TestAnthropicStreamRecvEOFOnBody(t *testing.T) {
	s := &anthropicStream{body: io.NopCloser(strings.NewReader(""))}
	_, err := s.Recv()
	if err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
}
