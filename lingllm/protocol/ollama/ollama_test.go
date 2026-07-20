package ollama

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/LingByte/lingllm/metrics"
	"github.com/LingByte/lingllm/protocol"
)

func TestNewClientDefaults(t *testing.T) {
	c, err := NewClient(Config{})
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	if c.cfg.BaseURL != "http://localhost:11434" {
		t.Errorf("unexpected base URL: %s", c.cfg.BaseURL)
	}
}

func TestChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"model":"llama3","message":{"role":"assistant","content":"Hello from Ollama"}}`)
	}))
	defer server.Close()

	client, _ := NewClient(Config{BaseURL: server.URL})
	resp, err := client.Chat(context.Background(), protocol.ChatRequest{
		Model:       "llama3",
		Messages:    []protocol.Message{{Role: protocol.RoleUser, Content: "hi"}},
		MaxTokens:   100,
		Temperature: 0.5,
	})
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if resp.FirstContent() != "Hello from Ollama" {
		t.Errorf("unexpected content: %s", resp.FirstContent())
	}
}

func TestChatHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "fail", http.StatusInternalServerError)
	}))
	defer server.Close()

	client, _ := NewClient(Config{BaseURL: server.URL})
	_, err := client.Chat(context.Background(), protocol.ChatRequest{
		Model:    "llama3",
		Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "hi"}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStreamChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"message":{"role":"assistant","content":"par"},"done":false}`+"\n")
		fmt.Fprint(w, `{"message":{"role":"assistant","content":"partial"},"done":true}`+"\n")
	}))
	defer server.Close()

	client, _ := NewClient(Config{BaseURL: server.URL})
	stream, err := client.StreamChat(context.Background(), protocol.ChatRequest{
		Model:    "llama3",
		Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("StreamChat failed: %v", err)
	}
	defer stream.Close()

	chunk, err := stream.Recv()
	if err != nil {
		t.Fatalf("Recv failed: %v", err)
	}
	if chunk.Delta != "par" {
		t.Errorf("unexpected delta: %s", chunk.Delta)
	}

	chunk, err = stream.Recv()
	if err != nil {
		t.Fatalf("Recv failed: %v", err)
	}
	if chunk.Delta != "tial" {
		t.Errorf("unexpected delta: %s", chunk.Delta)
	}

	if _, err := stream.Recv(); err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
}

func TestToOllamaMessagesAndResponse(t *testing.T) {
	msgs := toOllamaMessages([]protocol.Message{{Role: protocol.RoleUser, Content: "hi"}})
	if len(msgs) != 1 || msgs[0].Content != "hi" {
		t.Errorf("unexpected ollama messages: %+v", msgs)
	}

	raw := ollamaResponse{}
	raw.Message.Role = "assistant"
	raw.Message.Content = "ok"
	raw.Model = "llama3"
	resp := raw.toChatResponse()
	if resp.FirstContent() != "ok" || resp.Model != "llama3" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestFactoryRegistration(t *testing.T) {
	client, err := protocol.NewClient(protocol.ClientConfig{
		Provider: protocol.ProviderOllama,
	})
	if err != nil {
		t.Fatalf("factory registration failed: %v", err)
	}
	if client.Name() != "ollama" {
		t.Errorf("unexpected client name: %s", client.Name())
	}
}

func TestOllamaStreamMetrics(t *testing.T) {
	now := time.Now()
	s := &ollamaStream{
		startAt: now, firstAt: now, endAt: now, model: "llama3",
		chunks: 2, bytes: 50, httpStatus: 200, requestBytes: 10,
	}
	m := s.Metrics()
	if m.Provider != "ollama" || m.Model != "llama3" || m.Chunks != 2 {
		t.Errorf("unexpected metrics: %+v", m)
	}
	_ = metrics.CallMetrics{}
}

func TestChatValidationError(t *testing.T) {
	client, _ := NewClient(Config{})
	_, err := client.Chat(context.Background(), protocol.ChatRequest{})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestChatWithAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("missing auth header")
		}
		w.Write([]byte(`{"model":"llama3","message":{"role":"assistant","content":"ok"}}`))
	}))
	defer server.Close()

	client, _ := NewClient(Config{BaseURL: server.URL, APIKey: "secret"})
	resp, err := client.Chat(context.Background(), protocol.ChatRequest{
		Model: "llama3", Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "hi"}},
	})
	if err != nil || resp.FirstContent() != "ok" {
		t.Fatalf("Chat failed: resp=%+v err=%v", resp, err)
	}
}

func TestStreamChatHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "fail", http.StatusInternalServerError)
	}))
	defer server.Close()

	client, _ := NewClient(Config{BaseURL: server.URL})
	_, err := client.StreamChat(context.Background(), protocol.ChatRequest{
		Model: "llama3", Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "hi"}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOllamaStreamInvalidJSON(t *testing.T) {
	s := &ollamaStream{body: io.NopCloser(strings.NewReader("not-json\n"))}
	_, err := s.Recv()
	if err == nil {
		t.Fatal("expected decode error")
	}
}

func TestOllamaStreamClose(t *testing.T) {
	s := &ollamaStream{}
	if err := s.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestOllamaStreamEmptyLineEOF(t *testing.T) {
	s := &ollamaStream{body: io.NopCloser(strings.NewReader("\n"))}
	_, err := s.Recv()
	if err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
}
