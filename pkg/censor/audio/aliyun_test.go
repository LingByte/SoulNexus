package audio

import (
	"strings"
	"testing"
)

func TestNewAliyunAudioCensor_EmptyCredentials(t *testing.T) {
	_, err := NewAliyunAudioCensor("", "secret", "")
	if err == nil {
		t.Fatal("expected error for empty accessKeyID")
	}
	_, err = NewAliyunAudioCensor("id", "", "")
	if err == nil {
		t.Fatal("expected error for empty accessKeySecret")
	}
}

func TestNewAliyunAudioCensor_ValidArgs(t *testing.T) {
	c, err := NewAliyunAudioCensor("test-access-key-id", "test-access-key-secret", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Client == nil {
		t.Fatal("Client was not initialized")
	}
	if c.Endpoint != AliyunGreenDefaultEndpoint {
		t.Errorf("Endpoint = %q", c.Endpoint)
	}
	if c.Service != AliyunAudioService {
		t.Errorf("Service = %q", c.Service)
	}
}

func TestAliyunAudioCensor_SubmitEmptyURL(t *testing.T) {
	c, err := NewAliyunAudioCensor("id", "secret", "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.SubmitCensorAudio("")
	if err == nil || !strings.Contains(err.Error(), "cannot be empty") {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestAliyunAudioCensor_PollEmptyTaskID(t *testing.T) {
	c, err := NewAliyunAudioCensor("id", "secret", "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.PollCensorAudio("")
	if err == nil {
		t.Fatal("expected error")
	}
}
