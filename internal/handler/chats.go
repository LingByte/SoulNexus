package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/voicedialog"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type ChatRequest struct {
	AgentID      int64   `json:"agentId" binding:"required"`
	SystemPrompt string  `json:"systemPrompt"`
	Speaker      string  `json:"speaker"`
	Language     string  `json:"language"`
	ApiKey       string  `json:"apiKey"`
	ApiSecret    string  `json:"apiSecret"`
	PersonaTag   string  `json:"personaTag"`
	Temperature  float32 `json:"temperature"`
	MaxTokens    int     `json:"maxTokens"`
}

// ChatResponse 只是响应状态（实际处理是语音流）
type ChatResponse struct {
	Message string `json:"message"`
}

type ChatSessionMap struct {
	AgentID int64
	//SdkClient   *client.Client
}

func (h *Handlers) Chat(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User is not logged in.", nil)
		return
	}

	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效"})
		return
	}

	cred, err := models.GetUserCredentialByApiSecretAndApiKey(h.db, req.ApiKey, req.ApiSecret)
	if err != nil {
		response.Fail(c, "Database error: "+err.Error(), nil)
		return
	}
	if cred == nil {
		response.Fail(c, "Secret or key is invalid or wrong. Please check again.", nil)
		return
	}
}

func (h *Handlers) StopChat(c *gin.Context) {
	sessionID := c.Query("sessionId")
	if sessionID == "" {
		response.Fail(c, "not find param sessionId ", nil)
		return
	}
	response.Fail(c, "未找到活动通话", nil)
}

func (h *Handlers) ChatStream(c *gin.Context) {
	sessionId := c.Query("sessionId")
	if sessionId == "" {
		response.Fail(c, "not find param sessionId ", nil)
		return
	}

	// 设置 SSE 响应头
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Flush()
}

func (h *Handlers) getChatSessionLog(c *gin.Context) {
	// 获取当前登录用户
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User is not logged in.", nil)
		return
	}

	// 获取分页参数
	pageSize := c.DefaultQuery("pageSize", "10")
	cursor := c.DefaultQuery("cursor", "")

	pageSizeInt, _ := strconv.Atoi(pageSize)
	if pageSizeInt <= 0 {
		response.Fail(c, "Invalid pageSize", nil)
		return
	}

	// 解析游标
	var cursorID int64
	if cursor != "" {
		var err error
		cursorID, err = strconv.ParseInt(cursor, 10, 64)
		if err != nil {
			response.Fail(c, "Invalid cursor", nil)
			return
		}
	}

	// 使用新的模型方法获取聊天记录
	logs, err := models.GetChatSessionLogs(h.db, user.ID, pageSizeInt, cursorID)
	if err != nil {
		response.Fail(c, "Failed to fetch chat logs", err.Error())
		return
	}

	// 获取下一页的游标
	var nextCursor int64
	if len(logs) > 0 {
		nextCursor = logs[len(logs)-1].ID
	}

	// 返回成功的响应
	response.Success(c, "Fetched chat session logs successfully", map[string]interface{}{
		"logs":        logs,
		"nextCursor":  nextCursor,
		"hasMoreData": len(logs) == pageSizeInt,
	})
}

func (h *Handlers) getChatSessionLogDetail(c *gin.Context) {
	// 获取当前登录用户
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User is not logged in.", nil)
		return
	}

	id := c.Param("id")
	if id == "" {
		response.Fail(c, "not find param id", nil)
		return
	}

	// 解析ID
	logID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		response.Fail(c, "Invalid log ID", nil)
		return
	}

	// 使用新的模型方法获取聊天记录详情
	fmt.Printf("查询聊天记录详情: logID=%d, userID=%d\n", logID, user.ID)

	detail, err := models.GetChatSessionLogDetail(h.db, logID, user.ID)
	if err != nil {
		fmt.Printf("查询聊天记录详情失败: %v\n", err)
		response.Fail(c, "Failed to fetch chat log", err.Error())
		return
	}

	fmt.Printf("查询聊天记录详情成功: %+v\n", detail)

	response.Success(c, "Fetched chat log successfully", detail)
}

