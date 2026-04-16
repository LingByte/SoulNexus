package llm

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/coze-dev/coze-go"
)

type CozeHandler struct {
	client       coze.CozeAPI
	ctx          context.Context
	botID        string
	userID       string
	systemPrompt string
	interruptCh  chan struct{}
	toolManager  *FunctionToolManager
}

func NewCozeHandler(ctx context.Context, llmOptions *LLMOptions) (*CozeHandler, error) {
	var opts LLMOptions
	if llmOptions != nil {
		opts = *llmOptions
	}
	cfg := struct {
		BotID   string `json:"botId"`
		UserID  string `json:"userId"`
		BaseURL string `json:"baseUrl"`
	}{}
	if raw := strings.TrimSpace(opts.BaseURL); raw != "" {
		_ = json.Unmarshal([]byte(raw), &cfg)
		if cfg.BotID == "" && !strings.Contains(raw, "{") {
			cfg.BotID = raw
		}
	}
	if cfg.BotID == "" {
		return nil, errors.New("coze botId is required (set LLM BaseURL as JSON {botId,userId,baseUrl} or plain botId)")
	}
	if cfg.UserID == "" {
		cfg.UserID = "default_user"
	}
	authClient := coze.NewTokenAuth(strings.TrimSpace(opts.ApiKey))
	client := coze.NewCozeAPI(authClient)
	if strings.TrimSpace(cfg.BaseURL) != "" {
		client = coze.NewCozeAPI(authClient, coze.WithBaseURL(strings.TrimSpace(cfg.BaseURL)))
	}
	return &CozeHandler{
		client:       client,
		ctx:          ctx,
		botID:        cfg.BotID,
		userID:       cfg.UserID,
		systemPrompt: opts.SystemPrompt,
		interruptCh:  make(chan struct{}, 1),
		toolManager:  NewFunctionToolManager(),
	}, nil
}

func (h *CozeHandler) Query(text, model string) (string, error) {
	resp, err := h.QueryWithOptions(text, &QueryOptions{Model: model})
	if err != nil {
		return "", err
	}
	if resp == nil || len(resp.Choices) == 0 {
		return "", errors.New("empty response")
	}
	return resp.Choices[0].Content, nil
}

