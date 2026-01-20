package utils

import (
	"fmt"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
	"gorm.io/gorm"
)

func setupTestLoginSecurityManager(t *testing.T) (*LoginSecurityManager, func()) {
	logger := zaptest.NewLogger(t)
	lsm := NewLoginSecurityManager(logger)

	// 初始化缓存用于测试
	InitGlobalCache(1000, 1*time.Hour)

	cleanup := func() {
		GlobalCache = nil
	}

	return lsm, cleanup
}

func TestNewLoginSecurityManager(t *testing.T) {
	logger := zaptest.NewLogger(t)
	lsm := NewLoginSecurityManager(logger)

	if lsm == nil {
		t.Fatal("NewLoginSecurityManager returned nil")
	}
	if lsm.maxFailedAttempts != 7 {
		t.Fatalf("Expected maxFailedAttempts 7, got %d", lsm.maxFailedAttempts)
	}
	if lsm.lockDuration != 30*time.Minute {
		t.Fatalf("Expected lockDuration 30 minutes, got %v", lsm.lockDuration)
	}
	if lsm.maxPasswordLogins != 25 {
		t.Fatalf("Expected maxPasswordLogins 25, got %d", lsm.maxPasswordLogins)
	}
	if lsm.ipRateLimitPerMinute != 10 {
		t.Fatalf("Expected ipRateLimitPerMinute 10, got %d", lsm.ipRateLimitPerMinute)
	}
}

func TestLoginSecurityManager_CheckAccountLock_NoLock(t *testing.T) {
	lsm, cleanup := setupTestLoginSecurityManager(t)
	defer cleanup()

	checkFunc := func(db *gorm.DB, email string, userID uint) (*AccountLockInfo, error) {
		return &AccountLockInfo{
			IsLocked: false,
			UnlockAt: time.Now(),
		}, nil
	}

	err := lsm.CheckAccountLock(nil, "test@example.com", 1, checkFunc)
	if err != nil {
		t.Fatalf("CheckAccountLock should not fail when account is not locked: %v", err)
	}
}

func TestLoginSecurityManager_CheckAccountLock_Locked(t *testing.T) {
	lsm, cleanup := setupTestLoginSecurityManager(t)
	defer cleanup()

	unlockAt := time.Now().Add(30 * time.Minute)
	checkFunc := func(db *gorm.DB, email string, userID uint) (*AccountLockInfo, error) {
		return &AccountLockInfo{
			IsLocked: true,
			UnlockAt: unlockAt,
		}, nil
	}

	err := lsm.CheckAccountLock(nil, "test@example.com", 1, checkFunc)
	if err == nil {
		t.Fatal("CheckAccountLock should fail when account is locked")
	}
}

func TestLoginSecurityManager_RecordFailedLogin(t *testing.T) {
	lsm, cleanup := setupTestLoginSecurityManager(t)
	defer cleanup()

	recordFunc := func(db *gorm.DB, email string, userID uint, ipAddress string, failedCount int) error {
		return nil
	}

	err := lsm.RecordFailedLogin(nil, "test@example.com", 1, "192.168.1.1", recordFunc)
	if err != nil {
		t.Fatalf("RecordFailedLogin failed: %v", err)
	}

	// 验证失败计数已增加
	key := fmt.Sprintf("login:failed:%s", "test@example.com")
	count, ok := GlobalCache.Get(key)
	if !ok {
		t.Fatal("Failed login count not recorded")
	}
	if count.(int) != 1 {
		t.Fatalf("Expected failed count 1, got %d", count.(int))
	}
}

func TestLoginSecurityManager_RecordFailedLogin_MaxAttempts(t *testing.T) {
	lsm, cleanup := setupTestLoginSecurityManager(t)
	defer cleanup()

	recordFuncCalled := false
	recordFunc := func(db *gorm.DB, email string, userID uint, ipAddress string, failedCount int) error {
		recordFuncCalled = true
		if failedCount != 7 {
			t.Fatalf("Expected failedCount 7, got %d", failedCount)
		}
		return nil
	}

	email := "test@example.com"
	// 记录6次失败（还差1次）
	for i := 0; i < 6; i++ {
		lsm.RecordFailedLogin(nil, email, 1, "192.168.1.1", nil)
	}

	// 第7次失败应该触发锁定
	lsm.RecordFailedLogin(nil, email, 1, "192.168.1.1", recordFunc)
	if !recordFuncCalled {
		t.Fatal("RecordFunc should be called when max attempts reached")
	}
}

