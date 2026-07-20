// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package voiceprint

import (
	"context"
	"testing"
	"time"
)

// MockCache 模拟缓存实现
type MockCache struct {
	data map[string]interface{}
}

func NewMockCache() *MockCache {
	return &MockCache{
		data: make(map[string]interface{}),
	}
}

func (m *MockCache) Get(ctx context.Context, key string) (interface{}, bool) {
	val, ok := m.data[key]
	return val, ok
}

func (m *MockCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	m.data[key] = value
	return nil
}

func (m *MockCache) Exists(ctx context.Context, key string) bool {
	_, ok := m.data[key]
	return ok
}

func (m *MockCache) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func TestNewService(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		cache   Cache
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Enabled:       true,
				BaseURL:       "http://localhost:8005",
				APIKey:        "test-key",
				Timeout:       10 * time.Second,
				LogLevel:      "info",
				MaxCandidates: 10,
			},
			cache:   NewMockCache(),
			wantErr: false,
		},
		{
			name: "invalid config - no base url",
			config: &Config{
				Enabled:       true,
				APIKey:        "test-key",
				Timeout:       10 * time.Second,
				LogLevel:      "info",
				MaxCandidates: 10,
			},
			cache:   NewMockCache(),
			wantErr: true,
		},
		{
			name: "invalid config - no api key",
			config: &Config{
				Enabled:       true,
				BaseURL:       "http://localhost:8005",
				Timeout:       10 * time.Second,
				LogLevel:      "info",
				MaxCandidates: 10,
			},
			cache:   NewMockCache(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewService(tt.config, tt.cache)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Error("NewService() returned nil")
			}
		})
	}
}

func TestService_IsEnabled(t *testing.T) {
	config := &Config{
		Enabled:       true,
		BaseURL:       "http://localhost:8005",
		APIKey:        "test-key",
		MaxCandidates: 10,
	}
	service, _ := NewService(config, NewMockCache())

	if !service.IsEnabled() {
		t.Error("Service.IsEnabled() should return true")
	}

	config.Enabled = false
	service, _ = NewService(config, NewMockCache())
	if service.IsEnabled() {
		t.Error("Service.IsEnabled() should return false")
	}
}

func TestService_HealthCheck(t *testing.T) {
	tests := []struct {
		name      string
		enabled   bool
		wantErr   bool
		errString string
	}{
		{
			name:      "service disabled",
			enabled:   false,
			wantErr:   true,
			errString: "service is disabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Enabled:       tt.enabled,
				BaseURL:       "http://localhost:8005",
				APIKey:        "test-key",
				MaxCandidates: 10,
			}
			service, _ := NewService(config, NewMockCache())
			ctx := context.Background()

			_, err := service.HealthCheck(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("HealthCheck() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestService_GetStatistics(t *testing.T) {
	config := &Config{
		Enabled:       true,
		BaseURL:       "http://localhost:8005",
		APIKey:        "test-key",
		MaxCandidates: 10,
	}
	service, _ := NewService(config, NewMockCache())

	stats := service.GetStatistics()
	if stats == nil {
		t.Error("GetStatistics() returned nil")
	}
	if stats.TotalIdentifications != 0 {
		t.Errorf("GetStatistics() TotalIdentifications = %d, want 0", stats.TotalIdentifications)
	}
}

func TestService_Close(t *testing.T) {
	config := &Config{
		Enabled:       true,
		BaseURL:       "http://localhost:8005",
		APIKey:        "test-key",
		MaxCandidates: 10,
	}
	service, _ := NewService(config, NewMockCache())

	err := service.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestService_CacheOperations(t *testing.T) {
	config := &Config{
		Enabled:       true,
		BaseURL:       "http://localhost:8005",
		APIKey:        "test-key",
		CacheEnabled:  true,
		CacheTTL:      5 * time.Minute,
		MaxCandidates: 10,
	}
	cache := NewMockCache()
	_, _ = NewService(config, cache)
	ctx := context.Background()

	// Test cache set and get
	testKey := "test:key"
	testValue := "test:value"

	cache.Set(ctx, testKey, testValue, 5*time.Minute)
	val, found := cache.Get(ctx, testKey)

	if !found {
		t.Error("Cache.Get() should find the key")
	}
	if val != testValue {
		t.Errorf("Cache.Get() = %v, want %v", val, testValue)
	}

	// Test cache miss
	_, found = cache.Get(ctx, "nonexistent")
	if found {
		t.Error("Cache.Get() should not find nonexistent key")
	}
}

func TestService_Concurrent(t *testing.T) {
	config := &Config{
		Enabled:       true,
		BaseURL:       "http://localhost:8005",
		APIKey:        "test-key",
		MaxCandidates: 10,
	}
	service, _ := NewService(config, NewMockCache())

	// Test concurrent access to stats
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			stats := service.GetStatistics()
			if stats == nil {
				t.Error("GetStatistics() returned nil")
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestService_StatisticsUpdate(t *testing.T) {
	config := &Config{
		Enabled:       true,
		BaseURL:       "http://localhost:8005",
		APIKey:        "test-key",
		MaxCandidates: 10,
	}
	service, _ := NewService(config, NewMockCache())

	initialStats := service.GetStatistics()
	if initialStats.LastActivity.IsZero() {
		t.Error("Statistics.LastActivity should not be zero")
	}
}
