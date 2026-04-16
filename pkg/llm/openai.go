package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

var emojiRegex = regexp.MustCompile(`([\x{00A9}\x{00AE}\x{203C}\x{2049}\x{2122}\x{2139}\x{2194}-\x{2199}\x{21A9}-\x{21AA}\x{231A}-\x{231B}\x{2328}\x{23CF}\x{23E9}-\x{23F3}\x{23F8}-\x{23FA}\x{24C2}\x{25AA}-\x{25AB}\x{25B6}\x{25C0}\x{25FB}-\x{25FE}\x{2600}-\x{26FF}\x{2700}-\x{27BF}\x{2B05}-\x{2B07}\x{2B1B}-\x{2B1C}\x{2B50}\x{2B55}\x{3030}\x{303D}\x{3297}\x{3299}\x{1F004}\x{1F0CF}\x{1F170}-\x{1F251}\x{1F300}-\x{1F5FF}\x{1F600}-\x{1F64F}\x{1F680}-\x{1F6FF}\x{1F910}-\x{1F93E}\x{1F940}-\x{1F94C}\x{1F950}-\x{1F96B}\x{1F980}-\x{1F997}\x{1F9C0}-\x{1F9E6}\x{1FA70}-\x{1FA74}\x{1FA78}-\x{1FA7A}\x{1FA80}-\x{1FA86}\x{1FA90}-\x{1FAA8}\x{1FAB0}-\x{1FAB6}\x{1FAC0}-\x{1FAC2}\x{1FAD0}-\x{1FAD6}\x{1F1E6}-\x{1F1FF}\x{200D}\x{FE0F}])`)

const defaultMaxMemoryMessages = 40

const maxSummarizeInputChars = 12000

func buildTranscriptForSummary(messages []openai.ChatCompletionMessage) string {
	if len(messages) == 0 {
		return ""
	}
	b := strings.Builder{}
	for _, m := range messages {
		role := m.Role
		if role == "" {
			role = "unknown"
		}
		b.WriteString(role)
		b.WriteString(": ")
		b.WriteString(m.Content)
		b.WriteString("\n")
		if b.Len() >= maxSummarizeInputChars {
			break
		}
	}
	s := b.String()
	if len(s) > maxSummarizeInputChars {
		s = s[len(s)-maxSummarizeInputChars:]
	}
	return s
}

type OpenaiHandler struct {
	ctx               context.Context
	client            *openai.Client
	provider          string
	systemPrompt      string
	fewShotExamples   []FewShotExample
	baseUrl           string
	logger            *zap.Logger
	mutex             sync.Mutex
	messages          []openai.ChatCompletionMessage
	summary           string
	maxMemoryMessages int
	summarizing       atomic.Bool
	summarizeSeq      uint64
	cancelMu          sync.Mutex
	currentCancel     context.CancelFunc
	currentCancelID   uint64
	cancelSeq         uint64
	toolManager       *FunctionToolManager
}

const maxToolCallRounds = 5

func NewOpenaiHandler(ctx context.Context, llmOptions *LLMOptions) (*OpenaiHandler, error) {
	if llmOptions == nil {
		return nil, errors.New("options cannot be nil")
	}
	log := llmOptions.Logger
	if log == nil {
		log = zap.NewNop()
	}
	config := openai.DefaultConfig(llmOptions.ApiKey)
	config.BaseURL = llmOptions.BaseURL
	client := openai.NewClientWithConfig(config)
	return &OpenaiHandler{
		ctx:               ctx,
		client:            client,
		provider:          LLM_OPENAI,
		baseUrl:           llmOptions.BaseURL,
		systemPrompt:      llmOptions.SystemPrompt,
		fewShotExamples:   llmOptions.FewShotExamples,
		logger:            log,
		messages:          make([]openai.ChatCompletionMessage, 0),
		maxMemoryMessages: defaultMaxMemoryMessages,
		toolManager:       NewFunctionToolManager(),
	}, nil
}

func newOpenAICompatibleHandler(ctx context.Context, llmOptions *LLMOptions, provider string) (*OpenaiHandler, error) {
	h, err := NewOpenaiHandler(ctx, llmOptions)
	if err != nil {
		return nil, err
	}
	h.provider = provider
	return h, nil
}

func (oh *OpenaiHandler) ResetMemory() {
	oh.mutex.Lock()
	oh.messages = oh.messages[:0]
	oh.summary = ""
	oh.mutex.Unlock()
}

