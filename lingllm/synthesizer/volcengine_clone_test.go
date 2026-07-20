package synthesizer

import (
	"testing"
)

func TestNewAudioSynthesisEngineFromCredential_VolcengineClone(t *testing.T) {
	svc, err := NewAudioSynthesisEngineFromCredential(TTSCredentialConfig{
		"provider":    "volcengine_clone",
		"appId":       "5525383952",
		"accessToken": "test-token",
		"cluster":     "volcano_tts",
		"assetId":     "S_xiUH48cM1",
		"encoding":    "pcm",
		"sampleRate":  16000,
		"streaming":   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if svc.Provider() != ProviderVolcengineClone {
		t.Fatalf("provider = %s, want %s", svc.Provider(), ProviderVolcengineClone)
	}
	fmt := svc.Format()
	if fmt.SampleRate != 16000 {
		t.Fatalf("sample rate = %d, want 16000", fmt.SampleRate)
	}
	if c, ok := svc.(CapableSynthesisEngine); ok {
		cap := c.Capabilities()
		if !cap.StreamingTTFB {
			t.Fatal("expected streaming TTFB capability")
		}
	}
}
