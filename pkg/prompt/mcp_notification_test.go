package prompt

import (
	"context"
	"reflect"
	"testing"
)

// Mock sender 实现 notificationSender 接口
type mockSender struct {
	logMessages   []string
	progressCalls []float64
	methodsCalled []string
}

func (m *mockSender) SendLogMessage(level string, message string) error {
	m.logMessages = append(m.logMessages, level+": "+message)
	return nil
}

func (m *mockSender) SendProgress(progress float64, message string) error {
	m.progressCalls = append(m.progressCalls, progress)
	return nil
}

func (m *mockSender) SendCustomNotification(method string, params map[string]interface{}) error {
	m.methodsCalled = append(m.methodsCalled, method)
	return nil
}

func (m *mockSender) SendNotification(notification *Notification) error {
	m.methodsCalled = append(m.methodsCalled, notification.Method)
	return nil
}

// 测试 withNotificationSender + GetNotificationSender
func TestNotificationContextInjection(t *testing.T) {
	sender := &mockSender{}
	ctx := context.Background()
	ctx = withNotificationSender(ctx, sender)

	retrieved, ok := GetNotificationSender(ctx)
	if !ok {
		t.Fatal("expected to retrieve sender from context")
	}
	if retrieved != sender {
		t.Errorf("expected sender to match, got %+v", retrieved)
	}
}

// 测试 GetNotificationSender - 失败情况
func TestNotificationContext_NoSender(t *testing.T) {
	ctx := context.Background()
	sender, ok := GetNotificationSender(ctx)
	if ok {
		t.Error("expected no sender in context, but got one")
	}
	if sender != nil {
		t.Errorf("expected nil sender, got %+v", sender)
	}
}

// 测试 NewNotification 构建逻辑（含 _meta）
func TestNewNotification_WithMeta(t *testing.T) {
	params := map[string]interface{}{
		"_meta": map[string]interface{}{
			"traceId": "123",
		},
		"message": "hello",
		"code":    200,
	}

	notification := NewNotification(NotificationMethodMessage, params)

	if notification.Method != NotificationMethodMessage {
		t.Errorf("method mismatch: got %s", notification.Method)
	}

	expectedMeta := map[string]interface{}{"traceId": "123"}
	if !reflect.DeepEqual(notification.Params.Meta, expectedMeta) {
		t.Errorf("meta mismatch: %+v", notification.Params.Meta)
	}

	if notification.Params.AdditionalFields["message"] != "hello" {
		t.Error("expected message field")
	}
	if notification.Params.AdditionalFields["code"] != 200 {
		t.Error("expected code field")
	}
	if _, ok := notification.Params.AdditionalFields["_meta"]; ok {
		t.Error("_meta should not exist in additional fields")
	}
}

// 测试 NewNotification 没有 _meta 情况
func TestNewNotification_NoMeta(t *testing.T) {
	params := map[string]interface{}{
		"text": "hi",
	}
	notification := NewNotification("custom/method", params)

	if notification.Method != "custom/method" {
		t.Errorf("unexpected method: %s", notification.Method)
	}
	if len(notification.Params.Meta) != 0 {
		t.Errorf("expected empty meta, got %+v", notification.Params.Meta)
	}
	if notification.Params.AdditionalFields["text"] != "hi" {
		t.Errorf("expected field 'text', got %+v", notification.Params.AdditionalFields)
	}
}
