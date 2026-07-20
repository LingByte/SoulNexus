package video

import (
	"strings"
	"testing"
)

func TestNewQCloudVideoCensor_EmptyCredentials(t *testing.T) {
	_, err := NewQCloudVideoCensor("", "secret", "")
	if err == nil {
		t.Fatal("expected error for empty secretID")
	}
	if !strings.Contains(err.Error(), "cannot be empty") {
		t.Errorf("unexpected error: %v", err)
	}

	_, err = NewQCloudVideoCensor("id", "", "")
	if err == nil {
		t.Fatal("expected error for empty secretKey")
	}
}

func TestNewQCloudVideoCensor_ValidArgs(t *testing.T) {
	c, err := NewQCloudVideoCensor("test-secret-id", "test-secret-key", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.SecretID != "test-secret-id" {
		t.Errorf("SecretID = %q, want test-secret-id", c.SecretID)
	}
	if c.SecretKey != "test-secret-key" {
		t.Errorf("SecretKey = %q, want test-secret-key", c.SecretKey)
	}
	if c.Region != QCloudVODDefaultRegion {
		t.Errorf("Region = %q, want %q", c.Region, QCloudVODDefaultRegion)
	}
	if c.Client == nil {
		t.Error("Client was not initialized")
	}
}

func TestNewQCloudVideoCensor_CustomRegion(t *testing.T) {
	region := "ap-shanghai"
	c, err := NewQCloudVideoCensor("id", "secret", region)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Region != region {
		t.Errorf("Region = %q, want %q", c.Region, region)
	}
}

func TestQCloudVideoCensor_SubmitCensorVideo(t *testing.T) {
	c, err := NewQCloudVideoCensor("id", "secret", "")
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

func TestQCloudVideoCensor_SubmitCensorVideoByFileID_Empty(t *testing.T) {
	c, err := NewQCloudVideoCensor("id", "secret", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = c.SubmitCensorVideoByFileID("")
	if err == nil {
		t.Fatal("expected error for empty fileID")
	}
}

func TestQCloudVideoCensor_GetCensorResult_EmptyTaskID(t *testing.T) {
	c, err := NewQCloudVideoCensor("id", "secret", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = c.GetCensorResult("")
	if err == nil {
		t.Fatal("expected error for empty taskID")
	}
}
