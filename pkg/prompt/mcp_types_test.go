package prompt

import (
	"testing"
)

func TestNewTextContent(t *testing.T) {
	text := "Hello, World!"
	content := NewTextContent(text)

	if content.Type != ContentTypeText {
		t.Errorf("Expected type %s, got %s", ContentTypeText, content.Type)
	}

	if content.Text != text {
		t.Errorf("Expected text %s, got %s", text, content.Text)
	}
}

func TestTextContent_IsContent(t *testing.T) {
	content := NewTextContent("test")
	// This is a marker method, just verify it exists
	var _ Content = content
}

func TestNewImageContent(t *testing.T) {
	data := "base64encodeddata"
	mimeType := "image/png"
	content := NewImageContent(data, mimeType)

	if content.Type != ContentTypeImage {
		t.Errorf("Expected type %s, got %s", ContentTypeImage, content.Type)
	}

	if content.Data != data {
		t.Errorf("Expected data %s, got %s", data, content.Data)
	}

	if content.MimeType != mimeType {
		t.Errorf("Expected mimeType %s, got %s", mimeType, content.MimeType)
	}
}

func TestImageContent_IsContent(t *testing.T) {
	content := NewImageContent("data", "image/png")
	var _ Content = content
}

func TestNewAudioContent(t *testing.T) {
	data := "base64encodedaudiodata"
	mimeType := "audio/mpeg"
	content := NewAudioContent(data, mimeType)

	if content.Type != ContentTypeAudio {
		t.Errorf("Expected type %s, got %s", ContentTypeAudio, content.Type)
	}

	if content.Data != data {
		t.Errorf("Expected data %s, got %s", data, content.Data)
	}

	if content.MimeType != mimeType {
		t.Errorf("Expected mimeType %s, got %s", mimeType, content.MimeType)
	}
}

func TestAudioContent_IsContent(t *testing.T) {
	content := NewAudioContent("data", "audio/mpeg")
	var _ Content = content
}

func TestContentTypeConstants(t *testing.T) {
	if ContentTypeText != "text" {
		t.Errorf("Expected ContentTypeText 'text', got %s", ContentTypeText)
	}

	if ContentTypeImage != "image" {
		t.Errorf("Expected ContentTypeImage 'image', got %s", ContentTypeImage)
	}

	if ContentTypeAudio != "audio" {
		t.Errorf("Expected ContentTypeAudio 'audio', got %s", ContentTypeAudio)
	}

	if ContentTypeEmbeddedResource != "embedded_resource" {
		t.Errorf("Expected ContentTypeEmbeddedResource 'embedded_resource', got %s", ContentTypeEmbeddedResource)
	}
}

func TestAnnotated(t *testing.T) {
	content := NewTextContent("test")
	content.Annotations = &struct {
		Audience []Role  `json:"audience,omitempty"`
		Priority float64 `json:"priority,omitempty"`
	}{
		Audience: []Role{"user"},
		Priority: 1.0,
	}

	if content.Annotations == nil {
		t.Fatal("Expected annotations to be set")
	}

	if len(content.Annotations.Audience) != 1 {
		t.Errorf("Expected 1 audience, got %d", len(content.Annotations.Audience))
	}

	if content.Annotations.Priority != 1.0 {
		t.Errorf("Expected priority 1.0, got %f", content.Annotations.Priority)
	}
}

func TestCursor(t *testing.T) {
	cursor := Cursor("test-cursor")
	if string(cursor) != "test-cursor" {
		t.Errorf("Expected cursor 'test-cursor', got %s", cursor)
	}
}

func TestRole(t *testing.T) {
	role := Role("user")
	if string(role) != "user" {
		t.Errorf("Expected role 'user', got %s", role)
	}
}

func TestRequest(t *testing.T) {
	req := Request{
		Method: "test/method",
	}

	if req.Method != "test/method" {
		t.Errorf("Expected method 'test/method', got %s", req.Method)
	}
}

func TestResult(t *testing.T) {
	result := Result{
		Meta: map[string]interface{}{
			"key": "value",
		},
	}

	if result.Meta["key"] != "value" {
		t.Errorf("Expected meta key 'value', got %v", result.Meta["key"])
	}
}

func TestPaginatedResult(t *testing.T) {
	result := PaginatedResult{
		Result: Result{
			Meta: map[string]interface{}{"key": "value"},
		},
		NextCursor: Cursor("next"),
	}

	if result.Meta["key"] != "value" {
		t.Errorf("Expected meta key 'value', got %v", result.Meta["key"])
	}

	if result.NextCursor != Cursor("next") {
		t.Errorf("Expected next cursor 'next', got %s", result.NextCursor)
	}
}
