package voiceprint

import (
	"context"
	"testing"
	"time"
)

// MockCache 模拟缓存实现
type MockCache struct{}

func (m *MockCache) Get(ctx context.Context, key string) (interface{}, bool) {
	return nil, false
}

func (m *MockCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return nil
}

func (m *MockCache) Delete(ctx context.Context, key string) error {
	return nil
}

func (m *MockCache) Exists(ctx context.Context, key string) bool {
	return false
}

func (m *MockCache) Clear(ctx context.Context) error {
	return nil
}

func (m *MockCache) GetMulti(ctx context.Context, keys ...string) map[string]interface{} {
	return make(map[string]interface{})
}

func (m *MockCache) SetMulti(ctx context.Context, data map[string]interface{}, expiration time.Duration) error {
	return nil
}

func (m *MockCache) DeleteMulti(ctx context.Context, keys ...string) error {
	return nil
}

func (m *MockCache) Increment(ctx context.Context, key string, value int64) (int64, error) {
	return 0, nil
}

func (m *MockCache) Decrement(ctx context.Context, key string, value int64) (int64, error) {
	return 0, nil
}

func (m *MockCache) GetWithTTL(ctx context.Context, key string) (interface{}, time.Duration, bool) {
	return nil, 0, false
}

func (m *MockCache) Close() error {
	return nil
}

func TestServiceCreation(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = false // 禁用服务以避免实际网络调用

	mockCache := &MockCache{}

	service, err := NewService(config, mockCache)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	if service == nil {
		t.Fatal("Service should not be nil")
	}

	if service.IsEnabled() {
		t.Error("Service should be disabled")
	}

	// 测试关闭
	err = service.Close()
	if err != nil {
		t.Errorf("Failed to close service: %v", err)
	}
}

func TestServiceDisabled(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = false

	mockCache := &MockCache{}
	service, err := NewService(config, mockCache)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	ctx := context.Background()

	// 测试禁用服务的行为
	_, err = service.HealthCheck(ctx)
	if err != ErrServiceDisabled {
		t.Errorf("Expected ErrServiceDisabled, got %v", err)
	}

	_, err = service.RegisterVoiceprint(ctx, &RegisterRequest{})
	if err != ErrServiceDisabled {
		t.Errorf("Expected ErrServiceDisabled, got %v", err)
	}

	_, err = service.IdentifyVoiceprint(ctx, &IdentifyRequest{})
	if err != ErrServiceDisabled {
		t.Errorf("Expected ErrServiceDisabled, got %v", err)
	}

	_, err = service.DeleteVoiceprint(ctx, "test")
	if err != ErrServiceDisabled {
		t.Errorf("Expected ErrServiceDisabled, got %v", err)
	}
}

func TestStatistics(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = false

	mockCache := &MockCache{}
	service, err := NewService(config, mockCache)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	stats := service.GetStatistics()
	if stats == nil {
		t.Fatal("Statistics should not be nil")
	}

	if stats.TotalIdentifications != 0 {
		t.Errorf("Expected 0 total identifications, got %d", stats.TotalIdentifications)
	}
}
