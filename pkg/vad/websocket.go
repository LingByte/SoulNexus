package vad

import (
	"encoding/base64"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// WebSocketMessage WebSocket 消息
type WebSocketMessage struct {
	Type      string          `json:"type"`   // "audio", "reset"
	Data      string          `json:"data"`   // Base64 编码的音频数据
	Format    string          `json:"format"` // "pcm" 或 "opus"
	SessionID string          `json:"session_id"`
	Result    *DetectResponse `json:"result,omitempty"`
}

// WebSocketHandler WebSocket 处理器
type WebSocketHandler struct {
	sessionManager *SessionManager
	logger         *zap.Logger
	mu             sync.RWMutex
	connections    map[string]*WebSocketConnection
}

// WebSocketConnection WebSocket 连接
type WebSocketConnection struct {
	conn      *websocket.Conn
	sessionID string
	mu        sync.RWMutex
	logger    *zap.Logger
	closeChan chan struct{}
}

// NewWebSocketHandler 创建新的 WebSocket 处理器
func NewWebSocketHandler(sessionManager *SessionManager, logger *zap.Logger) *WebSocketHandler {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &WebSocketHandler{
		sessionManager: sessionManager,
		logger:         logger,
		connections:    make(map[string]*WebSocketConnection),
	}
}

// HandleConnection 处理 WebSocket 连接
func (h *WebSocketHandler) HandleConnection(conn *websocket.Conn, sessionID string) error {
	wsConn := &WebSocketConnection{
		conn:      conn,
		sessionID: sessionID,
		logger:    h.logger,
		closeChan: make(chan struct{}),
	}

	h.mu.Lock()
	h.connections[sessionID] = wsConn
	h.mu.Unlock()

	h.logger.Info("WebSocket connection established", zap.String("session_id", sessionID))

	defer func() {
		h.mu.Lock()
		delete(h.connections, sessionID)
		h.mu.Unlock()
		conn.Close()
		h.logger.Info("WebSocket connection closed", zap.String("session_id", sessionID))
	}()

	// 设置读取超时
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// 启动心跳
	go wsConn.startHeartbeat()

	// 处理消息
	for {
		select {
		case <-wsConn.closeChan:
			return nil
		default:
		}

		var msg WebSocketMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.logger.Error("WebSocket error", zap.Error(err))
			}
			return err
		}

		// 处理不同类型的消息
		switch msg.Type {
		case "audio":
			h.handleAudioMessage(wsConn, &msg)
		case "reset":
			h.handleResetMessage(wsConn, &msg)
		default:
			h.logger.Warn("unknown message type", zap.String("type", msg.Type))
		}
	}
}

// handleAudioMessage 处理音频消息
func (h *WebSocketHandler) handleAudioMessage(wsConn *WebSocketConnection, msg *WebSocketMessage) {
	// 解码 Base64 音频数据
	audioData, err := decodeBase64(msg.Data)
	if err != nil {
		h.logger.Error("failed to decode audio data", zap.Error(err))
		return
	}

	// 处理音频
	result, err := h.sessionManager.ProcessAudio(wsConn.sessionID, audioData, msg.Format)
	if err != nil {
		h.logger.Error("failed to process audio", zap.Error(err))
		return
	}

	// 发送结果
	response := WebSocketMessage{
		Type:      "vad_result",
		SessionID: wsConn.sessionID,
		Result:    result,
	}
	wsConn.sendMessage(&response)
}

// handleResetMessage 处理重置消息
func (h *WebSocketHandler) handleResetMessage(wsConn *WebSocketConnection, msg *WebSocketMessage) {
	err := h.sessionManager.ResetSession(wsConn.sessionID)
	if err != nil {
		h.logger.Error("failed to reset session", zap.Error(err))
		return
	}

	h.logger.Info("session reset", zap.String("session_id", wsConn.sessionID))
}

// startHeartbeat 启动心跳
func (wsConn *WebSocketConnection) startHeartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			wsConn.mu.RLock()
			conn := wsConn.conn
			wsConn.mu.RUnlock()

			if err := conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				wsConn.logger.Error("failed to send ping", zap.Error(err))
				close(wsConn.closeChan)
				return
			}
		case <-wsConn.closeChan:
			return
		}
	}
}

// sendMessage 发送消息
func (wsConn *WebSocketConnection) sendMessage(msg *WebSocketMessage) error {
	wsConn.mu.Lock()
	defer wsConn.mu.Unlock()

	wsConn.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return wsConn.conn.WriteJSON(msg)
}

// Close 关闭连接
func (wsConn *WebSocketConnection) Close() error {
	close(wsConn.closeChan)
	return wsConn.conn.Close()
}

// decodeBase64 解码 Base64 字符串
func decodeBase64(data string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(data)
}
