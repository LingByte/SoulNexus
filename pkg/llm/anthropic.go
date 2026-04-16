package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type AnthropicHandler struct {
	ctx          context.Context
	baseURL      string
	apiKey       string
	systemPrompt string
	interruptCh  chan struct{}
	client       *http.Client
	toolManager  *FunctionToolManager
}

func NewAnthropicHandler(ctx context.Context, llmOptions *LLMOptions) (*AnthropicHandler, error) {
	var opts LLMOptions
	if llmOptions != nil {
		opts = *llmOptions
	}
	if strings.TrimSpace(opts.BaseURL) == "" {
		opts.BaseURL = "https://api.anthropic.com"
	}
	return &AnthropicHandler{
		ctx:          ctx,
		baseURL:      strings.TrimRight(strings.TrimSpace(opts.BaseURL), "/"),
		apiKey:       strings.TrimSpace(opts.ApiKey),
		systemPrompt: opts.SystemPrompt,
		interruptCh:  make(chan struct{}, 1),
		client:       &http.Client{Timeout: 120 * time.Second},
		toolManager:  NewFunctionToolManager(),
	}, nil
}

func (h *AnthropicHandler) Query(text, model string) (string, error) {
	resp, err := h.QueryWithOptions(text, &QueryOptions{Model: model})
	if err != nil {
		return "", err
	}
	if resp == nil || len(resp.Choices) == 0 {
		return "", errors.New("empty response")
	}
	return resp.Choices[0].Content, nil
}

func (h *AnthropicHandler) QueryWithOptions(text string, options *QueryOptions) (*QueryResponse, error) {
	if options == nil {
		options = &QueryOptions{}
	}
	requestType := strings.TrimSpace(options.RequestType)
	if requestType == "" {
		requestType = "query"
	}
	model := strings.TrimSpace(options.Model)
	if model == "" {
		model = "claude-3-sonnet-20240229"
	}

	tracker := NewLLMRequestTracker(
		options.SessionID,
		options.UserID,
		"anthropic",
		model,
		h.baseURL,
		requestType,
	)
	reqCtx, cancel := context.WithCancel(h.ctx)
	defer cancel()
	go func() {
		select {
		case <-h.interruptCh:
			cancel()
		case <-reqCtx.Done():
		}
	}()

	var rewrite *QueryRewrite
	if options.EnableQueryRewrite {
		before := text
		rw, err := h.rewriteQueryAnthropic(reqCtx, before, options)
		if err == nil && rw != "" {
			rewrite = &QueryRewrite{Original: before, Rewritten: rw}
			text = rw
		}
	}

	var expansion *QueryExpansion
	if options.EnableQueryExpansion {
		expanded, terms, err := h.expandQueryAnthropic(reqCtx, text, options)
		if err == nil {
			expansion = &QueryExpansion{
				Original: text,
				Expanded: expanded,
				Terms:    terms,
				Debug:    map[string]any{},
			}
			text = expanded
		}
	}

	chatMessages := buildShortTermMessages(text, options)
	userMsgs := chatMessagesToAnthropic(chatMessages)
	reqBody := map[string]any{
		"model":       model,
		"max_tokens":  max(256, options.MaxTokens),
		"temperature": options.Temperature,
		"messages":    userMsgs,
	}
	if sys := appendEmotionalStyle(mergedSystemPrompt(h.systemPrompt, chatMessages), options); strings.TrimSpace(sys) != "" {
		reqBody["system"] = sys
	}
	raw, err := h.doAnthropic(reqCtx, reqBody)
	if err != nil {
		tracker.Error("ANTHROPIC_REQUEST_ERROR", err.Error())
		return nil, err
	}
	var parsed struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
		Usage      struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		tracker.Error("JSON_PARSE_ERROR", err.Error())
		return nil, err
	}
	var b strings.Builder
	for _, c := range parsed.Content {
		if c.Type == "text" {
			b.WriteString(c.Text)
		}
	}
	answer := strings.TrimSpace(b.String())
	reason := parsed.StopReason
	if reason == "" {
		reason = "stop"
	}
	resp := &QueryResponse{
		Provider:  h.Provider(),
		Model:     model,
		Choices:   []QueryChoice{{Index: 0, Content: answer, FinishReason: reason}},
		Expansion: expansion,
		Rewrite:   rewrite,
		Usage: &TokenUsage{
			PromptTokens:     parsed.Usage.InputTokens,
			CompletionTokens: parsed.Usage.OutputTokens,
			TotalTokens:      parsed.Usage.InputTokens + parsed.Usage.OutputTokens,
		},
	}

	tracker.Complete(resp)
	return resp, nil
}

