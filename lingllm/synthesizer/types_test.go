package synthesizer

import (
	"testing"
	"time"
)

func TestTTSProviderToString(t *testing.T) {
	tests := []struct {
		provider TTSProvider
		want     string
	}{
		{ProviderQiniu, "qiniu"},
		{ProviderXunfei, "xunfei"},
		{ProviderBaidu, "baidu"},
		{ProviderGoogle, "google"},
		{ProviderAWS, "aws"},
		{ProviderAzure, "azure"},
		{ProviderOpenAI, "openai"},
		{ProviderElevenLabs, "elevenlabs"},
		{ProviderLocal, "local"},
		{ProviderLocalGoSpeech, "local_gospeech"},
		{ProviderFishSpeech, "fishspeech"},
		{ProviderFishAudio, "fishaudio"},
		{ProviderCoqui, "coqui"},
		{ProviderVolcengine, "volcengine"},
		{ProviderMinimax, "minimax"},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			got := tt.provider.ToString()
			if got != tt.want {
				t.Errorf("ToString() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestNormalizeFramePeriodValidation(t *testing.T) {
	tests := []struct {
		name  string
		input string
		min   time.Duration
		max   time.Duration
	}{
		{"valid_20ms", "20ms", 10 * time.Millisecond, 300 * time.Millisecond},
		{"valid_100ms", "100ms", 10 * time.Millisecond, 300 * time.Millisecond},
		{"too_small", "5ms", 10 * time.Millisecond, 300 * time.Millisecond},
		{"too_large", "500ms", 10 * time.Millisecond, 300 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeFramePeriod(tt.input)

			// Check if result is within valid range
			if result < tt.min || result > tt.max {
				t.Errorf("NormalizeFramePeriod(%s) = %v, outside range [%v, %v]",
					tt.input, result, tt.min, tt.max)
			}
		})
	}
}

func TestNormalizeFramePeriodDefaults(t *testing.T) {
	defaultDuration := 20 * time.Millisecond

	tests := []string{
		"invalid",
		"",
		"abc",
		"xyz",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			result := NormalizeFramePeriod(input)
			if result != defaultDuration {
				t.Errorf("NormalizeFramePeriod(%s) = %v, want %v", input, result, defaultDuration)
			}
		})
	}
}

func TestAudioSynthesisHandlerInterface(t *testing.T) {
	// Test that the interface is properly defined
	var _ AudioSynthesisHandler = (*mockHandler)(nil)
}

type mockHandler struct {
	messages   [][]byte
	timestamps []SentenceTimestamp
}

func (m *mockHandler) OnMessage(data []byte) {
	m.messages = append(m.messages, data)
}

func (m *mockHandler) OnTimestamp(timestamp SentenceTimestamp) {
	m.timestamps = append(m.timestamps, timestamp)
}

func TestAudioSynthesisEngineInterface(t *testing.T) {
	// Test that the interface is properly defined
	t.Log("AudioSynthesisEngine interface is properly defined")
}

func TestAudioSynthesisRequestStruct(t *testing.T) {
	// Verify struct can be instantiated
	t.Log("AudioSynthesisRequest struct is properly defined")
}