// getChatSessionLogsBySession 获取指定会话的所有聊天记录
func (h *Handlers) getChatSessionLogsBySession(c *gin.Context) {
	// 获取当前登录用户
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User is not logged in.", nil)
		return
	}

	// 获取会话ID
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		response.Fail(c, "Session ID is required", nil)
		return
	}

	// 先确认 session 归属
	var session models.ChatSession
	if err := h.db.Where("id = ? AND user_id = ?", sessionID, fmt.Sprintf("%d", user.ID)).First(&session).Error; err != nil {
		response.Fail(c, "Failed to fetch chat logs", err.Error())
		return
	}

	var messages []models.ChatMessage
	if err := h.db.Where("session_id = ?", sessionID).Order("created_at ASC").Find(&messages).Error; err != nil {
		response.Fail(c, "Failed to fetch chat logs", err.Error())
		return
	}

	var agentName string
	_ = h.db.Table("agents").Where("id = ?", session.AgentID).Select("name").Scan(&agentName)

	var usages []models.LLMUsage
	_ = h.db.Where("session_id = ?", sessionID).Order("requested_at ASC, created_at ASC").Find(&usages).Error
	usageByRequestID := make(map[string]models.LLMUsage, len(usages))
	for _, usage := range usages {
		reqID := strings.TrimSpace(usage.RequestID)
		if reqID != "" {
			usageByRequestID[reqID] = usage
		}
	}
	usageCursor := 0

	details := make([]models.ChatSessionLogDetail, 0, len(messages))
	for i := 0; i < len(messages); i++ {
		if messages[i].Role != "user" {
			continue
		}
		userMsg := messages[i]

		var agentMsg *models.ChatMessage
		for j := i + 1; j < len(messages); j++ {
			if messages[j].Role == "assistant" {
				agentMsg = &messages[j]
				break
			}
		}

		detail := models.ChatSessionLogDetail{
			ID:          userMsg.CreatedAt.UnixMilli(),
			SessionID:   sessionID,
			AgentID:     session.AgentID,
			AgentName:   agentName,
			ChatType:    models.ChatTypeText,
			UserMessage: userMsg.Content,
			CreatedAt:   userMsg.CreatedAt,
			UpdatedAt:   userMsg.UpdatedAt,
		}
		if agentMsg != nil {
			detail.AgentMessage = agentMsg.Content
			detail.UpdatedAt = agentMsg.UpdatedAt
			reqID := strings.TrimSpace(agentMsg.RequestID)
			if reqID != "" {
				if usage, ok := usageByRequestID[reqID]; ok {
					usageCopy := usage
					detail.LLMUsage = &usageCopy
				}
			}
			if detail.LLMUsage == nil && usageCursor < len(usages) {
				usageCopy := usages[usageCursor]
				detail.LLMUsage = &usageCopy
				usageCursor++
			}
		}
		details = append(details, detail)
	}

	response.Success(c, "Fetched chat session logs successfully", details)
}

// getChatSessionLogByAgent returns chat logs for an agent
func (h *Handlers) getChatSessionLogByAgent(c *gin.Context) {
	// 获取当前登录用户
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User is not logged in.", nil)
		return
	}

	agentIDStr := c.Param("agentId")
	if agentIDStr == "" {
		response.Fail(c, "Agent ID is required", nil)
		return
	}

	agentID, err := strconv.ParseInt(agentIDStr, 10, 64)
	if err != nil {
		response.Fail(c, "Invalid agent ID", nil)
		return
	}

	// 获取分页参数
	pageSize := c.DefaultQuery("pageSize", "10")
	cursor := c.DefaultQuery("cursor", "")

	pageSizeInt, _ := strconv.Atoi(pageSize)
	if pageSizeInt <= 0 {
		response.Fail(c, "Invalid pageSize", nil)
		return
	}

	// 解析游标
	var cursorID int64
	if cursor != "" {
		cursorID, err = strconv.ParseInt(cursor, 10, 64)
		if err != nil {
			response.Fail(c, "Invalid cursor", nil)
			return
		}
	}

	// 使用新会话表查询并在内存中过滤助手
	allLogs, err := models.GetChatSessionLogs(h.db, user.ID, pageSizeInt*5, cursorID)
	if err != nil {
		response.Fail(c, "Failed to fetch chat logs", err.Error())
		return
	}
	logs := make([]models.ChatSessionLogSummary, 0, pageSizeInt)
	for _, item := range allLogs {
		if item.AgentID == agentID {
			logs = append(logs, item)
			if len(logs) >= pageSizeInt {
				break
			}
		}
	}

	// 获取下一页的游标
	var nextCursor int64
	if len(logs) > 0 {
		nextCursor = logs[len(logs)-1].ID
	}

	// 返回成功的响应
	response.Success(c, "Fetched chat session logs successfully", map[string]interface{}{
		"logs":        logs,
		"nextCursor":  nextCursor,
		"hasMoreData": len(logs) == pageSizeInt,
		"agentId":     agentID,
	})
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// overlayAuthFromPayloadQuery fills apiKey / apiSecret / agentId from the
// URL query "payload" when those top-level query keys are empty. Payload
// must be a JSON object (e.g. from cmd/voice WebRTC offer passthrough).
func overlayAuthFromPayloadQuery(apiKey, apiSecret, agentIDStr *string, payloadQuery string) error {
	payloadQuery = strings.TrimSpace(payloadQuery)
	if payloadQuery == "" {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(payloadQuery), &m); err != nil {
		return err
	}
	if *apiKey == "" {
		*apiKey = dialogPayloadString(m["apiKey"])
	}
	if *apiSecret == "" {
		*apiSecret = dialogPayloadString(m["apiSecret"])
	}
	if *agentIDStr == "" {
		*agentIDStr = dialogPayloadString(m["agentId"])
	}
	return nil
}

