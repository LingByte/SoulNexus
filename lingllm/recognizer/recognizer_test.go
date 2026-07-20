package recognizer

import (
	"testing"
	"time"
)

func TestRecognizerCreation(t *testing.T) {
	config := &Config{
		URL: "wss://example.com/asr",
		Audio: AudioConfig{
			Rate:    16000,
			Bits:    16,
			Channel: 1,
		},
		Buffer: BufferConfig{
			SegmentDurationMs: 100,
		},
	}

	recognizer := NewRecognizer(config)
	if recognizer == nil {
		t.Error("NewRecognizer should not return nil")
	}

	if recognizer.targetBufferSize == 0 {
		t.Error("targetBufferSize should be set")
	}
}

func TestRecognizerCallbacks(t *testing.T) {
	config := &Config{
		URL: "wss://example.com/asr",
		Audio: AudioConfig{
			Rate:    16000,
			Bits:    16,
			Channel: 1,
		},
	}

	recognizer := NewRecognizer(config)

	// Test OnResult callback
	recognizer.OnResult(func(result *Result) {
		// Callback registered
	})

	// Test OnError callback
	recognizer.OnError(func(err error) {
		// Callback registered
	})

	if recognizer.onResult == nil {
		t.Error("OnResult callback should be set")
	}

	if recognizer.onError == nil {
		t.Error("OnError callback should be set")
	}
}

func TestRecognizerAudioBuffer(t *testing.T) {
	config := &Config{
		URL: "wss://example.com/asr",
		Audio: AudioConfig{
			Rate:    16000,
			Bits:    16,
			Channel: 1,
		},
		Buffer: BufferConfig{
			SegmentDurationMs: 100,
		},
	}

	recognizer := NewRecognizer(config)

	// Test sending audio frame
	frame := []byte{0x01, 0x02, 0x03}
	err := recognizer.SendAudioFrame(frame, false)
	if err != nil {
		t.Errorf("SendAudioFrame failed: %v", err)
	}

	// Verify buffer contains data
	if len(recognizer.pendingAudio) == 0 {
		t.Error("pendingAudio should contain data after SendAudioFrame")
	}
}

func TestRecognizerEndFrame(t *testing.T) {
	config := &Config{
		URL: "wss://example.com/asr",
		Audio: AudioConfig{
			Rate:    16000,
			Bits:    16,
			Channel: 1,
		},
	}

	recognizer := NewRecognizer(config)

	// Send end frame
	frame := []byte{0x01, 0x02}
	err := recognizer.SendAudioFrame(frame, true)
	if err != nil {
		t.Errorf("SendAudioFrame with end=true failed: %v", err)
	}

	if !recognizer.isEndFrameSent {
		t.Error("isEndFrameSent should be true after sending end frame")
	}

	// Subsequent frames should be ignored
	err = recognizer.SendAudioFrame([]byte{0x03}, false)
	if err != nil {
		t.Errorf("SendAudioFrame after end frame should not error: %v", err)
	}
}

func TestRecognizerResult(t *testing.T) {
	resp := &Response{
		Code:          0,
		IsLastPackage: false,
	}

	config := &Config{
		URL: "wss://example.com/asr",
	}
	recognizer := NewRecognizer(config)

	result := recognizer.convertResponseToResult(resp)

	if result.Error != nil {
		t.Errorf("Result.Error should be nil for successful response, got %v", result.Error)
	}

	if result.IsFinal {
		t.Error("Result.IsFinal should be false")
	}
}

func TestRecognizerResultWithError(t *testing.T) {
	resp := &Response{
		Code:          500,
		IsLastPackage: false,
	}

	config := &Config{
		URL: "wss://example.com/asr",
	}
	recognizer := NewRecognizer(config)

	result := recognizer.convertResponseToResult(resp)

	if result.Error == nil {
		t.Error("Result.Error should not be nil for error response")
	}
}

func TestRecognizerTimeoutConfig(t *testing.T) {
	config := &Config{
		URL: "wss://example.com/asr",
		Audio: AudioConfig{
			Rate:    16000,
			Bits:    16,
			Channel: 1,
		},
	}

	recognizer := NewRecognizer(config)

	if recognizer.timeoutConfig.Send == 0 {
		t.Error("Send timeout should be set")
	}

	if recognizer.timeoutConfig.Read == 0 {
		t.Error("Read timeout should be set")
	}

	if recognizer.timeoutConfig.Send != 10*time.Second {
		t.Errorf("Send timeout = %v, want 10s", recognizer.timeoutConfig.Send)
	}

	if recognizer.timeoutConfig.Read != 30*time.Second {
		t.Errorf("Read timeout = %v, want 30s", recognizer.timeoutConfig.Read)
	}
}
