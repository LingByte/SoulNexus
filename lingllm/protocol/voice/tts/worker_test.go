package tts

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestTTSWorkerPoolCreation(t *testing.T) {
	mockService := &MockTTSService{}
	inputCh := make(chan TextSegment)
	outputCh := make(chan AudioFrame)

	config := TTSWorkerConfig{
		TTSService:  mockService,
		WorkerCount: 2,
	}

	pool, err := NewTTSWorkerPool(config, inputCh, outputCh)

	if err != nil {
		t.Errorf("NewTTSWorkerPool failed: %v", err)
	}

	if pool == nil {
		t.Error("TTSWorkerPool should not be nil")
	}
}

func TestTTSWorkerPoolNoTTSService(t *testing.T) {
	inputCh := make(chan TextSegment)
	outputCh := make(chan AudioFrame)

	config := TTSWorkerConfig{
		TTSService:  nil,
		WorkerCount: 1,
	}

	_, err := NewTTSWorkerPool(config, inputCh, outputCh)

	if err != ErrTTSServiceRequired {
		t.Errorf("Error = %v, want ErrTTSServiceRequired", err)
	}
}

func TestTTSWorkerPoolStart(t *testing.T) {
	mockService := &MockTTSService{}
	inputCh := make(chan TextSegment)
	outputCh := make(chan AudioFrame)

	config := TTSWorkerConfig{
		TTSService:  mockService,
		WorkerCount: 1,
	}

	pool, _ := NewTTSWorkerPool(config, inputCh, outputCh)
	err := pool.Start(context.Background())

	if err != nil {
		t.Errorf("Start failed: %v", err)
	}

	if pool.ctx == nil {
		t.Error("Context should be set after Start")
	}
}

func TestTTSWorkerPoolStop(t *testing.T) {
	mockService := &MockTTSService{}
	inputCh := make(chan TextSegment)
	outputCh := make(chan AudioFrame)

	config := TTSWorkerConfig{
		TTSService:  mockService,
		WorkerCount: 1,
	}

	pool, _ := NewTTSWorkerPool(config, inputCh, outputCh)
	pool.Start(context.Background())

	err := pool.Stop()

	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}
}

func TestTTSWorkerPoolClose(t *testing.T) {
	mockService := &MockTTSService{}
	inputCh := make(chan TextSegment)
	outputCh := make(chan AudioFrame)

	config := TTSWorkerConfig{
		TTSService:  mockService,
		WorkerCount: 1,
	}

	pool, _ := NewTTSWorkerPool(config, inputCh, outputCh)
	pool.Start(context.Background())

	err := pool.Close()

	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestTTSWorkerPoolSynthesis(t *testing.T) {
	var mu sync.Mutex
	synthesizeCalled := false
	mockService := &MockTTSService{
		synthesizeFunc: func(ctx context.Context, text string, callback func([]byte) error) error {
			mu.Lock()
			defer mu.Unlock()
			synthesizeCalled = true
			// Simulate TTS output
			return callback(make([]byte, 1920))
		},
	}

	inputCh := make(chan TextSegment, 10)
	outputCh := make(chan AudioFrame, 10)

	config := TTSWorkerConfig{
		TTSService:  mockService,
		WorkerCount: 1,
	}

	pool, _ := NewTTSWorkerPool(config, inputCh, outputCh)
	pool.Start(context.Background())

	// Send a text segment
	inputCh <- TextSegment{
		Text:    "Hello world",
		IsFinal: true,
		PlayID:  "test",
	}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if !synthesizeCalled {
		t.Error("TTS service should have been called")
	}

	pool.Stop()
}

func TestTTSWorkerPoolUpdateTTSService(t *testing.T) {
	mockService1 := &MockTTSService{}
	mockService2 := &MockTTSService{}

	inputCh := make(chan TextSegment)
	outputCh := make(chan AudioFrame)

	config := TTSWorkerConfig{
		TTSService:  mockService1,
		WorkerCount: 1,
	}

	pool, _ := NewTTSWorkerPool(config, inputCh, outputCh)

	err := pool.UpdateTTSService(mockService2)

	if err != nil {
		t.Errorf("UpdateTTSService failed: %v", err)
	}
}

func TestTTSWorkerPoolUpdateTTSServiceNil(t *testing.T) {
	mockService := &MockTTSService{}
	inputCh := make(chan TextSegment)
	outputCh := make(chan AudioFrame)

	config := TTSWorkerConfig{
		TTSService:  mockService,
		WorkerCount: 1,
	}

	pool, _ := NewTTSWorkerPool(config, inputCh, outputCh)

	err := pool.UpdateTTSService(nil)

	if err != ErrTTSServiceRequired {
		t.Errorf("Error = %v, want ErrTTSServiceRequired", err)
	}
}

func TestTTSWorkerPoolGetGlobalSequence(t *testing.T) {
	mockService := &MockTTSService{}
	inputCh := make(chan TextSegment)
	outputCh := make(chan AudioFrame)

	config := TTSWorkerConfig{
		TTSService:  mockService,
		WorkerCount: 1,
	}

	pool, _ := NewTTSWorkerPool(config, inputCh, outputCh)

	seq := pool.GetGlobalSequence()

	if seq != 0 {
		t.Errorf("Initial sequence = %d, want 0", seq)
	}
}

func TestTTSWorkerPoolDefaultWorkerCount(t *testing.T) {
	mockService := &MockTTSService{}
	inputCh := make(chan TextSegment)
	outputCh := make(chan AudioFrame)

	config := TTSWorkerConfig{
		TTSService:  mockService,
		WorkerCount: 0, // Should default to 1
	}

	pool, _ := NewTTSWorkerPool(config, inputCh, outputCh)

	if pool.config.WorkerCount != 1 {
		t.Errorf("WorkerCount = %d, want 1", pool.config.WorkerCount)
	}
}

func TestTTSWorkerPoolSetLogger(t *testing.T) {
	mockService := &MockTTSService{}
	inputCh := make(chan TextSegment)
	outputCh := make(chan AudioFrame)

	config := TTSWorkerConfig{
		TTSService:  mockService,
		WorkerCount: 1,
		Logger: func(msg string) {
			// Logger callback
		},
	}

	pool, _ := NewTTSWorkerPool(config, inputCh, outputCh)

	if pool.logger == nil {
		t.Error("Logger should be set")
	}
}

func TestTTSWorkerPoolMultipleWorkers(t *testing.T) {
	mockService := &MockTTSService{
		synthesizeFunc: func(ctx context.Context, text string, callback func([]byte) error) error {
			return callback(make([]byte, 1920))
		},
	}

	inputCh := make(chan TextSegment, 10)
	outputCh := make(chan AudioFrame, 10)

	config := TTSWorkerConfig{
		TTSService:  mockService,
		WorkerCount: 3,
	}

	pool, _ := NewTTSWorkerPool(config, inputCh, outputCh)
	pool.Start(context.Background())

	if pool.config.WorkerCount != 3 {
		t.Errorf("WorkerCount = %d, want 3", pool.config.WorkerCount)
	}

	pool.Stop()
}