func dialogPayloadString(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		return strconv.FormatInt(int64(t), 10)
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

func (h *Handlers) handleConnection(c *gin.Context) {
	apiKey := c.Query("apiKey")
	apiSecret := c.Query("apiSecret")
	agentIDStr := c.Query("agentId")

	if err := overlayAuthFromPayloadQuery(&apiKey, &apiSecret, &agentIDStr, c.Query("payload")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload query JSON: " + err.Error()})
		c.Abort()
		return
	}
	if apiKey == "" || apiSecret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required parameters: apiKey and apiSecret are required"})
		c.Abort()
		return
	}
	if agentIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required parameter: agentId is required"})
		c.Abort()
		return
	}

	cred, err := models.GetUserCredentialByApiSecretAndApiKey(h.db, apiKey, apiSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
		c.Abort()
		return
	}
	if cred == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		c.Abort()
		return
	}

	agentID, err := strconv.ParseInt(agentIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agentId format"})
		c.Abort()
		return
	}

	var agent models.Agent
	if err := h.db.First(&agent, agentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		c.Abort()
		return
	}

	if cred.GroupID != agent.GroupID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied: credential and agent are not in the same organization"})
		c.Abort()
		return
	}

	callID := strings.TrimSpace(c.Query("call_id"))
	if callID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "缺少 call_id。本 URL 仅供 cmd/voice 作为对话面拨入；请在拨号 URL 上携带 call_id（gateway 会自动追加）。浏览器实时语音请在 VoiceAssistant 内建 WebRTC 发起；鉴权参数放在 POST body 的 payload JSON 内或 query 的 apiKey/apiSecret/agentId。",
		})
		c.Abort()
		return
	}

	if strings.TrimSpace(cred.LLMProvider) == "" || strings.TrimSpace(cred.LLMApiKey) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Credential is missing LLM configuration (llmProvider, llmApiKey). Configure the API credential for this key pair."})
		c.Abort()
		return
	}

	systemPrompt := agent.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = "你是一个友好的AI助手，请用简洁明了的语言回答问题。"
	}

	maxTokens := agent.MaxTokens
	temperature := agent.Temperature
	if temperature == 0 {
		temperature = 0.7
	}

	model := strings.TrimSpace(agent.LLMModel)
	if model == "" {
		model = voicedialog.DefaultModelForProvider(cred.LLMProvider)
	}

	fallback := "您好，我现在没法回答这个问题，请稍后再试。"

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[chat/call] WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	cfg := voicedialog.Config{
		Provider:     cred.LLMProvider,
		APIKey:       cred.LLMApiKey,
		APIURL:       cred.LLMApiURL,
		Model:        model,
		SystemPrompt: systemPrompt,
		MaxTokens:    maxTokens,
		Temperature:  temperature,
	}
	voicedialog.Serve(context.Background(), conn, callID, cfg, fallback)
}
