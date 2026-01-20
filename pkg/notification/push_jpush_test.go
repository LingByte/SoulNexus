package notification

import (
	"context"
	"errors"
	"testing"
)

// mockJPushClient is a mock implementation of JPushClient
type mockJPushClient struct {
	pushFunc func(ctx context.Context, title, content string, audience map[string]interface{}, extras map[string]interface{}) error
}

func (m *mockJPushClient) Push(ctx context.Context, title, content string, audience map[string]interface{}, extras map[string]interface{}) error {
	if m.pushFunc != nil {
		return m.pushFunc(ctx, title, content, audience, extras)
	}
	return nil
}

func TestNewJPush(t *testing.T) {
	cfg := JPushConfig{
		AppKey:       "test-app-key",
		MasterSecret: "test-master-secret",
	}

	mockClient := &mockJPushClient{}
	jpush := NewJPush(cfg, mockClient)

	if jpush == nil {
		t.Fatal("NewJPush returned nil")
	}
	if jpush.cfg.AppKey != cfg.AppKey {
		t.Errorf("Expected AppKey %s, got %s", cfg.AppKey, jpush.cfg.AppKey)
	}
	if jpush.cli != mockClient {
		t.Error("Client not set correctly")
	}
}

func TestJPush_PushToAlias_Success(t *testing.T) {
	cfg := JPushConfig{
		AppKey:       "test-app-key",
		MasterSecret: "test-master-secret",
	}

	mockClient := &mockJPushClient{
		pushFunc: func(ctx context.Context, title, content string, audience map[string]interface{}, extras map[string]interface{}) error {
			if title != "Test Title" {
				t.Errorf("Expected title 'Test Title', got %s", title)
			}
			if content != "Test Content" {
				t.Errorf("Expected content 'Test Content', got %s", content)
			}
			alias, ok := audience["alias"].([]string)
			if !ok {
				t.Error("Expected audience alias to be []string")
			} else if len(alias) != 2 || alias[0] != "user1" || alias[1] != "user2" {
				t.Errorf("Expected alias [user1, user2], got %v", alias)
			}
			return nil
		},
	}

	jpush := NewJPush(cfg, mockClient)
	err := jpush.PushToAlias(context.Background(), []string{"user1", "user2"}, "Test Title", "Test Content", nil)
	if err != nil {
		t.Errorf("PushToAlias failed: %v", err)
	}
}

func TestJPush_PushToAlias_WithExtras(t *testing.T) {
	cfg := JPushConfig{
		AppKey:       "test-app-key",
		MasterSecret: "test-master-secret",
	}

	extras := map[string]interface{}{
		"key": "value",
	}

	mockClient := &mockJPushClient{
		pushFunc: func(ctx context.Context, title, content string, audience map[string]interface{}, extras map[string]interface{}) error {
			if extras["key"] != "value" {
				t.Errorf("Expected extras key 'value', got %v", extras["key"])
			}
			return nil
		},
	}

	jpush := NewJPush(cfg, mockClient)
	err := jpush.PushToAlias(context.Background(), []string{"user1"}, "Title", "Content", extras)
	if err != nil {
		t.Errorf("PushToAlias failed: %v", err)
	}
}

func TestJPush_PushToAlias_NoClient(t *testing.T) {
	cfg := JPushConfig{
		AppKey:       "test-app-key",
		MasterSecret: "test-master-secret",
	}

	jpush := NewJPush(cfg, nil)
	err := jpush.PushToAlias(context.Background(), []string{"user1"}, "Title", "Content", nil)
	if err == nil {
		t.Error("Expected error when client is nil, got nil")
	}
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestJPush_PushToAlias_ClientError(t *testing.T) {
	cfg := JPushConfig{
		AppKey:       "test-app-key",
		MasterSecret: "test-master-secret",
	}

	mockClient := &mockJPushClient{
		pushFunc: func(ctx context.Context, title, content string, audience map[string]interface{}, extras map[string]interface{}) error {
			return errors.New("push failed")
		},
	}

	jpush := NewJPush(cfg, mockClient)
	err := jpush.PushToAlias(context.Background(), []string{"user1"}, "Title", "Content", nil)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestJPush_PushToAll_Success(t *testing.T) {
	cfg := JPushConfig{
		AppKey:       "test-app-key",
		MasterSecret: "test-master-secret",
	}

	mockClient := &mockJPushClient{
		pushFunc: func(ctx context.Context, title, content string, audience map[string]interface{}, extras map[string]interface{}) error {
			if title != "Broadcast Title" {
				t.Errorf("Expected title 'Broadcast Title', got %s", title)
			}
			all, ok := audience["all"].(bool)
			if !ok || !all {
				t.Error("Expected audience all to be true")
			}
			return nil
		},
	}

	jpush := NewJPush(cfg, mockClient)
	err := jpush.PushToAll(context.Background(), "Broadcast Title", "Broadcast Content", nil)
	if err != nil {
		t.Errorf("PushToAll failed: %v", err)
	}
}

func TestJPush_PushToAll_NoClient(t *testing.T) {
	cfg := JPushConfig{
		AppKey:       "test-app-key",
		MasterSecret: "test-master-secret",
	}

	jpush := NewJPush(cfg, nil)
	err := jpush.PushToAll(context.Background(), "Title", "Content", nil)
	if err == nil {
		t.Error("Expected error when client is nil, got nil")
	}
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestJPush_PushToAll_ClientError(t *testing.T) {
	cfg := JPushConfig{
		AppKey:       "test-app-key",
		MasterSecret: "test-master-secret",
	}

	mockClient := &mockJPushClient{
		pushFunc: func(ctx context.Context, title, content string, audience map[string]interface{}, extras map[string]interface{}) error {
			return errors.New("push failed")
		},
	}

	jpush := NewJPush(cfg, mockClient)
	err := jpush.PushToAll(context.Background(), "Title", "Content", nil)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestJPushConfig_Structure(t *testing.T) {
	cfg := JPushConfig{
		AppKey:       "app-key",
		MasterSecret: "master-secret",
	}

	if cfg.AppKey == "" {
		t.Error("AppKey should not be empty")
	}
	if cfg.MasterSecret == "" {
		t.Error("MasterSecret should not be empty")
	}
}
