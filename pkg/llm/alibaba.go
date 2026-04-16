package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type AlibabaHandler struct {
	ctx          context.Context
	apiKey       string
	appID        string
	endpoint     string
	systemPrompt string
	client       *http.Client
	interruptCh  chan struct{}
	toolManager  *FunctionToolManager
}

func NewAlibabaHandler(ctx context.Context, llmOptions *LLMOptions) (*AlibabaHandler, error) {
	var opts LLMOptions
	if llmOptions != nil {
		opts = *llmOptions
	}
	timeout := 30 * time.Second
	if s := strings.TrimSpace(os.Getenv("ALIBABA_AI_TIMEOUT_SECONDS")); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			timeout = time.Duration(n) * time.Second
		}
	}
	endpoint := strings.TrimSpace(opts.BaseURL)
	if endpoint == "" {
		endpoint = "https://dashscope.aliyuncs.com"
	}
	appID := strings.TrimSpace(opts.BaseURL)
	if strings.Contains(endpoint, "://") {
		appID = strings.TrimSpace(os.Getenv("ALIBABA_APP_ID"))
	}
	if appID == "" {
		appID = strings.TrimSpace(os.Getenv("ALIBABA_APP_ID"))
	}
	if appID == "" {
		return nil, errors.New("alibaba app id is required (set LLM BaseURL as app id or ALIBABA_APP_ID)")
	}
	return &AlibabaHandler{
		ctx:          ctx,
		apiKey:       strings.TrimSpace(opts.ApiKey),
		appID:        appID,
		endpoint:     endpoint,
		systemPrompt: opts.SystemPrompt,
		client:       &http.Client{Timeout: timeout},
		interruptCh:  make(chan struct{}, 1),
		toolManager:  NewFunctionToolManager(),
	}, nil
}

func (h *AlibabaHandler) Interrupt() {
	select {
	case h.interruptCh <- struct{}{}:
	default:
	}
}

func (h *AlibabaHandler) RegisterFunctionTool(name, description string, parameters interface{}, callback FunctionToolCallback) {
	if h.toolManager == nil {
		h.toolManager = NewFunctionToolManager()
	}
	h.toolManager.RegisterTool(name, description, parameters, callback)
}

func (h *AlibabaHandler) RegisterFunctionToolDefinition(def *FunctionToolDefinition) {
	if h.toolManager == nil {
		h.toolManager = NewFunctionToolManager()
	}
	h.toolManager.RegisterToolDefinition(def)
}

func (h *AlibabaHandler) GetFunctionTools() []interface{} {
	if h.toolManager == nil {
		return nil
	}
	return h.toolManager.GetTools()
}

func (h *AlibabaHandler) ListFunctionTools() []string {
	if h.toolManager == nil {
		return nil
	}
	return h.toolManager.ListTools()
}

func (h *AlibabaHandler) Hangup() {
	h.Interrupt()
}

func (h *AlibabaHandler) Query(text, model string) (string, error) {
	resp, err := h.QueryWithOptions(text, &QueryOptions{Model: model})
	if err != nil {
		return "", err
	}
	if resp == nil || len(resp.Choices) == 0 {
		return "", errors.New("empty response")
	}
	return resp.Choices[0].Content, nil
}

