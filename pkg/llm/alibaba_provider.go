package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"go.uber.org/zap"
)

// AlibabaProvider 阿里云百炼 LLM 提供者实现
type AlibabaProvider struct {
	config          AlibabaAIConfig
	client          *http.Client
	ctx             context.Context
	systemMsg       string
	pendingAction   string
	mutex           sync.Mutex
	messages        []Message
	hangupChan      chan struct{}
	interruptCh     chan struct{}
	functionManager *FunctionToolManager
	lastUsage       Usage
	lastUsageValid  bool
}

// ConsumePendingAction returns and clears the latest resolved action.
func (p *AlibabaProvider) ConsumePendingAction() string {
	if p == nil {
		return ""
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	a := p.pendingAction
	p.pendingAction = ""
	return a
}

type alibabaMessagePayload struct {
	Message    string `json:"message"`
	NeedPerson int    `json:"needperson"`
	NeedHangup int    `json:"needhangup"`
	Action     string `json:"action"`
}

func parseAlibabaPayload(raw string) (alibabaMessagePayload, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return alibabaMessagePayload{}, false
	}
	var msg alibabaMessagePayload
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		return alibabaMessagePayload{}, false
	}
	// Some app templates nest the real JSON in message as a JSON string/object.
	nestedRaw := strings.TrimSpace(msg.Message)
	if nestedRaw != "" && strings.HasPrefix(nestedRaw, "{") && strings.HasSuffix(nestedRaw, "}") {
		var nested alibabaMessagePayload
		if err := json.Unmarshal([]byte(nestedRaw), &nested); err == nil {
			if strings.TrimSpace(nested.Message) != "" {
				msg.Message = nested.Message
			}
			if strings.TrimSpace(nested.Action) != "" {
				msg.Action = nested.Action
			}
			if nested.NeedPerson != 0 {
				msg.NeedPerson = nested.NeedPerson
			}
			if nested.NeedHangup != 0 {
				msg.NeedHangup = nested.NeedHangup
			}
		}
	}
	if strings.TrimSpace(msg.Message) == "" && strings.TrimSpace(msg.Action) == "" && msg.NeedPerson == 0 && msg.NeedHangup == 0 {
		// Generic fallback for templates with different key casing/naming.
		var anyMap map[string]interface{}
		if err := json.Unmarshal([]byte(raw), &anyMap); err == nil {
			if v, ok := anyMap["message"].(string); ok && strings.TrimSpace(v) != "" {
				msg.Message = v
			}
			if v, ok := anyMap["action"].(string); ok {
				msg.Action = v
			}
			if v, ok := anyMap["needperson"].(float64); ok {
				msg.NeedPerson = int(v)
			}
			if v, ok := anyMap["needhangup"].(float64); ok {
				msg.NeedHangup = int(v)
			}
		}
	}
	if strings.TrimSpace(msg.Message) == "" && strings.TrimSpace(msg.Action) == "" && msg.NeedPerson == 0 && msg.NeedHangup == 0 {
		return alibabaMessagePayload{}, false
	}
	return msg, true
}

func previewText(s string, n int) string {
	s = strings.TrimSpace(s)
	if n <= 0 || len(s) <= n {
		return s
	}
	return s[:n] + "...(truncated)"
}

// AlibabaAIConfig 阿里云百炼配置
type AlibabaAIConfig struct {
	APIKey    string
	AppID     string
	Endpoint  string
	Timeout   time.Duration // 总超时时间
	FirstByte time.Duration // 首字节超时时间（默认10秒）
}

// NewAlibabaProvider 创建阿里云百炼提供者
func NewAlibabaProvider(ctx context.Context, apiKey, appID, systemPrompt string, endpoint ...string) *AlibabaProvider {
	timeout := 30 * time.Second
	firstByte := 10 * time.Second
	if s := strings.TrimSpace(os.Getenv("ALIBABA_AI_TIMEOUT_SECONDS")); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			timeout = time.Duration(n) * time.Second
		}
	}
	if s := strings.TrimSpace(os.Getenv("ALIBABA_AI_FIRST_BYTE_TIMEOUT_SECONDS")); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			firstByte = time.Duration(n) * time.Second
		}
	}
	config := AlibabaAIConfig{
		APIKey:    apiKey,
		AppID:     appID,
		Timeout:   timeout,
		FirstByte: firstByte,
	}

	if len(endpoint) > 0 && endpoint[0] != "" {
		config.Endpoint = endpoint[0]
	} else {
		config.Endpoint = "https://dashscope.aliyuncs.com"
	}

	return &AlibabaProvider{
		config:          config,
		client:          &http.Client{Timeout: config.Timeout},
		ctx:             ctx,
		systemMsg:       systemPrompt,
		messages:        make([]Message, 0),
		hangupChan:      make(chan struct{}),
		interruptCh:     make(chan struct{}, 1),
		functionManager: NewFunctionToolManager(),
	}
}

