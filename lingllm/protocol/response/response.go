package response

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

const defaultBaseURL = "https://api.openai.com/v1"

// Config for OpenAI-compatible gateway clients (chat/completions on custom base URLs).
type Config struct {
	APIKey       string
	BaseURL      string
	HTTPClient   *http.Client
	Organization string
	Project      string
}

type Client struct {
	cfg        Config
	httpClient *http.Client
}

func NewClient(cfg Config) (*Client, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("openai api key is required")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 60 * time.Second}
	}
	return &Client{cfg: cfg, httpClient: hc}, nil
}

func (c *Client) Name() string { return "openai-responses" }

func init() {
	protocol.RegisterFactory(protocol.ProviderOpenAIResponse, func(cfg protocol.ClientConfig) (protocol.ChatModel, error) {
		return NewClient(Config{
			APIKey:       cfg.APIKey,
			BaseURL:      cfg.BaseURL,
			Organization: cfg.Organization,
			Project:      cfg.Project,
		})
	})
}

// Chat via OpenAI-compatible /chat/completions (non-stream).
func (c *Client) Chat(ctx context.Context, req protocol.ChatRequest) (*protocol.ChatResponse, error) {
	start := time.Now()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	payload := map[string]any{
		"model":    req.Model,
		"messages": toResponsesMessages(req.Messages),
	}
	if req.MaxTokens > 0 {
		payload["max_tokens"] = req.MaxTokens
	}
	if req.Temperature != 0 {
		payload["temperature"] = req.Temperature
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal responses payload: %w", err)
	}

	endpoint := fmt.Sprintf("%s/chat/completions", strings.TrimSuffix(c.cfg.BaseURL, "/"))
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build responses request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	if c.cfg.Organization != "" {
		httpReq.Header.Set("OpenAI-Organization", c.cfg.Organization)
	}
	if c.cfg.Project != "" {
		httpReq.Header.Set("OpenAI-Project", c.cfg.Project)
	}

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call responses: %w", err)
	}
	defer httpResp.Body.Close()

	respBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read responses: %w", err)
	}
	if httpResp.StatusCode >= 300 {
		return nil, fmt.Errorf("responses http %d: %s", httpResp.StatusCode, string(respBytes))
	}

	var raw struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(respBytes, &raw); err != nil {
		return nil, fmt.Errorf("decode responses: %w", err)
	}

	text := ""
	if len(raw.Choices) > 0 {
		text = raw.Choices[0].Message.Content
	}
	chatResp := &protocol.ChatResponse{
		ID:        raw.ID,
		Model:     raw.Model,
		CreatedAt: time.Now(),
		Choices: []protocol.Choice{{
			Index:        0,
			Message:      protocol.Message{Role: protocol.RoleAssistant, Content: text},
			FinishReason: "stop",
		}},
		Usage: protocol.TokenUsage{
			PromptTokens:     raw.Usage.PromptTokens,
			CompletionTokens: raw.Usage.CompletionTokens,
			TotalTokens:      raw.Usage.TotalTokens,
		},
	}
	now := time.Now()
	chatResp.Metrics = metrics.CallMetrics{
		Provider:      c.Name(),
		Model:         chatResp.Model,
		StartAt:       start,
		FirstAt:       now,
		EndAt:         now,
		Bytes:         len(respBytes),
		Chunks:        1,
		RequestBytes:  len(body),
		ResponseBytes: len(respBytes),
		HTTPStatus:    httpResp.StatusCode,
	}
	return chatResp, nil
}

// StreamChat uses SSE from /v1/responses with stream:true.
func (c *Client) StreamChat(ctx context.Context, req protocol.ChatRequest) (protocol.ChatStream, error) {
	start := time.Now()
	if err := req.Validate(); err != nil {
		return nil, err
	}

	payload := map[string]any{
		"model":    req.Model,
		"messages": toResponsesMessages(req.Messages),
		"stream":   true,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal responses payload: %w", err)
	}

	endpoint := fmt.Sprintf("%s/chat/completions", strings.TrimSuffix(c.cfg.BaseURL, "/"))
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build responses request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if c.cfg.Organization != "" {
		httpReq.Header.Set("OpenAI-Organization", c.cfg.Organization)
	}
	if c.cfg.Project != "" {
		httpReq.Header.Set("OpenAI-Project", c.cfg.Project)
	}

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call responses: %w", err)
	}
	if httpResp.StatusCode >= 300 {
		b, _ := io.ReadAll(httpResp.Body)
		httpResp.Body.Close()
		return nil, fmt.Errorf("responses http %d: %s", httpResp.StatusCode, string(b))
	}

	return &responsesStream{
		body:         httpResp.Body,
		startAt:      start,
		model:        req.Model,
		httpStatus:   httpResp.StatusCode,
		requestBytes: len(body),
	}, nil
}

type responsesStream struct {
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

func (s *responsesStream) Recv() (*protocol.ChatStreamChunk, error) {
	// SSE lines: data: {"choices":[{"delta":{"content":"..."}}]}
	for {
		line, err := s.readLine()
		if err != nil {
			if err == io.EOF {
				s.endAt = time.Now()
			}
			return nil, err
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if payload == "[DONE]" {
			s.endAt = time.Now()
			return nil, io.EOF
		}
		s.bytes += len(payload)
		s.responseBytes += len(payload)
		var raw struct {
			Choices []struct {
				Index int `json:"index"`
				Delta struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
			Usage struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(payload), &raw); err != nil {
			return nil, fmt.Errorf("decode responses stream: %w", err)
		}
		if s.firstAt.IsZero() {
			s.firstAt = time.Now()
		}
		s.chunks++
		if raw.Usage.TotalTokens > 0 {
			s.usage = protocol.TokenUsage{
				PromptTokens:     raw.Usage.PromptTokens,
				CompletionTokens: raw.Usage.CompletionTokens,
				TotalTokens:      raw.Usage.TotalTokens,
			}
		}
		if len(raw.Choices) == 0 || raw.Choices[0].Delta.Content == "" {
			continue
		}
		ch := raw.Choices[0]
		return &protocol.ChatStreamChunk{
			Index:        ch.Index,
			Role:         protocol.MessageRole(ch.Delta.Role),
			Delta:        ch.Delta.Content,
			FinishReason: ch.FinishReason,
		}, nil
	}
}

func (s *responsesStream) readLine() (string, error) {
	var line strings.Builder
	buf := make([]byte, 1)
	for {
		n, err := s.body.Read(buf)
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

func (s *responsesStream) Close() error {
	if s.body != nil {
		return s.body.Close()
	}
	return nil
}

func (s *responsesStream) Metrics() metrics.CallMetrics {
	return metrics.CallMetrics{
		Provider:         "openai-responses",
		Model:            s.model,
		StartAt:          s.startAt,
		FirstAt:          s.firstAt,
		EndAt:            s.endAt,
		Bytes:            s.bytes,
		Chunks:           s.chunks,
		RequestBytes:     s.requestBytes,
		ResponseBytes:    s.responseBytes,
		HTTPStatus:       s.httpStatus,
		PromptTokens:     s.usage.PromptTokens,
		CompletionTokens: s.usage.CompletionTokens,
		TotalTokens:      s.usage.TotalTokens,
	}
}

type responsesMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func toResponsesMessages(msgs []protocol.Message) []responsesMessage {
	out := make([]responsesMessage, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, responsesMessage{Role: string(m.Role), Content: m.Content})
	}
	return out
}
