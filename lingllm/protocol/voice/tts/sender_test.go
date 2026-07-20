package tts

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestAudioSenderCreation(t *testing.T) {
	config := AudioSenderConfig{
		OutputCodec:      "pcm",
		TargetSampleRate: 16000,
		SendCallback: func(data []byte) error {
			return nil
		},
	}

	sender, err := NewAudioSender(config)

	if err != nil {
		t.Errorf("NewAudioSender failed: %v", err)
	}

	if sender == nil {
		t.Error("AudioSender should not be nil")
	}
}

func TestAudioSenderNoSendCallback(t *testing.T) {
	config := AudioSenderConfig{
		OutputCodec:      "pcm",
		TargetSampleRate: 16000,
		SendCallback:     nil,
	}

	_, err := NewAudioSender(config)

	if err != ErrSendCallbackRequired {
		t.Errorf("Error = %v, want ErrSendCallbackRequired", err)
	}
}

func TestAudioSenderStart(t *testing.T) {
	config := AudioSenderConfig{
		OutputCodec:      "pcm",
		TargetSampleRate: 16000,
		SendCallback: func(data []byte) error {
			return nil
		},
	}

	sender, _ := NewAudioSender(config)
	err := sender.Start(context.Background())

	if err != nil {
		t.Errorf("Start failed: %v", err)
	}

	if sender.ctx == nil {
		t.Error("Context should be set after Start")
	}
}

func TestAudioSenderStop(t *testing.T) {
	config := AudioSenderConfig{
		OutputCodec:      "pcm",
		TargetSampleRate: 16000,
		SendCallback: func(data []byte) error {
			return nil
		},
	}

	sender, _ := NewAudioSender(config)
	sender.Start(context.Background())

	err := sender.Stop()

	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}
}

func TestAudioSenderProcessFrame(t *testing.T) {
	var mu sync.Mutex
	sendCount := 0
	config := AudioSenderConfig{
		OutputCodec:      "pcm",
		TargetSampleRate: 16000,
		SendCallback: func(data []byte) error {
			mu.Lock()
			defer mu.Unlock()
			sendCount++
			return nil
		},
	}

	sender, _ := NewAudioSender(config)
	sender.Start(context.Background())

	frame := AudioFrame{
		Data:       make([]byte, 1920),
		SampleRate: 16000,
		Channels:   1,
		PlayID:     "test",
		Sequence:   0,
	}

	err := sender.ProcessFrame(frame)

	if err != nil {
		t.Errorf("ProcessFrame failed: %v", err)
	}

	// Give time for async processing
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if sendCount == 0 {
		t.Error("SendCallback should have been called")
	}
}

func TestAudioSenderGetBufferLevel(t *testing.T) {
	config := AudioSenderConfig{
		OutputCodec:      "pcm",
		TargetSampleRate: 16000,
		SendCallback: func(data []byte) error {
			return nil
		},
	}

	sender, _ := NewAudioSender(config)

	level := sender.GetBufferLevel()

	if level != 0 {
		t.Errorf("Initial buffer level = %d, want 0", level)
	}
}

func TestAudioSenderReset(t *testing.T) {
	config := AudioSenderConfig{
		OutputCodec:      "pcm",
		TargetSampleRate: 16000,
		SendCallback: func(data []byte) error {
			return nil
		},
	}

	sender, _ := NewAudioSender(config)
	sender.Start(context.Background())

	frame := AudioFrame{
		Data:       make([]byte, 1920),
		SampleRate: 16000,
		Channels:   1,
		PlayID:     "test",
		Sequence:   0,
	}

	sender.ProcessFrame(frame)
	sender.Reset()

	level := sender.GetBufferLevel()

	if level != 0 {
		t.Errorf("Buffer level after reset = %d, want 0", level)
	}
}

func TestAudioSenderSetOutputCodec(t *testing.T) {
	config := AudioSenderConfig{
		OutputCodec:      "pcm",
		TargetSampleRate: 16000,
		SendCallback: func(data []byte) error {
			return nil
		},
	}

	sender, _ := NewAudioSender(config)

	err := sender.SetOutputCodec("pcm")

	if err != nil {
		t.Errorf("SetOutputCodec failed: %v", err)
	}
}

func TestAudioSenderSetLogger(t *testing.T) {
	config := AudioSenderConfig{
		OutputCodec:      "pcm",
		TargetSampleRate: 16000,
		SendCallback: func(data []byte) error {
			return nil
		},
	}

	sender, _ := NewAudioSender(config)

	sender.SetLogger(func(msg string) {
		// Logger callback
	})

	if sender.logger == nil {
		t.Error("Logger should be set")
	}
}

func TestAudioSenderClose(t *testing.T) {
	config := AudioSenderConfig{
		OutputCodec:      "pcm",
		TargetSampleRate: 16000,
		SendCallback: func(data []byte) error {
			return nil
		},
	}

	sender, _ := NewAudioSender(config)
	sender.Start(context.Background())

	err := sender.Close()

	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestAudioSenderDefaultConfig(t *testing.T) {
	config := AudioSenderConfig{
		SendCallback: func(data []byte) error {
			return nil
		},
	}

	sender, _ := NewAudioSender(config)

	if sender.config.TargetSampleRate != 16000 {
		t.Errorf("TargetSampleRate = %d, want 16000", sender.config.TargetSampleRate)
	}

	if sender.config.FrameDuration != 60*time.Millisecond {
		t.Errorf("FrameDuration = %v, want 60ms", sender.config.FrameDuration)
	}
}

func TestAudioSenderGetPendingCount(t *testing.T) {
	pendingCount := 5
	config := AudioSenderConfig{
		OutputCodec:      "pcm",
		TargetSampleRate: 16000,
		SendCallback: func(data []byte) error {
			return nil
		},
		GetPendingCountFunc: func() int {
			return pendingCount
		},
	}

	sender, _ := NewAudioSender(config)

	count := sender.GetPendingCount()

	if count != pendingCount {
		t.Errorf("GetPendingCount = %d, want %d", count, pendingCount)
	}
}
