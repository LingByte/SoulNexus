package tts

import (
	"context"
	"testing"
)

// MockTTSService is a mock implementation of TTSService for testing.
type MockTTSService struct {
	synthesizeFunc func(ctx context.Context, text string, callback func([]byte) error) error
}

func (m *MockTTSService) Synthesize(ctx context.Context, text string, callback func([]byte) error) error {
	if m.synthesizeFunc != nil {
		return m.synthesizeFunc(ctx, text, callback)
	}
	// Default: return some dummy PCM data
	return callback(make([]byte, 1920))
}

func TestTTSPipelineCreation(t *testing.T) {
	mockService := &MockTTSService{}

	config := TTSPipelineConfig{
		TTSService: mockService,
		SendCallback: func(data []byte) error {
			return nil
		},
	}

	pipeline, err := NewTTSPipeline(config)

	if err != nil {
		t.Errorf("NewTTSPipeline failed: %v", err)
	}

	if pipeline == nil {
		t.Error("Pipeline should not be nil")
	}
}

func TestTTSPipelineNoTTSService(t *testing.T) {
	config := TTSPipelineConfig{
		TTSService: nil,
		SendCallback: func(data []byte) error {
			return nil
		},
	}

	_, err := NewTTSPipeline(config)

	if err != ErrTTSServiceRequired {
		t.Errorf("Error = %v, want ErrTTSServiceRequired", err)
	}
}

func TestTTSPipelineNoSendCallback(t *testing.T) {
	mockService := &MockTTSService{}

	config := TTSPipelineConfig{
		TTSService:   mockService,
		SendCallback: nil,
	}

	_, err := NewTTSPipeline(config)

	if err != ErrSendCallbackRequired {
		t.Errorf("Error = %v, want ErrSendCallbackRequired", err)
	}
}

func TestTTSPipelineStart(t *testing.T) {
	mockService := &MockTTSService{}

	config := TTSPipelineConfig{
		TTSService: mockService,
		SendCallback: func(data []byte) error {
			return nil
		},
	}

	pipeline, _ := NewTTSPipeline(config)
	err := pipeline.Start(context.Background())

	if err != nil {
		t.Errorf("Start failed: %v", err)
	}

	if pipeline.ctx == nil {
		t.Error("Context should be set after Start")
	}
}

func TestTTSPipelineStop(t *testing.T) {
	mockService := &MockTTSService{}

	config := TTSPipelineConfig{
		TTSService: mockService,
		SendCallback: func(data []byte) error {
			return nil
		},
	}

	pipeline, _ := NewTTSPipeline(config)
	pipeline.Start(context.Background())

	err := pipeline.Stop()

	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}
}

func TestTTSPipelineClose(t *testing.T) {
	mockService := &MockTTSService{}

	config := TTSPipelineConfig{
		TTSService: mockService,
		SendCallback: func(data []byte) error {
			return nil
		},
	}

	pipeline, _ := NewTTSPipeline(config)
	pipeline.Start(context.Background())

	err := pipeline.Close()

	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestTTSPipelineSynthesize(t *testing.T) {
	synthesizeCalled := false
	mockService := &MockTTSService{
		synthesizeFunc: func(ctx context.Context, text string, callback func([]byte) error) error {
			synthesizeCalled = true
			// Simulate TTS output
			return callback(make([]byte, 1920))
		},
	}

	sendCalled := false
	config := TTSPipelineConfig{
		TTSService: mockService,
		SendCallback: func(data []byte) error {
			sendCalled = true
			return nil
		},
	}

	pipeline, _ := NewTTSPipeline(config)
	pipeline.Start(context.Background())

	err := pipeline.Synthesize(context.Background(), "Hello world")

	if err != nil {
		t.Errorf("Synthesize failed: %v", err)
	}

	if !synthesizeCalled {
		t.Error("TTS service should have been called")
	}

	if !sendCalled {
		t.Error("Send callback should have been called")
	}
}

func TestTTSPipelineSetLogger(t *testing.T) {
	mockService := &MockTTSService{}

	config := TTSPipelineConfig{
		TTSService: mockService,
		SendCallback: func(data []byte) error {
			return nil
		},
	}

	pipeline, _ := NewTTSPipeline(config)

	pipeline.SetLogger(func(msg string) {
		// Logger callback
	})

	if pipeline.logger == nil {
		t.Error("Logger should be set")
	}
}

func TestTTSPipelineSetOnCompleteFunc(t *testing.T) {
	mockService := &MockTTSService{}

	config := TTSPipelineConfig{
		TTSService: mockService,
		SendCallback: func(data []byte) error {
			return nil
		},
	}

	pipeline, _ := NewTTSPipeline(config)

	pipeline.SetOnCompleteFunc(func() {
		// Complete callback
	})

	if pipeline.onCompleteFunc == nil {
		t.Error("OnCompleteFunc should be set")
	}
}

func TestTTSPipelineGetConfig(t *testing.T) {
	mockService := &MockTTSService{}

	config := TTSPipelineConfig{
		TTSService:       mockService,
		OutputCodec:      "opus",
		TargetSampleRate: 48000,
		SendCallback: func(data []byte) error {
			return nil
		},
	}

	pipeline, _ := NewTTSPipeline(config)
	retrievedConfig := pipeline.GetConfig()

	if retrievedConfig.OutputCodec != "opus" {
		t.Errorf("OutputCodec = %s, want 'opus'", retrievedConfig.OutputCodec)
	}

	if retrievedConfig.TargetSampleRate != 48000 {
		t.Errorf("TargetSampleRate = %d, want 48000", retrievedConfig.TargetSampleRate)
	}
}

func TestTTSPipelineDefaultConfig(t *testing.T) {
	mockService := &MockTTSService{}

	config := DefaultTTSPipelineConfig(mockService)

	if config.TTSService == nil {
		t.Error("TTSService should be set")
	}

	if config.TargetSampleRate != 16000 {
		t.Errorf("TargetSampleRate = %d, want 16000", config.TargetSampleRate)
	}

	if config.OutputCodec != "pcm" {
		t.Errorf("OutputCodec = %s, want 'pcm'", config.OutputCodec)
	}
}

func TestTTSPipelineWithTextProcessors(t *testing.T) {
	mockService := &MockTTSService{}

	// Create a simple text processor that converts to uppercase
	textProcessor := &MockTextProcessor{
		processFunc: func(ctx context.Context, data interface{}) (interface{}, bool, error) {
			if text, ok := data.(string); ok {
				return text + " [processed]", true, nil
			}
			return data, true, nil
		},
	}

	config := TTSPipelineConfig{
		TTSService:     mockService,
		TextProcessors: []TTSPipelineComponent{textProcessor},
		SendCallback: func(data []byte) error {
			return nil
		},
	}

	pipeline, _ := NewTTSPipeline(config)
	pipeline.Start(context.Background())

	err := pipeline.Synthesize(context.Background(), "Hello")

	if err != nil {
		t.Errorf("Synthesize failed: %v", err)
	}
}

// MockTextProcessor is a mock text processor for testing.
type MockTextProcessor struct {
	processFunc func(ctx context.Context, data interface{}) (interface{}, bool, error)
}

func (m *MockTextProcessor) Name() string {
	return "mock_text_processor"
}

func (m *MockTextProcessor) Process(ctx context.Context, data interface{}) (interface{}, bool, error) {
	if m.processFunc != nil {
		return m.processFunc(ctx, data)
	}
	return data, true, nil
}
