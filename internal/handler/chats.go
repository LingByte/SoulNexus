package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/LingByte/SoulNexus/pkg/graph"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/webrtc/constants"
	"github.com/LingByte/SoulNexus/pkg/webrtc/rtcmedia"
	transports "github.com/LingByte/SoulNexus/pkg/webrtc/transport"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

// Constants
const (
	// Connection configuration
	maxConnectionRetries       = 50
	connectionRetryDelay       = 100 * time.Millisecond
	connectionStateLogInterval = 10
	connectionReadyDelay       = 200 * time.Millisecond
)

// ClientManager manages WebRTC client connections
type ClientManager struct {
	clients map[string]*transports.AIClient
	mutex   sync.RWMutex
}

func NewClientManager() *ClientManager {
	return &ClientManager{
		clients: make(map[string]*transports.AIClient),
	}
}

func (m *ClientManager) AddClient(sessionID string, client *transports.AIClient) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.clients[sessionID] = client
}

func (m *ClientManager) RemoveClient(sessionID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.clients, sessionID)
}

func (m *ClientManager) GetClient(sessionID string) (*transports.AIClient, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	client, exists := m.clients[sessionID]
	return client, exists
}

type ChatRequest struct {
	AssistantID  int64   `json:"assistantId" binding:"required"`
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
	AssistantID int64
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

	// 获取该会话的所有记录
	logs, err := models.GetChatSessionLogsBySession(h.db, sessionID, user.ID)
	if err != nil {
		response.Fail(c, "Failed to fetch chat logs", err.Error())
		return
	}

	// 转换为详情格式
	details := make([]models.ChatSessionLogDetail, 0, len(logs))
	for _, log := range logs {
		// 获取助手名称
		var assistantName string
		h.db.Table("assistants").Where("id = ?", log.AssistantID).Select("name").Scan(&assistantName)

		detail := models.ChatSessionLogDetail{
			ID:            log.ID,
			SessionID:     log.SessionID,
			AssistantID:   log.AssistantID,
			AssistantName: assistantName,
			ChatType:      log.ChatType,
			UserMessage:   log.UserMessage,
			AgentMessage:  log.AgentMessage,
			AudioURL:      log.AudioURL,
			Duration:      log.Duration,
			CreatedAt:     log.CreatedAt,
			UpdatedAt:     log.UpdatedAt,
		}

		details = append(details, detail)
	}

	response.Success(c, "Fetched chat session logs successfully", details)
}

// getChatSessionLogByAssistant 获取指定助手的聊天记录
func (h *Handlers) getChatSessionLogByAssistant(c *gin.Context) {
	// 获取当前登录用户
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User is not logged in.", nil)
		return
	}

	// 获取助手ID
	assistantIDStr := c.Param("assistantId")
	if assistantIDStr == "" {
		response.Fail(c, "Assistant ID is required", nil)
		return
	}

	assistantID, err := strconv.ParseInt(assistantIDStr, 10, 64)
	if err != nil {
		response.Fail(c, "Invalid assistant ID", nil)
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
		if item.AssistantID == assistantID {
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
		"assistantId": assistantID,
	})
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var manager = NewClientManager()

