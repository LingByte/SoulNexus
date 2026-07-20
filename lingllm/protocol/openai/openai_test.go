package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/LingByte/lingllm/protocol"
	"github.com/LingByte/lingllm/tools"
)

func TestNewClientRequiresAPIKey(t *testing.T) {
	_, err := NewClient(Config{})
	if err == nil {
		t.Fatal("expected error without api key")
	}
}

func TestNewClientDefaults(t *testing.T) {
	c, err := NewClient(Config{APIKey: "sk-test"})
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	if c.cfg.BaseURL != defaultBaseURL {
		t.Errorf("unexpected base URL: %s", c.cfg.BaseURL)
	}
	if c.Name() != "openai" {
		t.Errorf("unexpected name: %s", c.Name())
	}
}

func TestChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer sk-test" {
			t.Errorf("unexpected auth: %s", auth)
		}
		fmt.Fprint(w, `{
			"id":"chatcmpl-1",
			"created":1700000000,
			"model":"claude",
			"choices":[{"index":0,"message":{"role":"assistant","content":"Hello"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}
		}`)
	}))
	defer server.Close()

	client, err := NewClient(Config{
		APIKey:  "sk-test",
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	resp, err := client.Chat(context.Background(), protocol.ChatRequest{
		Model:    "claude",
		Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if resp.FirstContent() != "Hello" {
		t.Errorf("unexpected content: %s", resp.FirstContent())
	}
	if resp.Usage.TotalTokens != 15 {
		t.Errorf("unexpected usage: %+v", resp.Usage)
	}
	if resp.Metrics.HTTPStatus != 200 {
		t.Errorf("unexpected metrics: %+v", resp.Metrics)
	}
}

func TestChatHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()

	client, _ := NewClient(Config{APIKey: "sk-test", BaseURL: server.URL})
	_, err := client.Chat(context.Background(), protocol.ChatRequest{
		Model:    "claude",
		Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "hi"}},
	})
	if err == nil {
		t.Fatal("expected http error")
	}
}

func TestStreamChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"Hi\"}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	client, _ := NewClient(Config{APIKey: "sk-test", BaseURL: server.URL})
	stream, err := client.StreamChat(context.Background(), protocol.ChatRequest{
		Model:    "claude",
		Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("StreamChat failed: %v", err)
	}
	defer stream.Close()

	var content strings.Builder
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Recv failed: %v", err)
		}
		content.WriteString(chunk.Delta)
	}
	if content.String() != "Hi" {
		t.Errorf("unexpected stream content: %s", content.String())
	}
	if stream.Metrics().Chunks == 0 {
		t.Error("expected chunk metrics")
	}
}

func TestToOpenAIMessages(t *testing.T) {
	msgs := toOpenAIMessages([]protocol.Message{
		{Role: protocol.RoleUser, Content: "hello"},
		{
			Role: protocol.RoleAssistant,
			ToolCalls: []protocol.ToolCall{{
				ID: "call_1", Type: "function",
				Function: protocol.FunctionCall{Name: "current_time", Arguments: json.RawMessage(`{}`)},
			}},
		},
		{Role: protocol.RoleTool, ToolCallID: "call_1", Content: `{"now":"2026-07-15"}`},
	})
	if len(msgs) != 3 {
		t.Fatalf("unexpected message count: %+v", msgs)
	}
	if msgs[0].Role != "user" || derefString(msgs[0].Content) != "hello" {
		t.Errorf("unexpected user message: %+v", msgs[0])
	}
	if len(msgs[1].ToolCalls) != 1 || msgs[1].ToolCalls[0].Function.Name != "current_time" {
		t.Errorf("unexpected assistant tool_calls: %+v", msgs[1])
	}
	if msgs[2].Role != "tool" || msgs[2].ToolCallID != "call_1" {
		t.Errorf("unexpected tool message: %+v", msgs[2])
	}
}

