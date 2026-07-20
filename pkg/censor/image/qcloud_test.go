package image

import (
	"strings"
	"testing"
)

func TestNewQCloudImageCensor_EmptyCredentials(t *testing.T) {
	_, err := NewQCloudImageCensor("", "secret", "")
	if err == nil {
		t.Fatal("expected error for empty secretID")
	}
	if !strings.Contains(err.Error(), "cannot be empty") {
		t.Errorf("unexpected error: %v", err)
	}

	_, err = NewQCloudImageCensor("id", "", "")
	if err == nil {
		t.Fatal("expected error for empty secretKey")
	}
}

func TestNewQCloudImageCensor_ValidArgs(t *testing.T) {
	c, err := NewQCloudImageCensor("test-secret-id", "test-secret-key", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.SecretID != "test-secret-id" {
		t.Errorf("SecretID = %q, want test-secret-id", c.SecretID)
	}
	if c.SecretKey != "test-secret-key" {
		t.Errorf("SecretKey = %q, want test-secret-key", c.SecretKey)
	}
	if c.Region != QCloudIMSDefaultRegion {
		t.Errorf("Region = %q, want %q", c.Region, QCloudIMSDefaultRegion)
	}
	if c.Client == nil {
		t.Error("Client was not initialized")
	}
}

func TestNewQCloudImageCensor_CustomRegion(t *testing.T) {
	region := "ap-shanghai"
	c, err := NewQCloudImageCensor("id", "secret", region)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Region != region {
		t.Errorf("Region = %q, want %q", c.Region, region)
	}
}

func TestQCloudImageCensor_CensorImage(t *testing.T) {
	c, err := NewQCloudImageCensor("id", "secret", "")
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
}