func (oh *OpenaiHandler) SummarizeMemory(model string) (string, error) {
	// Snapshot under lock.
	oh.mutex.Lock()
	if len(oh.messages) == 0 {
		s := oh.summary
		oh.mutex.Unlock()
		return s, nil
	}
	if oh.summarizing.Load() {
		oh.mutex.Unlock()
		return "", errors.New("summarization already in progress")
	}
	oh.summarizing.Store(true)
	seq := atomic.AddUint64(&oh.summarizeSeq, 1)
	previousSummary := oh.summary
	snapshot := make([]openai.ChatCompletionMessage, len(oh.messages))
	copy(snapshot, oh.messages)
	oh.mutex.Unlock()

	newSummary, err := oh.generateSummary(model, snapshot, previousSummary)
	newSummary = strings.TrimSpace(newSummary)

	oh.mutex.Lock()
	defer oh.mutex.Unlock()
	defer oh.summarizing.Store(false)
	if seq != atomic.LoadUint64(&oh.summarizeSeq) {
		// Newer summarization request happened; don't overwrite.
		if err != nil {
			return "", err
		}
		return oh.summary, nil
	}
	if err != nil {
		return "", err
	}
	if newSummary == "" {
		return oh.summary, nil
	}
	oh.summary = newSummary
	oh.compactMessagesKeepNewer(len(snapshot))
	return oh.summary, nil
}

func (oh *OpenaiHandler) generateSummary(model string, messages []openai.ChatCompletionMessage, previousSummary string) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}
	transcript := buildTranscriptForSummary(messages)
	requestID := GenerateLingRequestID()

	system := "You are a conversation summarizer. Produce a concise, factual summary of the conversation so far. Preserve user preferences, facts, decisions, and open TODOs. Do not include any markdown."
	user := ""
	if previousSummary != "" {
		user += "Existing summary:\n" + previousSummary + "\n\n"
	}
	user += "Conversation transcript:\n" + transcript + "\n\n"
	user += "Return an updated summary in plain text."

	req := openai.ChatCompletionRequest{
		Model: model,
		User:  requestID,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: system},
			{Role: openai.ChatMessageRoleUser, Content: user},
		},
		MaxTokens:   512,
		Temperature: 0,
		TopP:        1,
	}

	resp, err := oh.client.CreateChatCompletion(oh.ctx, req)
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", nil
	}
	newSummary := strings.TrimSpace(resp.Choices[0].Message.Content)
	if newSummary == "" {
		return "", nil
	}
	if emojiRegex != nil {
		newSummary = emojiRegex.ReplaceAllString(newSummary, "")
	}
	return newSummary, nil
}

func (oh *OpenaiHandler) startAsyncSummarizeIfNeeded(model string, snapshot []openai.ChatCompletionMessage, previousSummary string, seq uint64) {
	snapshotLen := len(snapshot)
	go func() {
		newSummary, err := oh.generateSummary(model, snapshot, previousSummary)
		oh.mutex.Lock()
		defer oh.mutex.Unlock()
		defer oh.summarizing.Store(false)
		if seq != atomic.LoadUint64(&oh.summarizeSeq) {
			return
		}
		if err != nil {
			oh.logger.Warn("async memory summarization failed", zap.Error(err))
			return
		}
		newSummary = strings.TrimSpace(newSummary)
		if newSummary == "" {
			return
		}
		oh.summary = newSummary
		oh.compactMessagesKeepNewer(snapshotLen)
	}()
}

func (oh *OpenaiHandler) compactMessagesKeepNewer(snapshotLen int) {
	if snapshotLen <= 0 {
		oh.messages = oh.messages[:0]
		return
	}
	if len(oh.messages) <= snapshotLen {
		oh.messages = oh.messages[:0]
		return
	}
	tail := make([]openai.ChatCompletionMessage, len(oh.messages)-snapshotLen)
	copy(tail, oh.messages[snapshotLen:])
	oh.messages = tail
}

func (oh *OpenaiHandler) SetMaxMemoryMessages(n int) {
	if n <= 0 {
		n = defaultMaxMemoryMessages
	}
	oh.mutex.Lock()
	oh.maxMemoryMessages = n
	if len(oh.messages) > oh.maxMemoryMessages {
		oh.messages = oh.messages[len(oh.messages)-oh.maxMemoryMessages:]
	}
	oh.mutex.Unlock()
}

func (oh *OpenaiHandler) GetMaxMemoryMessages() int {
	oh.mutex.Lock()
	v := oh.maxMemoryMessages
	oh.mutex.Unlock()
	return v
}

