package video

import (
	"strings"
	"testing"
)

func TestNewAliyunVideoCensor_EmptyCredentials(t *testing.T) {
	_, err := NewAliyunVideoCensor("", "secret", "")
	if err == nil {
		t.Fatal("expected error for empty accessKeyID")
	}
	if !strings.Contains(err.Error(), "cannot be empty") {
		t.Errorf("unexpected error: %v", err)
	}

	_, err = NewAliyunVideoCensor("id", "", "")
	if err == nil {
		t.Fatal("expected error for empty accessKeySecret")
	}
}

func TestNewAliyunVideoCensor_ValidArgs(t *testing.T) {
	c, err := NewAliyunVideoCensor("test-access-key-id", "test-access-key-secret", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.AccessKeyID != "test-access-key-id" {
		t.Errorf("AccessKeyID = %q, want test-access-key-id", c.AccessKeyID)
	}
	if c.AccessKeySecret != "test-access-key-secret" {
		t.Errorf("AccessKeySecret = %q, want test-access-key-secret", c.AccessKeySecret)
	}
	if c.Endpoint != AliyunGreenDefaultEndpoint {
		t.Errorf("Endpoint = %q, want %q", c.Endpoint, AliyunGreenDefaultEndpoint)
	}
	if c.Client == nil {
		t.Error("Client was not initialized")
	}
}

func TestNewAliyunVideoCensor_CustomEndpoint(t *testing.T) {
	endpoint := "green-cip.cn-beijing.aliyuncs.com"
	c, err := NewAliyunVideoCensor("id", "secret", endpoint)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Endpoint != endpoint {
		t.Errorf("Endpoint = %q, want %q", c.Endpoint, endpoint)
	}
}

func TestAliyunVideoCensor_SubmitCensorVideo(t *testing.T) {
	c, err := NewAliyunVideoCensor("id", "secret", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = c.SubmitCensorVideo("")
	if err == nil {
		t.Fatal("expected error for empty videoURL")
	}

	taskID, err := c.SubmitCensorVideo("https://example.com/video.mp4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if taskID == "" {
		t.Error("expected non-empty taskID")
	}
}

func TestAliyunVideoCensor_GetCensorResult(t *testing.T) {
	c, err := NewAliyunVideoCensor("id", "secret", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = c.GetCensorResult("")
	if err == nil {
		t.Fatal("expected error for empty taskID")
	}

	result, err := c.GetCensorResult("task-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if m["taskId"] != "task-123" {
		t.Errorf("taskId = %v, want task-123", m["taskId"])
	}
}
