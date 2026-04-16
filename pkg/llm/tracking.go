package llm

import (
	"time"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/utils"
)

// LLMRequestTracker LLM请求跟踪器
type LLMRequestTracker struct {
	requestID       string
	sessionID       string
	userID          string
	provider        string
	model           string
	baseURL         string
	requestType     string
	startTime       time.Time
	startedAt       time.Time
	firstTokenAt    time.Time
	requestContent  string
	responseContent string
	userAgent       string
	ipAddress       string
	statusCode      int
}

// NewLLMRequestTracker 创建LLM请求跟踪器
func NewLLMRequestTracker(sessionID, userID, provider, model, baseURL, requestType string) *LLMRequestTracker {
	requestID := "ling-chatimpl-" + utils.SnowflakeUtil.GenID()
	now := time.Now()
	tracker := &LLMRequestTracker{
		requestID:   requestID,
		sessionID:   sessionID,
		userID:      userID,
		provider:    provider,
		model:       model,
		baseURL:     baseURL,
		requestType: requestType,
		startTime:   now,
		startedAt:   now,
	}
	startData := LLMRequestStartData{
		RequestID:   requestID,
		SessionID:   sessionID,
		UserID:      userID,
		Provider:    provider,
		Model:       model,
		RequestType: requestType,
		RequestedAt: tracker.startTime.UnixMilli(),
	}
	utils.Sig().Emit(SignalLLMRequestStart, tracker, startData)
	return tracker
}

// GetRequestID 获取请求ID
func (t *LLMRequestTracker) GetRequestID() string {
	return t.requestID
}

// SetRequestContent 设置请求内容
func (t *LLMRequestTracker) SetRequestContent(content string) {
	t.requestContent = content
}

// SetResponseContent 设置响应内容
func (t *LLMRequestTracker) SetResponseContent(content string) {
	t.responseContent = content
}

// SetUserAgent 设置用户代理
func (t *LLMRequestTracker) SetUserAgent(userAgent string) {
	t.userAgent = userAgent
}

// SetIPAddress 设置IP地址
func (t *LLMRequestTracker) SetIPAddress(ip string) {
	t.ipAddress = ip
}

// SetStatusCode 设置HTTP状态码
func (t *LLMRequestTracker) SetStatusCode(code int) {
	t.statusCode = code
}

// MarkStarted 标记实际开始处理时间
func (t *LLMRequestTracker) MarkStarted() {
	t.startedAt = time.Now()
}

// MarkFirstToken 标记首个token时间
func (t *LLMRequestTracker) MarkFirstToken() {
	t.firstTokenAt = time.Now()
}

// Complete 完成请求并记录成功信息
func (t *LLMRequestTracker) Complete(response *QueryResponse) {
	endTime := time.Now()
	latencyMs := endTime.Sub(t.startTime).Milliseconds()
	t.SetResponseContent("")

	response.RequestID = t.requestID
	response.SessionID = t.sessionID
	response.UserID = t.userID
	response.RequestedAt = t.startTime.UnixMilli()
	response.CompletedAt = endTime.UnixMilli()
	response.LatencyMs = latencyMs

	// 计算token使用量
	inputTokens := 0
	outputTokens := 0
	if response.Usage != nil {
		inputTokens = response.Usage.PromptTokens
		outputTokens = response.Usage.CompletionTokens
	}
	totalTokens := inputTokens + outputTokens

	// 获取输出内容
	output := ""
	if len(response.Choices) > 0 {
		output = response.Choices[0].Content
	}
	t.SetResponseContent(output)

	// 计算性能指标
	var ttftMs int64 = 0
	var tps float64 = 0
	var queueTimeMs int64 = 0

	if !t.firstTokenAt.IsZero() {
		ttftMs = t.firstTokenAt.Sub(t.startedAt).Milliseconds()
	}

	if outputTokens > 0 && latencyMs > 0 {
		tps = float64(outputTokens) / (float64(latencyMs) / 1000.0)
	}

	if !t.startedAt.IsZero() {
		queueTimeMs = t.startedAt.Sub(t.startTime).Milliseconds()
	}

	// 发送请求结束信号
	endData := LLMRequestEndData{
		RequestID:    t.requestID,
		SessionID:    t.sessionID,
		UserID:       t.userID,
		Provider:     t.provider,
		Model:        t.model,
		RequestType:  t.requestType,
		Success:      true,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  totalTokens,
		LatencyMs:    latencyMs,
		Output:       output,
		RequestedAt:  t.startTime.UnixMilli(),
		CompletedAt:  endTime.UnixMilli(),
	}

	utils.Sig().Emit(SignalLLMRequestEnd, t, endData)

	// 发送LLM用量信号（纯信号方式）
	usageInfo := map[string]interface{}{
		"request_id":       t.requestID,
		"session_id":       t.sessionID,
		"user_id":          t.userID,
		"provider":         t.provider,
		"model":            t.model,
		"base_url":         t.baseURL,
		"request_type":     t.requestType,
		"input_tokens":     inputTokens,
		"output_tokens":    outputTokens,
		"total_tokens":     totalTokens,
		"latency_ms":       latencyMs,
		"ttft_ms":          ttftMs,
		"tps":              tps,
		"queue_time_ms":    queueTimeMs,
		"request_content":  t.requestContent,
		"response_content": t.responseContent,
		"user_agent":       t.userAgent,
		"ip_address":       t.ipAddress,
		"status_code":      t.statusCode,
		"success":          true,
		"requested_at":     t.startTime.UnixMilli(),
		"started_at":       t.startedAt.UnixMilli(),
		"first_token_at":   t.firstTokenAt.UnixMilli(),
		"completed_at":     endTime.UnixMilli(),
	}

	utils.Sig().Emit(constants.SignalLLMUsage, usageInfo, "text", output)
}

