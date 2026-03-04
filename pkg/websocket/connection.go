package websocket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// newUpgrader 根据配置创建WebSocket升级器
func newUpgrader(cfg *Config) websocket.Upgrader {
	up := websocket.Upgrader{
		ReadBufferSize:  cfg.ReadBufferSize,
		WriteBufferSize: cfg.WriteBufferSize,
		CheckOrigin: func(r *http.Request) bool {
			// 在生产环境中应该检查Origin
			return true
		},
		EnableCompression: cfg.EnableCompression,
	}
	return up
}

// HandleWebSocket 处理WebSocket连接
func HandleWebSocket(hub *Hub, w http.ResponseWriter, r *http.Request, userID string) {
	// 升级HTTP连接为WebSocket
	upgrader := newUpgrader(hub.config)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logrus.Errorf("WebSocket升级失败: %v", err)
		return
	}

	// 压缩设置
	if hub.config.EnableCompression {
		conn.EnableWriteCompression(true)
		if hub.config.CompressionLevel != 0 {
			_ = conn.SetCompressionLevel(hub.config.CompressionLevel)
		}
	}

	// 创建连接实例
	connection := &Connection{
		ID:       generateConnectionID(),
		UserID:   userID,
		Conn:     conn,
		Send:     make(chan []byte, hub.config.MessageBufferSize),
		Hub:      hub,
		LastPing: time.Now(),
		IsAlive:  true,
		Groups:   make(map[string]bool),
		Metadata: make(map[string]interface{}),
	}

	// 注册连接到Hub
	hub.register <- connection

	// 启动读写协程
	go connection.writePump()
	go connection.readPump()
}

// generateConnectionID 生成唯一的连接ID
func generateConnectionID() string {
	return fmt.Sprintf("conn_%d", time.Now().UnixNano())
}

// readPump 读取消息的协程
func (c *Connection) readPump() {
	defer func() {
		c.Hub.unregister <- c
		// 优雅关闭连接
		c.Conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(int64(c.Hub.config.MaxMessageSize))
	c.Conn.SetReadDeadline(time.Now().Add(c.Hub.config.ConnectionTimeout))
	c.Conn.SetPongHandler(func(string) error {
		c.mu.Lock()
		c.LastPing = time.Now()
		c.mu.Unlock()
		c.Conn.SetReadDeadline(time.Now().Add(c.Hub.config.ConnectionTimeout))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			// 检查是否是正常的关闭
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				logrus.Debugf("WebSocket连接正常关闭: %s, 用户: %s", c.ID, c.UserID)
			} else {
				// 检查是否是 close 1005 (no status) - 这通常表示客户端直接关闭连接（如关闭浏览器标签页）
				// 这不应该被视为错误，而是正常的客户端行为
				errStr := err.Error()
				if strings.Contains(errStr, "close 1005") || strings.Contains(errStr, "no status") {
					logrus.Debugf("WebSocket连接被客户端关闭: %s, 用户: %s", c.ID, c.UserID)
				} else if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					// 意外的关闭错误，记录详细信息
					logrus.Errorf("WebSocket读取错误: %v, 连接ID: %s, 用户: %s", err, c.ID, c.UserID)
				} else {
					// 其他错误（如超时、网络错误等）
					logrus.Warnf("WebSocket连接错误: %v, 连接ID: %s, 用户: %s", err, c.ID, c.UserID)
				}
			}
			break
		}

		// 成功读取消息后，更新读取超时
		c.Conn.SetReadDeadline(time.Now().Add(c.Hub.config.ConnectionTimeout))

		// 处理接收到的消息
		c.handleMessage(message)
	}
}

// writePump 发送消息的协程
func (c *Connection) writePump() {
	var ticker *time.Ticker
	if !c.Hub.config.EnableGlobalPing {
		interval := c.Hub.config.HeartbeatInterval
		if interval <= 0 {
			interval = 30 * time.Second
		}
		pingEvery := time.Duration(float64(interval) * 0.9)
		ticker = time.NewTicker(pingEvery)
	}
	defer func() {
		if ticker != nil {
			ticker.Stop()
		}
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// 通道已关闭，发送关闭消息
				c.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				return
			}

			// 每条消息单独发送，避免多行 JSON 解析问题
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				// 写入错误，记录日志
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					logrus.Warnf("WebSocket写入错误: %v, 连接ID: %s, 用户: %s", err, c.ID, c.UserID)
				}
				return
			}
		case <-func() <-chan time.Time {
			if ticker != nil {
				return ticker.C
			}
			return make(chan time.Time)
		}():
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				// Ping 失败，记录日志
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					logrus.Warnf("WebSocket Ping失败: %v, 连接ID: %s, 用户: %s", err, c.ID, c.UserID)
				}
				return
			}
		}
	}
}

// handleMessage 处理接收到的消息
func (c *Connection) handleMessage(message []byte) {
	var msg Message
	if err := json.Unmarshal(message, &msg); err != nil {
		logrus.Errorf("消息解析失败: %v", err)
		return
	}

	// 设置发送者ID
	msg.From = c.UserID

	// 根据消息类型处理
	switch msg.Type {
	case "ping":
		c.handlePing()
	case "join_group":
		c.handleJoinGroup(msg)
	case "leave_group":
		c.handleLeaveGroup(msg)
	case "chat":
		c.handleChat(msg)
	case "notification":
		c.handleNotification(msg)
	case "status":
		c.handleStatus(msg)
	default:
		logrus.Warnf("未知的消息类型: %s", msg.Type)
	}
}