func (oh *OpenaiHandler) Provider() string {
	if strings.TrimSpace(oh.provider) == "" {
		return LLM_OPENAI
	}
	return oh.provider
}

func (oh *OpenaiHandler) Interrupt() {
	oh.cancelMu.Lock()
	cancel := oh.currentCancel
	oh.cancelMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (oh *OpenaiHandler) RegisterFunctionTool(name, description string, parameters interface{}, callback FunctionToolCallback) {
	if oh.toolManager == nil {
		oh.toolManager = NewFunctionToolManager()
	}
	oh.toolManager.RegisterTool(name, description, parameters, callback)
}

func (oh *OpenaiHandler) RegisterFunctionToolDefinition(def *FunctionToolDefinition) {
	if oh.toolManager == nil {
		oh.toolManager = NewFunctionToolManager()
	}
	oh.toolManager.RegisterToolDefinition(def)
}

func (oh *OpenaiHandler) GetFunctionTools() []interface{} {
	if oh.toolManager == nil {
		return nil
	}
	return oh.toolManager.GetTools()
}

func (oh *OpenaiHandler) ListFunctionTools() []string {
	if oh.toolManager == nil {
		return nil
	}
	return oh.toolManager.ListTools()
}

func (oh *OpenaiHandler) Hangup() {
	oh.Interrupt()
}

func (oh *OpenaiHandler) setCurrentCancel(cancel context.CancelFunc) uint64 {
	id := atomic.AddUint64(&oh.cancelSeq, 1)
	oh.cancelMu.Lock()
	oh.currentCancel = cancel
	oh.currentCancelID = id
	oh.cancelMu.Unlock()
	return id
}

func (oh *OpenaiHandler) clearCurrentCancel(id uint64) {
	oh.cancelMu.Lock()
	if oh.currentCancelID == id {
		oh.currentCancel = nil
		oh.currentCancelID = 0
	}
	oh.cancelMu.Unlock()
}

func (oh *OpenaiHandler) Query(text, model string) (string, error) {
	resp, err := oh.QueryWithOptions(text, &QueryOptions{
		Model: model,
	})
	if err != nil {
		return "", err
	}
	if resp == nil || len(resp.Choices) == 0 {
		return "", errors.New("empty response choices")
	}
	return resp.Choices[0].Content, nil
}

func (oh *OpenaiHandler) QueryWithOptions(text string, options *QueryOptions) (*QueryResponse, error) {
	if options == nil {
		options = &QueryOptions{}
	}
	requestType := strings.TrimSpace(options.RequestType)
	if requestType == "" {
		requestType = "query"
	}
	model := options.Model
	if model == "" {
		model = "gpt-3.5-turbo"
	}

	tracker := NewLLMRequestTracker(
		options.SessionID,
		options.UserID,
		"openai",
		model,
		oh.baseUrl,
		requestType,
	)
	tracker.SetRequestContent(text)
	tracker.SetUserAgent(options.UserAgent)
	tracker.SetIPAddress(options.IPAddress)
	tracker.SetStatusCode(200)

	// 构建消息 - 目前使用单条文本，后续可扩展支持消息历史

	var rewrite *QueryRewrite
	if options.EnableQueryRewrite {
		before := text
		rwOut, err := oh.rewriteQueryStateless(before, options)
		if err != nil && oh.logger != nil {
			oh.logger.Warn("Query rewrite failed", zap.Error(err))
		} else if err == nil && rwOut != "" {
			rewrite = &QueryRewrite{Original: before, Rewritten: rwOut}
			text = rwOut
		}
	}

	var expansion *QueryExpansion
	if options.EnableQueryExpansion {
		expandedText, terms, err := oh.expandQueryStateless(text, options)
		if err != nil && oh.logger != nil {
			oh.logger.Warn("Query expansion failed", zap.Error(err))
		} else if err == nil {
			expansion = &QueryExpansion{
				Original: text,
				Expanded: expandedText,
				Terms:    terms,
				Debug:    map[string]any{},
			}
			text = expandedText
		}
	}

	n := options.N
	if n <= 0 {
		n = 1
	}
	estimatedMaxOutputChars := 0
	if options.MaxTokens > 0 {
		estimatedMaxOutputChars = options.MaxTokens * 4
	}

	tracker = NewLLMRequestTracker(
		options.SessionID,
		options.UserID,
		oh.Provider(),
		model,
		oh.baseUrl,
		requestType,
	)
	tracker.SetRequestContent(text)
	tracker.SetUserAgent(options.UserAgent)
	tracker.SetIPAddress(options.IPAddress)
	tracker.SetStatusCode(200)

	requestID := GenerateLingRequestID()
	requestedOutputFormat := options.OutputFormat
	if requestedOutputFormat == "" && (options.EnableJSONOutput || options.EnableSelfQueryJSONOutput) {
		requestedOutputFormat = OutputFormatJSONObject
	}
	requestedOutputFormatLower := strings.ToLower(requestedOutputFormat)
	requiresJSONWrapper := requestedOutputFormatLower == OutputFormatXML ||
		requestedOutputFormatLower == OutputFormatHTML ||
		requestedOutputFormatLower == OutputFormatSQL
	responseFormatApplied := false
	appliedResponseFormat := ""
	var responseFormat *openai.ChatCompletionResponseFormat
	switch requestedOutputFormatLower {
	case "", OutputFormatText:
		// default
	case OutputFormatJSON, OutputFormatJSONObject:
		responseFormatApplied = true
		appliedResponseFormat = OutputFormatJSONObject
		responseFormat = &openai.ChatCompletionResponseFormat{Type: openai.ChatCompletionResponseFormatTypeJSONObject}
	case OutputFormatJSONSchema:
		// json_schema requires a schema object; since QueryOptions doesn't carry it yet, fallback to json_object.
		responseFormatApplied = true
		appliedResponseFormat = OutputFormatJSONObject
		responseFormat = &openai.ChatCompletionResponseFormat{Type: openai.ChatCompletionResponseFormatTypeJSONObject}
	default:
		if requiresJSONWrapper {
			responseFormatApplied = true
			appliedResponseFormat = OutputFormatJSONObject
			responseFormat = &openai.ChatCompletionResponseFormat{Type: openai.ChatCompletionResponseFormatTypeJSONObject}
		}
	}

	formatInstruction := ""
	if requiresJSONWrapper {
		formatInstruction = "Return a JSON object with keys: format, content. format must be \"" + requestedOutputFormatLower + "\". content must be strictly " + requestedOutputFormatLower + " and must not be wrapped in markdown."
	}

	extractStructuredContent := func(raw string) string {
		var obj map[string]any
		if err := json.Unmarshal([]byte(raw), &obj); err != nil {
			return raw
		}
		if v, ok := obj["content"]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return raw
	}

	externalShortTermMessages := buildShortTermMessages(text, options)
	hasExternalMessages := len(options.Messages) > 0
	sanitizedMessages := make([]openai.ChatCompletionMessage, 0)
	sysContent := appendEmotionalStyle(oh.systemPrompt, options)
	if strings.TrimSpace(sysContent) != "" {
		sanitizedMessages = append(sanitizedMessages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: sysContent,
		})
	}
	if len(oh.fewShotExamples) > 0 {
		for _, ex := range oh.fewShotExamples {
			u := strings.TrimSpace(ex.User)
			a := strings.TrimSpace(ex.Assistant)
			if u != "" {
				sanitizedMessages = append(sanitizedMessages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: u})
			}
			if a != "" {
				sanitizedMessages = append(sanitizedMessages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleAssistant, Content: a})
			}
		}
	}

	var summarySnapshot string
	var historySnapshot []openai.ChatCompletionMessage
	var maxMemoryMessages int
	oh.mutex.Lock()
	summarySnapshot = oh.summary
	maxMemoryMessages = oh.maxMemoryMessages
	if len(oh.messages) > 0 {
		historySnapshot = make([]openai.ChatCompletionMessage, len(oh.messages))
		copy(historySnapshot, oh.messages)
	}
	oh.mutex.Unlock()

	if hasExternalMessages {
		for _, m := range externalShortTermMessages {
			sanitizedMessages = append(sanitizedMessages, openai.ChatCompletionMessage{
				Role:    m.Role,
				Content: m.Content,
			})
		}
	} else {
		if summarySnapshot != "" {
			sanitizedMessages = append(sanitizedMessages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleSystem,
				Content: "Conversation summary so far: " + summarySnapshot,
			})
		}
		if len(historySnapshot) > 0 {
			if maxMemoryMessages <= 0 {
				maxMemoryMessages = defaultMaxMemoryMessages
			}
			if len(historySnapshot) > maxMemoryMessages {
				historySnapshot = historySnapshot[len(historySnapshot)-maxMemoryMessages:]
			}
			sanitizedMessages = append(sanitizedMessages, historySnapshot...)
		}
	}
	if formatInstruction != "" {
		sanitizedMessages = append(sanitizedMessages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: formatInstruction,
		})
	}
	if estimatedMaxOutputChars > 0 {
		sanitizedMessages = append(sanitizedMessages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: fmt.Sprintf("Limit the assistant output to at most %d characters.", estimatedMaxOutputChars),
		})
	}

	if !hasExternalMessages {
		sanitizedMessages = append(sanitizedMessages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: text,
		})
	}
	request := openai.ChatCompletionRequest{
		Model:          model,
		N:              n,
		User:           requestID,
		ResponseFormat: responseFormat,
		Messages:       sanitizedMessages,
	}
	if tools := oh.buildOpenAITools(); len(tools) > 0 {
		request.Tools = tools
	}
	if options.MaxTokens > 0 {
		request.MaxTokens = options.MaxTokens
	}
	if options.Temperature != 0 {
		request.Temperature = options.Temperature
	}
	if options.TopP != 0 {
		request.TopP = options.TopP
	}
	if options.LogitBias != nil {
		request.LogitBias = options.LogitBias
	}
	reqCtx, cancel := context.WithCancel(oh.ctx)
	cancelID := oh.setCurrentCancel(cancel)
	defer func() {
		oh.clearCurrentCancel(cancelID)
		cancel()
	}()
	response, err := oh.createChatCompletionWithTools(reqCtx, request)
	if err != nil {
		tracker.Error("API_ERROR", err.Error())
		return nil, err
	}
	choices := make([]QueryChoice, 0, len(response.Choices))
	for i, c := range response.Choices {
		content := c.Message.Content
		if requiresJSONWrapper {
			content = extractStructuredContent(content)
		}
		if options.FilterEmoji {
			content = emojiRegex.ReplaceAllString(content, "")
		}
		choices = append(choices, QueryChoice{
			Index:        i,
			Content:      content,
			FinishReason: string(c.FinishReason),
		})
	}
	assistantContent := ""
	if len(choices) > 0 {
		assistantContent = choices[0].Content
	}
	if !hasExternalMessages {
		oh.mutex.Lock()
		oh.messages = append(oh.messages,
			openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: text},
			openai.ChatCompletionMessage{Role: openai.ChatMessageRoleAssistant, Content: assistantContent},
		)
		if maxMemoryMessages <= 0 {
			maxMemoryMessages = defaultMaxMemoryMessages
		}
		if len(oh.messages) > maxMemoryMessages {
			oh.messages = oh.messages[len(oh.messages)-maxMemoryMessages:]
		}

		localSummary := oh.summary
		shouldStartSummarize := len(oh.messages) >= maxMemoryMessages
		var summarizeSnapshot []openai.ChatCompletionMessage
		if shouldStartSummarize && !oh.summarizing.Load() {
			oh.summarizing.Store(true)
			seq := atomic.AddUint64(&oh.summarizeSeq, 1)
			summarizeSnapshot = make([]openai.ChatCompletionMessage, len(oh.messages))
			copy(summarizeSnapshot, oh.messages)
			oh.mutex.Unlock()
			oh.startAsyncSummarizeIfNeeded(model, summarizeSnapshot, localSummary, seq)
		} else {
			oh.mutex.Unlock()
		}
	}

	resp := &QueryResponse{
		Provider: oh.Provider(),
		Model:    response.Model,
		Choices:  choices,
		Usage: &TokenUsage{
			PromptTokens:     response.Usage.PromptTokens,
			CompletionTokens: response.Usage.CompletionTokens,
			TotalTokens:      response.Usage.TotalTokens,
			PromptTokensDetails: func() *PromptTokensDetails {
				if response.Usage.PromptTokensDetails == nil {
					return nil
				}
				return &PromptTokensDetails{
					AudioTokens:  response.Usage.PromptTokensDetails.AudioTokens,
					CachedTokens: response.Usage.PromptTokensDetails.CachedTokens,
				}
			}(),
			CompletionTokensDetails: func() *CompletionTokensDetails {
				if response.Usage.CompletionTokensDetails == nil {
					return nil
				}
				return &CompletionTokensDetails{
					AudioTokens:              response.Usage.CompletionTokensDetails.AudioTokens,
					ReasoningTokens:          response.Usage.CompletionTokensDetails.ReasoningTokens,
					AcceptedPredictionTokens: response.Usage.CompletionTokensDetails.AcceptedPredictionTokens,
					RejectedPredictionTokens: response.Usage.CompletionTokensDetails.RejectedPredictionTokens,
				}
			}(),
		},
		Expansion: expansion,
		Rewrite:   rewrite,
	}

	llmDetails := &LLMDetails{
		RequestID:               requestID,
		Provider:                oh.Provider(),
		BaseURL:                 oh.baseUrl,
		Model:                   response.Model,
		Input:                   text,
		SystemPrompt:            oh.systemPrompt,
		N:                       n,
		MaxTokens:               options.MaxTokens,
		EstimatedMaxOutputChars: estimatedMaxOutputChars,
		FilterEmoji:             options.FilterEmoji,
		RequestedOutputFormat:   requestedOutputFormatLower,
		AppliedResponseFormat:   appliedResponseFormat,
		ResponseFormatApplied:   responseFormatApplied,
		ResponseID:              response.ID,
		Object:                  response.Object,
		Created:                 response.Created,
		SystemFingerprint:       response.SystemFingerprint,
		ChoicesCount:            len(response.Choices),
		Choices:                 resp.Choices,
		Usage:                   resp.Usage,
	}
	if b, e := json.Marshal(response.PromptFilterResults); e == nil {
		llmDetails.PromptFilterResultsJSON = string(b)
	}
	if b, e := json.Marshal(response.ServiceTier); e == nil {
		llmDetails.ServiceTierJSON = string(b)
	}
	if b, e := json.Marshal(response.Usage); e == nil {
		llmDetails.UsageRawJSON = string(b)
	}
	if b, e := json.Marshal(response.Choices); e == nil {
		llmDetails.ChoicesRawJSON = string(b)
	}
	if b, e := json.Marshal(response); e == nil {
		llmDetails.RawResponseJSON = string(b)
	}

	tracker.Complete(resp)
	return resp, nil
}