// Query 执行非流式查询
func (p *AlibabaProvider) Query(text, model string) (string, error) {
	return p.QueryWithOptions(text, QueryOptions{Model: model})
}

// QueryWithOptions 执行带完整参数的非流式查询
func (p *AlibabaProvider) QueryWithOptions(text string, options QueryOptions) (string, error) {
	startTime := time.Now()

	p.mutex.Lock()
	// 添加用户消息到历史
	p.messages = append(p.messages, Message{
		Role:    "user",
		Content: text,
	})
	p.mutex.Unlock()

	ctx, cancel := context.WithTimeout(p.ctx, p.config.Timeout)
	defer cancel()

	// 构建请求
	reqBody := map[string]interface{}{
		"input": map[string]string{
			"prompt":     p.composePrompt(text),
			"session_id": options.SessionID,
		},
		"parameters": map[string]interface{}{},
	}

	bodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/apps/%s/completion", p.config.Endpoint, p.config.AppID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyJSON))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.config.APIKey))
	req.Header.Set("Content-Type", "application/json")

	logger.Debug("Alibaba AI request started",
		zap.String("url", url),
		zap.String("prompt", text))

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()
	logger.Debug("Alibaba AI response headers received",
		zap.Int("status_code", resp.StatusCode),
		zap.String("url", url),
	)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	// 读取完整响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var result struct {
		Output struct {
			Text         string `json:"text"`
			FinishReason string `json:"finish_reason"`
		} `json:"output"`
		Usage struct {
			Models []struct {
				ModelID      string `json:"model_id"`
				InputTokens  int    `json:"input_tokens"`
				OutputTokens int    `json:"output_tokens"`
			} `json:"models"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	finalResponse := result.Output.Text

	logger.Debug("Alibaba AI raw output",
		zap.String("output_preview", previewText(finalResponse, 300)),
	)
	if msgResp, ok := parseAlibabaPayload(finalResponse); ok {
		logger.Info("Alibaba AI json parsed",
			zap.String("action", strings.TrimSpace(msgResp.Action)),
			zap.Int("needperson", msgResp.NeedPerson),
			zap.Int("needhangup", msgResp.NeedHangup),
			zap.String("message_preview", previewText(msgResp.Message, 120)),
		)
		if msgResp.Message != "" {
			finalResponse = msgResp.Message
		}
		_ = p.maybeInvokeActions(msgResp)
	} else {
		logger.Warn("Alibaba AI json parse failed (non-structured response)",
			zap.String("output_preview", previewText(finalResponse, 300)),
		)
	}

	// 更新消息历史
	p.mutex.Lock()
	p.messages = append(p.messages, Message{
		Role:    "assistant",
		Content: finalResponse,
	})
	p.mutex.Unlock()

	// 记录使用统计
	if len(result.Usage.Models) > 0 {
		p.lastUsage = Usage{
			PromptTokens:     result.Usage.Models[0].InputTokens,
			CompletionTokens: result.Usage.Models[0].OutputTokens,
			TotalTokens:      result.Usage.Models[0].InputTokens + result.Usage.Models[0].OutputTokens,
		}
		p.lastUsageValid = true
	}

	logger.Info("Alibaba AI request completed",
		zap.Duration("duration", time.Since(startTime)),
		zap.Int("response_length", len(finalResponse)))

	return finalResponse, nil
}

// QueryStream 执行流式查询
func (p *AlibabaProvider) QueryStream(text string, options QueryOptions, callback func(segment string, isComplete bool) error) (string, error) {
	startTime := time.Now()

	p.mutex.Lock()
	// 添加用户消息到历史
	p.messages = append(p.messages, Message{
		Role:    "user",
		Content: text,
	})
	p.mutex.Unlock()

	ctx, cancel := context.WithTimeout(p.ctx, p.config.Timeout)
	defer cancel()

	// 构建请求
	reqBody := map[string]interface{}{
		"input": map[string]string{
			"prompt":     p.composePrompt(text),
			"session_id": options.SessionID,
		},
		"parameters": map[string]interface{}{},
	}

	bodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/apps/%s/completion", p.config.Endpoint, p.config.AppID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyJSON))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.config.APIKey))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-DashScope-SSE", "enable")

	logger.Debug("Alibaba AI stream request started",
		zap.String("url", url),
		zap.String("prompt", text))

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()
	logger.Debug("Alibaba AI stream response headers received",
		zap.Int("status_code", resp.StatusCode),
		zap.String("url", url),
	)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	// 处理流式响应
	fullResponse, err := p.processStreamResponse(ctx, resp.Body, callback)
	if err != nil {
		return fullResponse, err
	}

	// 更新消息历史
	p.mutex.Lock()
	p.messages = append(p.messages, Message{
		Role:    "assistant",
		Content: fullResponse,
	})
	p.mutex.Unlock()

	logger.Info("Alibaba AI stream request completed",
		zap.Duration("duration", time.Since(startTime)),
		zap.Int("response_length", len(fullResponse)))

	return fullResponse, nil
}

// processStreamResponse 处理流式响应
func (p *AlibabaProvider) processStreamResponse(ctx context.Context, body io.Reader, callback func(text string, isComplete bool) error) (string, error) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	// 首字节超时检测
	firstByteTimer := time.NewTimer(p.config.FirstByte)
	defer firstByteTimer.Stop()

	firstByteChan := make(chan bool, 1)
	go func() {
		if scanner.Scan() {
			firstByteChan <- true
		} else {
			firstByteChan <- false
		}
	}()

	// 等待首字节或超时
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-firstByteTimer.C:
		// 超时，返回默认消息
		logger.Warn("Alibaba AI: First byte timeout", zap.Duration("timeout", p.config.FirstByte))
		if callback != nil {
			callback("您好,非常抱歉,您的这个问题我暂时无法解答,建议您提交工单申请处理。", true)
		}
		return "您好,非常抱歉,您的这个问题我暂时无法解答,建议您提交工单申请处理。", nil
	case success := <-firstByteChan:
		if !success {
			if err := scanner.Err(); err != nil {
				return "", fmt.Errorf("read first line: %w", err)
			}
			return "", fmt.Errorf("no data received")
		}
	}

	var fullResponse string

	// 处理第一条数据
	if err := p.processSSELine(scanner.Text(), &fullResponse, callback); err != nil {
		logger.Warn("Alibaba AI: Process first line error", zap.Error(err))
	}

	// 继续处理后续数据
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return fullResponse, ctx.Err()
		case <-p.interruptCh:
			return fullResponse, fmt.Errorf("interrupted")
		case <-p.hangupChan:
			return fullResponse, fmt.Errorf("hangup")
		default:
		}

		if err := p.processSSELine(scanner.Text(), &fullResponse, callback); err != nil {
			logger.Warn("Alibaba AI: Process line error", zap.Error(err))
		}
	}

	if err := scanner.Err(); err != nil {
		return fullResponse, fmt.Errorf("scan error: %w", err)
	}

	// 发送完成信号
	if callback != nil {
		callback("", true)
	}

	return fullResponse, nil
}

// processSSELine 处理 SSE 行
func (p *AlibabaProvider) processSSELine(line string, fullResponse *string, callback func(text string, isComplete bool) error) error {
	line = strings.TrimSpace(line)
	if line == "" || !strings.HasPrefix(line, "data:") {
		return nil
	}

	// 移除 "data:" 前缀
	data := strings.TrimPrefix(line, "data:")
	data = strings.TrimSpace(data)

	// 检查是否是流结束标记
	if data == "[DONE]" {
		logger.Debug("Alibaba AI: Stream completed")
		return nil
	}

	// 解析JSON
	var resp struct {
		Output struct {
			Text         string `json:"text"`
			FinishReason string `json:"finish_reason"`
		} `json:"output"`
	}

	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}

	// 提取文本内容
	if resp.Output.Text != "" {
		if msgResp, ok := parseAlibabaPayload(resp.Output.Text); ok && strings.TrimSpace(msgResp.Message) != "" {
			logger.Info("Alibaba AI stream json parsed",
				zap.String("action", strings.TrimSpace(msgResp.Action)),
				zap.Int("needperson", msgResp.NeedPerson),
				zap.Int("needhangup", msgResp.NeedHangup),
				zap.String("message_preview", previewText(msgResp.Message, 120)),
			)
			*fullResponse += msgResp.Message
			_ = p.maybeInvokeActions(msgResp)
			if callback != nil {
				if err := callback(msgResp.Message, resp.Output.FinishReason == "stop"); err != nil {
					return err
				}
			}
		} else {
			logger.Warn("Alibaba AI stream json parse failed (using raw text)",
				zap.String("output_preview", previewText(resp.Output.Text, 240)),
			)
			// Avoid speaking raw JSON payloads to callers.
			raw := strings.TrimSpace(resp.Output.Text)
			if strings.HasPrefix(raw, "{") && strings.HasSuffix(raw, "}") {
				return nil
			}
			*fullResponse += resp.Output.Text
			if callback != nil {
				if err := callback(resp.Output.Text, resp.Output.FinishReason == "stop"); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (p *AlibabaProvider) maybeInvokeActions(msg alibabaMessagePayload) error {
	action := strings.TrimSpace(strings.ToLower(msg.Action))
	if msg.NeedPerson == 1 || action == "transfer_to_agent" || action == "transfer" {
		logger.Info("Alibaba AI action resolved", zap.String("action", "transfer_to_agent"))
		p.mutex.Lock()
		p.pendingAction = "transfer_to_agent"
		p.mutex.Unlock()
		return nil
	}
	// hangup action intentionally ignored in SIP voice flow (no hangup tool registered).
	logger.Debug("Alibaba AI no actionable command in payload",
		zap.String("action", action),
		zap.Int("needperson", msg.NeedPerson),
		zap.Int("needhangup", msg.NeedHangup),
	)
	return nil
}

func (p *AlibabaProvider) invokeToolByName(name string, args map[string]interface{}) error {
	if p == nil || p.functionManager == nil {
		return nil
	}
	def, ok := p.functionManager.GetTool(name)
	if !ok || def == nil || def.Callback == nil {
		return nil
	}
	_, err := def.Callback(args, p.functionManager.GetLLMService())
	return err
}

func (p *AlibabaProvider) composePrompt(userText string) string {
	sys := strings.TrimSpace(p.systemMsg)
	userText = strings.TrimSpace(userText)
	contract := `你必须只输出单行JSON，不要输出任何额外文本、markdown或代码块。JSON结构：
{"message":"给用户播报的自然语言","action":"none|transfer_to_agent|hangup_call","needperson":0或1,"needhangup":0或1}
规则：
1) 若用户要求转人工，action=transfer_to_agent, needperson=1, needhangup=0。
2) 若用户要求挂断/结束通话（如“再见、挂了”），action=hangup_call, needhangup=1, needperson=0。
3) 其他情况 action=none, needperson=0, needhangup=0。
4) message 使用中文，简短自然。`
	if sys == "" {
		return fmt.Sprintf("%s\n\n用户输入：%s", contract, userText)
	}
	return fmt.Sprintf("系统指令：%s\n\n%s\n\n用户输入：%s", sys, contract, userText)
}

// RegisterFunctionTool 注册函数工具
func (p *AlibabaProvider) RegisterFunctionTool(name, description string, parameters interface{}, callback FunctionToolCallback) {
	var params json.RawMessage
	if parameters != nil {
		if raw, ok := parameters.(json.RawMessage); ok {
			params = raw
		} else {
			bytes, _ := json.Marshal(parameters)
			params = json.RawMessage(bytes)
		}
	}
	p.functionManager.RegisterTool(name, description, params, callback)
}

// RegisterFunctionToolDefinition 通过定义结构注册工具
func (p *AlibabaProvider) RegisterFunctionToolDefinition(def *FunctionToolDefinition) {
	p.functionManager.RegisterToolDefinition(def)
}

// GetFunctionTools 获取所有可用的函数工具
func (p *AlibabaProvider) GetFunctionTools() []interface{} {
	return []interface{}{}
}

// ListFunctionTools 列出所有已注册的工具名称
func (p *AlibabaProvider) ListFunctionTools() []string {
	return p.functionManager.ListTools()
}

// GetLastUsage 获取最后一次调用的使用统计信息
func (p *AlibabaProvider) GetLastUsage() (Usage, bool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.lastUsage, p.lastUsageValid
}

// ResetMessages 重置对话历史
func (p *AlibabaProvider) ResetMessages() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.messages = make([]Message, 0)
}

// SetSystemPrompt 设置系统提示词
func (p *AlibabaProvider) SetSystemPrompt(systemPrompt string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.systemMsg = systemPrompt
}

// GetMessages 获取当前对话历史
func (p *AlibabaProvider) GetMessages() []Message {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.messages
}

// Interrupt 中断当前请求
func (p *AlibabaProvider) Interrupt() {
	select {
	case p.interruptCh <- struct{}{}:
	default:
	}
}

// Hangup 挂断（清理资源）
func (p *AlibabaProvider) Hangup() {
	close(p.hangupChan)
}
