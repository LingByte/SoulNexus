package text

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/utils"
)

func TestTextCensorRequest_Marshal(t *testing.T) {
	req := TextCensorRequest{
		Data: TextCensorData{
			Text: "Qiniu text moderation example",
		},
		Params: TextCensorParams{
			Scenes: []string{"antispam"},
		},
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	// Verify JSON contains necessary fields
	jsonStr := string(jsonData)
	if !strings.Contains(jsonStr, "Qiniu text moderation example") {
		t.Error("Text was not serialized correctly")
	}
	if !strings.Contains(jsonStr, "antispam") {
		t.Error("Scenes was not serialized correctly")
	}
}

func TestNewQiniuTextCensor(t *testing.T) {
	accessKey := utils.GetEnv("QINIU_CENSOR_ACCESS_KEY")
	secretKey := utils.GetEnv("QINIU_CENSOR_SECRET_KEY")
	if accessKey == "" || secretKey == "" {
		t.Skip("QINIU_CENSOR_ACCESS_KEY or QINIU_CENSOR_SECRET_KEY not set")
	}
	client, err := NewQiniuTextCensor()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	if client.AccessKey != accessKey {
		t.Error("AccessKey was set incorrectly")
	}
	if client.SecretKey != secretKey {
		t.Error("SecretKey was set incorrectly")
	}
	if client.Host != TextCensorHost {
		t.Errorf("Host was set incorrectly: expected %s, got %s", TextCensorHost, client.Host)
	}
	if client.Client == nil {
		t.Error("HTTP Client was not initialized")
	}
}
