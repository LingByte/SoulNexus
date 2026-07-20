package text

import (
	"fmt"
	"strings"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/utils"
)

func TestGetTextCensor_UnknownKind(t *testing.T) {
	_, err := GetTextCensor("unknown")
	if err == nil {
		t.Fatal("expected error for unknown kind")
	}
	if !strings.Contains(err.Error(), "unknown text censor kind") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetTextCensor_EmptyKindDefaultsToQiniu(t *testing.T) {
	c, err := GetTextCensor("")
	if err != nil {
		// NewQiniuTextCensor may fail without env credentials
		if strings.Contains(err.Error(), "QINIU") || strings.Contains(err.Error(), "access") {
			t.Skipf("qiniu credentials not configured: %v", err)
		}
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil censor")
	}
}

func TestBuildCensorMsg(t *testing.T) {
	tests := []struct {
		label string
		want  string
	}{
		{LabelNormal, MsgNormal},
		{LabelSpam, MsgSpam},
		{LabelAd, MsgAd},
		{LabelPolitics, MsgPolitics},
		{LabelTerrorism, MsgTerrorism},
		{LabelAbuse, MsgAbuse},
		{LabelPorn, MsgPorn},
		{LabelFlood, MsgFlood},
		{LabelContraband, MsgContraband},
		{LabelMeaningless, MsgMeaningless},
		{"", MsgNormal},
		{"custom-label", fmt.Sprintf(MsgUnknownLabel, "custom-label")},
	}
	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			got := buildCensorMsg(tt.label)
			if got != tt.want {
				t.Errorf("buildCensorMsg(%q) = %q, want %q", tt.label, got, tt.want)
			}
		})
	}
}

func TestQiniuGetTextCensor(t *testing.T) {
	accessKey := utils.GetEnv("QINIU_CENSOR_ACCESS_KEY")
	secretKey := utils.GetEnv("QINIU_CENSOR_SECRET_KEY")
	if accessKey == "" || secretKey == "" {
		t.Skip("QINIU_CENSOR_ACCESS_KEY or QINIU_CENSOR_SECRET_KEY not set")
	}
	textCensor, err := GetTextCensor(KindQiNiu)
	if err != nil {
		t.Fatal(err)
	}
	result, err := textCensor.CensorText("hello world")
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Logf("Suggestion: %s, Label: %s, Score: %.4f, Msg: %s", result.Suggestion, result.Label, result.Score, result.Msg)
	}
}

func TestQCloudGetTextCensor(t *testing.T) {
	secretID := utils.GetEnv("QCLOUD_SECRET_ID")
	secretKey := utils.GetEnv("QCLOUD_SECRET_KEY")
	if secretID == "" || secretKey == "" {
		t.Skipf("not found QCLOUD_SECRET_ID or QCLOUD_SECRET_KEY")
	}
	textCensor, err := GetTextCensor(KindQCloud)
	if err != nil {
		t.Fatal(err)
	}
	result, err := textCensor.CensorText("hello world")
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Logf("Suggestion: %s, Label: %s, Score: %.4f, Msg: %s", result.Suggestion, result.Label, result.Score, result.Msg)
	}
}

func TestAliyunGetTextCensor(t *testing.T) {
	accessKeyID := utils.GetEnv("ALIYUN_ACCESS_KEY_ID")
	accessKeySecret := utils.GetEnv("ALIYUN_ACCESS_KEY_SECRET")
	if accessKeyID == "" || accessKeySecret == "" {
		t.Skipf("not found ALIYUN_ACCESS_KEY_ID or ALIYUN_ACCESS_KEY_SECRET")
	}
	textCensor, err := GetTextCensor(KindAliyun)
	if err != nil {
		t.Fatal(err)
	}
	result, err := textCensor.CensorText("hello world")
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Logf("Suggestion: %s, Label: %s, Score: %.4f, Msg: %s", result.Suggestion, result.Label, result.Score, result.Msg)
	}
}