func (oh *OpenaiHandler) QueryStream(text string, options *QueryOptions, callback func(segment string, isComplete bool) error) (*QueryResponse, error) {
	if options == nil {
		options = &QueryOptions{}
	}
	if len(oh.buildOpenAITools()) > 0 {
		resp, err := oh.QueryWithOptions(text, options)
		if err != nil {
			return nil, err
		}
		if callback != nil && resp != nil && len(resp.Choices) > 0 {
			if err := callback(resp.Choices[0].Content, false); err != nil {
				return nil, err
			}
			if err := callback("", true); err != nil {
				return nil, err
			}
		}
		return resp, nil
	}
	requestType := strings.TrimSpace(options.RequestType)
	if requestType == "" {
		requestType = "query_stream"
	}
	var streamRewrite *QueryRewrite
	if options.EnableQueryRewrite {
		before := text
		rwOut, err := oh.rewriteQueryStateless(before, options)
		if err != nil && oh.logger != nil {
			oh.logger.Warn("Query rewrite failed", zap.Error(err))
		} else if err == nil && rwOut != "" {
			streamRewrite = &QueryRewrite{Original: before, Rewritten: rwOut}
			text = rwOut
		}
	}
	var streamExpansion *QueryExpansion
	if options.EnableQueryExpansion {
		expandedText, terms, err := oh.expandQueryStateless(text, options)
		if err != nil && oh.logger != nil {
			oh.logger.Warn("Query expansion failed", zap.Error(err))
		} else if err == nil {
			streamExpansion = &QueryExpansion{
				Original: text,
				Expanded: expandedText,
				Terms:    terms,
				Debug:    map[string]any{},
			}
			text = expandedText
		}
	}
	n := options.N
	if n <= 0 {
		n = 1
	}
	model := options.Model
	if model == "" {
		model = "qwen-plus"
	}

	tracker := NewLLMRequestTracker(
		options.SessionID,
		options.UserID,
		oh.Provider(),
		model,
		oh.baseUrl,
		requestType,
	)
	tracker.SetRequestContent(text)
	tracker.SetUserAgent(options.UserAgent)
	tracker.SetIPAddress(options.IPAddress)
	tracker.SetStatusCode(200)

	requestID := GenerateLingRequestID()
	externalShortTermMessages := buildShortTermMessages(text, options)
	hasExternalMessages := len(options.Messages) > 0
	messages := make([]openai.ChatCompletionMessage, 0)
	sysContent := appendEmotionalStyle(oh.systemPrompt, options)
	if strings.TrimSpace(sysContent) != "" {
		messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleSystem, Content: sysContent})
	}
	if len(oh.fewShotExamples) > 0 {
		for _, ex := range oh.fewShotExamples {
			u := strings.TrimSpace(ex.User)
			a := strings.TrimSpace(ex.Assistant)
			if u != "" {
				messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: u})
			}
			if a != "" {
				messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleAssistant, Content: a})
			}
		}
	}

	oh.mutex.Lock()
	summarySnapshot := oh.summary
	maxMemoryMessages := oh.maxMemoryMessages
	historySnapshot := make([]openai.ChatCompletionMessage, len(oh.messages))
	copy(historySnapshot, oh.messages)
	oh.mutex.Unlock()
	if hasExternalMessages {
		for _, m := range externalShortTermMessages {
			messages = append(messages, openai.ChatCompletionMessage{Role: m.Role, Content: m.Content})
		}
	} else {
		if summarySnapshot != "" {
			messages = append(messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleSystem,
				Content: "Conversation summary so far: " + summarySnapshot,
			})
		}
		if len(historySnapshot) > 0 {
			if maxMemoryMessages <= 0 {
				maxMemoryMessages = defaultMaxMemoryMessages
			}
			if len(historySnapshot) > maxMemoryMessages {
				historySnapshot = historySnapshot[len(historySnapshot)-maxMemoryMessages:]
			}
			messages = append(messages, historySnapshot...)
		}
	}
	if !hasExternalMessages {
		messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: text})
	}

	request := openai.ChatCompletionRequest{
		Model:    model,
		N:        n,
		User:     requestID,
		Messages: messages,
		Stream:   true,
	}
	if options.MaxTokens > 0 {
		request.MaxTokens = options.MaxTokens
	}
	if options.Temperature != 0 {
		request.Temperature = options.Temperature
	}
	if options.TopP != 0 {
		request.TopP = options.TopP
	}
	if options.LogitBias != nil {
		request.LogitBias = options.LogitBias
	}

	reqCtx, cancel := context.WithCancel(oh.ctx)
	cancelID := oh.setCurrentCancel(cancel)
	defer func() {
		oh.clearCurrentCancel(cancelID)
		cancel()
	}()
	stream, err := oh.client.CreateChatCompletionStream(reqCtx, request)
	if err != nil {
		tracker.Error("API_ERROR", err.Error())
		return nil, err
	}
	defer stream.Close()

	var content strings.Builder
	for {
		chunkResp, e := stream.Recv()
		if errors.Is(e, context.Canceled) {
			return nil, e
		}
		if e != nil {
			if errors.Is(e, io.EOF) {
				break
			}
			tracker.Error("STREAM_ERROR", e.Error())
			return nil, e
		}
		if len(chunkResp.Choices) == 0 {
			continue
		}
		delta := chunkResp.Choices[0].Delta.Content
		if delta == "" {
			continue
		}
		if options.FilterEmoji {
			delta = emojiRegex.ReplaceAllString(delta, "")
		}
		content.WriteString(delta)
		if callback != nil {
			if err := callback(delta, false); err != nil {
				return nil, err
			}
		}
	}
	if callback != nil {
		if err := callback("", true); err != nil {
			return nil, err
		}
	}

	finalText := strings.TrimSpace(content.String())
	if !hasExternalMessages {
		oh.mutex.Lock()
		oh.messages = append(oh.messages,
			openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: text},
			openai.ChatCompletionMessage{Role: openai.ChatMessageRoleAssistant, Content: finalText},
		)
		if maxMemoryMessages <= 0 {
			maxMemoryMessages = defaultMaxMemoryMessages
		}
		if len(oh.messages) > maxMemoryMessages {
			oh.messages = oh.messages[len(oh.messages)-maxMemoryMessages:]
		}
		localSummary := oh.summary
		shouldStartSummarize := len(oh.messages) >= maxMemoryMessages
		if shouldStartSummarize && !oh.summarizing.Load() {
			oh.summarizing.Store(true)
			seq := atomic.AddUint64(&oh.summarizeSeq, 1)
			snapshot := make([]openai.ChatCompletionMessage, len(oh.messages))
			copy(snapshot, oh.messages)
			oh.mutex.Unlock()
			oh.startAsyncSummarizeIfNeeded(model, snapshot, localSummary, seq)
		} else {
			oh.mutex.Unlock()
		}
	}

	resp := &QueryResponse{
		Provider: oh.Provider(),
		Model:    model,
		Choices: []QueryChoice{
			{Index: 0, Content: finalText, FinishReason: "stop"},
		},
		Rewrite:   streamRewrite,
		Expansion: streamExpansion,
	}
	tracker.Complete(resp)
	return resp, nil
}

