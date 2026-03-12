package utils

import (
	"testing"
	"time"
)

func TestNewMemoryLock(t *testing.T) {
	lock := NewMemoryLock()
	if lock == nil {
		t.Fatal("NewMemoryLock returned nil")
	}
	if lock.locks == nil {
		t.Fatal("locks map is nil")
	}
}

func TestMemoryLock_Lock(t *testing.T) {
	lock := NewMemoryLock()
	key := "test-key"
	ttl := 5 * time.Minute

	success, err := lock.Lock(key, ttl)
	if err != nil {
		t.Fatalf("Lock failed: %v", err)
	}
	if !success {
		t.Fatal("Lock should succeed")
	}

	// 验证锁已设置
	if _, exists := lock.locks[key]; !exists {
		t.Fatal("Lock should be set")
	}
}

func TestMemoryLock_Lock_AlreadyExists(t *testing.T) {
	lock := NewMemoryLock()
	key := "test-key"
	ttl := 5 * time.Minute

	// 第一次获取锁
	success, err := lock.Lock(key, ttl)
	if err != nil {
		t.Fatalf("Lock failed: %v", err)
	}
	if !success {
		t.Fatal("Lock should succeed")
	}

	// 尝试再次获取锁（应该失败）
	success, err = lock.Lock(key, ttl)
	if err == nil {
		t.Fatal("Expected error when lock already exists")
	}
	if success {
		t.Fatal("Lock should fail when already exists")
	}
}

func TestMemoryLock_Lock_Expired(t *testing.T) {
	lock := NewMemoryLock()
	key := "test-key"
	ttl := 100 * time.Millisecond

	// 第一次获取锁
	success, err := lock.Lock(key, ttl)
	if err != nil {
		t.Fatalf("Lock failed: %v", err)
	}
	if !success {
		t.Fatal("Lock should succeed")
	}

	// 等待锁过期
	time.Sleep(150 * time.Millisecond)

	// 应该可以再次获取锁
	success, err = lock.Lock(key, ttl)
	if err != nil {
		t.Fatalf("Lock failed after expiration: %v", err)
	}
	if !success {
		t.Fatal("Lock should succeed after expiration")
	}
}

func TestMemoryLock_Unlock(t *testing.T) {
	lock := NewMemoryLock()
	key := "test-key"
	ttl := 5 * time.Minute

	// 获取锁
	success, err := lock.Lock(key, ttl)
	if err != nil {
		t.Fatalf("Lock failed: %v", err)
	}
	if !success {
		t.Fatal("Lock should succeed")
	}

	// 释放锁
	err = lock.Unlock(key)
	if err != nil {
		t.Fatalf("Unlock failed: %v", err)
	}

	// 验证锁已删除
	if _, exists := lock.locks[key]; exists {
		t.Fatal("Lock should be removed after unlock")
	}
}

func TestMemoryLock_Unlock_NotExists(t *testing.T) {
	lock := NewMemoryLock()
	key := "non-existent-key"

	// 释放不存在的锁应该不报错
	err := lock.Unlock(key)
	if err != nil {
		t.Fatalf("Unlock should not fail for non-existent key: %v", err)
	}
}

func TestMemoryLock_TryLock(t *testing.T) {
	lock := NewMemoryLock()
	key := "test-key"
	ttl := 5 * time.Minute

	success, err := lock.TryLock(key, ttl)
	if err != nil {
		t.Fatalf("TryLock failed: %v", err)
	}
	if !success {
		t.Fatal("TryLock should succeed")
	}
}

func TestInitGlobalDistributedLock(t *testing.T) {
	// 初始化缓存
	InitGlobalCache(1000, 1*time.Hour)
	defer func() {
		GlobalCache = nil
		GlobalDistributedLock = nil
	}()

	InitGlobalDistributedLock()
	if GlobalDistributedLock == nil {
		t.Fatal("GlobalDistributedLock should be initialized")
	}
}

