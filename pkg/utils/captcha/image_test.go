package captcha

import (
	"image"
	"strings"
	"testing"
	"time"
)

func TestNewImageCaptcha(t *testing.T) {
	captcha := NewImageCaptcha(200, 60, 4, 5*time.Minute, nil)
	if captcha == nil {
		t.Fatal("NewImageCaptcha returned nil")
	}
	if captcha.width != 200 {
		t.Fatalf("Expected width 200, got %d", captcha.width)
	}
	if captcha.height != 60 {
		t.Fatalf("Expected height 60, got %d", captcha.height)
	}
	if captcha.length != 4 {
		t.Fatalf("Expected length 4, got %d", captcha.length)
	}
}

func TestImageCaptcha_Generate(t *testing.T) {
	captcha := NewImageCaptcha(200, 60, 4, 5*time.Minute, nil)
	result, err := captcha.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if result == nil {
		t.Fatal("Generate returned nil result")
	}
	if result.ID == "" {
		t.Fatal("Result ID is empty")
	}
	if result.Type != TypeImage {
		t.Fatalf("Expected type %s, got %s", TypeImage, result.Type)
	}
	if result.Data == nil {
		t.Fatal("Result data is nil")
	}
	imageData, ok := result.Data["image"].(string)
	if !ok || imageData == "" {
		t.Fatal("Result image is empty")
	}
	if !strings.HasPrefix(imageData, "data:image/png;base64,") {
		t.Fatal("Result image should be base64 encoded PNG")
	}
	code, ok := result.Data["code"].(string)
	if !ok || len(code) != 4 {
		t.Fatalf("Expected code length 4, got %d", len(code))
	}
	if result.Expires.Before(time.Now()) {
		t.Fatal("Result expires time should be in the future")
	}
}

func TestImageCaptcha_Verify(t *testing.T) {
	captcha := NewImageCaptcha(200, 60, 4, 5*time.Minute, nil)
	result, err := captcha.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	code, ok := result.Data["code"].(string)
	if !ok {
		t.Fatal("Code not found in result data")
	}

	// 正确验证码
	valid, err := captcha.Verify(result.ID, code)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !valid {
		t.Fatal("Expected valid verification")
	}

	// 再次生成验证码用于测试错误验证码
	result2, err := captcha.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 错误验证码
	valid, err = captcha.Verify(result2.ID, "WRONG")
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if valid {
		t.Fatal("Expected invalid verification")
	}
}

func TestImageCaptcha_VerifyCaseInsensitive(t *testing.T) {
	captcha := NewImageCaptcha(200, 60, 4, 5*time.Minute, nil)
	result, err := captcha.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	code, ok := result.Data["code"].(string)
	if !ok {
		t.Fatal("Code not found in result data")
	}

	// 小写验证码应该也能通过
	valid, err := captcha.Verify(result.ID, strings.ToLower(code))
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !valid {
		t.Fatal("Expected valid verification with lowercase")
	}
}

func TestImageCaptcha_generateImage(t *testing.T) {
	captcha := NewImageCaptcha(200, 60, 4, 5*time.Minute, nil)
	code := "TEST"
	img, err := captcha.generateImage(code)
	if err != nil {
		t.Fatalf("generateImage failed: %v", err)
	}
	if img == nil {
		t.Fatal("generateImage returned nil")
	}
	bounds := img.Bounds()
	if bounds.Dx() != 200 {
		t.Fatalf("Expected width 200, got %d", bounds.Dx())
	}
	if bounds.Dy() != 60 {
		t.Fatalf("Expected height 60, got %d", bounds.Dy())
	}
}

func TestImageCaptcha_imageToBase64(t *testing.T) {
	captcha := NewImageCaptcha(200, 60, 4, 5*time.Minute, nil)
	code := "TEST"
	img, err := captcha.generateImage(code)
	if err != nil {
		t.Fatalf("generateImage failed: %v", err)
	}

	base64, err := captcha.imageToBase64(img)
	if err != nil {
		t.Fatalf("imageToBase64 failed: %v", err)
	}
	if base64 == "" {
		t.Fatal("imageToBase64 returned empty string")
	}
	if !strings.HasPrefix(base64, "data:image/png;base64,") {
		t.Fatal("imageToBase64 should return base64 encoded PNG")
	}
}

func TestDrawLine(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	drawLine(img, 0, 0, 50, 50, image.Black)
	// 如果函数执行没有panic，就认为测试通过
}

func TestAbs(t *testing.T) {
	if abs(-5) != 5 {
		t.Fatalf("Expected abs(-5) = 5, got %d", abs(-5))
	}
	if abs(5) != 5 {
		t.Fatalf("Expected abs(5) = 5, got %d", abs(5))
	}
	if abs(0) != 0 {
		t.Fatalf("Expected abs(0) = 0, got %d", abs(0))
	}
}
