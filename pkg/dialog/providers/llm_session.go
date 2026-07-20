package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/LingByte/SoulNexus/pkg/dialog/callbinding"
	"github.com/LingByte/SoulNexus/pkg/dialog/cascaded"
	stageknow "github.com/LingByte/SoulNexus/pkg/dialog/stages/knowledge"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/lingllm/protocol"
	"github.com/LingByte/lingllm/tools"
)

const defaultLLMTurnTimeout = 45 * time.Second

func llmRequestContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		timeout = defaultLLMTurnTimeout
	}
	if parent == nil {
		parent = context.Background()
	}
	if _, hasDeadline := parent.Deadline(); hasDeadline {
		return context.WithCancel(parent)
	}
	return context.WithTimeout(parent, timeout)
}

type chatLLMSession struct {
	ctx           context.Context
	model         protocol.ChatModel
	meter         *usageAccumulatingModel
	provider      string
	defaultModel  string
	mu            sync.Mutex
	messages      []protocol.Message
	systemPrompt  string
	tools         *llmFunctionToolManager
	lastUsage     LLMUsage
	lastUsageOK   bool
	lastToolTrace []LLMToolCall
	interruptCh   chan struct{}
	hangupCh      chan struct{}
}

func newChatLLMSession(ctx context.Context, provider string, model protocol.ChatModel, systemPrompt string) *chatLLMSession {
	return &chatLLMSession{
		ctx:          ctx,
		model:        model,
		meter:        wrapUsageAccumulatingModel(model),
		provider:     strings.TrimSpace(provider),
		defaultModel: "gpt-4o",
		systemPrompt: strings.TrimSpace(systemPrompt),
		tools:        newLLMFunctionToolManager(),
		interruptCh:  make(chan struct{}, 1),
		hangupCh:     make(chan struct{}),
	}
}

func (s *chatLLMSession) Query(text, model string) (string, error) {
	temp := float32(0.7)
	return s.QueryWithOptions(text, LLMQueryOptions{Model: model, Temperature: &temp})
}

func (s *chatLLMSession) QueryWithOptions(text string, options LLMQueryOptions) (string, error) {
	if s == nil || s.model == nil {
		return "", fmt.Errorf("%w", utils.ErrSessionNotInitialized)
	}
	start := time.Now()
	userText := text
	if callID := strings.TrimSpace(options.KnowledgeCallID); callID != "" {
		// Enrich outside the session lock — recall can take seconds and must not block tools.
		userText = stageknow.EnrichUserText(context.Background(), callID, text, logger.Lg)
	}
	s.mu.Lock()
	s.appendUser(userText)
	req := s.buildRequest(options)
	if s.meter != nil {
		s.meter.usage = protocol.TokenUsage{}
	}
	executor := newLLMToolExecutor(s.tools, s)
	chainModel := s.model
	if s.meter != nil {
		chainModel = s.meter
	}
	maxRounds := options.MaxToolRounds
	if maxRounds <= 0 {
		maxRounds = cascaded.MaxToolRounds(options.Context)
	}
	if maxRounds <= 0 {
		maxRounds = 10 // voice / unspecified default
	}
	chain := tools.NewToolChain(chainModel, executor).WithMaxRounds(maxRounds)
	s.mu.Unlock()

	reqCtx, cancel := llmRequestContext(options.Context, 45*time.Second)
	defer cancel()
	resp, err := chain.ExecuteWithTools(reqCtx, req)
	if err != nil {
		s.rollbackLastUser(userText)
		s.logInvocation(start, options, LLMUsage{}, false, userText, "", err)
		return "", fmt.Errorf("%w: %v", utils.ErrLLMCallFailed, err)
	}
	reply := strings.TrimSpace(resp.FirstContent())
	s.mu.Lock()
	defer s.mu.Unlock()
	usage := resp.Usage
	if s.meter != nil {
		acc := s.meter.accumulatedUsage()
		if acc.TotalTokens > 0 || acc.PromptTokens > 0 {
			usage = acc
		}
	}
	s.recordUsage(usage, options)
	s.logInvocation(start, options, s.lastUsage, s.lastUsageOK, userText, reply, nil)
	var toolCalls []protocol.ToolCall
	if msg := resp.FirstMessage(); msg != nil {
		toolCalls = msg.ToolCalls
	}
	s.appendAssistant(reply, toolCalls)
	s.lastToolTrace = protocolToolCallsToLLM(toolCalls)
	return reply, nil
}