// handlePing 处理ping消息
func (c *Connection) handlePing() {
	c.mu.Lock()
	c.LastPing = time.Now()
	c.mu.Unlock()

	// 发送pong响应
	response := Message{
		Type:      "pong",
		Timestamp: time.Now().Unix(),
	}

	data, _ := json.Marshal(response)
	select {
	case c.Send <- data:
	default:
		logrus.Warnf("连接 %s 发送缓冲区已满", c.ID)
	}
}

// handleJoinGroup 处理加入组消息
func (c *Connection) handleJoinGroup(msg Message) {
	groupName, ok := msg.Data.(string)
	if !ok {
		logrus.Warnf("无效的组名: %v", msg.Data)
		return
	}

	c.mu.Lock()
	c.Groups[groupName] = true
	c.mu.Unlock()

	// 通知Hub更新组连接映射
	c.Hub.mu.Lock()
	if c.Hub.groupConnections[groupName] == nil {
		c.Hub.groupConnections[groupName] = make(map[string]bool)
	}
	c.Hub.groupConnections[groupName][c.ID] = true
	c.Hub.mu.Unlock()

	// 发送确认消息
	response := Message{
		Type:      "group_joined",
		Data:      groupName,
		Timestamp: time.Now().Unix(),
	}

	data, _ := json.Marshal(response)
	select {
	case c.Send <- data:
	default:
		logrus.Warnf("连接 %s 发送缓冲区已满", c.ID)
	}

	logrus.Infof("用户 %s 加入组 %s", c.UserID, groupName)
}

// handleLeaveGroup 处理离开组消息
func (c *Connection) handleLeaveGroup(msg Message) {
	groupName, ok := msg.Data.(string)
	if !ok {
		logrus.Warnf("无效的组名: %v", msg.Data)
		return
	}

	c.mu.Lock()
	delete(c.Groups, groupName)
	c.mu.Unlock()

	// 通知Hub更新组连接映射
	c.Hub.mu.Lock()
	if c.Hub.groupConnections[groupName] != nil {
		delete(c.Hub.groupConnections[groupName], c.ID)
		if len(c.Hub.groupConnections[groupName]) == 0 {
			delete(c.Hub.groupConnections, groupName)
		}
	}
	c.Hub.mu.Unlock()

	// 发送确认消息
	response := Message{
		Type:      "group_left",
		Data:      groupName,
		Timestamp: time.Now().Unix(),
	}

	data, _ := json.Marshal(response)
	select {
	case c.Send <- data:
	default:
		logrus.Warnf("连接 %s 发送缓冲区已满", c.ID)
	}

	logrus.Infof("用户 %s 离开组 %s", c.UserID, groupName)
}

// handleChat 处理聊天消息
func (c *Connection) handleChat(msg Message) {
	// 验证消息数据
	if _, ok := msg.Data.(map[string]interface{}); !ok {
		logrus.Warnf("无效的聊天数据: %v", msg.Data)
		return
	}

	// 检查是否有目标用户或组
	if msg.To == "" && msg.Group == "" {
		logrus.Warnf("聊天消息缺少目标")
		return
	}

	// 广播消息
	c.Hub.broadcast <- &msg
}

// handleNotification 处理通知消息
func (c *Connection) handleNotification(msg Message) {
	// 验证通知数据
	if _, ok := msg.Data.(map[string]interface{}); !ok {
		logrus.Warnf("无效的通知数据: %v", msg.Data)
		return
	}

	// 广播通知
	c.Hub.broadcast <- &msg
}

// handleStatus 处理状态消息
func (c *Connection) handleStatus(msg Message) {
	// 更新连接状态
	if statusData, ok := msg.Data.(map[string]interface{}); ok {
		c.mu.Lock()
		for key, value := range statusData {
			c.Metadata[key] = value
		}
		c.mu.Unlock()
	}

	// 发送状态确认
	response := Message{
		Type:      "status_updated",
		Timestamp: time.Now().Unix(),
	}

	data, _ := json.Marshal(response)
	select {
	case c.Send <- data:
	default:
		logrus.Warnf("连接 %s 发送缓冲区已满", c.ID)
	}
}

// SendMessage 发送消息给当前连接
func (c *Connection) SendMessage(message *Message) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	select {
	case c.Send <- data:
		return nil
	default:
		return fmt.Errorf("发送缓冲区已满")
	}
}

// JoinGroup 加入组
func (c *Connection) JoinGroup(groupName string) {
	c.mu.Lock()
	c.Groups[groupName] = true
	c.mu.Unlock()

	// 通知Hub更新组连接映射
	c.Hub.mu.Lock()
	if c.Hub.groupConnections[groupName] == nil {
		c.Hub.groupConnections[groupName] = make(map[string]bool)
	}
	c.Hub.groupConnections[groupName][c.ID] = true
	c.Hub.mu.Unlock()
}

// LeaveGroup 离开组
func (c *Connection) LeaveGroup(groupName string) {
	c.mu.Lock()
	delete(c.Groups, groupName)
	c.mu.Unlock()

	// 通知Hub更新组连接映射
	c.Hub.mu.Lock()
	if c.Hub.groupConnections[groupName] != nil {
		delete(c.Hub.groupConnections[groupName], c.ID)
		if len(c.Hub.groupConnections[groupName]) == 0 {
			delete(c.Hub.groupConnections, groupName)
		}
	}
	c.Hub.mu.Unlock()
}

// IsInGroup 检查是否在指定组中
func (c *Connection) IsInGroup(groupName string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Groups[groupName]
}

// GetGroups 获取连接所属的组
func (c *Connection) GetGroups() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	groups := make([]string, 0, len(c.Groups))
	for group := range c.Groups {
		groups = append(groups, group)
	}
	return groups
}
