package video

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/utils"
)

func TestNewQiniuVideoCensor(t *testing.T) {
	c := NewQiniuVideoCensor("test-ak", "test-sk")
	if c.AccessKey != "test-ak" {
		t.Errorf("AccessKey = %q, want test-ak", c.AccessKey)
	}
	if c.SecretKey != "test-sk" {
		t.Errorf("SecretKey = %q, want test-sk", c.SecretKey)
	}
	if c.Host != VideoCensorHost {
		t.Errorf("Host = %q, want %q", c.Host, VideoCensorHost)
	}
	if c.Client == nil {
		t.Error("HTTP Client was not initialized")
	}
}

func TestVideoCensorRequest_Marshal(t *testing.T) {
	req := VideoCensorRequest{
		Data: VideoCensorData{
			URI: "https://example.com/video.mp4",
			ID:  "video-1",
		},
		Params: VideoCensorParams{
			Scenes: []string{"pulp", "terror"},
			CutParam: &VideoCutParam{
				IntervalMsecs: 5000,
			},
		},
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	jsonStr := string(jsonData)
	if !strings.Contains(jsonStr, "https://example.com/video.mp4") {
		t.Error("URI was not serialized correctly")
	}
	if !strings.Contains(jsonStr, "interval_msecs") {
		t.Error("cut_param was not serialized correctly")
	}
}

func TestQiniuVideoCensor_SubmitCensorVideo_EmptyURL(t *testing.T) {
	c := NewQiniuVideoCensor("ak", "sk")
	_, err := c.SubmitCensorVideo("")
	if err == nil {
		t.Fatal("expected error for empty videoURL")
	}
}

func TestQiniuVideoCensor_GetCensorResult_EmptyTaskID(t *testing.T) {
	c := NewQiniuVideoCensor("ak", "sk")
	_, err := c.GetCensorResult("")
	if err == nil {
		t.Fatal("expected error for empty taskID")
	}
	_, err = c.GetCensorResultFull("")
	if err == nil {
		t.Fatal("expected error for empty jobID in GetCensorResultFull")
	}
}

func TestQiniuVideoCensor_Network(t *testing.T) {
	accessKey := utils.GetEnv("QINIU_ACCESS_KEY")
	secretKey := utils.GetEnv("QINIU_SECRET_KEY")
	if accessKey == "" || secretKey == "" {
		t.Skip("QINIU_ACCESS_KEY or QINIU_SECRET_KEY not set")
	}
	c := NewQiniuVideoCensor(accessKey, secretKey)
	_, err := c.SubmitCensorVideo("https://example.com/video.mp4")
	if err != nil {
		t.Logf("SubmitCensorVideo (network): %v", err)
	}
}
