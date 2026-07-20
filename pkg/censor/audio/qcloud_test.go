package audio

import (
	"strings"
	"testing"
)

func TestNewQCloudAudioCensor_EmptyCredentials(t *testing.T) {
	_, err := NewQCloudAudioCensor("", "secret", "")
	if err == nil {
		t.Fatal("expected error for empty secretID")
	}
	_, err = NewQCloudAudioCensor("id", "", "")
	if err == nil {
		t.Fatal("expected error for empty secretKey")
	}
}

func TestNewQCloudAudioCensor_ValidArgs(t *testing.T) {
	c, err := NewQCloudAudioCensor("test-secret-id", "test-secret-key", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Client == nil {
		t.Fatal("Client was not initialized")
	}
	if c.Region != QCloudAMSDefaultRegion {
		t.Errorf("Region = %q", c.Region)
	}
	if c.BizType != "default" {
		t.Errorf("BizType = %q", c.BizType)
	}
}

func TestQCloudAudioCensor_SubmitEmptyURL(t *testing.T) {
	c, err := NewQCloudAudioCensor("id", "secret", "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.SubmitCensorAudio("")
	if err == nil || !strings.Contains(err.Error(), "cannot be empty") {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestQCloudAudioCensor_PollEmptyTaskID(t *testing.T) {
	c, err := NewQCloudAudioCensor("id", "secret", "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.PollCensorAudio("")
	if err == nil {
		t.Fatal("expected error")
	}
}
