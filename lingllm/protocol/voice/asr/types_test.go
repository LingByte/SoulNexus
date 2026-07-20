package asr

import (
	"context"
	"testing"
)

func TestStandardPipelineCreation(t *testing.T) {
	inputStages := []PipelineComponent{
		NewPassthroughComponent("input"),
	}

	outputStages := []PipelineComponent{
		NewPassthroughComponent("output"),
	}

	pipeline, err := NewStandardPipeline(inputStages, outputStages)
	if err != nil {
		t.Fatalf("NewStandardPipeline failed: %v", err)
	}

	if pipeline == nil {
		t.Error("Pipeline should not be nil")
	}

	if err := pipeline.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestStandardPipelineEmptyInputStages(t *testing.T) {
	_, err := NewStandardPipeline(nil, []PipelineComponent{NewPassthroughComponent("output")})
	if err == nil {
		t.Error("NewStandardPipeline with empty input stages should return error")
	}

	if err != ErrEmptyInputStages {
		t.Errorf("Error = %v, want ErrEmptyInputStages", err)
	}
}

func TestStandardPipelineProcess(t *testing.T) {
	inputStages := []PipelineComponent{
		NewPassthroughComponent("input"),
	}

	outputStages := []PipelineComponent{
		NewPassthroughComponent("output"),
	}

	pipeline, _ := NewStandardPipeline(inputStages, outputStages)
	defer pipeline.Close()

	data := []byte{0x00, 0x01, 0x02}
	result, err := pipeline.Process(context.Background(), data)

	if err != nil {
		t.Errorf("Process failed: %v", err)
	}

	if result == nil {
		t.Error("Result should not be nil")
	}

	resultBytes, ok := result.([]byte)
	if !ok {
		t.Errorf("Result type = %T, want []byte", result)
	}

	if len(resultBytes) != len(data) {
		t.Errorf("Result length = %d, want %d", len(resultBytes), len(data))
	}
}

func TestStandardPipelineSetOutputCallback(t *testing.T) {
	pipeline, _ := NewStandardPipeline(
		[]PipelineComponent{NewPassthroughComponent("input")},
		[]PipelineComponent{NewPassthroughComponent("output")},
	)
	defer pipeline.Close()

	callCount := 0
	pipeline.SetOutputCallback(func(text string, isFinal bool) {
		callCount++
	})

	if pipeline.onOutput == nil {
		t.Error("OnOutput callback should be set")
	}
}

// Helper function for types_test.go
func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestStandardPipelineSetPCMAudioCallback(t *testing.T) {
	pipeline, _ := NewStandardPipeline(
		[]PipelineComponent{NewPassthroughComponent("input")},
		[]PipelineComponent{NewPassthroughComponent("output")},
	)
	defer pipeline.Close()

	pipeline.SetPCMAudioCallback(func(data []byte) error {
		return nil
	})

	if pipeline.onPCMAudio == nil {
		t.Error("OnPCMAudio callback should be set")
	}
}

func TestStandardPipelineTTSPlaying(t *testing.T) {
	pipeline, _ := NewStandardPipeline(
		[]PipelineComponent{NewPassthroughComponent("input")},
		[]PipelineComponent{NewPassthroughComponent("output")},
	)
	defer pipeline.Close()

	if pipeline.IsTTSPlaying() {
		t.Error("TTS should not be playing initially")
	}

	pipeline.SetTTSPlaying(true)
	if !pipeline.IsTTSPlaying() {
		t.Error("TTS should be playing after SetTTSPlaying(true)")
	}

	pipeline.ClearTTSState()
	if pipeline.IsTTSPlaying() {
		t.Error("TTS should not be playing after ClearTTSState()")
	}
}

func TestStandardPipelineResetState(t *testing.T) {
	pipeline, _ := NewStandardPipeline(
		[]PipelineComponent{NewPassthroughComponent("input")},
		[]PipelineComponent{NewPassthroughComponent("output")},
	)
	defer pipeline.Close()

	// Process some data to populate metrics
	pipeline.Process(context.Background(), []byte{0x00, 0x01})

	// Reset state
	pipeline.ResetState()

	metrics := pipeline.GetMetrics()
	if !metrics.FirstPacketTime.IsZero() {
		t.Error("FirstPacketTime should be zero after reset")
	}

	if metrics.TotalAudioBytes != 0 {
		t.Errorf("TotalAudioBytes = %d, want 0", metrics.TotalAudioBytes)
	}
}

func TestStandardPipelineGetMetrics(t *testing.T) {
	pipeline, _ := NewStandardPipeline(
		[]PipelineComponent{NewPassthroughComponent("input")},
		[]PipelineComponent{NewPassthroughComponent("output")},
	)
	defer pipeline.Close()

	// Process some data first
	pipeline.Process(context.Background(), []byte{0x00, 0x01})

	metrics := pipeline.GetMetrics()
	if metrics.FirstPacketTime.IsZero() {
		t.Error("FirstPacketTime should be set after Process")
	}
}

func TestPCMInputComponent(t *testing.T) {
	comp := &PCMInputComponent{}

	if comp.Name() != "pcm_input" {
		t.Errorf("Name = %s, want 'pcm_input'", comp.Name())
	}

	data := []byte{0x00, 0x01, 0x02}
	result, shouldContinue, err := comp.Process(context.Background(), data)

	if err != nil {
		t.Errorf("Process failed: %v", err)
	}

	if !shouldContinue {
		t.Error("shouldContinue should be true")
	}

	resultBytes, ok := result.([]byte)
	if !ok {
		t.Errorf("Result type = %T, want []byte", result)
	}

	if len(resultBytes) != len(data) {
		t.Errorf("Result length = %d, want %d", len(resultBytes), len(data))
	}
}

func TestPCMInputComponentInvalidData(t *testing.T) {
	comp := &PCMInputComponent{}

	_, _, err := comp.Process(context.Background(), "invalid")
	if err == nil {
		t.Error("Process with invalid data should return error")
	}
}

func TestPCMInputComponentEmptyData(t *testing.T) {
	comp := &PCMInputComponent{}

	_, _, err := comp.Process(context.Background(), []byte{})
	if err == nil {
		t.Error("Process with empty data should return error")
	}
}

func TestEchoFilterComponent(t *testing.T) {
	gate := NewPlaybackGate(func() bool { return false }, func() int { return 0 }, 0)
	comp := NewEchoFilterComponent(gate)

	if comp.Name() != "echo_filter" {
		t.Errorf("Name = %s, want 'echo_filter'", comp.Name())
	}

	data := []byte{0x00, 0x01, 0x02}
	result, shouldContinue, err := comp.Process(context.Background(), data)

	if err != nil {
		t.Errorf("Process failed: %v", err)
	}

	if !shouldContinue {
		t.Error("shouldContinue should be true")
	}

	// When TTS is not playing, should return original data
	resultBytes, ok := result.([]byte)
	if !ok {
		t.Errorf("Result type = %T, want []byte", result)
	}

	if len(resultBytes) != len(data) {
		t.Errorf("Result length = %d, want %d", len(resultBytes), len(data))
	}
}

func TestEchoFilterComponentWithTTS(t *testing.T) {
	gate := NewPlaybackGate(func() bool { return true }, func() int { return 0 }, 0)
	comp := NewEchoFilterComponent(gate)

	data := []byte{0x00, 0x01, 0x02}
	result, shouldContinue, err := comp.Process(context.Background(), data)

	if err != nil {
		t.Errorf("Process failed: %v", err)
	}

	if !shouldContinue {
		t.Error("shouldContinue should be true")
	}

	// When TTS is playing, should return silence
	resultBytes, ok := result.([]byte)
	if !ok {
		t.Errorf("Result type = %T, want []byte", result)
	}

	if len(resultBytes) != len(data) {
		t.Errorf("Result length = %d, want %d", len(resultBytes), len(data))
	}

	// Check that it's all zeros (silence)
	for i, b := range resultBytes {
		if b != 0 {
			t.Errorf("Byte %d = %d, want 0 (silence)", i, b)
		}
	}
}

func TestSensitiveFilterComponent(t *testing.T) {
	config := SensitiveFilterConfig{
		Blacklist:   []string{"password", "secret"},
		FilterEmoji: false,
	}
	comp, err := NewSensitiveFilterComponent(config)

	if err != nil {
		t.Errorf("NewSensitiveFilterComponent failed: %v", err)
	}

	if comp.Name() != "sensitive_filter" {
		t.Errorf("Name = %s, want 'sensitive_filter'", comp.Name())
	}

	text := "hello world"
	result, shouldContinue, errProcess := comp.Process(context.Background(), text)

	if errProcess != nil {
		t.Errorf("Process failed: %v", errProcess)
	}

	if !shouldContinue {
		t.Error("shouldContinue should be true")
	}

	resultText, ok := result.(string)
	if !ok {
		t.Errorf("Result type = %T, want string", result)
	}

	if resultText != text {
		t.Error("Result should be the same as input (no sensitive words)")
	}
}

func TestPassthroughComponent(t *testing.T) {
	comp := NewPassthroughComponent("test")

	if comp.Name() != "test" {
		t.Errorf("Name = %s, want 'test'", comp.Name())
	}

	data := []byte{0x00, 0x01, 0x02}
	result, shouldContinue, err := comp.Process(context.Background(), data)

	if err != nil {
		t.Errorf("Process failed: %v", err)
	}

	if !shouldContinue {
		t.Error("shouldContinue should be true")
	}

	resultBytes, ok := result.([]byte)
	if !ok {
		t.Errorf("Result type = %T, want []byte", result)
	}

	if len(resultBytes) != len(data) {
		t.Errorf("Result length = %d, want %d", len(resultBytes), len(data))
	}
}

func TestLoggingComponent(t *testing.T) {
	logCount := 0
	logger := func(msg string) {
		logCount++
	}

	comp := NewLoggingComponent("test_log", logger)

	if comp.Name() != "test_log" {
		t.Errorf("Name = %s, want 'test_log'", comp.Name())
	}

	data := []byte{0x00, 0x01, 0x02}
	result, shouldContinue, err := comp.Process(context.Background(), data)

	if err != nil {
		t.Errorf("Process failed: %v", err)
	}

	if !shouldContinue {
		t.Error("shouldContinue should be true")
	}

	if logCount == 0 {
		t.Error("Logger should have been called")
	}

	resultBytes, ok := result.([]byte)
	if !ok {
		t.Errorf("Result type = %T, want []byte", result)
	}

	if len(resultBytes) != len(data) {
		t.Errorf("Result length = %d, want %d", len(resultBytes), len(data))
	}
}
