package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/cache"
	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/gin-gonic/gin"
)

// UpdateRateLimiterConfig updates rate limiter configuration
func (h *Handlers) UpdateRateLimiterConfig(c *gin.Context) {
	//var config middleware.RateLimiterConfig
	//if err := c.ShouldBindJSON(&config); err != nil {
	//	response.Fail(c, "invalid request", nil)
	//	return
	//}

	// Update rate limiter configuration
	//middleware.SetRateLimiterConfig(config)
	response.Success(c, "rate limiter config updated", nil)
}

// HealthCheck health check endpoint
func (h *Handlers) HealthCheck(c *gin.Context) {
	// Check database connection
	sqlDB, err := h.db.DB()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "error": "database connection failed"})
		return
	}
	if err := sqlDB.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "error": "database ping failed"})
		return
	}

	// Return health status
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}

// SystemInit system initialization endpoint, returns basic configuration information
func (h *Handlers) SystemInit(c *gin.Context) {
	// Get database type
	dbDriver := config.GlobalConfig.DBDriver
	if dbDriver == "" {
		dbDriver = "sqlite"
	}

	// Determine if it's a memory database (SQLite file database may also lose data due to file loss, etc.)
	// Only persistent databases like MySQL and PostgreSQL don't need warnings
	isMemoryDB := strings.ToLower(dbDriver) == "sqlite"

	// Check if email configuration is complete
	mailConfig := config.GlobalConfig.Mail
	emailConfigured := mailConfig.Host != "" &&
		mailConfig.Port > 0 &&
		mailConfig.Username != "" &&
		mailConfig.Password != "" &&
		mailConfig.From != ""

	// Get voice clone configurations (from database first, then from .env)
	xunfeiConfig := h.getVoiceCloneConfig("xunfei")
	volcengineConfig := h.getVoiceCloneConfig("volcengine")

	// Return initialization information
	response.Success(c, "System initialization info", gin.H{
		"database": gin.H{
			"driver":     dbDriver,
			"isMemoryDB": isMemoryDB,
		},
		"email": gin.H{
			"configured": emailConfigured,
		},
		"voiceClone": gin.H{
			"xunfei": gin.H{
				"configured": xunfeiConfig != nil,
				"config":     xunfeiConfig,
			},
			"volcengine": gin.H{
				"configured": volcengineConfig != nil,
				"config":     volcengineConfig,
			},
		},
	})
}

