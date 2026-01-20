package middleware

import (
	"log"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/mssola/user_agent"
	"gorm.io/gorm"
)

// 全局配置实例
var operationLogConfig = DefaultOperationLogConfig()

// OperationLogMiddleware 记录操作日志
func OperationLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		db := c.MustGet(constants.DbField).(*gorm.DB)

		// 先执行后续处理，确保用户信息已经设置
		c.Next()

		// 获取用户信息，如果没有用户信息则跳过记录
		user, exists := c.Get(constants.UserField)
		if !exists {
			return
		}

		// 类型断言获取用户信息
		userModel, ok := user.(*models.User)
		if !ok {
			return
		}

		// 基于配置智能判断是否应该记录此操作
		method := c.Request.Method
		path := c.Request.URL.Path
		if !operationLogConfig.ShouldLogOperation(method, path) {
			return
		}

		// 获取请求的 IP 地址
		ipAddress := c.ClientIP()

		// 获取用户代理信息
		userAgent := c.GetHeader("User-Agent")

		// 获取请求来源页面
		referer := c.GetHeader("Referer")

		ua := user_agent.New(c.GetHeader("User-Agent"))
		device := ua.Platform()
		browser, version := ua.Browser()
		os := ua.OS()

		// 获取地理位置信息（根据 IP 获取）
		location := getGeoLocation(ipAddress)

		// 生成更详细的操作描述
		action := c.Request.Method
		target := c.Request.URL.Path
		details := operationLogConfig.GetOperationDescription(action, target)

		// 记录操作日志（异步执行，避免影响响应时间）
		go func() {
			err := CreateOperationLog(db, userModel.ID, userModel.DisplayName, action, target, details, ipAddress, userAgent, referer, device, browser+version, os, location.(string), action)
			if err != nil {
				// 记录错误但不影响主流程
				log.Printf("Failed to record operation log: %v", err)
			}
		}()
	}
}

// OperationLog 记录用户操作日志
type OperationLog struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	UserID          uint      `gorm:"not null" json:"user_id"`          // 操作的用户 ID
	Username        string    `gorm:"not null" json:"username"`         // 操作的用户名
	Action          string    `gorm:"not null" json:"action"`           // 操作类型（如：创建、删除、更新等）
	Target          string    `gorm:"not null" json:"target"`           // 操作目标（如：用户、订单等）
	Details         string    `gorm:"not null" json:"details"`          // 操作详细描述
	IPAddress       string    `gorm:"not null" json:"ip_address"`       // 用户 IP 地址
	UserAgent       string    `gorm:"not null" json:"user_agent"`       // 用户的浏览器信息
	Referer         string    `gorm:"not null" json:"referer"`          // 请求来源页面
	Device          string    `gorm:"not null" json:"device"`           // 用户设备（手机、桌面等）
	Browser         string    `gorm:"not null" json:"browser"`          // 浏览器信息（如 Chrome, Firefox 等）
	OperatingSystem string    `gorm:"not null" json:"operating_system"` // 操作系统（如 Windows, MacOS 等）
	Location        string    `gorm:"not null" json:"location"`         // 用户的地理位置
	RequestMethod   string    `gorm:"not null" json:"request_method"`   // HTTP 请求方法（GET、POST等）
	CreatedAt       time.Time `json:"created_at"`                       // 操作时间
}

// TableName 指定表名
func (OperationLog) TableName() string {
	return "operation_logs"
}

// CreateOperationLog 创建操作日志
func CreateOperationLog(db *gorm.DB, userID uint, username, action, target, details, ipAddress, userAgent, referer, device, browser, operatingSystem, location, requestMethod string) error {
	log := OperationLog{
		UserID:          userID,
		Username:        username,
		Action:          action,
		Target:          target,
		Details:         details,
		IPAddress:       ipAddress,
		UserAgent:       userAgent,
		Referer:         referer,
		Device:          device,
		Browser:         browser,
		OperatingSystem: operatingSystem,
		Location:        location,
		RequestMethod:   requestMethod,
		CreatedAt:       time.Now(),
	}

	// 保存操作日志到数据库
	if err := db.Create(&log).Error; err != nil {
		return err
	}
	return nil
}

func getGeoLocation(address string) interface{} {
	// 使用IP地理位置查询API获取真实地址
	return utils.GetRealAddressByIP(address)
}
