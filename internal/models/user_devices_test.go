package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupUserDevicesTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&User{}, &UserDevice{}, &LoginHistory{}, &AccountLock{})
	require.NoError(t, err)

	return db
}

func createTestUserForDevices(t *testing.T, db *gorm.DB) *User {
	user := &User{
		Email:    "devices@example.com",
		Password: "hashedpassword",
	}
	err := db.Create(user).Error
	require.NoError(t, err)
	return user
}

func TestUserDevice_CRUD(t *testing.T) {
	db := setupUserDevicesTestDB(t)
	user := createTestUserForDevices(t, db)

	// 测试创建用户设备
	device := &UserDevice{
		UserID:     user.ID,
		DeviceID:   "device-12345",
		DeviceName: "iPhone 13 Pro",
		DeviceType: "mobile",
		OS:         "iOS 15.0",
		Browser:    "Safari",
		UserAgent:  "Mozilla/5.0 (iPhone; CPU iPhone OS 15_0 like Mac OS X)",
		IPAddress:  "192.168.1.100",
		Location:   "北京市朝阳区",
		IsTrusted:  false,
		IsActive:   true,
		LastUsedAt: time.Now(),
	}

	err := db.Create(device).Error
	assert.NoError(t, err)
	assert.NotZero(t, device.ID)
	assert.NotZero(t, device.CreatedAt)

	// 测试读取用户设备
	var retrieved UserDevice
	err = db.First(&retrieved, device.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, user.ID, retrieved.UserID)
	assert.Equal(t, "device-12345", retrieved.DeviceID)
	assert.Equal(t, "iPhone 13 Pro", retrieved.DeviceName)
	assert.Equal(t, "mobile", retrieved.DeviceType)
	assert.Equal(t, "iOS 15.0", retrieved.OS)
	assert.Equal(t, "Safari", retrieved.Browser)
	assert.Equal(t, "192.168.1.100", retrieved.IPAddress)
	assert.Equal(t, "北京市朝阳区", retrieved.Location)
	assert.False(t, retrieved.IsTrusted)
	assert.True(t, retrieved.IsActive)

	// 测试更新用户设备
	retrieved.DeviceName = "iPhone 14 Pro"
	retrieved.OS = "iOS 16.0"
	retrieved.IsTrusted = true
	err = db.Save(&retrieved).Error
	assert.NoError(t, err)

	var updated UserDevice
	err = db.First(&updated, device.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "iPhone 14 Pro", updated.DeviceName)
	assert.Equal(t, "iOS 16.0", updated.OS)
	assert.True(t, updated.IsTrusted)

	// 测试删除用户设备（软删除通过设置IsActive为false）
	err = DeleteUserDevice(db, user.ID, "device-12345")
	assert.NoError(t, err)

	var deleted UserDevice
	err = db.Where("user_id = ? AND device_id = ?", user.ID, "device-12345").First(&deleted).Error
	assert.NoError(t, err)
	assert.False(t, deleted.IsActive)
}

func TestCreateOrUpdateUserDevice(t *testing.T) {
	db := setupUserDevicesTestDB(t)
	user := createTestUserForDevices(t, db)

	// 测试创建新设备
	device, err := CreateOrUpdateUserDevice(db, user.ID, "device-001", "MacBook Pro", "laptop", "macOS", "Chrome", "Mozilla/5.0 (Macintosh)", "192.168.1.101", "上海市浦东新区")
	assert.NoError(t, err)
	assert.NotNil(t, device)
	assert.Equal(t, "device-001", device.DeviceID)
	assert.Equal(t, "MacBook Pro", device.DeviceName)
	assert.Equal(t, "laptop", device.DeviceType)
	assert.False(t, device.IsTrusted)
	assert.True(t, device.IsActive)

	// 测试更新现有设备
	updatedDevice, err := CreateOrUpdateUserDevice(db, user.ID, "device-001", "MacBook Pro M2", "laptop", "macOS 13.0", "Chrome 108", "Mozilla/5.0 (Macintosh; Intel Mac OS X)", "192.168.1.102", "上海市黄浦区")
	assert.NoError(t, err)
	assert.NotNil(t, updatedDevice)
	assert.Equal(t, device.ID, updatedDevice.ID) // 应该是同一个设备
	assert.Equal(t, "MacBook Pro M2", updatedDevice.DeviceName)
	assert.Equal(t, "macOS 13.0", updatedDevice.OS)
	assert.Equal(t, "Chrome 108", updatedDevice.Browser)
	assert.Equal(t, "192.168.1.102", updatedDevice.IPAddress)
	assert.Equal(t, "上海市黄浦区", updatedDevice.Location)
}