func TestInitGlobalDistributedLock_NoCache(t *testing.T) {
	// 确保缓存为nil
	GlobalCache = nil
	defer func() {
		GlobalDistributedLock = nil
	}()

	InitGlobalDistributedLock()
	if GlobalDistributedLock == nil {
		t.Fatal("GlobalDistributedLock should be initialized even without cache")
	}
}

func TestAcquireRegistrationLock(t *testing.T) {
	InitGlobalCache(1000, 1*time.Hour)
	InitGlobalDistributedLock()
	defer func() {
		GlobalCache = nil
		GlobalDistributedLock = nil
	}()

	email := "test@example.com"
	success, err := AcquireRegistrationLock(email)
	if err != nil {
		t.Fatalf("AcquireRegistrationLock failed: %v", err)
	}
	if !success {
		t.Fatal("AcquireRegistrationLock should succeed")
	}
}

func TestAcquireRegistrationLock_AlreadyLocked(t *testing.T) {
	InitGlobalCache(1000, 1*time.Hour)
	InitGlobalDistributedLock()
	defer func() {
		GlobalCache = nil
		GlobalDistributedLock = nil
	}()

	email := "test@example.com"

	// 第一次获取锁
	success, err := AcquireRegistrationLock(email)
	if err != nil {
		t.Fatalf("AcquireRegistrationLock failed: %v", err)
	}
	if !success {
		t.Fatal("AcquireRegistrationLock should succeed")
	}

	// 尝试再次获取锁（应该失败）
	success, err = AcquireRegistrationLock(email)
	if err == nil {
		t.Fatal("Expected error when lock already exists")
	}
	if success {
		t.Fatal("AcquireRegistrationLock should fail when already locked")
	}
}

func TestReleaseRegistrationLock(t *testing.T) {
	InitGlobalCache(1000, 1*time.Hour)
	InitGlobalDistributedLock()
	defer func() {
		GlobalCache = nil
		GlobalDistributedLock = nil
	}()

	email := "test@example.com"

	// 获取锁
	success, err := AcquireRegistrationLock(email)
	if err != nil {
		t.Fatalf("AcquireRegistrationLock failed: %v", err)
	}
	if !success {
		t.Fatal("AcquireRegistrationLock should succeed")
	}

	// 释放锁
	err = ReleaseRegistrationLock(email)
	if err != nil {
		t.Fatalf("ReleaseRegistrationLock failed: %v", err)
	}

	// 应该可以再次获取锁
	success, err = AcquireRegistrationLock(email)
	if err != nil {
		t.Fatalf("AcquireRegistrationLock failed after release: %v", err)
	}
	if !success {
		t.Fatal("AcquireRegistrationLock should succeed after release")
	}
}

func TestAcquireRegistrationLock_NoDistributedLock(t *testing.T) {
	InitGlobalCache(1000, 1*time.Hour)
	GlobalDistributedLock = nil // 清除分布式锁
	defer func() {
		GlobalCache = nil
		GlobalDistributedLock = nil
	}()

	email := "test@example.com"
	success, err := AcquireRegistrationLock(email)
	if err != nil {
		t.Fatalf("AcquireRegistrationLock failed: %v", err)
	}
	if !success {
		t.Fatal("AcquireRegistrationLock should succeed using cache")
	}
}

func TestAcquireRegistrationLock_NoCacheNoLock(t *testing.T) {
	GlobalCache = nil
	GlobalDistributedLock = nil
	defer func() {
		GlobalDistributedLock = nil
	}()

	email := "test@example.com"
	success, err := AcquireRegistrationLock(email)
	if err != nil {
		t.Fatalf("AcquireRegistrationLock should not fail when no cache or lock: %v", err)
	}
	// 当没有缓存和锁时，应该返回true（允许注册，但无法防止并发）
	if !success {
		t.Fatal("AcquireRegistrationLock should return true when no cache or lock")
	}
}
