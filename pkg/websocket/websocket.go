package websocket

import (
	"context"
	"encoding/json"
	"hash/fnv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// Message 定义WebSocket消息结构
type Message struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
	From      string      `json:"from,omitempty"`
	To        string      `json:"to,omitempty"`
	Group     string      `json:"group,omitempty"`
}

// Connection 表示一个WebSocket连接
type Connection struct {
	ID       string
	UserID   string
	Conn     *websocket.Conn
	Send     chan []byte
	Hub      *Hub
	LastPing time.Time
	IsAlive  bool
	mu       sync.RWMutex
	Groups   map[string]bool
	Metadata map[string]interface{}
}

// Hub 管理所有WebSocket连接
type Hub struct {
	// 注册的连接
	connections map[string]*Connection
	// 用户ID到连接ID的映射
	userConnections map[string]map[string]bool
	// 组到连接ID的映射
	groupConnections map[string]map[string]bool
	// 广播消息通道
	broadcast chan *Message
	// 注册连接通道
	register chan *Connection
	// 注销连接通道
	unregister chan *Connection
	// 连接计数
	connectionCount int64
	// 配置
	config *Config
	// 互斥锁
	mu sync.RWMutex
	// 上下文
	ctx    context.Context
	cancel context.CancelFunc

	// shards and locks to reduce contention when fanout
	shardCount int
	shardConns []map[string]*Connection
	shardLocks []sync.RWMutex

	// broadcast worker pool
	broadcastJobs chan broadcastJob

	// global ping
	pingJobs chan int
}

const (
	_broadcastAll = iota
)

type broadcastJob struct {
	kind  int
	shard int
	data  []byte
}

// Config WebSocket配置
type Config struct {
	// 最大连接数
	MaxConnections int64
	// 心跳间隔
	HeartbeatInterval time.Duration
	// 连接超时时间
	ConnectionTimeout time.Duration
	// 消息缓冲区大小
	MessageBufferSize int
	// 读缓冲区大小
	ReadBufferSize int
	// 写缓冲区大小
	WriteBufferSize int
	// 最大消息大小
	MaxMessageSize int
	// 是否启用压缩
	EnableCompression bool
	// 是否启用消息队列
	EnableMessageQueue bool
	// 消息队列大小
	MessageQueueSize int
	// 是否启用集群模式
	EnableCluster bool
	// 集群节点ID
	ClusterNodeID string
	// 分片数量
	ShardCount int
	// 广播worker数量
	BroadcastWorkerCount int
	// 发送缓冲区满时是否丢弃
	DropOnFull bool
	// 压缩等级（-2..9）
	CompressionLevel int
	// 慢消费者策略：背压触发时直接断开
	CloseOnBackpressure bool
	// 发送阻塞超时（用于非 DropOnFull 模式）
	SendTimeout time.Duration
	// 启用全局心跳
	EnableGlobalPing bool
	// 全局心跳workers
	PingWorkerCount int
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		MaxConnections:       100000, // 10万连接
		HeartbeatInterval:    30 * time.Second,
		ConnectionTimeout:    60 * time.Second,
		MessageBufferSize:    256,
		ReadBufferSize:       1024,
		WriteBufferSize:      1024,
		MaxMessageSize:       512,
		EnableCompression:    true,
		EnableMessageQueue:   true,
		MessageQueueSize:     1000,
		EnableCluster:        false,
		ClusterNodeID:        "",
		ShardCount:           16,
		BroadcastWorkerCount: 32,
		DropOnFull:           true,
		CompressionLevel:     -2,
		CloseOnBackpressure:  false,
		SendTimeout:          50 * time.Millisecond,
		EnableGlobalPing:     false,
		PingWorkerCount:      8,
	}
}