func TestGetUserDevice(t *testing.T) {
	db := setupUserDevicesTestDB(t)
	user := createTestUserForDevices(t, db)

	// 创建测试设备
	device, err := CreateOrUpdateUserDevice(db, user.ID, "test-device", "Test Device", "desktop", "Windows", "Edge", "Mozilla/5.0", "192.168.1.100", "北京")
	require.NoError(t, err)

	// 测试获取存在的设备
	retrieved, err := GetUserDevice(db, user.ID, "test-device")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, device.ID, retrieved.ID)
	assert.Equal(t, "test-device", retrieved.DeviceID)

	// 测试获取不存在的设备
	notFound, err := GetUserDevice(db, user.ID, "nonexistent-device")
	assert.NoError(t, err)
	assert.Nil(t, notFound)

	// 测试获取已删除的设备
	err = DeleteUserDevice(db, user.ID, "test-device")
	require.NoError(t, err)

	deleted, err := GetUserDevice(db, user.ID, "test-device")
	assert.NoError(t, err)
	assert.Nil(t, deleted)
}

func TestTrustUserDevice(t *testing.T) {
	db := setupUserDevicesTestDB(t)
	user := createTestUserForDevices(t, db)

	// 创建测试设备
	device, err := CreateOrUpdateUserDevice(db, user.ID, "trust-device", "Trust Device", "mobile", "Android", "Chrome", "Mozilla/5.0", "192.168.1.100", "广州")
	require.NoError(t, err)
	assert.False(t, device.IsTrusted)

	// 信任设备
	err = TrustUserDevice(db, user.ID, "trust-device")
	assert.NoError(t, err)

	// 验证设备已被信任
	trusted, err := GetUserDevice(db, user.ID, "trust-device")
	assert.NoError(t, err)
	assert.NotNil(t, trusted)
	assert.True(t, trusted.IsTrusted)

	// 测试信任不存在的设备（应该创建一个新的信任设备）
	err = TrustUserDevice(db, user.ID, "new-trust-device")
	assert.NoError(t, err)

	newTrusted, err := GetUserDevice(db, user.ID, "new-trust-device")
	assert.NoError(t, err)
	assert.NotNil(t, newTrusted)
	assert.True(t, newTrusted.IsTrusted)
	assert.Equal(t, "Unknown Device", newTrusted.DeviceName)
}

func TestUntrustUserDevice(t *testing.T) {
	db := setupUserDevicesTestDB(t)
	user := createTestUserForDevices(t, db)

	// 创建并信任设备
	_, err := CreateOrUpdateUserDevice(db, user.ID, "untrust-device", "Untrust Device", "tablet", "iPadOS", "Safari", "Mozilla/5.0", "192.168.1.100", "深圳")
	require.NoError(t, err)

	err = TrustUserDevice(db, user.ID, "untrust-device")
	require.NoError(t, err)

	// 取消信任
	err = UntrustUserDevice(db, user.ID, "untrust-device")
	assert.NoError(t, err)

	// 验证设备已取消信任
	untrusted, err := GetUserDevice(db, user.ID, "untrust-device")
	assert.NoError(t, err)
	assert.NotNil(t, untrusted)
	assert.False(t, untrusted.IsTrusted)
}

