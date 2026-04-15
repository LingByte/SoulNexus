package captcha

import (
	"strings"
	"testing"
	"time"
)

func TestNewClickCaptcha(t *testing.T) {
	captcha := NewClickCaptcha(300, 200, 3, 10, 5*time.Minute, nil)
	if captcha == nil {
		t.Fatal("NewClickCaptcha returned nil")
	}
	if captcha.width != 300 {
		t.Fatalf("Expected width 300, got %d", captcha.width)
	}
	if captcha.height != 200 {
		t.Fatalf("Expected height 200, got %d", captcha.height)
	}
	if captcha.count != 3 {
		t.Fatalf("Expected count 3, got %d", captcha.count)
	}
}

func TestClickCaptcha_Generate(t *testing.T) {
	captcha := NewClickCaptcha(300, 200, 3, 10, 5*time.Minute, nil)
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
	if result.Type != TypeClick {
		t.Fatalf("Expected type %s, got %s", TypeClick, result.Type)
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
	positions, ok := result.Data["positions"].([]Point)
	if !ok || len(positions) != 3 {
		t.Fatalf("Expected 3 positions, got %d", len(positions))
	}
	if result.Expires.Before(time.Now()) {
		t.Fatal("Result expires time should be in the future")
	}
}

func TestClickCaptcha_Verify(t *testing.T) {
	captcha := NewClickCaptcha(300, 200, 3, 10, 5*time.Minute, nil)
	result, err := captcha.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	positions, ok := result.Data["positions"].([]Point)
	if !ok {
		t.Fatal("Positions not found in result data")
	}

	// 正确位置
	valid, err := captcha.Verify(result.ID, positions)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !valid {
		t.Fatal("Expected valid verification")
	}

	// 再次生成验证码用于测试错误位置
	result2, err := captcha.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 错误位置
	wrongPositions := []Point{{X: 0, Y: 0}, {X: 10, Y: 10}, {X: 20, Y: 20}}
	valid, err = captcha.Verify(result2.ID, wrongPositions)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if valid {
		t.Fatal("Expected invalid verification")
	}
}

func TestClickCaptcha_VerifyWithTolerance(t *testing.T) {
	captcha := NewClickCaptcha(300, 200, 3, 10, 5*time.Minute, nil)
	result, err := captcha.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	positions, ok := result.Data["positions"].([]Point)
	if !ok {
		t.Fatal("Positions not found in result data")
	}

	// 在容差范围内
	tolerancePositions := make([]Point, len(positions))
	for i, pos := range positions {
		tolerancePositions[i] = Point{X: pos.X + 5, Y: pos.Y + 5}
	}

	valid, err := captcha.Verify(result.ID, tolerancePositions)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !valid {
		t.Fatal("Expected valid verification within tolerance")
	}
}

func TestClickCaptcha_VerifyWrongCount(t *testing.T) {
	captcha := NewClickCaptcha(300, 200, 3, 10, 5*time.Minute, nil)
	result, err := captcha.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 错误的数量
	wrongPositions := []Point{{X: 0, Y: 0}, {X: 10, Y: 10}}
	valid, err := captcha.Verify(result.ID, wrongPositions)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if valid {
		t.Fatal("Expected invalid verification for wrong count")
	}
}