func (oh *OpenaiHandler) buildOpenAITools() []openai.Tool {
	if oh.toolManager == nil {
		return nil
	}
	defs := oh.toolManager.GetTools()
	if len(defs) == 0 {
		return nil
	}
	tools := make([]openai.Tool, 0, len(defs))
	for _, item := range defs {
		def, ok := item.(*FunctionToolDefinition)
		if !ok || def == nil || strings.TrimSpace(def.Name) == "" {
			continue
		}
		tools = append(tools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        def.Name,
				Description: def.Description,
				Parameters:  def.Parameters,
			},
		})
	}
	return tools
}

func (oh *OpenaiHandler) createChatCompletionWithTools(ctx context.Context, request openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	current := request
	for round := 0; round < maxToolCallRounds; round++ {
		resp, err := oh.client.CreateChatCompletion(ctx, current)
		if err != nil {
			return openai.ChatCompletionResponse{}, err
		}
		if len(resp.Choices) == 0 {
			return resp, nil
		}
		assistantMsg := resp.Choices[0].Message
		if len(assistantMsg.ToolCalls) == 0 {
			return resp, nil
		}
		current.Messages = append(current.Messages, assistantMsg)
		for _, tc := range assistantMsg.ToolCalls {
			toolResult := oh.executeToolCall(tc)
			current.Messages = append(current.Messages, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				ToolCallID: tc.ID,
				Content:    toolResult,
			})
		}
	}
	return oh.client.CreateChatCompletion(ctx, current)
}

