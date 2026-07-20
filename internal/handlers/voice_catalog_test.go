package handlers

import "testing"

func TestListVoiceCatalogQCloudTTS(t *testing.T) {
	out, err := listVoiceCatalog("qcloud", "tts")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(out.Voices) == 0 {
		t.Fatal("expected qcloud voices")
	}
	if out.VoiceField != "voiceType" {
		t.Fatalf("voiceField=%q", out.VoiceField)
	}
}

func TestListVoiceCatalogAliyunOmniRealtime(t *testing.T) {
	out, err := listVoiceCatalog("qwen_omni", "realtime")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if out.Provider != "aliyun_omni" {
		t.Fatalf("provider=%q", out.Provider)
	}
	foundCherry := false
	for _, v := range out.Voices {
		if v.ID == "Cherry" {
			foundCherry = true
			break
		}
	}
	if !foundCherry {
		t.Fatal("missing Cherry voice")
	}
}
