package server

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	svcmodels "github.com/LingByte/SoulNexus/internal/models/server"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/stores"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/cache"
	"github.com/LingByte/SoulNexus/pkg/voiceprint"
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
	dbDriver := config.GlobalConfig.Database.Driver
	if dbDriver == "" {
		dbDriver = "sqlite"
	}

	// Determine if it's a memory database (SQLite file database may also lose data due to file loss, etc.)
	// Only persistent databases like MySQL and PostgreSQL don't need warnings
	isMemoryDB := strings.ToLower(dbDriver) == "sqlite"

	// Check if any enabled email notification channel exists in DB.
	var enabledMailChannels int64
	_ = h.db.Model(&svcmodels.NotificationChannel{}).
		Where("type = ? AND enabled = ?", svcmodels.NotificationChannelTypeEmail, true).
		Count(&enabledMailChannels).Error
	emailConfigured := enabledMailChannels > 0

	// Check volcengine clone config from env
	volcengineConfigured := utils.GetEnv("VOLCENGINE_CLONE_APP_ID") != "" && utils.GetEnv("VOLCENGINE_CLONE_TOKEN") != ""

	// Get voiceprint recognition configuration
	voiceprintConfig := h.getVoiceprintConfig()

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
			"volcengine": gin.H{
				"configured": volcengineConfigured,
			},
		},
		"voiceprint": gin.H{
			"enabled":    voiceprintConfig["enabled"],
			"configured": voiceprintConfig["configured"],
			"config":     voiceprintConfig["config"],
		},
		"features": gin.H{
			"voiceprintEnabled": voiceprintConfig["enabled"], // 专门用于前端sidebar显示控制
		},
	})
}

// getVoiceprintConfig gets voiceprint recognition configuration
func (h *Handlers) getVoiceprintConfig() map[string]interface{} {
	// 使用新的配置方法
	cfg := voiceprint.DefaultConfig()

	// 从数据库或环境变量读取启用状态
	enabledStr := utils.GetValue(h.db, constants.KEY_VOICEPRINT_ENABLED)
	enabled := false
	if enabledStr != "" {
		enabled = enabledStr == "true"
	} else {
		// 回退到环境变量
		enabled = cfg.Enabled
	}

	// 只有在配置完整且显式启用时才启用
	configured := cfg.BaseURL != "" && cfg.APIKey != ""
	enabled = enabled && configured

	config := map[string]interface{}{
		"service_url":          cfg.BaseURL,
		"api_key":              cfg.APIKey,
		"similarity_threshold": cfg.SimilarityThreshold,
		"max_candidates":       cfg.MaxCandidates,
		"cache_enabled":        cfg.CacheEnabled,
		"log_enabled":          cfg.LogEnabled,
	}

	return map[string]interface{}{
		"enabled":    enabled,
		"configured": configured,
		"config":     config,
	}
}

// SaveVoiceprintConfig 保存声纹识别配置
func (h *Handlers) SaveVoiceprintConfig(c *gin.Context) {
	var req struct {
		Enabled bool                   `json:"enabled"`
		Config  map[string]interface{} `json:"config" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "参数错误", err.Error())
		return
	}

	// 验证配置
	if req.Config == nil {
		response.Fail(c, "配置无效", "配置不能为空")
		return
	}

	serviceURL, _ := req.Config["service_url"].(string)
	apiKey, _ := req.Config["api_key"].(string)

	if serviceURL == "" || apiKey == "" {
		response.Fail(c, "配置无效", "服务地址和API密钥不能为空")
		return
	}

	// 序列化配置为 JSON
	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		response.Fail(c, "序列化配置失败", err.Error())
		return
	}

	// 保存配置到数据库
	utils.SetValue(h.db, constants.KEY_VOICEPRINT_CONFIG, string(configJSON), "json", true, true)

	// 保存启用状态
	enabledStr := "false"
	if req.Enabled {
		enabledStr = "true"
	}
	utils.SetValue(h.db, constants.KEY_VOICEPRINT_ENABLED, enabledStr, "string", true, true)

	response.Success(c, "声纹识别配置保存成功", nil)
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
	h.db.Model(&svcmodels.UsageRecord{}).
		Where("DATE(usage_time) = ?", today.Format("2006-01-02")).
		Count(&todayPV)
	h.db.Model(&svcmodels.UsageRecord{}).
		Where("DATE(usage_time) = ?", yesterday.Format("2006-01-02")).
		Count(&yesterdayPV)

	// UV：统计独立用户数（今天和昨天）
	var todayUserIDs []uint
	var yesterdayUserIDs []uint
	h.db.Model(&svcmodels.UsageRecord{}).
		Where("DATE(usage_time) = ?", today.Format("2006-01-02")).
		Distinct("user_id").
		Pluck("user_id", &todayUserIDs)
	todayUV = int64(len(todayUserIDs))

	h.db.Model(&svcmodels.UsageRecord{}).
		Where("DATE(usage_time) = ?", yesterday.Format("2006-01-02")).
		Distinct("user_id").
		Pluck("user_id", &yesterdayUserIDs)
	yesterdayUV = int64(len(yesterdayUserIDs))

	// API调用次数：统计UsageType为API的记录（今天和昨天）
	h.db.Model(&svcmodels.UsageRecord{}).
		Where("DATE(usage_time) = ? AND usage_type = ?", today.Format("2006-01-02"), svcmodels.UsageTypeAPI).
		Count(&todayAPICalls)
	h.db.Model(&svcmodels.UsageRecord{}).
		Where("DATE(usage_time) = ? AND usage_type = ?", yesterday.Format("2006-01-02"), svcmodels.UsageTypeAPI).
		Count(&yesterdayAPICalls)

	// 活跃用户：统计最近24小时内有活动的用户数
	last24Hours := now.Add(-24 * time.Hour)
	h.db.Model(&svcmodels.UsageRecord{}).
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

// UploadAudio uploads audio file
func (h *Handlers) UploadAudio(c *gin.Context) {
	// Get uploaded file
	file, header, err := c.Request.FormFile("audio")
	if err != nil {
		response.Fail(c, "Failed to get uploaded file: "+err.Error(), nil)
		return
	}
	defer file.Close()

	// Check file type
	contentType := header.Header.Get("Content-Type")
	if contentType != "audio/webm" && contentType != "audio/wav" && contentType != "audio/mp3" {
		response.Fail(c, "Unsupported file type: "+contentType, nil)
		return
	}

	// Generate storage key (relative to storage root)
	timestamp := time.Now().Unix()
	randomStr := utils.RandString(8)
	fileName := fmt.Sprintf("audio_%d_%s.webm", timestamp, randomStr)
	storageKey := fmt.Sprintf("audio/%s", fileName)
	st := stores.Default()
	if err := st.Write(storageKey, file); err != nil {
		response.Fail(c, "Failed to upload file: "+err.Error(), nil)
		return
	}
	audioURL := strings.TrimSpace(st.PublicURL(storageKey))

	// Return success response
	response.Success(c, "音频文件上传成功", map[string]interface{}{
		"fileName":   fileName,
		"filePath":   audioURL,
		"fileSize":   header.Size,
		"uploadTime": time.Now().Format(time.RFC3339),
		"url":        audioURL,
	})
}
