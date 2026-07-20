package voiceclone

import (
	"testing"
)

func TestNewVolcengineCloneService(t *testing.T) {
	config := VolcengineCloneConfig{
		AppID: "test_app",
		Token: "test_token",
	}

	service := NewVolcengineCloneService(config)

	if service == nil {
		t.Error("NewVolcengineCloneService should return a non-nil service")
	}

	if service.config.AppID != "test_app" {
		t.Errorf("AppID = %s, want test_app", service.config.AppID)
	}

	if service.config.Token != "test_token" {
		t.Errorf("Token = %s, want test_token", service.config.Token)
	}
}

func TestNewVolcengineCloneServiceDefaults(t *testing.T) {
	config := VolcengineCloneConfig{
		AppID: "test_app",
		Token: "test_token",
	}

	service := NewVolcengineCloneService(config)

	if service.config.Cluster != "volcano_icl" {
		t.Errorf("Cluster = %s, want volcano_icl", service.config.Cluster)
	}

	if service.config.Encoding != "pcm" {
		t.Errorf("Encoding = %s, want pcm", service.config.Encoding)
	}

	if service.config.SampleRate != 8000 {
		t.Errorf("SampleRate = %d, want 8000", service.config.SampleRate)
	}

	if service.config.BitDepth != 16 {
		t.Errorf("BitDepth = %d, want 16", service.config.BitDepth)
	}

	if service.config.Channels != 1 {
		t.Errorf("Channels = %d, want 1", service.config.Channels)
	}

	if service.config.FrameDuration != "20ms" {
		t.Errorf("FrameDuration = %s, want 20ms", service.config.FrameDuration)
	}

	if service.config.SpeedRatio != 1.0 {
		t.Errorf("SpeedRatio = %f, want 1.0", service.config.SpeedRatio)
	}

	if service.config.TrainingTimes != 1 {
		t.Errorf("TrainingTimes = %d, want 1", service.config.TrainingTimes)
	}
}

func TestNewVolcengineCloneServiceCustomDefaults(t *testing.T) {
	config := VolcengineCloneConfig{
		AppID:         "test_app",
		Token:         "test_token",
		Cluster:       "custom_cluster",
		VoiceType:     "voice123",
		Encoding:      "wav",
		SampleRate:    16000,
		BitDepth:      24,
		Channels:      2,
		FrameDuration: "40ms",
		SpeedRatio:    1.5,
		TrainingTimes: 3,
	}

	service := NewVolcengineCloneService(config)

	if service.config.Cluster != "custom_cluster" {
		t.Errorf("Cluster = %s, want custom_cluster", service.config.Cluster)
	}

	if service.config.VoiceType != "voice123" {
		t.Errorf("VoiceType = %s, want voice123", service.config.VoiceType)
	}

	if service.config.Encoding != "wav" {
		t.Errorf("Encoding = %s, want wav", service.config.Encoding)
	}

	if service.config.SampleRate != 16000 {
		t.Errorf("SampleRate = %d, want 16000", service.config.SampleRate)
	}

	if service.config.BitDepth != 24 {
		t.Errorf("BitDepth = %d, want 24", service.config.BitDepth)
	}

	if service.config.Channels != 2 {
		t.Errorf("Channels = %d, want 2", service.config.Channels)
	}

	if service.config.FrameDuration != "40ms" {
		t.Errorf("FrameDuration = %s, want 40ms", service.config.FrameDuration)
	}

	if service.config.SpeedRatio != 1.5 {
		t.Errorf("SpeedRatio = %f, want 1.5", service.config.SpeedRatio)
	}

	if service.config.TrainingTimes != 3 {
		t.Errorf("TrainingTimes = %d, want 3", service.config.TrainingTimes)
	}
}

func TestVolcengineCloneServiceProvider(t *testing.T) {
	config := VolcengineCloneConfig{
		AppID: "test_app",
		Token: "test_token",
	}

	service := NewVolcengineCloneService(config)

	if service.Provider() != ProviderVolcengine {
		t.Errorf("Provider = %s, want volcengine", service.Provider())
	}
}

func TestVolcengineCloneServiceHTTPClient(t *testing.T) {
	config := VolcengineCloneConfig{
		AppID: "test_app",
		Token: "test_token",
	}

	service := NewVolcengineCloneService(config)

	if service.httpClient == nil {
		t.Error("httpClient should not be nil")
	}
}

