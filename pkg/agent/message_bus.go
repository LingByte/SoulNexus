package agent

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// MessageBus Agent间消息总线
type MessageBus struct {
	subscribers map[string][]MessageHandler
	mu          sync.RWMutex
	logger      *zap.Logger
}

// MessageHandler 消息处理器
type MessageHandler func(ctx context.Context, message *Message) error

// NewMessageBus 创建新的消息总线
func NewMessageBus(logger *zap.Logger) *MessageBus {
	return &MessageBus{
		subscribers: make(map[string][]MessageHandler),
		logger:      logger,
	}
}

// Subscribe 订阅消息
func (mb *MessageBus) Subscribe(topic string, handler MessageHandler) {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	if mb.subscribers[topic] == nil {
		mb.subscribers[topic] = make([]MessageHandler, 0)
	}

	mb.subscribers[topic] = append(mb.subscribers[topic], handler)
	mb.logger.Info("Subscribed to topic", zap.String("topic", topic))
}

// Publish 发布消息
func (mb *MessageBus) Publish(ctx context.Context, topic string, message *Message) error {
	mb.mu.RLock()
	handlers := mb.subscribers[topic]
	mb.mu.RUnlock()

	if len(handlers) == 0 {
		mb.logger.Debug("No subscribers for topic", zap.String("topic", topic))
		return nil
	}

	// 设置消息ID和时间戳（如果没有）
	if message.ID == "" {
		message.ID = generateMessageID()
	}
	if message.CreatedAt.IsZero() {
		message.CreatedAt = time.Now()
	}

	// 异步调用所有处理器
	var wg sync.WaitGroup
	for _, handler := range handlers {
		wg.Add(1)
		go func(h MessageHandler) {
			defer wg.Done()
			if err := h(ctx, message); err != nil {
				mb.logger.Error("Message handler failed",
					zap.String("topic", topic),
					zap.String("messageID", message.ID),
					zap.Error(err),
				)
			}
		}(handler)
	}

	wg.Wait()
	mb.logger.Debug("Message published",
		zap.String("topic", topic),
		zap.String("messageID", message.ID),
		zap.Int("subscribers", len(handlers)),
	)

	return nil
}

// Unsubscribe 取消订阅（简化实现，实际可能需要更复杂的逻辑）
func (mb *MessageBus) Unsubscribe(topic string) {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	delete(mb.subscribers, topic)
	mb.logger.Info("Unsubscribed from topic", zap.String("topic", topic))
}

// generateMessageID 生成消息ID
func generateMessageID() string {
	return "msg_" + time.Now().Format("20060102150405") + "_" + time.Now().Format("000000000")
}

// 消息主题常量
const (
	TopicTaskCreated   = "task.created"
	TopicTaskCompleted = "task.completed"
	TopicTaskFailed    = "task.failed"
	TopicAgentStatus   = "agent.status"
	TopicWorkflowStart = "workflow.start"
	TopicWorkflowEnd   = "workflow.end"
)