func (h *AnthropicHandler) QueryStream(text string, options *QueryOptions, callback func(segment string, isComplete bool) error) (*QueryResponse, error) {
	if options == nil {
		options = &QueryOptions{}
	}
	model := strings.TrimSpace(options.Model)
	if model == "" {
		model = "claude-3-5-sonnet-20241022"
	}
	reqCtx, cancel := context.WithCancel(h.ctx)
	defer cancel()
	go func() {
		select {
		case <-h.interruptCh:
			cancel()
		case <-reqCtx.Done():
		}
	}()

	var streamRewrite *QueryRewrite
	if options.EnableQueryRewrite {
		before := text
		rw, err := h.rewriteQueryAnthropic(reqCtx, before, options)
		if err == nil && rw != "" {
			streamRewrite = &QueryRewrite{Original: before, Rewritten: rw}
			text = rw
		}
	}
	var streamExpansion *QueryExpansion
	if options.EnableQueryExpansion {
		expanded, terms, err := h.expandQueryAnthropic(reqCtx, text, options)
		if err == nil {
			streamExpansion = &QueryExpansion{
				Original: text,
				Expanded: expanded,
				Terms:    terms,
				Debug:    map[string]any{},
			}
			text = expanded
		}
	}

	chatMessages := buildShortTermMessages(text, options)
	reqBody := map[string]any{
		"model":       model,
		"max_tokens":  max(256, options.MaxTokens),
		"temperature": options.Temperature,
		"messages":    chatMessagesToAnthropic(chatMessages),
		"stream":      true,
	}
	if sys := appendEmotionalStyle(mergedSystemPrompt(h.systemPrompt, chatMessages), options); strings.TrimSpace(sys) != "" {
		reqBody["system"] = sys
	}
	b, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, h.baseURL+"/messages", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", h.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		rb, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("anthropic stream failed: status=%d body=%s", resp.StatusCode, string(rb))
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
	var out strings.Builder
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			break
		}
		var evt struct {
			Type  string `json:"type"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
		}
		if json.Unmarshal([]byte(payload), &evt) != nil {
			continue
		}
		if evt.Type == "content_block_delta" && evt.Delta.Type == "text_delta" && evt.Delta.Text != "" {
			out.WriteString(evt.Delta.Text)
			if callback != nil {
				if err := callback(evt.Delta.Text, false); err != nil {
					return nil, err
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if callback != nil {
		if err := callback("", true); err != nil {
			return nil, err
		}
	}
	answer := strings.TrimSpace(out.String())
	return &QueryResponse{
		Provider:  h.Provider(),
		Model:     model,
		Choices:   []QueryChoice{{Index: 0, Content: answer, FinishReason: "stop"}},
		Rewrite:   streamRewrite,
		Expansion: streamExpansion,
	}, nil
}

func (h *AnthropicHandler) Provider() string { return LLM_ANTHROPIC }

func (h *AnthropicHandler) Interrupt() {
	select {
	case h.interruptCh <- struct{}{}:
	default:
	}
}

func (h *AnthropicHandler) RegisterFunctionTool(name, description string, parameters interface{}, callback FunctionToolCallback) {
	if h.toolManager == nil {
		h.toolManager = NewFunctionToolManager()
	}
	h.toolManager.RegisterTool(name, description, parameters, callback)
}

func (h *AnthropicHandler) RegisterFunctionToolDefinition(def *FunctionToolDefinition) {
	if h.toolManager == nil {
		h.toolManager = NewFunctionToolManager()
	}
	h.toolManager.RegisterToolDefinition(def)
}

func (h *AnthropicHandler) GetFunctionTools() []interface{} {
	if h.toolManager == nil {
		return nil
	}
	return h.toolManager.GetTools()
}

func (h *AnthropicHandler) ListFunctionTools() []string {
	if h.toolManager == nil {
		return nil
	}
	return h.toolManager.ListTools()
}

func (h *AnthropicHandler) Hangup() {
	h.Interrupt()
}

func (h *AnthropicHandler) rewriteQueryAnthropic(ctx context.Context, text string, options *QueryOptions) (string, error) {
	if options == nil {
		options = &QueryOptions{}
	}
	model := queryRewriteModel(options, "claude-3-5-sonnet-20241022")
	if model == "" {
		model = "claude-3-5-sonnet-20241022"
	}
	prompt := BuildQueryRewriteUserPrompt(text, options.QueryRewriteInstruction)
	body := map[string]any{
		"model":       model,
		"max_tokens":  128,
		"temperature": 0.2,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]string{
					{"type": "text", "text": prompt},
				},
			},
		},
	}
	raw, err := h.doAnthropic(ctx, body)
	if err != nil {
		return "", err
	}
	var parsed struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", err
	}
	var b strings.Builder
	for _, c := range parsed.Content {
		if c.Type == "text" {
			b.WriteString(c.Text)
		}
	}
	out := NormalizeRewrittenQuery(b.String())
	if out == "" {
		return strings.TrimSpace(text), nil
	}
	return out, nil
}

func (h *AnthropicHandler) expandQueryAnthropic(ctx context.Context, text string, options *QueryOptions) (string, []string, error) {
	if options == nil {
		options = &QueryOptions{}
	}
	maxTerms := expansionMaxTerms(options)
	sep := expansionSeparator(options)
	model := strings.TrimSpace(options.Model)
	if model == "" {
		model = "claude-3-5-sonnet-20241022"
	}
	prompt := BuildQueryExpansionUserPrompt(text, maxTerms)
	body := map[string]any{
		"model":       model,
		"max_tokens":  256,
		"temperature": 0.2,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]string{
					{"type": "text", "text": prompt},
				},
			},
		},
	}
	raw, err := h.doAnthropic(ctx, body)
	if err != nil {
		return "", nil, err
	}
	var parsed struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", nil, err
	}
	var b strings.Builder
	for _, c := range parsed.Content {
		if c.Type == "text" {
			b.WriteString(c.Text)
		}
	}
	out := strings.TrimSpace(b.String())
	expanded, terms := ExpandedQueryFromModelAnswer(text, out, maxTerms, sep)
	return expanded, terms, nil
}

func (h *AnthropicHandler) doAnthropic(ctx context.Context, body map[string]any) ([]byte, error) {
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.baseURL+"/messages", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", h.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("anthropic request failed: status=%d body=%s", resp.StatusCode, string(raw))
	}
	return raw, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
