package utils

import (
	"fmt"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
)

func setupTestRegistrationGuard(t *testing.T) (*RegistrationGuard, func()) {
	logger := zaptest.NewLogger(t)
	rg := NewRegistrationGuard(logger)

	// 初始化缓存用于测试
	InitGlobalCache(1000, 1*time.Hour)

	cleanup := func() {
		GlobalCache = nil
	}

	return rg, cleanup
}

func TestNewRegistrationGuard(t *testing.T) {
	logger := zaptest.NewLogger(t)
	rg := NewRegistrationGuard(logger)

	if rg == nil {
		t.Fatal("NewRegistrationGuard returned nil")
	}
	if rg.maxRegistrationsPerIP != 3 {
		t.Fatalf("Expected maxRegistrationsPerIP 3, got %d", rg.maxRegistrationsPerIP)
	}
	if rg.maxFailedAttemptsPerIP != 5 {
		t.Fatalf("Expected maxFailedAttemptsPerIP 5, got %d", rg.maxFailedAttemptsPerIP)
	}
	if rg.minPasswordLength != 8 {
		t.Fatalf("Expected minPasswordLength 8, got %d", rg.minPasswordLength)
	}
}

func TestRegistrationGuard_AddIPToBlacklist(t *testing.T) {
	rg, cleanup := setupTestRegistrationGuard(t)
	defer cleanup()

	err := rg.AddIPToBlacklist("192.168.1.0/24")
	if err != nil {
		t.Fatalf("AddIPToBlacklist failed: %v", err)
	}

	if len(rg.ipBlacklist) == 0 {
		t.Fatal("IP blacklist should not be empty after adding")
	}
}

func TestRegistrationGuard_AddIPToBlacklist_InvalidCIDR(t *testing.T) {
	rg, cleanup := setupTestRegistrationGuard(t)
	defer cleanup()

	err := rg.AddIPToBlacklist("invalid-cidr")
	if err == nil {
		t.Fatal("Expected error for invalid CIDR")
	}
}

// Note: RemoveIPFromBlacklist method is not implemented in RegistrationGuard
// This test is skipped until the method is implemented

func TestRegistrationGuard_CheckIPRateLimit_NoCache(t *testing.T) {
	rg, cleanup := setupTestRegistrationGuard(t)
	defer cleanup()
	GlobalCache = nil // 清除缓存

	err := rg.CheckIPRateLimit("192.168.1.1")
	if err != nil {
		t.Fatalf("CheckIPRateLimit should not fail when cache is nil, got: %v", err)
	}
}

func TestRegistrationGuard_CheckIPRateLimit_WithinLimit(t *testing.T) {
	rg, cleanup := setupTestRegistrationGuard(t)
	defer cleanup()

	err := rg.CheckIPRateLimit("192.168.1.1")
	if err != nil {
		t.Fatalf("CheckIPRateLimit failed: %v", err)
	}
}

func TestRegistrationGuard_CheckIPRateLimit_Exceeded(t *testing.T) {
	rg, cleanup := setupTestRegistrationGuard(t)
	defer cleanup()

	ip := "192.168.1.1"
	// 记录多次注册尝试
	for i := 0; i < rg.maxRegistrationsPerIP; i++ {
		rg.RecordRegistrationAttempt(ip, "test@example.com", true, "")
	}

	err := rg.CheckIPRateLimit(ip)
	if err == nil {
		t.Fatal("Expected error when rate limit exceeded")
	}
}

func TestRegistrationGuard_RecordRegistrationAttempt_Success(t *testing.T) {
	rg, cleanup := setupTestRegistrationGuard(t)
	defer cleanup()

	ip := "192.168.1.1"
	email := "test@example.com"

	rg.RecordRegistrationAttempt(ip, email, true, "")

	// 验证记录已增加
	key := fmt.Sprintf("reg:ip:%s", ip)
	count, ok := GlobalCache.Get(key)
	if !ok {
		t.Fatal("Registration attempt not recorded")
	}
	if count.(int) != 1 {
		t.Fatalf("Expected count 1, got %d", count.(int))
	}
}

func TestRegistrationGuard_RecordRegistrationAttempt_Failure(t *testing.T) {
	rg, cleanup := setupTestRegistrationGuard(t)
	defer cleanup()

	ip := "192.168.1.1"
	email := "test@example.com"

	rg.RecordRegistrationAttempt(ip, email, false, "invalid email")

	// 验证失败记录已增加
	failedKey := fmt.Sprintf("reg:failed:ip:%s", ip)
	failedCount, ok := GlobalCache.Get(failedKey)
	if !ok {
		t.Fatal("Failed registration attempt not recorded")
	}
	if failedCount.(int) != 1 {
		t.Fatalf("Expected failed count 1, got %d", failedCount.(int))
	}
}

func TestRegistrationGuard_CheckFailedAttempts_NoCache(t *testing.T) {
	rg, cleanup := setupTestRegistrationGuard(t)
	defer cleanup()
	GlobalCache = nil // 清除缓存

	err := rg.CheckFailedAttempts("192.168.1.1")
	if err != nil {
		t.Fatalf("CheckFailedAttempts should not fail when cache is nil, got: %v", err)
	}
}