// NewHub 创建新的Hub实例
func NewHub(config *Config) *Hub {
	if config == nil {
		config = DefaultConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	hub := &Hub{
		connections:      make(map[string]*Connection),
		userConnections:  make(map[string]map[string]bool),
		groupConnections: make(map[string]map[string]bool),
		broadcast:        make(chan *Message, config.MessageQueueSize),
		register:         make(chan *Connection, 1000),
		unregister:       make(chan *Connection, 1000),
		config:           config,
		ctx:              ctx,
		cancel:           cancel,
	}

	// init shards
	if hub.config.ShardCount <= 0 {
		hub.config.ShardCount = 1
	}
	hub.shardCount = hub.config.ShardCount
	hub.shardConns = make([]map[string]*Connection, hub.shardCount)
	hub.shardLocks = make([]sync.RWMutex, hub.shardCount)
	for i := 0; i < hub.shardCount; i++ {
		hub.shardConns[i] = make(map[string]*Connection)
	}

	// init broadcast workers
	if hub.config.BroadcastWorkerCount <= 0 {
		hub.config.BroadcastWorkerCount = 1
	}
	hub.broadcastJobs = make(chan broadcastJob, hub.config.MessageQueueSize)
	for i := 0; i < hub.config.BroadcastWorkerCount; i++ {
		go hub.broadcastWorker()
	}

	// init global ping workers
	if hub.config.EnableGlobalPing {
		if hub.config.PingWorkerCount <= 0 {
			hub.config.PingWorkerCount = 1
		}
		hub.pingJobs = make(chan int, hub.shardCount)
		for i := 0; i < hub.config.PingWorkerCount; i++ {
			go hub.pingWorker()
		}
	}

	go hub.run()
	return hub
}

// run Hub主循环
func (h *Hub) run() {
	ticker := time.NewTicker(h.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case conn := <-h.register:
			h.registerConnection(conn)
		case conn := <-h.unregister:
			h.unregisterConnection(conn)
		case message := <-h.broadcast:
			// 单次序列化减少重复开销
			if message.Timestamp == 0 {
				message.Timestamp = time.Now().Unix()
			}
			data, err := json.Marshal(message)
			if err != nil {
				logrus.Errorf("消息序列化失败: %v", err)
				continue
			}
			switch {
			case message.To != "":
				h.sendToUser(message.To, data)
			case message.Group != "":
				h.sendToGroup(message.Group, data)
			default:
				h.enqueueBroadcastAll(data)
			}
		case <-ticker.C:
			if h.config.EnableGlobalPing {
				// 使用分片维度触发 ping
				for i := 0; i < h.shardCount; i++ {
					select {
					case h.pingJobs <- i:
					default:
					}
				}
			}
			h.checkHeartbeats()
		}
	}
}

// pingWorker 全局心跳worker
func (h *Hub) pingWorker() {
	for shard := range h.pingJobs {
		h.shardLocks[shard].RLock()
		for _, conn := range h.shardConns[shard] {
			if conn.IsAlive {
				_ = conn.Conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second))
			}
		}
		h.shardLocks[shard].RUnlock()
	}
}

// registerConnection 注册连接
func (h *Hub) registerConnection(conn *Connection) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 检查最大连接数
	if atomic.LoadInt64(&h.connectionCount) >= h.config.MaxConnections {
		conn.Conn.Close()
		logrus.Warnf("达到最大连接数限制: %d", h.config.MaxConnections)
		return
	}

	h.connections[conn.ID] = conn
	atomic.AddInt64(&h.connectionCount, 1)

	// 放入分片
	sh := h.shardIndex(conn.ID)
	h.shardLocks[sh].Lock()
	h.shardConns[sh][conn.ID] = conn
	h.shardLocks[sh].Unlock()

	// 添加到用户连接映射
	if conn.UserID != "" {
		if h.userConnections[conn.UserID] == nil {
			h.userConnections[conn.UserID] = make(map[string]bool)
		}
		h.userConnections[conn.UserID][conn.ID] = true
	}

	// 添加到组连接映射
	for group := range conn.Groups {
		if h.groupConnections[group] == nil {
			h.groupConnections[group] = make(map[string]bool)
		}
		h.groupConnections[group][conn.ID] = true
	}

	logrus.Infof("WebSocket连接已注册: %s, 用户: %s, 当前连接数: %d",
		conn.ID, conn.UserID, atomic.LoadInt64(&h.connectionCount))
}

// unregisterConnection 注销连接
func (h *Hub) unregisterConnection(conn *Connection) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.connections[conn.ID]; exists {
		delete(h.connections, conn.ID)
		atomic.AddInt64(&h.connectionCount, -1)

		// 从分片移除
		sh := h.shardIndex(conn.ID)
		h.shardLocks[sh].Lock()
		delete(h.shardConns[sh], conn.ID)
		h.shardLocks[sh].Unlock()

		// 从用户连接映射中移除
		if conn.UserID != "" && h.userConnections[conn.UserID] != nil {
			delete(h.userConnections[conn.UserID], conn.ID)
			if len(h.userConnections[conn.UserID]) == 0 {
				delete(h.userConnections, conn.UserID)
			}
		}

		// 从组连接映射中移除
		for group := range conn.Groups {
			if h.groupConnections[group] != nil {
				delete(h.groupConnections[group], conn.ID)
				if len(h.groupConnections[group]) == 0 {
					delete(h.groupConnections, group)
				}
			}
		}

		close(conn.Send)
		logrus.Infof("WebSocket连接已注销: %s, 当前连接数: %d",
			conn.ID, atomic.LoadInt64(&h.connectionCount))
	}
}

// broadcastMessage 广播消息
func (h *Hub) broadcastMessage(message *Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// 设置时间戳
	if message.Timestamp == 0 {
		message.Timestamp = time.Now().Unix()
	}

	// 序列化消息
	data, err := json.Marshal(message)
	if err != nil {
		logrus.Errorf("消息序列化失败: %v", err)
		return
	}

	// 根据消息类型决定发送策略
	switch {
	case message.To != "":
		// 发送给特定用户
		h.sendToUser(message.To, data)
	case message.Group != "":
		// 发送给特定组
		h.sendToGroup(message.Group, data)
	default:
		// 广播给所有连接
		h.sendToAll(data)
	}
}