func TestCheckDeviceTrust(t *testing.T) {
	db := setupUserDevicesTestDB(t)
	user := createTestUserForDevices(t, db)

	// 创建信任的设备
	_, err := CreateOrUpdateUserDevice(db, user.ID, "trusted-device", "Trusted Device", "laptop", "macOS", "Safari", "Mozilla/5.0", "192.168.1.100", "杭州")
	require.NoError(t, err)
	err = TrustUserDevice(db, user.ID, "trusted-device")
	require.NoError(t, err)

	// 创建不信任的设备
	_, err = CreateOrUpdateUserDevice(db, user.ID, "untrusted-device", "Untrusted Device", "mobile", "iOS", "Safari", "Mozilla/5.0", "192.168.1.101", "南京")
	require.NoError(t, err)

	// 检查信任状态
	trusted, err := CheckDeviceTrust(db, user.ID, "trusted-device")
	assert.NoError(t, err)
	assert.True(t, trusted)

	untrusted, err := CheckDeviceTrust(db, user.ID, "untrusted-device")
	assert.NoError(t, err)
	assert.False(t, untrusted)

	// 检查不存在的设备
	notFound, err := CheckDeviceTrust(db, user.ID, "nonexistent-device")
	assert.NoError(t, err)
	assert.False(t, notFound)
}

func TestGetUserLoginDevices(t *testing.T) {
	db := setupUserDevicesTestDB(t)
	user := createTestUserForDevices(t, db)

	// 创建多个设备
	devices := []struct {
		deviceID   string
		deviceName string
		lastUsed   time.Time
	}{
		{"device-1", "Device 1", time.Now().Add(-1 * time.Hour)},
		{"device-2", "Device 2", time.Now().Add(-2 * time.Hour)},
		{"device-3", "Device 3", time.Now()},
	}

	for _, d := range devices {
		device, err := CreateOrUpdateUserDevice(db, user.ID, d.deviceID, d.deviceName, "mobile", "iOS", "Safari", "Mozilla/5.0", "192.168.1.100", "北京")
		require.NoError(t, err)

		// 更新最后使用时间
		device.LastUsedAt = d.lastUsed
		err = db.Save(device).Error
		require.NoError(t, err)
	}

	// 获取用户的登录设备列表
	loginDevices, err := GetUserLoginDevices(db, user.ID)
	assert.NoError(t, err)
	assert.Len(t, loginDevices, 3)

	// 验证按最后使用时间倒序排列
	assert.Equal(t, "device-3", loginDevices[0].DeviceID) // 最近使用的
	assert.Equal(t, "device-1", loginDevices[1].DeviceID)
	assert.Equal(t, "device-2", loginDevices[2].DeviceID) // 最早使用的
}

func TestLoginHistory_CRUD(t *testing.T) {
	db := setupUserDevicesTestDB(t)
	user := createTestUserForDevices(t, db)

	// 测试记录登录历史
	err := RecordLoginHistory(db, user.ID, user.Email, "192.168.1.100", "北京市朝阳区", "中国", "北京", "Mozilla/5.0", "device-001", "password", true, "", false)
	assert.NoError(t, err)

	// 验证登录历史记录
	var history LoginHistory
	err = db.Where("user_id = ?", user.ID).First(&history).Error
	assert.NoError(t, err)
	assert.Equal(t, user.ID, history.UserID)
	assert.Equal(t, user.Email, history.Email)
	assert.Equal(t, "192.168.1.100", history.IPAddress)
	assert.Equal(t, "北京市朝阳区", history.Location)
	assert.Equal(t, "中国", history.Country)
	assert.Equal(t, "北京", history.City)
	assert.Equal(t, "device-001", history.DeviceID)
	assert.Equal(t, "password", history.LoginType)
	assert.True(t, history.Success)
	assert.False(t, history.IsSuspicious)

	// 测试记录失败的登录
	err = RecordLoginHistory(db, user.ID, user.Email, "203.0.113.10", "未知位置", "未知", "未知", "Mozilla/5.0", "device-002", "password", false, "密码错误", true)
	assert.NoError(t, err)

	var failedHistory LoginHistory
	err = db.Where("user_id = ? AND success = ?", user.ID, false).First(&failedHistory).Error
	assert.NoError(t, err)
	assert.False(t, failedHistory.Success)
	assert.Equal(t, "密码错误", failedHistory.FailureReason)
	assert.True(t, failedHistory.IsSuspicious)
}