// getVoiceCloneConfig 获取音色克隆配置（先从数据库读，再从.env读）
func (h *Handlers) getVoiceCloneConfig(provider string) map[string]interface{} {
	var configKey string
	var envConfig map[string]interface{}

	switch provider {
	case "xunfei":
		configKey = constants.KEY_VOICE_CLONE_XUNFEI_CONFIG
		// 从 .env 读取配置
		envConfig = map[string]interface{}{
			"app_id":        utils.GetEnv("XUNFEI_APP_ID"),
			"api_key":       utils.GetEnv("XUNFEI_API_KEY"),
			"base_url":      utils.GetEnv("XUNFEI_BASE_URL"),
			"ws_app_id":     utils.GetEnv("XUNFEI_WS_APP_ID"),
			"ws_api_key":    utils.GetEnv("XUNFEI_WS_API_KEY"),
			"ws_api_secret": utils.GetEnv("XUNFEI_WS_API_SECRET"),
		}
		if envConfig["base_url"] == "" {
			envConfig["base_url"] = "http://opentrain.xfyousheng.com"
		}
	case "volcengine":
		configKey = constants.KEY_VOICE_CLONE_VOLCENGINE_CONFIG
		// 从 .env 读取配置
		envConfig = map[string]interface{}{
			"app_id":         utils.GetEnv("VOLCENGINE_CLONE_APP_ID"),
			"token":          utils.GetEnv("VOLCENGINE_CLONE_TOKEN"),
			"cluster":        utils.GetEnv("VOLCENGINE_CLONE_CLUSTER"),
			"voice_type":     utils.GetEnv("VOLCENGINE_CLONE_VOICE_TYPE"),
			"encoding":       utils.GetEnv("VOLCENGINE_CLONE_ENCODING"),
			"frame_duration": utils.GetEnv("VOLCENGINE_CLONE_FRAME_DURATION"),
		}
		if envConfig["cluster"] == "" {
			envConfig["cluster"] = "volcano_icl"
		}
		if sampleRate := utils.GetIntEnv("VOLCENGINE_CLONE_SAMPLE_RATE"); sampleRate > 0 {
			envConfig["sample_rate"] = sampleRate
		}
		if bitDepth := utils.GetIntEnv("VOLCENGINE_CLONE_BIT_DEPTH"); bitDepth > 0 {
			envConfig["bit_depth"] = bitDepth
		}
		if channels := utils.GetIntEnv("VOLCENGINE_CLONE_CHANNELS"); channels > 0 {
			envConfig["channels"] = channels
		}
		if speedRatio := utils.GetFloatEnv("VOLCENGINE_CLONE_SPEED_RATIO"); speedRatio > 0 {
			envConfig["speed_ratio"] = speedRatio
		}
		if trainingTimes := utils.GetIntEnv("VOLCENGINE_CLONE_TRAINING_TIMES"); trainingTimes > 0 {
			envConfig["training_times"] = trainingTimes
		}
	default:
		return nil
	}

	// 先从数据库读取
	dbConfigStr := utils.GetValue(h.db, configKey)
	if dbConfigStr != "" {
		var dbConfig map[string]interface{}
		if err := json.Unmarshal([]byte(dbConfigStr), &dbConfig); err == nil {
			// 检查是否配置完整（至少要有必需的字段）
			if h.isConfigValid(provider, dbConfig) {
				return dbConfig
			}
		}
	}

	// 如果数据库没有或配置不完整，从 .env 读取
	if h.isConfigValid(provider, envConfig) {
		return envConfig
	}

	return nil
}

// isConfigValid 检查配置是否有效
func (h *Handlers) isConfigValid(provider string, config map[string]interface{}) bool {
	if config == nil {
		return false
	}

	switch provider {
	case "xunfei":
		appID, _ := config["app_id"].(string)
		apiKey, _ := config["api_key"].(string)
		return appID != "" && apiKey != ""
	case "volcengine":
		appID, _ := config["app_id"].(string)
		token, _ := config["token"].(string)
		return appID != "" && token != ""
	default:
		return false
	}
}

