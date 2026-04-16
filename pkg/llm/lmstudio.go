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

type LMStudioHandler struct {
	ctx          context.Context
	baseURL      string
	apiKey       string
	systemPrompt string
	interruptCh  chan struct{}
	client       *http.Client
	toolManager  *FunctionToolManager
}

func NewLMStudioHandler(ctx context.Context, llmOptions *LLMOptions) (*LMStudioHandler, error) {
	var opts LLMOptions
	if llmOptions != nil {
		opts = *llmOptions
	}
	if strings.TrimSpace(opts.BaseURL) == "" {
		opts.BaseURL = "http://localhost:1234/v1"
	}
	if strings.TrimSpace(opts.ApiKey) == "" {
		opts.ApiKey = "lmstudio"
	}
	return &LMStudioHandler{
		ctx:          ctx,
		baseURL:      strings.TrimRight(strings.TrimSpace(opts.BaseURL), "/"),
		apiKey:       strings.TrimSpace(opts.ApiKey),
		systemPrompt: opts.SystemPrompt,
		interruptCh:  make(chan struct{}, 1),
		client:       &http.Client{Timeout: 120 * time.Second},
		toolManager:  NewFunctionToolManager(),
	}, nil
}

func (h *LMStudioHandler) Query(text, model string) (string, error) {
	resp, err := h.QueryWithOptions(text, &QueryOptions{Model: model})
	if err != nil {
		return "", err
	}
	if resp == nil || len(resp.Choices) == 0 {
		return "", errors.New("empty response")
	}
	return resp.Choices[0].Content, nil
}