// sendToUser 发送消息给特定用户
func (h *Hub) sendToUser(userID string, data []byte) {
	if connections, exists := h.userConnections[userID]; exists {
		for connID := range connections {
			if conn, ok := h.connections[connID]; ok && conn.IsAlive {
				h.trySend(conn, data, func() { logrus.Warnf("用户 %s 的连接 %s 发送缓冲区已满", userID, connID) })
			}
		}
	}
}

// sendToGroup 发送消息给特定组
func (h *Hub) sendToGroup(group string, data []byte) {
	if connections, exists := h.groupConnections[group]; exists {
		for connID := range connections {
			if conn, ok := h.connections[connID]; ok && conn.IsAlive {
				h.trySend(conn, data, func() { logrus.Warnf("组 %s 的连接 %s 发送缓冲区已满", group, connID) })
			}
		}
	}
}

// sendToAll 发送消息给所有连接
func (h *Hub) sendToAll(data []byte) {
	for i := 0; i < h.shardCount; i++ {
		select {
		case h.broadcastJobs <- broadcastJob{kind: _broadcastAll, shard: i, data: data}:
		default:
			logrus.Warnf("广播作业队列已满，消息被丢弃")
		}
	}
}

// checkHeartbeats 检查心跳
func (h *Hub) checkHeartbeats() {
	h.mu.RLock()
	defer h.mu.RUnlock()

	now := time.Now()
	for _, conn := range h.connections {
		if now.Sub(conn.LastPing) > h.config.ConnectionTimeout {
			logrus.Warnf("连接 %s 心跳超时，准备关闭", conn.ID)
			conn.IsAlive = false
			conn.Conn.Close()
		}
	}
}

// GetConnectionCount 获取当前连接数
func (h *Hub) GetConnectionCount() int64 {
	return atomic.LoadInt64(&h.connectionCount)
}

// GetUserConnections 获取用户的连接数
func (h *Hub) GetUserConnections(userID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if connections, exists := h.userConnections[userID]; exists {
		return len(connections)
	}
	return 0
}

// GetGroupConnections 获取组的连接数
func (h *Hub) GetGroupConnections(group string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if connections, exists := h.groupConnections[group]; exists {
		return len(connections)
	}
	return 0
}

// GetBroadcastChannel 获取广播通道（用于外部发送消息）
func (h *Hub) GetBroadcastChannel() chan<- *Message {
	return h.broadcast
}

// Close 关闭Hub
func (h *Hub) Close() {
	h.cancel()

	// 关闭所有连接
	h.mu.Lock()
	for _, conn := range h.connections {
		conn.Conn.Close()
	}
	h.mu.Unlock()

	logrus.Info("WebSocket Hub已关闭")
}

// shardIndex 计算分片索引
func (h *Hub) shardIndex(id string) int {
	if h.shardCount <= 1 {
		return 0
	}
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(id))
	return int(hasher.Sum32() % uint32(h.shardCount))
}

// enqueueBroadcastAll 将广播任务按分片入队
func (h *Hub) enqueueBroadcastAll(data []byte) {
	for i := 0; i < h.shardCount; i++ {
		select {
		case h.broadcastJobs <- broadcastJob{kind: _broadcastAll, shard: i, data: data}:
		default:
			logrus.Warnf("广播作业队列已满，消息被丢弃")
		}
	}
}

// broadcastWorker 广播worker
func (h *Hub) broadcastWorker() {
	for job := range h.broadcastJobs {
		switch job.kind {
		case _broadcastAll:
			h.shardLocks[job.shard].RLock()
			for _, conn := range h.shardConns[job.shard] {
				if conn.IsAlive {
					h.trySend(conn, job.data, func() { logrus.Debugf("连接 %s 发送缓冲区满，已按策略处理", conn.ID) })
				}
			}
			h.shardLocks[job.shard].RUnlock()
		}
	}
}

// trySend 背压策略
func (h *Hub) trySend(conn *Connection, data []byte, onDrop func()) {
	if h.config.DropOnFull {
		select {
		case conn.Send <- data:
		default:
			onDrop()
			if h.config.CloseOnBackpressure {
				conn.Conn.Close()
			}
		}
		return
	}
	// 非丢弃模式：限定等待时长
	timeout := h.config.SendTimeout
	if timeout <= 0 {
		timeout = 50 * time.Millisecond
	}
	select {
	case conn.Send <- data:
	case <-time.After(timeout):
		onDrop()
		if h.config.CloseOnBackpressure {
			conn.Conn.Close()
		}
	}
}