func (s *chatLLMSession) QueryStream(text string, options LLMQueryOptions, callback func(segment string, isComplete bool) error) (string, error) {
	if s == nil || s.model == nil {
		return "", fmt.Errorf("%w", utils.ErrSessionNotInitialized)
	}
	if callback == nil {
		return "", fmt.Errorf("conversation: nil stream callback")
	}
	start := time.Now()
	var firstTokenAt time.Time
	userText := text
	if callID := strings.TrimSpace(options.KnowledgeCallID); callID != "" {
		userText = stageknow.EnrichUserText(context.Background(), callID, text, logger.Lg)
	}
	s.mu.Lock()
	s.appendUser(userText)
	req := s.buildRequest(options)
	s.mu.Unlock()

	reqCtx, cancel := llmRequestContext(options.Context, defaultLLMTurnTimeout)
	defer cancel()
	stream, err := s.model.StreamChat(reqCtx, req)
	if err != nil {
		s.rollbackLastUser(userText)
		s.logInvocation(start, options, LLMUsage{}, false, userText, "", err)
		return "", fmt.Errorf("%w: %v", utils.ErrLLMCallFailed, err)
	}
	defer stream.Close()

	var full strings.Builder
	for {
		if s.interrupted() {
			reply := full.String()
			s.rollbackLastUser(userText)
			s.logInvocation(start, options, LLMUsage{}, false, userText, reply, fmt.Errorf("stream interrupted"))
			return reply, fmt.Errorf("stream interrupted")
		}
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			reply := full.String()
			s.rollbackLastUser(userText)
			s.logInvocation(start, options, LLMUsage{}, false, userText, reply, err)
			return reply, err
		}
		if chunk == nil || chunk.Delta == "" {
			continue
		}
		if firstTokenAt.IsZero() {
			firstTokenAt = time.Now()
		}
		full.WriteString(chunk.Delta)
		if err := callback(chunk.Delta, false); err != nil {
			reply := full.String()
			s.rollbackLastUser(userText)
			s.logInvocation(start, options, LLMUsage{}, false, userText, reply, err)
			return reply, err
		}
	}
	reply := full.String()
	if err := callback("", true); err != nil {
		s.logInvocation(start, options, LLMUsage{}, false, userText, reply, err)
		return reply, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	var usage LLMUsage
	usageOK := false
	if m := stream.Metrics(); m.TotalTokens > 0 {
		usage = LLMUsage{
			PromptTokens:     m.PromptTokens,
			CompletionTokens: m.CompletionTokens,
			TotalTokens:      m.TotalTokens,
		}
		usageOK = true
		s.lastUsage = usage
		s.lastUsageOK = true
	}
	s.appendAssistant(reply, nil)
	s.logInvocationWithFirstToken(start, firstTokenAt, options, usage, usageOK, userText, reply, nil)
	return reply, nil
}

func (s *chatLLMSession) RegisterFunctionTool(name, description string, parameters interface{}, callback LLMFunctionToolCallback) {
	var params json.RawMessage
	if parameters != nil {
		if raw, ok := parameters.(json.RawMessage); ok {
			params = raw
		} else {
			b, _ := json.Marshal(parameters)
			params = json.RawMessage(b)
		}
	}
	s.tools.registerTool(name, description, params, callback)
}

func (s *chatLLMSession) RegisterFunctionToolDefinition(def *LLMFunctionToolDefinition) {
	s.tools.registerToolDefinition(def)
}

func (s *chatLLMSession) GetFunctionTools() []interface{} {
	out := make([]interface{}, 0, len(s.tools.listTools()))
	for _, name := range s.tools.listTools() {
		out = append(out, name)
	}
	return out
}

func (s *chatLLMSession) ListFunctionTools() []string { return s.tools.listTools() }

func (s *chatLLMSession) GetLastUsage() (LLMUsage, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastUsage, s.lastUsageOK
}

func (s *chatLLMSession) ResetMessages() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = nil
}

