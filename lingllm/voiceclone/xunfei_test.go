package voiceclone

import (
	"testing"
	"time"
)

func TestNewXunfeiCloneService(t *testing.T) {
	config := XunfeiCloneConfig{
		AppID:  "test_app",
		APIKey: "test_key",
	}

	service := NewXunfeiCloneService(config)

	if service == nil {
		t.Error("NewXunfeiCloneService should return a non-nil service")
	}

	if service.config.AppID != "test_app" {
		t.Errorf("AppID = %s, want test_app", service.config.AppID)
	}

	if service.config.APIKey != "test_key" {
		t.Errorf("APIKey = %s, want test_key", service.config.APIKey)
	}
}

func TestNewXunfeiCloneServiceDefaults(t *testing.T) {
	config := XunfeiCloneConfig{
		AppID:  "test_app",
		APIKey: "test_key",
	}

	service := NewXunfeiCloneService(config)

	if service.config.BaseURL != "http://opentrain.xfyousheng.com" {
		t.Errorf("BaseURL = %s, want http://opentrain.xfyousheng.com", service.config.BaseURL)
	}

	if service.config.Timeout != 30 {
		t.Errorf("Timeout = %d, want 30", service.config.Timeout)
	}

	if service.config.EngineVersion != "omni_v1" {
		t.Errorf("EngineVersion = %s, want omni_v1", service.config.EngineVersion)
	}

	if service.config.VCN != "x6_clone" {
		t.Errorf("VCN = %s, want x6_clone", service.config.VCN)
	}
}

func TestNewXunfeiCloneServiceCustomDefaults(t *testing.T) {
	config := XunfeiCloneConfig{
		AppID:         "test_app",
		APIKey:        "test_key",
		BaseURL:       "http://custom.url",
		Timeout:       60,
		EngineVersion: "standard",
		VCN:           "x5_clone",
	}

	service := NewXunfeiCloneService(config)

	if service.config.BaseURL != "http://custom.url" {
		t.Errorf("BaseURL = %s, want http://custom.url", service.config.BaseURL)
	}

	if service.config.Timeout != 60 {
		t.Errorf("Timeout = %d, want 60", service.config.Timeout)
	}

	if service.config.VCN != "x5_clone" {
		t.Errorf("VCN = %s, want x5_clone", service.config.VCN)
	}
}

func TestXunfeiCloneServiceProvider(t *testing.T) {
	config := XunfeiCloneConfig{
		AppID:  "test_app",
		APIKey: "test_key",
	}

	service := NewXunfeiCloneService(config)

	if service.Provider() != ProviderXunfei {
		t.Errorf("Provider = %s, want xunfei", service.Provider())
	}
}

