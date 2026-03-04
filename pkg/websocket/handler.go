package websocket

import (
	"fmt"
	"net/http"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler WebSocket HTTP处理器
type Handler struct {
	hub *Hub
}

// NewHandler 创建新的WebSocket处理器
func NewHandler(hub *Hub) *Handler {
	return &Handler{
		hub: hub,
	}
}

// RegisterRoutes 统一注册路由
func RegisterRoutes(r *gin.Engine, handler *Handler) {
	r.GET(RouteWebSocket, handler.HandleWebSocket)
	r.GET(RouteWebSocketStats, handler.GetStats)
	r.GET(RouteWebSocketHealth, handler.HealthCheck)
	r.POST(RouteWebSocketMessage, handler.SendMessage)
	r.POST(RouteWebSocketBroadcast, handler.BroadcastMessage)
}

// HandleWebSocket 处理WebSocket连接请求
func (h *Handler) HandleWebSocket(c *gin.Context) {
	// 获取用户ID（从认证中间件中获取）
	userObj, exists := c.Get(constants.UserField)
	if !exists {
		logger.Error("未认证的用户")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证的用户"})
		return
	}

	var userIDStr string
	// 处理不同类型的用户ID：可能是 *User 对象或字符串
	switch v := userObj.(type) {
	case *models.User:
		// 从 User 对象中提取 ID 并转换为字符串
		userIDStr = fmt.Sprintf("%d", v.ID)
	case string:
		// 如果已经是字符串，直接使用
		userIDStr = v
	default:
		logger.Error("无效的用户ID类型", zap.String("type", fmt.Sprintf("%T", userObj)))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无效的用户ID"})
		return
	}

	if userIDStr == "" {
		logger.Error("用户ID为空")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "用户ID为空"})
		return
	}

	// 处理WebSocket升级
	HandleWebSocket(h.hub, c.Writer, c.Request, userIDStr)
}

// HandleAnonymousWebSocket 处理匿名WebSocket连接（可选）
func (h *Handler) HandleAnonymousWebSocket(c *gin.Context) {
	// 生成匿名用户ID
	anonymousID := "anonymous_" + c.Request.Header.Get("X-Request-ID")
	if anonymousID == "anonymous_0" {
		anonymousID = "anonymous_" + c.Request.Header.Get("X-Real-IP")
	}

	// 处理WebSocket升级
	HandleWebSocket(h.hub, c.Writer, c.Request, anonymousID)
}

// GetStats 获取WebSocket统计信息
func (h *Handler) GetStats(c *gin.Context) {
	stats := gin.H{
		"total_connections":    h.hub.GetConnectionCount(),
		"max_connections":      h.hub.config.MaxConnections,
		"heartbeat_interval":   h.hub.config.HeartbeatInterval.String(),
		"connection_timeout":   h.hub.config.ConnectionTimeout.String(),
		"message_buffer_size":  h.hub.config.MessageBufferSize,
		"enable_compression":   h.hub.config.EnableCompression,
		"enable_message_queue": h.hub.config.EnableMessageQueue,
		"message_queue_size":   h.hub.config.MessageQueueSize,
		"enable_cluster":       h.hub.config.EnableCluster,
		"cluster_node_id":      h.hub.config.ClusterNodeID,
		"read_buffer_size":     h.hub.config.ReadBufferSize,
		"write_buffer_size":    h.hub.config.WriteBufferSize,
		"max_message_size":     h.hub.config.MaxMessageSize,
		"shard_count":          h.hub.config.ShardCount,
		"broadcast_workers":    h.hub.config.BroadcastWorkerCount,
		"drop_on_full":         h.hub.config.DropOnFull,
		"compression_level":    h.hub.config.CompressionLevel,
	}

	c.JSON(http.StatusOK, stats)
}

// GetUserStats 获取特定用户的连接统计
func (h *Handler) GetUserStats(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户ID不能为空"})
		return
	}

	connectionCount := h.hub.GetUserConnections(userID)
	stats := gin.H{
		"user_id":          userID,
		"connection_count": connectionCount,
		"max_connections":  h.hub.config.MaxConnections,
	}

	c.JSON(http.StatusOK, stats)
}

