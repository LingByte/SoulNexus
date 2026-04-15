package captcha

import (
	"strings"
	"testing"
	"time"
)

func TestNewMemoryStore(t *testing.T) {
	store := NewMemoryStore()
	if store == nil {
		t.Fatal("NewMemoryStore returned nil")
	}
	if store.data == nil {
		t.Fatal("store.data is nil")
	}
}

func TestMemoryStore_SetGet(t *testing.T) {
	store := NewMemoryStore()
	id := "test-id"
	code := "ABCD"
	expires := time.Now().Add(5 * time.Minute)

	err := store.Set(id, code, expires)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	retrieved, err := store.Get(id)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	retrievedCode, ok := retrieved.(string)
	if !ok {
		t.Fatal("Retrieved data is not string")
	}
	if retrievedCode != code {
		t.Fatalf("Expected code %s, got %s", code, retrievedCode)
	}
}

func TestMemoryStore_GetNotFound(t *testing.T) {
	store := NewMemoryStore()
	_, err := store.Get("non-existent")
	if err == nil {
		t.Fatal("Expected error for non-existent captcha")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("Expected 'not found' error, got: %v", err)
	}
}

func TestMemoryStore_GetExpired(t *testing.T) {
	store := NewMemoryStore()
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

func TestMemoryStore_Delete(t *testing.T) {
	store := NewMemoryStore()
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

func TestMemoryStore_VerifyWithFunc(t *testing.T) {
	store := NewMemoryStore()
	id := "test-id"
	code := "ABCD"
	expires := time.Now().Add(5 * time.Minute)

	err := store.Set(id, code, expires)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// 正确验证码
	valid, err := store.VerifyWithFunc(id, code, func(stored, input interface{}) bool {
		storedStr, ok1 := stored.(string)
		inputStr, ok2 := input.(string)
		if !ok1 || !ok2 {
			return false
		}
		return strings.ToLower(storedStr) == strings.ToLower(inputStr)
	})
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

func TestMemoryStore_VerifyWithFuncCaseInsensitive(t *testing.T) {
	store := NewMemoryStore()
	id := "test-id"
	code := "ABCD"
	expires := time.Now().Add(5 * time.Minute)

	err := store.Set(id, code, expires)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// 小写验证码应该也能通过
	valid, err := store.VerifyWithFunc(id, "abcd", func(stored, input interface{}) bool {
		storedStr, ok1 := stored.(string)
		inputStr, ok2 := input.(string)
		if !ok1 || !ok2 {
			return false
		}
		return strings.ToLower(storedStr) == strings.ToLower(inputStr)
	})
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !valid {
		t.Fatal("Expected valid verification with lowercase")
	}
}

func TestMemoryStore_VerifyWithFuncWrongCode(t *testing.T) {
	store := NewMemoryStore()
	id := "test-id"
	code := "ABCD"
	expires := time.Now().Add(5 * time.Minute)

	err := store.Set(id, code, expires)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// 错误验证码
	valid, err := store.VerifyWithFunc(id, "WRONG", func(stored, input interface{}) bool {
		storedStr, ok1 := stored.(string)
		inputStr, ok2 := input.(string)
		if !ok1 || !ok2 {
			return false
		}
		return strings.ToLower(storedStr) == strings.ToLower(inputStr)
	})
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if valid {
		t.Fatal("Expected invalid verification")
	}

	// 验证码应该还在（因为验证失败）
	retrieved, err := store.Get(id)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	retrievedCode, ok := retrieved.(string)
	if !ok {
		t.Fatal("Retrieved data is not string")
	}
	if retrievedCode != code {
		t.Fatalf("Expected code %s, got %s", code, retrievedCode)
	}
}
