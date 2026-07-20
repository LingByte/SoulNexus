package voiceclone

import (
	"encoding/json"
	"testing"
)

func TestNewFactory(t *testing.T) {
	factory := NewFactory()

	if factory == nil {
		t.Error("NewFactory should return a non-nil factory")
	}
}

func TestFactoryCreateServiceWithNilConfig(t *testing.T) {
	factory := NewFactory()

	_, err := factory.CreateService(nil)

	if err == nil {
		t.Error("CreateService with nil config should return an error")
	}
}

func TestFactoryCreateServiceXunfei(t *testing.T) {
	factory := NewFactory()
	config := &Config{
		Provider: ProviderXunfei,
		Options: map[string]interface{}{
			"app_id":  "test_app",
			"api_key": "test_key",
		},
	}

	service, err := factory.CreateService(config)

	if err != nil {
		t.Errorf("CreateService failed: %v", err)
	}

	if service == nil {
		t.Error("Service should not be nil")
	}

	if service.Provider() != ProviderXunfei {
		t.Errorf("Provider = %s, want xunfei", service.Provider())
	}
}

func TestFactoryCreateServiceXunfeiMissingAppID(t *testing.T) {
	factory := NewFactory()
	config := &Config{
		Provider: ProviderXunfei,
		Options: map[string]interface{}{
			"api_key": "test_key",
		},
	}

	_, err := factory.CreateService(config)

	if err == nil {
		t.Error("CreateService without app_id should return an error")
	}
}

func TestFactoryCreateServiceXunfeiMissingAPIKey(t *testing.T) {
	factory := NewFactory()
	config := &Config{
		Provider: ProviderXunfei,
		Options: map[string]interface{}{
			"app_id": "test_app",
		},
	}

	_, err := factory.CreateService(config)

	if err == nil {
		t.Error("CreateService without api_key should return an error")
	}
}

func TestFactoryCreateServiceVolcengine(t *testing.T) {
	factory := NewFactory()
	config := &Config{
		Provider: ProviderVolcengine,
		Options: map[string]interface{}{
			"app_id": "test_app",
			"token":  "test_token",
		},
	}

	service, err := factory.CreateService(config)

	if err != nil {
		t.Errorf("CreateService failed: %v", err)
	}

	if service == nil {
		t.Error("Service should not be nil")
	}

	if service.Provider() != ProviderVolcengine {
		t.Errorf("Provider = %s, want volcengine", service.Provider())
	}
}

func TestFactoryCreateServiceVolcengineMissingToken(t *testing.T) {
	factory := NewFactory()
	config := &Config{
		Provider: ProviderVolcengine,
		Options: map[string]interface{}{
			"app_id": "test_app",
		},
	}

	_, err := factory.CreateService(config)

	if err == nil {
		t.Error("CreateService without token should return an error")
	}
}

func TestFactoryCreateServiceUnsupportedProvider(t *testing.T) {
	factory := NewFactory()
	config := &Config{
		Provider: Provider("unsupported"),
		Options:  map[string]interface{}{},
	}

	_, err := factory.CreateService(config)

	if err == nil {
		t.Error("CreateService with unsupported provider should return an error")
	}
}

func TestFactoryCreateServiceFromJSON(t *testing.T) {
	factory := NewFactory()
	jsonConfig := `{
		"provider": "xunfei",
		"options": {
			"app_id": "test_app",
			"api_key": "test_key"
		}
	}`

	service, err := factory.CreateServiceFromJSON(jsonConfig)

	if err != nil {
		t.Errorf("CreateServiceFromJSON failed: %v", err)
	}

	if service == nil {
		t.Error("Service should not be nil")
	}
}

func TestFactoryCreateServiceFromJSONInvalid(t *testing.T) {
	factory := NewFactory()
	jsonConfig := "invalid json"

	_, err := factory.CreateServiceFromJSON(jsonConfig)

	if err == nil {
		t.Error("CreateServiceFromJSON with invalid JSON should return an error")
	}
}

func TestFactoryValidateConfigNil(t *testing.T) {
	factory := NewFactory()

	err := factory.ValidateConfig(nil)

	if err == nil {
		t.Error("ValidateConfig with nil config should return an error")
	}
}

func TestFactoryValidateConfigXunfei(t *testing.T) {
	factory := NewFactory()
	config := &Config{
		Provider: ProviderXunfei,
		Options: map[string]interface{}{
			"app_id":  "test_app",
			"api_key": "test_key",
		},
	}

	err := factory.ValidateConfig(config)

	if err != nil {
		t.Errorf("ValidateConfig failed: %v", err)
	}
}