// GetGroupStats 获取特定组的连接统计
func (h *Handler) GetGroupStats(c *gin.Context) {
	groupName := c.Param("group")
	if groupName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "组名不能为空"})
		return
	}

	connectionCount := h.hub.GetGroupConnections(groupName)
	stats := gin.H{
		"group":            groupName,
		"connection_count": connectionCount,
		"max_connections":  h.hub.config.MaxConnections,
	}

	c.JSON(http.StatusOK, stats)
}

// SendMessage 发送消息到指定用户或组
func (h *Handler) SendMessage(c *gin.Context) {
	var request struct {
		Type  string      `json:"type" binding:"required"`
		Data  interface{} `json:"data"`
		To    string      `json:"to,omitempty"`
		Group string      `json:"group,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}

	// 验证请求
	if request.To == "" && request.Group == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "必须指定目标用户或组"})
		return
	}

	// 创建消息
	message := &Message{
		Type:  request.Type,
		Data:  request.Data,
		To:    request.To,
		Group: request.Group,
	}

	// 广播消息
	h.hub.broadcast <- message

	c.JSON(http.StatusOK, gin.H{"message": "消息已发送"})
}

// BroadcastMessage 广播消息给所有连接
func (h *Handler) BroadcastMessage(c *gin.Context) {
	var request struct {
		Type string      `json:"type" binding:"required"`
		Data interface{} `json:"data"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}

	// 创建消息
	message := &Message{
		Type: request.Type,
		Data: request.Data,
	}

	// 广播消息
	h.hub.broadcast <- message

	c.JSON(http.StatusOK, gin.H{"message": "广播消息已发送"})
}

// DisconnectUser 断开指定用户的所有连接
func (h *Handler) DisconnectUser(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户ID不能为空"})
		return
	}

	// 获取用户的所有连接
	h.hub.mu.RLock()
	connections, exists := h.hub.userConnections[userID]
	h.hub.mu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户没有活跃连接"})
		return
	}

	// 断开所有连接
	disconnectedCount := 0
	for connID := range connections {
		if conn, ok := h.hub.connections[connID]; ok {
			conn.Conn.Close()
			disconnectedCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":            "用户连接已断开",
		"user_id":            userID,
		"disconnected_count": disconnectedCount,
	})
}

// DisconnectGroup 断开指定组的所有连接
func (h *Handler) DisconnectGroup(c *gin.Context) {
	groupName := c.Param("group")
	if groupName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "组名不能为空"})
		return
	}

	// 获取组的所有连接
	h.hub.mu.RLock()
	connections, exists := h.hub.groupConnections[groupName]
	h.hub.mu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "组没有活跃连接"})
		return
	}

	// 断开所有连接
	disconnectedCount := 0
	for connID := range connections {
		if conn, ok := h.hub.connections[connID]; ok {
			conn.Conn.Close()
			disconnectedCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":            "组连接已断开",
		"group":              groupName,
		"disconnected_count": disconnectedCount,
	})
}

// HealthCheck WebSocket健康检查
func (h *Handler) HealthCheck(c *gin.Context) {
	// 检查Hub是否正常运行
	if h.hub.ctx.Err() != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "unhealthy",
			"error":   "WebSocket Hub已关闭",
			"details": h.hub.ctx.Err().Error(),
		})
		return
	}

	// 检查连接数是否正常
	totalConnections := h.hub.GetConnectionCount()
	maxConnections := h.hub.config.MaxConnections

	status := "healthy"
	if totalConnections >= maxConnections*9/10 { // 90%以上认为警告
		status = "warning"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":            status,
		"total_connections": totalConnections,
		"max_connections":   maxConnections,
		"connection_usage":  float64(totalConnections) / float64(maxConnections) * 100,
		"hub_running":       true,
		"timestamp":         time.Now().Unix(),
	})
}
