package image

import (
	"strings"
	"testing"
)

func TestNewAliyunImageCensor_EmptyCredentials(t *testing.T) {
	_, err := NewAliyunImageCensor("", "secret", "")
	if err == nil {
		t.Fatal("expected error for empty accessKeyID")
	}
	if !strings.Contains(err.Error(), "cannot be empty") {
		t.Errorf("unexpected error: %v", err)
	}

	_, err = NewAliyunImageCensor("id", "", "")
	if err == nil {
		t.Fatal("expected error for empty accessKeySecret")
	}
}

func TestNewAliyunImageCensor_ValidArgs(t *testing.T) {
	c, err := NewAliyunImageCensor("test-access-key-id", "test-access-key-secret", "")
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

func TestNewAliyunImageCensor_CustomEndpoint(t *testing.T) {
	endpoint := "green-cip.cn-beijing.aliyuncs.com"
	c, err := NewAliyunImageCensor("id", "secret", endpoint)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Endpoint != endpoint {
		t.Errorf("Endpoint = %q, want %q", c.Endpoint, endpoint)
	}
}

func TestAliyunImageCensor_CensorImage(t *testing.T) {
	c, err := NewAliyunImageCensor("id", "secret", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = c.CensorImage("")
	if err == nil {
		t.Fatal("expected error for empty imageURL")
	}

	result, err := c.CensorImage("https://example.com/image.jpg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if m["imageUrl"] != "https://example.com/image.jpg" {
		t.Errorf("imageUrl = %v, want https://example.com/image.jpg", m["imageUrl"])
	}
	if m["result"] != "pass" {
		t.Errorf("result = %v, want pass", m["result"])
	}
}