func (h *CozeHandler) QueryWithOptions(text string, options *QueryOptions) (*QueryResponse, error) {
	if options == nil {
		options = &QueryOptions{}
	}
	requestType := strings.TrimSpace(options.RequestType)
	if requestType == "" {
		requestType = "query"
	}
	model := options.Model
	if model == "" {
		model = "coze-chat"
	}

	tracker := NewLLMRequestTracker(
		options.SessionID,
		options.UserID,
		"coze",
		model,
		"https://api.coze.com",
		requestType,
	)
	var rewrite *QueryRewrite
	if options.EnableQueryRewrite {
		before := text
		rw, err := h.rewriteQueryCoze(h.ctx, before, options)
		if err == nil && rw != "" {
			rewrite = &QueryRewrite{Original: before, Rewritten: rw}
			text = rw
		}
	}

	var expansion *QueryExpansion
	if options.EnableQueryExpansion {
		expanded, terms, err := h.expandQueryCoze(h.ctx, text, options)
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

	msgs := h.cozeMessagesForChat(text, options)
	streamFlag := false
	req := &coze.CreateChatsReq{
		BotID:    h.botID,
		UserID:   h.userID,
		Messages: toCozePtrs(msgs),
		Stream:   &streamFlag,
	}
	ctx, cancel := context.WithTimeout(h.ctx, 60*time.Second)
	defer cancel()
	stream, err := h.client.Chat.Stream(ctx, req)
	if err != nil {
		tracker.Error("COZE_REQUEST_ERROR", err.Error())
		return nil, err
	}
	defer stream.Close()
	var out strings.Builder
	for {
		select {
		case <-h.interruptCh:
			return nil, errors.New("interrupted")
		default:
		}
		ev, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		if ev.Message != nil && ev.Message.Content != "" {
			out.WriteString(ev.Message.Content)
		}
		if ev.IsDone() || ev.Event == coze.ChatEventConversationMessageCompleted {
			break
		}
	}
	answer := strings.TrimSpace(out.String())
	resp := &QueryResponse{
		Model:     options.Model,
		Choices:   []QueryChoice{{Index: 0, Content: answer, FinishReason: "stop"}},
		Expansion: expansion,
		Rewrite:   rewrite,
	}
	tracker.Complete(resp)
	return resp, nil
}

func (h *CozeHandler) QueryStream(text string, options *QueryOptions, callback func(segment string, isComplete bool) error) (*QueryResponse, error) {
	if options == nil {
		options = &QueryOptions{}
	}
	requestType := strings.TrimSpace(options.RequestType)
	if requestType == "" {
		requestType = "stream"
	}
	model := options.Model
	if model == "" {
		model = "coze-chat"
	}

	tracker := NewLLMRequestTracker(
		options.SessionID,
		options.UserID,
		"coze",
		model,
		"https://api.coze.com",
		requestType,
	)
	var streamRewrite *QueryRewrite
	if options.EnableQueryRewrite {
		before := text
		rw, err := h.rewriteQueryCoze(h.ctx, before, options)
		if err == nil && rw != "" {
			streamRewrite = &QueryRewrite{Original: before, Rewritten: rw}
			text = rw
		}
	}
	var streamExpansion *QueryExpansion
	if options.EnableQueryExpansion {
		expanded, terms, err := h.expandQueryCoze(h.ctx, text, options)
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
	msgs := h.cozeMessagesForChat(text, options)
	streamFlag := true
	req := &coze.CreateChatsReq{
		BotID:    h.botID,
		UserID:   h.userID,
		Messages: toCozePtrs(msgs),
		Stream:   &streamFlag,
	}
	ctx, cancel := context.WithTimeout(h.ctx, 60*time.Second)
	defer cancel()
	stream, err := h.client.Chat.Stream(ctx, req)
	if err != nil {
		tracker.Error("COZE_REQUEST_ERROR", err.Error())
		return nil, err
	}
	defer stream.Close()
	var out strings.Builder
	for {
		select {
		case <-h.interruptCh:
			return nil, errors.New("interrupted")
		default:
		}
		ev, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		if ev.Message != nil && ev.Message.Content != "" {
			seg := ev.Message.Content
			out.WriteString(seg)
			if callback != nil {
				if err := callback(seg, false); err != nil {
					return nil, err
				}
			}
		}
		if ev.IsDone() || ev.Event == coze.ChatEventConversationMessageCompleted {
			break
		}
	}
	if callback != nil {
		if err := callback("", true); err != nil {
			return nil, err
		}
	}
	answer := strings.TrimSpace(out.String())
	return &QueryResponse{
		Model:     options.Model,
		Choices:   []QueryChoice{{Index: 0, Content: answer, FinishReason: "stop"}},
		Rewrite:   streamRewrite,
		Expansion: streamExpansion,
	}, nil
}

func (h *CozeHandler) Provider() string { return LLM_COZE }

func (h *CozeHandler) Interrupt() {
	select {
	case h.interruptCh <- struct{}{}:
	default:
	}
}

func (h *CozeHandler) RegisterFunctionTool(name, description string, parameters interface{}, callback FunctionToolCallback) {
	if h.toolManager == nil {
		h.toolManager = NewFunctionToolManager()
	}
	h.toolManager.RegisterTool(name, description, parameters, callback)
}

func (h *CozeHandler) RegisterFunctionToolDefinition(def *FunctionToolDefinition) {
	if h.toolManager == nil {
		h.toolManager = NewFunctionToolManager()
	}
	h.toolManager.RegisterToolDefinition(def)
}

func (h *CozeHandler) GetFunctionTools() []interface{} {
	if h.toolManager == nil {
		return nil
	}
	return h.toolManager.GetTools()
}

func (h *CozeHandler) ListFunctionTools() []string {
	if h.toolManager == nil {
		return nil
	}
	return h.toolManager.ListTools()
}

func (h *CozeHandler) Hangup() {
	h.Interrupt()
}

func (h *CozeHandler) cozeMessagesForChat(userText string, opts *QueryOptions) []coze.Message {
	out := make([]coze.Message, 0, 8)
	chatMessages := buildShortTermMessages(userText, opts)
	sysCore := appendEmotionalStyle(mergedSystemPrompt(h.systemPrompt, chatMessages), opts)
	if strings.TrimSpace(sysCore) != "" {
		out = append(out, coze.Message{Role: coze.MessageRoleUser, Content: "System: " + sysCore})
	}
	for _, m := range chatMessages {
		switch m.Role {
		case "assistant":
			out = append(out, coze.Message{Role: coze.MessageRoleAssistant, Content: m.Content})
		case "system":
			// system is merged into a synthetic system line.
		default:
			out = append(out, coze.Message{Role: coze.MessageRoleUser, Content: m.Content})
		}
	}
	return out
}

func (h *CozeHandler) rewriteQueryCoze(ctx context.Context, text string, options *QueryOptions) (string, error) {
	if options == nil {
		options = &QueryOptions{}
	}
	prompt := BuildQueryRewriteUserPrompt(text, options.QueryRewriteInstruction)
	streamFlag := false
	req := &coze.CreateChatsReq{
		BotID:  h.botID,
		UserID: h.userID + "_ling_rewrite",
		Messages: []*coze.Message{
			{Role: coze.MessageRoleUser, Content: prompt},
		},
		Stream: &streamFlag,
	}
	sctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	stream, err := h.client.Chat.Stream(sctx, req)
	if err != nil {
		return "", err
	}
	defer stream.Close()
	var out strings.Builder
	for {
		ev, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", err
		}
		if ev.Message != nil && ev.Message.Content != "" {
			out.WriteString(ev.Message.Content)
		}
		if ev.IsDone() || ev.Event == coze.ChatEventConversationMessageCompleted {
			break
		}
	}
	answer := NormalizeRewrittenQuery(out.String())
	if answer == "" {
		return strings.TrimSpace(text), nil
	}
	return answer, nil
}

func (h *CozeHandler) expandQueryCoze(ctx context.Context, text string, options *QueryOptions) (string, []string, error) {
	if options == nil {
		options = &QueryOptions{}
	}
	maxTerms := expansionMaxTerms(options)
	sep := expansionSeparator(options)
	prompt := BuildQueryExpansionUserPrompt(text, maxTerms)
	streamFlag := false
	req := &coze.CreateChatsReq{
		BotID:  h.botID,
		UserID: h.userID + "_ling_expand",
		Messages: []*coze.Message{
			{Role: coze.MessageRoleUser, Content: prompt},
		},
		Stream: &streamFlag,
	}
	sctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	stream, err := h.client.Chat.Stream(sctx, req)
	if err != nil {
		return "", nil, err
	}
	defer stream.Close()
	var out strings.Builder
	for {
		ev, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", nil, err
		}
		if ev.Message != nil && ev.Message.Content != "" {
			out.WriteString(ev.Message.Content)
		}
		if ev.IsDone() || ev.Event == coze.ChatEventConversationMessageCompleted {
			break
		}
	}
	answer := strings.TrimSpace(out.String())
	expanded, terms := ExpandedQueryFromModelAnswer(text, answer, maxTerms, sep)
	return expanded, terms, nil
}