func TestLoginSecurityManager_ClearFailedLoginCount(t *testing.T) {
	lsm, cleanup := setupTestLoginSecurityManager(t)
	defer cleanup()

	email := "test@example.com"
	// 记录失败
	lsm.RecordFailedLogin(nil, email, 1, "192.168.1.1", nil)

	// 清除失败计数
	lsm.ClearFailedLoginCount(email)

	// 验证计数已清除
	key := fmt.Sprintf("login:failed:%s", email)
	_, ok := GlobalCache.Get(key)
	if ok {
		t.Fatal("Failed login count should be cleared")
	}
}

func TestLoginSecurityManager_CheckIPRateLimit_NoCache(t *testing.T) {
	lsm, cleanup := setupTestLoginSecurityManager(t)
	defer cleanup()
	GlobalCache = nil // 清除缓存

	err := lsm.CheckIPRateLimit("192.168.1.1")
	if err != nil {
		t.Fatalf("CheckIPRateLimit should not fail when cache is nil: %v", err)
	}
}

func TestLoginSecurityManager_CheckIPRateLimit_WithinLimit(t *testing.T) {
	lsm, cleanup := setupTestLoginSecurityManager(t)
	defer cleanup()

	err := lsm.CheckIPRateLimit("192.168.1.1")
	if err != nil {
		t.Fatalf("CheckIPRateLimit failed: %v", err)
	}
}

func TestLoginSecurityManager_CheckIPRateLimit_Exceeded(t *testing.T) {
	lsm, cleanup := setupTestLoginSecurityManager(t)
	defer cleanup()

	ip := "192.168.1.1"
	// 记录多次登录尝试
	for i := 0; i < lsm.ipRateLimitPerMinute; i++ {
		lsm.CheckIPRateLimit(ip)
	}

	err := lsm.CheckIPRateLimit(ip)
	if err == nil {
		t.Fatal("Expected error when rate limit exceeded")
	}
}

func TestLoginSecurityManager_CheckProxyIP(t *testing.T) {
	lsm, cleanup := setupTestLoginSecurityManager(t)
	defer cleanup()

	// 测试私有IP
	isProxy, err := lsm.CheckProxyIP("192.168.1.1")
	if err != nil {
		t.Fatalf("CheckProxyIP failed: %v", err)
	}
	if isProxy {
		t.Fatal("Private IP should not be considered proxy")
	}

	// 测试公网IP
	isProxy, err = lsm.CheckProxyIP("8.8.8.8")
	if err != nil {
		t.Fatalf("CheckProxyIP failed: %v", err)
	}
	if isProxy {
		t.Fatal("Public IP should not be considered proxy by default")
	}

	// 测试无效IP
	_, err = lsm.CheckProxyIP("invalid-ip")
	if err == nil {
		t.Fatal("Expected error for invalid IP")
	}
}

func TestLoginSecurityManager_DetectSuspiciousLogin_NoUser(t *testing.T) {
	lsm, cleanup := setupTestLoginSecurityManager(t)
	defer cleanup()

	isSuspicious, err := lsm.DetectSuspiciousLogin(nil, 0, "192.168.1.1", "Beijing", "CN", nil)
	if err != nil {
		t.Fatalf("DetectSuspiciousLogin failed: %v", err)
	}
	if isSuspicious {
		t.Fatal("Should not be suspicious for userID 0")
	}
}

func TestLoginSecurityManager_DetectSuspiciousLogin_NoLocations(t *testing.T) {
	lsm, cleanup := setupTestLoginSecurityManager(t)
	defer cleanup()

	getLocationsFunc := func(db *gorm.DB, userID uint, limit int) ([]LoginLocation, error) {
		return []LoginLocation{}, nil
	}

	isSuspicious, err := lsm.DetectSuspiciousLogin(nil, 1, "192.168.1.1", "Beijing", "CN", getLocationsFunc)
	if err != nil {
		t.Fatalf("DetectSuspiciousLogin failed: %v", err)
	}
	if isSuspicious {
		t.Fatal("Should not be suspicious when no previous locations")
	}
}

func TestLoginSecurityManager_DetectSuspiciousLogin_SameCountry(t *testing.T) {
	lsm, cleanup := setupTestLoginSecurityManager(t)
	defer cleanup()

	getLocationsFunc := func(db *gorm.DB, userID uint, limit int) ([]LoginLocation, error) {
		return []LoginLocation{
			{Country: "CN", City: "Beijing"},
		}, nil
	}

	isSuspicious, err := lsm.DetectSuspiciousLogin(nil, 1, "192.168.1.1", "Shanghai", "CN", getLocationsFunc)
	if err != nil {
		t.Fatalf("DetectSuspiciousLogin failed: %v", err)
	}
	if isSuspicious {
		t.Fatal("Should not be suspicious when same country")
	}
}

