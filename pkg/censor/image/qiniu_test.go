package image

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/utils"
)

func TestNewQiniuImageCensor(t *testing.T) {
	c := NewQiniuImageCensor("test-ak", "test-sk")
	if c.AccessKey != "test-ak" {
		t.Errorf("AccessKey = %q, want test-ak", c.AccessKey)
	}
	if c.SecretKey != "test-sk" {
		t.Errorf("SecretKey = %q, want test-sk", c.SecretKey)
	}
	if c.Host != ImageCensorHost {
		t.Errorf("Host = %q, want %q", c.Host, ImageCensorHost)
	}
	if c.Client == nil {
		t.Error("HTTP Client was not initialized")
	}
}

func TestImageCensorRequest_Marshal(t *testing.T) {
	req := ImageCensorRequest{
		Data: ImageCensorData{
			URI: "https://example.com/image.jpg",
		},
		Params: ImageCensorParams{
			Scenes: []string{"pulp", "terror"},
		},
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	jsonStr := string(jsonData)
	if !strings.Contains(jsonStr, "https://example.com/image.jpg") {
		t.Error("URI was not serialized correctly")
	}
	if !strings.Contains(jsonStr, "pulp") {
		t.Error("scenes was not serialized correctly")
	}
}

func TestQiniuImageCensor_CensorImage_EmptyURL(t *testing.T) {
	c := NewQiniuImageCensor("ak", "sk")
	_, err := c.CensorImage("")
	if err == nil {
		t.Fatal("expected error for empty imageURL")
	}
}

func TestQiniuImageCensor_Network(t *testing.T) {
	accessKey := utils.GetEnv("QINIU_ACCESS_KEY")
	secretKey := utils.GetEnv("QINIU_SECRET_KEY")
	if accessKey == "" || secretKey == "" {
		t.Skip("QINIU_ACCESS_KEY or QINIU_SECRET_KEY not set")
	}
	c := NewQiniuImageCensor(accessKey, secretKey)
	_, err := c.CensorImage("https://example.com/image.jpg")
	if err != nil {
		t.Logf("CensorImage (network): %v", err)
	}
}
