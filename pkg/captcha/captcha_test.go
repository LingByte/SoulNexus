package captcha

import (
	"image"
	"strings"
	"testing"
	"time"
)

func TestNewMemoryCaptchaStore(t *testing.T) {
	store := NewMemoryCaptchaStore()
	if store == nil {
		t.Fatal("NewMemoryCaptchaStore returned nil")
	}
	if store.data == nil {
		t.Fatal("store.data is nil")
	}
}

func TestMemoryCaptchaStore_SetGet(t *testing.T) {
	store := NewMemoryCaptchaStore()
	id := "test-id"
	code := "ABCD"
	expires := time.Now().Add(5 * time.Minute)

	err := store.Set(id, code, expires)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	retrievedCode, err := store.Get(id)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrievedCode != code {
		t.Fatalf("Expected code %s, got %s", code, retrievedCode)
	}
}

func TestMemoryCaptchaStore_GetNotFound(t *testing.T) {
	store := NewMemoryCaptchaStore()
	_, err := store.Get("non-existent")
	if err == nil {
		t.Fatal("Expected error for non-existent captcha")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("Expected 'not found' error, got: %v", err)
	}
}

func TestMemoryCaptchaStore_GetExpired(t *testing.T) {
	store := NewMemoryCaptchaStore()
	id := "expired-id"
	code := "ABCD"
	expires := time.Now().Add(-1 * time.Minute) // 已过期

	err := store.Set(id, code, expires)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// 等待一下确保过期
	time.Sleep(100 * time.Millisecond)

	// cleanup可能在后台运行，所以可能返回not found或expired
	_, err = store.Get(id)
	if err == nil {
		t.Fatal("Expected error for expired captcha")
	}
	// 接受两种错误：expired 或 not found（因为cleanup可能已删除）
	if !strings.Contains(err.Error(), "expired") && !strings.Contains(err.Error(), "not found") {
		t.Fatalf("Expected 'expired' or 'not found' error, got: %v", err)
	}
}

func TestMemoryCaptchaStore_Delete(t *testing.T) {
	store := NewMemoryCaptchaStore()
	id := "test-id"
	code := "ABCD"
	expires := time.Now().Add(5 * time.Minute)

	err := store.Set(id, code, expires)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	err = store.Delete(id)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = store.Get(id)
	if err == nil {
		t.Fatal("Expected error after delete")
	}
}

func TestMemoryCaptchaStore_Verify(t *testing.T) {
	store := NewMemoryCaptchaStore()
	id := "test-id"
	code := "ABCD"
	expires := time.Now().Add(5 * time.Minute)

	err := store.Set(id, code, expires)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// 正确验证码
	valid, err := store.Verify(id, code)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !valid {
		t.Fatal("Expected valid verification")
	}

	// 验证后应该被删除
	_, err = store.Get(id)
	if err == nil {
		t.Fatal("Captcha should be deleted after verification")
	}
}

func TestMemoryCaptchaStore_VerifyCaseInsensitive(t *testing.T) {
	store := NewMemoryCaptchaStore()
	id := "test-id"
	code := "ABCD"
	expires := time.Now().Add(5 * time.Minute)

	err := store.Set(id, code, expires)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// 小写验证码应该也能通过
	valid, err := store.Verify(id, "abcd")
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !valid {
		t.Fatal("Expected valid verification with lowercase")
	}
}

func TestMemoryCaptchaStore_VerifyWrongCode(t *testing.T) {
	store := NewMemoryCaptchaStore()
	id := "test-id"
	code := "ABCD"
	expires := time.Now().Add(5 * time.Minute)

	err := store.Set(id, code, expires)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// 错误验证码
	valid, err := store.Verify(id, "WRONG")
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if valid {
		t.Fatal("Expected invalid verification")
	}

	// 验证码应该还在（因为验证失败）
	retrievedCode, err := store.Get(id)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrievedCode != code {
		t.Fatalf("Expected code %s, got %s", code, retrievedCode)
	}
}

func TestNewCaptchaManager(t *testing.T) {
	manager := NewCaptchaManager(200, 60, 4, 5*time.Minute, nil)
	if manager == nil {
		t.Fatal("NewCaptchaManager returned nil")
	}
	if manager.width != 200 {
		t.Fatalf("Expected width 200, got %d", manager.width)
	}
	if manager.height != 60 {
		t.Fatalf("Expected height 60, got %d", manager.height)
	}
	if manager.length != 4 {
		t.Fatalf("Expected length 4, got %d", manager.length)
	}
}

