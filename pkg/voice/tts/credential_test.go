package tts

import (
	"testing"

	"github.com/LingByte/lingllm/synthesizer"
)

func TestNewFromCredential_Aliyun(t *testing.T) {
	handle, err := NewFromCredential(synthesizer.TTSCredentialConfig{
		"provider": "aliyun",
		"apiKey":   "sk-test",
	})
	if err != nil {
		t.Fatalf("NewFromCredential: %v", err)
	}
	if handle.Engine.Provider() != synthesizer.ProviderAliyun {
		t.Fatalf("provider=%q", handle.Engine.Provider())
	}
	if handle.SampleRate != 24000 {
		t.Fatalf("sampleRate=%d", handle.SampleRate)
	}
}

func TestNewFromCredential_AliyunMissingAPIKey(t *testing.T) {
	_, err := NewFromCredential(synthesizer.TTSCredentialConfig{"provider": "aliyun"})
	if err == nil {
		t.Fatal("expected error")
	}
}