func TestVolcengineCloneServiceInitialState(t *testing.T) {
	config := VolcengineCloneConfig{
		AppID: "test_app",
		Token: "test_token",
	}

	service := NewVolcengineCloneService(config)

	if service == nil {
		t.Error("Service should be initialized")
	}

	if service.config == nil {
		t.Error("Config should be initialized")
	}
}

func TestVolcengineCloneServiceConfig(t *testing.T) {
	config := VolcengineCloneConfig{
		AppID:         "test_app",
		Token:         "test_token",
		Cluster:       "test_cluster",
		VoiceType:     "voice123",
		Encoding:      "pcm",
		SampleRate:    16000,
		BitDepth:      16,
		Channels:      1,
		FrameDuration: "20ms",
		SpeedRatio:    1.0,
		TrainingTimes: 1,
	}

	service := NewVolcengineCloneService(config)

	if service.config.AppID != "test_app" {
		t.Errorf("AppID = %s, want test_app", service.config.AppID)
	}

	if service.config.Token != "test_token" {
		t.Errorf("Token = %s, want test_token", service.config.Token)
	}

	if service.config.Cluster != "test_cluster" {
		t.Errorf("Cluster = %s, want test_cluster", service.config.Cluster)
	}

	if service.config.VoiceType != "voice123" {
		t.Errorf("VoiceType = %s, want voice123", service.config.VoiceType)
	}
}

func TestVolcengineCloneServiceConfigPointer(t *testing.T) {
	config := VolcengineCloneConfig{
		AppID: "test_app",
		Token: "test_token",
	}

	service := NewVolcengineCloneService(config)

	if service.config == nil {
		t.Error("config should not be nil")
	}

	if service.config.AppID != "test_app" {
		t.Errorf("config.AppID = %s, want test_app", service.config.AppID)
	}
}

func TestVolcengineCloneServiceMultipleInstances(t *testing.T) {
	config1 := VolcengineCloneConfig{
		AppID: "app1",
		Token: "token1",
	}

	config2 := VolcengineCloneConfig{
		AppID: "app2",
		Token: "token2",
	}

	service1 := NewVolcengineCloneService(config1)
	service2 := NewVolcengineCloneService(config2)

	if service1.config.AppID == service2.config.AppID {
		t.Error("Different services should have different AppIDs")
	}

	if service1.httpClient == service2.httpClient {
		t.Error("Different services should have different HTTP clients")
	}
}

func TestVolcengineCloneServiceConfigModification(t *testing.T) {
	config := VolcengineCloneConfig{
		AppID: "test_app",
		Token: "test_token",
	}

	service := NewVolcengineCloneService(config)

	// Verify that modifying the original config doesn't affect the service
	config.AppID = "modified_app"

	if service.config.AppID != "test_app" {
		t.Errorf("Service AppID should not be affected by config modification")
	}
}

func TestVolcengineCloneServiceEmptyConfig(t *testing.T) {
	config := VolcengineCloneConfig{}

	service := NewVolcengineCloneService(config)

	if service == nil {
		t.Error("Service should still be created with empty config")
	}

	if service.config.Cluster != "volcano_icl" {
		t.Error("Default Cluster should be set")
	}
}

func TestVolcengineCloneServiceZeroValues(t *testing.T) {
	config := VolcengineCloneConfig{
		AppID:      "test_app",
		Token:      "test_token",
		SampleRate: 0,
		BitDepth:   0,
		Channels:   0,
		SpeedRatio: 0,
	}

	service := NewVolcengineCloneService(config)

	// Constructor sets defaults for zero values
	if service.config.SampleRate != 8000 {
		t.Errorf("SampleRate = %d, want 8000 (default)", service.config.SampleRate)
	}

	if service.config.BitDepth != 16 {
		t.Errorf("BitDepth = %d, want 16 (default)", service.config.BitDepth)
	}

	if service.config.Channels != 1 {
		t.Errorf("Channels = %d, want 1 (default)", service.config.Channels)
	}

	if service.config.SpeedRatio != 1.0 {
		t.Errorf("SpeedRatio = %f, want 1.0 (default)", service.config.SpeedRatio)
	}
}

func TestVolcengineCloneServiceDefaultCluster(t *testing.T) {
	config := VolcengineCloneConfig{
		AppID: "test_app",
		Token: "test_token",
	}

	service := NewVolcengineCloneService(config)

	if service.config.Cluster != VolcengineCloneCluster {
		t.Errorf("Cluster = %s, want %s", service.config.Cluster, VolcengineCloneCluster)
	}
}