func (h *LMStudioHandler) QueryWithOptions(text string, options *QueryOptions) (*QueryResponse, error) {
	if options == nil {
		options = &QueryOptions{}
	}
	requestType := strings.TrimSpace(options.RequestType)
	if requestType == "" {
		requestType = "query"
	}
	model := strings.TrimSpace(options.Model)
	if model == "" {
		model = "llama3.1"
	}

	tracker := NewLLMRequestTracker(
		options.SessionID,
		options.UserID,
		"lmstudio",
		model,
		h.baseURL,
		requestType,
	)
	tracker.SetRequestContent(text)
	tracker.SetUserAgent(options.UserAgent)
	tracker.SetIPAddress(options.IPAddress)
	tracker.SetStatusCode(200)
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
		rw, err := h.rewriteQueryChatCompletions(reqCtx, before, options)
		if err == nil && rw != "" {
			rewrite = &QueryRewrite{Original: before, Rewritten: rw}
			text = rw
		}
	}

	var expansion *QueryExpansion
	if options.EnableQueryExpansion {
		expanded, terms, err := h.expandQueryChatCompletions(reqCtx, text, options)
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

	body := map[string]any{
		"model":       model,
		"messages":    chatMessagesToMap(buildShortTermMessages(text, options)),
		"stream":      false,
		"temperature": options.Temperature,
	}
	raw, err := h.doChatCompletion(reqCtx, body)
	if err != nil {
		tracker.Error("LMSTUDIO_REQUEST_ERROR", err.Error())
		return nil, err
	}
	var parsed struct {
		Choices []struct {
			Message struct {
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
	if err := json.Unmarshal(raw, &parsed); err != nil {
		tracker.Error("JSON_PARSE_ERROR", err.Error())
		return nil, err
	}
	content := ""
	reason := "stop"
	if len(parsed.Choices) > 0 {
		content = strings.TrimSpace(parsed.Choices[0].Message.Content)
		if parsed.Choices[0].FinishReason != "" {
			reason = parsed.Choices[0].FinishReason
		}
	}
	resp := &QueryResponse{
		Provider:  h.Provider(),
		Model:     model,
		Choices:   []QueryChoice{{Index: 0, Content: content, FinishReason: reason}},
		Expansion: expansion,
		Rewrite:   rewrite,
		Usage: &TokenUsage{
			PromptTokens:     parsed.Usage.PromptTokens,
			CompletionTokens: parsed.Usage.CompletionTokens,
			TotalTokens:      parsed.Usage.TotalTokens,
		},
	}

	tracker.Complete(resp)
	return resp, nil
}

func (h *LMStudioHandler) QueryStream(text string, options *QueryOptions, callback func(segment string, isComplete bool) error) (*QueryResponse, error) {
	if options == nil {
		options = &QueryOptions{}
	}
	requestType := strings.TrimSpace(options.RequestType)
	if requestType == "" {
		requestType = "stream"
	}
	model := strings.TrimSpace(options.Model)
	if model == "" {
		model = "local-model"
	}

	tracker := NewLLMRequestTracker(
		options.SessionID,
		options.UserID,
		"lmstudio",
		model,
		h.baseURL,
		requestType,
	)
	tracker.SetRequestContent(text)
	tracker.SetUserAgent(options.UserAgent)
	tracker.SetIPAddress(options.IPAddress)
	tracker.SetStatusCode(200)
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
		rw, err := h.rewriteQueryChatCompletions(reqCtx, before, options)
		if err == nil && rw != "" {
			streamRewrite = &QueryRewrite{Original: before, Rewritten: rw}
			text = rw
		}
	}
	var streamExpansion *QueryExpansion
	if options.EnableQueryExpansion {
		expanded, terms, err := h.expandQueryChatCompletions(reqCtx, text, options)
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

	body := map[string]any{
		"model":       model,
		"messages":    chatMessagesToMap(buildShortTermMessages(text, options)),
		"stream":      true,
		"temperature": options.Temperature,
	}
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, h.baseURL+"/chat/completions", bytes.NewReader(b))
	if err != nil {
		tracker.Error("LMSTUDIO_REQUEST_ERROR", err.Error())
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if h.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+h.apiKey)
	}
	resp, err := h.client.Do(req)
	if err != nil {
		tracker.Error("LMSTUDIO_REQUEST_ERROR", err.Error())
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		rb, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("lmstudio stream failed: status=%d body=%s", resp.StatusCode, string(rb))
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
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if json.Unmarshal([]byte(payload), &chunk) != nil || len(chunk.Choices) == 0 {
			continue
		}
		seg := chunk.Choices[0].Delta.Content
		if seg == "" {
			continue
		}
		out.WriteString(seg)
		if callback != nil {
			if err := callback(seg, false); err != nil {
				return nil, err
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
	content := strings.TrimSpace(out.String())
	return &QueryResponse{
		Provider:  h.Provider(),
		Model:     model,
		Choices:   []QueryChoice{{Index: 0, Content: content, FinishReason: "stop"}},
		Rewrite:   streamRewrite,
		Expansion: streamExpansion,
	}, nil
}

func (h *LMStudioHandler) Provider() string { return LLM_LMSTUDIO }

func (h *LMStudioHandler) Interrupt() {
	select {
	case h.interruptCh <- struct{}{}:
	default:
	}
}

func (h *LMStudioHandler) RegisterFunctionTool(name, description string, parameters interface{}, callback FunctionToolCallback) {
	if h.toolManager == nil {
		h.toolManager = NewFunctionToolManager()
	}
	h.toolManager.RegisterTool(name, description, parameters, callback)
}

func (h *LMStudioHandler) RegisterFunctionToolDefinition(def *FunctionToolDefinition) {
	if h.toolManager == nil {
		h.toolManager = NewFunctionToolManager()
	}
	h.toolManager.RegisterToolDefinition(def)
}

func (h *LMStudioHandler) GetFunctionTools() []interface{} {
	if h.toolManager == nil {
		return nil
	}
	return h.toolManager.GetTools()
}

func (h *LMStudioHandler) ListFunctionTools() []string {
	if h.toolManager == nil {
		return nil
	}
	return h.toolManager.ListTools()
}

func (h *LMStudioHandler) Hangup() {
	h.Interrupt()
}

func (h *LMStudioHandler) rewriteQueryChatCompletions(ctx context.Context, text string, options *QueryOptions) (string, error) {
	if options == nil {
		options = &QueryOptions{}
	}
	model := queryRewriteModel(options, "local-model")
	if model == "" {
		model = "local-model"
	}
	prompt := BuildQueryRewriteUserPrompt(text, options.QueryRewriteInstruction)
	body := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"stream":      false,
		"temperature": 0.2,
		"max_tokens":  128,
	}
	raw, err := h.doChatCompletion(ctx, body)
	if err != nil {
		return "", err
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", err
	}
	out := ""
	if len(parsed.Choices) > 0 {
		out = strings.TrimSpace(parsed.Choices[0].Message.Content)
	}
	out = NormalizeRewrittenQuery(out)
	if out == "" {
		return strings.TrimSpace(text), nil
	}
	return out, nil
}

func (h *LMStudioHandler) expandQueryChatCompletions(ctx context.Context, text string, options *QueryOptions) (string, []string, error) {
	if options == nil {
		options = &QueryOptions{}
	}
	maxTerms := expansionMaxTerms(options)
	sep := expansionSeparator(options)
	model := strings.TrimSpace(options.Model)
	if model == "" {
		model = "local-model"
	}
	prompt := BuildQueryExpansionUserPrompt(text, maxTerms)
	body := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"stream":      false,
		"temperature": 0.2,
		"max_tokens":  256,
	}
	raw, err := h.doChatCompletion(ctx, body)
	if err != nil {
		return "", nil, err
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", nil, err
	}
	out := ""
	if len(parsed.Choices) > 0 {
		out = strings.TrimSpace(parsed.Choices[0].Message.Content)
	}
	expanded, terms := ExpandedQueryFromModelAnswer(text, out, maxTerms, sep)
	return expanded, terms, nil
}

func (h *LMStudioHandler) doChatCompletion(ctx context.Context, body map[string]any) ([]byte, error) {
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.baseURL+"/chat/completions", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if h.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+h.apiKey)
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("lmstudio request failed: status=%d body=%s", resp.StatusCode, string(raw))
	}
	return raw, nil
}