// Error 记录请求错误
func (t *LLMRequestTracker) Error(errCode, errorMessage string) {
	endTime := time.Now()
	latencyMs := endTime.Sub(t.startTime).Milliseconds()

	// 计算排队时间
	var queueTimeMs int64 = 0
	if !t.startedAt.IsZero() {
		queueTimeMs = t.startedAt.Sub(t.startTime).Milliseconds()
	}

	errorData := LLMRequestErrorData{
		RequestID:    t.requestID,
		SessionID:    t.sessionID,
		UserID:       t.userID,
		Provider:     t.provider,
		Model:        t.model,
		RequestType:  t.requestType,
		ErrorCode:    errCode,
		ErrorMessage: errorMessage,
		LatencyMs:    latencyMs,
		RequestedAt:  t.startTime.UnixMilli(),
		CompletedAt:  endTime.UnixMilli(),
	}
	utils.Sig().Emit(SignalLLMRequestError, t, errorData)

	usageInfo := map[string]interface{}{
		"request_id":       t.requestID,
		"session_id":       t.sessionID,
		"user_id":          t.userID,
		"provider":         t.provider,
		"model":            t.model,
		"base_url":         t.baseURL,
		"request_type":     t.requestType,
		"input_tokens":     0,
		"output_tokens":    0,
		"total_tokens":     0,
		"latency_ms":       latencyMs,
		"ttft_ms":          0,
		"tps":              0,
		"queue_time_ms":    queueTimeMs,
		"request_content":  t.requestContent,
		"response_content": t.responseContent,
		"user_agent":       t.userAgent,
		"ip_address":       t.ipAddress,
		"status_code":      t.statusCode,
		"success":          false,
		"error_code":       errCode,
		"error_message":    errorMessage,
		"requested_at":     t.startTime.UnixMilli(),
		"started_at":       t.startedAt.UnixMilli(),
		"first_token_at":   t.firstTokenAt.UnixMilli(),
		"completed_at":     endTime.UnixMilli(),
	}
	utils.Sig().Emit(constants.SignalLLMUsage, usageInfo, "text", "")
}

// CreateSession 创建会话并发送信号
func CreateSession(sessionID, userID string, assistantID int64, title, provider, model, systemPrompt string) {
	startData := SessionCreatedData{
		SessionID:    sessionID,
		UserID:       userID,
		AssistantID:  assistantID,
		Title:        title,
		Provider:     provider,
		Model:        model,
		SystemPrompt: systemPrompt,
		CreatedAt:    time.Now().UnixMilli(),
	}
	utils.Sig().Emit(SignalSessionCreated, nil, startData)
}

// CreateMessage 创建消息并发送信号
func CreateMessage(messageID, sessionID, role, content string, tokenCount int, model, provider, requestID string) {
	startData := MessageCreatedData{
		MessageID:  messageID,
		SessionID:  sessionID,
		Role:       role,
		Content:    content,
		TokenCount: tokenCount,
		Model:      model,
		Provider:   provider,
		RequestID:  requestID,
		CreatedAt:  time.Now().UnixMilli(),
	}
	utils.Sig().Emit(SignalMessageCreated, nil, startData)
}