func (s *chatLLMSession) SeedMessages(msgs []LLMMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = s.messages[:0]
	for _, m := range msgs {
		role := strings.ToLower(strings.TrimSpace(m.Role))
		content := strings.TrimSpace(m.Content)
		if content == "" {
			continue
		}
		switch role {
		case "user":
			s.messages = append(s.messages, protocol.Message{Role: protocol.RoleUser, Content: content})
		case "assistant":
			s.messages = append(s.messages, protocol.Message{Role: protocol.RoleAssistant, Content: content})
		}
	}
}

func (s *chatLLMSession) SetSystemPrompt(systemPrompt string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.systemPrompt = strings.TrimSpace(systemPrompt)
}

func (s *chatLLMSession) GetMessages() []LLMMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]LLMMessage, len(s.messages))
	for i, m := range s.messages {
		out[i] = protocolMessageToLLM(m)
	}
	return out
}

func (s *chatLLMSession) LastToolTrace() []LLMToolCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.lastToolTrace) == 0 {
		return nil
	}
	out := make([]LLMToolCall, len(s.lastToolTrace))
	copy(out, s.lastToolTrace)
	return out
}

func protocolToolCallsToLLM(calls []protocol.ToolCall) []LLMToolCall {
	if len(calls) == 0 {
		return nil
	}
	out := make([]LLMToolCall, 0, len(calls))
	for _, c := range calls {
		out = append(out, LLMToolCall{
			ID:   c.ID,
			Type: c.Type,
			Function: LLMFunctionCall{
				Name:      c.Function.Name,
				Arguments: string(c.Function.Arguments),
			},
		})
	}
	return out
}

func (s *chatLLMSession) Interrupt() {
	select {
	case s.interruptCh <- struct{}{}:
	default:
	}
}

func (s *chatLLMSession) Hangup() {
	select {
	case <-s.hangupCh:
	default:
		close(s.hangupCh)
	}
}

func (s *chatLLMSession) interrupted() bool {
	select {
	case <-s.interruptCh:
		return true
	case <-s.hangupCh:
		return true
	case <-s.ctx.Done():
		return true
	default:
		return false
	}
}

func (s *chatLLMSession) appendUser(text string) {
	s.messages = append(s.messages, protocol.Message{Role: protocol.RoleUser, Content: text})
}

func (s *chatLLMSession) appendAssistant(text string, toolCalls []protocol.ToolCall) {
	s.messages = append(s.messages, protocol.Message{Role: protocol.RoleAssistant, Content: text, ToolCalls: toolCalls})
}

func (s *chatLLMSession) rollbackLastUser(text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.messages) == 0 {
		return
	}
	last := s.messages[len(s.messages)-1]
	if last.Role == protocol.RoleUser && last.Content == text {
		s.messages = s.messages[:len(s.messages)-1]
	}
}

func (s *chatLLMSession) recordUsage(u protocol.TokenUsage, _ LLMQueryOptions) {
	s.lastUsage = LLMUsage{
		PromptTokens:     u.PromptTokens,
		CompletionTokens: u.CompletionTokens,
		TotalTokens:      u.TotalTokens,
	}
	s.lastUsageOK = true
}

func (s *chatLLMSession) logInvocation(start time.Time, options LLMQueryOptions, usage LLMUsage, usageOK bool, inputText, outputText string, err error) {
	s.logInvocationWithFirstToken(start, time.Time{}, options, usage, usageOK, inputText, outputText, err)
}

func (s *chatLLMSession) logInvocationWithFirstToken(start, firstTokenAt time.Time, options LLMQueryOptions, usage LLMUsage, usageOK bool, inputText, outputText string, err error) {
	if s == nil {
		return
	}
	model := strings.TrimSpace(options.Model)
	if model == "" {
		model = s.defaultModel
	}
	callID := strings.TrimSpace(options.InvocationCallID)
	if callID == "" {
		callID = strings.TrimSpace(options.KnowledgeCallID)
	}
	entry := callbinding.AIInvocationRecord{
		Component:    callbinding.AIComponentLLM,
		Provider:     s.provider,
		Model:        model,
		CallID:       callID,
		Source:       llmInvocationSource(options, callID),
		LatencyMs:    time.Since(start).Milliseconds(),
		InputChars:   utf8.RuneCountInString(inputText),
		OutputChars:  utf8.RuneCountInString(outputText),
		RequestText:  inputText,
		ResponseText: outputText,
	}
	if !firstTokenAt.IsZero() {
		entry.FirstTokenMs = firstTokenAt.Sub(start).Milliseconds()
	}
	if usageOK {
		entry.PromptTokens = usage.PromptTokens
		entry.CompletionTokens = usage.CompletionTokens
		entry.TotalTokens = usage.TotalTokens
	}
	if err != nil {
		entry.Status = callbinding.AIStatusError
		entry.ErrorMsg = err.Error()
	} else {
		entry.Status = callbinding.AIStatusOK
	}
	meta := map[string]any{}
	if options.Stream {
		meta["stream"] = true
	}
	if jsID := callbinding.GetJSSourceID(callID); jsID != "" {
		meta["js_source_id"] = jsID
	}
	if len(meta) > 0 {
		entry.Meta = meta
	}
	callbinding.RecordAIInvocation(entry)
}

