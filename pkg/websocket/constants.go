package websocket

// WebSocket消息类型常量
const (
	// 系统消息类型
	MessageTypePing          = "ping"
	MessageTypePong          = "pong"
	MessageTypeJoinGroup     = "join_group"
	MessageTypeLeaveGroup    = "leave_group"
	MessageTypeGroupJoined   = "group_joined"
	MessageTypeGroupLeft     = "group_left"
	MessageTypeStatus        = "status"
	MessageTypeStatusUpdated = "status_updated"

	// 业务消息类型
	MessageTypeChat         = "chat"
	MessageTypeNotification = "notification"
	MessageTypeSystem       = "system"
	MessageTypeError        = "error"
	MessageTypeSuccess      = "success"

	// 连接状态
	ConnectionStatusConnected    = "connected"
	ConnectionStatusDisconnected = "disconnected"
	ConnectionStatusReconnecting = "reconnecting"
	ConnectionStatusError        = "error"

	// 默认配置值
	DefaultMaxConnections    = 100000
	DefaultHeartbeatInterval = 30
	DefaultConnectionTimeout = 60
	DefaultMessageBufferSize = 256
	DefaultMessageQueueSize  = 1000
	DefaultReadBufferSize    = 1024
	DefaultWriteBufferSize   = 1024
	DefaultMaxMessageSize    = 512

	// 环境变量配置键
	EnvWebSocketMaxConnections      = "WEBSOCKET_MAX_CONNECTIONS"
	EnvWebSocketHeartbeatInterval   = "WEBSOCKET_HEARTBEAT_INTERVAL"
	EnvWebSocketConnectionTimeout   = "WEBSOCKET_CONNECTION_TIMEOUT"
	EnvWebSocketMessageBufferSize   = "WEBSOCKET_MESSAGE_BUFFER_SIZE"
	EnvWebSocketMessageQueueSize    = "WEBSOCKET_MESSAGE_QUEUE_SIZE"
	EnvWebSocketEnableCompression   = "WEBSOCKET_ENABLE_COMPRESSION"
	EnvWebSocketEnableMessageQueue  = "WEBSOCKET_ENABLE_MESSAGE_QUEUE"
	EnvWebSocketEnableCluster       = "WEBSOCKET_ENABLE_CLUSTER"
	EnvWebSocketClusterNodeID       = "WEBSOCKET_CLUSTER_NODE_ID"
	EnvWebSocketShardCount          = "WEBSOCKET_SHARD_COUNT"
	EnvWebSocketBroadcastWorkers    = "WEBSOCKET_BROADCAST_WORKERS"
	EnvWebSocketDropOnFull          = "WEBSOCKET_DROP_ON_FULL"
	EnvWebSocketCompressionLevel    = "WEBSOCKET_COMPRESSION_LEVEL"
	EnvWebSocketReadBufferSize      = "WEBSOCKET_READ_BUFFER_SIZE"
	EnvWebSocketWriteBufferSize     = "WEBSOCKET_WRITE_BUFFER_SIZE"
	EnvWebSocketMaxMessageSize      = "WEBSOCKET_MAX_MESSAGE_SIZE"
	EnvWebSocketCloseOnBackpressure = "WEBSOCKET_CLOSE_ON_BACKPRESSURE"
	EnvWebSocketSendTimeoutMs       = "WEBSOCKET_SEND_TIMEOUT_MS"
	EnvWebSocketEnableGlobalPing    = "WEBSOCKET_ENABLE_GLOBAL_PING"
	EnvWebSocketPingWorkers         = "WEBSOCKET_PING_WORKERS"

	// 错误消息
	ErrConnectionLimitExceeded = "连接数已达到上限"
	ErrInvalidMessageType      = "无效的消息类型"
	ErrInvalidMessageData      = "无效的消息数据"
	ErrUserNotFound            = "用户不存在"
	ErrGroupNotFound           = "组不存在"
	ErrConnectionClosed        = "连接已关闭"
	ErrSendBufferFull          = "发送缓冲区已满"
	ErrReadTimeout             = "读取超时"
	ErrWriteTimeout            = "写入超时"

	// 成功消息
	MsgConnectionEstablished = "连接已建立"
	MsgMessageSent           = "消息已发送"
	MsgGroupJoined           = "已加入组"
	MsgGroupLeft             = "已离开组"
	MsgStatusUpdated         = "状态已更新"

	// 路由路径
	RouteWebSocket          = "/ws"
	RouteWebSocketStats     = "/ws/stats"
	RouteWebSocketHealth    = "/ws/health"
	RouteWebSocketMessage   = "/ws/message"
	RouteWebSocketBroadcast = "/ws/broadcast"
	RouteWebSocketUser      = "/ws/user/:user_id"
	RouteWebSocketGroup     = "/ws/group/:group"
)