func TestGetRecentLoginLocations(t *testing.T) {
	db := setupUserDevicesTestDB(t)
	user := createTestUserForDevices(t, db)

	// 记录多个登录历史
	locations := []string{"北京", "上海", "广州", "深圳", "杭州"}
	for i, location := range locations {
		err := RecordLoginHistory(db, user.ID, user.Email, "192.168.1."+string(rune('1'+i)), location, "中国", location, "Mozilla/5.0", "device-"+string(rune('1'+i)), "password", true, "", false)
		require.NoError(t, err)
		time.Sleep(time.Millisecond) // 确保时间戳不同
	}

	// 获取最近的登录位置
	recentLocations, err := GetRecentLoginLocations(db, user.ID, 3)
	assert.NoError(t, err)
	assert.Len(t, recentLocations, 3)

	// 验证按时间倒序排列
	assert.Equal(t, "杭州", recentLocations[0].City)
	assert.Equal(t, "深圳", recentLocations[1].City)
	assert.Equal(t, "广州", recentLocations[2].City)
}

func TestAccountLock_CRUD(t *testing.T) {
	db := setupUserDevicesTestDB(t)
	user := createTestUserForDevices(t, db)

	// 测试创建账号锁定记录
	lock, err := CreateOrUpdateAccountLock(db, user.Email, user.ID, "192.168.1.100", 5)
	assert.NoError(t, err)
	assert.NotNil(t, lock)
	assert.Equal(t, user.Email, lock.Email)
	assert.Equal(t, user.ID, lock.UserID)
	assert.Equal(t, "192.168.1.100", lock.IPAddress)
	assert.Equal(t, 5, lock.FailedAttempts)
	assert.Equal(t, "Too many failed login attempts", lock.Reason)
	assert.True(t, lock.IsActive)
	assert.True(t, lock.IsLocked())

	// 测试更新现有锁定记录
	updatedLock, err := CreateOrUpdateAccountLock(db, user.Email, user.ID, "192.168.1.101", 8)
	assert.NoError(t, err)
	assert.Equal(t, lock.ID, updatedLock.ID) // 应该是同一个记录
	assert.Equal(t, 8, updatedLock.FailedAttempts)
	assert.Equal(t, "192.168.1.101", updatedLock.IPAddress)

	// 测试获取账号锁定记录
	retrievedLock, err := GetAccountLock(db, user.Email, user.ID)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedLock)
	assert.Equal(t, lock.ID, retrievedLock.ID)

	// 测试解锁账号
	err = UnlockAccount(db, user.Email, user.ID)
	assert.NoError(t, err)

	unlockedAccount, err := GetAccountLock(db, user.Email, user.ID)
	assert.NoError(t, err)
	assert.Nil(t, unlockedAccount) // 应该找不到活跃的锁定记录
}

func TestAccountLock_IsLocked(t *testing.T) {
	// 测试活跃且未过期的锁定
	activeLock := &AccountLock{
		IsActive: true,
		UnlockAt: time.Now().Add(time.Hour),
	}
	assert.True(t, activeLock.IsLocked())

	// 测试活跃但已过期的锁定
	expiredLock := &AccountLock{
		IsActive: true,
		UnlockAt: time.Now().Add(-time.Hour),
	}
	assert.False(t, expiredLock.IsLocked())

	// 测试非活跃的锁定
	inactiveLock := &AccountLock{
		IsActive: false,
		UnlockAt: time.Now().Add(time.Hour),
	}
	assert.False(t, inactiveLock.IsLocked())
}

func TestAccountLock_ByEmailOnly(t *testing.T) {
	db := setupUserDevicesTestDB(t)

	// 测试仅通过邮箱创建锁定记录（用户ID为0）
	lock, err := CreateOrUpdateAccountLock(db, "test@example.com", 0, "192.168.1.100", 3)
	assert.NoError(t, err)
	assert.NotNil(t, lock)
	assert.Equal(t, "test@example.com", lock.Email)
	assert.Equal(t, uint(0), lock.UserID)

	// 测试仅通过邮箱获取锁定记录
	retrievedLock, err := GetAccountLock(db, "test@example.com", 0)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedLock)
	assert.Equal(t, lock.ID, retrievedLock.ID)

	// 测试仅通过邮箱解锁
	err = UnlockAccount(db, "test@example.com", 0)
	assert.NoError(t, err)

	unlockedAccount, err := GetAccountLock(db, "test@example.com", 0)
	assert.NoError(t, err)
	assert.Nil(t, unlockedAccount)
}

