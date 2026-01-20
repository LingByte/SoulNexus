package prompt

import "context"

// Notification 标识通知类型的常量
const (
	// NotificationMethodMessage 表示日志消息类通知的方法名，通常用于发送文本信息给客户端
	NotificationMethodMessage = "notifications/message"

	// NotificationMethodProgress 表示进度更新类通知的方法名，用于通知客户端当前操作的进度变化
	NotificationMethodProgress = "notifications/progress"
)

// 用作上下文中的键类型。通过将其定义为新的类型而不是直接使用 string，可以防止与其他包或标准库中的上下文键发生命名冲突。
type contextKey string

// Notification 用于在 context.Context 中存储和检索 notificationSender 实例的键常量
const notificationSenderKey contextKey = "notificationSender"

// notificationSender defines the notification sender interface
type notificationSender interface {
	// SendLogMessage 发送日志消息通知
	SendLogMessage(level string, message string) error

	// SendProgress 发送进度更新通知
	SendProgress(progress float64, message string) error

	// SendCustomNotification 发送自定义方法的通知
	SendCustomNotification(method string, params map[string]interface{}) error

	// SendNotification 发送通用通知对象
	SendNotification(notification *Notification) error
}

// withNotificationSender adds a notification sender to the context
func withNotificationSender(ctx context.Context, sender notificationSender) context.Context {
	return context.WithValue(ctx, notificationSenderKey, sender)
}

// GetNotificationSender retrieves the notification sender from the context
func GetNotificationSender(ctx context.Context) (notificationSender, bool) {
	sender, ok := ctx.Value(notificationSenderKey).(notificationSender)
	return sender, ok
}

// NewNotification creates a new notification with the given method and parameters
func NewNotification(method string, params map[string]interface{}) *Notification {
	notificationParams := NotificationParams{
		AdditionalFields: make(map[string]interface{}),
	}

	// Extract meta-field if present
	if meta, ok := params["_meta"]; ok {
		if metaMap, ok := meta.(map[string]interface{}); ok {
			notificationParams.Meta = metaMap
		}
		delete(params, "_meta")
	}

	// Add remaining fields to AdditionalFields
	for k, v := range params {
		notificationParams.AdditionalFields[k] = v
	}

	return &Notification{
		Method: method,
		Params: notificationParams,
	}
}