func TestFactoryValidateConfigXunfeiMissingAppID(t *testing.T) {
	factory := NewFactory()
	config := &Config{
		Provider: ProviderXunfei,
		Options: map[string]interface{}{
			"api_key": "test_key",
		},
	}

	err := factory.ValidateConfig(config)

	if err == nil {
		t.Error("ValidateConfig without app_id should return an error")
	}
}

func TestFactoryValidateConfigVolcengine(t *testing.T) {
	factory := NewFactory()
	config := &Config{
		Provider: ProviderVolcengine,
		Options: map[string]interface{}{
			"app_id": "test_app",
			"token":  "test_token",
		},
	}

	err := factory.ValidateConfig(config)

	if err != nil {
		t.Errorf("ValidateConfig failed: %v", err)
	}
}

func TestFactoryValidateConfigVolcengineMissingToken(t *testing.T) {
	factory := NewFactory()
	config := &Config{
		Provider: ProviderVolcengine,
		Options: map[string]interface{}{
			"app_id": "test_app",
		},
	}

	err := factory.ValidateConfig(config)

	if err == nil {
		t.Error("ValidateConfig without token should return an error")
	}
}

func TestFactoryValidateConfigUnsupportedProvider(t *testing.T) {
	factory := NewFactory()
	config := &Config{
		Provider: Provider("unsupported"),
		Options:  map[string]interface{}{},
	}

	err := factory.ValidateConfig(config)

	if err == nil {
		t.Error("ValidateConfig with unsupported provider should return an error")
	}
}

func TestFactoryGetSupportedProviders(t *testing.T) {
	factory := NewFactory()
	providers := factory.GetSupportedProviders()

	if len(providers) != 2 {
		t.Errorf("GetSupportedProviders returned %d providers, want 2", len(providers))
	}

	if providers[0] != ProviderXunfei {
		t.Errorf("First provider = %s, want xunfei", providers[0])
	}

	if providers[1] != ProviderVolcengine {
		t.Errorf("Second provider = %s, want volcengine", providers[1])
	}
}

func TestFactoryCreateXunfeiServiceWithDefaults(t *testing.T) {
	factory := NewFactory()
	config := &Config{
		Provider: ProviderXunfei,
		Options: map[string]interface{}{
			"app_id":  "test_app",
			"api_key": "test_key",
		},
	}

	service, err := factory.CreateService(config)

	if err != nil {
		t.Errorf("CreateService failed: %v", err)
	}

	xunfeiService, ok := service.(*XunfeiCloneService)
	if !ok {
		t.Error("Service should be XunfeiCloneService")
	}

	if xunfeiService.config.BaseURL != "http://opentrain.xfyousheng.com" {
		t.Errorf("BaseURL = %s, want http://opentrain.xfyousheng.com", xunfeiService.config.BaseURL)
	}

	if xunfeiService.config.Timeout != 30 {
		t.Errorf("Timeout = %d, want 30", xunfeiService.config.Timeout)
	}
}

func TestFactoryCreateVolcengineServiceWithDefaults(t *testing.T) {
	factory := NewFactory()
	config := &Config{
		Provider: ProviderVolcengine,
		Options: map[string]interface{}{
			"app_id": "test_app",
			"token":  "test_token",
		},
	}

	service, err := factory.CreateService(config)

	if err != nil {
		t.Errorf("CreateService failed: %v", err)
	}

	volcengineService, ok := service.(*VolcengineCloneService)
	if !ok {
		t.Error("Service should be VolcengineCloneService")
	}

	if volcengineService.config.Cluster != "volcano_icl" {
		t.Errorf("Cluster = %s, want volcano_icl", volcengineService.config.Cluster)
	}
}

func TestFactoryCreateServiceFromJSONWithAllOptions(t *testing.T) {
	factory := NewFactory()

	config := Config{
		Provider: ProviderXunfei,
		Options: map[string]interface{}{
			"app_id":         "test_app",
			"api_key":        "test_key",
			"base_url":       "http://custom.url",
			"timeout":        60,
			"engine_version": "omni_v1",
			"vcn":            "x6_clone",
			"ws_app_id":      "ws_app",
			"ws_api_key":     "ws_key",
			"ws_api_secret":  "ws_secret",
		},
	}

	jsonData, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	service, err := factory.CreateServiceFromJSON(string(jsonData))

	if err != nil {
		t.Errorf("CreateServiceFromJSON failed: %v", err)
	}

	if service == nil {
		t.Error("Service should not be nil")
	}
}
