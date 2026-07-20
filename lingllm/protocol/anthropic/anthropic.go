package anthropic

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

const (
	defaultBaseURL = "https://api.anthropic.com"
	apiVersion     = "2023-06-01"
)

// Config configures the Claude messages API client.
type Config struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
}

// Client implements llm.ChatModel for Anthropic's /v1/messages API.
type Client struct {
	cfg        Config
	httpClient *http.Client
}

// NewClient constructs an Anthropic client with defaults.
func NewClient(cfg Config) (*Client, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("anthropic api key is required")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{cfg: cfg, httpClient: client}, nil
}

func (c *Client) Name() string { return "anthropic" }

func init() {
	protocol.RegisterFactory(protocol.ProviderAnthropic, func(cfg protocol.ClientConfig) (protocol.ChatModel, error) {
		return NewClient(Config{
			APIKey:  cfg.APIKey,
			BaseURL: cfg.BaseURL,
		})
	})
}

// Chat sends a message request to Anthropic and maps the response back to ChatResponse.
func (c *Client) Chat(ctx context.Context, req protocol.ChatRequest) (*protocol.ChatResponse, error) {
	start := time.Now()
	if err := req.Validate(); err != nil {
		return nil, err
	}

	payload := request{
		Model:         req.Model,
		MaxTokens:     max(req.MaxTokens, 1),
		Messages:      toMessages(req.Messages),
		Temperature:   req.Temperature,
		TopP:          req.TopP,
		StopSequences: req.Stop,
		System:        extractSystemPrompt(req.Messages),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal anthropic payload: %w", err)
	}

	endpoint := fmt.Sprintf("%s/v1/messages", strings.TrimSuffix(c.cfg.BaseURL, "/"))
	reqHTTP, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build anthropic request: %w", err)
	}
	reqHTTP.Header.Set("x-api-key", c.cfg.APIKey)
	reqHTTP.Header.Set("Content-Type", "application/json")
	reqHTTP.Header.Set("anthropic-version", apiVersion)

	httpResp, err := c.httpClient.Do(reqHTTP)
	if err != nil {
		return nil, fmt.Errorf("call anthropic: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read anthropic response: %w", err)
	}
	if httpResp.StatusCode >= 300 {
		return nil, fmt.Errorf("anthropic http %d: %s", httpResp.StatusCode, string(respBody))
	}

	var raw response
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return nil, fmt.Errorf("decode anthropic response: %w", err)
	}

	resp := raw.toChatResponse()
	now := time.Now()
	resp.Metrics = metrics.CallMetrics{
		Provider:         c.Name(),
		Model:            resp.Model,
		StartAt:          start,
		FirstAt:          now,
		EndAt:            now,
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
		TotalTokens:      resp.Usage.TotalTokens,
		Chunks:           1,
		Bytes:            len(respBody),
		RequestBytes:     len(body),
		ResponseBytes:    len(respBody),
		HTTPStatus:       httpResp.StatusCode,
	}
	return resp, nil
}

// StreamChat uses SSE-style streaming from Anthropic messages endpoint.
func (c *Client) StreamChat(ctx context.Context, req protocol.ChatRequest) (protocol.ChatStream, error) {
	start := time.Now()
	if err := req.Validate(); err != nil {
		return nil, err
	}

	payload := request{
		Model:         req.Model,
		MaxTokens:     max(req.MaxTokens, 1),
		Messages:      toMessages(req.Messages),
		Temperature:   req.Temperature,
		TopP:          req.TopP,
		StopSequences: req.Stop,
		System:        extractSystemPrompt(req.Messages),
		Stream:        true,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal anthropic payload: %w", err)
	}

	endpoint := fmt.Sprintf("%s/v1/messages", strings.TrimSuffix(c.cfg.BaseURL, "/"))
	reqHTTP, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build anthropic request: %w", err)
	}
	reqHTTP.Header.Set("x-api-key", c.cfg.APIKey)
	reqHTTP.Header.Set("Content-Type", "application/json")
	reqHTTP.Header.Set("anthropic-version", apiVersion)
	reqHTTP.Header.Set("Accept", "text/event-stream")
	httpResp, err := c.httpClient.Do(reqHTTP)
	if err != nil {
		return nil, fmt.Errorf("call anthropic: %w", err)
	}

	if httpResp.StatusCode >= 300 {
		b, _ := io.ReadAll(httpResp.Body)
		httpResp.Body.Close()
		return nil, fmt.Errorf("anthropic http %d: %s", httpResp.StatusCode, string(b))
	}

	stream := &anthropicStream{
		body:         httpResp.Body,
		startAt:      start,
		model:        req.Model,
		httpStatus:   httpResp.StatusCode,
		requestBytes: len(body),
	}
	return stream, nil
}

type request struct {
	Model         string    `json:"model"`
	MaxTokens     int       `json:"max_tokens"`
	Messages      []message `json:"messages"`
	System        string    `json:"system,omitempty"`
	Temperature   float32   `json:"temperature,omitempty"`
	TopP          float32   `json:"top_p,omitempty"`
	StopSequences []string  `json:"stop_sequences,omitempty"`
	Stream        bool      `json:"stream,omitempty"`
}

type message struct {
	Role    string      `json:"role"`
	Content []textBlock `json:"content"`
}

type textBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type response struct {
	ID         string      `json:"id"`
	Model      string      `json:"model"`
	Role       string      `json:"role"`
	Content    []textBlock `json:"content"`
	StopReason string      `json:"stop_reason"`
	Usage      usage       `json:"usage"`
	CreatedAt  int64       `json:"created_at,omitempty"`
}

// anthropicStream implements llm.ChatStream for Anthropic SSE.
type anthropicStream struct {
	body          io.ReadCloser
	startAt       time.Time
	firstAt       time.Time
	endAt         time.Time
	model         string
	usage         protocol.TokenUsage
	chunks        int
	bytes         int
	responseBytes int
	requestBytes  int
	httpStatus    int
}

func (s *anthropicStream) Recv() (*protocol.ChatStreamChunk, error) {
	for {
		line, err := s.readLine()
		if err != nil {
			if err == io.EOF {
				s.endAt = time.Now()
			}
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "event:") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			if payload == "[DONE]" {
				s.endAt = time.Now()
				return nil, io.EOF
			}
			continue
		}

		var event struct {
			Type  string `json:"type"`
			Delta struct {
				Type       string `json:"type"`
				Text       string `json:"text"`
				StopReason string `json:"stop_reason"`
			} `json:"delta"`
			Usage usage `json:"usage"`
		}
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			return nil, fmt.Errorf("decode anthropic stream chunk: %w", err)
		}

		total := event.Usage.InputTokens + event.Usage.OutputTokens
		if total > 0 {
			s.usage = protocol.TokenUsage{
				PromptTokens:     event.Usage.InputTokens,
				CompletionTokens: event.Usage.OutputTokens,
				TotalTokens:      total,
			}
		}

		switch event.Type {
		case "content_block_delta":
			text := event.Delta.Text
			if text == "" {
				continue
			}
			if s.firstAt.IsZero() {
				s.firstAt = time.Now()
			}
			s.chunks++
			s.bytes += len(payload)
			s.responseBytes += len(payload)
			return &protocol.ChatStreamChunk{
				Index: 0,
				Role:  protocol.RoleAssistant,
				Delta: text,
			}, nil
		case "message_delta":
			if event.Delta.StopReason != "" {
				s.endAt = time.Now()
				return &protocol.ChatStreamChunk{
					Role:         protocol.RoleAssistant,
					FinishReason: event.Delta.StopReason,
				}, nil
			}
			continue
		case "message_stop":
			s.endAt = time.Now()
			return nil, io.EOF
		default:
			continue
		}
	}
}

func (s *anthropicStream) readLine() (string, error) {
	var buf [1]byte
	var line strings.Builder
	for {
		n, err := s.body.Read(buf[:])
		if n > 0 {
			s.bytes += n
			if buf[0] == '\n' {
				return line.String(), nil
			}
			line.WriteByte(buf[0])
		}
		if err != nil {
			if err == io.EOF && line.Len() > 0 {
				return line.String(), io.EOF
			}
			return "", err
		}
	}
}

func (s *anthropicStream) Close() error {
	if s.body != nil {
		return s.body.Close()
	}
	return nil
}

func (s *anthropicStream) Metrics() metrics.CallMetrics {
	return metrics.CallMetrics{
		Provider:         "anthropic",
		Model:            s.model,
		StartAt:          s.startAt,
		FirstAt:          s.firstAt,
		EndAt:            s.endAt,
		Bytes:            s.bytes,
		Chunks:           s.chunks,
		PromptTokens:     s.usage.PromptTokens,
		CompletionTokens: s.usage.CompletionTokens,
		TotalTokens:      s.usage.TotalTokens,
	}
}

func toMessages(msgs []protocol.Message) []message {
	out := make([]message, 0, len(msgs))
	for _, m := range msgs {
		if m.Role == protocol.RoleSystem {
			continue
		}
		role := string(m.Role)
		switch m.Role {
		case protocol.RoleAssistant:
			role = "assistant"
		case protocol.RoleUser, protocol.RoleTool:
			role = "user"
		}
		out = append(out, message{
			Role: role,
			Content: []textBlock{{
				Type: "text",
				Text: m.Content,
			}},
		})
	}
	return out
}

func extractSystemPrompt(msgs []protocol.Message) string {
	var parts []string
	for _, m := range msgs {
		if m.Role == protocol.RoleSystem {
			parts = append(parts, m.Content)
		}
	}
	return strings.Join(parts, "\n")
}

func (r response) toChatResponse() *protocol.ChatResponse {
	// Anthropic returns a single assistant message in content array.
	var contentBuilder strings.Builder
	for i, block := range r.Content {
		if block.Type != "text" {
			continue
		}
		if i > 0 {
			contentBuilder.WriteString("\n")
		}
		contentBuilder.WriteString(block.Text)
	}

	return &protocol.ChatResponse{
		ID:        r.ID,
		Model:     r.Model,
		CreatedAt: time.Unix(r.CreatedAt, 0),
		Choices: []protocol.Choice{
			{
				Index: 0,
				Message: protocol.Message{
					Role:    protocol.RoleAssistant,
					Content: contentBuilder.String(),
				},
				FinishReason: r.StopReason,
			},
		},
		Usage: protocol.TokenUsage{
			PromptTokens:     r.Usage.InputTokens,
			CompletionTokens: r.Usage.OutputTokens,
			TotalTokens:      r.Usage.InputTokens + r.Usage.OutputTokens,
		},
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