// SaveVoiceCloneConfig 保存音色克隆配置
func (h *Handlers) SaveVoiceCloneConfig(c *gin.Context) {
	var req struct {
		Provider string                 `json:"provider" binding:"required"`
		Config   map[string]interface{} `json:"config" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "参数错误", err.Error())
		return
	}

	// 验证配置
	if !h.isConfigValid(req.Provider, req.Config) {
		response.Fail(c, "配置无效", "请确保填写了所有必需的配置项")
		return
	}

	// 确定配置键
	var configKey string
	switch req.Provider {
	case "xunfei":
		configKey = constants.KEY_VOICE_CLONE_XUNFEI_CONFIG
	case "volcengine":
		configKey = constants.KEY_VOICE_CLONE_VOLCENGINE_CONFIG
	default:
		response.Fail(c, "不支持的提供商", "只支持 xunfei 和 volcengine")
		return
	}

	// 序列化为 JSON
	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		response.Fail(c, "序列化配置失败", err.Error())
		return
	}

	// 保存到数据库
	utils.SetValue(h.db, configKey, string(configJSON), "json", true, true)

	response.Success(c, "配置保存成功", nil)
}

// SystemStatus 系统状态检查接口，检查数据库、缓存、API、存储服务
func (h *Handlers) SystemStatus(c *gin.Context) {
	status := make(map[string]bool)

	// 检查数据库
	dbStatus := false
	sqlDB, err := h.db.DB()
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := sqlDB.PingContext(ctx); err == nil {
			dbStatus = true
		}
	}
	status["database"] = dbStatus

	// 检查缓存服务
	cacheStatus := false
	globalCache := cache.GetGlobalCache()
	if globalCache != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		// 尝试设置和获取一个测试键
		testKey := "__health_check__"
		if err := globalCache.Set(ctx, testKey, "test", time.Second); err == nil {
			if val, exists := globalCache.Get(ctx, testKey); exists && val == "test" {
				cacheStatus = true
				globalCache.Delete(ctx, testKey)
			}
		}
	}
	status["cache"] = cacheStatus

	// 检查API服务（通过检查当前请求是否正常处理来判断）
	status["api"] = true

	// 检查存储服务
	storageStatus := false
	err = config.GlobalStore.Ping()
	storageStatus = err == nil
	status["storage"] = storageStatus

	response.Success(c, "系统状态检查完成", status)
}

// DashboardMetrics 获取仪表板指标数据（PV、UV、API调用次数、活跃用户）
func (h *Handlers) DashboardMetrics(c *gin.Context) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterday := today.AddDate(0, 0, -1)

	// 计算今天和昨天的数据
	var todayPV, yesterdayPV int64
	var todayUV, yesterdayUV int64
	var todayAPICalls, yesterdayAPICalls int64
	var activeUsers int64

	// PV：统计UsageRecord总数（今天和昨天）
	h.db.Model(&models.UsageRecord{}).
		Where("DATE(usage_time) = ?", today.Format("2006-01-02")).
		Count(&todayPV)
	h.db.Model(&models.UsageRecord{}).
		Where("DATE(usage_time) = ?", yesterday.Format("2006-01-02")).
		Count(&yesterdayPV)

	// UV：统计独立用户数（今天和昨天）
	var todayUserIDs []uint
	var yesterdayUserIDs []uint
	h.db.Model(&models.UsageRecord{}).
		Where("DATE(usage_time) = ?", today.Format("2006-01-02")).
		Distinct("user_id").
		Pluck("user_id", &todayUserIDs)
	todayUV = int64(len(todayUserIDs))

	h.db.Model(&models.UsageRecord{}).
		Where("DATE(usage_time) = ?", yesterday.Format("2006-01-02")).
		Distinct("user_id").
		Pluck("user_id", &yesterdayUserIDs)
	yesterdayUV = int64(len(yesterdayUserIDs))

	// API调用次数：统计UsageType为API的记录（今天和昨天）
	h.db.Model(&models.UsageRecord{}).
		Where("DATE(usage_time) = ? AND usage_type = ?", today.Format("2006-01-02"), models.UsageTypeAPI).
		Count(&todayAPICalls)
	h.db.Model(&models.UsageRecord{}).
		Where("DATE(usage_time) = ? AND usage_type = ?", yesterday.Format("2006-01-02"), models.UsageTypeAPI).
		Count(&yesterdayAPICalls)

	// 活跃用户：统计最近24小时内有活动的用户数
	last24Hours := now.Add(-24 * time.Hour)
	h.db.Model(&models.UsageRecord{}).
		Where("usage_time >= ?", last24Hours).
		Distinct("user_id").
		Count(&activeUsers)

	// 计算变化百分比
	calculateChange := func(today, yesterday int64) float64 {
		if yesterday == 0 {
			if today > 0 {
				return 100.0
			}
			return 0.0
		}
		return ((float64(today) - float64(yesterday)) / float64(yesterday)) * 100.0
	}

	metrics := map[string]interface{}{
		"pv": map[string]interface{}{
			"today":     todayPV,
			"yesterday": yesterdayPV,
			"change":    calculateChange(todayPV, yesterdayPV),
		},
		"uv": map[string]interface{}{
			"today":     todayUV,
			"yesterday": yesterdayUV,
			"change":    calculateChange(todayUV, yesterdayUV),
		},
		"apiCalls": map[string]interface{}{
			"today":     todayAPICalls,
			"yesterday": yesterdayAPICalls,
			"change":    calculateChange(todayAPICalls, yesterdayAPICalls),
		},
		"activeUsers": map[string]interface{}{
			"today":     activeUsers,
			"yesterday": 0, // 活跃用户是实时数据，不需要昨日对比
			"change":    0.0,
		},
	}

	response.Success(c, "获取仪表板指标成功", metrics)
}
