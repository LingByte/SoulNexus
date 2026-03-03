package constants

import (
	"time"
)

const (
	//SigUserLogin: user *User, c *gin.Context
	SigUserLogin = "user.login"
	//SigUserLogout: user *User, c *gin.Context
	SigUserLogout = "user.logout"
	//SigUserCreate: user *User, c *gin.Context
	SigUserCreate = "user.create"
	//SigUserVerifyEmail: user *User, hash, clientIp, userAgent string, db *gorm.DB
	SigUserVerifyEmail = "user.verifyemail"
	//SigUserResetPassword: user *User, hash, clientIp, userAgent string, db *gorm.DB
	SigUserResetPassword = "user.resetpassword"
	//SigUserChangeEmail: user *User, hash, clientIp, userAgent, newEmail string
	SigUserChangeEmail = "user.changeemail"
	//SigUserChangeEmailDone: user *User, oldEmail, newEmail string
	SigUserChangeEmailDone = "user.changeemaildone"
	//SigUserNewDeviceLogin: user *User, deviceInfo map[string]interface{}, db *gorm.DB
	SigUserNewDeviceLogin = "user.newdevicelogin"
)

// 缓存键前缀
const (
	CacheKeyUserByID    = "user:id:"
	CacheKeyUserByEmail = "user:email:"
)

// UserCacheExpiration 用户缓存过期时间
const UserCacheExpiration = 10 * time.Minute

// clearUserCache 清除用户相关的缓存
//func clearUserCache(user *User) {
//	if user == nil {
//		return
//	}
//	ctx := context.Background()
//	if user.ID > 0 {
//		cache.Delete(ctx, cacheKeyUserByID+fmt.Sprintf("%d", user.ID))
//	}
//	if user.Email != "" {
//		cache.Delete(ctx, cacheKeyUserByEmail+strings.ToLower(user.Email))
//	}
//}