func llmInvocationSource(options LLMQueryOptions, callID string) string {
	src := strings.TrimSpace(options.InvocationSource)
	if src == "" {
		src = callbinding.GetAISource(callID)
	}
	if src != "" {
		if options.Stream && !strings.HasSuffix(src, "_stream") {
			switch src {
			case "voice", "assistant_debug_voice", "assistant_debug_text", "js_template", "js_embed":
				return src + "_stream"
			}
		}
		return src
	}
	if strings.TrimSpace(options.KnowledgeCallID) != "" {
		if options.Stream {
			return "voice_stream"
		}
		return "voice"
	}
	return "api"
}

func (s *chatLLMSession) buildRequest(options LLMQueryOptions) protocol.ChatRequest {
	model := strings.TrimSpace(options.Model)
	if model == "" {
		model = s.defaultModel
	}
	msgs := make([]protocol.Message, 0, len(s.messages)+1)
	if s.systemPrompt != "" {
		sys := s.systemPrompt
		if options.EnableJSONOutput {
			sys += "\nRespond with valid JSON only."
		}
		msgs = append(msgs, protocol.Message{Role: protocol.RoleSystem, Content: sys})
	}
	msgs = append(msgs, s.messages...)
	req := protocol.ChatRequest{
		Model:    model,
		Messages: msgs,
	}
	if !options.DisableTools {
		req.Tools = s.tools.protocolTools()
		if len(req.Tools) > 0 {
			req.ToolChoice = protocol.ToolChoiceAuto
		}
	}
	if options.MaxCompletionTokens != nil && *options.MaxCompletionTokens > 0 {
		req.MaxTokens = *options.MaxCompletionTokens
	} else if options.MaxTokens != nil && *options.MaxTokens > 0 {
		req.MaxTokens = *options.MaxTokens
	}
	if options.Temperature != nil {
		req.Temperature = *options.Temperature
	}
	if options.TopP != nil {
		req.TopP = *options.TopP
	}
	if len(options.Stop) > 0 {
		req.Stop = options.Stop
	}
	return req
}

func protocolMessageToLLM(m protocol.Message) LLMMessage {
	out := LLMMessage{Role: string(m.Role), Content: m.Content}
	if len(m.ToolCalls) > 0 {
		out.ToolCalls = make([]LLMToolCall, len(m.ToolCalls))
		for i, tc := range m.ToolCalls {
			out.ToolCalls[i] = LLMToolCall{
				ID:   tc.ID,
				Type: tc.Type,
				Function: LLMFunctionCall{
					Name:      tc.Function.Name,
					Arguments: string(tc.Function.Arguments),
				},
			}
		}
	}
	return out
}

type llmToolExecutor struct {
	mgr   *llmFunctionToolManager
	owner interface{}
}

func newLLMToolExecutor(mgr *llmFunctionToolManager, owner interface{}) *llmToolExecutor {
	if mgr != nil {
		mgr.setOwner(owner)
	}
	return &llmToolExecutor{mgr: mgr, owner: owner}
}

func (e *llmToolExecutor) Execute(_ context.Context, toolName string, args json.RawMessage) (string, error) {
	if e == nil || e.mgr == nil {
		return "", fmt.Errorf("conversation: nil tool executor")
	}
	return e.mgr.executeProtocolTool(toolName, args)
}

func (e *llmToolExecutor) GetTools() []protocol.Tool {
	if e == nil || e.mgr == nil {
		return nil
	}
	return e.mgr.protocolTools()
}