func TestRegistrationGuard_CheckFailedAttempts_WithinLimit(t *testing.T) {
	rg, cleanup := setupTestRegistrationGuard(t)
	defer cleanup()

	err := rg.CheckFailedAttempts("192.168.1.1")
	if err != nil {
		t.Fatalf("CheckFailedAttempts failed: %v", err)
	}
}

func TestRegistrationGuard_CheckFailedAttempts_Exceeded(t *testing.T) {
	rg, cleanup := setupTestRegistrationGuard(t)
	defer cleanup()

	ip := "192.168.1.1"
	// 记录多次失败尝试
	for i := 0; i < rg.maxFailedAttemptsPerIP; i++ {
		rg.RecordRegistrationAttempt(ip, "test@example.com", false, "invalid")
	}

	err := rg.CheckFailedAttempts(ip)
	if err == nil {
		t.Fatal("Expected error when failed attempts exceeded")
	}
}

func TestRegistrationGuard_ValidateEmail_Valid(t *testing.T) {
	rg, cleanup := setupTestRegistrationGuard(t)
	defer cleanup()

	emails := []string{
		"test@example.com",
		"user.name@example.co.uk",
		"user+tag@example.com",
	}

	for _, email := range emails {
		err := rg.ValidateEmail(email)
		if err != nil {
			t.Fatalf("ValidateEmail failed for valid email %s: %v", email, err)
		}
	}
}

func TestRegistrationGuard_ValidateEmail_InvalidFormat(t *testing.T) {
	rg, cleanup := setupTestRegistrationGuard(t)
	defer cleanup()

	emails := []string{
		"invalid",
		"@example.com",
		"user@",
		"user@example",
	}

	for _, email := range emails {
		err := rg.ValidateEmail(email)
		if err == nil {
			t.Fatalf("ValidateEmail should fail for invalid email %s", email)
		}
	}
}

func TestRegistrationGuard_ValidateEmail_Disposable(t *testing.T) {
	rg, cleanup := setupTestRegistrationGuard(t)
	defer cleanup()

	err := rg.ValidateEmail("test@10minutemail.com")
	if err == nil {
		t.Fatal("ValidateEmail should fail for disposable email")
	}
}

func TestRegistrationGuard_ValidatePassword_Valid(t *testing.T) {
	rg, cleanup := setupTestRegistrationGuard(t)
	defer cleanup()

	// 根据默认配置，密码只需要至少8个字符，且需要小写字母
	passwords := []string{
		"password123",
		"longpassword",
		"validpass",
	}

	for _, password := range passwords {
		err := rg.ValidatePassword(password)
		if err != nil {
			t.Fatalf("ValidatePassword failed for valid password %s: %v", password, err)
		}
	}
}

func TestRegistrationGuard_ValidatePassword_TooShort(t *testing.T) {
	rg, cleanup := setupTestRegistrationGuard(t)
	defer cleanup()

	err := rg.ValidatePassword("short")
	if err == nil {
		t.Fatal("ValidatePassword should fail for short password")
	}
}

func TestRegistrationGuard_ValidateIP_Valid(t *testing.T) {
	rg, cleanup := setupTestRegistrationGuard(t)
	defer cleanup()

	ips := []string{
		"192.168.1.1",
		"8.8.8.8",
		"2001:0db8:85a3:0000:0000:8a2e:0370:7334",
	}

	for _, ip := range ips {
		err := rg.ValidateIP(ip)
		if err != nil {
			t.Fatalf("ValidateIP failed for valid IP %s: %v", ip, err)
		}
	}
}

func TestRegistrationGuard_ValidateIP_Invalid(t *testing.T) {
	rg, cleanup := setupTestRegistrationGuard(t)
	defer cleanup()

	err := rg.ValidateIP("invalid-ip")
	if err == nil {
		t.Fatal("ValidateIP should fail for invalid IP")
	}
}

func TestRegistrationGuard_ValidateIP_Blacklisted(t *testing.T) {
	rg, cleanup := setupTestRegistrationGuard(t)
	defer cleanup()

	err := rg.AddIPToBlacklist("192.168.1.0/24")
	if err != nil {
		t.Fatalf("AddIPToBlacklist failed: %v", err)
	}

	err = rg.ValidateIP("192.168.1.100")
	if err == nil {
		t.Fatal("ValidateIP should fail for blacklisted IP")
	}
}

func TestRegistrationGuard_CheckRegistrationAllowed(t *testing.T) {
	rg, cleanup := setupTestRegistrationGuard(t)
	defer cleanup()

	err := rg.CheckRegistrationAllowed("192.168.1.1", "test@example.com", "password123")
	if err != nil {
		t.Fatalf("CheckRegistrationAllowed failed: %v", err)
	}
}

func TestInitGlobalRegistrationGuard(t *testing.T) {
	logger := zaptest.NewLogger(t)
	InitGlobalRegistrationGuard(logger)

	if GlobalRegistrationGuard == nil {
		t.Fatal("GlobalRegistrationGuard should be initialized")
	}
}