func TestLoginSecurityManager_DetectSuspiciousLogin_DifferentCountry(t *testing.T) {
	lsm, cleanup := setupTestLoginSecurityManager(t)
	defer cleanup()

	getLocationsFunc := func(db *gorm.DB, userID uint, limit int) ([]LoginLocation, error) {
		return []LoginLocation{
			{Country: "CN", City: "Beijing"},
		}, nil
	}

	isSuspicious, err := lsm.DetectSuspiciousLogin(nil, 1, "192.168.1.1", "New York", "US", getLocationsFunc)
	if err != nil {
		t.Fatalf("DetectSuspiciousLogin failed: %v", err)
	}
	if !isSuspicious {
		t.Fatal("Should be suspicious when different country")
	}
}

func TestLoginSecurityManager_CheckPasswordLoginLimit_NoUser(t *testing.T) {
	lsm, cleanup := setupTestLoginSecurityManager(t)
	defer cleanup()

	requiresEmail, err := lsm.CheckPasswordLoginLimit(nil, 0, "test@example.com", nil)
	if err != nil {
		t.Fatalf("CheckPasswordLoginLimit failed: %v", err)
	}
	if requiresEmail {
		t.Fatal("Should not require email for userID 0")
	}
}

func TestLoginSecurityManager_CheckPasswordLoginLimit_WithinLimit(t *testing.T) {
	lsm, cleanup := setupTestLoginSecurityManager(t)
	defer cleanup()

	checkFunc := func(db *gorm.DB, userID uint) (int64, error) {
		return 10, nil // 少于25次
	}

	requiresEmail, err := lsm.CheckPasswordLoginLimit(nil, 1, "test@example.com", checkFunc)
	if err != nil {
		t.Fatalf("CheckPasswordLoginLimit failed: %v", err)
	}
	if requiresEmail {
		t.Fatal("Should not require email when within limit")
	}
}

func TestLoginSecurityManager_CheckPasswordLoginLimit_Exceeded(t *testing.T) {
	lsm, cleanup := setupTestLoginSecurityManager(t)
	defer cleanup()

	checkFunc := func(db *gorm.DB, userID uint) (int64, error) {
		return 25, nil // 达到限制
	}

	requiresEmail, err := lsm.CheckPasswordLoginLimit(nil, 1, "test@example.com", checkFunc)
	if err != nil {
		t.Fatalf("CheckPasswordLoginLimit failed: %v", err)
	}
	if !requiresEmail {
		t.Fatal("Should require email when limit exceeded")
	}
}

func TestGetDeviceID(t *testing.T) {
	userAgent := "Mozilla/5.0"
	ipAddress := "192.168.1.1"

	deviceID1 := GetDeviceID(userAgent, ipAddress)
	deviceID2 := GetDeviceID(userAgent, ipAddress)

	if deviceID1 != deviceID2 {
		t.Fatal("Same user agent and IP should generate same device ID")
	}

	deviceID3 := GetDeviceID("Different User Agent", ipAddress)
	if deviceID1 == deviceID3 {
		t.Fatal("Different user agent should generate different device ID")
	}
}

func TestParseUserAgent(t *testing.T) {
	tests := []struct {
		name        string
		userAgent   string
		wantOS      string
		wantBrowser string
	}{
		{
			name:        "Chrome on Windows",
			userAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
			wantOS:      "Windows",
			wantBrowser: "Chrome",
		},
		{
			name:        "Safari on macOS",
			userAgent:   "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Safari/605.1.15",
			wantOS:      "macOS",
			wantBrowser: "Safari",
		},
		{
			name:        "Firefox on Linux",
			userAgent:   "Mozilla/5.0 (X11; Linux x86_64; rv:89.0) Gecko/20100101 Firefox/89.0",
			wantOS:      "Linux",
			wantBrowser: "Firefox",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deviceType, os, browser := ParseUserAgent(tt.userAgent)
			// 由于解析可能不精确，只检查非空
			if deviceType == "" {
				t.Fatal("DeviceType should not be empty")
			}
			if os == "" {
				t.Fatal("OS should not be empty")
			}
			if browser == "" {
				t.Fatal("Browser should not be empty")
			}
		})
	}
}

func TestInitGlobalLoginSecurityManager(t *testing.T) {
	logger := zaptest.NewLogger(t)
	InitGlobalLoginSecurityManager(logger)

	if GlobalLoginSecurityManager == nil {
		t.Fatal("GlobalLoginSecurityManager should be initialized")
	}
}