func (h *AlibabaHandler) QueryWithOptions(text string, options *QueryOptions) (*QueryResponse, error) {
	if options == nil {
		options = &QueryOptions{}
	}
	requestType := strings.TrimSpace(options.RequestType)
	if requestType == "" {
		requestType = "query"
	}
	model := options.Model
	if model == "" {
		model = "qwen-plus"
	}

	tracker := NewLLMRequestTracker(
		options.SessionID,
		options.UserID,
		"alibaba",
		model,
		h.endpoint,
		requestType,
	)
	tracker.SetRequestContent(text)
	tracker.SetUserAgent(options.UserAgent)
	tracker.SetIPAddress(options.IPAddress)
	tracker.SetStatusCode(200)
	select {
	case <-h.interruptCh:
		return nil, errors.New("interrupted")
	default:
	}
	var rewrite *QueryRewrite
	promptUser := text
	if options.EnableQueryRewrite {
		rw, err := h.rewriteQueryAlibaba(h.ctx, promptUser, options)
		if err == nil && rw != "" {
			rewrite = &QueryRewrite{Original: promptUser, Rewritten: rw}
			promptUser = rw
		}
	}

	var expansion *QueryExpansion
	if options.EnableQueryExpansion {
		expanded, terms, err := h.expandQueryAlibaba(h.ctx, promptUser, options)
		if err == nil {
			expansion = &QueryExpansion{
				Original: promptUser,
				Expanded: expanded,
				Terms:    terms,
				Debug:    map[string]any{},
			}
			promptUser = expanded
		}
	}

	shortTermPrompt := chatMessagesToPrompt(buildShortTermMessages(promptUser, options))
	if strings.TrimSpace(shortTermPrompt) == "" {
		shortTermPrompt = promptUser
	}
	reqBody := map[string]any{
		"input": map[string]string{
			"prompt": shortTermPrompt,
		},
		"parameters": map[string]any{},
	}
	b, err := json.Marshal(reqBody)
	if err != nil {
		tracker.Error("REQUEST_ERROR", err.Error())
		return nil, err
	}
	url := fmt.Sprintf("%s/api/v1/apps/%s/completion", strings.TrimRight(h.endpoint, "/"), h.appID)
	req, err := http.NewRequestWithContext(h.ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		tracker.Error("REQUEST_ERROR", err.Error())
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+h.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.client.Do(req)
	if err != nil {
		tracker.Error("REQUEST_ERROR", err.Error())
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		tracker.Error("HTTP_ERROR", fmt.Sprintf("status=%d body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("alibaba request failed: status=%d body=%s", resp.StatusCode, string(body))
	}
	var parsed struct {
		Output struct {
			Text string `json:"text"`
		} `json:"output"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		tracker.Error("JSON_PARSE_ERROR", err.Error())
		return nil, err
	}
	answer := strings.TrimSpace(parsed.Output.Text)
	queryResp := &QueryResponse{
		Provider:  h.Provider(),
		Model:     options.Model,
		Choices:   []QueryChoice{{Index: 0, Content: answer, FinishReason: "stop"}},
		Expansion: expansion,
		Rewrite:   rewrite,
	}
	tracker.Complete(queryResp)
	return queryResp, nil
}

func (h *AlibabaHandler) QueryStream(text string, options *QueryOptions, callback func(segment string, isComplete bool) error) (*QueryResponse, error) {
	resp, err := h.QueryWithOptions(text, options)
	if err != nil {
		return nil, err
	}
	if callback != nil && len(resp.Choices) > 0 {
		if err := callback(resp.Choices[0].Content, false); err != nil {
			return nil, err
		}
		if err := callback("", true); err != nil {
			return nil, err
		}
	}
	return resp, nil
}

func (h *AlibabaHandler) Provider() string { return LLM_ALIBABA }

func (h *AlibabaHandler) rewriteQueryAlibaba(ctx context.Context, text string, options *QueryOptions) (string, error) {
	if options == nil {
		options = &QueryOptions{}
	}
	prompt := BuildQueryRewriteUserPrompt(text, options.QueryRewriteInstruction)
	reqBody := map[string]any{
		"input": map[string]string{
			"prompt": prompt,
		},
		"parameters": map[string]any{},
	}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	url := fmt.Sprintf("%s/api/v1/apps/%s/completion", strings.TrimRight(h.endpoint, "/"), h.appID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+h.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("alibaba rewrite failed: status=%d body=%s", resp.StatusCode, string(body))
	}
	var parsed struct {
		Output struct {
			Text string `json:"text"`
		} `json:"output"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	out := NormalizeRewrittenQuery(parsed.Output.Text)
	if out == "" {
		return strings.TrimSpace(text), nil
	}
	return out, nil
}

func (h *AlibabaHandler) expandQueryAlibaba(ctx context.Context, text string, options *QueryOptions) (string, []string, error) {
	if options == nil {
		options = &QueryOptions{}
	}
	maxTerms := expansionMaxTerms(options)
	sep := expansionSeparator(options)
	prompt := BuildQueryExpansionUserPrompt(text, maxTerms)
	reqBody := map[string]any{
		"input": map[string]string{
			"prompt": prompt,
		},
		"parameters": map[string]any{},
	}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return "", nil, err
	}
	url := fmt.Sprintf("%s/api/v1/apps/%s/completion", strings.TrimRight(h.endpoint, "/"), h.appID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Authorization", "Bearer "+h.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", nil, fmt.Errorf("alibaba expansion failed: status=%d body=%s", resp.StatusCode, string(body))
	}
	var parsed struct {
		Output struct {
			Text string `json:"text"`
		} `json:"output"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", nil, err
	}
	out := strings.TrimSpace(parsed.Output.Text)
	expanded, terms := ExpandedQueryFromModelAnswer(text, out, maxTerms, sep)
	return expanded, terms, nil
}
