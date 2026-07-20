package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/LingByte/lingllm/metrics"
	"github.com/LingByte/lingllm/protocol"
)

// Config for Ollama HTTP API.
type Config struct {
	BaseURL    string
	APIKey     string // optional, if gateway uses key
	HTTPClient *http.Client
}

// Client implements llm.ChatModel over Ollama /api/chat.
type Client struct {
	cfg        Config
	httpClient *http.Client
}

func NewClient(cfg Config) (*Client, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:11434"
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 60 * time.Second}
	}
	return &Client{cfg: cfg, httpClient: hc}, nil
}

func (c *Client) Name() string { return "ollama" }

func init() {
	protocol.RegisterFactory(protocol.ProviderOllama, func(cfg protocol.ClientConfig) (protocol.ChatModel, error) {
		return NewClient(Config{
			APIKey:  cfg.APIKey,
			BaseURL: cfg.BaseURL,
		})
	})
}

// Chat (non-streaming).
func (c *Client) Chat(ctx context.Context, req protocol.ChatRequest) (*protocol.ChatResponse, error) {
	start := time.Now()
	if err := req.Validate(); err != nil {
		return nil, err
	}

	payload := map[string]any{
		"model":    req.Model,
		"messages": toOllamaMessages(req.Messages),
		"stream":   false,
	}
	if req.MaxTokens > 0 {
		payload["options"] = map[string]any{"num_predict": req.MaxTokens}
	}
	if req.Temperature != 0 {
		opt := payload["options"].(map[string]any)
		opt["temperature"] = req.Temperature
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal ollama payload: %w", err)
	}

	endpoint := fmt.Sprintf("%s/api/chat", c.cfg.BaseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build ollama request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call ollama: %w", err)
	}
	defer httpResp.Body.Close()

	respBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read ollama response: %w", err)
	}
	if httpResp.StatusCode >= 300 {
		return nil, fmt.Errorf("ollama http %d: %s", httpResp.StatusCode, string(respBytes))
	}

	var raw ollamaResponse
	if err := json.Unmarshal(respBytes, &raw); err != nil {
		return nil, fmt.Errorf("decode ollama response: %w", err)
	}

	chatResp := raw.toChatResponse()
	now := time.Now()
	chatResp.Metrics = metrics.CallMetrics{
		Provider:      c.Name(),
		Model:         chatResp.Model,
		StartAt:       start,
		EndAt:         now,
		FirstAt:       now,
		Bytes:         len(respBytes),
		Chunks:        1,
		RequestBytes:  len(body),
		ResponseBytes: len(respBytes),
		HTTPStatus:    httpResp.StatusCode,
	}
	return chatResp, nil
}

// StreamChat streams /api/chat with stream=true (line-delimited JSON).
func (c *Client) StreamChat(ctx context.Context, req protocol.ChatRequest) (protocol.ChatStream, error) {
	start := time.Now()
	if err := req.Validate(); err != nil {
		return nil, err
	}

	payload := map[string]any{
		"model":    req.Model,
		"messages": toOllamaMessages(req.Messages),
		"stream":   true,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal ollama payload: %w", err)
	}

	endpoint := fmt.Sprintf("%s/api/chat", c.cfg.BaseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build ollama request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call ollama: %w", err)
	}
	if httpResp.StatusCode >= 300 {
		b, _ := io.ReadAll(httpResp.Body)
		httpResp.Body.Close()
		return nil, fmt.Errorf("ollama http %d: %s", httpResp.StatusCode, string(b))
	}

	return &ollamaStream{
		body:         httpResp.Body,
		startAt:      start,
		model:        req.Model,
		requestBytes: len(body),
		httpStatus:   httpResp.StatusCode,
	}, nil
}

// --- stream impl ---

type ollamaStream struct {
	body         io.ReadCloser
	startAt      time.Time
	firstAt      time.Time
	endAt        time.Time
	model        string
	prevContent  string
	chunks       int
	bytes        int
	requestBytes int
	httpStatus   int
}

func (s *ollamaStream) Recv() (*protocol.ChatStreamChunk, error) {
	var line bytes.Buffer
	buf := make([]byte, 1)
	for {
		n, err := s.body.Read(buf)
		if n > 0 {
			s.bytes += n
			if buf[0] == '\n' {
				break
			}
			line.WriteByte(buf[0])
		}
		if err != nil {
			if err == io.EOF && line.Len() == 0 {
				s.endAt = time.Now()
				return nil, io.EOF
			}
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if line.Len() == 0 {
			continue
		}
	}
	payload := line.String()
	if payload == "" {
		return nil, io.EOF
	}

	var raw struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		Done bool `json:"done"`
	}
	if err := json.Unmarshal([]byte(payload), &raw); err != nil {
		return nil, fmt.Errorf("decode ollama stream chunk: %w", err)
	}

	if s.firstAt.IsZero() {
		s.firstAt = time.Now()
	}
	s.chunks++

	// Ollama streams cumulative content; emit only the new suffix as delta.
	content := raw.Message.Content
	delta := content
	if strings.HasPrefix(content, s.prevContent) {
		delta = content[len(s.prevContent):]
	}
	s.prevContent = content
	if raw.Done {
		s.endAt = time.Now()
	}
	return &protocol.ChatStreamChunk{
		Index: s.chunks - 1,
		Role:  protocol.MessageRole(raw.Message.Role),
		Delta: delta,
		FinishReason: func() string {
			if raw.Done {
				return "stop"
			}
			return ""
		}(),
	}, nil
}

func (s *ollamaStream) Close() error {
	if s.body != nil {
		return s.body.Close()
	}
	return nil
}

func (s *ollamaStream) Metrics() metrics.CallMetrics {
	return metrics.CallMetrics{
		Provider:      "ollama",
		Model:         s.model,
		StartAt:       s.startAt,
		FirstAt:       s.firstAt,
		EndAt:         s.endAt,
		Bytes:         s.bytes,
		Chunks:        s.chunks,
		RequestBytes:  s.requestBytes,
		ResponseBytes: s.bytes,
		HTTPStatus:    s.httpStatus,
	}
}

// --- helpers ---

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaResponse struct {
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	Model     string `json:"model"`
	EvalCount int    `json:"eval_count"`
}

func (o ollamaResponse) toChatResponse() *protocol.ChatResponse {
	msg := protocol.Message{Role: protocol.MessageRole(o.Message.Role), Content: o.Message.Content}
	return &protocol.ChatResponse{
		ID:        "",
		Model:     o.Model,
		CreatedAt: time.Now(),
		Choices: []protocol.Choice{{
			Index:        0,
			Message:      msg,
			FinishReason: "stop",
		}},
		Usage: protocol.TokenUsage{},
	}
}

func toOllamaMessages(msgs []protocol.Message) []ollamaMessage {
	out := make([]ollamaMessage, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, ollamaMessage{Role: string(m.Role), Content: m.Content})
	}
	return out
}