// SignalMessage represents a WebSocket signaling message
type SignalMessage struct {
	Type      string      `json:"type"`
	SessionID string      `json:"session_id,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}

func (h *Handlers) handleConnection(c *gin.Context) {
	// 从 URL 参数中获取认证信息
	apiKey := c.Query("apiKey")
	apiSecret := c.Query("apiSecret")
	assistantIDStr := c.Query("assistantId")

	// 验证必需参数 - 在 WebSocket 升级之前验证，失败时直接返回 HTTP 错误
	if apiKey == "" || apiSecret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required parameters: apiKey and apiSecret are required"})
		c.Abort()
		return
	}
	if assistantIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required parameter: assistantId is required"})
		c.Abort()
		return
	}

	// 立即验证认证信息
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

	// 解析 assistantId
	assistantID, err := strconv.ParseInt(assistantIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid assistantId format"})
		c.Abort()
		return
	}

	// 查询 assistant 配置
	var assistant models.Assistant
	if err := h.db.First(&assistant, assistantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Assistant not found"})
		c.Abort()
		return
	}

	// 验证 assistant 是否属于该用户（通过 credential 的 UserID）
	if assistant.UserID != cred.UserID {
		// 检查是否是组织共享的助手
		if assistant.GroupID == nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied: assistant does not belong to you"})
			c.Abort()
			return
		}
		// TODO: 可以在这里添加组织成员权限检查
	}

	systemPrompt := assistant.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = "你是一个友好的AI助手，请用简洁明了的语言回答问题。"
	}

	// 如果开启了图记忆功能，则尝试从 Neo4j 中获取该用户的长期偏好主题，并拼接到系统提示词中
	if config.GlobalConfig.Services.GraphMemory.Neo4j.Enabled && assistant.EnableGraphMemory {
		if store := graph.GetDefaultStore(); store != nil {
			ctx := c.Request.Context()
			if userCtx, err := store.GetUserContext(ctx, cred.UserID, assistantID); err == nil {
				if len(userCtx.Topics) > 0 {
					// 构建一段自然语言描述用户长期偏好
					preferenceText := fmt.Sprintf("该用户在历史对话中经常讨论这些主题：%s。请在回答时优先从这些兴趣和习惯的角度来组织内容，让风格尽量贴近他的偏好。",
						strings.Join(userCtx.Topics, "、"))
					if systemPrompt == "" {
						systemPrompt = preferenceText
					} else {
						systemPrompt = systemPrompt + "\n\n" + preferenceText
					}
				}
			}
		}
	}

	maxTokens := assistant.MaxTokens
	if maxTokens == 0 {
		maxTokens = 0 // 0 表示不限制
	}

	temperature := assistant.Temperature
	if temperature == 0 {
		temperature = 0.7 // 默认值
	}

	language := assistant.Language
	if language == "" {
		language = "zh" // 默认中文
	}

	speaker := assistant.Speaker
	llmModel := assistant.LLMModel

	// 转换 assistantID 为 *uint
	aid := uint(assistantID)

	// 升级 HTTP 请求为 WebSocket 连接
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Error upgrading connection:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to upgrade connection"})
		return
	}
	defer conn.Close()
	sessionID := fmt.Sprintf("session_%d", time.Now().UnixNano())

	// Create WebRTC transport
	transport := rtcmedia.NewWebRTCTransport(rtcmedia.WebRTCOption{
		Codec: constants.CodecPCMA,
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{
				"stun:stun.l.google.com:19302",
				"stun:stun1.l.google.com:19302",
				"stun:stun2.l.google.com:19302",
				"stun:stun3.l.google.com:19302",
				"stun:stun4.l.google.com:19302",
				"stun:stun.cloudflare.com:3478",
				"stun:stun.voipbuster.com:3478",
				"stun:stun.voipstunt.com:3478",
				"stun:stun.sipgate.net:3478",
				"stun:stun.miwifi.com:3478",
			}},
		},
		StreamID:   "lingecho_ai_server",
		ICETimeout: constants.DefaultICETimeout,
	})
	if err := transport.NewPeerConnection(); err != nil {
		log.Printf("webrtc: NewPeerConnection: %v", err)
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"webrtc init failed"}`))
		return
	}

	// Use credential and assistant configuration to initialize services
	aiClient, err := transports.NewAIClientWithCredential(
		conn,
		transport,
		sessionID,
		"",
		h.db,
		cred.UserID,
		cred.ID,
		&aid,
		cred,
		systemPrompt,
		maxTokens,
		temperature,
		language,
		speaker,
		llmModel,
	)
	if err != nil {
		log.Printf("[Server] Failed to create AI client: %v", err)
		return
	}

	// Set up OnTrack callback BEFORE handling any signaling messages
	// This is critical - OnTrack must be set up early to catch the track when it arrives
	var wsWriteMu sync.Mutex
	sendSignal := func(msg SignalMessage) {
		wsWriteMu.Lock()
		defer wsWriteMu.Unlock()
		if err := conn.WriteJSON(msg); err != nil {
			log.Printf("[Server] Failed to push signal message %s: %v", msg.Type, err)
		}
	}

	aiClient.SetEventCallbacks(
		func(text string, isFinal bool) {
			sendSignal(SignalMessage{
				Type:      "asrFinal",
				SessionID: sessionID,
				Data: map[string]interface{}{
					"text":    text,
					"isFinal": isFinal,
				},
			})
		},
		func(text string) {
			sendSignal(SignalMessage{
				Type:      "llm_response",
				SessionID: sessionID,
				Data: map[string]interface{}{
					"text": text,
				},
			})
		},
	)

	transport.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		aiClient.Mu.Lock()
		if !aiClient.AudioReceived {
			aiClient.AudioReceived = true
			aiClient.Mu.Unlock()
			fmt.Printf("[Server] ===== OnTrack callback FIRED! =====\n")
			fmt.Printf("[Server] Track codec: %s, SSRC: %d, ID: %s\n",
				track.Codec().MimeType, track.SSRC(), track.ID())

			// Start audio receiver in a separate goroutine
			go func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("[Server] Recovered from panic in audio receiver: %v", r)
					}
				}()
				if err := aiClient.StartAudioReceiverFromTrack(track); err != nil {
					log.Printf("[Server] Error in audio receiver: %v", err)
				}
			}()
		} else {
			aiClient.Mu.Unlock()
		}
	})
	fmt.Printf("[Server] OnTrack callback registered for client %s\n", sessionID)

	manager.AddClient(sessionID, aiClient)
	defer manager.RemoveClient(sessionID)
	defer aiClient.Close()

	// Send session ID to client
	initMsg := SignalMessage{
		Type:      "init",
		SessionID: sessionID,
	}
	if err := conn.WriteJSON(initMsg); err != nil {
		log.Printf("[Server] Failed to send init message: %v", err)
		return
	}

	// Handle incoming messages
	for {
		var msg SignalMessage
		if err := conn.ReadJSON(&msg); err != nil {
			// WebSocket 连接关闭或出错
			log.Printf("[Server] WebSocket connection closed or error: %v", err)
			// 确保清理资源
			if aiClient != nil {
				aiClient.Close()
			}
			break
		}

		// 处理关闭消息
		if msg.Type == "close" || msg.Type == "disconnect" {
			log.Printf("[Server] Received close/disconnect message from client")
			if aiClient != nil {
				aiClient.Close()
			}
			break
		}

		handleSignalMessage(aiClient, msg)
	}
}

