package response

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

func TestChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
			"id":"resp-1",
			"model":"claude",
			"choices":[{"index":0,"message":{"role":"assistant","content":"Hello"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":5,"completion_tokens":3,"total_tokens":8}
		}`)
	}))
	defer server.Close()

	client, _ := NewClient(Config{APIKey: "sk-test", BaseURL: server.URL})
	resp, err := client.Chat(context.Background(), protocol.ChatRequest{
		Model:       "claude",
		Messages:    []protocol.Message{{Role: protocol.RoleUser, Content: "hi"}},
		MaxTokens:   50,
		Temperature: 0.2,
	})
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if resp.FirstContent() != "Hello" || resp.Usage.TotalTokens != 8 {
		t.Errorf("unexpected response: %+v", resp)
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

	chunk, err := stream.Recv()
	if err != nil && err != io.EOF {
		t.Fatalf("Recv failed: %v", err)
	}
	if chunk != nil && chunk.Delta != "Hi" {
		t.Errorf("unexpected delta: %s", chunk.Delta)
	}
	if stream.Metrics().Provider != "openai-responses" {
		t.Errorf("unexpected provider metric: %s", stream.Metrics().Provider)
	}
}

func TestName(t *testing.T) {
	client, _ := NewClient(Config{APIKey: "sk-test"})
	if client.Name() != "openai-responses" {
		t.Errorf("unexpected name: %s", client.Name())
	}
}

func TestToResponsesMessages(t *testing.T) {
	msgs := toResponsesMessages([]protocol.Message{{Role: protocol.RoleUser, Content: "hi"}})
	if len(msgs) != 1 || msgs[0].Content != "hi" {
		t.Errorf("unexpected messages: %+v", msgs)
	}
}

func TestFactoryRegistration(t *testing.T) {
	client, err := protocol.NewClient(protocol.ClientConfig{
		Provider: protocol.ProviderOpenAIResponse,
		APIKey:   "sk-test",
	})
	if err != nil {
		t.Fatalf("factory registration failed: %v", err)
	}
	if client.Name() != "openai-responses" {
		t.Errorf("unexpected client name: %s", client.Name())
	}
}

func TestResponsesStreamMetrics(t *testing.T) {
	now := time.Now()
	s := &responsesStream{
		startAt: now, firstAt: now, endAt: now, model: "claude",
		usage:  protocol.TokenUsage{TotalTokens: 5},
		chunks: 1, bytes: 20, httpStatus: 200,
	}
	m := s.Metrics()
	if m.Provider != "openai-responses" || m.TotalTokens != 5 {
		t.Errorf("unexpected metrics: %+v", m)
	}
}

func TestResponsesStreamDoneAndSkipEmptyDelta(t *testing.T) {
	body := "data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"\"}}]}\n\n" +
		"data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"z\"}}]}\n\n" +
		"data: [DONE]\n\n"
	s := &responsesStream{body: io.NopCloser(strings.NewReader(body)), model: "claude"}
	chunk, err := s.Recv()
	if err != nil || chunk.Delta != "z" {
		t.Fatalf("Recv failed: chunk=%+v err=%v", chunk, err)
	}
}

func TestResponsesStreamReadLinePartialEOF(t *testing.T) {
	s := &responsesStream{body: io.NopCloser(strings.NewReader("line"))}
	line, err := s.readLine()
	if line != "line" || err != io.EOF {
		t.Fatalf("unexpected: %q err=%v", line, err)
	}
}

func TestChatHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "fail", http.StatusBadRequest)
	}))
	defer server.Close()

	client, _ := NewClient(Config{APIKey: "sk-test", BaseURL: server.URL})
	_, err := client.Chat(context.Background(), protocol.ChatRequest{
		Model: "claude", Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "hi"}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestChatValidationError(t *testing.T) {
	client, _ := NewClient(Config{APIKey: "sk-test"})
	_, err := client.Chat(context.Background(), protocol.ChatRequest{})
	if err == nil {
		t.Fatal("expected validation error")
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
		t.Fatal("expected error")
	}
}

func TestResponsesStreamInvalidJSON(t *testing.T) {
	s := &responsesStream{body: io.NopCloser(strings.NewReader("data: bad\n\n"))}
	_, err := s.Recv()
	if err == nil {
		t.Fatal("expected decode error")
	}
}

func TestResponsesStreamClose(t *testing.T) {
	s := &responsesStream{}
	if err := s.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestChatWithOrgProjectHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("OpenAI-Organization") != "org" {
			t.Errorf("missing org header")
		}
		w.Write([]byte(`{"id":"1","model":"claude","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{}}`))
	}))
	defer server.Close()

	client, _ := NewClient(Config{
		APIKey: "sk-test", BaseURL: server.URL, Organization: "org", Project: "proj",
	})
	_, err := client.Chat(context.Background(), protocol.ChatRequest{
		Model: "claude", Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
}