func TestToChatResponse(t *testing.T) {
	raw := openAIResponse{
		ID: "id-1", Created: 1700000000, Model: "claude",
		Choices: []openAIChoice{{
			Index:        0,
			Message:      openAIMessage{Role: "assistant", Content: strPtr("ok")},
			FinishReason: "stop",
		}},
		Usage: openAIUsage{PromptTokens: 1, CompletionTokens: 2, TotalTokens: 3},
	}
	resp := raw.toChatResponse()
	if resp.ID != "id-1" || resp.FirstContent() != "ok" || resp.Usage.TotalTokens != 3 {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestChatWithTools(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		tools, ok := body["tools"].([]any)
		if !ok || len(tools) != 1 {
			t.Fatalf("expected tools in request, got: %#v", body["tools"])
		}
		if body["tool_choice"] != "auto" {
			t.Fatalf("expected tool_choice auto, got: %#v", body["tool_choice"])
		}
		fmt.Fprint(w, `{
			"id":"chatcmpl-tools",
			"created":1700000000,
			"model":"gpt-4o",
			"choices":[{"index":0,"message":{
				"role":"assistant",
				"content":null,
				"tool_calls":[{"id":"call_1","type":"function","function":{"name":"current_time","arguments":"{}"}}]
			},"finish_reason":"tool_calls"}],
			"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}
		}`)
	}))
	defer server.Close()

	client, err := NewClient(Config{APIKey: "sk-test", BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	resp, err := client.Chat(context.Background(), protocol.ChatRequest{
		Model:    "gpt-4o",
		Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "现在几点"}},
		Tools: []protocol.Tool{{
			Name:        "current_time",
			Description: "Get current time",
			Parameters:  map[string]interface{}{"type": "object"},
		}},
		ToolChoice: protocol.ToolChoiceAuto,
	})
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	msg := resp.Choices[0].Message
	if len(msg.ToolCalls) != 1 || msg.ToolCalls[0].Function.Name != "current_time" {
		t.Fatalf("unexpected tool calls: %+v", msg.ToolCalls)
	}
}

func TestChatToolRoundTrip(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		msgs, _ := body["messages"].([]any)
		if calls == 1 {
			if len(msgs) != 1 {
				t.Fatalf("round 1 expected 1 message, got %d", len(msgs))
			}
			fmt.Fprint(w, `{
				"id":"chatcmpl-1","created":1700000000,"model":"gpt-4o",
				"choices":[{"index":0,"message":{
					"role":"assistant","content":null,
					"tool_calls":[{"id":"call_1","type":"function","function":{"name":"current_time","arguments":"{}"}}]
				},"finish_reason":"tool_calls"}],
				"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}
			}`)
			return
		}
		if len(msgs) != 3 {
			t.Fatalf("round 2 expected 3 messages, got %d: %#v", len(msgs), msgs)
		}
		last, _ := msgs[2].(map[string]any)
		if last["role"] != "tool" {
			t.Fatalf("round 2 last message should be tool, got %#v", last)
		}
		fmt.Fprint(w, `{
			"id":"chatcmpl-2","created":1700000001,"model":"gpt-4o",
			"choices":[{"index":0,"message":{"role":"assistant","content":"今天是2026年7月15日"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}
		}`)
	}))
	defer server.Close()

	client, err := NewClient(Config{APIKey: "sk-test", BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	executor := tools.NewSimpleToolExecutor()
	executor.RegisterTool(tools.MakeTool("current_time", "time", map[string]interface{}{"type": "object"}),
		func(json.RawMessage) (string, error) {
			return `{"spoken_zh":"2026年7月15日"}`, nil
		})
	tc := tools.NewToolChain(client, executor)
	resp, err := tc.ExecuteWithTools(context.Background(), protocol.ChatRequest{
		Model:    "gpt-4o",
		Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "今天几号"}},
		Tools:    executor.GetTools(),
	})
	if err != nil {
		t.Fatalf("ExecuteWithTools: %v", err)
	}
	if resp.FirstContent() != "今天是2026年7月15日" {
		t.Fatalf("unexpected reply: %q", resp.FirstContent())
	}
	if calls != 2 {
		t.Fatalf("expected 2 HTTP calls, got %d", calls)
	}
}

func TestFactoryRegistration(t *testing.T) {
	client, err := protocol.NewClient(protocol.ClientConfig{
		Provider: protocol.ProviderOpenAI,
		APIKey:   "sk-test",
	})
	if err != nil {
		t.Fatalf("factory registration failed: %v", err)
	}
	if client.Name() != "openai" {
		t.Errorf("unexpected client name: %s", client.Name())
	}
}