// handleSignalMessage routes signaling messages
func handleSignalMessage(client *transports.AIClient, msg SignalMessage) {
	switch msg.Type {
	case "offer":
		handleOffer(client, msg)
	case "ice-candidate":
		handleICECandidate(client, msg)
	case "connected":
		handleConnection(client, msg)
	default:
		log.Printf("[Server] Unknown message type: %s", msg.Type)
	}
}

// handleICECandidate handles trickle ICE candidates from client
func handleICECandidate(client *transports.AIClient, msg SignalMessage) {
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		log.Println("[Server] Invalid ice-candidate data")
		return
	}
	candidate, _ := data["candidate"].(string)
	if candidate == "" {
		return
	}
	if err := client.Transport.AddICECandidate(candidate); err != nil {
		log.Printf("[Server] Failed to add ICE candidate: %v", err)
	}
}

// handleOffer handles the WebRTC offer
func handleOffer(client *transports.AIClient, msg SignalMessage) {
	offerData, ok := msg.Data.(map[string]interface{})
	if !ok {
		log.Println("[Server] Invalid offer data")
		return
	}

	offerStr, ok := offerData["sdp"].(string)
	if !ok {
		log.Println("[Server] Invalid offer SDP")
		return
	}

	// Debug: Check if offer SDP contains audio media
	if strings.Contains(offerStr, "m=audio") {
		fmt.Printf("[Server] Offer SDP contains audio media description\n")
	} else {
		fmt.Printf("[Server] WARNING: Offer SDP does NOT contain audio media description!\n")
		previewLen := 200
		if len(offerStr) < previewLen {
			previewLen = len(offerStr)
		}
		fmt.Printf("[Server] Offer SDP preview: %s...\n", offerStr[:previewLen])
	}

	if err := client.Transport.SetRemoteDescription(offerStr); err != nil {
		log.Printf("[Server] Error setting remote description: %v", err)
		return
	}
	fmt.Printf("[Server] Remote description set successfully\n")

	candidates, ok := offerData["candidates"].([]interface{})
	if !ok {
		log.Println("[Server] Invalid candidates data")
		return
	}

	candidateStrs := extractCandidates(candidates)
	answer, serverCandidates, err := client.Transport.CreateAnswer(candidateStrs)
	if err != nil {
		log.Printf("[Server] Error creating answer: %v", err)
		return
	}

	// Debug: Check if answer SDP contains audio media
	if strings.Contains(answer, "m=audio") {
		fmt.Printf("[Server] Answer SDP contains audio media description\n")
	} else {
		fmt.Printf("[Server] WARNING: Answer SDP does NOT contain audio media description!\n")
	}

	answerMsg := SignalMessage{
		Type:      "answer",
		SessionID: client.SessionID,
		Data: map[string]interface{}{
			"sdp":        answer,
			"candidates": serverCandidates,
		},
	}

	if err := client.Conn.WriteJSON(answerMsg); err != nil {
		log.Printf("[Server] Error sending answer: %v", err)
		return
	}

	fmt.Printf("[Server] Sent answer to client %s\n", client.SessionID)

	// Note: Audio receiving is now handled by the OnTrack callback
	// which is set up in websocketHandler before any signaling messages are processed
	// No need to wait here - OnTrack will fire automatically when the track arrives
	fmt.Println("[Server] Answer sent, waiting for OnTrack callback to fire when client sends audio...")
}