func TestUserDevice_DeviceTypes(t *testing.T) {
	db := setupUserDevicesTestDB(t)
	user := createTestUserForDevices(t, db)

	deviceTypes := []struct {
		deviceType string
		os         string
		browser    string
	}{
		{"mobile", "iOS", "Safari"},
		{"mobile", "Android", "Chrome"},
		{"desktop", "Windows", "Edge"},
		{"desktop", "macOS", "Safari"},
		{"desktop", "Linux", "Firefox"},
		{"tablet", "iPadOS", "Safari"},
		{"tablet", "Android", "Chrome"},
	}

	for i, dt := range deviceTypes {
		device, err := CreateOrUpdateUserDevice(db, user.ID, "device-"+string(rune('1'+i)), "Device "+string(rune('1'+i)), dt.deviceType, dt.os, dt.browser, "Mozilla/5.0", "192.168.1.100", "北京")
		require.NoError(t, err)
		assert.Equal(t, dt.deviceType, device.DeviceType)
		assert.Equal(t, dt.os, device.OS)
		assert.Equal(t, dt.browser, device.Browser)
	}

	// 验证所有设备都已创建
	devices, err := GetUserLoginDevices(db, user.ID)
	assert.NoError(t, err)
	assert.Len(t, devices, len(deviceTypes))
}

func TestLoginHistory_SuspiciousActivity(t *testing.T) {
	db := setupUserDevicesTestDB(t)
	user := createTestUserForDevices(t, db)

	// 记录正常登录
	err := RecordLoginHistory(db, user.ID, user.Email, "192.168.1.100", "北京市", "中国", "北京", "Mozilla/5.0", "device-001", "password", true, "", false)
	require.NoError(t, err)

	// 记录可疑登录
	err = RecordLoginHistory(db, user.ID, user.Email, "203.0.113.10", "纽约", "美国", "纽约", "Mozilla/5.0", "device-002", "password", true, "", true)
	require.NoError(t, err)

	// 查询可疑登录记录
	var suspiciousLogins []LoginHistory
	err = db.Where("user_id = ? AND is_suspicious = ?", user.ID, true).Find(&suspiciousLogins).Error
	assert.NoError(t, err)
	assert.Len(t, suspiciousLogins, 1)
	assert.Equal(t, "纽约", suspiciousLogins[0].City)
	assert.Equal(t, "美国", suspiciousLogins[0].Country)
}

// Benchmark tests
func BenchmarkCreateOrUpdateUserDevice(b *testing.B) {
	db := setupUserDevicesTestDB(&testing.T{})
	user := createTestUserForDevices(&testing.T{}, db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CreateOrUpdateUserDevice(db, user.ID, "device-"+string(rune(i)), "Device "+string(rune(i)), "mobile", "iOS", "Safari", "Mozilla/5.0", "192.168.1.100", "北京")
	}
}

func BenchmarkGetUserDevice(b *testing.B) {
	db := setupUserDevicesTestDB(&testing.T{})
	user := createTestUserForDevices(&testing.T{}, db)

	CreateOrUpdateUserDevice(db, user.ID, "benchmark-device", "Benchmark Device", "mobile", "iOS", "Safari", "Mozilla/5.0", "192.168.1.100", "北京")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetUserDevice(db, user.ID, "benchmark-device")
	}
}

func BenchmarkRecordLoginHistory(b *testing.B) {
	db := setupUserDevicesTestDB(&testing.T{})
	user := createTestUserForDevices(&testing.T{}, db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RecordLoginHistory(db, user.ID, user.Email, "192.168.1.100", "北京", "中国", "北京", "Mozilla/5.0", "device-001", "password", true, "", false)
	}
}