func TestOpenAIStreamMetrics(t *testing.T) {
	now := time.Now()
	s := &openAIStream{
		startAt: now,
		firstAt: now,
		endAt:   now,
		model:   "claude",
		usage: protocol.TokenUsage{
			PromptTokens: 1, CompletionTokens: 2, TotalTokens: 3,
		},
		chunks: 2, bytes: 100, httpStatus: 200,
	}

	m := s.Metrics()
	if m.Provider != "openai" || m.Model != "claude" || m.TotalTokens != 3 {
		t.Errorf("unexpected metrics: %+v", m)
	}
}

func TestOpenAIStreamRecvUsageAndEmptyChoices(t *testing.T) {
	body := "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":1,\"completion_tokens\":2,\"total_tokens\":3}}\n\n" +
		"data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"y\"}}]}\n\n" +
		"data: [DONE]\n\n"
	s := &openAIStream{body: io.NopCloser(strings.NewReader(body)), model: "claude"}
	chunk, err := s.Recv()
	if err != nil || chunk.Delta != "y" {
		t.Fatalf("Recv failed: chunk=%+v err=%v", chunk, err)
	}
	if s.Metrics().TotalTokens != 3 {
		t.Errorf("expected usage in metrics")
	}
}

func TestOpenAIStreamReadLineError(t *testing.T) {
	s := &openAIStream{body: io.NopCloser(&failReader{})}
	_, err := s.readLine()
	if err == nil {
		t.Fatal("expected read error")
	}
}

type failReader struct{}

func (f *failReader) Read(p []byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestChatValidationError(t *testing.T) {
	client, _ := NewClient(Config{APIKey: "sk-test"})
	_, err := client.Chat(context.Background(), protocol.ChatRequest{})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestChatWithOrgAndProjectHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("OpenAI-Organization") != "org" || r.Header.Get("OpenAI-Project") != "proj" {
			t.Errorf("missing org/project headers")
		}
		fmt.Fprint(w, `{"id":"1","created":1,"model":"claude","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{}}`)
	}))
	defer server.Close()

	client, _ := NewClient(Config{
		APIKey: "sk-test", BaseURL: server.URL,
		Organization: "org", Project: "proj",
	})
	_, err := client.Chat(context.Background(), protocol.ChatRequest{
		Model: "claude", Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
}

func TestStreamChatHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "fail", http.StatusUnauthorized)
	}))
	defer server.Close()

	client, _ := NewClient(Config{APIKey: "sk-test", BaseURL: server.URL})
	_, err := client.StreamChat(context.Background(), protocol.ChatRequest{
		Model: "claude", Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "hi"}},
	})
	if err == nil {
		t.Fatal("expected stream error")
	}
}

func TestStreamChatInvalidChunk(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "data: not-json\n\n")
	}))
	defer server.Close()

	client, _ := NewClient(Config{APIKey: "sk-test", BaseURL: server.URL})
	stream, _ := client.StreamChat(context.Background(), protocol.ChatRequest{
		Model: "claude", Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "hi"}},
	})
	_, err := stream.Recv()
	if err == nil {
		t.Fatal("expected decode error")
	}
}

func TestOpenAIStreamClose(t *testing.T) {
	s := &openAIStream{}
	if err := s.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestOpenAIStreamReadLineEOFWithPartialLine(t *testing.T) {
	s := &openAIStream{body: io.NopCloser(strings.NewReader("partial"))}
	line, err := s.readLine()
	if line != "partial" || err != io.EOF {
		t.Fatalf("unexpected readLine result: %q err=%v", line, err)
	}
}

func TestOpenAIStreamSkipsNonDataLines(t *testing.T) {
	body := "event: ping\n\ndata: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"x\"}}]}\n\ndata: [DONE]\n\n"
	s := &openAIStream{body: io.NopCloser(strings.NewReader(body))}
	chunk, err := s.Recv()
	if err != nil || chunk.Delta != "x" {
		t.Fatalf("Recv failed: chunk=%+v err=%v", chunk, err)
	}
}