func TestMarshalXunfeiAuthBodyFieldOrder(t *testing.T) {
	body, err := marshalXunfeiAuthBody("app123", 1700000000123)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"base":{"appid":"app123","version":"v1","timestamp":"1700000000123"},"model":"remote"}`
	if string(body) != want {
		t.Errorf("auth body = %s, want %s", string(body), want)
	}
}

func TestXunfeiCloneServiceHTTPClient(t *testing.T) {
	config := XunfeiCloneConfig{
		AppID:   "test_app",
		APIKey:  "test_key",
		Timeout: 45,
	}

	service := NewXunfeiCloneService(config)

	if service.httpClient == nil {
		t.Error("httpClient should not be nil")
	}

	expectedTimeout := 45 * time.Second
	if service.httpClient.Timeout != expectedTimeout {
		t.Errorf("httpClient.Timeout = %v, want %v", service.httpClient.Timeout, expectedTimeout)
	}
}

func TestXunfeiCloneServiceWebSocketAppID(t *testing.T) {
	config := XunfeiCloneConfig{
		AppID:          "test_app",
		APIKey:         "test_key",
		WebSocketAppID: "ws_app",
	}

	service := NewXunfeiCloneService(config)

	if service.config.WebSocketAppID != "ws_app" {
		t.Errorf("WebSocketAppID = %s, want ws_app", service.config.WebSocketAppID)
	}
}

func TestXunfeiCloneServiceWebSocketAppIDDefault(t *testing.T) {
	config := XunfeiCloneConfig{
		AppID:  "test_app",
		APIKey: "test_key",
	}

	service := NewXunfeiCloneService(config)

	if service.config.WebSocketAppID != "test_app" {
		t.Errorf("WebSocketAppID = %s, want test_app", service.config.WebSocketAppID)
	}
}

func TestXunfeiCloneServiceEngineVersionStandard(t *testing.T) {
	config := XunfeiCloneConfig{
		AppID:         "test_app",
		APIKey:        "test_key",
		EngineVersion: "standard",
	}

	service := NewXunfeiCloneService(config)

	if service.config.VCN != "x5_clone" {
		t.Errorf("VCN = %s, want x5_clone for standard engine", service.config.VCN)
	}
}

func TestXunfeiCloneServiceConfig(t *testing.T) {
	config := XunfeiCloneConfig{
		AppID:              "test_app",
		APIKey:             "test_key",
		BaseURL:            "http://test.url",
		Timeout:            50,
		EngineVersion:      "omni_v1",
		VCN:                "x6_clone",
		WebSocketAppID:     "ws_app",
		WebSocketAPIKey:    "ws_key",
		WebSocketAPISecret: "ws_secret",
	}

	service := NewXunfeiCloneService(config)

	if service.config.AppID != "test_app" {
		t.Errorf("AppID = %s, want test_app", service.config.AppID)
	}

	if service.config.APIKey != "test_key" {
		t.Errorf("APIKey = %s, want test_key", service.config.APIKey)
	}

	if service.config.WebSocketAPIKey != "ws_key" {
		t.Errorf("WebSocketAPIKey = %s, want ws_key", service.config.WebSocketAPIKey)
	}

	if service.config.WebSocketAPISecret != "ws_secret" {
		t.Errorf("WebSocketAPISecret = %s, want ws_secret", service.config.WebSocketAPISecret)
	}
}

func TestXunfeiCloneServiceInitialState(t *testing.T) {
	config := XunfeiCloneConfig{
		AppID:  "test_app",
		APIKey: "test_key",
	}

	service := NewXunfeiCloneService(config)

	if service.token != nil {
		t.Error("token should be nil initially")
	}

	if !service.tokenExpiry.IsZero() {
		t.Error("tokenExpiry should be zero initially")
	}
}

func TestXunfeiCloneServiceConfigPointer(t *testing.T) {
	config := XunfeiCloneConfig{
		AppID:  "test_app",
		APIKey: "test_key",
	}

	service := NewXunfeiCloneService(config)

	if service.config == nil {
		t.Error("config should not be nil")
	}

	if service.config.AppID != "test_app" {
		t.Errorf("config.AppID = %s, want test_app", service.config.AppID)
	}
}

func TestXunfeiCloneServiceMultipleInstances(t *testing.T) {
	config1 := XunfeiCloneConfig{
		AppID:  "app1",
		APIKey: "key1",
	}

	config2 := XunfeiCloneConfig{
		AppID:  "app2",
		APIKey: "key2",
	}

	service1 := NewXunfeiCloneService(config1)
	service2 := NewXunfeiCloneService(config2)

	if service1.config.AppID == service2.config.AppID {
		t.Error("Different services should have different AppIDs")
	}

	if service1.httpClient == service2.httpClient {
		t.Error("Different services should have different HTTP clients")
	}
}

func TestXunfeiCloneServiceConfigModification(t *testing.T) {
	config := XunfeiCloneConfig{
		AppID:  "test_app",
		APIKey: "test_key",
	}

	service := NewXunfeiCloneService(config)

	// Verify that modifying the original config doesn't affect the service
	config.AppID = "modified_app"

	if service.config.AppID != "test_app" {
		t.Errorf("Service AppID should not be affected by config modification")
	}
}

func TestXunfeiCloneServiceEmptyConfig(t *testing.T) {
	config := XunfeiCloneConfig{}

	service := NewXunfeiCloneService(config)

	if service == nil {
		t.Error("Service should still be created with empty config")
	}

	if service.config.BaseURL != "http://opentrain.xfyousheng.com" {
		t.Error("Default BaseURL should be set")
	}
}

func TestXunfeiCloneServiceZeroTimeout(t *testing.T) {
	config := XunfeiCloneConfig{
		AppID:   "test_app",
		APIKey:  "test_key",
		Timeout: 0,
	}

	service := NewXunfeiCloneService(config)

	if service.config.Timeout != 30 {
		t.Errorf("Timeout = %d, want 30 (default)", service.config.Timeout)
	}
}

func TestXunfeiCloneServiceNegativeTimeout(t *testing.T) {
	config := XunfeiCloneConfig{
		AppID:   "test_app",
		APIKey:  "test_key",
		Timeout: -1,
	}

	service := NewXunfeiCloneService(config)

	// Negative timeout is preserved
	if service.config.Timeout != -1 {
		t.Errorf("Timeout = %d, want -1", service.config.Timeout)
	}
}
