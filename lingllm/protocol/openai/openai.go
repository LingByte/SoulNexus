package openai

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

// Config configures the OpenAI-compatible chat client.
type Config struct {
	APIKey       string
	BaseURL      string
	HTTPClient   *http.Client
	Organization string
	Project      string
}

// Client implements llm.ChatModel for OpenAI's /chat/completions endpoint.
type Client struct {
	cfg        Config
	httpClient *http.Client
}

// NewClient constructs a client with sane defaults.
func NewClient(cfg Config) (*Client, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("openai api key is required")
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

func (c *Client) Name() string { return "openai" }

func init() {
	protocol.RegisterFactory(protocol.ProviderOpenAI, func(cfg protocol.ClientConfig) (protocol.ChatModel, error) {
		return NewClient(Config{
			APIKey:       cfg.APIKey,
			BaseURL:      cfg.BaseURL,
			Organization: cfg.Organization,
			Project:      cfg.Project,
		})
	})
}

// Chat executes a chat completion request against OpenAI.
func (c *Client) Chat(ctx context.Context, req protocol.ChatRequest) (*protocol.ChatResponse, error) {
	start := time.Now()
	if err := req.Validate(); err != nil {
		return nil, err
	}

	payload := map[string]any{
		"model":       req.Model,
		"messages":    toOpenAIMessages(req.Messages),
		"max_tokens":  req.MaxTokens,
		"temperature": req.Temperature,
		"top_p":       req.TopP,
		"stop":        req.Stop,
	}
	if req.MaxTokens == 0 {
		delete(payload, "max_tokens")
	}
	if req.Temperature == 0 {
		delete(payload, "temperature")
	}
	if req.TopP == 0 {
		delete(payload, "top_p")
	}
	if len(req.Stop) == 0 {
		delete(payload, "stop")
	}
	if len(req.Tools) > 0 {
		payload["tools"] = toOpenAITools(req.Tools)
		if tc := openAIToolChoice(req.ToolChoice); tc != nil {
			payload["tool_choice"] = tc
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal openai payload: %w", err)
	}

	endpoint := fmt.Sprintf("%s/chat/completions", c.cfg.BaseURL)
	reqHTTP, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build openai request: %w", err)
	}
	reqHTTP.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	reqHTTP.Header.Set("Content-Type", "application/json")
	if c.cfg.Organization != "" {
		reqHTTP.Header.Set("OpenAI-Organization", c.cfg.Organization)
	}
	if c.cfg.Project != "" {
		reqHTTP.Header.Set("OpenAI-Project", c.cfg.Project)
	}

	httpResp, err := c.httpClient.Do(reqHTTP)
	if err != nil {
		return nil, fmt.Errorf("call openai: %w", err)
	}
	defer httpResp.Body.Close()

	bodyBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read openai response: %w", err)
	}
	if httpResp.StatusCode >= 300 {
		return nil, fmt.Errorf("openai http %d: %s", httpResp.StatusCode, string(bodyBytes))
	}

	var raw openAIResponse
	if err := json.Unmarshal(bodyBytes, &raw); err != nil {
		return nil, fmt.Errorf("decode openai response: %w", err)
	}

	chatResp := raw.toChatResponse()
	now := time.Now()
	chatResp.Metrics = metrics.CallMetrics{
		Provider:         c.Name(),
		Model:            chatResp.Model,
		StartAt:          start,
		EndAt:            now,
		FirstAt:          now,
		PromptTokens:     chatResp.Usage.PromptTokens,
		CompletionTokens: chatResp.Usage.CompletionTokens,
		TotalTokens:      chatResp.Usage.TotalTokens,
		Chunks:           1,
		Bytes:            len(bodyBytes),
		RequestBytes:     len(body),
		ResponseBytes:    len(bodyBytes),
		HTTPStatus:       httpResp.StatusCode,
	}
	return chatResp, nil
}

// StreamChat uses SSE stream from /chat/completions with stream=true.
func (c *Client) StreamChat(ctx context.Context, req protocol.ChatRequest) (protocol.ChatStream, error) {
	start := time.Now()
	if err := req.Validate(); err != nil {
		return nil, err
	}

	payload := struct {
		Model       string          `json:"model"`
		Messages    []openAIMessage `json:"messages"`
		MaxTokens   int             `json:"max_tokens,omitempty"`
		Temperature float32         `json:"temperature,omitempty"`
		TopP        float32         `json:"top_p,omitempty"`
		Stop        []string        `json:"stop,omitempty"`
		Stream      bool            `json:"stream"`
	}{
		Model:       req.Model,
		Messages:    toOpenAIMessages(req.Messages),
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stop:        req.Stop,
		Stream:      true,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal openai payload: %w", err)
	}

	endpoint := fmt.Sprintf("%s/chat/completions", c.cfg.BaseURL)
	reqHTTP, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build openai request: %w", err)
	}
	reqHTTP.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	reqHTTP.Header.Set("Content-Type", "application/json")
	reqHTTP.Header.Set("Accept", "text/event-stream")
	if c.cfg.Organization != "" {
		reqHTTP.Header.Set("OpenAI-Organization", c.cfg.Organization)
	}
	if c.cfg.Project != "" {
		reqHTTP.Header.Set("OpenAI-Project", c.cfg.Project)
	}

	httpResp, err := c.httpClient.Do(reqHTTP)
	if err != nil {
		return nil, fmt.Errorf("call openai: %w", err)
	}

	if httpResp.StatusCode >= 300 {
		b, _ := io.ReadAll(httpResp.Body)
		httpResp.Body.Close()
		return nil, fmt.Errorf("openai http %d: %s", httpResp.StatusCode, string(b))
	}

	stream := &openAIStream{
		body:         httpResp.Body,
		startAt:      start,
		model:        req.Model,
		httpStatus:   httpResp.StatusCode,
		requestBytes: len(body),
	}
	return stream, nil
}

type openAIToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type openAIFunctionTool struct {
	Type     string           `json:"type"`
	Function openAIFunctionDef `json:"function"`
}

type openAIFunctionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

type openAIMessage struct {
	Role       string           `json:"role"`
	Content    *string          `json:"content"`
	Name       string           `json:"name,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
}

type openAIChoice struct {
	Index        int           `json:"index"`
	Message      openAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type openAIResponse struct {
	ID      string         `json:"id"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []openAIChoice `json:"choices"`
	Usage   openAIUsage    `json:"usage"`
}

// openAIStream implements llm.ChatStream for SSE responses.
type openAIStream struct {
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

func (s *openAIStream) Recv() (*protocol.ChatStreamChunk, error) {
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
		var raw struct {
			Choices []struct {
				Index int `json:"index"`
				Delta struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
			Usage openAIUsage `json:"usage"`
		}
		if err := json.Unmarshal([]byte(payload), &raw); err != nil {
			return nil, fmt.Errorf("decode openai stream chunk: %w", err)
		}
		if s.firstAt.IsZero() {
			s.firstAt = time.Now()
		}
		s.chunks++
		s.bytes += len(payload)
		s.responseBytes += len(payload)
		if raw.Usage.TotalTokens > 0 {
			s.usage = protocol.TokenUsage{
				PromptTokens:     raw.Usage.PromptTokens,
				CompletionTokens: raw.Usage.CompletionTokens,
				TotalTokens:      raw.Usage.TotalTokens,
			}
		}
		if len(raw.Choices) == 0 {
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

func (s *openAIStream) readLine() (string, error) {
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

func (s *openAIStream) Close() error {
	if s.body != nil {
		return s.body.Close()
	}
	return nil
}

func (s *openAIStream) Metrics() metrics.CallMetrics {
	return metrics.CallMetrics{
		Provider:         "openai",
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

func toOpenAITools(tools []protocol.Tool) []openAIFunctionTool {
	out := make([]openAIFunctionTool, 0, len(tools))
	for _, tool := range tools {
		out = append(out, openAIFunctionTool{
			Type: "function",
			Function: openAIFunctionDef{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			},
		})
	}
	return out
}

func openAIToolChoice(tc protocol.ToolChoice) any {
	switch tc {
	case protocol.ToolChoiceRequired:
		return "required"
	case protocol.ToolChoiceNone:
		return "none"
	case protocol.ToolChoiceAuto:
		return "auto"
	default:
		if strings.TrimSpace(string(tc)) != "" {
			return string(tc)
		}
		return nil
	}
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func toOpenAIMessages(msgs []protocol.Message) []openAIMessage {
	out := make([]openAIMessage, 0, len(msgs))
	for _, m := range msgs {
		role := string(m.Role)
		switch m.Role {
		case protocol.RoleTool:
			role = "tool"
		}
		om := openAIMessage{
			Role:       role,
			ToolCallID: m.ToolCallID,
		}
		if len(m.ToolCalls) > 0 {
			om.Content = strPtr(m.Content)
			om.ToolCalls = make([]openAIToolCall, 0, len(m.ToolCalls))
			for _, tc := range m.ToolCalls {
				call := openAIToolCall{
					ID:   tc.ID,
					Type: tc.Type,
				}
				if call.Type == "" {
					call.Type = "function"
				}
				call.Function.Name = tc.Function.Name
				call.Function.Arguments = string(tc.Function.Arguments)
				om.ToolCalls = append(om.ToolCalls, call)
			}
		} else {
			om.Content = strPtr(m.Content)
		}
		out = append(out, om)
	}
	return out
}

func (r openAIResponse) toChatResponse() *protocol.ChatResponse {
	choices := make([]protocol.Choice, 0, len(r.Choices))
	for _, ch := range r.Choices {
		msg := protocol.Message{
			Role:    protocol.MessageRole(ch.Message.Role),
			Content: derefString(ch.Message.Content),
		}
		if len(ch.Message.ToolCalls) > 0 {
			msg.ToolCalls = make([]protocol.ToolCall, 0, len(ch.Message.ToolCalls))
			for _, tc := range ch.Message.ToolCalls {
				typ := tc.Type
				if typ == "" {
					typ = "function"
				}
				msg.ToolCalls = append(msg.ToolCalls, protocol.ToolCall{
					ID:   tc.ID,
					Type: typ,
					Function: protocol.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: json.RawMessage(tc.Function.Arguments),
					},
				})
			}
		}
		choices = append(choices, protocol.Choice{
			Index:        ch.Index,
			Message:      msg,
			FinishReason: ch.FinishReason,
		})
	}
	return &protocol.ChatResponse{
		ID:        r.ID,
		Model:     r.Model,
		CreatedAt: time.Unix(r.Created, 0),
		Choices:   choices,
		Usage: protocol.TokenUsage{
			PromptTokens:     r.Usage.PromptTokens,
			CompletionTokens: r.Usage.CompletionTokens,
			TotalTokens:      r.Usage.TotalTokens,
		},
	}
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