func TestNewCaptchaManager_WithStore(t *testing.T) {
	store := NewMemoryCaptchaStore()
	manager := NewCaptchaManager(200, 60, 4, 5*time.Minute, store)
	if manager == nil {
		t.Fatal("NewCaptchaManager returned nil")
	}
	if manager.store != store {
		t.Fatal("Store not set correctly")
	}
}

func TestCaptchaManager_Generate(t *testing.T) {
	manager := NewCaptchaManager(200, 60, 4, 5*time.Minute, nil)
	captcha, err := manager.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if captcha == nil {
		t.Fatal("Generate returned nil captcha")
	}
	if captcha.ID == "" {
		t.Fatal("Captcha ID is empty")
	}
	if captcha.Code == "" {
		t.Fatal("Captcha code is empty")
	}
	if len(captcha.Code) != 4 {
		t.Fatalf("Expected code length 4, got %d", len(captcha.Code))
	}
	if captcha.Image == "" {
		t.Fatal("Captcha image is empty")
	}
	if !strings.HasPrefix(captcha.Image, "data:image/png;base64,") {
		t.Fatal("Captcha image should be base64 encoded PNG")
	}
	if captcha.Expires.Before(time.Now()) {
		t.Fatal("Captcha expires time should be in the future")
	}
}

func TestCaptchaManager_Verify(t *testing.T) {
	store := NewMemoryCaptchaStore()
	manager := NewCaptchaManager(200, 60, 4, 5*time.Minute, store)
	captcha, err := manager.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 正确验证码
	valid, err := manager.Verify(captcha.ID, captcha.Code)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !valid {
		t.Fatal("Expected valid verification")
	}

	// 再次生成验证码用于测试错误验证码
	captcha2, err := manager.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 错误验证码
	valid, err = manager.Verify(captcha2.ID, "WRONG")
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if valid {
		t.Fatal("Expected invalid verification")
	}
}

func TestCaptchaManager_VerifyCaseInsensitive(t *testing.T) {
	manager := NewCaptchaManager(200, 60, 4, 5*time.Minute, nil)
	captcha, err := manager.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 小写验证码应该也能通过
	valid, err := manager.Verify(captcha.ID, strings.ToLower(captcha.Code))
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !valid {
		t.Fatal("Expected valid verification with lowercase")
	}
}

func TestCaptchaManager_VerifyNonExistent(t *testing.T) {
	manager := NewCaptchaManager(200, 60, 4, 5*time.Minute, nil)

	// 不存在的验证码
	valid, err := manager.Verify("non-existent", "CODE")
	if err != nil {
		// 允许返回错误
		return
	}
	if valid {
		t.Fatal("Expected invalid verification for non-existent captcha")
	}
}

func TestCaptchaManager_generateCode(t *testing.T) {
	manager := NewCaptchaManager(200, 60, 4, 5*time.Minute, nil)
	code := manager.generateCode()
	if len(code) != 4 {
		t.Fatalf("Expected code length 4, got %d", len(code))
	}

	// 生成多个验证码，确保它们是不同的
	codes := make(map[string]bool)
	for i := 0; i < 10; i++ {
		code := manager.generateCode()
		if codes[code] {
			t.Fatalf("Generated duplicate code: %s", code)
		}
		codes[code] = true
	}
}

func TestCaptchaManager_generateID(t *testing.T) {
	manager := NewCaptchaManager(200, 60, 4, 5*time.Minute, nil)
	id := manager.generateID()
	if len(id) != 32 {
		t.Fatalf("Expected ID length 32, got %d", len(id))
	}

	// 生成多个ID，确保它们是不同的
	ids := make(map[string]bool)
	for i := 0; i < 10; i++ {
		id := manager.generateID()
		if ids[id] {
			t.Fatalf("Generated duplicate ID: %s", id)
		}
		ids[id] = true
	}
}

func TestCaptchaManager_generateImage(t *testing.T) {
	manager := NewCaptchaManager(200, 60, 4, 5*time.Minute, nil)
	code := "TEST"
	img, err := manager.generateImage(code)
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

func TestCaptchaManager_imageToBase64(t *testing.T) {
	manager := NewCaptchaManager(200, 60, 4, 5*time.Minute, nil)
	code := "TEST"
	img, err := manager.generateImage(code)
	if err != nil {
		t.Fatalf("generateImage failed: %v", err)
	}

	base64, err := manager.imageToBase64(img)
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

func TestInitGlobalCaptchaManager(t *testing.T) {
	store := NewMemoryCaptchaStore()
	InitGlobalCaptchaManager(store)
	if GlobalCaptchaManager == nil {
		t.Fatal("GlobalCaptchaManager should be initialized")
	}
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
