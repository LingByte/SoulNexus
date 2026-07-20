package recognizer

import (
	"testing"

	tcAsr "github.com/tencentcloud/tencentcloud-speech-sdk-go/asr"
	"github.com/tencentcloud/tencentcloud-speech-sdk-go/common"
)

func TestQCloudASRPackage(t *testing.T) {
	t.Run("qcloud_asr_available", func(t *testing.T) {
		if true {
			t.Log("QCloud ASR package is available")
		}
	})
}

func TestQCloudEffectiveVadSilenceTime(t *testing.T) {
	cases := []struct {
		in   int
		want int
	}{
		{0, defaultQCloudVadSilenceMs},
		{-1, defaultQCloudVadSilenceMs},
		{400, 400},
		{100, 240},
		{3000, 2000},
		{1000, 1000},
	}
	for _, c := range cases {
		got := QCloudASROption{VadSilenceTime: c.in}.effectiveVadSilenceTime()
		if got != c.want {
			t.Errorf("VadSilenceTime=%d → %d, want %d", c.in, got, c.want)
		}
	}
}

func TestApplyQCloudRecognizerParams(t *testing.T) {
	cred := common.NewCredential("id", "key")
	r := tcAsr.NewSpeechRecognizer("app", cred, "16k_zh", nil)
	applyQCloudRecognizerParams(r, QCloudASROption{VadSilenceTime: 400})
	if r.NeedVad != 1 {
		t.Errorf("NeedVad=%d, want 1", r.NeedVad)
	}
	if r.VadSilenceTime != 400 {
		t.Errorf("VadSilenceTime=%d, want 400", r.VadSilenceTime)
	}
}

func TestBuildQCloudConfig_VadSilenceTime(t *testing.T) {
	opt, err := buildQCloudConfig(map[string]interface{}{
		"appId":          "1",
		"secretId":       "s",
		"secretKey":      "k",
		"vadSilenceTime": 600,
	})
	if err != nil {
		t.Fatal(err)
	}
	if opt.VadSilenceTime != 600 {
		t.Errorf("VadSilenceTime=%d, want 600", opt.VadSilenceTime)
	}
}