func (oh *OpenaiHandler) executeToolCall(tc openai.ToolCall) string {
	if oh.toolManager == nil {
		return "工具系统未初始化"
	}
	toolName := strings.TrimSpace(tc.Function.Name)
	if toolName == "" {
		return "工具名称为空"
	}
	var def *FunctionToolDefinition
	for _, item := range oh.toolManager.GetTools() {
		one, ok := item.(*FunctionToolDefinition)
		if ok && one != nil && one.Name == toolName {
			def = one
			break
		}
	}
	if def == nil || def.Callback == nil {
		return fmt.Sprintf("未找到工具: %s", toolName)
	}
	args := map[string]interface{}{}
	rawArgs := strings.TrimSpace(tc.Function.Arguments)
	if rawArgs != "" {
		if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
			return fmt.Sprintf("工具参数解析失败: %v", err)
		}
	}
	result, err := def.Callback(args, oh)
	if err != nil {
		return fmt.Sprintf("工具执行失败: %v", err)
	}
	return result
}

// expandQueryStateless runs expansion via a one-shot completion without touching conversation memory or oh.mutex.
func (oh *OpenaiHandler) expandQueryStateless(text string, options *QueryOptions) (string, []string, error) {
	if options == nil {
		options = &QueryOptions{}
	}
	maxTerms := expansionMaxTerms(options)
	separator := expansionSeparator(options)
	prompt := BuildQueryExpansionUserPrompt(text, maxTerms)
	model := strings.TrimSpace(options.Model)
	if model == "" {
		model = "gpt-4o-mini"
	}
	req := openai.ChatCompletionRequest{
		Model: model,
		User:  GenerateLingRequestID(),
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
		MaxTokens:   256,
		Temperature: 0.2,
		TopP:        1,
	}
	resp, err := oh.client.CreateChatCompletion(oh.ctx, req)
	if err != nil {
		return "", nil, err
	}
	if len(resp.Choices) == 0 {
		return strings.TrimSpace(text), nil, nil
	}
	out := strings.TrimSpace(resp.Choices[0].Message.Content)
	expanded, terms := ExpandedQueryFromModelAnswer(text, out, maxTerms, separator)
	return expanded, terms, nil
}

func (oh *OpenaiHandler) rewriteQueryStateless(text string, options *QueryOptions) (string, error) {
	if options == nil {
		options = &QueryOptions{}
	}
	prompt := BuildQueryRewriteUserPrompt(text, options.QueryRewriteInstruction)
	model := queryRewriteModel(options, "gpt-4o-mini")
	req := openai.ChatCompletionRequest{
		Model: model,
		User:  GenerateLingRequestID(),
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
		MaxTokens:   128,
		Temperature: 0.2,
		TopP:        1,
	}
	resp, err := oh.client.CreateChatCompletion(oh.ctx, req)
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return strings.TrimSpace(text), nil
	}
	out := NormalizeRewrittenQuery(resp.Choices[0].Message.Content)
	if out == "" {
		return strings.TrimSpace(text), nil
	}
	return out, nil
}