// extractCandidates extracts candidate strings
func extractCandidates(candidates []interface{}) []string {
	var candidateStrs []string
	for _, c := range candidates {
		if candidateStr, ok := c.(string); ok {
			candidateStrs = append(candidateStrs, candidateStr)
		}
	}
	return candidateStrs
}

// handleConnection handles connection established message (client confirmation)
func handleConnection(client *transports.AIClient, msg SignalMessage) {
	fmt.Printf("[Server] Client confirmed connection for session %s\n", client.SessionID)

	// Wait for connection to be established, then send greeting
	go func() {
		if err := waitForConnection(client.Transport); err != nil {
			log.Printf("[Server] Connection not established: %v", err)
			return
		}

		// Wait for txTrack to be ready
		maxWait := 50
		for i := 0; i < maxWait; i++ {
			txTrack := client.Transport.GetTxTrack()
			if txTrack != nil {
				fmt.Printf("[Server] txTrack is ready after %d attempts\n", i+1)
				break
			}
			if i == 0 {
				fmt.Printf("[Server] Waiting for txTrack to be ready...\n")
			}
			time.Sleep(50 * time.Millisecond)
		}

		// Keep this short to reduce first-response latency after connected.
		time.Sleep(80 * time.Millisecond)

		// Send greeting to start the conversation
		greeting := "你好，我是AI助手，很高兴和你对话。"
		fmt.Printf("[Server] Sending greeting: %s\n", greeting)
		client.GenerateTTS(greeting)
	}()
}

// waitForConnection waits for WebRTC connection
func waitForConnection(transport *rtcmedia.WebRTCTransport) error {
	for i := 0; i < maxConnectionRetries; i++ {
		state := transport.GetConnectionState()
		if state == webrtc.PeerConnectionStateConnected {
			return nil
		}

		if i%connectionStateLogInterval == 0 {
			fmt.Printf("[Server] Waiting for connection... (state: %s)\n", state.String())
		}

		time.Sleep(connectionRetryDelay)
	}

	return fmt.Errorf("connection timeout")
}