func (h *CozeHandler) summarizeCoze(ctx context.Context, model string, transcript string, previousSummary string) (string, error) {
	_ = model
	prompt := "You are a conversation summarizer. Produce a concise, factual summary of the conversation so far. Preserve user preferences, facts, decisions, and open TODOs. Do not include any markdown.\n\n"
	if strings.TrimSpace(previousSummary) != "" {
		prompt += "Existing summary:\n" + previousSummary + "\n\n"
	}
	prompt += "Conversation transcript:\n" + transcript + "\n\nReturn an updated summary in plain text."
	streamFlag := false
	req := &coze.CreateChatsReq{
		BotID:  h.botID,
		UserID: h.userID + "_ling_mem",
		Messages: []*coze.Message{
			{Role: coze.MessageRoleUser, Content: prompt},
		},
		Stream: &streamFlag,
	}
	sctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	stream, err := h.client.Chat.Stream(sctx, req)
	if err != nil {
		return "", err
	}
	defer stream.Close()
	var out strings.Builder
	for {
		ev, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", err
		}
		if ev.Message != nil && ev.Message.Content != "" {
			out.WriteString(ev.Message.Content)
		}
		if ev.IsDone() || ev.Event == coze.ChatEventConversationMessageCompleted {
			break
		}
	}
	return strings.TrimSpace(out.String()), nil
}

func toCozePtrs(in []coze.Message) []*coze.Message {
	out := make([]*coze.Message, 0, len(in))
	for i := range in {
		m := in[i]
		out = append(out, &coze.Message{Role: m.Role, Content: m.Content})
	}
	return out
}
